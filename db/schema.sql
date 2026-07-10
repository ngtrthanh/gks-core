-- ============================================================================
-- Governance Computing — Knowledge-Theory (K-hat) IR schema
-- Target: PostgreSQL 18 (see docker-compose.yml service `db`)
-- ----------------------------------------------------------------------------
-- Architectural decision: TimescaleDB is REJECTED. The kernel requires 2D
-- BITEMPORAL algebra (validity-in-text `t_text` x validity-in-fact `t_fact`),
-- not 1D time-series metrics. Native `tstzrange` + `btree_gist` express the
-- bitemporal constraints exactly and decidably (spec D1.1 §2, Invariant I6).
-- ============================================================================

-- btree_gist lets a GiST EXCLUDE constraint mix scalar equality (uuid `=`)
-- with range overlap (tstzrange `&&`) in one index.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- The seven closed kernel constructors (Invariant I3, kernel closure).
-- The six instantiable kernel constructors. TIX is NOT an instantiable
-- constructor: bitemporality is realized columnar via t_text/t_fact (I6).
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'kernel_constructor') THEN
    CREATE TYPE kernel_constructor AS ENUM
      ('NRM', 'CLS', 'PWR', 'GRD', 'REF', 'VAL');
  END IF;
END $$;

-- ----------------------------------------------------------------------------
-- kernel_instance: one bitemporally-indexed instance of a kernel constructor.
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kernel_instance (
    -- Physical, monotone surface key (append-only log position).
    pk           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- Logical identity of the kernel instance. A single logical instance may
    -- have several rows: successive, non-overlapping text-validity slices.
    instance_id  uuid        NOT NULL DEFAULT gen_random_uuid(),

    constructor  kernel_constructor NOT NULL,

    -- Layer S: the Semantic Algebra T AST / constructor fields, as JSONB.
    payload      jsonb       NOT NULL,

    -- TIX bitemporal index (Invariant I6). Half-open ranges '[)'; `infinity`
    -- for still-valid slices.
    t_text       tstzrange   NOT NULL,   -- validity-in-text time
    t_fact       tstzrange   NOT NULL,   -- validity-in-fact time

    -- Provenance of the append (single writer = execution layer, Invariant I2).
    recorded_at  timestamptz NOT NULL DEFAULT now(),

    -- Well-formedness guards.
    CONSTRAINT payload_is_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT t_text_nonempty   CHECK (NOT isempty(t_text)),
    CONSTRAINT t_fact_nonempty   CHECK (NOT isempty(t_fact)),
    -- I6 (bitemporal totality): explicit, finite lower coordinate required.
    CONSTRAINT tix_explicit_lower CHECK (NOT lower_inf(t_text) AND NOT lower_inf(t_fact)),

    -- CRITICAL (I6 totality + consistency): two rows sharing a logical
    -- instance_id may NOT overlap in their text-validity interval.
    CONSTRAINT no_overlapping_text_validity
        EXCLUDE USING gist (instance_id WITH =, t_text WITH &&)
);

COMMENT ON TABLE  kernel_instance IS
  'Append-only bitemporal store of kernel constructor instances (K-hat).';
COMMENT ON COLUMN kernel_instance.payload IS
  'Layer S semantic-algebra AST / constructor fields (JSONB).';
COMMENT ON CONSTRAINT no_overlapping_text_validity ON kernel_instance IS
  'No two slices of the same instance_id may overlap in t_text.';

-- ----------------------------------------------------------------------------
-- Indexes
-- ----------------------------------------------------------------------------
-- Containment/overlap queries over the two temporal dimensions.
CREATE INDEX IF NOT EXISTS idx_kernel_instance_ttext_gist
    ON kernel_instance USING gist (t_text);
CREATE INDEX IF NOT EXISTS idx_kernel_instance_tfact_gist
    ON kernel_instance USING gist (t_fact);

-- JSONB containment (@>) probes into the AST payload.
CREATE INDEX IF NOT EXISTS idx_kernel_instance_payload_gin
    ON kernel_instance USING gin (payload jsonb_path_ops);

-- Fast lookup of all slices of a logical instance, and by constructor.
CREATE INDEX IF NOT EXISTS idx_kernel_instance_instance_id
    ON kernel_instance (instance_id);
CREATE INDEX IF NOT EXISTS idx_kernel_instance_constructor
    ON kernel_instance (constructor);

-- ----------------------------------------------------------------------------
-- registry: finite, versioned, non-semantic parameter space R (Invariant I4).
-- Semantically inert lookup table (token -> value).
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS registry (
    token       text        NOT NULL,
    version     integer     NOT NULL DEFAULT 1,
    value       jsonb       NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (token, version)
);

COMMENT ON TABLE registry IS
  'Finite, versioned, semantically-inert registry R (Invariant I4).';

-- ============================================================================
-- Hardening (Phase 3.5) — kept in sync with db/migrations/0002_hardening.sql
-- ============================================================================

-- I2: append-only trigger — no UPDATE/DELETE on kernel_instance (single writer).
CREATE OR REPLACE FUNCTION kernel_instance_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'kernel_instance is append-only (Invariant I2): % is denied', TG_OP
    USING ERRCODE = 'restrict_violation';
END;
$$;

DROP TRIGGER IF EXISTS trg_kernel_instance_append_only ON kernel_instance;
CREATE TRIGGER trg_kernel_instance_append_only
  BEFORE UPDATE OR DELETE ON kernel_instance
  FOR EACH ROW EXECUTE FUNCTION kernel_instance_append_only();

-- G2: source map — bijective link to source U-coordinates (I9).
-- UNIQUE(instance_pk) enforces I9 injectivity (see migrations/0003); totality
-- (no unmapped kernel row) is asserted by the validation harness.
CREATE TABLE IF NOT EXISTS source_map (
  id          uuid      PRIMARY KEY DEFAULT gen_random_uuid(),
  instance_pk bigint    NOT NULL UNIQUE REFERENCES kernel_instance(pk) ON DELETE RESTRICT,
  locus       text      NOT NULL,
  kind        text      NOT NULL,
  span        int4range NOT NULL
);

-- G4: temporal-validity view — all engine reads go through this function.
CREATE OR REPLACE FUNCTION kernel_instance_at(tt timestamptz, tf timestamptz)
RETURNS SETOF kernel_instance
LANGUAGE sql STABLE AS $$
  SELECT * FROM kernel_instance
  WHERE t_text @> tt AND t_fact @> tf;
$$;

-- G1: RBAC — least-privilege engine role (SELECT + INSERT only).
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'e_writer') THEN
    CREATE ROLE e_writer LOGIN PASSWORD 'e_writer_dev';
  END IF;
END $$;

GRANT CONNECT ON DATABASE governance TO e_writer;
GRANT USAGE ON SCHEMA public TO e_writer;
GRANT SELECT, INSERT ON kernel_instance, registry, source_map TO e_writer;
REVOKE UPDATE, DELETE, TRUNCATE ON kernel_instance FROM e_writer;
GRANT EXECUTE ON FUNCTION kernel_instance_at(timestamptz, timestamptz) TO e_writer;
REVOKE UPDATE, DELETE ON kernel_instance FROM governance;

-- ============================================================================
-- E-layer (WP-3) — kept in sync with db/migrations/0004_e_layer.sql
-- ============================================================================

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'e_state') THEN
    CREATE TYPE e_state AS ENUM
      ('proposed','in-force','suspended','violated','discharged','extinguished');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'e_mutation') THEN
    CREATE TYPE e_mutation AS ENUM ('deadline-shift','target-rebind');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'e_verdict') THEN
    CREATE TYPE e_verdict AS ENUM ('compliant','violated','conditional','inapplicable','defeated');
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS world_event (
  event_id    uuid        PRIMARY KEY,
  event_type  text        NOT NULL,
  agent_iri   text        NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL,
  payload     jsonb       NOT NULL DEFAULT '{}'::jsonb,
  recorded_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT world_event_payload_object CHECK (jsonb_typeof(payload) = 'object')
);
CREATE INDEX IF NOT EXISTS idx_world_event_order ON world_event (occurred_at, event_id);

CREATE TABLE IF NOT EXISTS e_machine (
  machine_id       uuid    PRIMARY KEY,
  subject_instance uuid    NOT NULL UNIQUE,  -- one machine per kernel subject
  state            e_state NOT NULL DEFAULT 'proposed',
  updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transition_log (
  id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  machine_id   uuid        NOT NULL REFERENCES e_machine(machine_id),
  event_id     uuid        NULL REFERENCES world_event(event_id),
  from_state   e_state     NULL,           -- NULL = machine creation
  to_state     e_state     NOT NULL,
  pwr_instance uuid        NULL,           -- authorizing PWR for K̂-affecting writes
  mutation     e_mutation  NULL,           -- deadline-shift | target-rebind
  at           timestamptz NOT NULL,
  recorded_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_transition_log_machine ON transition_log (machine_id, id);

CREATE TABLE IF NOT EXISTS verdict (
  id               bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  run              text        NOT NULL,
  subject_instance uuid        NOT NULL,
  eval_t_text      timestamptz NOT NULL,   -- I6: coordinates are part of the verdict
  eval_t_fact      timestamptz NOT NULL,
  result           e_verdict   NOT NULL,
  conditional_on   text        NULL,       -- boundary token for 'conditional'
  recorded_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT conditional_has_token CHECK
    ((result = 'conditional') = (conditional_on IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS idx_verdict_subject ON verdict (subject_instance, recorded_at);

-- Append-only protection: the record of what happened is immutable.
CREATE OR REPLACE FUNCTION e_layer_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION '% is append-only (Invariant I2/Ê journal): % is denied', TG_TABLE_NAME, TG_OP
    USING ERRCODE = 'restrict_violation';
END;
$$;

DROP TRIGGER IF EXISTS trg_world_event_append_only ON world_event;
CREATE TRIGGER trg_world_event_append_only
  BEFORE UPDATE OR DELETE ON world_event
  FOR EACH ROW EXECUTE FUNCTION e_layer_append_only();

DROP TRIGGER IF EXISTS trg_transition_log_append_only ON transition_log;
CREATE TRIGGER trg_transition_log_append_only
  BEFORE UPDATE OR DELETE ON transition_log
  FOR EACH ROW EXECUTE FUNCTION e_layer_append_only();

-- θ discipline: e_machine.state is the CURRENT-state map and is only ever
-- changed BY the journal. Appending a transition_log row is the single write
-- path; this trigger checks journal linearity (from_state must match the
-- machine's current state) and applies the new state. Direct UPDATE/DELETE
-- of e_machine is denied below.
CREATE OR REPLACE FUNCTION transition_apply()
RETURNS trigger LANGUAGE plpgsql
  SECURITY DEFINER SET search_path = public, pg_temp AS $$
DECLARE
  cur e_state;
BEGIN
  SELECT state INTO cur FROM e_machine WHERE machine_id = NEW.machine_id FOR UPDATE;
  IF cur IS NULL THEN
    RAISE EXCEPTION 'transition for unknown machine %', NEW.machine_id;
  END IF;
  IF NEW.from_state IS DISTINCT FROM cur THEN
    RAISE EXCEPTION 'non-linear transition: machine % is %, journal says from %',
      NEW.machine_id, cur, NEW.from_state
      USING ERRCODE = 'restrict_violation';
  END IF;
  PERFORM set_config('gks.via_transition', 'on', true);
  UPDATE e_machine SET state = NEW.to_state, updated_at = now()
    WHERE machine_id = NEW.machine_id;
  PERFORM set_config('gks.via_transition', 'off', true);
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_transition_apply ON transition_log;
CREATE TRIGGER trg_transition_apply
  AFTER INSERT ON transition_log
  FOR EACH ROW EXECUTE FUNCTION transition_apply();

CREATE OR REPLACE FUNCTION e_machine_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'e_machine rows cannot be deleted (Ê journal integrity)'
      USING ERRCODE = 'restrict_violation';
  END IF;
  IF current_setting('gks.via_transition', true) IS DISTINCT FROM 'on' THEN
    RAISE EXCEPTION 'e_machine.state changes only via transition_log (Ê journal)'
      USING ERRCODE = 'restrict_violation';
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_e_machine_guard ON e_machine;
CREATE TRIGGER trg_e_machine_guard
  BEFORE UPDATE OR DELETE ON e_machine
  FOR EACH ROW EXECUTE FUNCTION e_machine_guard();

-- RBAC: the Ê writer appends events/transitions/verdicts and creates machines;
-- θ updates happen only through the transition trigger.
GRANT SELECT, INSERT ON world_event, transition_log, verdict TO e_writer;
-- e_writer may create machines (INSERT) and read them, but MUST NOT update θ
-- directly: state changes only through transition_log -> transition_apply()
-- (SECURITY DEFINER). Minor-8 fix: privilege-enforced, not via a forgeable GUC.
GRANT SELECT, INSERT ON e_machine TO e_writer;

-- ---- Continuous-ingestion ledger (migration 0006) --------------------------
-- Append-only record of ingestion runs; idempotency is by source_digest.
CREATE TABLE IF NOT EXISTS ingestion_run (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  corpus           text        NOT NULL,
  source_path      text        NOT NULL,
  source_digest    text        NOT NULL,
  ingester         text        NOT NULL,
  instances_before integer     NOT NULL,
  instances_after  integer     NOT NULL,
  outcome          text        NOT NULL CHECK (outcome IN ('ingested','reconciled','skipped')),
  ran_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ingestion_run_corpus ON ingestion_run (corpus, ran_at DESC);
GRANT SELECT, INSERT ON ingestion_run TO e_writer;
