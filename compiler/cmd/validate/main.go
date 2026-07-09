// Command validate runs the reproducibility / inter-compiler agreement harness
// (WP-8, D0 §8.2) over N compilers' CNF exports and a shared verdict suite:
//
//   - Fleiss' κ over per-locus constructor assignment (open-texture loci
//     excluded from the denominator) — constitutional floor κ ≥ 0.70;
//   - verdict-agreement ratio over the shared event-trace suite — floor ≥ 0.90;
//   - a FALSIFICATION-CANDIDATE screen demonstrating the halt path.
//
// Floors are ASSERTED, not configurable: the command exits non-zero when either
// is breached. Input directory defaults to ../validation/testdata.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"computable-governance/compiler/internal/kernel"
	"computable-governance/compiler/internal/validation"
)

const (
	kappaFloor = 0.70
	vaFloor    = 0.90
)

func main() {
	dir := "../validation/testdata"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	bar := "══════════════════════════════════════════════════════════════════════════"
	fmt.Println(bar)
	fmt.Println(" VALIDATION HARNESS — inter-compiler agreement (D0 §8.2, WP-8)")
	fmt.Printf(" corpus of CNF exports: %s\n", dir)
	fmt.Println(bar)

	failed := false

	// ---- Fleiss' κ over constructor assignment per locus ------------------
	a, err := validation.LoadAssignments(filepath.Join(dir, "compilerA.cnf"))
	fatal(err)
	b, err := validation.LoadAssignments(filepath.Join(dir, "compilerB.cnf"))
	fatal(err)
	ratings := validation.Align([]map[string]string{a, b})
	kappa, nK := validation.FleissKappa(ratings)
	fmt.Printf(" Fleiss' κ (constructor/locus, %d shared loci, boundary excluded): %.4f\n", nK, kappa)
	if kappa < kappaFloor {
		fmt.Printf("   ✗ BELOW FLOOR κ ≥ %.2f\n", kappaFloor)
		failed = true
	} else {
		fmt.Printf("   ✓ meets floor κ ≥ %.2f\n", kappaFloor)
	}

	// ---- Verdict agreement over the shared trace suite --------------------
	va, err := validation.LoadVerdicts(filepath.Join(dir, "verdicts_A.ndjson"))
	fatal(err)
	vb, err := validation.LoadVerdicts(filepath.Join(dir, "verdicts_B.ndjson"))
	fatal(err)
	shared := map[string][]string{}
	for s, r := range va {
		if r2, ok := vb[s]; ok {
			shared[s] = []string{r, r2}
		}
	}
	vaRatio, nV := validation.VerdictAgreement(shared)
	fmt.Printf(" verdict agreement (%d shared subjects): %.4f\n", nV, vaRatio)
	if vaRatio < vaFloor {
		fmt.Printf("   ✗ BELOW FLOOR VA ≥ %.2f\n", vaFloor)
		failed = true
	} else {
		fmt.Printf("   ✓ meets floor VA ≥ %.2f\n", vaFloor)
	}

	// ---- FALSIFICATION-CANDIDATE screen (Iron Rule I3) --------------------
	fmt.Println(bar)
	if err := runFalsification(filepath.Join(dir, "falsification_input.json")); err != nil {
		fmt.Printf(" ✗ falsification screen FAILED to flag a kernel-breaking input: %v\n", err)
		failed = true
	}

	fmt.Println(bar)
	if failed {
		fmt.Println(" RESULT: FAIL — a constitutional floor was breached")
		os.Exit(1)
	}
	fmt.Println(" RESULT: PASS — floors met; kernel intact")
}

// runFalsification loads the negative fixture and asserts the screen halts it.
func runFalsification(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var in struct {
		Unit        string       `json:"unit"`
		Constructor string       `json:"constructor"`
		AST         *kernel.Expr `json:"ast"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return err
	}
	f := validation.Screen(in.Unit, in.Constructor, in.AST)
	if f == nil {
		return fmt.Errorf("input was admitted; expected a candidate")
	}
	// Feature, not bug: the unit halts and the kernel is left untouched.
	fmt.Println(" FALSIFICATION-CANDIDATE screen (negative fixture):")
	fmt.Printf("   %s\n", f.Error())
	fmt.Println("   → unit HALTED; kernel ⟨B,T⟩ not extended (I3 preserved).")
	return nil
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "validate:", err)
		os.Exit(2)
	}
}
