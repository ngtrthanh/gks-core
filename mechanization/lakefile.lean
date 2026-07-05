import Lake
open Lake DSL

/-!
Governance Computing — Phase 2 (Mechanized Semantics).

Lean 4 package mechanizing the Semantic Algebra `T` (spec `D1.3`) and its
proof obligations (spec `D1.5`). The library root is `Governance.lean`, which
re-exports every submodule under `Governance/`.
-/

package «Governance» where
  leanOptions := #[
    ⟨`autoImplicit, false⟩,          -- require explicit binders (strict typing)
    ⟨`relaxedAutoImplicit, false⟩
  ]

@[default_target]
lean_lib «Governance» where
  -- Include all submodules under `Governance/` (AST, Semantics, ...).
  globs := #[.submodules `Governance]

lean_exe «governance» where
  root := `Main
