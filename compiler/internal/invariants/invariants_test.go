// Package invariants hosts the DB-backed invariant test suite from the
// handoff Definition of Done. Tests connect to the live dev database (same
// env conventions as the cmds) and skip when it is unreachable, so plain
// `go test ./...` stays green without infrastructure.
package invariants

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "governance"), getenv("PGPASSWORD", "governance_dev"),
		getenv("PGHOST", "localhost"), getenv("PGPORT", "5435"),
		getenv("PGDATABASE", "governance"))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func connect(t *testing.T) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), connString())
	if err != nil {
		t.Skipf("database unreachable, skipping invariant test: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// I2: kernel_instance is append-only — UPDATE and DELETE must be rejected
// regardless of role (trigger-enforced).
func TestI2AppendOnly(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	_, err := conn.Exec(ctx, `UPDATE kernel_instance SET constructor = 'CLS' WHERE pk = (SELECT min(pk) FROM kernel_instance)`)
	if err == nil {
		t.Fatal("UPDATE on kernel_instance succeeded; I2 (append-only) is not enforced")
	}
	_, err = conn.Exec(ctx, `DELETE FROM kernel_instance WHERE pk = (SELECT min(pk) FROM kernel_instance)`)
	if err == nil {
		t.Fatal("DELETE on kernel_instance succeeded; I2 (append-only) is not enforced")
	}
}

// I9 (injectivity): at most one source_map row per kernel row — the UNIQUE
// constraint from migration 0003 must exist and hold.
func TestI9Injectivity(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	var hasUnique bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'source_map_instance_pk_unique')`,
	).Scan(&hasUnique); err != nil {
		t.Fatalf("query pg_constraint: %v", err)
	}
	if !hasUnique {
		t.Fatal("UNIQUE(source_map.instance_pk) missing; apply db/migrations/0003_i9_bijection.sql")
	}

	var dups int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM (SELECT instance_pk FROM source_map GROUP BY 1 HAVING count(*) > 1) d`,
	).Scan(&dups); err != nil {
		t.Fatalf("query duplicates: %v", err)
	}
	if dups != 0 {
		t.Fatalf("%d kernel rows have multiple source_map rows (I9 injectivity violated)", dups)
	}
}

// I9 (totality): every kernel row has a source_map row.
func TestI9Totality(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	var total, unmapped int
	if err := conn.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE s.id IS NULL)
		FROM kernel_instance k
		LEFT JOIN source_map s ON s.instance_pk = k.pk`,
	).Scan(&total, &unmapped); err != nil {
		t.Fatalf("query: %v", err)
	}
	if total == 0 {
		t.Skip("empty store; nothing to assert")
	}
	if unmapped != 0 {
		t.Fatalf("%d of %d kernel rows have no source_map row (I9 totality violated); run the ingesters' -backfill-sourcemap mode", unmapped, total)
	}
}
