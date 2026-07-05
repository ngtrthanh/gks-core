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
CREATE TABLE IF NOT EXISTS source_map (
  id          uuid      PRIMARY KEY DEFAULT gen_random_uuid(),
  instance_pk bigint    NOT NULL REFERENCES kernel_instance(pk) ON DELETE RESTRICT,
  locus       text      NOT NULL,
  kind        text      NOT NULL,
  span        int4range NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_source_map_instance ON source_map(instance_pk);

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
