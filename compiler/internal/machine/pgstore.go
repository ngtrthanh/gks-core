package machine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/kernel"
)

// PGStore persists Ê state to PostgreSQL (the tables of migration 0004). It
// is the production Store; the engine never sees the connection (I1 — only
// this adapter touches the DB).
type PGStore struct {
	Conn *pgx.Conn
	Ctx  context.Context
}

func (s *PGStore) EnsureMachine(subject kernel.UUID) (kernel.UUID, State, error) {
	mid := DetUUID("machine|" + subject.String())
	if _, err := s.Conn.Exec(s.Ctx, `
		INSERT INTO e_machine (machine_id, subject_instance, state)
		VALUES ($1::uuid, $2::uuid, 'proposed')
		ON CONFLICT (machine_id) DO NOTHING`,
		mid.String(), subject.String()); err != nil {
		return mid, "", fmt.Errorf("pgstore: ensure machine for %s: %w", subject, err)
	}
	var st string
	if err := s.Conn.QueryRow(s.Ctx,
		`SELECT state FROM e_machine WHERE machine_id = $1::uuid`, mid.String(),
	).Scan(&st); err != nil {
		return mid, "", fmt.Errorf("pgstore: read machine %s: %w", mid, err)
	}
	return mid, State(st), nil
}

func (s *PGStore) AppendTransition(t Transition) error {
	var eventID, pwr *string
	if t.EventID != nil {
		v := t.EventID.String()
		eventID = &v
	}
	if t.PWR != nil {
		v := t.PWR.String()
		pwr = &v
	}
	_, err := s.Conn.Exec(s.Ctx, `
		INSERT INTO transition_log (machine_id, event_id, from_state, to_state, pwr_instance, mutation, at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7)`,
		t.MachineID.String(), eventID, string(t.From), string(t.To), pwr, t.Mutation, t.At)
	if err != nil {
		return fmt.Errorf("pgstore: transition %s -> %s for %s: %w", t.From, t.To, t.MachineID, err)
	}
	return nil
}

func (s *PGStore) InsertKernel(ki kernel.KernelInstance, locus, kind string) (bool, error) {
	var exists bool
	if err := s.Conn.QueryRow(s.Ctx,
		`SELECT EXISTS (SELECT 1 FROM kernel_instance WHERE instance_id = $1::uuid)`,
		ki.InstanceID.String(),
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("pgstore: probe %s: %w", ki.InstanceID, err)
	}
	if exists {
		return false, nil
	}
	tText, err := ki.TText.Value()
	if err != nil {
		return false, err
	}
	tFact, err := ki.TFact.Value()
	if err != nil {
		return false, err
	}
	var pk int64
	if err := s.Conn.QueryRow(s.Ctx, `
		INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
		VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)
		RETURNING pk`,
		ki.InstanceID.String(), string(ki.Constructor), string(ki.Payload), tText, tFact,
	).Scan(&pk); err != nil {
		return false, fmt.Errorf("pgstore: insert %s: %w", ki.InstanceID, err)
	}
	if _, err := s.Conn.Exec(s.Ctx, `
		INSERT INTO source_map (instance_pk, locus, kind, span)
		VALUES ($1, $2, $3, int4range(0, $4))`,
		pk, locus, kind, len(ki.Payload)); err != nil {
		return false, fmt.Errorf("pgstore: source_map for %s: %w", ki.InstanceID, err)
	}
	return true, nil
}

func (s *PGStore) InsertVerdict(v Verdict) error {
	_, err := s.Conn.Exec(s.Ctx, `
		INSERT INTO verdict (run, subject_instance, eval_t_text, eval_t_fact, result, conditional_on)
		VALUES ($1, $2::uuid, $3, $4, $5, $6)`,
		v.Run, v.Subject.String(), v.TText, v.TFact, v.Result, v.ConditionalOn)
	if err != nil {
		return fmt.Errorf("pgstore: verdict for %s: %w", v.Subject, err)
	}
	return nil
}

// InsertEvent records a world_event row (idempotent on event_id, so fixture
// traces can be re-declared safely).
func (s *PGStore) InsertEvent(ev Event) error {
	raw, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("pgstore: marshal event %s payload: %w", ev.ID, err)
	}
	_, err = s.Conn.Exec(s.Ctx, `
		INSERT INTO world_event (event_id, event_type, agent_iri, occurred_at, payload)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb)
		ON CONFLICT (event_id) DO NOTHING`,
		ev.ID.String(), ev.Type, ev.Agent, ev.At, string(raw))
	if err != nil {
		return fmt.Errorf("pgstore: insert event %s: %w", ev.ID, err)
	}
	return nil
}

// LoadEvents returns the stored trace in replay order, optionally restricted
// to the given event ids.
func (s *PGStore) LoadEvents(ids []string) ([]Event, error) {
	q := `SELECT event_id::text, event_type, agent_iri, occurred_at, payload
	      FROM world_event`
	args := []any{}
	if len(ids) > 0 {
		q += ` WHERE event_id::text = ANY($1)`
		args = append(args, ids)
	}
	q += ` ORDER BY occurred_at, event_id`
	rows, err := s.Conn.Query(s.Ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("pgstore: load events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var (
			id, typ, agent string
			at             time.Time
			raw            []byte
		)
		if err := rows.Scan(&id, &typ, &agent, &at, &raw); err != nil {
			return nil, err
		}
		uid, err := kernel.ParseUUID(id)
		if err != nil {
			return nil, err
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("pgstore: event %s payload: %w", id, err)
		}
		out = append(out, Event{ID: uid, Type: typ, Agent: agent, At: at, Payload: payload})
	}
	return out, rows.Err()
}

// LoadKernel reads the K̂ snapshot at the eval coordinates via the temporal
// read discipline (kernel_instance_at, G4).
func (s *PGStore) LoadKernel(tt, tf time.Time) ([]kernel.KernelInstance, error) {
	rows, err := s.Conn.Query(s.Ctx, `
		SELECT instance_id::text, constructor::text, payload
		FROM kernel_instance_at($1, $2)`, tt, tf)
	if err != nil {
		return nil, fmt.Errorf("pgstore: load kernel: %w", err)
	}
	defer rows.Close()

	var out []kernel.KernelInstance
	for rows.Next() {
		var id, ctor string
		var payload []byte
		if err := rows.Scan(&id, &ctor, &payload); err != nil {
			return nil, err
		}
		uid, err := kernel.ParseUUID(id)
		if err != nil {
			return nil, err
		}
		out = append(out, kernel.KernelInstance{
			InstanceID:  uid,
			Constructor: kernel.Constructor(ctor),
			Payload:     kernel.JSONB(payload),
		})
	}
	return out, rows.Err()
}
