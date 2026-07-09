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
| **8.3 Formal Mechanization** | 🟡 **Partial** | I1/I8/I2 proofs written (Lean, mathlib-free); **compilation pending** a Lean toolchain in CI; T8 (I7) still `sorry`. I5 and I7's operational content are **mechanically tested in Go** (not Lean-*proved*): presentation-erasure over 392 instances, acyclic reflection graph + store-wide sub-Turing screen (410 instances). Distinction: *tested* ≠ *machine-proved*. |
| **8.4 Continuous Ingestion** | 🟡 **Partial** | Store spans **4 real normative domains** (VN labour statute, ISO 9001, US tax §121, KPI/policy). `TestRegistryLawBoundedBasisAcrossDomains` proves the basis stays = B (6 constructors, Θ(1)) across all domains — no domain adds a constructor. **Track D (2026-07-09):** corpus is already clause-atomic (`cmd/trackd`, segmentation ×1.00). **Gap:** ingestion is not yet *continuous*/automated (one-shot per corpus); residual classifier disagreement is semantic (cue modelling). |

---

## 3. Invariants I1–I9 — enforcement status

| # | Invariant | Enforcement | Status |
| --- | --- | --- | --- |
| I1 | Read-only algebra | `Environment` copy-on-bind; no DB handle in `Eval`; Lean T2 | 🟢 by construction (proof uncompiled) |
| I2 | Single writer / append-only | DB trigger + RBAC (`e_writer`); invariant tests reject UPDATE/DELETE; Lean T6 | 🟢 enforced + tested |
| I3 | Kernel closure | 6-constructor enum; `FALSIFICATION-CANDIDATE` screen halts extensions; whole store screened clean (410 rows) | 🟢 held (Track C + store-wide) |
| I4 | Registry inertness | pure rename-stability test (`internal/registry`) | 🟢 tested (Go); Lean T4 open |
| I5 | Presentation erasure | verdict identifier = `CanonicalHash(ast)` only; erasure + adversarial-mutation test over 392 stored instances | 🟢 **tested** |
| I6 | Bitemporal totality | `tix_explicit_lower` CHECK; verdicts carry coordinates; temporal-read CLI | 🟢 enforced |
| I7 | Stratified reflection | Lean T8 `sorry`; **operationally verified** — REF reflection graph acyclic + store-wide sub-Turing screen (410 instances) | 🟡 proof pending, empirically held |
| I8 | Pass determinism | byte-stable CNF; resolver tie-break; no float64; Lean T3 | 🟢 held + proof (uncompiled) |
| I9 | Source anchoring | UNIQUE(source_map.instance_pk) + totality (410/410 mapped, 0 unmapped) + tests | 🟢 enforced + tested |

**Weakest links:** the *compilation* of the I1/I2/I8 Lean proofs, and the Lean
T8 (I7) `sorry` — though I7's operational content (acyclic reflection graph +
store-wide sub-Turing screen over 410 instances) is now mechanically verified.
(I5 was closed 2026-07-09 with `TestI5PresentationErasure{Pure,Corpus}`.)

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
3. **I7 Lean proof pending.** T8 (well-foundedness) is still `sorry` in Lean; its
   operational content is now mechanically verified over the store (acyclic REF
   graph + store-wide sub-Turing screen, 2026-07-09). Formal discharge still needs
   the (blocked) Lean toolchain.
4. **Extraction depth (Track D) — resolved.** `cmd/trackd` showed the corpus is
   already clause-atomic (segmentation ×1.00, 0 multi-modal); two fresh
   classifiers agree at κ=0.8380 (> the 0.7877 stored baseline), so the residual
   gap is *semantic cue modelling*, not structural splitting.
5. **Pending Agent-0 decisions** (unchanged): TIX-enum drop vs. handoff §6;
   NRM Force O|P|F trichotomy; resolver→verdict mapping.

## 6. Recommended next steps (priority order)

1. Compile the Lean proofs in a GitHub-reachable CI env; discharge **T8** and
   close Phase 2's tractable set.
2. ~~Add the **I5 erasure test**~~ — **DONE** 2026-07-09
   (`TestI5PresentationErasure{Pure,Corpus}`); the invariant scorecard now has
   no untested invariant.
3. ~~**Track D** extraction depth~~ — **DONE** 2026-07-09 (`cmd/trackd`): corpus
   already clause-atomic; lever redirected to cue modelling. See
   `validation/trackd/REPORT.md`.
4. **Continuous ingestion** — multi-domain Registry Law now **tested**
   (`TestRegistryLawBoundedBasisAcrossDomains`, 4 real domains, basis = B). Still
   open: *automated/continuous* ingestion (currently one-shot per corpus).
5. Escalate the three Agent-0 decisions for a ruling.
