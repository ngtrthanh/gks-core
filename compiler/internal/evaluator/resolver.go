package evaluator

import (
	"fmt"
	"sort"

	"computable-governance/compiler/internal/cnf"
	"computable-governance/compiler/internal/kernel"
)

// Guard is a resolved GRD instance ready for priority resolution.
type Guard struct {
	ID        string
	Priority  int
	Condition *kernel.Expr
	Body      []string // instance_ids this guard activates
	Defeats   []string // instance_ids this guard defeats
}

// Step records the evaluation of one guard during resolution (for the trace).
type Step struct {
	GuardID  string
	Priority int
	CondMet  bool
	Relation string // "defeats" | "activates" | "irrelevant"
	Decisive bool   // whether this step determined the verdict
}

// Resolution is the outcome of resolving a norm against its guards.
type Resolution struct {
	NormID  string
	Verdict string // "IN_FORCE" | "DEFEATED" | "INACTIVE"
	Steps   []Step
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// condKey is a store-independent tie-break key: the canonical hash of the guard's
// condition (empty when the guard is unconditional). Semantically identical
// guards hash identically across independent stores, so equal-priority ordering
// no longer depends on random store UUIDs (exit-review M6).
func condKey(g Guard) string {
	if g.Condition == nil {
		return ""
	}
	return cnf.CanonicalHash(g.Condition)
}

// Resolve applies defeasible priority resolution (D1.4 §3.2). Guards are
// considered highest-priority first; the first guard whose condition holds and
// that references the norm decides the verdict. A higher-priority `defeats`
// therefore disables the norm even if a lower-priority guard would activate it.
//
// I8 (determinism) + cross-store convergence (exit-review M6): equal priorities
// are tie-broken by a CONTENT key — the canonical hash of the guard's condition,
// not the store-assigned UUID. Two independent implementations ingesting the same
// corpus therefore order equal-priority guards identically (the store UUID is only
// a last resort for genuine content-duplicates, and cannot be relied on across
// stores; content-addressing the norm references is tracked as the WS-E verdict
// contract). Ordering never depends on DB row order or map iteration.
func Resolve(normID string, guards []Guard, env Environment) (Resolution, error) {
	ordered := make([]Guard, len(guards))
	copy(ordered, guards)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority > ordered[j].Priority
		}
		if ci, cj := condKey(ordered[i]), condKey(ordered[j]); ci != cj {
			return ci < cj
		}
		return ordered[i].ID < ordered[j].ID
	})

	res := Resolution{NormID: normID, Verdict: "INACTIVE"}
	decided := false
	for _, g := range ordered {
		v, err := Eval(g.Condition, env)
		if err != nil {
			return res, fmt.Errorf("resolve: guard %s: %w", g.ID, err)
		}
		met := v.Bool()

		step := Step{GuardID: g.ID, Priority: g.Priority, CondMet: met, Relation: "irrelevant"}
		switch {
		case contains(g.Defeats, normID):
			step.Relation = "defeats"
		case contains(g.Body, normID):
			step.Relation = "activates"
		}

		if met && !decided && step.Relation != "irrelevant" {
			step.Decisive = true
			decided = true
			if step.Relation == "defeats" {
				res.Verdict = "DEFEATED"
			} else {
				res.Verdict = "IN_FORCE"
			}
		}
		res.Steps = append(res.Steps, step)
	}
	return res, nil
}
