-- Migration 0006 — continuous-ingestion ledger (append-only)
-- ---------------------------------------------------------------------------
-- Turns one-shot ingestion into a repeatable/automated pipeline: `ingest_run`
-- records one row per corpus per run, keyed by the SHA-256 of the source bytes.
-- A later run whose source digest already appears for that corpus is UP-TO-DATE
-- and is skipped — idempotency lives here (in the control plane), because the
-- kernel_instance EXCLUDE constraint deliberately rejects overlapping re-inserts,
-- so re-invoking an ingester on an unchanged corpus is neither safe nor needed.
--
-- Append-only for the least-privileged writer (INSERT/SELECT only).
CREATE TABLE IF NOT EXISTS ingestion_run (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  corpus           text        NOT NULL,           -- manifest corpus name
  source_path      text        NOT NULL,
  source_digest    text        NOT NULL,           -- sha256 hex of source bytes
  ingester         text        NOT NULL,           -- cmd invoked, or 'reconcile'
  instances_before integer     NOT NULL,
  instances_after  integer     NOT NULL,
  outcome          text        NOT NULL CHECK (outcome IN ('ingested','reconciled','skipped')),
  ran_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_run_corpus
    ON ingestion_run (corpus, ran_at DESC);

GRANT SELECT, INSERT ON ingestion_run TO e_writer;
