# HANDOFF v3: gks-core — Remaining Conformance Work (WP-4, WP-6, WP-8)

**From:** Claude Code (Engineering) → **To:** Kiro (Engineering)
**Supersedes:** `handoff.md` (v2). That document's work queue is now ~60% landed;
this file records what is DONE, what REMAINS, and the constitution-level guardrails
that do not change. Read `handoff.md` for the original framing and the D8 fixture
narratives; read THIS file for current state and your queue.
**Binding frame:** `spec/D1.1–D1.5` (FROZEN kernel ⟨B, T⟩), `deliver/D8.md` (benchmark
fixtures), `CHANGELOG.md` (what changed and why).

---

## 1. Context in 30 seconds

The Governance Kernel ⟨B, T⟩ is frozen: 6 instantiable constructors
{NRM, CLS, PWR, GRD, REF, VAL} (TIX is realized columnar as `t_text`/`t_fact`,
not a row constructor), a sub-Turing term algebra, four layers (K̂ knowledge /
Ŝ algebra / Ê execution / P̂ presentation), invariants I1–I9. The reference
implementation is **Go + PostgreSQL in `compiler/`** (NOT `engine/` — the v2
handoff had that path wrong). It builds, vets, and tests clean; the dev DB is a
docker Postgres on host port 5435.

**Your job:** land the last three work packages — WP-4 (temporal read CLI),
WP-6 (registry-snapshot Lookup + I4 test), WP-8 (validation harness). Do not
redesign what works; do not touch the frozen kernel.

## 2. Current state (verified 2026-07-06)

Live dev DB (`governance`):

| Thing | Count | Notes |
| --- | --- | --- |
| `kernel_instance` | 407 | CLS 81, GRD 79, NRM 173, PWR 73, VAL 1 |
| `source_map` | 407 | I9 total: every kernel row mapped |
| `world_event` | 4 | D8 Run 1 (1) + Run 2 (3) traces |
| `verdict` | 5 | Run 1 compliant; Run 2 = {inapplicable, conditional/OT-1, compliant, compliant} |
| `registry` | 1 token | `policy-p11-§4.threshold` = 1 (exact rational) |

`go build ./... && go vet ./... && go test ./...` — all green. Test packages:
`internal/cnf`, `internal/evaluator`, `internal/machine`, `internal/invariants`
(DB-backed, skips when DB down), `cmd/ingest_benchmark`.

### What is DONE (do not redo)

- **WP-1 — I2 by construction.** `db/migrations/0002`: append-only trigger on
  `kernel_instance`, least-privilege `e_writer` role (SELECT+INSERT, no
  UPDATE/DELETE). Invariant test `TestI2AppendOnly` proves UPDATE/DELETE fail.
- **WP-2 — P̂ source_map.** All three ingesters write one `source_map` row per
  kernel row in the same tx; each has a `-backfill-sourcemap` mode. I9
  injectivity is a UNIQUE constraint (`db/migrations/0003`); totality is
  asserted by `TestI9Totality`. 407/407 mapped.
- **WP-3 — Ê persisted** (the v2 "big one"). `db/migrations/0004`: `world_event`,
  `e_machine` (θ over the frozen 6-state alphabet), append-only `transition_log`
  (`pwr_instance` records the authorizing power on every K̂-affecting write),
  `verdict` (carries `eval_t_text`/`eval_t_fact`, I6). θ changes ONLY through
  `transition_log` (trigger-enforced; direct mutation of any journal table is
  rejected — `TestI2ELayerJournal`). `internal/machine` replays the trace
  through D1.4 rules S-Activate/S-Defeat/S-Violate/S-Exercise; S-Exercise is the
  sole K̂-extending path. `cmd/replay_d8` (`make replay-d8`) runs Runs 1–2
  end-to-end from `world_event` to persisted verdicts.
- **WP-5 — α-renamed CNF export.** `cmd/cnf_export` orders records by
  (constructor, locus, identity-masked payload shape, t_text, t_fact) and
  assigns sequential ids `k000001…`, rewriting payload references through the
  same map, so independent compilers emit comparable dumps. Byte-stable digest
  verified across runs. Ed25519 seal via `seal_export`/`verify_seal`.
- **WP-7 — exact-rational VAL.** No float64 on the verdict path: `KRat`
  (`math/big.Rat`) value kind, `Lit.Rat` literal, `OpRatio` exact division;
  `VALPayload` is AST-driven (`Measure`/`Target`/`Comparator`, via `AsExpr()`).
  `cmd/ingest_kpi` persists + evaluates D8 Run 6 exactly.

### AGENT-0 decisions still provisional (flagged in code — do not "resolve" these)

- `AGENT-0-DECISION-2` (`internal/machine`): S-Violate is the ONLY branch on the
  NRM Force O|P|F trichotomy. Do not add P/F branching.
- `AGENT-0-DECISION-3` (`internal/machine` `evaluate`): resolver→verdict mapping
  is provisional — IN_FORCE→compliant-path, DEFEATED→inapplicable
  (guard-suppressed, machine state untouched), INACTIVE→inapplicable. Documented
  in code; leave until Agent 0 rules.

## 3. Environment

```sh
docker compose up -d db          # Postgres 18, host port 5435, db/user governance
                                 # schema auto-applies from db/schema.sql on first init
cd compiler && go build ./... && go vet ./... && go test ./...
```

Go ≥ 1.25, pgx/v5, stdlib-first. `DATABASE_URL` or `PG*` env vars override the
default DSN (default user `e_writer`, password `e_writer_dev`). Migrations after
the baseline are numbered files in `db/migrations/`, applied in order as
superuser (`governance`); `db/schema.sql` is kept in sync for fresh inits — when
you add migration `0005`, append its body to `schema.sql` too and verify a fresh
init with a throwaway database (see the pattern in the WP-3 commit).

## 4. Your work queue (strict order)

### WP-4: Temporal read discipline — CLI flags + impact query (small)

Reads already go through the SQL function `kernel_instance_at(tt, tf)`. What's
missing is operator control of the coordinates and the REF-traversal query.

1. Add `--at-text` / `--at-fact` flags (RFC3339; default `now()`) to every
   READING command: `verify_db`, `cnf_export`, `replay_d8`, `simulate_case`,
   `simulate_iso`. Thread the parsed coordinates into the existing
   `kernel_instance_at($1,$2)` calls and into `Engine.TText/TFact`.
2. **REF is not yet a persisted constructor in any fixture.** The D8 runs
   describe REF edges (`n2a→8.6`, KPI→P-11) but no ingester writes REF rows or a
   REF adjacency. Before the impact query you must decide the REF payload shape
   (`REFPayload` exists in `kernel/models.go`: Source, TargetIRI, Mode) and how
   edges are stored/queried. Recommended: a recursive CTE over REF instances
   whose `target_iri` matches, filtered by TIX overlap at the eval coordinates.
3. New cmd `impact <target_iri> --at-text --at-fact`: recursive REF traversal
   returning every instance transitively referencing the target, valid at the
   coordinates (the "Monday-Morning query" from D8 Run 3/6). Test: amend a
   referenced instance's `t_text` (append a new slice) and assert the impacted
   set changes across coordinates.

**Guardrail:** no `time.Now()` inside evaluation — coordinates are inputs
(`Environment.Now` already is one). Flags parse to explicit times.

### WP-6: Registry-snapshot Lookup + I4 rename-stability test (small)

`OpLookup` reads `Environment.Registry` (a `map[string]Value`). `cmd/ingest_kpi`
already has a `loadRegistry` that reads the versioned `registry` table into
exact `Value`s (including `{"rat":"..."}` → `KRat`) — **promote that into a
shared package** (e.g. `internal/registry`) and have every evaluating command
load a registry SNAPSHOT at eval start (latest version per token, or the version
valid at `--at-text`). The registry is INSERT-only for `e_writer` and versioned
(I4) — never mutate in place; append a new version (see the `registrySQL`
pattern in `ingest_kpi`).

I4 test (the point of the invariant): bijectively rename every registry token
(and every `Lookup` reference to it), re-run a verdict suite, assert identical
verdicts. Registry is semantically inert — renaming tokens must not move any
verdict. Put it in `internal/invariants` (DB-backed) or as a pure test over a
fixture registry + norm set.

### WP-8: Validation harness + text-derived coordinates (medium — do last)

Everything above feeds this. Location: `validation/`.

1. **Blocker to fix first:** ingesters stamp `t_text`/`t_fact` with wall-clock
   `time.Now()`. Two independent compilers therefore never byte-agree on a CNF
   export even for identical input — which defeats the whole agreement program.
   Ingesters must DERIVE coordinates from the corpus (promulgation/effective
   dates in the text; for the docx corpus, parse the "có hiệu lực" / effective-
   date lines, else a declared fixed epoch per corpus). Until this lands,
   cross-compiler κ is meaningless; document the assumption if you scope it out.
2. Comparison driver here: inputs = N CNF exports of the same corpus from
   independent compilers. Outputs = Fleiss' κ over constructor-type assignment
   per source-map locus (exclude open-texture boundary tokens from the
   denominator), and a verdict-agreement ratio over a shared event-trace suite.
3. Constitutional floors are ASSERTED, not configurable: κ ≥ 0.70, VA ≥ 0.90.
4. One intentional **negative fixture** demonstrating the `FALSIFICATION-CANDIDATE`
   path: an input that appears to need a 7th constructor / an 8th Ê state / an
   unbounded quantifier → the unit emits a `FALSIFICATION-CANDIDATE` record and
   halts, rather than stretching the kernel. (This is a Definition-of-Done item
   from v2 that is still unbuilt.)
5. Wire `make validate` to actually run the harness (currently a placeholder).

## 5. Hard guardrails (constitution-level — unchanged from v2)

- **I3 / Iron Rule:** never add a constructor, T-op, T-sort, or Ê state. Input
  that seems to need one → emit a `FALSIFICATION-CANDIDATE` and halt that unit.
  Feature, not bug.
- **Frozen alphabets:** Ê states are exactly the six in the `e_state` enum.
  Resolver vocabulary (IN_FORCE/DEFEATED/INACTIVE) is π₃-internal and never
  persisted.
- **I1:** no DB handle reaches `Eval`; only `internal/machine.PGStore` touches
  the DB. Keep `Environment` copy-on-bind.
- **I8:** evaluation and export are pure in ⟨DB snapshot, eval coordinates⟩ — no
  `time.Now()` in evaluation, no map-iteration-order dependence (sort before
  emit, everywhere), no float64 in verdict paths (use `math/big.Rat` — the
  machinery exists).
- **Open texture:** never "resolve" an `OpBoundary` to make a test pass;
  conditional verdicts are the correct output. `evaluator.IsBoundary` /
  `*BoundaryError` distinguish this from a real failure — use it.
- **Migrations:** numbered files in `db/migrations/00X_*.sql`; sync `schema.sql`;
  never edit an applied migration.

## 6. Definition of done (for your three WPs)

`go build ./... && go vet ./... && go test ./...` green. WP-4/6/8 landed as
separate commits referencing their WP and invariant. New invariant tests: I4
(rename-stability). `impact` query returns coordinate-sensitive results. CNF
exports from two runs on the same DB state are byte-identical (already true —
keep it true). One `FALSIFICATION-CANDIDATE` negative fixture demonstrates the
halt path. `make validate` runs the κ / verdict-agreement harness and asserts
the floors. `CHANGELOG.md` updated per change; README status table updated.

## 7. Known caveats discovered during WP-1..7 (context, not tasks)

- **docx extraction is shallow by design.** `ingest_docx` is heuristic and
  LLM-free: ~29% paragraph yield, one constructor per paragraph (first cue
  wins — a paragraph with both a right and a prohibition is classified once),
  no bearer/counterparty/act tuple extraction. It scales fine (~15k paragraphs/s,
  memory linear; tested to 270k paragraphs). If WP-8's κ needs richer structure,
  clause-level splitting and multi-modality are the levers — but hold until the
  harness gives you a number to improve against.
- **`ngày làm việc` (working days) is conflated with calendar days** in the ISO
  duration heuristic. Minor; flag if it matters for a temporal fixture.
- The two `simulate_*` commands are pre-Ê in-memory demos, superseded by
  `replay_d8`. Keep or retire at your discretion; they still pass.

*Blocked on something constitutional? File it as an `agent-0-decision` note and
pick the conservative reading — do not amend the kernel to unblock.*
