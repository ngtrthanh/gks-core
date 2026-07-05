# Governance Computing — Formal Specification and Compiler Repository

This repository is the **Formal Specification and Compiler Repository for Governance
Computing**. It hosts the machine-checkable specification, mechanized-proof scaffolding,
reference compiler, and (planned) validation harnesses for the kernel $\langle B, T \rangle$
frozen in D0 v1.1 (Constitutional Specification).

Phase 0 (kernel discovery, narrative) is closed. Phase 1 restated every normative claim
in D0 as an axiom, grammar, inference rule, or proof obligation (`spec/`). The tree is
now in **Phase 3.5 — conformance hardening** of the working Go + PostgreSQL
implementation (see `handoff.md` for the open work queue).

## Repository Layout

| Path | Contents | State |
| --- | --- | --- |
| `spec/` | Formal mathematical specifications (D1 series): objects, type system, grammar, operational semantics, proof obligations. | Draft, complete |
| `db/` | PostgreSQL bitemporal K̂ store: baseline `schema.sql` + numbered migrations. | Working |
| `compiler/` | Go reference implementation: kernel domain model, sub-Turing T evaluator, defeasible resolver, CNF export, ingest/simulate/verify commands. | Working |
| `mechanization/` | Lean 4 / mathlib4 development for the D1.5 obligations. | Scaffold (compiles with `sorry`s; toolchain not wired into `make verify`) |
| `validation/` | Reproducibility and inter-compiler agreement harnesses (Fleiss' $\kappa$, verdict agreement). | Not implemented |
| `deliver/`, `D8.md` | Benchmark fixture narratives (D8 Runs) with expected outcomes. | Reference |
| `data/` | Source corpora for ingestion (e.g. Vietnamese consolidated statute .docx). | Reference |
| `export/` | CNF dumps + Ed25519 seal material. **Gitignored — never committed.** | Generated |

## Specification Index (`spec/`)

| Document | Subject | Key invariants |
| --- | --- | --- |
| `D1.1-Mathematical-Objects.md` | Sets, relations, tuple definitions of the 7 constructors | I2, I6 |
| `D1.2-Type-System.md` | Base sorts, roles, typing judgments $\Gamma \vdash e : \tau$ | I3 |
| `D1.3-Grammar.md` | EBNF for the semantic algebra $T$; sub-Turing constraints | I1, I7 |
| `D1.4-Operational-Semantics.md` | Big-/small-step rules for layer $\hat{E}$ | I2, I8 |
| `D1.5-Proof-Obligations.md` | Theorem/conjecture ledger for Lean 4 | I1–I8, Hyp. 2 |

## Foundational Commitments

- **Kernel closure (I3).** The basis $B = \{\text{NRM}, \text{CLS}, \text{PWR}, \text{GRD}, \text{REF}, \text{VAL}, \text{TIX}\}$ and the language $T$ are closed. No eighth constructor. (In the store, TIX is realized *columnar* as the bitemporal coordinates `t_text`/`t_fact`, not as an instantiable row constructor — pending Agent 0 decision #1.)
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
| `simulate_case` | D8 Run 1 execution-layer simulation → verdict trace |
| `simulate_iso` | D8 Run 2 concession workflow (conditional verdict, PWR exercise, defeat) |
| `cnf_export` | Deterministic α-renamed Canonical Normal Form dump (WP-5) |
| `seal_export` / `verify_seal` | Detached Ed25519 signature over the CNF export |

Known open gaps (tracked in `handoff.md`, in its severity order): Ê persistence
(`world_event` / `e_machine` / `transition_log` / `verdict` tables — WP-3, verdicts are
currently computed in the simulate commands and not persisted), `source_map` population
by the ingesters (WP-2 backfill; the I9 UNIQUE constraint is in place), temporal read
discipline flags (WP-4 partial), registry wiring + I4 test (WP-6), exact numeric VAL
(WP-7 — `VALPayload.Target` is still `float64`), validation harness (WP-8).

## Build & Run

```sh
docker compose up -d db   # PostgreSQL 18, host port 5435 (schema auto-applies)
make test-compiler        # go vet + go test ./... in compiler/
make cnf-export           # deterministic CNF dump -> export/dump.cnf
make seal verify-seal     # Ed25519 seal + verification
make spec                 # sanity-check the D1 documents
make verify               # Lean mechanization — PLACEHOLDER (no toolchain wired)
make validate             # agreement harnesses — PLACEHOLDER (validation/ empty)
```

Migrations after the baseline are numbered files in `db/migrations/`; apply in order
with `psql` as superuser. The baseline `schema.sql` is kept in sync for fresh inits.

## Status

`spec/` — DRAFT. `compiler/`, `db/` — working, under conformance hardening (WP queue in
`handoff.md`). `mechanization/` — compiling scaffold, obligations still `sorry`.
`validation/` — not implemented. All proof obligations in `spec/D1.5` remain
**conjectures**. See `CHANGELOG.md` for the change history.
