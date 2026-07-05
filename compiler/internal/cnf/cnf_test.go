package cnf

import (
	"testing"

	"computable-governance/compiler/internal/kernel"
)

// Commutativity: Canonicalize(A && B) == Canonicalize(B && A).
func TestCommutativity(t *testing.T) {
	a := kernel.Pred("a", kernel.Var("x"))
	b := kernel.Pred("b", kernel.Var("y"))
	if CanonicalHash(kernel.And(a, b)) != CanonicalHash(kernel.And(b, a)) {
		t.Fatalf("A && B and B && A produced different canonical hashes")
	}
	// Same for OR.
	if CanonicalHash(kernel.Or(a, b)) != CanonicalHash(kernel.Or(b, a)) {
		t.Fatalf("A || B and B || A produced different canonical hashes")
	}
}

// Idempotency: A && A == A ; A || A == A.
func TestIdempotency(t *testing.T) {
	a := kernel.Pred("a")
	if CanonicalHash(kernel.And(a, a)) != CanonicalHash(a) {
		t.Fatalf("A && A != A")
	}
	if CanonicalHash(kernel.Or(a, a)) != CanonicalHash(a) {
		t.Fatalf("A || A != A")
	}
}

// Double negation: !!A == A.
func TestDoubleNegation(t *testing.T) {
	a := kernel.Pred("a")
	if CanonicalHash(kernel.Not(kernel.Not(a))) != CanonicalHash(a) {
		t.Fatalf("!!A != A")
	}
	// Triple negation reduces to !A.
	if CanonicalHash(kernel.Not(kernel.Not(kernel.Not(a)))) != CanonicalHash(kernel.Not(a)) {
		t.Fatalf("!!!A != !A")
	}
}

// Constant folding: 1 + 2 -> 3 ; (1+2)*4 -> 12.
func TestArithFold(t *testing.T) {
	got := Canonicalize(kernel.Arith(kernel.ArithAdd, kernel.LitInt(1), kernel.LitInt(2)))
	if got.Op != kernel.OpLit || got.Lit == nil || got.Lit.Int == nil || *got.Lit.Int != 3 {
		t.Fatalf("1 + 2 did not fold to 3: %+v", got)
	}
	nested := kernel.Arith(kernel.ArithMul,
		kernel.Arith(kernel.ArithAdd, kernel.LitInt(1), kernel.LitInt(2)),
		kernel.LitInt(4))
	got = Canonicalize(nested)
	if got.Op != kernel.OpLit || got.Lit == nil || got.Lit.Int == nil || *got.Lit.Int != 12 {
		t.Fatalf("(1+2)*4 did not fold to 12: %+v", got)
	}
}

// Flattening: AND(AND(A,B),C) == AND(A,B,C).
func TestFlatten(t *testing.T) {
	a, b, c := kernel.Pred("a"), kernel.Pred("b"), kernel.Pred("c")
	nested := kernel.And(kernel.And(a, b), c)
	flat := kernel.And(a, b, c)
	if CanonicalHash(nested) != CanonicalHash(flat) {
		t.Fatalf("AND(AND(A,B),C) != AND(A,B,C)")
	}
	// Combined: commutativity + flattening + idempotency.
	messy := kernel.And(c, kernel.And(b, a), a)
	if CanonicalHash(messy) != CanonicalHash(flat) {
		t.Fatalf("AND(C, AND(B,A), A) did not normalize to AND(A,B,C)")
	}
}

// Determinism: canonicalizing twice yields the identical hash.
func TestStable(t *testing.T) {
	e := kernel.And(kernel.Pred("z"), kernel.Or(kernel.Pred("b"), kernel.Pred("a")))
	h1 := CanonicalHash(e)
	h2 := CanonicalHash(Canonicalize(e))
	if h1 != h2 {
		t.Fatalf("canonical hash not idempotent: %s vs %s", h1, h2)
	}
}
