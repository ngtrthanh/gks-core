# Changelog

All notable changes to gks-core. Dates are UTC.

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
