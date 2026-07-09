# PROGRESS REPORT — gks-core vs. the D0 Maturity Roadmap

**Date:** 2026-07-09 UTC (current through tag `v0.6.0`) · **Author:** Kiro (Engineering)
**Baselines:** `spec/D0v5.md` (D0 v1.1, FROZEN), `spec/D1.1–D1.5`,
`handoff.md`/`handoff2.md`/`handoff3.md`, `CHANGELOG.md`.
**Verification basis:** `go build/vet/test ./...` green; live dev DB (Postgres 18,
port 5435); `make validate` PASS; `cmd/interop`, `cmd/falsify`, `cmd/trackd` run;
invariant suite incl. I5-erasure, I7-well-foundedness, store-wide screen, and the
multi-domain Registry Law all pass; the Lean mechanization **compiles in CI**
(`.github/workflows/lean.yml`, `lake build` on Lean 4.31.0, zero `sorry`).

> One-line status: **Phase 3 (industrial compiler) built and conformance-hardened;
> Phase 2's tractable proof set is now CI-compiled** — T2 (I1), T3 (I8), T6 (I2)
> and T8 (I7) type-check in GitHub Actions (Lean 4.31.0, zero `sorry`). The kernel
> remains provisionally intact (no falsification triggered) but NOT yet *confirmed*.
> Remaining open items are research-grade, not engineering blockers: a genuinely
> independent second compiler (Phase 1), and the T1/C1 conjectures.

---

## 1. Maturity Roadmap (D0 §10)

| Phase | Definition | Status | Evidence / gap |
| --- | --- | --- | --- |
| **0 — Kernel Discovery** | Establish ⟨B, T⟩ and the constitution | ✅ **Complete** | D0 v1.1 frozen; `spec/D1.1–D1.5`. |
| **1 — Kernel Validation** | Empirical stress-testing; independent multi-compiler verification | 🟡 **Partial** | Two benchmarks (§121, ISO 8.7) + KPI + a 400+-instance labour corpus across **4 domains**; real inter-compiler κ=0.7877 (`cmd/interop`) and a fresh-vs-fresh κ=0.8380 (`cmd/trackd`); falsification campaign (`cmd/falsify`) + whole-store screen clean (410 rows); Registry Law Θ(1) verified across domains. **Gap:** no *organizationally independent* second compiler; single implementation. |
| **2 — Mechanized Semantics** | Machine-checked invariant proofs | 🟢 **Tractable set done** | T2 (I1), T3 (I8), T6 (I2), T8 (I7) proved mathlib-free and **CI-compiled** (GitHub Actions, Lean 4.31.0, zero `sorry`). **Remaining (research-grade):** T1 (decidability), C1 (minimality) open conjectures. |
| **3 — Industrial Compiler** | Production-scale passes enforcing invariants by construction | 🟢 **Substantial** | Go + PostgreSQL: bitemporal K̂ store, pure Ŝ evaluator + defeasible resolver, persisted Ê (replay → verdicts), CNF export + Ed25519 seal, temporal-read CLI, registry snapshots, exact-rational VAL. WP-1…WP-8 landed. **Gap:** Track D showed extraction is already clause-atomic; remaining gaps are *scale* beyond the dev corpus and *automated/continuous* ingestion. |

---

## 2. Four-Dimensional Rigor Program (D0 §8)

| Dimension | Status | Detail |
| --- | --- | --- |
| **8.1 Reproducibility** | 🟢 **Met** | CNF export byte-identical across runs (same digest, I8); α-renamed content-ordered ids; corpus-derived coordinates (no `time.Now()` in ingest). |
| **8.2 Independent Validation** | 🟡 **Partial** | Harness computes real Fleiss' κ and verdict-agreement with asserted floors (κ≥0.70, VA≥0.90). Live-corpus κ=**0.7877** (392 loci). **Caveat:** single team maintains both classifiers — measures rule-robustness, not true independence. Verdict-agreement over a *second verdict engine* not yet exercised. **Track D datapoint:** two *fresh* independent classifiers agree at κ=0.8380 (`cmd/trackd`), isolating the Track B gap to the older stored assignments rather than textual ambiguity. |
| **8.3 Formal Mechanization** | 🟢 **Met (tractable set)** | T2 (I1), T3 (I8), T6 (I2), T8 (I7) are mathlib-free Lean proofs that **compile in CI** (`.github/workflows/lean.yml`, Lean 4.31.0, zero `sorry`). I5 and additional I7 content are also **Go-tested** over the store. Open: T1/C1 research conjectures. |
| **8.4 Continuous Ingestion** | 🟡 **Partial** | Store spans **4 real normative domains** (VN labour statute, ISO 9001, US tax §121, KPI/policy). `TestRegistryLawBoundedBasisAcrossDomains` proves the basis stays = B (6 constructors, Θ(1)) across all domains — no domain adds a constructor. **Track D (2026-07-09):** corpus is already clause-atomic (`cmd/trackd`, segmentation ×1.00). **Gap:** ingestion is not yet *continuous*/automated (one-shot per corpus); residual classifier disagreement is semantic (cue modelling). |

---

## 3. Invariants I1–I9 — enforcement status

| # | Invariant | Enforcement | Status |
| --- | --- | --- | --- |
| I1 | Read-only algebra | `Environment` copy-on-bind; no DB handle in `Eval`; Lean T2 (CI-compiled) | 🟢 by construction + proof CI-compiled |
| I2 | Single writer / append-only | DB trigger + RBAC (`e_writer`); invariant tests reject UPDATE/DELETE; Lean T6 (CI-compiled) | 🟢 enforced + tested + proved |
| I3 | Kernel closure | 6-constructor enum; `FALSIFICATION-CANDIDATE` screen halts extensions; whole store screened clean (410 rows) | 🟢 held (Track C + store-wide) |
| I4 | Registry inertness | pure rename-stability test (`internal/registry`) | 🟢 tested (Go); Lean T4 open |
| I5 | Presentation erasure | verdict identifier = `CanonicalHash(ast)` only; erasure + adversarial-mutation test over 392 stored instances | 🟢 **tested** |
| I6 | Bitemporal totality | `tix_explicit_lower` CHECK; verdicts carry coordinates; temporal-read CLI | 🟢 enforced |
| I7 | Stratified reflection | Lean T8 **proved** (`Nat.lt_wfRel.wf`, CI-compiled) + REF graph acyclic + store-wide sub-Turing screen (410) | 🟢 proved + tested |
| I8 | Pass determinism | byte-stable CNF; resolver tie-break; no float64; Lean T3 (CI-compiled) | 🟢 held + proof CI-compiled |
| I9 | Source anchoring | UNIQUE(source_map.instance_pk) + totality (410/410 mapped, 0 unmapped) + tests | 🟢 enforced + tested |

**Weakest links — all closed at the invariant level.** Every invariant now has a
passing test and/or a CI-compiled Lean proof (I5 closed 2026-07-09; T8/I7 proved
2026-07-09 via `Nat.lt_wfRel.wf`). The remaining open items are the *research
conjectures* T1 (decidability) and C1 (minimality) — not invariant gaps.

---

## 4. Falsification surface (D0 §9.1) — none triggered

All ten falsification criteria remain **un-triggered**:
- no corpus has forced a 7th constructor / Turing-complete T / new Ê state
  (adversarial inputs were halted, not accommodated — Track C);
- published CNF runs reproduce byte-identically;
- validation floors (κ≥0.70, VA≥0.90) currently hold on the tested corpora;
- Registry Law held (Θ(1) basis) across **4 domains**, and the whole store
  (410 rows) screens falsification-clean.

**Confirmation (D0 §9.2) is NOT yet earned:** it requires independent
compilation convergence at the verdict stratum and evidence that no smaller
spanning pair exists — neither is demonstrated (single implementation, and
minimality C1 is an open conjecture). Status is correctly *provisional*.

---

## 5. Top risks / open items

1. **Mechanization CI-verified — resolved.** T2/T3/T6/T8 compile in GitHub Actions
   (`.github/workflows/lean.yml`, Lean 4.31.0, zero `sorry`). The local toolchain
   block (elan DNS / GitHub assets) is bypassed by running `lake build` on GitHub
   runners. Open (research): T1 decidability, C1 minimality.
2. **No truly independent second compiler.** The κ number is real but
   intra-team. A genuine Phase-1 claim needs a separate implementation.
3. **I7 discharged — resolved.** T8 (`strata_wellFounded`) proved via the Lean-core
   `Nat.lt_wfRel.wf` and CI-compiled, alongside the operational store checks
   (acyclic REF graph + store-wide sub-Turing screen).
4. **Extraction depth (Track D) — resolved.** `cmd/trackd` showed the corpus is
   already clause-atomic (segmentation ×1.00, 0 multi-modal); two fresh
   classifiers agree at κ=0.8380 (> the 0.7877 stored baseline), so the residual
   gap is *semantic cue modelling*, not structural splitting.
5. **Agent-0 decisions — RULED & implemented** (2026-07-09, `AGENT-0-DECISIONS.md`):
   TIX ratified as the temporal index, not a constructor (|B|=6, Option A); NRM
   `Force` deprecated → obligation-only (Option B); `DEFEATED` added as a
   first-class verdict distinct from `INAPPLICABLE` (Option B, migration 0005).
   All `AGENT-0-DECISION-*` markers removed; spec (D1.1/D1.2), README, and tests
   reconciled; build/tests green.

## 6. Recommended next steps (priority order)

1. ~~Compile the Lean proofs in a GitHub-reachable CI env; discharge **T8**~~ —
   **DONE** 2026-07-09: `.github/workflows/lean.yml` compiles T2/T3/T6/T8 on
   GitHub Actions (Lean 4.31.0, zero `sorry`); T8 discharged. Remaining Phase 2:
   the **T1** (decidability) and **C1** (minimality) research conjectures.
2. ~~Add the **I5 erasure test**~~ — **DONE** 2026-07-09
   (`TestI5PresentationErasure{Pure,Corpus}`); the invariant scorecard now has
   no untested invariant.
3. ~~**Track D** extraction depth~~ — **DONE** 2026-07-09 (`cmd/trackd`): corpus
   already clause-atomic; lever redirected to cue modelling. See
   `validation/trackd/REPORT.md`.
4. **Continuous ingestion** — multi-domain Registry Law now **tested**
   (`TestRegistryLawBoundedBasisAcrossDomains`, 4 real domains, basis = B). Still
   open: *automated/continuous* ingestion (currently one-shot per corpus).
5. ~~Escalate the three Agent-0 decisions~~ — **RULED & implemented** 2026-07-09
   (TIX index / obligation-only NRM / first-class `DEFEATED`); see
   `AGENT-0-DECISIONS.md`.
