import Governance.Algebra.AST

/-!
# Governance.Algebra.Typing

A **total, decidable** static analysis for the Semantic Algebra `T` (mechanizing
the D1.2 typing judgment over the `Expr` fragment). Together with
`eval_terminates` (Semantics), this discharges **D1.5 §T1** — D0 Assumption 4,
*static analyzability* — for the mechanized fragment:

* typing is **decidable** (`typing_decidable`): the analysis always terminates
  with a definite yes/no on well-typedness;
* sorts are **unique** (`typing_unique`): the D1.2 "Uniqueness of Sorts"
  metatheorem.

The analysis is a *total structural function* `infer`, so it cannot diverge —
mirroring the fixpoint-free, sub-Turing character of `T` (Invariant I1/I7).
-/

namespace Governance
namespace Algebra

/-- Static sorts for the `Expr` fragment — the subset of the D1.2 sort grammar
    inhabited by `Value` (`Bool`, numeric `Val`, string-carried `Tok`/`Iri`). -/
inductive Ty where
  | TBool
  | TInt
  | TString
deriving DecidableEq, Repr, Inhabited

/-- The sort of a ground value. -/
def Value.ty : Value → Ty
  | .VBool _   => .TBool
  | .VInt _    => .TInt
  | .VString _ => .TString

/-- Variable typing context `Γ` (partial map from identifiers to sorts). -/
abbrev Ctx := String → Option Ty

/--
Total structural sort inference — the functional form of the D1.2 judgment
`Γ ⊢ e : τ`. A term is well-typed iff `infer` returns `some τ`. Every case is a
finite structural read; there is no recursion that is not on a strict subterm,
so `infer` is total (accepted by the equation compiler as a complete, gap-free
definition — no unproven obligation).
-/
def infer (Γ : Ctx) : Expr → Option Ty
  | .lit v        => some v.ty
  | .var x        => Γ x
  | .eAnd a b     =>
      match infer Γ a, infer Γ b with
      | some .TBool, some .TBool => some .TBool
      | _, _ => none
  | .eOr a b      =>
      match infer Γ a, infer Γ b with
      | some .TBool, some .TBool => some .TBool
      | _, _ => none
  | .eNot a       =>
      match infer Γ a with
      | some .TBool => some .TBool
      | _ => none
  | .eCmp _ a b   =>
      match infer Γ a, infer Γ b with
      | some .TInt, some .TInt => some .TBool
      | _, _ => none
  | .eArith _ a b =>
      match infer Γ a, infer Γ b with
      | some .TInt, some .TInt => some .TInt
      | _, _ => none
  | .eLookup _    => some .TBool
  | .ePredicate _ => some .TBool

/-- Typing judgment `Γ ⊢ e : t`, as the graph of `infer`. -/
def HasType (Γ : Ctx) (e : Expr) (t : Ty) : Prop := infer Γ e = some t

/--
**Uniqueness of Sorts** (D1.2 metatheorem). A term has at most one sort: if
`Γ ⊢ e : t₁` and `Γ ⊢ e : t₂` then `t₁ = t₂`.
-/
theorem typing_unique {Γ : Ctx} {e : Expr} {t₁ t₂ : Ty}
    (h₁ : HasType Γ e t₁) (h₂ : HasType Γ e t₂) : t₁ = t₂ := by
  unfold HasType at h₁ h₂
  rw [h₁] at h₂
  exact Option.some.inj h₂

/--
**T1 (typing-decidability leg).** Whether a term is well-typed in a context is
decidable — the static analysis always terminates with a definite answer
(D0 Assumption 4 / D1.5 §T1, over the mechanized `Expr` fragment).
-/
instance typing_decidable (Γ : Ctx) (e : Expr) :
    Decidable (∃ t : Ty, HasType Γ e t) := by
  simp only [HasType]
  cases infer Γ e with
  | none   => exact isFalse (by rintro ⟨t, ht⟩; simp at ht)
  | some t => exact isTrue ⟨t, rfl⟩

end Algebra
end Governance
