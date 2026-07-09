// Package validation implements the reproducibility / inter-compiler agreement
// harness (D0 §8.2, WP-8): Fleiss' κ over per-locus constructor assignment,
// verdict-agreement over a shared event-trace suite, and the
// FALSIFICATION-CANDIDATE screen that halts a unit rather than stretching the
// frozen kernel ⟨B, T⟩ (I3 Iron Rule).
//
// Constitutional floors κ ≥ 0.70 and VA ≥ 0.90 are asserted by the caller
// (cmd/validate); this package computes the numbers.
package validation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"computable-governance/compiler/internal/kernel"
)

// BoundaryCategory marks a locus as open-texture; such loci are EXCLUDED from
// the κ denominator (a boundary token is not a constructor disagreement).
const BoundaryCategory = "BOUNDARY"

// cnfLine is the subset of a CNF export record the harness needs.
type cnfLine struct {
	Locus       string          `json:"locus"`
	Constructor string          `json:"constructor"`
	Payload     json.RawMessage `json:"payload"`
}

// LoadAssignments reads one compiler's CNF export (NDJSON) and returns the
// category assigned at each locus: the constructor, or BoundaryCategory when
// the record carries an open-texture boundary token.
func LoadAssignments(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var r cnfLine
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if r.Locus == "" {
			continue
		}
		cat := r.Constructor
		if cat == BoundaryCategory || containsBoundary(r.Payload) {
			cat = BoundaryCategory
		}
		out[r.Locus] = cat
	}
	return out, sc.Err()
}

func containsBoundary(payload json.RawMessage) bool {
	// An open-texture boundary token serializes as {"op":"boundary",...}.
	return len(payload) > 0 && bytesContains(payload, `"op":"boundary"`)
}

func bytesContains(b []byte, sub string) bool {
	s, n := string(b), len(sub)
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return true
		}
	}
	return false
}

// Align builds per-locus rater vectors from N compilers' assignment maps,
// keeping only loci present in EVERY compiler and excluding boundary loci
// (open texture is excluded from the κ denominator).
func Align(perCompiler []map[string]string) map[string][]string {
	if len(perCompiler) == 0 {
		return nil
	}
	ratings := map[string][]string{}
	for locus := range perCompiler[0] {
		vec := make([]string, 0, len(perCompiler))
		boundary, missing := false, false
		for _, m := range perCompiler {
			c, ok := m[locus]
			if !ok {
				missing = true
				break
			}
			if c == BoundaryCategory {
				boundary = true
			}
			vec = append(vec, c)
		}
		if missing || boundary {
			continue
		}
		ratings[locus] = vec
	}
	return ratings
}

// FleissKappa computes Fleiss' κ over the rating vectors (each subject rated by
// the same n ≥ 2 raters). Returns κ and the subject count used.
func FleissKappa(ratings map[string][]string) (kappa float64, subjects int) {
	var n int
	catSet := map[string]bool{}
	for _, rs := range ratings {
		if n == 0 {
			n = len(rs)
		}
		for _, c := range rs {
			catSet[c] = true
		}
	}
	if n < 2 {
		return 0, 0
	}
	cats := make([]string, 0, len(catSet))
	for c := range catSet {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	idx := map[string]int{}
	for i, c := range cats {
		idx[c] = i
	}

	var sumPi float64
	pj := make([]float64, len(cats))
	N := 0
	for _, rs := range ratings {
		if len(rs) != n {
			continue
		}
		N++
		counts := make([]int, len(cats))
		for _, c := range rs {
			counts[idx[c]]++
		}
		sq := 0
		for j, cnt := range counts {
			sq += cnt * cnt
			pj[j] += float64(cnt)
		}
		sumPi += (float64(sq) - float64(n)) / float64(n*(n-1))
	}
	if N == 0 {
		return 0, 0
	}
	pBar := sumPi / float64(N)
	var pe float64
	for j := range pj {
		p := pj[j] / float64(N*n)
		pe += p * p
	}
	if 1-pe == 0 { // all raters, all subjects, one category → perfect
		return 1, N
	}
	return (pBar - pe) / (1 - pe), N
}

// VerdictAgreement is the fraction of subjects on which all raters' verdicts
// coincide, over a shared event-trace suite.
func VerdictAgreement(perSubject map[string][]string) (ratio float64, subjects int) {
	agree, total := 0, 0
	for _, rs := range perSubject {
		if len(rs) < 2 {
			continue
		}
		total++
		all := true
		for _, r := range rs[1:] {
			if r != rs[0] {
				all = false
				break
			}
		}
		if all {
			agree++
		}
	}
	if total == 0 {
		return 1, 0
	}
	return float64(agree) / float64(total), total
}

// Falsification is a FALSIFICATION-CANDIDATE record: an input that appears to
// require extending the frozen kernel. It is a feature, not a bug — the unit
// halts and the kernel is left untouched (I3).
type Falsification struct {
	Unit   string
	Reason string
}

func (f *Falsification) Error() string {
	return fmt.Sprintf("FALSIFICATION-CANDIDATE [%s]: %s", f.Unit, f.Reason)
}

// closedOps is the sub-Turing algebra T's closed operator set (spec D1.3).
var closedOps = map[kernel.Op]bool{
	kernel.OpLit: true, kernel.OpVar: true, kernel.OpNot: true, kernel.OpAnd: true,
	kernel.OpOr: true, kernel.OpCmp: true, kernel.OpArith: true, kernel.OpLookup: true,
	kernel.OpPred: true, kernel.OpCount: true, kernel.OpWindow: true, kernel.OpRatio: true,
	kernel.OpBoundary: true,
}

// Screen inspects a proposed unit. It returns a *Falsification (and the unit
// must halt) when the input would require an 8th constructor or an operator
// outside the closed algebra T (e.g. unbounded quantification). nil = admissible.
func Screen(unit, constructor string, ast *kernel.Expr) *Falsification {
	if !kernel.Constructor(constructor).Valid() {
		return &Falsification{unit, fmt.Sprintf(
			"constructor %q is outside the closed basis B={NRM,CLS,PWR,GRD,REF,VAL} — would require an eighth constructor (I3)", constructor)}
	}
	if op := unknownOp(ast); op != "" {
		return &Falsification{unit, fmt.Sprintf(
			"term uses operator %q outside the sub-Turing algebra T — would require unbounded/undecidable expressiveness (I1/I3)", op)}
	}
	return nil
}

func unknownOp(e *kernel.Expr) string {
	if e == nil {
		return ""
	}
	if e.Op != "" && !closedOps[e.Op] {
		return string(e.Op)
	}
	for _, a := range e.Args {
		if op := unknownOp(a); op != "" {
			return op
		}
	}
	if e.Count != nil {
		if op := unknownOp(e.Count.Where); op != "" {
			return op
		}
	}
	if e.Window != nil {
		if op := unknownOp(e.Window.Body); op != "" {
			return op
		}
	}
	return ""
}

// LoadVerdicts reads a verdict NDJSON file: {"subject":..., "result":...}.
func LoadVerdicts(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var v struct {
			Subject string `json:"subject"`
			Result  string `json:"result"`
		}
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if v.Subject != "" {
			out[v.Subject] = v.Result
		}
	}
	return out, sc.Err()
}
