package validation

import (
	"math"
	"testing"

	"computable-governance/compiler/internal/kernel"
)

func TestFleissPerfectAgreement(t *testing.T) {
	ratings := map[string][]string{
		"L1": {"NRM", "NRM"},
		"L2": {"CLS", "CLS"},
		"L3": {"GRD", "GRD"},
	}
	k, n := FleissKappa(ratings)
	if n != 3 || k != 1 {
		t.Fatalf("perfect agreement: got κ=%v N=%d, want κ=1 N=3", k, n)
	}
}

// Hand-computed reference: 2 raters, 3 subjects [NRM,NRM],[CLS,CLS],[NRM,CLS].
// P̄ = (1+1+0)/3 = 0.6667; Pe = 0.5² + 0.5² = 0.5; κ = 0.1667/0.5 = 0.3333.
func TestFleissKnownValue(t *testing.T) {
	ratings := map[string][]string{
		"L1": {"NRM", "NRM"},
		"L2": {"CLS", "CLS"},
		"L3": {"NRM", "CLS"},
	}
	k, n := FleissKappa(ratings)
	if n != 3 {
		t.Fatalf("N=%d, want 3", n)
	}
	if math.Abs(k-1.0/3.0) > 1e-9 {
		t.Fatalf("κ=%v, want 0.3333", k)
	}
}

func TestVerdictAgreement(t *testing.T) {
	v := map[string][]string{
		"s1": {"compliant", "compliant"},
		"s2": {"violated", "violated"},
		"s3": {"compliant", "violated"}, // disagree
	}
	r, n := VerdictAgreement(v)
	if n != 3 || math.Abs(r-2.0/3.0) > 1e-9 {
		t.Fatalf("VA=%v N=%d, want 0.6667/3", r, n)
	}
}

func TestScreenFalsification(t *testing.T) {
	// 8th constructor → candidate.
	if Screen("u1", "OBL8", nil) == nil {
		t.Error("expected FALSIFICATION-CANDIDATE for constructor outside B")
	}
	// operator outside T (unbounded quantifier) → candidate.
	unbounded := &kernel.Expr{Op: kernel.Op("forall"), Args: []*kernel.Expr{kernel.Pred("p")}}
	if Screen("u2", "NRM", unbounded) == nil {
		t.Error("expected FALSIFICATION-CANDIDATE for op outside T")
	}
	// admissible: valid constructor + closed algebra → nil.
	ok := kernel.And(kernel.Pred("a"), kernel.Cmp(kernel.CmpGE, kernel.Var("x"), kernel.LitInt(1)))
	if f := Screen("u3", "NRM", ok); f != nil {
		t.Errorf("admissible unit flagged: %v", f)
	}
}

func TestAlignExcludesBoundaryAndMissing(t *testing.T) {
	a := map[string]string{"L1": "NRM", "L2": BoundaryCategory, "L3": "CLS"}
	b := map[string]string{"L1": "NRM", "L2": "NRM", "L4": "CLS"} // L3 missing in b
	r := Align([]map[string]string{a, b})
	if _, ok := r["L2"]; ok {
		t.Error("boundary locus L2 was not excluded")
	}
	if _, ok := r["L3"]; ok {
		t.Error("locus L3 (missing in one compiler) was not excluded")
	}
	if len(r) != 1 {
		t.Fatalf("aligned loci = %d, want 1 (L1)", len(r))
	}
}
