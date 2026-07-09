package machine

import (
	"testing"
	"time"

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
)

// fakeStore is the in-memory Store used by the unit suite.
type fakeStore struct {
	machines    map[string]State // subject id -> state
	transitions []Transition
	created     []kernel.KernelInstance
	verdicts    []Verdict
}

func newFake() *fakeStore { return &fakeStore{machines: map[string]State{}} }

func (f *fakeStore) EnsureMachine(subject kernel.UUID) (kernel.UUID, State, error) {
	id := subject.String()
	if _, ok := f.machines[id]; !ok {
		f.machines[id] = Proposed
	}
	return DetUUID("machine|" + id), f.machines[id], nil
}

func (f *fakeStore) AppendTransition(t Transition) error {
	f.transitions = append(f.transitions, t)
	for subject, _ := range f.machines {
		if DetUUID("machine|"+subject) == t.MachineID {
			f.machines[subject] = t.To
		}
	}
	return nil
}

func (f *fakeStore) InsertKernel(ki kernel.KernelInstance, locus, kind string) (bool, error) {
	for _, c := range f.created {
		if c.InstanceID == ki.InstanceID {
			return false, nil
		}
	}
	f.created = append(f.created, ki)
	return true, nil
}

func (f *fakeStore) InsertVerdict(v Verdict) error {
	f.verdicts = append(f.verdicts, v)
	return nil
}

// ---- fixtures mirroring D8 Run 1 (26 U.S.C. §121) ---------------------------

const (
	tN1 = "a1000000-0000-4000-8000-000000000001"
	tG1 = "d1000000-0000-4000-8000-000000000001"
	tG2 = "d2000000-0000-4000-8000-000000000001"
	tP1 = "b1000000-0000-4000-8000-000000000002"
	tNA = "2a000000-0000-4000-8000-000000000002" // n2a identify & control
	tNB = "2b000000-0000-4000-8000-000000000002" // n2b appropriate action (OT-1)
)

func run1View() *View {
	return &View{
		Norms: map[string]kernel.NRMPayload{
			tN1: {Bearer: "taxpayer", Counterparty: "irs",
				Act: "exclude_gain_from_gross_income", Sign: "+", Force: "P"},
		},
		Guards: []evaluator.Guard{
			{ID: tG1, Priority: 100,
				Condition: kernel.And(
					kernel.Pred("classified"),
					kernel.Cmp(kernel.CmpLE, kernel.Var("realized_gain"), kernel.LitInt(250_000))),
				Body: []string{tN1}},
			{ID: tG2, Priority: 200,
				Condition: kernel.CountCmp("taxpayer_prior_sales",
					kernel.Within("P2Y", "", kernel.Pred("section121_exclusion_applied")),
					kernel.CmpGE, 1),
				Defeats: []string{tN1}},
		},
		Powers: map[string]kernel.PWRPayload{},
	}
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func saleEvent(priorSale bool) Event {
	payload := map[string]any{
		"predicates": map[string]any{"classified": true},
		"vars":       map[string]any{"realized_gain": float64(200_000)},
	}
	if priorSale {
		payload["domains"] = map[string]any{
			"taxpayer_prior_sales": []any{
				map[string]any{"name": "section121_exclusion_applied", "time": "2025-01-15T00:00:00Z"},
			},
		}
	}
	return Event{ID: DetUUID("ev|sale"), Type: "fact", At: at("2026-01-10T00:00:00Z"), Payload: payload}
}

func engine(s Store, subjects ...string) *Engine {
	return &Engine{Store: s, Run: "test",
		TText: at("2026-01-10T00:00:00Z"), TFact: at("2026-01-10T00:00:00Z"),
		Subjects: subjects}
}

// Run 1, scenario A: no prior §121 sale — g1 activates n1, verdict compliant.
func TestRun1Compliant(t *testing.T) {
	s := newFake()
	res, err := engine(s).Replay(run1View(), []Event{saleEvent(false)})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(res.Verdicts) != 1 || res.Verdicts[0].Result != "compliant" {
		t.Fatalf("want one compliant verdict, got %+v", res.Verdicts)
	}
	if s.machines[tN1] != InForce {
		t.Fatalf("n1 machine should be in-force (S-Activate), got %s", s.machines[tN1])
	}
}

// Run 1, scenario B: a prior sale within P2Y — g2 (priority 200) defeats n1.
// Agent-0 Ruling 3: DEFEATED is a first-class verdict distinct from
// inapplicable; the defeat must not advance the machine.
func TestRun1AntiStackingDefeats(t *testing.T) {
	s := newFake()
	res, err := engine(s).Replay(run1View(), []Event{saleEvent(true)})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res.Verdicts[0].Result != "defeated" {
		t.Fatalf("want defeated (guard-suppressed), got %s", res.Verdicts[0].Result)
	}
	if s.machines[tN1] != Proposed {
		t.Fatalf("defeat must not advance the machine, got %s", s.machines[tN1])
	}
}

// ---- fixtures mirroring D8 Run 2 (ISO 9001 §8.7) ----------------------------

func run2View() *View {
	return &View{
		Norms: map[string]kernel.NRMPayload{
			tNA: {Bearer: "org", Counterparty: "qms",
				Act: "identify_and_control", Sign: "+", Force: "O"},
			tNB: {Bearer: "org", Counterparty: "qms",
				Act: "take_action", Sign: "+", Force: "O",
				Qualifier: kernel.Boundary("OT-1", "appropriate")},
		},
		Powers: map[string]kernel.PWRPayload{
			tP1: {Holder: "c3", Effect: "create", Event: "concession-record",
				Operand: &kernel.GRDPayload{
					Condition: kernel.Pred("concession_granted"),
					Defeats:   []string{tNA},
					Priority:  150,
				}},
		},
	}
}

func TestRun2ConcessionSuspends(t *testing.T) {
	s := newFake()
	events := []Event{
		{ID: DetUUID("ev|nc"), Type: "fact", At: at("2026-02-01T00:00:00Z"),
			Payload: map[string]any{"predicates": map[string]any{
				"conforms_to_requirements":       false,
				"performed:identify_and_control": true,
			}}},
		{ID: DetUUID("ev|concession"), Type: "pwr-exercise", Agent: "person:qm",
			At: at("2026-02-02T00:00:00Z"),
			Payload: map[string]any{
				"pwr":        tP1,
				"exercise":   "concession-record",
				"predicates": map[string]any{"concession_granted": true},
			}},
	}
	res, err := engine(s, tNA, tNB).Replay(run2View(), events)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	// S-Exercise extended K̂ by exactly the operand GRD.
	if len(res.Created) != 1 {
		t.Fatalf("expected 1 created instance, got %v", res.Created)
	}
	if len(s.created) != 1 || s.created[0].Constructor != kernel.GRD {
		t.Fatalf("created instance must be the operand GRD")
	}

	// n2a suspended with the authorizing power journaled (I2 wiring).
	if s.machines[tNA] != Suspended {
		t.Fatalf("n2a should be suspended, got %s", s.machines[tNA])
	}
	var found bool
	for _, tr := range s.transitions {
		if tr.To == Suspended && tr.PWR != nil && tr.PWR.String() == tP1 {
			found = true
		}
	}
	if !found {
		t.Fatal("suspension transition must record pwr_instance = p1")
	}

	// Verdicts: n2a inapplicable (suspended); n2b conditional on OT-1.
	byID := map[string]Verdict{}
	for _, v := range res.Verdicts {
		byID[v.Subject.String()] = v
	}
	if byID[tNA].Result != "inapplicable" {
		t.Fatalf("n2a verdict: want inapplicable, got %s", byID[tNA].Result)
	}
	if byID[tNB].Result != "conditional" || byID[tNB].ConditionalOn == nil || *byID[tNB].ConditionalOn != "OT-1" {
		t.Fatalf("n2b verdict: want conditional on OT-1, got %+v", byID[tNB])
	}

	// Replay idempotency: the deterministic GRD id is not created twice.
	res2, err := engine(s, tNA, tNB).Replay(run2View(), events)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if len(res2.Created) != 0 {
		t.Fatalf("second replay must not re-create the operand GRD, got %v", res2.Created)
	}
}

// S-Violate: an in-force positive obligation whose act was not performed.
func TestObligationViolated(t *testing.T) {
	s := newFake()
	view := &View{Norms: map[string]kernel.NRMPayload{
		tNA: {Bearer: "org", Counterparty: "qms",
			Act: "identify_and_control", Sign: "+", Force: "O"},
	}}
	res, err := engine(s).Replay(view, []Event{
		{ID: DetUUID("ev|nothing"), Type: "fact", At: at("2026-02-01T00:00:00Z"),
			Payload: map[string]any{}},
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if res.Verdicts[0].Result != "violated" {
		t.Fatalf("want violated, got %s", res.Verdicts[0].Result)
	}
	if s.machines[tNA] != Violated {
		t.Fatalf("machine should be violated, got %s", s.machines[tNA])
	}
}

// I8: event input order must not change the outcome (replay sorts by
// occurred_at then event id).
func TestReplayOrderIndependent(t *testing.T) {
	mk := func(reversed bool) []Verdict {
		s := newFake()
		e1 := saleEvent(false)
		e2 := Event{ID: DetUUID("ev|later"), Type: "fact", At: at("2026-01-11T00:00:00Z"),
			Payload: map[string]any{"vars": map[string]any{"realized_gain": float64(300_000)}}}
		events := []Event{e1, e2}
		if reversed {
			events = []Event{e2, e1}
		}
		res, err := engine(s).Replay(run1View(), events)
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		return res.Verdicts
	}
	a, b := mk(false), mk(true)
	if a[0].Result != b[0].Result {
		t.Fatalf("verdict depends on event input order: %s vs %s", a[0].Result, b[0].Result)
	}
	// The later event raises the gain above the cap, so g1 no longer fires.
	if a[0].Result != "inapplicable" {
		t.Fatalf("want inapplicable at final configuration, got %s", a[0].Result)
	}
}
