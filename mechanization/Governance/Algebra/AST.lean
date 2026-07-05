/-!
# Governance.Algebra.AST

Abstract syntax for the Semantic Algebra `T`, mechanizing the EBNF of spec
`D1.3-Grammar.md`. The term algebra is **fixpoint-free** and **read-only**:
`Expr` is a finite inductive type with no constructor denoting assignment,
mutation, or any state write (Invariant **I1**). Because `Expr` is an ordinary
inductive type, every term is finite and well-founded, so structural recursion
over it terminates (supports the decidability obligation `D1.5 §T1`).
-/

namespace Governance
namespace Algebra

/-- Ground values. Corresponds to `Value` in `D1.1`
    (sorts `Bool`, `Val`/numeric, and string-carried `Tok`/`Iri`). -/
inductive Value where
  | VBool   : Bool → Value
  | VInt    : Int → Value
  | VString : String → Value
deriving Repr, DecidableEq, Inhabited

/-- Comparison operators (`CmpOp` in `D1.3 §2`). -/
inductive CmpOp where
  | lt | le | eq | ge | gt
deriving Repr, DecidableEq, Inhabited

/-- Bounded arithmetic operators (`Arith`/`Term` in `D1.3 §2`, `§5`). -/
inductive ArithOp where
  | add | sub | mul | div
deriving Repr, DecidableEq, Inhabited

/--
Well-formed terms of `T`. This mirrors the `Expr` productions of `D1.3 §2`.

Every constructor is a *reader*: it inspects literals, bound identifiers, the
read-only registry, or read-only state/event predicates. There is deliberately
**no** constructor for `:=`, `set`, `write`, or `emit` — the read-only property
(Invariant I1) therefore holds by construction, mechanizing `D1.3 Prop. 3.1`.
-/
inductive Expr where
  /-- Value literal. -/
  | lit        : Value → Expr
  /-- Bound identifier lookup (variable in context `Γ`). -/
  | var        : String → Expr
  /-- Boolean conjunction (`AndExpr`). -/
  | eAnd       : Expr → Expr → Expr
  /-- Boolean disjunction (`OrExpr`). -/
  | eOr        : Expr → Expr → Expr
  /-- Boolean negation (`NotExpr`). -/
  | eNot       : Expr → Expr
  /-- Comparison relation (`Rel` via `CmpOp`). -/
  | eCmp       : CmpOp → Expr → Expr → Expr
  /-- Bounded arithmetic (`Arith`). -/
  | eArith     : ArithOp → Expr → Expr → Expr
  /-- Read-only registry lookup `reg(#token)` (`Lookup`). -/
  | eLookup    : String → Expr
  /-- Read-only state/event predicate `holds(_)` / `event(_)` (`Predicate`). -/
  | ePredicate : String → Expr
deriving Repr, DecidableEq, Inhabited

end Algebra
end Governance
