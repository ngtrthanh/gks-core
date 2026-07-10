# GKS Core — Phase-1 Exit Review (Final External Review)

**Reviewer:** Fable (external, per `FR.md`) · **Date:** 2026-07-10 UTC
**Object under review:** repository `ngtrthanh/gks-core` at `8db2b60` (tag `v1.0.0`)
**Standard applied:** publication-quality scrutiny; every claim guilty until proven by in-repository evidence.

---

## Executive Summary

GKS Core claims to be the accepted Phase-1 reference realization of a "governance computing kernel" ⟨B, T⟩: six constructors, a sub-Turing term algebra, nine invariants (I1–I9) each "tested and/or CI-proved", eight theorem obligations (T1–T8) "all proved" in Lean, empirical validation floors met, and a constitutional acceptance verdict dated 2026-07-10.

**What the repository actually demonstrates.** A small (~7.5 kLOC Go, ~400 lines Lean, ~22 KB of specs), well-engineered, genuinely reproducible software framework. This reviewer independently verified: `go build/vet/test ./...` green; the live store holds 410 kernel rows using exactly the six enum constructors; source-map totality holds (0 unmapped rows); two independent CNF export runs produce **byte-identical output** (SHA-256 `8b6304e1…`, 410 records); the GitHub Actions Lean workflow exists and its latest runs are green. The append-only trigger, RBAC role, bitemporal EXCLUDE constraints, exact-rational arithmetic (no float64 on the verdict path), and deterministic α-renamed exports are real and implemented with care. As engineering, this is honest, disciplined work.

**What the repository does not demonstrate.** The scientific claims that distinguish "a reproducible computational governance kernel" from "an interesting software framework" are not supported by the evidence:

1. **The mechanization is not what it is claimed to be.** The Lean development proves statements about drastically simplified toy structures, several of which are true by definition. T2 (purity) is `rfl` over an instrumented evaluator *defined* to return its environment unchanged (`Semantics.lean:72-84`). T5 (erasure) is `rfl` over a `verdict` function *defined* to ignore presentation (`Stability.lean:64-73`). T6 models K̂ as `List Nat` and proves `i ∈ k → i ∈ k ++ [x]` (`Invariants.lean:25-39`). T8 is the Lean-core fact `Nat.lt_wfRel.wf` with no `Schema` type defined anywhere (`Invariants.lean:47-48`). T3, T6 and T7 as stated in D1.5 quantify over the D1.4 small-step relation `Step c η c'` — **no such relation is formalized anywhere in `mechanization/`**. The claims "T1–T8 all proved" (README, PROGRESS, D1.5 ledger, CHANGELOG) and "every invariant with a Go test now also has a machine-checked proof" are therefore materially overstated.

2. **Phase-1 acceptance contradicts the frozen constitution.** D0 (`Computable Governance.md`, "PERMANENTLY FROZEN") defines Phase 1 as "Empirical stress-testing and **independent verification via multi-compiler panels**" (§10.2) and makes independent compilation convergence a condition of provisional confirmation (§9.2.3). The repository concedes no independent compiler exists (PROGRESS §5.2; `validation/interop/REPORT.md` caveat). The acceptance verdict resolves this by *reclassifying* the missing requirement out of Phase 1 — an amendment of the frozen text by the same internal authority ("Agent-0") that issued the acceptance.

3. **The constitution contradicts the implementation on its central object.** Frozen D0 defines B as **seven** constructors including TIX (§3.1, §4.1) and Hypothesis 1 asserts "a fixed basis of seven irreducible constructors." The entire implementation, D1.1 (as amended), and all tests assert |B| = 6. Under the program's own falsificationist logic, Hypothesis 1 as frozen has been refuted (a "irreducible" constructor was reduced to metadata) — yet the ledger reports "no falsification criterion has triggered."

4. **The empirical validation is self-referential.** The κ = 0.7877 figure compares two rule-based classifiers written and maintained by the same team, with no human-annotated gold standard anywhere in the tree. The VA ≥ 0.90 floor has only ever been exercised against 590-byte fixture files authored to pass (`validation/testdata/`, wired to `make validate`). The falsification campaign screens hand-authored JSON whose "adversarial" property is an out-of-vocabulary op string (`"forall"`, `"fix"`, `"OBLIGATION2"`) checked against an allowlist (`validation.Screen`, `validation.go:210-230`) — the hard representability judgment was made by the human who wrote the JSON, not tested by the harness.

The entire repository history spans five days (2026-07-05 → 07-10); the twelve release tags v0.4.0 → v1.0.0 were created within 36 hours; the review that drove the hardening (`CONFORMANCE-REVIEW.md`) is absent from the tree; there is no LICENSE, no citation metadata, and **no related-work positioning of any kind** despite substantial overlapping prior art (LegalRuleML, Catala, defeasible deontic logic, Akoma Ntoso's bitemporal text model, Hohfeldian taxonomies, stratified Datalog).

**Bottom line.** The repository demonstrates a reproducible, deterministic governance-rule engine of good engineering quality, wrapped in scientific claims its evidence does not support. The gap is not cosmetic: it is between "proved" and "modeled", between "independently validated" and "self-agreed", and between "constitutionally accepted" and "self-accepted against the frozen text". These are correctable — the honest artifact underneath is worth correcting.

---

## Acceptance Decision

# **REVISE**

Not REJECT: the artifact is real, reproducible, internally documented, and several of its caveats are stated with unusual candor (PROGRESS §8.2, the interop report's independence caveat, the α-rename `ambiguous` accounting). Not ACCEPT-with-reservations: the headline claims (Phase-1 accepted; T1–T8 proved; validation floors met) are the claims under review, and each fails on evidence.

---

## Evidence Matrix

| # | Claim (source) | Evidence found | Confidence | Missing evidence |
|---|---|---|---|---|
| 1 | Byte-deterministic CNF export, I8 (README §Foundational) | **Independently reproduced by reviewer**: two runs, identical SHA-256, 410 records. `cnf/alpha.go` content-ordered ids with explicit collision accounting. | **High** | Published reference digest in-tree (`export/` is gitignored; "published compilation runs" per D0 §9.1.7 do not exist). |
| 2 | Kernel closure I3, six constructors (README) | 6-value enum (`schema.sql:21-23`); live store: 6 distinct constructors over 410 rows (verified); screen halts unknown ops. | High (as an engineering constraint) | Closure of the *theory*: frozen D0 says 7. See Fatal F3. |
| 3 | Append-only / single writer I2 (README) | Trigger `kernel_instance_append_only` + RBAC (`schema.sql:109-154`); invariant tests. | High for `kernel_instance` | `e_machine` θ-discipline bypassable by any `e_writer` session via `set_config('gks.via_transition','on',true)` (`schema.sql:277`). Lean T6 does not evidence this claim (toy list lemma). |
| 4 | "T1–T8 all proved, CI-compiled, zero sorry" (README, D1.5 ledger, PROGRESS) | CI workflow exists, latest runs green (verified via `gh run list`). Lean compiles. | **Low that the D1.5 statements are proved** | T3/T6/T7 D1.5 statements quantify over `Step` — never formalized. T2/T5 are definitional (`rfl`). T8 is a library fact. T1 honest but scoped to a 9-production fragment with no type-soundness link between `infer` and `eval`. |
| 5 | I1 read-only algebra (README) | Go evaluator takes no DB handle, `Now` a parameter; AST has no write ops. | High (by construction) | A proof about the actual system. `eval_is_pure` is vacuous: `evalWithEnv` returns `env` by definition. |
| 6 | I6 bitemporal totality | `tix_explicit_lower` CHECK + non-empty range CHECKs (`schema.sql:52-55`); verdicts carry coordinates. | High | Registry is versioned by integer, **not bitemporal** — historical verdict stability across registry versions unspecified. |
| 7 | I7 stratified reflection "proved + tested" (PROGRESS §3) | Lean `strata_wellFounded` = `Nat.lt` well-foundedness; "tested" via REF-graph acyclicity + op-name screen. | **Very low** | Expression-schemas (`Schema@level`, D1.3 §3) are **implemented nowhere** — no parser, no store rows, no evaluator support. I7 is vacuously satisfied by absence of the feature it constrains. REF acyclicity is a different property. |
| 8 | I9 source-anchoring "bijection" (D0 §6-I9) | UNIQUE(instance_pk) (migration 0003); totality verified live by reviewer (0 unmapped / 410). | High for *total single-valued map* | Bijectivity is false by design: D8 Run 2 maps four instances (`n2a…n2d`) to one clause locus. D0 overstates; no doc corrects it. |
| 9 | Inter-compiler κ = 0.7877 ≥ 0.70 (PROGRESS §8.2) | `cmd/interop` computes real Fleiss' κ over 392 loci; report + caveat in-tree. | Low as *independent validation* | Both "compilers" same-team rule classifiers; no human gold standard; 2 raters vs D0's "panels"; boundary loci excluded from denominator; floor cleared by 0.09. |
| 10 | Verdict agreement ≥ 0.90 (README, D0 §8.2) | `VerdictAgreement` implemented; `make validate` runs it on `validation/testdata` fixtures (590 B, authored). | **Very low** | Never exercised against a second verdict engine (admitted, PROGRESS §8.2). |
| 11 | Falsification campaign, none triggered (PROGRESS §4) | `cmd/falsify` + REPORT: 3 HALTs on hand-authored adversarial JSON. | Low | Screen checks op-name allowlist on pre-encoded input; representability decided by the encoder. No held-out corpus encoded by a non-author. Hyp-1 revision (7→6) not logged as a falsification datum. |
| 12 | Registry Law Θ(1) across 4 domains (PROGRESS §8.4) | `TestRegistryLawBoundedBasisAcrossDomains`; live basis = 6. | Low for the asymptotic claim | n = 4 curated domains, 3 of which are single-fixture hand-ingestions (§121 excerpt, ISO §8.7, one KPI). Θ(1) from n=4 non-random points is not an asymptotic result. |
| 13 | Ê persistence end-to-end (README WP-3) | Tables + triggers + `replay_d8`; journal linearity trigger. | Medium | Live store: **4 events, 6 verdicts**; `defeated` and `violated` never persisted. The Ê layer has been exercised on one small benchmark trace. |
| 14 | C1 minimality (D1.5) | Correctly labeled open conjecture. | — | No elimination experiment for any constructor exists; "representable"/"simulate" never defined, so C1 is not yet well-posed. Two reductions already happened (TIX, Force) — evidence *against* the frozen basis's minimality. |
| 15 | Phase 1 accepted (README, AGENT-0-DECISIONS) | Internal verdict text dated 2026-07-10. | **None as external validation** | D0 §10.2 multi-compiler panels; §9.2.3 convergence. Accepting authority is internal to the program. |

---

## Fatal Findings

Issues capable of invalidating the scientific claims as stated.

**F1 — The mechanization claims are materially false as stated.**
D1.5's ledger marks T1–T8 "proved"; README says "T1–T8 all proved and CI-compiled"; CHANGELOG says "every D1.5 theorem obligation is machine-checked." Examination of `mechanization/`:
- **T2** (`eval_is_pure`, `Semantics.lean:83`): `evalWithEnv env e := (eval env e, env)`; the theorem `(evalWithEnv env e).2 = env` is true by the definition just given. It cannot fail regardless of what `eval` does. Zero evidentiary content.
- **T5** (`verdict_erases_presentation`, `Stability.lean:72`): `verdict env p := eval env p.expr`; the theorem is `rfl` because presentation was never an input. Same pattern.
- **T3** as stated in D1.5 is determinism of the relation `Step c η c'` (the D1.4 §3 transition system). What is proved (`eval_deterministic`) is that a Lean *function* equals itself — true of any function, and about the expression evaluator, not Ê. **The D1.4 transition system is not formalized anywhere.**
- **T6** as stated quantifies over `Step`; what is proved is `i ∈ k → i ∈ k ++ [x]` for `KB := List Nat`, where "the only admissible mutation is append" is an assumption *encoded in the model*, not a conclusion.
- **T7**: a two-case list lemma of the same character.
- **T8** as stated concerns schema-matching; what is proved is `WellFounded (· < · : Nat → Nat → Prop)` — a Lean-core instance. No `Schema`, no match relation, no connection to D1.3 §3.
- **T1** is the only obligation with honest scoping ("mechanized `Expr` fragment") — but `HasType` is *defined as the graph of* `infer` (`Typing.lean:74`), making decidability true by construction rather than a metatheorem about the D1.2 relational system, and no soundness theorem links `infer` to `eval` (indeed `eLookup` is inferred `TBool` while `eval` may return a registry `VInt` — the fragment is not even type-sound).
The pattern is uniform: for each obligation, a model was constructed in which the theorem is immediate, then the D1.5 ledger was stamped "proved". D1.4's Lemma 1 ("proved by rule induction; D1.5 §T2") refers to a rule induction that does not exist. This invalidates the mechanization pillar (D0 §8.3 "Met") of the acceptance.

**F2 — Phase-1 acceptance violates the repository's own frozen constitution.**
D0 §10.2 defines Phase 1 to include "independent verification via multi-compiler panels"; D0 §9.2.3 requires "independent compilation converges at the verdict stratum" for provisional confirmation. The repository concedes both are absent (PROGRESS §5.2: "A genuine Phase-1 claim needs a separate implementation"; interop REPORT caveat). The final verdict resolves the deficit by reclassifying independent replication as post-acceptance "scientific confirmation" — i.e., the acceptance criterion was moved after the fact, by the internal authority issuing the acceptance, against a document labeled PERMANENTLY FROZEN. Whatever the engineering merits, "Phase 1 accepted" is not a scientifically valid status.

**F3 — The frozen constitution contradicts the implemented theory on the basis B.**
`Computable Governance.md` (D0 v1.1, "PERMANENTLY FROZEN"): B = {NRM, CLS, PWR, GRD, REF, VAL, **TIX**} (§3.1, §4.1); Hypothesis 1: "a fixed basis of **seven** irreducible constructors"; NRM signature includes ×Force (§4.1). The implementation, amended D1.1, and every test assert |B| = 6 with Force deprecated. Two constitutional objects declared irreducible were reduced within one week (TIX → column metadata; Force → GRD/absence). Either (a) D0 was amended, violating its freeze and requiring an amendment protocol D0 does not define, or (b) D0 stands and the implementation is non-conforming. The repository asserts both "D0 v1.1 frozen" and |B| = 6, and simultaneously reports "no falsification criterion has triggered" — but the refutation of Hypothesis 1 *as frozen* is exactly the kind of datum the falsification ledger exists to record. The evidentiary chain of the acceptance is unsound at its root.

**F4 — The independent-validation pillar is circular.**
Every number offered for D0 §8.2 is produced by artifacts authored by the same team that authored the hypothesis: (i) κ = 0.7877 compares the stored ingester's assignments to a second rule-classifier by the same authors — it measures rule-choice robustness (as the report itself admits), not inter-compiler convergence, and there is no human-annotated ground truth anywhere in the tree; (ii) the VA ≥ 0.90 floor is asserted only over `validation/testdata` fixtures authored to pass; (iii) the falsification campaign's adversarial inputs are self-encoded JSON rejected by an op-name allowlist — the question the campaign purports to test (can a real governance text force the kernel open?) is decided upstream by the author doing the encoding. No held-out corpus was encoded by anyone outside the program. The validation dimension therefore contributes no independent confirmation, and the acceptance's empirical basis reduces to: the system agrees with itself.

---

## Major Findings

**M1 — Three non-identical languages are all called T.** D1.3's EBNF contains `PREV/ONCE/SINCE`, Allen interval relations, and `Schema@level`; the Go AST (`kernel/ast.go`) implements none of these but adds `count`, `window(within)`, `ratio`, and `boundary` — none of which appear in the EBNF; the Lean `Expr` is a third, 9-production fragment. The "closed algebra T" (I3) is closed over whichever vocabulary each component happens to use. The sub-Turing screen (`closedOps`) canonizes the *implementation's* op set, not the specification's.

**M2 — The flagship benchmark narrative does not match the executable encoding.** D8 Run 1's designed "kill-shot" is g2 defeating *guard* g1 (`defeats=[REF(g1)]`) — exception-to-exception defeat, as in D1.4 S-Defeat (Δ₂ ⊆ D₁ over guard target sets). The shipped encoding is `Defeats: []string{idN1}` (`ingest_benchmark/main.go:90`) — g2 defeats the *norm*. The single-pass resolver (`Resolve`) matches guards only against the norm ID and cannot express guard-vs-guard defeat chains at all. The documented capability ("the run's designed kill-shot lands inside frozen T₁") was not what ran.

**M3 — D1.4 is under-specified and stale.** S-Violate still keys on the deprecated Force (`NRM(b,c,a,+,O)`) post-Ruling-2; `due(ν)`, `performed(b,a)`, `defeated(Δ,D,θ)`, and `ρ_η` are used without definition (NRM's tuple has no deadline component from which `due` could be derived); S-Defeat does not require the defeater's own condition to hold (the implementation does); there is no rule producing `discharged`, no suspension-lift, and no rule emitting conditional verdicts despite D0 §7.2 mandating them. The §5 determinism theorem asserts "ties are impossible" while the implementation exists precisely because ties are possible (guard-ID tie-break, `resolver.go:51-59`).

**M4 — The lifecycle alphabet is inconsistent across documents and store.** D1.1 Axiom 1.3: Σ = {inforce, suspended, violated, discharged, **terminated**} (5 states). DB `e_state` and D8: {**proposed**, in-force, suspended, violated, discharged, **extinguished**} (6 states). The implemented alphabet is neither the specified one nor documented as an amendment. (D0 §9.1.4 makes "an un-modeled Ê alphabet state" a falsification trigger.)

**M5 — The verdict vocabulary is implementation-defined.** {compliant, violated, conditional, inapplicable, defeated} appears in no specification; D1.4 §4 defines the verdict as a θ-projection instead. The semantically consequential default — an unguarded norm with no activator is judged against by default, and `INACTIVE` downgrades only when an activating guard exists and did not fire (`machine.go:371-441`) — is documented only in code comments. Different reasonable defaults yield different verdicts on identical stores; nothing in D0/D1.x pins this down for an independent reimplementer, which directly undermines the verdict-stratum convergence target.

**M6 — Equal-priority resolution is arbitrary across independent stores.** Ties break on guard ID = store UUID (random at ingest). Within one store this is deterministic (I8 holds); across two independent compilers ingesting the same corpus, equal-priority conflicting guards may resolve in opposite orders, diverging exactly where convergence is the confirmation criterion. The CNF α-rename fixes exported identities but verdict computation runs over store UUIDs.

**M7 — `impact` diverges on cyclic REF graphs, and the code comment claims the opposite.** `refgraph.go:32` states "UNION (not UNION ALL) dedups the working set, so cyclic REF graphs terminate" — but `depth` is a recursion column, so a cycle yields ever-new `(iri, depth+k)` tuples and the CTE does not terminate. Nothing in the schema prevents inserting cyclic REFs (`refgraph_test` checks the current store only).

**M8 — The Registry Law and corpus-breadth claims outrun the data.** "400+-instance corpus across 4 domains" = one Vietnamese labour-code .docx (392 units) plus three hand-built single-fixture ingestions. Θ(1) basis growth is asserted from four curated, author-selected points.

**M9 — Primary evidence is missing from the tree.** `CONFORMANCE-REVIEW.md` (the G1–G10 review the hardening answered), `research/D0v5-formal-spec.md`, and `coverage-matrix.md` (D8's companion) are all referenced and all absent. PROGRESS's stated baseline `spec/D0v5.md` does not exist (D0 actually lives at the root as `Computable Governance.md`). An auditor cannot re-derive the review trail.

**M10 — The registry is not bitemporal.** `registry(token, version int)` has no temporal coordinates; `kernel_instance_at()` does not resolve registry state at ⟨t_text, t_fact⟩. Verdicts at historical coordinates against a later registry version are unspecified — a direct gap in the I6/I8 story for exactly the retroactive-amendment scenarios the bitemporal design exists to serve.

---

## Minor Findings

1. **No LICENSE, no CITATION** — the repository cannot be legally reused or formally cited as a scientific artifact.
2. **Copyrighted ISO 9001:2015 PDF committed** (887 KB, root). Blocks any public release; contradicts D8's stated "structure-preserving paraphrase" discipline for licensed texts.
3. **Binary duplicates committed**: `Computable Governance.docx` alongside the .md; `D8.md` duplicated at root and `deliver/`.
4. **Release history is ceremonial**: 12 tags v0.4.0→v1.0.0 in ~36 hours; whole history spans 5 days. Tags do not correspond to externally meaningful releases.
5. **Makefile drift**: `make verify` fallback text says "T8, T1, C1 remain 'sorry'", contradicting the T1–T8-proved claims; header still says "recipes are placeholders".
6. **D1.5 header** says "Target prover: Lean 4 (mathlib4)" while the development is (deliberately, and correctly) mathlib-free.
7. **The CI `sorry`-guard is a plain grep** (`lean.yml:47`); commit `1d027e4` rewords a comment to dodge it — the guard cannot distinguish proof holes from prose and invites exactly this workaround.
8. **`e_writer` can bypass θ-discipline**: the `gks.via_transition` sentinel is settable by any session (`SELECT set_config('gks.via_transition','on',true)`), then `UPDATE e_machine` succeeds despite the guard trigger. Journal linearity is advisory against the writer role.
9. **`make validate` runs on authored fixtures** while README presents it as "the inter-compiler agreement harness"; the real-corpus number requires `cmd/interop` and a live DB.
10. **"Fleiss' κ" with n = 2 raters** throughout; D0 Appendix C defines κ as Fleiss' over panels. Not wrong, but the terminology dresses a pairwise comparison as a panel study.
11. **D1.3 §5's multiplication-boundedness checker** ("the checker rejects unbounded products") does not exist anywhere in the tree.
12. **Unbound variables evaluate to `false`** silently (`eval.go:159-162`) rather than failing; combined with the missing type-checker on ingest (DB validates only `jsonb_typeof(payload)='object'`), malformed payloads yield verdicts instead of errors.
13. **Persisted Ê coverage is thin**: 4 events, 6 verdicts in the live store; `defeated`/`violated` verdicts and the `deadline-shift`/`target-rebind` mutations have no persisted exemplar.

---

## Constitutional Compliance

**No.**

Faithfully implemented: the three Agent-0 rulings of 2026-07-09 (TIX columnar, |B|=6 enum since migration 0002; Force deprecated across D1.1/D1.2 with machine logic unchanged; `defeated` as a first-class verdict, migration 0005 + resolver mapping + tests). The I2/I6/I9 mechanisms match their constitutional statements at the `kernel_instance` level. Open-texture Detect/Contain (boundary tokens → conditional verdicts) matches D0 §7.1–7.2.

Non-compliant: (a) the frozen D0 text still mandates a seven-constructor basis and a Force-bearing NRM — the rulings amended a document that by its own terms cannot be amended, with no amendment protocol; (b) Phase-1 acceptance was issued without D0 §10.2's multi-compiler panels or §9.2.3's convergence, by redefining the phase boundary at acceptance time; (c) D0 §7.3's Record obligation (adjudicated resolutions entering K̂ as dated CLS via PWR) has no persisted exemplar; (d) I9 is constitutionally a bijection and implementationally a total single-valued map — the constitution's own text is violated by the benchmark's four-instances-per-locus mapping; (e) the Ê alphabet in the store is not the alphabet in D1.1 (§9.1.4 territory). Compliance with the rulings is real; compliance with the constitution as frozen is not.

---

## Scientific Credibility

| Dimension | Score | Justification |
|---|---|---|
| Honesty of claims | **4/10** | Genuine candor exists in the caveat layer (PROGRESS §8.2 "measures rule-robustness, not true independence"; interop REPORT caveat; α-rename `ambiguous` accounting; T1 scope note). But the headline layer — "T1–T8 all proved", "every invariant machine-checked", "Phase 1 ACCEPTED", "8.3 Met" — asserts what the caveat layer refutes, and the D1.5 ledger stamps "proved" on statements that were never formalized (F1). The caveats do not travel with the claims. |
| Scope discipline | **6/10** | Appendix A exclusions are respected; the sub-Turing discipline is real and consistently enforced in code; open texture is contained rather than faked. Deductions: scope was retroactively narrowed at the two moments it was binding (Phase-1 definition, F2; "proved" redefined per-obligation, F1). |
| Evidence quality | **3/10** | Strong: deterministic exports (reviewer-reproduced), DB-enforced invariants, real CI. Weak: the three pillars of acceptance — mechanization (vacuous models), validation (self-agreement), falsification (allowlist on self-encoded input) — plus a flagship benchmark whose executable diverges from its narrative (M2) and missing primary review documents (M9). |
| Reproducibility | **7/10** | Best dimension. Clean build; green tests; byte-identical CNF verified independently; schema auto-applies; corpus .docx committed; CI public within the private repo. Deductions: no published reference digests (export/ gitignored), κ baseline depends on live dev-DB history, licensed corpus (ISO PDF) not redistributable, no LICENSE. |
| Novelty | **3/10** | Zero related-work positioning — not one citation in the tree. The constructor set maps onto standard notions (NRM≈Hohfeldian duty, CLS≈counts-as, PWR≈legal power, GRD≈defeasible override, REF≈citation edge, VAL≈quantitative constraint); defeasible deontic engines (LegalRuleML/Regorous), mechanized legal DSLs (Catala, with peer-reviewed, machine-checked default-logic semantics), and bitemporal legal-text models (Akoma Ntoso; SQL:2011) are all prior art the repository must be measured against and never mentions. What is plausibly novel is the integration plus the falsificationist protocol (κ floors, CNF sealing, falsification ledger) — engineering-methodological, not mathematical. |
| Scientific rigor | **3/10** | The falsifiable *design* is commendable and rare. The *execution* is circular: one team (with an internal constitutional authority) authored the hypotheses, the encodings, the adversarial inputs, the floors, the tests, and the acceptance, in five days, and the one hypothesis the week actually falsified (seven irreducible constructors) was processed as a ruling rather than logged as a falsification datum. |

---

## Technical Debt

**Low-to-moderate, and well-contained.** The codebase is small, idiomatic, stdlib+pgx only, panic-free on production paths, with real tests (unit, invariant, adversarial-mutation, DB-integration). Deterministic-ID discipline and the append-only journal are maintainable foundations. Concrete debt items, in expected carrying-cost order:

1. **Spec-implementation divergence** (M1, M3, M4, M5) — every future contributor must reverse-engineer which of the three T's and which alphabet is authoritative; this is the debt most likely to compound.
2. **Unspecified verdict semantics** (M5) — any second implementation will fork behavior here first.
3. **Registry versioning vs bitemporality** (M10).
4. **Payload well-formedness** enforced only in Go constructors, not at the store boundary.
5. **Doc drift already accumulating at day 5** (phantom paths, stale Makefile text, duplicated D8) — with a single maintainer and no doc-consistency check in CI, expect linear growth.
6. Single maintainer / bus factor 1; the "Agent-0/Kiro/Guild A" role structure has no external continuity.

Estimated burden: one engineer-week to reconcile specs with implementation; ongoing low cost thereafter *if* a single-source-of-truth rule is adopted for grammar, alphabet, and verdict vocabulary.

---

## Research Debt

**Engineering (prerequisites to any confirmation claim):**
- A genuinely independent second implementation (different team, different stack) targeting the CNF/verdict contract — which first requires the contract to be *specified* (M5, M6: verdict vocabulary, defaults, tie semantics).
- A human-annotated gold corpus with documented guidelines; κ measured compiler-vs-human, panels ≥ 3.
- Either implement expression-schemas/stratified reflection or strike I7 from the invariant scorecard; its current "proved+tested" status is vacuous.
- Reconcile D1.3's grammar with the implemented AST; add the missing boundedness checker or delete the claim.
- Published, in-tree reference CNF digests per tagged release.

**Mathematics:**
- Make C1 well-posed: define representability and constructor simulation. Then settle the obvious first cases — this reviewer's analysis suggests **VAL is prima facie eliminable** (VAL(φ,u,⊕,t) reduces to a GRD condition using T's comparators over registry lookups; the campaign's own "wage ≥ minimum" unit demonstrates the overlap), and REF's indispensability turns entirely on the sorting of `source : Inst`. Two constructors (TIX, Force) have already been reduced; Hypothesis 2 currently trends toward *false*, and no experiment in the tree pushes either way.
- Actually formalize the D1.4 transition system (`Config`, `Step`) and prove T3/T6/T7 as stated — including a determinism theorem for the *implemented* resolver with its tie-break, replacing the false "ties are impossible".
- Type soundness for T (progress/preservation linking `infer` to `eval`); extend T1 over `Count`/`Window`.
- A formal sub-Turing statement: the complexity class of T-evaluation and its proof.

**Scientific validation:**
- External replication with preregistered floors, held-out corpora encoded by non-authors, and adversarial inputs generated outside the team.
- A versioned amendment protocol for D0, with the 7→6 reduction retroactively logged as a partial falsification of Hypothesis 1.
- Related-work survey positioning against LegalRuleML/LKIF/SBVR, Catala, defeasible deontic logic/Regorous, event calculus/InstAL, Datalog/ASP, and Akoma Ntoso — with an argument for what ⟨B, T⟩ adds beyond terminology.

---

## Recommendation for Phase-2

**Do not proceed to Phase 2 as currently defined. Revise Phase 1's claims to match its evidence, then re-submit.** Specifically, in order:

1. **Retract or re-scope the mechanization claims.** Relabel T2/T3/T5/T6/T7/T8 in the D1.5 ledger from "proved" to "modeled" (or "proved for a toy abstraction"), or do the real work: formalize `Step` and prove the stated theorems. The current ledger is the single largest threat to the program's credibility.
2. **Withdraw "Phase 1 ACCEPTED" status.** Reinstate D0 §10.2's own exit criterion (independent multi-compiler verification) as the Phase-1 gate, and treat the existing work as *Phase-1 preparation* — which, as engineering, it is genuinely good preparation.
3. **Issue D0 v1.2 through a defined amendment protocol**, recording: |B| = 7→6 as a partial falsification of Hypothesis 1; the Force deprecation; the actual Ê alphabet; the verdict vocabulary and its defaults; I9 as a total map, not a bijection. A constitution that can only be amended by pretending it wasn't frozen will invalidate every future acceptance issued under it.
4. **Specify the verdict contract** (vocabulary, unguarded-norm default, tie semantics — replace UUID tie-break with a content-derived key) so that an independent implementation is even possible, then commission one. This is the only experiment that can confirm the kernel hypothesis, and no amount of further first-party code substitutes for it.
5. **Add the scholarly apparatus**: related-work survey, LICENSE, citation metadata; remove the ISO PDF (restore the paraphrase discipline D8 already articulates).

No feature additions are required, and none should be accepted until items 1–4 land: every open scientific question this review identified is answerable with the constructors and machinery already in the tree.

---

*Every criticism above cites in-repository evidence; where evidence was absent it is marked as such. Verified by this reviewer on the live artifact: build/vet/test green; 410 rows / 6 constructors / 0 unmapped; CNF byte-determinism (SHA-256 `8b6304e196841257cca8a24b3a6f0177ad1fd5c4116ba63992ef25c703da9388`, two runs); CI run history green on `lean.yml`.*
