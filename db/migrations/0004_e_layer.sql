-- ============================================================================
-- 0004_e_layer.sql — persist the execution layer Ê (WP-3, spec D1.4).
--
-- Ê is the sole writer of K̂ (I2). Its state lives here:
--   world_event    — the event trace η₁…η_m (append-only record of fact).
--   e_machine      — θ: the CURRENT lifecycle state per subject instance.
--                    The frozen alphabet 𝒜 is exactly six states (I3) —
--                    resolver vocabulary (IN_FORCE/DEFEATED/INACTIVE) is
--                    π₃-internal and never appears here.
--   transition_log — append-only journal of every θ change; pwr_instance
--                    records the authorizing power for every K̂-affecting
--                    write (this is how I2 and Ê wire together).
--   verdict        — π₃ verdicts, each carrying the exact bitemporal
--                    coordinates used (I6).
-- Idempotent. Apply as superuser (governance).
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
    CREATE TYPE e_verdict AS ENUM ('compliant','violated','conditional','inapplicable');
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
RETURNS trigger LANGUAGE plpgsql AS $$
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
GRANT SELECT, INSERT, UPDATE ON e_machine TO e_writer;
