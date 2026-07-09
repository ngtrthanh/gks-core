# PROGRESS REPORT — gks-core vs. the D0 Maturity Roadmap

**Date:** 2026-07-09 UTC · **Author:** Kiro (Engineering)
**Baselines:** `spec/D0v5.md` (D0 v1.1, FROZEN), `spec/D1.1–D1.5`,
`handoff.md`/`handoff2.md`/`handoff3.md`, `CHANGELOG.md`.
**Verification basis:** `go build/vet/test ./...` green; live dev DB (Postgres 18,
port 5435); `make validate` PASS; `cmd/interop`, `cmd/falsify` run.

> One-line status: **Phase 3 (industrial compiler) is substantially built and
> conformance-hardened; Phases 1–2 (validation + mechanization) are partially
> met and are now the frontier.** No falsification criterion has triggered; the
> kernel remains provisionally intact but NOT yet confirmed.

---

## 1. Maturity Roadmap (D0 §10)

| Phase | Definition | Status | Evidence / gap |
| --- | --- | --- | --- |
| **0 — Kernel Discovery** | Establish ⟨B, T⟩ and the constitution | ✅ **Complete** | D0 v1.1 frozen; `spec/D1.1–D1.5`. |
| **1 — Kernel Validation** | Empirical stress-testing; independent multi-compiler verification | 🟡 **Partial** | Two benchmarks (§121, ISO 8.7) + KPI + a 400+-instance labour corpus ingested; real inter-compiler κ=0.7877 (`cmd/interop`); falsification campaign started (`cmd/falsify`). **Gap:** no *organizationally independent* second compiler; corpus breadth still narrow. |
| **2 — Mechanized Semantics** | Machine-checked invariant proofs | 🟡 **Partial** | T2 (I1), T3 (I8), T6 (I2) proved mathlib-free in `mechanization/`. **Gap:** proofs **not CI-compiled** (Lean toolchain unreachable in this env); T8 (I7) `sorry`; T1 (decidability), C1 (minimality) open. |
| **3 — Industrial Compiler** | Production-scale passes enforcing invariants by construction | 🟢 **Substantial** | Go + PostgreSQL: bitemporal K̂ store, pure Ŝ evaluator + defeasible resolver, persisted Ê (replay → verdicts), CNF export + Ed25519 seal, temporal-read CLI, registry snapshots, exact-rational VAL. WP-1…WP-8 landed. **Gap:** extraction depth (Track D); scale beyond dev corpus. |

---

## 2. Four-Dimensional Rigor Program (D0 §8)

| Dimension | Status | Detail |
| --- | --- | --- |
| **8.1 Reproducibility** | 🟢 **Met** | CNF export byte-identical across runs (same digest, I8); α-renamed content-ordered ids; corpus-derived coordinates (no `time.Now()` in ingest). |
| **8.2 Independent Validation** | 🟡 **Partial** | Harness computes real Fleiss' κ and verdict-agreement with asserted floors (κ≥0.70, VA≥0.90). Live-corpus κ=**0.7877** (392 loci). **Caveat:** single team maintains both classifiers — measures rule-robustness, not true independence. Verdict-agreement over a *second verdict engine* not yet exercised. |
| **8.3 Formal Mechanization** | 🟡 **Partial** | I1/I8/I2 proofs written (Lean, mathlib-free); **compilation pending** a Lean toolchain in CI. I5, I7 not mechanized. |
| **8.4 Continuous Ingestion** | 🟡 **Partial** | One unsupervised corpus (VN Labour Code) ingested; Registry Law held (basis stayed at 6, Θ(1)). **Gap:** not continuous, not multi-domain; docx extraction is shallow (~29% yield, single-cue). |

---

## 3. Invariants I1–I9 — enforcement status

| # | Invariant | Enforcement | Status |
| --- | --- | --- | --- |
| I1 | Read-only algebra | `Environment` copy-on-bind; no DB handle in `Eval`; Lean T2 | 🟢 by construction (proof uncompiled) |
| I2 | Single writer / append-only | DB trigger + RBAC (`e_writer`); invariant tests reject UPDATE/DELETE; Lean T6 | 🟢 enforced + tested |
| I3 | Kernel closure | 6-constructor enum; `FALSIFICATION-CANDIDATE` screen halts extensions | 🟢 held (Track C) |
| I4 | Registry inertness | pure rename-stability test (`internal/registry`) | 🟢 tested (Go); Lean T4 open |
| I5 | Presentation erasure | source_map is P̂; drop-P̂-preserves-verdicts | 🔴 **not tested** |
| I6 | Bitemporal totality | `tix_explicit_lower` CHECK; verdicts carry coordinates; temporal-read CLI | 🟢 enforced |
| I7 | Stratified reflection | Lean T8 stated | 🟡 `sorry` (unproven) |
| I8 | Pass determinism | byte-stable CNF; resolver tie-break; no float64; Lean T3 | 🟢 held + proof (uncompiled) |
| I9 | Source anchoring | UNIQUE(source_map.instance_pk) + totality (410/410 mapped, 0 unmapped) + tests | 🟢 enforced + tested |

**Weakest links:** I5 (untested), I7 (unproven), and the *compilation* of the I1/I2/I8 proofs.

---

## 4. Falsification surface (D0 §9.1) — none triggered

All ten falsification criteria remain **un-triggered**:
- no corpus has forced a 7th constructor / Turing-complete T / new Ê state
  (adversarial inputs were halted, not accommodated — Track C);
- published CNF runs reproduce byte-identically;
- validation floors (κ≥0.70, VA≥0.90) currently hold on the tested corpora;
- Registry Law held (Θ(1) basis) on the ingested domain.

**Confirmation (D0 §9.2) is NOT yet earned:** it requires independent
compilation convergence at the verdict stratum and evidence that no smaller
spanning pair exists — neither is demonstrated (single implementation, and
minimality C1 is an open conjecture). Status is correctly *provisional*.

---

## 5. Top risks / open items

1. **Mechanization not CI-verified.** Lean toolchain download is blocked in this
   environment (elan DNS fail; GitHub release assets unreachable). Proofs are
   review-ready but unproven-in-CI. → run in an env with GitHub-asset access.
2. **No truly independent second compiler.** The κ number is real but
   intra-team. A genuine Phase-1 claim needs a separate implementation.
3. **I5 has no test; I7 unproven.** Two invariants weaker than the rest.
4. **Extraction depth (Track D).** κ baseline is 0.7877; clause-level splitting +
   multi-modality is the lever to raise it.
5. **Pending Agent-0 decisions** (unchanged): TIX-enum drop vs. handoff §6;
   NRM Force O|P|F trichotomy; resolver→verdict mapping.

## 6. Recommended next steps (priority order)

1. Compile the Lean proofs in a GitHub-reachable CI env; discharge **T8** and
   close Phase 2's tractable set.
2. Add the **I5 erasure test** (drop P̂/source_map → verdicts unchanged) — cheap,
   closes the weakest invariant gap.
3. **Track D** extraction depth against the 0.7877 κ baseline.
4. Broaden **continuous ingestion** to a second domain to genuinely exercise the
   Registry Law.
5. Escalate the three Agent-0 decisions for a ruling.
