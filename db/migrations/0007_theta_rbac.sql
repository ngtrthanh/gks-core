-- Migration 0007 — close the θ-guard bypass (exit-review Minor-8)
-- ---------------------------------------------------------------------------
-- Before: e_machine.state changes were gated only by the session GUC
-- `gks.via_transition`, which any `e_writer` session could forge
-- (`SELECT set_config('gks.via_transition','on',true); UPDATE e_machine ...`),
-- making the journal-linearity discipline advisory rather than enforced.
--
-- After: the discipline is privilege-enforced. `e_writer` loses direct UPDATE on
-- e_machine; the ONLY write path is INSERT into transition_log, whose
-- AFTER-INSERT trigger `transition_apply()` is now SECURITY DEFINER (runs as the
-- table owner) and performs the state update. e_writer keeps SELECT + INSERT
-- (machine creation) but can no longer mutate θ directly, forged GUC or not.
--
-- Apply as the schema owner.
ALTER FUNCTION transition_apply() SECURITY DEFINER;
ALTER FUNCTION transition_apply() SET search_path = public, pg_temp;
REVOKE UPDATE ON e_machine FROM e_writer;
