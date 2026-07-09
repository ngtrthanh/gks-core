// Package machine implements the execution layer Ê (spec D1.4 §3): lifecycle
// machines over the frozen six-state alphabet, driven by replaying the
// world_event trace through the transition rules S-Activate, S-Defeat,
// S-Violate and S-Exercise. It is the SINGLE writer of K̂ (Invariant I2): the
// only K̂-extending path is a PWR exercise, and every such write records its
// authorizing power in transition_log.pwr_instance.
//
// Determinism (I8): events are replayed in (occurred_at, event_id) order; the
// evaluation environment is derived only from the event payloads and the eval
// coordinates — no wall clock, no map-iteration order (norms are evaluated in
// sorted subject order).
package machine

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
)

// State is the frozen Ê lifecycle alphabet 𝒜 (I3: exactly these six; the
// resolver vocabulary IN_FORCE/DEFEATED/INACTIVE is π₃-internal and never
// persisted).
type State string

const (
	Proposed     State = "proposed"
	InForce      State = "in-force"
	Suspended    State = "suspended"
	Violated     State = "violated"
	Discharged   State = "discharged"
	Extinguished State = "extinguished"
)

// Event is one world_event row: η ∈ ℕ × TIX (D1.4 §1).
//
// Payload conventions (all optional):
//
//	"predicates": {name: bool}          ground predicate truths
//	"vars":       {name: number|string} scalar bindings
//	"domains":    {name: [{"name": string, "time": RFC3339}]}
//	"pwr":        "<instance uuid>"     for event_type "pwr-exercise"
//	"subject":    "<instance uuid>"     for mutation events
type Event struct {
	ID      kernel.UUID
	Type    string // "fact" | "pwr-exercise" | "deadline-shift" | "target-rebind"
	Agent   string
	At      time.Time
	Payload map[string]any
}

// Transition is one appended transition_log row; the DB trigger applies it to
// e_machine (θ) and checks linearity.
type Transition struct {
	MachineID kernel.UUID
	EventID   *kernel.UUID
	From      State
	To        State
	PWR       *kernel.UUID // authorizing power for K̂-affecting writes (I2)
	Mutation  *string      // "deadline-shift" | "target-rebind"
	At        time.Time
}

// Verdict is one persisted π₃ verdict, carrying its coordinates (I6).
type Verdict struct {
	Run           string
	Subject       kernel.UUID
	TText, TFact  time.Time
	Result        string // compliant | violated | conditional | inapplicable
	ConditionalOn *string
}

// Store abstracts Ê persistence so the engine is unit-testable in memory.
type Store interface {
	// EnsureMachine creates the subject's machine in 'proposed' if absent and
	// returns its id and current state.
	EnsureMachine(subject kernel.UUID) (kernel.UUID, State, error)
	// AppendTransition journals a θ change (the trigger applies it).
	AppendTransition(t Transition) error
	// InsertKernel appends a PWR-created instance to K̂ with its source-map
	// row (I9); it is a no-op returning false when the instance already
	// exists (deterministic ids make replays idempotent).
	InsertKernel(ki kernel.KernelInstance, locus, kind string) (bool, error)
	// InsertVerdict persists one verdict row.
	InsertVerdict(v Verdict) error
}

// View is the decoded K̂ content the engine executes against.
type View struct {
	Norms  map[string]kernel.NRMPayload // instance id -> payload
	Guards []evaluator.Guard
	Powers map[string]kernel.PWRPayload
}

// BuildView decodes typed constructor payloads from kernel rows. Rows whose
// payload is extraction provenance (unsupervised docx ingest) rather than a
// typed payload are skipped: guards need a Condition and a Body/Defeats,
// norms an Act, powers an Operand or OperandSchema.
func BuildView(instances []kernel.KernelInstance) *View {
	v := &View{Norms: map[string]kernel.NRMPayload{}, Powers: map[string]kernel.PWRPayload{}}
	for _, ki := range instances {
		id := ki.InstanceID.String()
		switch ki.Constructor {
		case kernel.NRM:
			p, err := kernel.DecodePayload[kernel.NRMPayload](ki)
			if err == nil && p.Act != "" {
				v.Norms[id] = p
			}
		case kernel.GRD:
			p, err := kernel.DecodePayload[kernel.GRDPayload](ki)
			if err == nil && p.Condition != nil && (len(p.Body) > 0 || len(p.Defeats) > 0) {
				v.Guards = append(v.Guards, evaluator.Guard{
					ID: id, Priority: p.Priority, Condition: p.Condition,
					Body: p.Body, Defeats: p.Defeats,
				})
			}
		case kernel.PWR:
			p, err := kernel.DecodePayload[kernel.PWRPayload](ki)
			if err == nil && (p.Operand != nil || p.OperandSchema != nil) {
				v.Powers[id] = p
			}
		}
	}
	return v
}

// Engine replays an event trace against a K̂ view and persists machines,
// transitions, PWR-created instances and verdicts.
type Engine struct {
	Store Store
	Run   string
	// Eval coordinates ⟨t_text, t_fact⟩ (I6). Environment.Now = TFact.
	TText, TFact time.Time
	// Registry is the R snapshot loaded at eval start (WP-6); OpLookup reads it.
	Registry map[string]evaluator.Value
	// Subjects restricts verdict evaluation to these norm instance ids
	// (empty = all norms in the view).
	Subjects []string
}

// Result summarizes one replay.
type Result struct {
	Events      int
	Created     []string // instance ids appended to K̂ via S-Exercise
	Transitions int
	Verdicts    []Verdict
}

type machineHandle struct {
	id    kernel.UUID
	state State
}

// DetUUID derives a deterministic RFC-4122-shaped UUID (v5-style, SHA-1) from
// a seed string. Used for machine ids and PWR-created instance ids so replays
// are idempotent and cross-compiler comparable (I8).
func DetUUID(seed string) kernel.UUID {
	h := sha1.Sum([]byte(seed))
	var u kernel.UUID
	copy(u[:], h[:16])
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}

// Replay executes the trace (D1.4 §4 Trace rule) and evaluates verdicts.
func (e *Engine) Replay(view *View, events []Event) (*Result, error) {
	res := &Result{Events: len(events)}

	// I8: strict deterministic event order.
	sort.Slice(events, func(i, j int) bool {
		if !events[i].At.Equal(events[j].At) {
			return events[i].At.Before(events[j].At)
		}
		return events[i].ID.String() < events[j].ID.String()
	})

	env := evaluator.Environment{
		Now:        e.TFact,
		Vars:       map[string]evaluator.Value{},
		Registry:   e.Registry,
		Predicates: map[string]bool{},
		Domains:    map[string][]evaluator.Fact{},
	}

	machines := map[string]*machineHandle{}
	ensure := func(subject string) (*machineHandle, error) {
		if m, ok := machines[subject]; ok {
			return m, nil
		}
		uid, err := kernel.ParseUUID(subject)
		if err != nil {
			return nil, fmt.Errorf("machine: bad subject %q: %w", subject, err)
		}
		mid, st, err := e.Store.EnsureMachine(uid)
		if err != nil {
			return nil, err
		}
		m := &machineHandle{id: mid, state: st}
		machines[subject] = m
		return m, nil
	}
	transition := func(m *machineHandle, to State, ev *kernel.UUID, pwr *kernel.UUID, mut *string, at time.Time) error {
		if m.state == to && mut == nil {
			return nil
		}
		t := Transition{MachineID: m.id, EventID: ev, From: m.state, To: to, PWR: pwr, Mutation: mut, At: at}
		if err := e.Store.AppendTransition(t); err != nil {
			return err
		}
		m.state = to
		res.Transitions++
		return nil
	}

	// Machines exist for every norm up front (θ₀: all 'proposed' or restored).
	for _, id := range sortedKeys(view.Norms) {
		if _, err := ensure(id); err != nil {
			return nil, err
		}
	}

	// ---- Small-step phase: fold the event trace ----------------------------
	for i := range events {
		ev := &events[i]
		mergeFacts(&env, ev.Payload)

		switch ev.Type {
		case "fact":
			// facts only extend ρ; no transition.

		case "pwr-exercise":
			if err := e.exercise(view, ev, env, ensure, transition, res); err != nil {
				return nil, err
			}

		case "deadline-shift", "target-rebind":
			// Mutation hooks: journaled against the subject's machine with the
			// authorizing power; the lifecycle state is unchanged.
			subject, _ := ev.Payload["subject"].(string)
			if subject == "" {
				return nil, fmt.Errorf("machine: %s event %s lacks subject", ev.Type, ev.ID)
			}
			m, err := ensure(subject)
			if err != nil {
				return nil, err
			}
			mut := ev.Type
			var pwr *kernel.UUID
			if p, ok := ev.Payload["pwr"].(string); ok {
				if u, err := kernel.ParseUUID(p); err == nil {
					pwr = &u
				}
			}
			if err := transition(m, m.state, &ev.ID, pwr, &mut, ev.At); err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("machine: unknown event type %q (%s)", ev.Type, ev.ID)
		}
	}

	// ---- Verdict phase (D1.4 §4): π₃ over the final configuration ----------
	subjects := e.Subjects
	if len(subjects) == 0 {
		subjects = sortedKeys(view.Norms)
	} else {
		subjects = append([]string(nil), subjects...)
		sort.Strings(subjects)
	}
	for _, id := range subjects {
		norm, ok := view.Norms[id]
		if !ok {
			return nil, fmt.Errorf("machine: subject %s is not a norm in the view", id)
		}
		m, err := ensure(id)
		if err != nil {
			return nil, err
		}
		v, err := e.evaluate(id, norm, m, view, env, transition)
		if err != nil {
			return nil, err
		}
		if err := e.Store.InsertVerdict(v); err != nil {
			return nil, err
		}
		res.Verdicts = append(res.Verdicts, v)
	}
	return res, nil
}

// exercise implements S-Exercise (D1.4 §3.4): the ONLY rule extending K̂.
// The operand GRD becomes a dated kernel instance; machines of every defeated
// subject transition to 'suspended' with the authorizing power recorded.
func (e *Engine) exercise(view *View, ev *Event, env evaluator.Environment,
	ensure func(string) (*machineHandle, error),
	transition func(*machineHandle, State, *kernel.UUID, *kernel.UUID, *string, time.Time) error,
	res *Result,
) error {
	pwrID, _ := ev.Payload["pwr"].(string)
	p, ok := view.Powers[pwrID]
	if !ok {
		return fmt.Errorf("machine: pwr-exercise %s names unknown power %q", ev.ID, pwrID)
	}
	if p.Operand == nil {
		return fmt.Errorf("machine: power %s has no operand", pwrID)
	}
	pwrUUID, err := kernel.ParseUUID(pwrID)
	if err != nil {
		return fmt.Errorf("machine: bad pwr id %q: %w", pwrID, err)
	}

	// match(ω, η): when the event names its exercise label, it must match the
	// power's declared exercise event.
	if lbl, ok := ev.Payload["exercise"].(string); ok && p.Event != "" && lbl != p.Event {
		return fmt.Errorf("machine: event %s exercise %q does not match power %s event %q",
			ev.ID, lbl, pwrID, p.Event)
	}

	// apply(ε, τ): dated operand instance with a deterministic identity.
	newID := DetUUID("exercise|" + pwrID + "|" + ev.ID.String())
	raw, err := json.Marshal(p.Operand)
	if err != nil {
		return fmt.Errorf("machine: marshal operand of %s: %w", pwrID, err)
	}
	ki := kernel.KernelInstance{
		InstanceID:  newID,
		Constructor: kernel.GRD,
		Payload:     kernel.JSONB(raw),
		TText:       kernel.Since(ev.At),
		TFact:       kernel.Since(ev.At),
	}
	locus := fmt.Sprintf("Ê : pwr-exercise %s : event %s", pwrID, ev.ID)
	created, err := e.Store.InsertKernel(ki, locus, "exercise")
	if err != nil {
		return err
	}
	if created {
		res.Created = append(res.Created, newID.String())
	}

	// The created guard joins the active set immediately.
	view.Guards = append(view.Guards, evaluator.Guard{
		ID: newID.String(), Priority: p.Operand.Priority,
		Condition: p.Operand.Condition, Body: p.Operand.Body, Defeats: p.Operand.Defeats,
	})

	// Subject suspension: every norm the operand defeats transitions to
	// 'suspended', journaled with the authorizing power (I2 wiring).
	for _, subject := range p.Operand.Defeats {
		m, err := ensure(subject)
		if err != nil {
			return err
		}
		if err := transition(m, Suspended, &ev.ID, &pwrUUID, nil, ev.At); err != nil {
			return err
		}
	}
	return nil
}

// evaluate maps the final configuration to one verdict.
//
// AGENT-0-DECISION-3 (provisional resolver-state mapping, per handoff §6.3):
// the resolver vocabulary is π₃-internal. IN_FORCE → compliant path;
// DEFEATED → guard-suppressed, the machine stays as it is and the verdict is
// 'inapplicable'; INACTIVE → 'inapplicable'. A norm no guard activates is
// in force by default (norms apply unless guarded); INACTIVE downgrades it
// only when an activating guard exists and did not fire.
func (e *Engine) evaluate(id string, norm kernel.NRMPayload, m *machineHandle,
	view *View, env evaluator.Environment,
	transition func(*machineHandle, State, *kernel.UUID, *kernel.UUID, *string, time.Time) error,
) (Verdict, error) {
	subjectUUID, err := kernel.ParseUUID(id)
	if err != nil {
		return Verdict{}, err
	}
	v := Verdict{Run: e.Run, Subject: subjectUUID, TText: e.TText, TFact: e.TFact}

	conditional := func(token string) Verdict {
		v.Result = "conditional"
		v.ConditionalOn = &token
		return v
	}

	// Open texture on the norm's own qualifier → conditional, never definite.
	if norm.Qualifier != nil {
		if _, err := evaluator.Eval(norm.Qualifier, env); err != nil {
			var b *evaluator.BoundaryError
			if errors.As(err, &b) {
				return conditional(b.Token), nil
			}
			return v, fmt.Errorf("machine: qualifier of %s: %w", id, err)
		}
	}

	// A machine suspended (concession) or extinguished is out of scope.
	if m.state == Suspended || m.state == Extinguished {
		v.Result = "inapplicable"
		return v, nil
	}

	relevant := make([]evaluator.Guard, 0, len(view.Guards))
	hasActivator := false
	for _, g := range view.Guards {
		refs := false
		for _, b := range g.Body {
			if b == id {
				refs, hasActivator = true, true
			}
		}
		for _, d := range g.Defeats {
			if d == id {
				refs = true
			}
		}
		if refs {
			relevant = append(relevant, g)
		}
	}

	resolution, err := evaluator.Resolve(id, relevant, env)
	if err != nil {
		var b *evaluator.BoundaryError
		if errors.As(err, &b) {
			return conditional(b.Token), nil
		}
		return v, err
	}

	inForce := resolution.Verdict == "IN_FORCE" ||
		(resolution.Verdict == "INACTIVE" && !hasActivator)

	switch {
	case resolution.Verdict == "DEFEATED":
		// AGENT-0-DECISION-3: guard-suppressed; machine state untouched.
		v.Result = "inapplicable"

	case !inForce:
		v.Result = "inapplicable"

	default:
		// S-Activate.
		if m.state == Proposed {
			if err := transition(m, InForce, nil, nil, nil, e.TFact); err != nil {
				return v, err
			}
		}
		// S-Violate applies to positive obligations only. AGENT-0-DECISION-2:
		// no further branching on the O|P|F trichotomy.
		if norm.Force == "O" && norm.Sign == "+" && !env.Predicates["performed:"+norm.Act] {
			if err := transition(m, Violated, nil, nil, nil, e.TFact); err != nil {
				return v, err
			}
			v.Result = "violated"
		} else {
			v.Result = "compliant"
		}
	}
	return v, nil
}

// mergeFacts folds an event payload into ρ (predicates, vars, domains).
func mergeFacts(env *evaluator.Environment, payload map[string]any) {
	if preds, ok := payload["predicates"].(map[string]any); ok {
		for k, raw := range preds {
			if b, ok := raw.(bool); ok {
				env.Predicates[k] = b
			}
		}
	}
	if vars, ok := payload["vars"].(map[string]any); ok {
		for k, raw := range vars {
			switch t := raw.(type) {
			case float64: // JSON numbers decode to float64; T is integer-valued
				env.Vars[k] = evaluator.VInt(int64(t))
			case int64:
				env.Vars[k] = evaluator.VInt(t)
			case int:
				env.Vars[k] = evaluator.VInt(int64(t))
			case string:
				env.Vars[k] = evaluator.VStr(t)
			case bool:
				env.Vars[k] = evaluator.VBool(t)
			}
		}
	}
	if doms, ok := payload["domains"].(map[string]any); ok {
		for name, raw := range doms {
			items, ok := raw.([]any)
			if !ok {
				continue
			}
			for _, it := range items {
				f, ok := it.(map[string]any)
				if !ok {
					continue
				}
				fact := evaluator.Fact{}
				if n, ok := f["name"].(string); ok {
					fact.Name = n
				}
				if ts, ok := f["time"].(string); ok {
					if t, err := time.Parse(time.RFC3339, ts); err == nil {
						fact.Time = t
					}
				}
				env.Domains[name] = append(env.Domains[name], fact)
			}
			// I8: domain order must not depend on event/map order.
			sort.Slice(env.Domains[name], func(i, j int) bool {
				a, b := env.Domains[name][i], env.Domains[name][j]
				if !a.Time.Equal(b.Time) {
					return a.Time.Before(b.Time)
				}
				return a.Name < b.Name
			})
		}
	}
}

func sortedKeys[T any](m map[string]T) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
