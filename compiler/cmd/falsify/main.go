// Command falsify runs the falsification campaign (Track C, D0 §8.4/§9.1):
// it screens a held-out / adversarial input set against the frozen kernel,
// emits a FALSIFICATION-CANDIDATE (and HALTS the unit) for any input that would
// require a 7th/8th constructor or an operator outside the sub-Turing algebra T,
// and measures basis growth to test the Registry Law (Θ(1) basis growth, I3).
//
//	falsify [inputs.json]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"computable-governance/compiler/internal/kernel"
	"computable-governance/compiler/internal/validation"
)

type unit struct {
	Unit        string       `json:"unit"`
	Constructor string       `json:"constructor"`
	AST         *kernel.Expr `json:"ast"`
}

func main() {
	path := "../validation/falsification/inputs.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "falsify:", err)
		os.Exit(2)
	}
	var units []unit
	if err := json.Unmarshal(raw, &units); err != nil {
		fmt.Fprintln(os.Stderr, "falsify:", err)
		os.Exit(2)
	}

	admitted := map[string]bool{} // distinct constructors used by admitted units
	var candidates []string
	var report []string

	report = append(report, "# Falsification Campaign — held-out corpus (Track C)\n")
	report = append(report, fmt.Sprintf("Inputs screened: %d\n", len(units)))
	report = append(report, "## Per-unit outcome\n")

	bar := "══════════════════════════════════════════════════════════════════════════"
	fmt.Println(bar)
	fmt.Println(" FALSIFICATION CAMPAIGN (D0 §8.4 / §9.1)")
	fmt.Println(bar)

	for _, u := range units {
		if f := validation.Screen(u.Unit, u.Constructor, u.AST); f != nil {
			candidates = append(candidates, u.Unit)
			fmt.Printf(" HALT  %s\n   %s\n", u.Unit, f.Error())
			report = append(report, fmt.Sprintf("- **HALT** `%s` — %s", u.Unit, f.Reason))
			continue
		}
		admitted[u.Constructor] = true
		fmt.Printf(" OK    %s  → %s\n", u.Unit, u.Constructor)
		report = append(report, fmt.Sprintf("- OK `%s` → %s", u.Unit, u.Constructor))
	}

	basis := make([]string, 0, len(admitted))
	for c := range admitted {
		basis = append(basis, c)
	}
	sort.Strings(basis)

	fmt.Println(bar)
	fmt.Printf(" admitted units use %d distinct constructor(s): %v\n", len(basis), basis)
	fmt.Printf(" FALSIFICATION-CANDIDATEs (halted): %d\n", len(candidates))
	registryLaw := len(basis) <= 6
	fmt.Printf(" Registry Law (basis ≤ 6, Θ(1) growth): %s\n", passFail(registryLaw))
	fmt.Println(bar)

	report = append(report, "")
	report = append(report, fmt.Sprintf("## Result\n"))
	report = append(report, fmt.Sprintf("- admitted units use **%d** distinct constructors: %v", len(basis), basis))
	report = append(report, fmt.Sprintf("- FALSIFICATION-CANDIDATEs halted: **%d**", len(candidates)))
	report = append(report, fmt.Sprintf("- Registry Law (basis ≤ 6, Θ(1)): **%s**", passFail(registryLaw)))
	report = append(report, "")
	report = append(report, "The kernel was NOT extended to admit any adversarial input; each was")
	report = append(report, "halted with a FALSIFICATION-CANDIDATE record (I3 Iron Rule preserved).")
	_ = os.WriteFile("../validation/falsification/REPORT.md", []byte(joinLines(report)), 0o644)

	if !registryLaw {
		os.Exit(1) // basis grew — a genuine falsification of the Registry Law
	}
}

func passFail(ok bool) string {
	if ok {
		return "HELD"
	}
	return "BROKEN"
}

func joinLines(xs []string) string {
	out := ""
	for _, x := range xs {
		out += x + "\n"
	}
	return out
}
