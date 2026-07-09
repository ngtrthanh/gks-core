package refgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/kernel"
)

// Fixture REF edges (deterministic ids; seeded once, append-only-safe):
//
//	rA:  urn:gkstest:A  --cite-->  urn:gkstest:T   valid from 2000
//	rB:  urn:gkstest:B  --cite-->  urn:gkstest:A   valid from 2020
//
// So Impact(T) is coordinate-sensitive: {A} before 2020, {A,B} after.
const (
	idRefA = "efac0000-0000-4000-8000-0000000000a1"
	idRefB = "efac0000-0000-4000-8000-0000000000b2"
)

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env("PGUSER", "governance"), env("PGPASSWORD", "governance_dev"),
		env("PGHOST", "localhost"), env("PGPORT", "5435"),
		env("PGDATABASE", "governance"))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func seedREF(t *testing.T, conn *pgx.Conn, id, source, target string, lower time.Time) {
	t.Helper()
	ctx := context.Background()
	var exists bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM kernel_instance WHERE instance_id = $1::uuid)`, id,
	).Scan(&exists); err != nil {
		t.Fatalf("probe %s: %v", id, err)
	}
	if exists {
		return
	}
	payload, _ := json.Marshal(kernel.REFPayload{Source: source, TargetIRI: target, Mode: "cite"})
	rng, err := kernel.Since(lower).Value()
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	var pk int64
	if err := conn.QueryRow(ctx, `
		INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
		VALUES ($1::uuid, 'REF', $2::jsonb, $3::tstzrange, $3::tstzrange)
		RETURNING pk`, id, string(payload), rng).Scan(&pk); err != nil {
		t.Fatalf("insert REF %s: %v", id, err)
	}
	if _, err := conn.Exec(ctx, `
		INSERT INTO source_map (instance_pk, locus, kind, span)
		VALUES ($1, $2, 'ref', int4range(0, $3))`,
		pk, "test:refgraph:"+source, len(payload)); err != nil {
		t.Fatalf("source_map %s: %v", id, err)
	}
}

func has(nodes []Node, iri string) bool {
	for _, n := range nodes {
		if n.IRI == iri {
			return true
		}
	}
	return false
}

// TestImpactCoordinateSensitive proves the impacted set changes with the read
// coordinates: rB's text-validity starts in 2020, so B is impacted only when
// querying at/after 2020 (WP-4 DoD).
func TestImpactCoordinateSensitive(t *testing.T) {
	conn, err := pgx.Connect(context.Background(), connString())
	if err != nil {
		t.Skipf("database unreachable, skipping impact test: %v", err)
	}
	defer conn.Close(context.Background())
	ctx := context.Background()

	e2000 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	e2020 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	seedREF(t, conn, idRefA, "urn:gkstest:A", "urn:gkstest:T", e2000)
	seedREF(t, conn, idRefB, "urn:gkstest:B", "urn:gkstest:A", e2020)

	before := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	after := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	early, err := Impact(ctx, conn, "urn:gkstest:T", before, before)
	if err != nil {
		t.Fatalf("impact@2010: %v", err)
	}
	late, err := Impact(ctx, conn, "urn:gkstest:T", after, after)
	if err != nil {
		t.Fatalf("impact@2021: %v", err)
	}

	if !has(early, "urn:gkstest:A") {
		t.Errorf("@2010 expected A impacted, got %v", early)
	}
	if has(early, "urn:gkstest:B") {
		t.Errorf("@2010 B should NOT be impacted (its REF slice starts 2020), got %v", early)
	}
	if !has(late, "urn:gkstest:A") || !has(late, "urn:gkstest:B") {
		t.Errorf("@2021 expected {A,B} impacted, got %v", late)
	}
	if !(len(late) > len(early)) {
		t.Fatalf("impacted set not coordinate-sensitive: early=%v late=%v", early, late)
	}
}
