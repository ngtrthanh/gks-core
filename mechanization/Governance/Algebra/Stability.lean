import Governance.Algebra.Semantics

/-!
# Governance.Algebra.Stability

Two invariance metatheorems about evaluation, mechanizing D1.5 §T4 and §T5:

* **T4 / I4 — Registry rename-stability.** Token identities in the registry are
  *inert*: renaming every token by an injective σ (consistently in the term and
  the registry) does not change any verdict. Proved by induction on `Expr`.
* **T5 / I5 — Presentation erasure.** The verdict of a term is a function of the
  term alone; the presentation `P̂` (surface text, locus, cue, …) carries no
  semantic weight. Proved by `rfl` — presentation is not even an argument to
  evaluation.
-/

namespace Governance
namespace Algebra

/-- Apply a token renaming σ to every registry lookup `reg(#tok)` in a term. -/
def renameExpr (σ : String → String) : Expr → Expr
  | .lit v        => .lit v
  | .var x        => .var x
  | .eAnd a b     => .eAnd (renameExpr σ a) (renameExpr σ b)
  | .eOr a b      => .eOr (renameExpr σ a) (renameExpr σ b)
  | .eNot a       => .eNot (renameExpr σ a)
  | .eCmp o a b   => .eCmp o (renameExpr σ a) (renameExpr σ b)
  | .eArith o a b => .eArith o (renameExpr σ a) (renameExpr σ b)
  | .eLookup tok  => .eLookup (σ tok)
  | .ePredicate p => .ePredicate p

/-- Rename the registry so that the renamed token `σ t` resolves to whatever `t`
    resolved to before (via a left inverse `σinv`). Bindings and trace are the
    non-registry parts of `ρ` and are untouched. -/
def renameEnv (σinv : String → String) (env : Env) : Env :=
  { env with registry := fun t => env.registry (σinv t) }

/--
**T4 — Registry rename-stability (Invariant I4).** If `σinv` is a left inverse of
`σ`, then evaluating the σ-renamed term against the σ-renamed registry yields the
same value as the original: registry token identities are inert.
-/
theorem eval_rename_stable (σ σinv : String → String)
    (hinv : ∀ t, σinv (σ t) = t) (env : Env) (e : Expr) :
    eval (renameEnv σinv env) (renameExpr σ e) = eval env e := by
  induction e with
  | lit v          => simp [eval, renameExpr]
  | var x          => simp [eval, renameExpr, renameEnv]
  | eAnd a b ia ib => simp [eval, renameExpr, ia, ib]
  | eOr a b ia ib  => simp [eval, renameExpr, ia, ib]
  | eNot a ia      => simp [eval, renameExpr, ia]
  | eCmp o a b ia ib   => simp [eval, renameExpr, ia, ib]
  | eArith o a b ia ib => simp [eval, renameExpr, ia, ib]
  | eLookup tok    => simp [eval, renameExpr, renameEnv, hinv]
  | ePredicate p   => simp [eval, renameExpr, renameEnv]

/-- A term carried together with its presentation `P̂` (an opaque surface
    rendering: source text, locus, modality cue, …). -/
structure Presented where
  expr         : Expr
  presentation : String

/-- The verdict of a presented term — a function of its `expr` only. -/
def verdict (env : Env) (p : Presented) : Value := eval env p.expr

/--
**T5 — Presentation erasure (Invariant I5).** Two presented terms with the same
`expr` but ANY presentations yield the same verdict; equivalently, erasing or
replacing the presentation never changes the verdict. Holds by `rfl` because
`verdict` never reads the presentation.
-/
theorem verdict_erases_presentation (env : Env) (e : Expr) (p q : String) :
    verdict env ⟨e, p⟩ = verdict env ⟨e, q⟩ := rfl

end Algebra
end Governance
