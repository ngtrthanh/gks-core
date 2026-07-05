// Package evaluator implements the execution semantics of layer E-hat
// (spec D1.4): a read-only big-step evaluator over the Semantic Algebra T AST
// and the defeasible priority resolver.
//
// State (the Environment: event traces, facts, registry) is kept strictly
// separate from Knowledge (the kernel instances / AST). Eval never mutates the
// Environment — it only reads it (Invariant I1).
package evaluator

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"computable-governance/compiler/internal/kernel"
)

// BoundaryError signals that evaluation reached an open-texture boundary token
// (D0 §7). It is NOT an evaluation failure: the token is deliberately
// unresolved, and the caller must emit a CONDITIONAL verdict. Distinguish it
// from genuine evaluator bugs with errors.As / IsBoundary.
type BoundaryError struct {
	Token string // stable boundary identifier (e.g. "OT-1")
	Label string // source term (e.g. "appropriate")
}

func (e *BoundaryError) Error() string {
	return fmt.Sprintf("eval: open-texture token %q (%q) is unresolved; conditional verdict required", e.Token, e.Label)
}

// IsBoundary reports whether err (anywhere in its chain) is a BoundaryError,
// i.e. the correct outcome is a conditional verdict, not a failure.
func IsBoundary(err error) bool {
	var b *BoundaryError
	return errors.As(err, &b)
}

// Kind tags a runtime Value.
type Kind int

const (
	KBool Kind = iota
	KInt
	KStr
)

// Value is the result of evaluating a T expression.
type Value struct {
	Kind Kind
	B    bool
	I    int64
	S    string
}

func VBool(b bool) Value  { return Value{Kind: KBool, B: b} }
func VInt(i int64) Value  { return Value{Kind: KInt, I: i} }
func VStr(s string) Value { return Value{Kind: KStr, S: s} }

// Bool coerces to a boolean (only a true KBool is true).
func (v Value) Bool() bool { return v.Kind == KBool && v.B }

// Int coerces to an integer (bools map to 1/0).
func (v Value) Int() int64 {
	switch v.Kind {
	case KInt:
		return v.I
	case KBool:
		if v.B {
			return 1
		}
	}
	return 0
}

func (v Value) String() string {
	switch v.Kind {
	case KStr:
		return v.S
	case KInt:
		return strconv.FormatInt(v.I, 10)
	case KBool:
		return strconv.FormatBool(v.B)
	}
	return ""
}

// Fact is one element of an event trace / count domain. Time supports the
// bounded temporal windows of D1.3.
type Fact struct {
	Name  string
	Attrs map[string]string
	Time  time.Time
}

// Environment is the read-only evaluation state (ρ in D1.4): scalar variables,
// registry lookups, ground predicate truths, and finite domains for bounded
// quantifiers. `current` is the internal binding of the element under
// iteration inside a count().
type Environment struct {
	Now        time.Time
	Vars       map[string]Value
	Registry   map[string]Value
	Predicates map[string]bool
	Domains    map[string][]Fact

	current *Fact
}

// withCurrent returns a copy of env bound to a count-iteration element. The
// receiver is unchanged (no mutation).
func (env Environment) withCurrent(f Fact) Environment {
	env.current = &f
	return env
}

// Eval is the big-step evaluator ρ ⊢ e ⇓ v (D1.4 §2). It is total over
// well-formed ASTs and performs no writes.
func Eval(e *kernel.Expr, env Environment) (Value, error) {
	if e == nil {
		return VBool(false), fmt.Errorf("eval: nil expression")
	}
	switch e.Op {
	case kernel.OpLit:
		return evalLit(e.Lit)
	case kernel.OpVar:
		if v, ok := env.Vars[e.Name]; ok {
			return v, nil
		}
		return VBool(false), nil
	case kernel.OpLookup:
		if v, ok := env.Registry[e.Name]; ok {
			return v, nil
		}
		return VBool(false), nil
	case kernel.OpNot:
		v, err := Eval(arg(e, 0), env)
		if err != nil {
			return v, err
		}
		return VBool(!v.Bool()), nil
	case kernel.OpAnd:
		for _, a := range e.Args {
			v, err := Eval(a, env)
			if err != nil {
				return v, err
			}
			if !v.Bool() {
				return VBool(false), nil
			}
		}
		return VBool(true), nil
	case kernel.OpOr:
		for _, a := range e.Args {
			v, err := Eval(a, env)
			if err != nil {
				return v, err
			}
			if v.Bool() {
				return VBool(true), nil
			}
		}
		return VBool(false), nil
	case kernel.OpCmp:
		return evalCmp(e, env)
	case kernel.OpArith:
		return evalArith(e, env)
	case kernel.OpPred:
		return evalPred(e, env)
	case kernel.OpWindow:
		return evalWindow(e.Window, env)
	case kernel.OpCount:
		return evalCount(e.Count, env)
	case kernel.OpBoundary:
		// Open texture (D0 §7): the token is deliberately unresolved. Evaluation
		// cannot return a definite verdict — the caller must emit a conditional
		// verdict. We signal this rather than fabricating a boolean.
		return VBool(false), &BoundaryError{Token: e.Name, Label: e.Label}
	default:
		return VBool(false), fmt.Errorf("eval: unknown op %q", e.Op)
	}
}

func arg(e *kernel.Expr, i int) *kernel.Expr {
	if i < len(e.Args) {
		return e.Args[i]
	}
	return nil
}

func evalLit(l *kernel.Lit) (Value, error) {
	switch {
	case l == nil:
		return VBool(false), fmt.Errorf("eval: empty literal")
	case l.Bool != nil:
		return VBool(*l.Bool), nil
	case l.Int != nil:
		return VInt(*l.Int), nil
	case l.Str != nil:
		return VStr(*l.Str), nil
	}
	return VBool(false), fmt.Errorf("eval: malformed literal")
}

func evalCmp(e *kernel.Expr, env Environment) (Value, error) {
	l, err := Eval(arg(e, 0), env)
	if err != nil {
		return l, err
	}
	r, err := Eval(arg(e, 1), env)
	if err != nil {
		return r, err
	}
	if l.Kind == KStr || r.Kind == KStr {
		if e.Cmp == kernel.CmpEQ {
			return VBool(l.String() == r.String()), nil
		}
		return VBool(false), fmt.Errorf("eval: comparator %q not defined on strings", e.Cmp)
	}
	a, b := l.Int(), r.Int()
	switch e.Cmp {
	case kernel.CmpLT:
		return VBool(a < b), nil
	case kernel.CmpLE:
		return VBool(a <= b), nil
	case kernel.CmpEQ:
		return VBool(a == b), nil
	case kernel.CmpGE:
		return VBool(a >= b), nil
	case kernel.CmpGT:
		return VBool(a > b), nil
	}
	return VBool(false), fmt.Errorf("eval: unknown comparator %q", e.Cmp)
}

func evalArith(e *kernel.Expr, env Environment) (Value, error) {
	l, err := Eval(arg(e, 0), env)
	if err != nil {
		return l, err
	}
	r, err := Eval(arg(e, 1), env)
	if err != nil {
		return r, err
	}
	a, b := l.Int(), r.Int()
	switch e.Arith {
	case kernel.ArithAdd:
		return VInt(a + b), nil
	case kernel.ArithSub:
		return VInt(a - b), nil
	case kernel.ArithMul:
		return VInt(a * b), nil
	case kernel.ArithDiv:
		if b == 0 {
			return VInt(0), fmt.Errorf("eval: division by zero")
		}
		return VInt(a / b), nil
	}
	return VInt(0), fmt.Errorf("eval: unknown arithmetic operator %q", e.Arith)
}

// evalPred reads a predicate. Inside a count() iteration a predicate identifies
// the KIND the current domain element must match; standalone, it reads the
// ground predicate truth supplied by the scenario.
func evalPred(e *kernel.Expr, env Environment) (Value, error) {
	if env.current != nil {
		return VBool(env.current.Name == e.Name), nil
	}
	return VBool(env.Predicates[e.Name]), nil
}

// evalWindow evaluates a bounded past-time window. When bound to a current
// element, the element's timestamp must fall within [Now-Lower, Now].
func evalWindow(w *kernel.Window, env Environment) (Value, error) {
	if w == nil {
		return VBool(false), fmt.Errorf("eval: nil window")
	}
	inWindow := true
	if env.current != nil {
		cutoff, err := windowCutoff(w, env.Now)
		if err != nil {
			return VBool(false), err
		}
		t := env.current.Time
		inWindow = !t.Before(cutoff) && !t.After(env.Now)
	}
	body, err := Eval(w.Body, env)
	if err != nil {
		return body, err
	}
	return VBool(inWindow && body.Bool()), nil
}

func windowCutoff(w *kernel.Window, now time.Time) (time.Time, error) {
	y, m, d, err := parseISODuration(w.Lower)
	if err != nil {
		return now, err
	}
	return now.AddDate(-y, -m, -d), nil
}

// evalCount is the bounded quantifier: tally the domain elements satisfying
// Where, then compare against Bound.
func evalCount(c *kernel.Count, env Environment) (Value, error) {
	if c == nil {
		return VBool(false), fmt.Errorf("eval: nil count")
	}
	var n int64
	for _, f := range env.Domains[c.Domain] {
		v, err := Eval(c.Where, env.withCurrent(f))
		if err != nil {
			return VBool(false), err
		}
		if v.Bool() {
			n++
		}
	}
	switch c.Cmp {
	case kernel.CmpLT:
		return VBool(n < c.Bound), nil
	case kernel.CmpLE:
		return VBool(n <= c.Bound), nil
	case kernel.CmpEQ:
		return VBool(n == c.Bound), nil
	case kernel.CmpGE:
		return VBool(n >= c.Bound), nil
	case kernel.CmpGT:
		return VBool(n > c.Bound), nil
	}
	return VBool(false), fmt.Errorf("eval: unknown count comparator %q", c.Cmp)
}

// parseISODuration parses the date portion of an ISO-8601 duration (e.g. "P2Y",
// "P5Y", "P18M", "P90D", "P1Y6M"). Time components after 'T' are ignored.
func parseISODuration(s string) (years, months, days int, err error) {
	if len(s) == 0 || s[0] != 'P' {
		return 0, 0, 0, fmt.Errorf("eval: invalid ISO-8601 duration %q", s)
	}
	var num strings.Builder
	for _, r := range s[1:] {
		switch {
		case r >= '0' && r <= '9':
			num.WriteRune(r)
		case r == 'Y' || r == 'M' || r == 'W' || r == 'D':
			if num.Len() == 0 {
				return 0, 0, 0, fmt.Errorf("eval: malformed duration %q", s)
			}
			n, _ := strconv.Atoi(num.String())
			num.Reset()
			switch r {
			case 'Y':
				years += n
			case 'M':
				months += n
			case 'W':
				days += 7 * n
			case 'D':
				days += n
			}
		case r == 'T':
			return years, months, days, nil
		default:
			return 0, 0, 0, fmt.Errorf("eval: unsupported unit %q in duration %q", string(r), s)
		}
	}
	return years, months, days, nil
}
