import Governance.Kernel.Invariants

/-!
# Governance.Kernel.Transition

A Lean model of the **D1.4 §3 small-step transition system**
`⟨K̂, θ⟩ --η--> ⟨K̂', θ'⟩`, built so the kernel invariants are proved as *real*
theorems about the transition — by case analysis over the actual rules — rather
than as definitional identities.

**Honest scope (exit-review F1).** This is a *model* of `Step`: a `Config` carries
an append-only `K̂` of indexed instances and a lifecycle map `θ`. It is NOT the Go
implementation (correspondence to the running compiler is via the Go tests), and
`Step` is modelled as a total function, so determinism (T3) reflects the transition
system being **nondeterminism-free by construction** (rule selection is by the
event's constructor). It is nonetheless a genuine upgrade over the prior `List Nat`
lemmas: T6 (monotonicity) and T7 (index-totality) now require case analysis over a
real ⟨K̂, θ⟩ transition. The full correspondence proof (extraction/refinement to the
Go resolver, including its priority + content-key tie-break) remains open.
-/

namespace Governance
namespace Kernel

/-- Lifecycle states (D1.1 Axiom 1.3, amended A03). -/
inductive LState where
  | proposed | inforce | suspended | violated | discharged | extinguished
deriving DecidableEq, Repr

/-- A kernel instance: an opaque id and its bitemporal index τ (D1.1 §3). -/
structure Inst where
  id  : Nat
  tix : TIX

/-- θ — the lifecycle-state map of a configuration (D1.4 §1). -/
abbrev Theta := Nat → LState

/-- A configuration `⟨K̂, θ⟩` (D1.4 §1). -/
structure Config where
  kb : List Inst
  th : Theta

/-- Events, tagged by constructor so each triggers exactly one rule — the source
    of the transition system's determinism (D1.4 §5). -/
inductive Event where
  | activate  (n : Nat)          -- S-Activate: norm n enters force
  | violate   (n : Nat)          -- S-Violate:  norm n is breached
  | discharge (n : Nat)          -- S-Discharge: norm n is satisfied/closed
  | exercise  (n : Nat) (i : Inst) -- S-Exercise: append operand i, n in force

/-- Point update of θ. -/
def setθ (th : Theta) (n : Nat) (s : LState) : Theta :=
  fun m => if m = n then s else th m

/-- The small-step transition as a total function (D1.4 §3). K̂ is append-only;
    only `exercise` extends it (with the operand instance). -/
def step (c : Config) : Event → Config
  | .activate n   => { c with th := setθ c.th n .inforce }
  | .violate n    => { c with th := setθ c.th n .violated }
  | .discharge n  => { c with th := setθ c.th n .discharged }
  | .exercise n i => { kb := c.kb ++ [i], th := setθ c.th n .inforce }

/--
**T6 — Append-only monotonicity (Invariant I2).** No step removes an instance from
K̂: every instance present before a step is present after it. Proved by case
analysis over the four rules (three keep K̂; `exercise` appends).
-/
theorem step_monotone (c : Config) (η : Event) {x : Inst} (h : x ∈ c.kb) :
    x ∈ (step c η).kb := by
  cases η with
  | activate n   => exact h
  | violate n    => exact h
  | discharge n  => exact h
  | exercise n i => exact List.mem_append_left _ h

/-- Totality of the bitemporal index over a whole configuration (Invariant I6). -/
def AllValidC (c : Config) : Prop := ∀ i ∈ c.kb, i.tix.valid

/-- A well-formed event: any instance it appends carries a valid index. -/
def validEvent : Event → Prop
  | .exercise _ i => i.tix.valid
  | _             => True

/--
**T7 — Bitemporal totality (Invariant I6).** A well-formed step preserves
index-totality: if every instance in K̂ has a valid τ and the event appends only a
valid-τ instance, then every instance in K̂' still has a valid τ. Proved by case
analysis (the appended element is discharged by the event's well-formedness).
-/
theorem step_tix_preserved (c : Config) (η : Event)
    (hc : AllValidC c) (hη : validEvent η) : AllValidC (step c η) := by
  intro x hx
  cases η with
  | activate n   => exact hc x hx
  | violate n    => exact hc x hx
  | discharge n  => exact hc x hx
  | exercise n i =>
      simp only [step, List.mem_append, List.mem_singleton] at hx
      rcases hx with h | h
      · exact hc x h
      · subst h; exact hη

/-- The transition relation induced by the `step` function. -/
def StepRel (c : Config) (η : Event) (c' : Config) : Prop := step c η = c'

/--
**T3 — Determinism (Invariant I8).** The transition is a total function, hence the
induced relation is deterministic: an event admits at most one result. The genuine
content is *nondeterminism-freedom* — rule selection is by the event's constructor,
so no two rules compete (cf. the Go resolver, whose remaining choice, equal-priority
guard order, is fixed by the content-key tie-break, D1.4 §5 / M6).
-/
theorem step_deterministic {c : Config} {η : Event} {c₁ c₂ : Config}
    (h₁ : StepRel c η c₁) (h₂ : StepRel c η c₂) : c₁ = c₂ := by
  unfold StepRel at h₁ h₂
  exact h₁.symm.trans h₂

end Kernel
end Governance
