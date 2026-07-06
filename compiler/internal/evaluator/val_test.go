package evaluator

import (
	"math/big"
	"testing"

	"computable-governance/compiler/internal/kernel"
)

// D8 Run 6 — KPI-SEC-03 "on-time incident logging rate":
//
//	measure = ratio(count(log-events ≤ 24h in Q), count(incidents in Q))
//	target  = 0.95 × reg(policy-p11-§4.threshold)   [P-11 sets the threshold = 1]
//	verdict : measure ≥ target
//
// The target is arithmetic over a normative REFERENCE embedded in a formula
// (D1 falsification vector #7); it must compile and evaluate EXACTLY.
func run6VAL() kernel.VALPayload {
	return kernel.VALPayload{
		Function:   "on_time_logging_rate",
		Unit:       "ratio",
		Comparator: kernel.CmpGE,
		Measure: kernel.Ratio(
			kernel.CountCmp("log_events", kernel.Pred("on_time"), kernel.CmpGE, 0), // placeholder; overridden per-case below
			kernel.Var("total_incidents"),
		),
		Target: kernel.Arith(kernel.ArithMul,
			kernel.LitRat("0.95"),
			kernel.Lookup("policy-p11-§4.threshold")),
	}
}

// evalRate builds the Run-6 VAL with an explicit measured ratio and evaluates
// its boolean projection against the registry-referenced target.
func evalRate(t *testing.T, onTime, total int64, threshold *big.Rat) bool {
	t.Helper()
	val := run6VAL()
	val.Measure = kernel.Ratio(kernel.LitInt(onTime), kernel.LitInt(total))

	e := Environment{
		Registry: map[string]Value{"policy-p11-§4.threshold": VRat(threshold)},
	}
	v, err := Eval(val.AsExpr(), e)
	if err != nil {
		t.Fatalf("eval VAL: %v", err)
	}
	return v.Bool()
}

func TestRun6KPIExactRational(t *testing.T) {
	full := big.NewRat(1, 1) // P-11 threshold = 100%

	// 19/20 = 0.95 exactly equals the target 0.95 × 1 → satisfied (≥).
	if !evalRate(t, 19, 20, full) {
		t.Fatal("19/20 should meet the 0.95 target exactly")
	}
	// 189/200 = 0.945 < 0.95 → not satisfied. A float64 pipeline is where
	// this boundary case silently flips; big.Rat keeps it exact.
	if evalRate(t, 189, 200, full) {
		t.Fatal("0.945 must fall below the 0.95 target")
	}
	// 96/100 = 0.96 > 0.95 → satisfied.
	if !evalRate(t, 96, 100, full) {
		t.Fatal("0.96 should exceed the target")
	}
}

// The target tracks the referenced threshold: if P-11 §4 is amended to 0.80,
// the same measure re-verdicts against 0.95 × 0.80 = 0.76.
func TestRun6TargetFollowsReference(t *testing.T) {
	lowered := big.NewRat(80, 100) // P-11 amended to 80%
	// measure 0.77 ≥ 0.76 target → satisfied only under the lowered threshold.
	if !evalRate(t, 77, 100, lowered) {
		t.Fatal("0.77 should meet 0.95 × 0.80 = 0.76")
	}
	if evalRate(t, 77, 100, big.NewRat(1, 1)) {
		t.Fatal("0.77 should fail against the full 0.95 target")
	}
}

// Exactness guarantee: a repeating fraction that has no finite float64
// representation compares correctly.
func TestRatioExactNonTerminating(t *testing.T) {
	// 1/3 vs 33333/100000 (=0.33333): 1/3 is strictly greater.
	e := Environment{}
	expr := kernel.Cmp(kernel.CmpGT,
		kernel.Ratio(kernel.LitInt(1), kernel.LitInt(3)),
		kernel.LitRat("33333/100000"))
	v, err := Eval(expr, e)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if !v.Bool() {
		t.Fatal("1/3 must be strictly greater than 0.33333 (exact rational ordering)")
	}
}
