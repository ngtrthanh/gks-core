/-!
# Governance.Kernel.Invariants

Structural kernel-invariant obligations from `spec/D1.5`. These proofs are
mathlib-free (they concern our own inductive model), so `lake build` needs only
the Lean toolchain — no mathlib download.

* **T6 / I2** — append-only monotonicity of K̂: PROVED (`append_only_monotone`).
* **T8 / I7** — well-foundedness of the stratification order: PROVED
  (`strata_wellFounded`, via the Lean-core instance `Nat.lt_wfRel.wf`).

`T1` (decidability of `T`) and `C1` (minimality of ⟨B, T⟩) are research-grade
and are NOT stated as trivial placeholders here — they remain conjectures in
`spec/D1.5` (C1 is targeted empirically by the falsification campaign, Track C).
-/

namespace Governance
namespace Kernel

/-- K̂ modeled as a finite, append-only log of instances (opaque ids). -/
abbrev KB := List Nat

/-- The single admissible mutation of K̂ (Invariant I2): append one instance.
    Ê never deletes or overwrites — so extension is monotone by construction. -/
def extend (k : KB) (x : Nat) : KB := k ++ [x]

/--
**T6 — Append-only monotonicity (Invariant I2).** Every instance already in K̂
remains present after an Ê extension: `k ⊆ extend k x`. The sole writer (Ê) can
only grow the store; it never deletes or overwrites.
-/
theorem append_only_monotone (k : KB) (x i : Nat) (h : i ∈ k) :
    i ∈ extend k x := by
  simp [extend, h]

/--
**T8 — Stratified-reflection well-foundedness (Invariant I7).** Schema-level
matching strictly decreases the stratum (a `Nat`), so the match relation is
well-founded and reflection terminates. Discharged by the Lean-core instance
`Nat.lt_wfRel.wf : WellFounded Nat.lt` (the `<`-relation on `Nat` used by the
termination checker); no mathlib required.
-/
theorem strata_wellFounded : WellFounded (fun a b : Nat => a < b) :=
  Nat.lt_wfRel.wf

end Kernel
end Governance
