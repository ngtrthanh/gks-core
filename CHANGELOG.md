# Changelog

All notable changes to gks-core. Dates are UTC.

## 2026-07-07 — Tracks A/B/C (post-handoff2; see handoff3.md)

- **Track A — mechanized semantics (Phase 2).** Mathlib-free Lean proofs in
  `mechanization/`: **T2** (I1 purity, `eval_is_pure := rfl`), **T3** (I8
  determinism, `eval_deterministic`), **T6** (I2 append-only monotonicity,
  `append_only_monotone`). **T8** (I7 well-foundedness) stated with `sorry`;
  T1/C1 remain conjectures (not faked). `make verify` now runs `lake build`
  where a Lean toolchain is present, else prints status. `spec/D1.5` ledger
  updated. NOTE: proofs are **not yet compiled in CI** — the Lean toolchain
  download is blocked in the current environment (elan host DNS fail, GitHub
  release assets unreachable); they are review-/CI-ready.
- **Track B — real inter-compiler agreement** (`cmd/interop`). An independent
  second constructor classifier (different cue set/precedence) re-classifies the
  live labour-code corpus; Fleiss' κ over **392** real loci = **0.7877**
  (≥ 0.70, floor met), 59 disagreements. Report + both CNF exports under
  `validation/interop/`. Caveat: one team maintains both classifiers → measures
  rule-robustness, not organizational independence.
- **Track C — falsification campaign** (`cmd/falsify`, `validation/falsification/`).
  Screens a held-out/adversarial input set: 5 benign units admitted (5
  constructors), 3 adversarial units (∀-unbounded, 8th constructor, fixpoint)
  **halted** with FALSIFICATION-CANDIDATE. Registry Law (basis ≤ 6, Θ(1))
  **HELD**; kernel not extended (I3).
- Track D (clause-level / multi-modality docx extraction) is the next step,
  now with a κ baseline (0.7877) to improve against.

## 2026-07-06 — WP-4, WP-6, WP-8 (handoff2)

- **WP-4 temporal read discipline (I6/G4).** New `internal/coord` parses
  `--at-text`/`--at-fact` (RFC3339, default now); every reading command
  (`verify_db`, `cnf_export`, `replay_d8`, `simulate_case`, `simulate_iso`)
  threads the coordinates into `kernel_instance_at($1,$2)` / `Engine.TText/TFact`.
  New `internal/refgraph.Impact` (recursive CTE over REF instances, temporally
  filtered) and `cmd/impact <target_iri>`; `ingest_kpi` now seeds the REF edge
  v4→P-11. DB-backed `refgraph` test proves the impacted set is
  coordinate-sensitive.
- **WP-6 registry snapshot + I4 (registry inertness).** `internal/registry`
  (`Snapshot`/`SnapshotAt`, promoted from `ingest_kpi.loadRegistry`) loads the
  versioned registry into exact Values; `Engine` gained a `Registry` field and
  every evaluating command loads a snapshot at eval start. `RenameTokens`/
  `RenameLookups` + a pure `TestI4RenameStability`: a bijective token rename
  leaves the verdict suite identical.
- **WP-8 validation harness (D0 §8.2).** `internal/validation`: Fleiss' κ over
  per-locus constructor assignment (open-texture loci excluded), verdict
  agreement over a shared suite, and the `FALSIFICATION-CANDIDATE` screen
  (constructor ∉ B or operator ∉ T ⇒ halt, kernel untouched). `cmd/validate`
  asserts the constitutional floors κ ≥ 0.70, VA ≥ 0.90; `make validate` runs
  it (testdata: κ=0.8425, VA=0.9000, halt demonstrated).
- **WP-8 corpus-derived coordinates (I8).** Ingesters no longer stamp
  `time.Now()`: `ingest_benchmark` uses the §121 statutory epoch, `ingest_iso`
  the ISO 9001:2015 date, `ingest_kpi` a declared policy epoch, and
  `ingest_docx` parses the promulgated "có hiệu lực" effective date (fixed
  fallback) — so CNF exports are reproducible across compilers.
- **Makefile:** `make validate` wired; `impact` available. CNF export remains
  byte-stable across runs on the same DB state (verified).

## 2026-07-06 — WP-7

- **Exact-rational VAL** (no float64 in verdict paths, I8): the evaluator gains
  a `KRat` value kind (`math/big.Rat`), a rational literal (`Lit.Rat`,
  `kernel.LitRat`) and an exact-division `OpRatio` (`kernel.Ratio`). Comparison
  and arithmetic promote to exact rationals when either operand is rational.
- `VALPayload` is now AST-driven — `Measure`/`Target` T-expressions and a
  comparator, evaluated via `VALPayload.AsExpr()`; the old `Target float64`
  field is gone.
- **`cmd/ingest_kpi`** persists D8 Run 6 (KPI-SEC-03): registry threshold
  `policy-p11-§4.threshold = 1` (append-only versioned, I4), v4 (VAL) and n7
  (NRM). It reads the registry back and evaluates the target
  `0.95 × reg(threshold)` exactly: 96/100 = 24/25 ≥ 19/20 → COMPLIANT,
  94/100 = 47/50 < 19/20 → VIOLATED. The 0.945-vs-0.95 boundary (where a
  float64 pipeline silently flips) is covered by unit tests.

## 2026-07-06 — WP-3

- **Ê execution layer persisted** (migration `0004_e_layer.sql`): `world_event`
  (append-only trace), `e_machine` (θ over the frozen six-state alphabet),
  `transition_log` (append-only journal; `pwr_instance` records the authorizing
  power on every K̂-affecting write — the I2↔Ê wiring), `verdict` (each row
  carries its `eval_t_text`/`eval_t_fact` coordinates, I6). θ changes only
  through `transition_log`, enforced by triggers (checked: direct UPDATE/DELETE
  of any journal table is rejected).
- **`internal/machine`**: replays the event trace through the D1.4 rules
  S-Activate / S-Defeat / S-Violate / S-Exercise. S-Exercise is the sole
  K̂-extending path (operand GRD appended with a deterministic id + source_map).
  Pure over ⟨event payloads, coordinates⟩; events replayed in
  (occurred_at, event_id) order, domains sorted (I8). Resolver→verdict mapping
  flagged `AGENT-0-DECISION-3`; no branching on the O|P|F trichotomy beyond
  S-Violate (`AGENT-0-DECISION-2`).
- **`cmd/replay_d8`** (`make replay-d8`): D8 Run 1 replays from `world_event`
  to a persisted `compliant` verdict; Run 2's concession-record PWR exercise
  appends the operand GRD, suspends n2a with `pwr_instance` set, and leaves
  n2b `conditional` on OT-1 — all persisted end-to-end.
- Invariant suite extended: I2 Ê-journal immutability. Unit tests for the
  machine cover both runs, S-Violate, replay idempotency and order-independence.

## 2026-07-06 — WP-2

- **ingesters populate `source_map` (I9 totality):** all three ingest
  commands insert one source-map row per kernel row in the same transaction
  (D8 §6 loci for Runs 1–2; `⟦corpus : Điều N ¶i : article|clause⟧` for docx).
  Each gains a `-backfill-sourcemap` mode that maps previously-ingested
  instances (docx rows matched by re-running the deterministic extraction
  walk); the live store's 404 rows are fully mapped.
- **invariant test suite started** (`internal/invariants`, skips without a
  DB): I2 UPDATE/DELETE rejected, I9 injectivity, I9 totality.
- CNF export re-sealed: records now carry real loci in their content key.

## 2026-07-06

- **Repository put under git**; baseline commit of the Phase 3.5 tree.
  `export/` (Ed25519 key material, CNF dumps) is gitignored.
- **evaluator:** open-texture boundary tokens now signal via typed
  `*BoundaryError` (`evaluator.IsBoundary`) instead of an untyped error, so
  callers can distinguish "conditional verdict required" from a genuine
  failure. `simulate_iso` asserts the typed signal.
- **evaluator (I8):** `Resolve` tie-breaks equal-priority guards by guard ID;
  verdicts no longer depend on DB row order. First tests for the semantic
  core (evaluator, resolver).
- **db (I9):** migration `0003_i9_bijection.sql` — `UNIQUE(source_map.instance_pk)`
  (injectivity enforced declaratively; totality remains a harness assertion).
  Baseline `schema.sql` synced. Note: `source_map` is still unpopulated —
  ingester backfill (WP-2) is open.
- **cnf (WP-5, I8):** exports α-rename store UUIDs to sequential ids in
  content order (constructor, locus, identity-masked payload shape, t_text,
  t_fact); payload-embedded references rewritten through the same map;
  content-key collisions warned. Byte-identical digest across runs verified
  on the live store. Pure-function tests for identity/order-independence.
- **docs:** README, `handoff.md` errata header, `docker-compose.yml` comment,
  and `validation/README.md` reconciled with the actual tree (`engine/` →
  `compiler/`; statuses now reflect what is implemented vs. open).

## Earlier (pre-git, summarized)

- Phase 3.5 hardening (migration 0002): TIX dropped from the constructor
  enum (columnar bitemporality), I2 append-only trigger, `source_map`,
  `kernel_instance_at()`, `e_writer` RBAC.
- Phase 3.1: Python MVP compiler dropped; Go + PostgreSQL only.
- Phase 1: D1.1–D1.5 formal specifications; Lean 4 scaffold.
