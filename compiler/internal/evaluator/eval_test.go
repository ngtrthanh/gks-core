package evaluator

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"computable-governance/compiler/internal/kernel"
)

func env() Environment {
	return Environment{
		Now:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Vars:       map[string]Value{"x": VInt(3)},
		Registry:   map[string]Value{"threshold": VInt(5)},
		Predicates: map[string]bool{"granted": true},
		Domains:    map[string][]Fact{},
	}
}

func TestEvalBasics(t *testing.T) {
	cases := []struct {
		name string
		e    *kernel.Expr
		want Value
	}{
		{"lit", kernel.LitInt(7), VInt(7)},
		{"var", kernel.Var("x"), VInt(3)},
		{"lookup", kernel.Lookup("threshold"), VInt(5)},
		{"pred", kernel.Pred("granted"), VBool(true)},
		{"cmp", kernel.Cmp(kernel.CmpLT, kernel.Var("x"), kernel.Lookup("threshold")), VBool(true)},
		{"arith", kernel.Arith(kernel.ArithAdd, kernel.LitInt(2), kernel.LitInt(2)), VInt(4)},
		{"and-short", kernel.And(kernel.LitBool(false), kernel.Boundary("OT-9", "never reached")), VBool(false)},
		{"or-short", kernel.Or(kernel.LitBool(true), kernel.Boundary("OT-9", "never reached")), VBool(true)},
		{"not", kernel.Not(kernel.LitBool(false)), VBool(true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.e, env())
			if err != nil {
				t.Fatalf("Eval: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestBoundaryIsTypedSignal(t *testing.T) {
	_, err := Eval(kernel.Boundary("OT-1", "appropriate"), env())
	if err == nil {
		t.Fatal("boundary token must not resolve to a definite value")
	}
	var b *BoundaryError
	if !errors.As(err, &b) {
		t.Fatalf("expected *BoundaryError, got %T: %v", err, err)
	}
	if b.Token != "OT-1" || b.Label != "appropriate" {
		t.Fatalf("boundary fields lost: %+v", b)
	}
	if !IsBoundary(err) {
		t.Fatal("IsBoundary must detect a BoundaryError")
	}
	if !IsBoundary(fmt.Errorf("resolve: guard g1: %w", err)) {
		t.Fatal("IsBoundary must detect a wrapped BoundaryError")
	}
	if IsBoundary(errors.New("genuine failure")) {
		t.Fatal("IsBoundary must reject ordinary errors")
	}
}

func TestCountBoundedQuantifier(t *testing.T) {
	e := env()
	e.Domains["complaints"] = []Fact{
		{Name: "complaint", Time: e.Now.AddDate(0, -1, 0)},
		{Name: "complaint", Time: e.Now.AddDate(-3, 0, 0)}, // outside P2Y window
		{Name: "audit", Time: e.Now},
	}
	// count(complaints where within P2Y: is-a complaint) >= 2  → false (only 1)
	expr := kernel.CountCmp("complaints",
		kernel.Within("P2Y", "", kernel.Pred("complaint")), kernel.CmpGE, 2)
	got, err := Eval(expr, e)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got.Bool() {
		t.Fatal("only one complaint falls inside the P2Y window; count >= 2 must be false")
	}
}
