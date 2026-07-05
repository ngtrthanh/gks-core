-- ============================================================================
-- 0003_i9_bijection.sql — enforce the injective half of I9 in the schema.
--
-- I9 (source-map bijection): every kernel instance maps to exactly one source
-- U-coordinate. Injectivity (at most one source_map row per kernel row) is
-- enforceable declaratively: UNIQUE(instance_pk). Totality (at least one row —
-- i.e. no unmapped kernel instance) cannot be a row-level constraint; it is
-- asserted by the validation harness / invariant test suite instead.
-- Idempotent. Apply as superuser (governance).
-- ============================================================================

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'source_map_instance_pk_unique'
  ) THEN
    ALTER TABLE source_map
      ADD CONSTRAINT source_map_instance_pk_unique UNIQUE (instance_pk);
  END IF;
END $$;

COMMENT ON CONSTRAINT source_map_instance_pk_unique ON source_map IS
  'I9 injectivity: at most one source U-coordinate per kernel instance.';

-- The plain index from 0002 is redundant next to the UNIQUE index.
DROP INDEX IF EXISTS idx_source_map_instance;
