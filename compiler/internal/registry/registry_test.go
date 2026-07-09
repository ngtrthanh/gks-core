package registry

import (
	"math/big"
	"testing"

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
)

// TestI4RenameStability witnesses Invariant I4: a bijective renaming of every
// registry token — applied to BOTH the registry keys and every Lookup that
// references them — leaves every verdict unchanged. Registry tokens are
// semantically inert. Pure test (no DB).
func TestI4RenameStability(t *testing.T) {
	reg := map[string]evaluator.Value{
		"policy-p11-§4.threshold": evaluator.VRat(big.NewRat(1, 1)),
		"kpi.floor":               evaluator.VRat(big.NewRat(19, 20)), // 0.95
	}

	// A verdict suite exercising Lookup in comparisons and arithmetic.
	suite := []*kernel.Expr{
		// measure(0.96) >= 0.95 × reg(threshold=1)  -> true
		kernel.Cmp(kernel.CmpGE,
			kernel.LitRat("0.96"),
			kernel.Arith(kernel.ArithMul, kernel.LitRat("0.95"), kernel.Lookup("policy-p11-§4.threshold"))),
		// reg(floor) < reg(threshold)  -> 0.95 < 1 -> true
		kernel.Cmp(kernel.CmpLT, kernel.Lookup("kpi.floor"), kernel.Lookup("policy-p11-§4.threshold")),
		// reg(floor) >= 0.96  -> 0.95 >= 0.96 -> false
		kernel.Cmp(kernel.CmpGE, kernel.Lookup("kpi.floor"), kernel.LitRat("0.96")),
	}

	evalAll := func(exprs []*kernel.Expr, env evaluator.Environment) []bool {
		out := make([]bool, len(exprs))
		for i, e := range exprs {
			v, err := evaluator.Eval(e, env)
			if err != nil {
				t.Fatalf("eval[%d]: %v", i, err)
			}
			out[i] = v.Bool()
		}
		return out
	}

	base := evalAll(suite, evaluator.Environment{Registry: reg})

	// Bijective rename σ over all tokens.
	sigma := map[string]string{
		"policy-p11-§4.threshold": "τ-001",
		"kpi.floor":               "τ-002",
	}
	regR := RenameTokens(reg, sigma)
	suiteR := make([]*kernel.Expr, len(suite))
	for i, e := range suite {
		suiteR[i] = RenameLookups(e, sigma)
	}
	got := evalAll(suiteR, evaluator.Environment{Registry: regR})

	if len(base) != len(got) {
		t.Fatalf("verdict count changed: %d vs %d", len(base), len(got))
	}
	for i := range base {
		if base[i] != got[i] {
			t.Fatalf("I4 violated: verdict[%d] changed under token rename (%v -> %v)", i, base[i], got[i])
		}
	}

	// Sanity: the rename must actually have changed the token names, else the
	// test proves nothing.
	if _, stillOld := regR["kpi.floor"]; stillOld {
		t.Fatal("rename did not apply to registry keys")
	}
	if RenameLookups(kernel.Lookup("kpi.floor"), sigma).Name != "τ-002" {
		t.Fatal("rename did not apply to Lookup nodes")
	}
}
