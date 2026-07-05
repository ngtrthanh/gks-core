# HANDOFF: gks-core — Conformance Patches & Validation Harness

> **ERRATA / STATUS (2026-07-06, Engineering)** — kept verbatim below; read with these corrections:
>
> - **Path drift:** the Go module lives in `compiler/`, not `engine/`. Every `engine/…` reference below (and in `docker-compose.yml` comments) means `compiler/…`. The Makefile already points at `compiler/` — the WP-8 "phantom compiler/" repair is moot.
> - **Missing referenced files:** `gks-core/CONFORMANCE-REVIEW.md`, `research/D0v5-formal-spec.md`, and `engineering/B1-kernel-schema.sql` are **not in this tree**. The G1–G10 findings survive only as summarized here and in `db/migrations/0002_hardening.sql` comments.
> - **Work-queue state:** WP-1 (RBAC + append-only trigger, single `e_writer` role instead of the reader/writer pair) and WP-4's `kernel_instance_at()` landed in migration 0002. I9 injectivity (`UNIQUE(source_map.instance_pk)`) landed in migration 0003, but `source_map` is **unpopulated** — the WP-2 locus backfill in the ingesters is still open. WP-5 α-renamed CNF export is done (`cnf_export`, tested identity/order-independent). **Open: WP-2 backfill, WP-3 (Ê persistence — the big one), WP-4 CLI flags, WP-6, WP-7, WP-8.**
> - Resolver determinism (I8 guard-ID tie-break) and the typed open-texture signal (`evaluator.BoundaryError`) are implemented with tests.

**From:** Guild A (Research) → **To:** Claude Code (Engineering)
**Version:** 2.0 (supersedes the v1 SQLite-MVP handoff — that plan is DEAD; a superior Go + PostgreSQL implementation now exists in `gks-core/` and all work continues there)
**Binding frame:** `research/D0v5-formal-spec.md` (D0 v1.1, FROZEN) · `gks-core/spec/D1.1–D1.5` · review findings in `gks-core/CONFORMANCE-REVIEW.md`

---

## 1. Context in 30 Seconds

The Governance Kernel ⟨B, T⟩ is frozen: 7 constructors {NRM, CLS, PWR, GRD, REF, VAL, TIX}, a sub-Turing term algebra, four decoupled layers (K̂ knowledge / Ŝ algebra / Ê execution / P̂ presentation), nine invariants I1–I9. `gks-core/` implements K̂ and Ŝ well (bitemporal PostgreSQL store, pure Go evaluator, defeasible resolver, Lean 4 scaffolding). **Your job: close the conformance gaps G1–G10 from the review, then build the validation harness.** Do not redesign what works; do not touch the kernel.

## 2. Required Reading (in order)

1. `gks-core/CONFORMANCE-REVIEW.md` — your work queue. G1–G10 with severities and recommended fixes.
2. `research/D0v5-formal-spec.md` — the constitution: constructors, T restrictions, invariants I1–I9, falsification criteria §9.1.
3. `gks-core/db/schema.sql` + `gks-core/engine/internal/kernel/` + `gks-core/engine/internal/evaluator/` — current state; read before changing.
4. `gks-core/spec/D1.4-Operational-Semantics.md` — Ê transition rules you'll be persisting.
5. `gks-core/deliver/D8.md` — benchmark fixtures with expected outcomes.
Reference only: `engineering/B1-kernel-schema.sql` (the retired SQLite design — mine it for the Ê/P̂ table shapes and writer-gate idea; do NOT resurrect SQLite).

## 3. Environment

`docker compose up -d db` from `gks-core/` (PostgreSQL 18-alpine, host port **5435**, db/user `governance`, password via `POSTGRES_PASSWORD` env or dev default). Schema auto-applies on first init from `db/schema.sql`. Go ≥ 1.25 (`engine/go.mod`), pgx/v5, stdlib-first. Lean 4 + mathlib4 for `mechanization/` (do not attempt proofs unless asked; keep them compiling as `sorry`). Before starting: run `go build ./... && go vet ./... && go test ./...` in `engine/` — the review was static; fix anything that doesn't build first.

## 4. Work Queue (strict order)

### WP-1: Enforce I2 by construction (review G1) — small
Migration: dedicated roles (`gks_reader`, `gks_writer`); `REVOKE UPDATE, DELETE ON kernel_instance FROM PUBLIC, gks_reader`; grant INSERT only to `gks_writer`; belt-and-braces trigger raising an exception on UPDATE/DELETE. All engine cmds except the Ê writer connect as `gks_reader`. Add a test proving an UPDATE attempt fails.

### WP-2: Add P̂ — artifact + source_map (G2) — small
New tables per the review: `artifact(artifact_id, corpus, title, content_hash UNIQUE)`; `source_map(instance_id UNIQUE NOT NULL → kernel_instance, artifact_id, locus, kind_token, span_start, span_end)` (I9 bijection via UNIQUE + NOT NULL). Backfill the D8 Run-1 loci from `deliver/D8.md` §Run-1 item 6 in `ingest_benchmark`. Test: every kernel row has exactly one source_map row.

### WP-3: Persist Ê (G3) — the big one
Tables: `world_event(event_id, event_type, agent_iri, occurred_at, payload jsonb)`; `e_machine(machine_id, subject_instance uuid, state ENUM('proposed','in-force','suspended','violated','discharged','extinguished'))` — the frozen alphabet, nothing else; `transition_log(machine_id, event_id, from_state, to_state, pwr_instance uuid NULL, mutation ENUM(NULL,'deadline-shift','target-rebind'), at)` — append-only (trigger-protected), with `pwr_instance` recording the authorizing power for every K̂-affecting write (this is how I2 and Ê wire together); `verdict(run …, subject_instance, eval_t_text timestamptz, eval_t_fact timestamptz, result ENUM('compliant','violated','conditional','inapplicable'), conditional_on text NULL)`.
Engine: a `machine` package replaying `world_event` through D1.4 transition rules; PWR-exercise processing (Run 2's concession → GRD creation + subject suspension); the two mutation hooks. **Resolver mapping (pending Agent 0 decision #3):** until ruled, treat resolver states IN_FORCE/DEFEATED/INACTIVE as π₃-internal and map: IN_FORCE→compliant-path, DEFEATED→(guard-suppressed, machine stays in-force), INACTIVE→inapplicable; document the mapping in code and flag it `// AGENT-0-DECISION-3`.

### WP-4: Temporal read discipline (G4) — small
SQL function `kernel_instance_at(tt timestamptz, tf timestamptz)` returning rows where `t_text @> tt AND t_fact @> tf`; every SELECT in every cmd goes through it; every verdict row records the coordinates used (I6). Add the Monday-Morning query: recursive REF traversal filtered by TIX overlap (`impact <target_iri> --at-text --at-fact`).

### WP-5: Canonical Normal Form export + determinism test (G7) — medium
`govc export-cnf` (new cmd): deterministic serialization — instances ordered by (constructor, source_map locus, stable tiebreak), UUIDs α-renamed to sequential IDs in that order, JSONB keys sorted, term trees normalized. Byte-identical across two runs on the same DB state (test this). Then `cnf-diff a b` reporting divergence by source-map anchor. This unblocks the entire §8 validation program.

### WP-6: Registry wiring + I4 test (G8) — small
`Lookup` resolution reads the versioned `registry` table (read-only, snapshot at eval start). Test T4-shape: bijectively rename all tokens, re-run verdicts, assert equality.

### WP-7: Numeric kind for VAL (G9) — small
Add a decimal/rational `Value` kind (NOT float64 — verdict determinism; use `math/big.Rat` or fixed-point int64 with declared scale). Wire `VALPayload` evaluation; fixture: D8 Run 6 KPI (ratio ≥ 0.95 × referenced threshold).

### WP-8: Validation harness scaffold (`validation/`) — medium
Inputs: N CNF exports of the same corpus from independent compilers. Outputs: Fleiss' κ over constructor-type assignment per source-map locus (exclude boundary tokens from denominator), verdict-agreement ratio over a shared event-trace suite. Constitutional floors κ ≥ 0.70, VA ≥ 0.90 are asserted, not configurable. Plus `make` target repairs: point Makefile at `engine/` (not the phantom `compiler/`), wire `test-compiler` to `go test ./...`.

## 5. Hard Guardrails (constitution-level)

- **I3 / Iron Rule:** never add a constructor, T-op, T-sort, or Ê state. An input that seems to need one → emit a `FALSIFICATION-CANDIDATE` record and halt that unit. Feature, not bug.
- **Frozen alphabet:** Ê states are exactly the six listed. Resolver vocabulary is π₃-internal only (WP-3 note).
- **I1:** no DB handle ever reaches `Eval`; keep `Environment` copy-on-bind.
- **I8:** π₂–π₄ pure in ⟨DB snapshot, eval coordinates⟩ — no `time.Now()` inside evaluation paths (accept `Now` as parameter, as `Environment.Now` already does), no map-iteration-order dependence (sort before emit, everywhere), no floats in verdict paths.
- **Open texture:** never "resolve" an `OpBoundary` to make a test pass; conditional verdicts are the correct output.
- **Determinism of migrations:** schema changes as numbered migration files (`db/migrations/00X_*.sql`), never edits to the applied baseline.

## 6. Pending Agent 0 Decisions — do NOT implement ahead of them

1. **`'TIX'` in the constructor ENUM** (review G6): leave the ENUM as-is; do not insert TIX rows; mark `TIXPayload` deprecated-pending-decision.
2. **`NRMPayload.Force` = "O|P|F" trichotomy** (G5): leave the field; add `// AGENT-0-DECISION-2` and do not build new logic that branches on P/F.
3. **Resolver-state vocabulary** (G3): implement the provisional mapping in WP-3, flagged.
File anything else that smells constitutional as an issue tagged `agent-0-decision` and pick the conservative reading.

## 7. Definition of Done

`go build ./... && go vet ./... && go test ./...` green; WP-1…WP-8 landed as separate commits referencing their WP and invariant (e.g. `WP-3: persist E-layer transition_log (I2,I8)`); invariant test suite covers I2 (UPDATE rejected), I4 (rename-stability), I5 (drop P̂ tables → verdicts unchanged), I6 (verdict rows carry coordinates), I9 (bijection); D8 Run 1 replays end-to-end from `world_event` to a persisted `verdict` row; Run 2 concession PWR executes (suspension via transition_log with `pwr_instance` set); CNF export byte-stable; one intentional negative fixture demonstrates the `FALSIFICATION-CANDIDATE` path; `CHANGELOG.md` updated per change; README/Makefile path drift fixed.

*Blocked? File `agent-0-decision` issues; everything else, decide conservatively and document in the PR description.*