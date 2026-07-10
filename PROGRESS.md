# PROGRESS REPORT — gks-core vs. the D0 Maturity Roadmap

**Date:** 2026-07-10 UTC (post external review; supersedes the tag `v1.0.0` claims) · **Author:** Kiro (Engineering)
**Baselines:** `Computable Governance.md` (D0 v1.1, FROZEN), `spec/D1.1–D1.5`,
`AGENT-0-DECISIONS.md`, `PHASE1-EXIT-REVIEW.md`, `CHANGELOG.md`.
**Verification basis:** `go build/vet/test ./...` green; live dev DB (Postgres 18,
port 5435); `cmd/interop`/`cmd/falsify`/`cmd/trackd` run; the Lean mechanization
**compiles in CI** (`.github/workflows/lean.yml`, Lean 4.31.0, zero `sorry`) — but
see the mechanization-scope correction below.

> **STATUS CORRECTION (2026-07-10) — external Phase-1 exit review returned REVISE.**
> The prior "Phase 1 ACCEPTED" self-verdict is **WITHDRAWN**. An external reviewer
> (`PHASE1-EXIT-REVIEW.md`) found the *headline* claims overstated the evidence:
> **(F1)** the Lean proofs are over **simplified / definitional models**, not the D1.5
> statements as written — "T1–T8 all proved" is corrected to honest per-obligation
> status in `spec/D1.5`; **(F2)** Phase-1 acceptance lacked D0 §10.2's independent
> multi-compiler verification and is reinstated as **Phase-1 preparation**; **(F3)** the
> frozen D0 basis is seven constructors — the 7→6 (TIX) and Force reductions are now
> **logged as a partial falsification of Hypothesis 1** (§4); **(F4)** the validation
> numbers are first-party (no human gold standard / no independent compiler). The
> engineering artifact is real and reproducible; the *scientific* claims are being
> brought back in line with it. Remediation is tracked as WS-A…WS-G (§6).

> One-line status: **Phase-1 PREPARATION complete; NOT accepted.** A reproducible,
> deterministic governance-rule engine (Phase 3) with DB-enforced invariants and a
> mathlib-free Lean scaffold that CI-compiles. Acceptance requires independent
> multi-compiler convergence (D0 §10.2 / §9.2.3), which does **not** exist. The
> falsification design is real but first-party-executed; Hypothesis 1 (seven
> constructors) is partially falsified by the program's own 7→6 reduction.

---

## 1. Maturity Roadmap (D0 §10)

| Phase | Definition | Status | Evidence / gap |
| --- | --- | --- | --- |
| **0 — Kernel Discovery** | Establish ⟨B, T⟩ and the constitution | ✅ **Complete** | D0 v1.1 frozen; `spec/D1.1–D1.5`. |
| **1 — Kernel Validation** | Empirical stress-testing; independent multi-compiler verification | ⚠️ **Preparation (acceptance WITHDRAWN)** | Benchmarks + a 400+-instance labour corpus; first-party κ=0.7877 (`cmd/interop`) and fresh-vs-fresh κ=0.8380 (`cmd/trackd`); falsification harness; store-wide screen; Registry Law across 4 curated domains. **The D0 §10.2 gate — independent multi-compiler verification — is NOT met** (single implementation, no human gold standard). Reinstated as *preparation* per the exit review (F2/F4). |
| **2 — Mechanized Semantics** | Machine-checked invariant proofs | 🟡 **Scaffold only (models, not the stated theorems)** | Mathlib-free Lean that CI-compiles (Lean 4.31.0, zero `sorry`), but (F1): T2/T5 are `rfl`/definitional; T3/T6/T7/T8 are lemmas over simplified structures — the D1.4 `Step` relation and D1.3 `Schema@level` are **not formalized**; only **T1** is a genuine theorem, over a reduced `Expr` fragment. See the corrected `spec/D1.5` ledger. **Remaining:** formalize `Step` + prove T3/T6/T7/T8 as stated; `infer`↔`eval` soundness; C1 (minimality, not yet well-posed). |
| **3 — Industrial Compiler** | Production-scale passes enforcing invariants by construction | 🟢 **Substantial** | Go + PostgreSQL: bitemporal K̂ store, pure Ŝ evaluator + defeasible resolver, persisted Ê (replay → verdicts), CNF export + Ed25519 seal, temporal-read CLI, registry snapshots, exact-rational VAL. WP-1…WP-8 landed. **Gap:** Track D showed extraction is already clause-atomic; remaining gaps are *scale* beyond the dev corpus and *automated/continuous* ingestion. |

---

## 2. Four-Dimensional Rigor Program (D0 §8)

| Dimension | Status | Detail |
| --- | --- | --- |
| **8.1 Reproducibility** | 🟢 **Met** | CNF export byte-identical across runs (same digest, I8); α-renamed content-ordered ids; corpus-derived coordinates (no `time.Now()` in ingest). |
| **8.2 Independent Validation** | 🟡 **Partial** | Harness computes real Fleiss' κ and verdict-agreement with asserted floors (κ≥0.70, VA≥0.90). Live-corpus κ=**0.7877** (392 loci). **Caveat:** single team maintains both classifiers — measures rule-robustness, not true independence. Verdict-agreement over a *second verdict engine* not yet exercised. **Track D datapoint:** two *fresh* independent classifiers agree at κ=0.8380 (`cmd/trackd`), isolating the Track B gap to the older stored assignments rather than textual ambiguity. |
| **8.3 Formal Mechanization** | 🟡 **Partial (models, not the stated theorems)** | The Lean development CI-compiles (zero `sorry`) but proves *simplified models*, not the D1.5 relational obligations (F1): T2/T5 definitional (`rfl`), T3/T6/T7/T8 model-lemmas over unformalized `Step`/`Schema`, only T1 a genuine scoped theorem. Real mechanization (formalize `Step`; prove T3/T6/T7/T8 as stated; type-soundness) is Phase-2 research. |
| **8.4 Continuous Ingestion** | 🟢 **Substantial** | Store spans **4 real normative domains**; `TestRegistryLawBoundedBasisAcrossDomains` proves basis = B (Θ(1)) across all. **Continuous control plane** (`cmd/ingest_run` + `ingestion_run` ledger, migration 0006): manifest-driven, **digest-idempotent**, ledgered — a re-run over an unchanged corpus is a safe no-op (UP-TO-DATE→skip; verified 0-delta with Registry Law HELD). Idempotency is enforced in the control plane because `kernel_instance`'s EXCLUDE constraint rejects overlapping re-inserts. **Track D:** corpus already clause-atomic. **Gap:** scheduling is external (cron/CI); one unsupervised corpus so far. |

---

## 3. Invariants I1–I9 — enforcement status

| # | Invariant | Enforcement | Status |
| --- | --- | --- | --- |
| I1 | Read-only algebra | `Environment` copy-on-bind; no DB handle in `Eval`; AST has no write op | 🟢 by construction (Go). Lean T2 is definitional (`rfl`) — non-evidentiary |
| I2 | Single writer / append-only | DB trigger + RBAC (`e_writer`); invariant tests reject UPDATE/DELETE | 🟢 enforced + tested (DB/Go). Lean T6 is a `List Nat` model-lemma — not evidence. ⚠ θ-guard bypassable by `e_writer` via `gks.via_transition` (WS-D) |
| I3 | Kernel closure | 6-constructor enum; `FALSIFICATION-CANDIDATE` screen halts extensions; whole store screened clean (410 rows) | 🟢 held (Track C + store-wide) |
| I4 | Registry inertness | pure rename-stability test (`internal/registry`, Go) | 🟢 tested (Go). Lean T4 is a genuine induction but over the `Expr` model, not the K̂-level `verdict` statement |
| I5 | Presentation erasure | verdict identifier = `CanonicalHash(ast)` only; erasure + adversarial-mutation test over 392 stored instances (Go) | 🟢 tested (Go). Lean T5 is definitional (`rfl`) — non-evidentiary |
| I6 | Bitemporal totality | `tix_explicit_lower` CHECK; verdicts carry coordinates; temporal-read CLI | 🟢 enforced (DB). Lean T7 is a list model-lemma. ⚠ registry is versioned, **not bitemporal** (M10) |
| I7 | Stratified reflection | REF-graph acyclicity test + op-name screen | 🔴 **vacuous** — `Schema@level` reflection is **not implemented** anywhere; REF-acyclicity ≠ stratification; Lean T8 is a `Nat` library fact with no `Schema` (F1/finding-7) |
| I8 | Pass determinism | byte-stable CNF (reviewer-reproduced); resolver tie-break; no float64 | 🟢 held within a store (Go). ⚠ tie-break on store-UUID diverges across independent stores (M6). Lean T3 is a model-lemma |
| I9 | Source anchoring | UNIQUE(source_map.instance_pk) + totality (410/410 mapped) + tests | 🟢 total single-valued map. NOTE: D0 calls it a *bijection* but it is many-to-one (4 instances→1 locus in D8 Run 2); spec to be corrected |

**Weakest links (corrected per the exit review).** DB-enforced + Go-tested invariants
(I2, I3, I6, I9 at the `kernel_instance` level; I8 within a store; I1 by construction)
are real. The **Lean "proofs" add no evidential weight** (F1): T2/T5 definitional,
T3/T6/T7/T8 lemmas over unformalized structures. **I7 is vacuous** (constrains an
unimplemented feature). Open items: real `Step`/`Schema` mechanization; independent
verification (D0 §10.2); and the *research conjecture* C1 (minimality, not yet
well-posed) — not an invariant gap.

---

## 4. Falsification surface (D0 §9.1)

**One criterion HAS fired (logged 2026-07-10, per exit-review F3).** D0 Hypothesis 1
asserts "a fixed basis of **seven** irreducible constructors" including TIX. The
program itself **reduced** two of those "irreducible" objects — TIX → columnar
metadata (Ruling 1) and Force → GRD/absence (Ruling 2) — so **Hypothesis 1 as
frozen is partially falsified.** This is recorded here as a falsification datum,
not silently absorbed as a "ruling." The live basis is six; whether *six* is
minimal is the open C1 question (which now trends *against* the frozen claim).

The other criteria are **un-triggered but weakly tested** (exit-review F4 — the
tests are first-party):
- no *encoded* corpus has forced a 7th (live) constructor / Turing-complete T /
  new Ê state — but representability was judged by the authors who wrote the
  adversarial JSON, not by an independent encoder (the `Screen` allowlist only
  rejects out-of-vocabulary op strings);
- CNF runs reproduce byte-identically (reviewer-reproduced) — but no reference
  digest is published in-tree yet (WS-F);
- the κ/VA floors hold only on first-party classifiers / authored fixtures — **no
  human gold standard, no independent compiler** exists;
- Registry Law held across **4 curated** domains — n=4 is not an asymptotic Θ(1)
  result (M8/M12).

**Confirmation (D0 §9.2) is NOT earned and Phase 1 is NOT accepted.** It requires
independent multi-compiler convergence at the verdict stratum (D0 §10.2) — a
genuinely separate implementation, which does not exist. Status: *provisional,
under revision.*

---

## 5. Remediation work-streams (response to `PHASE1-EXIT-REVIEW.md`, verdict REVISE)

Prior "resolved" items (Lean CI, Track D, ingestion, Agent-0 rulings) remain done as
*engineering*; the review does not dispute the code, it disputes the *claims*. The
open work is truthfulness, reconciliation, real bugs, and the real science.

| WS | Scope | Findings | Status |
| --- | --- | --- | --- |
| **A** | Retract/re-scope mechanization claims | F1 | ✅ done — D1.5 ledger + PROGRESS + README relabeled (definitional / model-lemma / scoped) |
| **B** | Withdraw self-acceptance; log Hyp-1 falsification | F2, F3 | ✅ done — acceptance withdrawn (this doc); Hyp-1 partial falsification logged (§4) |
| **C** | D0 amendment protocol + D0 v1.2 | F3, M4, M5 | ✅ done — `spec/D0-AMENDMENTS.md` (protocol + A01–A05; FD-1/FD-2 falsification log) |
| **D** | Confirmed code bugs | M7 cycle, θ-bypass (min-8), M6 tie-break, M2 benchmark, unbound-var (min-12) | ◑ mostly — M7 fixed+test, M6 fixed, M2+min-12 documented; **θ-bypass flagged, not yet code-fixed** (needs RBAC refactor, WS-E) |
| **E** | Spec single-source-of-truth | M1, M3, M10 | ⏳ planned (est. ~1 eng-week) |
| **F** | Scholarly apparatus + hygiene | LICENSE/CITATION, ISO-PDF, dupes, M9 phantom paths, related work | ◑ done except the related-work survey (⏳) |
| **G** | Real confirmation science (needs external parties) | F4, C1, formalize `Step` | ⏳ scaffold/spec only |

## 6. What acceptance actually requires (D0 §10.2 / §9.2.3)

1. **Specify the verdict contract** (vocabulary, unguarded-norm default, tie
   semantics — replace UUID tie-break with a content key) so an independent
   implementation is *possible*.
2. **Commission a genuinely independent second implementation** (different team/
   stack) and demonstrate verdict-stratum convergence. *This is the only experiment
   that can confirm the kernel hypothesis; no further first-party code substitutes.*
3. **Human-annotated gold corpus**; κ measured compiler-vs-human, panels ≥ 3.
4. **Formalize the D1.4 `Step` system** and prove T3/T6/T7/T8 as stated; add
   type-soundness (`infer`↔`eval`); implement or strike I7.
5. **Make C1 well-posed** (define representability); run constructor-elimination
   experiments (the reviewer flags VAL as prima-facie eliminable).
