package evaluator

import (
	"testing"

	"computable-governance/compiler/internal/kernel"
)

func guard(id string, prio int, defeats, body []string) Guard {
	return Guard{ID: id, Priority: prio, Condition: kernel.LitBool(true), Defeats: defeats, Body: body}
}

func TestResolveHigherPriorityDefeats(t *testing.T) {
	guards := []Guard{
		guard("g:activate", 1, nil, []string{"n1"}),
		guard("g:concession", 10, []string{"n1"}, nil),
	}
	res, err := Resolve("n1", guards, env())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Verdict != "DEFEATED" {
		t.Fatalf("higher-priority defeats must win, got %s", res.Verdict)
	}
}

func TestResolveUnmetConditionIsSkipped(t *testing.T) {
	g := guard("g:defeat", 10, []string{"n1"}, nil)
	g.Condition = kernel.LitBool(false)
	guards := []Guard{g, guard("g:activate", 1, nil, []string{"n1"})}
	res, err := Resolve("n1", guards, env())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Verdict != "IN_FORCE" {
		t.Fatalf("unmet defeater must not decide, got %s", res.Verdict)
	}
}

// I8: with equal priorities the verdict must not depend on input order —
// ties are broken by guard ID.
func TestResolveTieBreakIsInputOrderIndependent(t *testing.T) {
	a := guard("g:a-defeats", 5, []string{"n1"}, nil)
	b := guard("g:b-activates", 5, nil, []string{"n1"})

	r1, err := Resolve("n1", []Guard{a, b}, env())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	r2, err := Resolve("n1", []Guard{b, a}, env())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r1.Verdict != r2.Verdict {
		t.Fatalf("verdict depends on input order: %s vs %s (I8 violation)", r1.Verdict, r2.Verdict)
	}
	// "g:a-defeats" < "g:b-activates" lexicographically, so the defeater decides.
	if r1.Verdict != "DEFEATED" {
		t.Fatalf("ID tie-break must pick g:a-defeats first, got %s", r1.Verdict)
	}
	for i := range r1.Steps {
		if r1.Steps[i].GuardID != r2.Steps[i].GuardID {
			t.Fatalf("step order depends on input order at %d: %s vs %s",
				i, r1.Steps[i].GuardID, r2.Steps[i].GuardID)
		}
	}
}
