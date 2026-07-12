# Governance Computing — Formal Specification and Compiler Repository

This repository is the **Formal Specification and Compiler Repository for Governance
Computing**. It hosts the machine-checkable specification, mechanized-proof scaffolding,
reference compiler, and (planned) validation harnesses for the kernel $\langle B, T \rangle$
frozen in D0 v1.1 (Constitutional Specification).

Phase 0 (kernel discovery, narrative) is closed. Phase 1 restated every normative claim
in D0 as an axiom, grammar, inference rule, or proof obligation (`spec/`). An
external Phase-1 exit review (`PHASE1-EXIT-REVIEW.md`, 2026-07-10) returned
**REVISE**: the engineering artifact is real and reproducible, but the earlier
"Phase 1 ACCEPTED" self-verdict overstated the evidence and **has been
withdrawn**. The project is now at **Phase-1 preparation**; acceptance requires
independent multi-compiler verification (D0 §10.2) that does not yet exist. The
mechanization is honest scaffolding, not machine-checked proof of the D0
invariants (see the corrected `spec/D1.5` ledger). Remediation is tracked in
`PROGRESS.md` (WS-A…WS-G) and `AGENT-0-DECISIONS.md`.

## Repository Layout

| Path | Contents | State |
| --- | --- | --- |
| `spec/` | Formal mathematical specifications (D1 series): objects, type system, grammar, operational semantics, proof obligations. | Draft, complete |
| `db/` | PostgreSQL bitemporal K̂ store: baseline `schema.sql` + numbered migrations. | Working |
| `compiler/` | Go reference implementation: kernel domain model, sub-Turing T evaluator, defeasible resolver, CNF export, ingest/simulate/verify commands. | Working |
| `mechanization/` | Lean 4 (mathlib-free) scaffold for the D1.5 obligations. | CI-compiles (Lean 4.31.0, zero `sorry`) but proves **simplified models, not the stated theorems** (F1): T2/T5 definitional, T3/T6/T7/T8 model-lemmas over unformalized `Step`/`Schema`, only T1 a scoped theorem. See corrected `spec/D1.5`. |
| — | Ê execution layer: bitemporal event replay → persisted verdicts. | Working (`db/migrations/0004`, `compiler/internal/machine`, `cmd/replay_d8`) |
| `validation/` | Reproducibility and inter-compiler agreement harnesses (Fleiss' $\kappa$, verdict agreement). | Implemented (`make validate`) |
| `D8.md` | Benchmark fixture narratives (D8 Runs) with expected outcomes. | Reference |
| `data/` | Source corpora for ingestion (e.g. Vietnamese consolidated statute .docx). | Reference |
| `export/` | CNF dumps + Ed25519 seal material. **Gitignored — never committed.** | Generated |

## Specification Index (`spec/`)

| Document | Subject | Key invariants |
| --- | --- | --- |
| `D1.1-Mathematical-Objects.md` | Sets, relations, tuple definitions of the 6 constructors | I2, I6 |
| `D1.2-Type-System.md` | Base sorts, roles, typing judgments $\Gamma \vdash e : \tau$ | I3 |
| `D1.3-Grammar.md` | EBNF for the semantic algebra $T$; sub-Turing constraints | I1, I7 |
| `D1.4-Operational-Semantics.md` | Big-/small-step rules for layer $\hat{E}$ | I2, I8 |
| `D1.5-Proof-Obligations.md` | Theorem/conjecture ledger for Lean 4 | I1–I8, Hyp. 2 |

## Foundational Commitments

- **Kernel closure (I3).** The basis $B = \{\text{NRM}, \text{CLS}, \text{PWR}, \text{GRD}, \text{REF}, \text{VAL}\}$ and the language $T$ are closed. No seventh constructor. TIX is **not** a constructor but the bitemporal index $\tau(x)$ every instance carries, realized *columnar* as `t_text`/`t_fact` (Agent-0 Ruling 1).
- **Read-only algebra (I1).** $T$ has empty write-effect by construction; the evaluator receives no DB handle and takes `Now` as a parameter.
- **Single writer (I2).** Only $\hat{E}$ extends $\hat{K}$, append-only — enforced by trigger and RBAC in `db/schema.sql`.
- **Determinism (I8).** Evaluation and export are pure in ⟨DB snapshot, eval coordinates⟩: guard ties break on guard ID, exports α-rename UUIDs to content-ordered ids.

## Implementation (`compiler/`)

Commands (all Go, stdlib + pgx; run with the `db` service up):

| Command | Purpose |
| --- | --- |
| `ingest_benchmark` | Persist D8 Run 1 (26 U.S.C. §121) kernel instances |
| `ingest_iso` | Persist D8 Run 2 (ISO 9001:2015 §8.7) incl. PWR + open-texture boundary |
| `ingest_docx` | Unsupervised structural ingestion of a legal .docx corpus |
| `verify_db` | Read-only report of stored instances (identity, TIX bounds, AST) |
| `simulate_case` | D8 Run 1 in-memory verdict trace (pre-Ê demo; superseded by `replay_d8`) |
| `simulate_iso` | D8 Run 2 in-memory concession workflow (pre-Ê demo; superseded by `replay_d8`) |
| `replay_d8` | Replay D8 traces through the **persisted** Ê layer → `verdict` rows |
| `ingest_kpi` | Persist + evaluate D8 Run 6 VAL (exact-rational KPI vs referenced threshold) |
| `cnf_export` | Deterministic α-renamed Canonical Normal Form dump (WP-5) |
| `seal_export` / `verify_seal` | Detached Ed25519 signature over the CNF export |
| `serve` | Read-only web console over the live store (`make serve`, port 8787) |

All work packages in `handoff.md` / `handoff2.md` are now landed. Done: WP-1
(RBAC + append-only), WP-2 (`source_map` population, I9), WP-3 (Ê persistence),
WP-4 (temporal-read CLI flags `--at-text`/`--at-fact` + `impact` REF-traversal),
WP-5 (α-renamed CNF), WP-6 (registry-snapshot `Lookup` via `internal/registry`
+ I4 rename-stability test), WP-7 (exact-rational VAL — no float64), WP-8
(validation harness: Fleiss' $\kappa$ + verdict-agreement with asserted floors,
corpus-derived coordinates, `FALSIFICATION-CANDIDATE` halt path).

## Build & Run

```sh
docker compose up -d db   # PostgreSQL 18, host port 5435 (schema auto-applies)
make test-compiler        # go vet + go test ./... in compiler/
make cnf-export           # deterministic CNF dump -> export/dump.cnf
make seal verify-seal     # Ed25519 seal + verification
make spec                 # sanity-check the D1 documents
make verify               # Lean mechanization — lake build if Lean present, else status
make validate             # inter-compiler agreement harness (κ≥0.70, VA≥0.90)
make serve                # kernel console (web UI) -> http://localhost:8787
```

Temporal reads accept `--at-text`/`--at-fact` (RFC3339; default now); the
`impact <target_iri>` command traverses REF edges valid at those coordinates.

### Kernel console (`cmd/serve`)

`make serve` starts a strictly read-only HTTP console on `http://localhost:8787`
(override with `-addr`; store DSN via `DATABASE_URL`/`PG*` env, same convention as
the other commands). Every read goes through `kernel_instance_at(tt, tf)` at the
coordinates set by the **temporal cursor** in the header — move it to re-read the
entire kernel as of any $(t_{\text{text}}, t_{\text{fact}})$ pair. Views: the
instance ledger (census by constructor, search, per-instance drawer with rendered
T-AST formulas, bitemporal slices, and inbound REF designations), the verdict
ledger (status + open-texture `conditional_on` tokens), and REF impact traversal.
The server issues only `SELECT`s; it holds no write path (I1/I2 are untouched).

Migrations after the baseline are numbered files in `db/migrations/`; apply in order
with `psql` as superuser. The baseline `schema.sql` is kept in sync for fresh inits.

## Status

`spec/` — DRAFT. `compiler/`, `db/` — working; the `handoff.md`/`handoff2.md` WP
queue is fully landed. `mechanization/` — a mathlib-free Lean scaffold that
**CI-compiles** (`.github/workflows/lean.yml`, Lean 4.31.0, zero `sorry`) but, per
the exit review (F1), proves *simplified models* rather than the D1.5 statements as
written: T2/T5 are definitional, T3/T6/T7/T8 are model-lemmas over the unformalized
`Step`/`Schema` structures, and only T1 is a genuine (scoped) theorem. `validation/`
— implemented (`make validate`), but the numbers are **first-party** (no human gold
standard, no independent compiler — D0 §10.2 acceptance gate unmet). See
`spec/D1.5`, `PROGRESS.md`, and `PHASE1-EXIT-REVIEW.md`. C1 (minimality) is an open,
not-yet-well-posed conjecture. See `CHANGELOG.md` for history.
