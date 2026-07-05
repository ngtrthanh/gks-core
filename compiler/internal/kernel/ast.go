package kernel

// Semantic Algebra T — abstract syntax (spec D1.3), JSONB-serializable.
//
// The AST is read-only and sub-Turing by construction: there are no assignment
// nodes, no lambda/abstraction, no recursion operators, and quantification is
// only ever *bounded* (see Count). Every node serializes to a compact JSONB
// object discriminated by its `op` field, so it round-trips through the
// `payload` column of `kernel_instance`.

// Op is the node discriminator of a T expression.
type Op string

const (
	OpLit      Op = "lit"      // literal value
	OpVar      Op = "var"      // bound identifier (typing context Γ)
	OpNot      Op = "not"      // boolean negation
	OpAnd      Op = "and"      // boolean conjunction
	OpOr       Op = "or"       // boolean disjunction
	OpCmp      Op = "cmp"      // comparison relation
	OpArith    Op = "arith"    // bounded arithmetic
	OpLookup   Op = "lookup"   // reg(token) — read-only registry access
	OpPred     Op = "pred"     // holds(state)/event(name) — read-only predicate
	OpCount    Op = "count"    // bounded quantifier
	OpWindow   Op = "window"   // past-time temporal window
	OpBoundary Op = "boundary" // open-texture token (D0 §7): opaque, unresolved predicate
)

// Comparison operators (D1.3 §2, CmpOp).
const (
	CmpLT = "<"
	CmpLE = "<="
	CmpEQ = "="
	CmpGE = ">="
	CmpGT = ">"
)

// Arithmetic operators (D1.3 §2/§5, bounded).
const (
	ArithAdd = "+"
	ArithSub = "-"
	ArithMul = "*"
	ArithDiv = "/"
)

// Lit is a ground literal; exactly one field is non-nil.
type Lit struct {
	Bool *bool   `json:"bool,omitempty"`
	Int  *int64  `json:"int,omitempty"`
	Str  *string `json:"str,omitempty"`
}

// Count is a BOUNDED quantifier: `count(domain where Where) Cmp Bound`.
// It counts elements of a finite domain satisfying Where and compares the
// tally against a static Bound. Bounded by construction — this is how T
// expresses "aggregating N or more" / "no more than one" without admitting
// unbounded quantification (Invariant I1, sub-Turing grammar).
type Count struct {
	Domain string `json:"domain"`          // finite collection identifier
	Where  *Expr  `json:"where,omitempty"` // filter predicate over the domain
	Cmp    string `json:"cmp"`             // comparator against Bound
	Bound  int64  `json:"bound"`           // static numeric bound
}

// Window is a past-time temporal operator over a finite lookback window
// (D1.3 TemporalPast). Lower/Upper are ISO-8601 durations measured back from
// evaluation time (e.g. "P5Y" = five years). Op ∈ {within, once, since, prev}.
type Window struct {
	Op    string `json:"op"`
	Lower string `json:"lower,omitempty"` // start of window, back from now (e.g. "P5Y")
	Upper string `json:"upper,omitempty"` // end of window (empty = now)
	Body  *Expr  `json:"body"`
}

// Expr is a node of the T AST.
type Expr struct {
	Op     Op      `json:"op"`
	Cmp    string  `json:"cmp,omitempty"`
	Arith  string  `json:"arith,omitempty"`
	Name   string  `json:"name,omitempty"`  // var / lookup token / predicate / boundary token
	Label  string  `json:"label,omitempty"` // human label for a boundary (open-texture) token
	Lit    *Lit    `json:"lit,omitempty"`
	Args   []*Expr `json:"args,omitempty"`
	Count  *Count  `json:"count,omitempty"`
	Window *Window `json:"window,omitempty"`
}

// --- Smart constructors (ergonomic AST building) ---------------------------

// LitBool builds a boolean literal.
func LitBool(v bool) *Expr { return &Expr{Op: OpLit, Lit: &Lit{Bool: &v}} }

// LitInt builds an integer literal.
func LitInt(v int64) *Expr { return &Expr{Op: OpLit, Lit: &Lit{Int: &v}} }

// LitStr builds a string literal.
func LitStr(v string) *Expr { return &Expr{Op: OpLit, Lit: &Lit{Str: &v}} }

// Var references a bound identifier.
func Var(name string) *Expr { return &Expr{Op: OpVar, Name: name} }

// Lookup is a read-only registry access reg(token).
func Lookup(token string) *Expr { return &Expr{Op: OpLookup, Name: token} }

// Pred is a read-only state/event predicate applied to arguments.
func Pred(name string, args ...*Expr) *Expr {
	return &Expr{Op: OpPred, Name: name, Args: args}
}

// Not negates a boolean expression.
func Not(e *Expr) *Expr { return &Expr{Op: OpNot, Args: []*Expr{e}} }

// And / Or are variadic boolean connectives.
func And(es ...*Expr) *Expr { return &Expr{Op: OpAnd, Args: es} }
func Or(es ...*Expr) *Expr  { return &Expr{Op: OpOr, Args: es} }

// Cmp builds a comparison relation `l <op> r`.
func Cmp(op string, l, r *Expr) *Expr {
	return &Expr{Op: OpCmp, Cmp: op, Args: []*Expr{l, r}}
}

// Arith builds a bounded arithmetic term `l <op> r`.
func Arith(op string, l, r *Expr) *Expr {
	return &Expr{Op: OpArith, Arith: op, Args: []*Expr{l, r}}
}

// CountAtLeast / CountAtMost / CountCmp build bounded quantifiers.
func CountCmp(domain string, where *Expr, cmp string, bound int64) *Expr {
	return &Expr{Op: OpCount, Count: &Count{Domain: domain, Where: where, Cmp: cmp, Bound: bound}}
}

// Within wraps a body in a bounded past-time window [now-lower, now-upper].
func Within(lower, upper string, body *Expr) *Expr {
	return &Expr{Op: OpWindow, Window: &Window{Op: "within", Lower: lower, Upper: upper, Body: body}}
}

// Boundary builds an open-texture boundary token (D0 §7): an opaque predicate
// whose extension is deliberately deferred to future adjudication. `token` is a
// stable identifier (e.g. "OT-1"); `label` is the source term (e.g. "appropriate").
// The evaluator must treat it as unresolved and emit only conditional verdicts.
func Boundary(token, label string) *Expr {
	return &Expr{Op: OpBoundary, Name: token, Label: label}
}
