-- ============================================================================
-- 0002_hardening.sql — Phase 3.5 conformance remediation (Guild A findings)
-- Idempotent where possible. Apply as a superuser (governance).
--   G6 drop TIX from enum · I6 constraint · I2 append-only trigger ·
--   G2 source_map · G4 kernel_instance_at() · G1 e_writer RBAC
-- ============================================================================

-- ---- G6: drop TIX from the constructor enum --------------------------------
-- Bitemporality is realized columnar (t_text/t_fact); TIX is not an
-- instantiable constructor. Guarded so re-application is a no-op.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typname = 'kernel_constructor' AND e.enumlabel = 'TIX'
  ) THEN
    ALTER TYPE kernel_constructor RENAME TO kernel_constructor_old;
    CREATE TYPE kernel_constructor AS ENUM ('NRM','CLS','PWR','GRD','REF','VAL');
    ALTER TABLE kernel_instance
      ALTER COLUMN constructor TYPE kernel_constructor
      USING constructor::text::kernel_constructor;
    DROP TYPE kernel_constructor_old;
  END IF;
END $$;

-- ---- I6: promote bitemporal totality to a CHECK constraint -----------------
-- Every instance must carry an explicit (finite) lower coordinate in both
-- temporal dimensions — verdicts are computed relative to explicit coordinates.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'tix_explicit_lower'
  ) THEN
    ALTER TABLE kernel_instance
      ADD CONSTRAINT tix_explicit_lower
      CHECK (NOT lower_inf(t_text) AND NOT lower_inf(t_fact));
  END IF;
END $$;

-- ---- I2: append-only trigger (single writer; no UPDATE/DELETE) -------------
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

-- ---- G2: source map (debug-symbol contract, Presentation layer) ------------
CREATE TABLE IF NOT EXISTS source_map (
  id          uuid      PRIMARY KEY DEFAULT gen_random_uuid(),
  instance_pk bigint    NOT NULL REFERENCES kernel_instance(pk) ON DELETE RESTRICT,
  locus       text      NOT NULL,     -- source locator (e.g. "Điều 20, khoản 2")
  kind        text      NOT NULL,     -- U-kind: article|clause|point|sentence|cell
  span        int4range NOT NULL      -- character span within the source unit
);
CREATE INDEX IF NOT EXISTS idx_source_map_instance ON source_map(instance_pk);

COMMENT ON TABLE source_map IS
  'Source-Map: bijective link from kernel instances to source U-coordinates (I9).';

-- ---- G4: temporal-validity view (parameterized function) -------------------
-- Encapsulates the bitemporal WHERE clause; all engine reads go through it.
CREATE OR REPLACE FUNCTION kernel_instance_at(tt timestamptz, tf timestamptz)
RETURNS SETOF kernel_instance
LANGUAGE sql STABLE AS $$
  SELECT * FROM kernel_instance
  WHERE t_text @> tt AND t_fact @> tf;
$$;

-- ---- G1: role-based access control -----------------------------------------
-- Least-privilege engine role: SELECT + INSERT only, never UPDATE/DELETE.
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

-- Lock the interactive owner out of casual mutation. NOTE: `governance` is a
-- superuser in the dev container and bypasses privilege checks; the append-only
-- TRIGGER above is the hard, role-independent enforcement of I2.
REVOKE UPDATE, DELETE ON kernel_instance FROM governance;
