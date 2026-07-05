// Package cnf implements the Canonical Normal Form pass (π_4) over the Semantic
// Algebra T AST: a strictly deterministic normalization used for exact syntactic
// diffing of verdicts (Invariant I8, reproducibility).
//
// Canonicalization is pointer-/address-independent: it depends only on node
// content. Ordering of commutative operands is by a stable content hash.
package cnf

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"computable-governance/compiler/internal/kernel"
)

// canonicalMarshal serializes v deterministically and without HTML-escaping.
// The kernel.Expr tree is pointer-free in content (struct field order is fixed,
// slices are ordered), so the encoding depends only on the tree's content.
func canonicalMarshal(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sortKey hashes an assumed-canonical node, used for stable operand ordering.
func sortKey(e *kernel.Expr) string { return hashHex(canonicalMarshal(e)) }

// CanonicalJSON returns the canonical JSON encoding of Canonicalize(e).
func CanonicalJSON(e *kernel.Expr) []byte { return canonicalMarshal(Canonicalize(e)) }

// CanonicalHash returns the SHA-256 (hex) of the canonical JSON — the stable
// "Verdict Identifier".
func CanonicalHash(e *kernel.Expr) string { return hashHex(CanonicalJSON(e)) }

func isIntLit(e *kernel.Expr) bool {
	return e != nil && e.Op == kernel.OpLit && e.Lit != nil && e.Lit.Int != nil
}

func foldArith(op string, a, b int64) (int64, bool) {
	switch op {
	case kernel.ArithAdd:
		return a + b, true
	case kernel.ArithSub:
		return a - b, true
	case kernel.ArithMul:
		return a * b, true
	case kernel.ArithDiv:
		if b == 0 {
			return 0, false
		}
		return a / b, true
	}
	return 0, false
}

// Canonicalize recursively normalizes a T expression:
//   - children canonicalized first (bottom-up);
//   - double negation:      !!A            -> A
//   - constant folding:     1 + 2          -> 3
//   - flattening:           AND(AND(A,B),C)-> AND(A,B,C)   (same for OR)
//   - idempotency:          A && A         -> A            (same for OR)
//   - commutativity:        AND/OR operands sorted by content hash
//   - collapse:             AND(A)         -> A
//
// The input is never mutated (a shallow copy is taken per node).
func Canonicalize(e *kernel.Expr) *kernel.Expr {
	if e == nil {
		return nil
	}
	n := *e // shallow copy; we rebuild Args/Count/Window below

	if len(e.Args) > 0 {
		args := make([]*kernel.Expr, len(e.Args))
		for i, a := range e.Args {
			args[i] = Canonicalize(a)
		}
		n.Args = args
	}
	if e.Count != nil {
		c := *e.Count
		c.Where = Canonicalize(e.Count.Where)
		n.Count = &c
	}
	if e.Window != nil {
		w := *e.Window
		w.Body = Canonicalize(e.Window.Body)
		n.Window = &w
	}

	switch n.Op {
	case kernel.OpNot:
		if len(n.Args) == 1 {
			if in := n.Args[0]; in != nil && in.Op == kernel.OpNot && len(in.Args) == 1 {
				return in.Args[0] // !!A -> A (already canonical)
			}
		}

	case kernel.OpArith:
		if len(n.Args) == 2 && isIntLit(n.Args[0]) && isIntLit(n.Args[1]) {
			if v, ok := foldArith(n.Arith, *n.Args[0].Lit.Int, *n.Args[1].Lit.Int); ok {
				return kernel.LitInt(v)
			}
		}

	case kernel.OpAnd, kernel.OpOr:
		// Flatten nested same-op nodes.
		flat := make([]*kernel.Expr, 0, len(n.Args))
		for _, a := range n.Args {
			if a != nil && a.Op == n.Op {
				flat = append(flat, a.Args...)
			} else {
				flat = append(flat, a)
			}
		}
		// Idempotency: drop content-duplicate operands.
		seen := make(map[string]bool, len(flat))
		uniq := make([]*kernel.Expr, 0, len(flat))
		for _, a := range flat {
			k := sortKey(a)
			if !seen[k] {
				seen[k] = true
				uniq = append(uniq, a)
			}
		}
		// Commutativity: stable content-hash order.
		sort.SliceStable(uniq, func(i, j int) bool { return sortKey(uniq[i]) < sortKey(uniq[j]) })
		if len(uniq) == 1 {
			return uniq[0]
		}
		n.Args = uniq
	}

	return &n
}
