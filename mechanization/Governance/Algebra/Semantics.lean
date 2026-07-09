import Governance.Algebra.AST

/-!
# Governance.Algebra.Semantics

A total, big-step evaluator for the Semantic Algebra `T` (mechanizing `D1.4 §2`)
together with the statement of the **Purity** theorem (Invariant **I1**).

The evaluator has type `Env → Expr → Value`. Its very signature encodes read-only
evaluation: it consumes an environment and produces a `Value`, and there is no
way for it to yield a modified `Env`. The `eval_is_pure` theorem below makes this
"no write-effect" property explicit against an instrumented evaluator.
-/

namespace Governance
namespace Algebra

/-- Read-only evaluation environment (`ρ` in `D1.4`).
    * `registry` — the versioned token table `R` (Invariant I4), read-only.
    * `bindings` — values of identifiers bound in the typing context `Γ`.
    * `trace`    — the finite, past event/state trace (read-only). -/
structure Env where
  registry : String → Option Value
  bindings : String → Option Value
  trace    : List String

/-- Interpret a `Value` as a boolean (non-booleans default to `false`). -/
def Value.asBool : Value → Bool
  | .VBool b => b
  | _        => false

/-- Interpret a `Value` as an integer (non-integers default to `0`). -/
def Value.asInt : Value → Int
  | .VInt n => n
  | _       => 0

/-- Denotation of comparison operators. -/
def CmpOp.denote : CmpOp → Int → Int → Bool
  | .lt, a, b => decide (a < b)
  | .le, a, b => decide (a ≤ b)
  | .eq, a, b => decide (a = b)
  | .ge, a, b => decide (a ≥ b)
  | .gt, a, b => decide (a > b)

/-- Denotation of bounded arithmetic operators. -/
def ArithOp.denote : ArithOp → Int → Int → Int
  | .add, a, b => a + b
  | .sub, a, b => a - b
  | .mul, a, b => a * b
  | .div, a, b => a / b

/--
Big-step evaluation `ρ ⊢ e ⇓ v` (`D1.4 §2`), realized as a total function.

Structural recursion over the finite term `e` guarantees termination; every
case is a pure read of the environment. No case updates `env`.
-/
def eval (env : Env) : Expr → Value
  | .lit v         => v
  | .var x         => (env.bindings x).getD (Value.VBool false)
  | .eAnd a b      => Value.VBool ((eval env a).asBool && (eval env b).asBool)
  | .eOr a b       => Value.VBool ((eval env a).asBool || (eval env b).asBool)
  | .eNot a        => Value.VBool (!(eval env a).asBool)
  | .eCmp op a b   => Value.VBool (op.denote (eval env a).asInt (eval env b).asInt)
  | .eArith op a b => Value.VInt  (op.denote (eval env a).asInt (eval env b).asInt)
  | .eLookup tok   => (env.registry tok).getD (Value.VBool false)
  | .ePredicate p  => Value.VBool (env.trace.contains p)

/-- Evaluator instrumented to also surface the environment it "returns".
    A pure evaluator threads the environment through **unchanged**; a mutating
    evaluator would return some `env' ≠ env` here. -/
def evalWithEnv (env : Env) (e : Expr) : Value × Env :=
  (eval env e, env)

/--
**T2 — Purity (Invariant I1).** Evaluating any term of `T` has empty
write-effect: the environment produced is *identical* to the environment
received. Hence `eval` returns a `Value` and never a mutated `Env`.

Mechanizes `D1.3 Prop. 3.1` / `D1.5 §T2`. Discharged by `rfl`:
`(evalWithEnv env e).2` reduces definitionally to `env`.
-/
theorem eval_is_pure (env : Env) (e : Expr) :
    (evalWithEnv env e).2 = env := rfl

/--
**T3 — Determinism (Invariant I8).** Evaluation is a (total) function of
⟨environment, term⟩: any two results of evaluating the same term in the same
environment coincide. This is the functional form of the D1.4 §5 confluence
theorem (there is no nondeterministic rule selection in `eval`).

Mechanizes `D1.5 §T3`.
-/
theorem eval_deterministic (env : Env) (e : Expr) {v₁ v₂ : Value}
    (h₁ : eval env e = v₁) (h₂ : eval env e = v₂) : v₁ = v₂ :=
  h₁.symm.trans h₂

end Algebra
end Governance
