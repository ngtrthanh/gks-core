// Command ingest_run is the continuous-ingestion control plane (D0 §8.4). It
// turns the one-shot ingesters into a repeatable, idempotent, ledgered pipeline:
// a manifest lists corpora; each run compares the SHA-256 of every corpus's
// source bytes to the latest `ingestion_run` ledger entry and classifies it as
//
//	NEW         — never ingested (no ledger row)
//	CHANGED      — digest differs from the last recorded run
//	UP-TO-DATE   — digest matches the last run → nothing to do
//
// Idempotency lives HERE, not in the ingesters: kernel_instance's EXCLUDE
// constraint deliberately rejects overlapping re-inserts, so re-invoking an
// ingester on an unchanged corpus is neither safe nor needed. The ledger lets a
// scheduled/automated re-run be a safe no-op.
//
//	ingest_run                 # dry-run: report NEW/CHANGED/UP-TO-DATE
//	ingest_run --reconcile     # record current digests as the baseline (no ingest)
//	ingest_run --apply         # ingest NEW/CHANGED corpora; skip UP-TO-DATE
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env("PGUSER", "e_writer"), env("PGPASSWORD", "e_writer_dev"),
		env("PGHOST", "localhost"), env("PGPORT", "5435"), env("PGDATABASE", "governance"))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

type corpus struct {
	Name     string `json:"name"`
	Path     string `json:"path"`     // relative to the manifest's directory
	Ingester string `json:"ingester"` // cmd under ./cmd to invoke on NEW/CHANGED
}

func digestOf(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func main() {
	manifest := flag.String("manifest", "../data/corpora.json", "corpus manifest (JSON)")
	apply := flag.Bool("apply", false, "ingest NEW/CHANGED corpora (default: dry-run report)")
	reconcile := flag.Bool("reconcile", false, "record current digests as the baseline WITHOUT ingesting")
	flag.Parse()

	raw, err := os.ReadFile(*manifest)
	if err != nil {
		fatal(err)
	}
	var corpora []corpus
	if err := json.Unmarshal(raw, &corpora); err != nil {
		fatal(err)
	}
	baseDir := filepath.Dir(*manifest)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		fatal(err)
	}
	defer conn.Close(ctx)

	mode := "dry-run"
	if *reconcile {
		mode = "reconcile"
	} else if *apply {
		mode = "apply"
	}
	fmt.Printf("ingest_run [%s] — %d corpus(es) in %s\n", mode, len(corpora), *manifest)
	fmt.Println("────────────────────────────────────────────────────────────")

	for _, c := range corpora {
		path := filepath.Join(baseDir, c.Path)
		digest, err := digestOf(path)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", c.Name, err))
		}

		var last string
		err = conn.QueryRow(ctx,
			`SELECT source_digest FROM ingestion_run WHERE corpus=$1 ORDER BY ran_at DESC LIMIT 1`,
			c.Name).Scan(&last)
		status := "NEW"
		switch {
		case err == nil && last == digest:
			status = "UP-TO-DATE"
		case err == nil && last != digest:
			status = "CHANGED"
		case err != nil && err != pgx.ErrNoRows:
			fatal(err)
		}

		before := countInstances(ctx, conn)
		fmt.Printf(" %-14s %-12s digest=%s…\n", c.Name, status, digest[:12])

		switch {
		case *reconcile && status != "UP-TO-DATE":
			record(ctx, conn, c, path, digest, "reconcile", before, before, "reconciled")
			fmt.Printf("   → baseline recorded (no ingestion)\n")

		case *apply && status == "UP-TO-DATE":
			record(ctx, conn, c, path, digest, "-", before, before, "skipped")
			fmt.Printf("   → skipped (idempotent no-op)\n")

		case *apply && status != "UP-TO-DATE":
			fmt.Printf("   → ingesting via cmd/%s …\n", c.Ingester)
			cmd := exec.Command("go", "run", "./cmd/"+c.Ingester, path)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				fatal(fmt.Errorf("%s ingest: %w", c.Name, err))
			}
			after := countInstances(ctx, conn)
			record(ctx, conn, c, path, digest, c.Ingester, before, after, "ingested")
			fmt.Printf("   → ingested (+%d instances)\n", after-before)
		}
	}

	// Registry Law (Θ(1) basis): the store never exceeds the six constructors.
	fmt.Println("────────────────────────────────────────────────────────────")
	var basis string
	var n int
	if err := conn.QueryRow(ctx,
		`SELECT count(DISTINCT constructor), coalesce(string_agg(DISTINCT constructor::text, ',' ORDER BY constructor::text), '')
		 FROM kernel_instance`).Scan(&n, &basis); err != nil {
		fatal(err)
	}
	law := "HELD"
	if n > 6 {
		law = "BROKEN"
	}
	fmt.Printf("Registry Law: basis={%s} (|B|=%d ≤ 6): %s\n", basis, n, law)
	if n > 6 {
		os.Exit(1)
	}
}

func countInstances(ctx context.Context, conn *pgx.Conn) int {
	var n int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM kernel_instance`).Scan(&n); err != nil {
		fatal(err)
	}
	return n
}

func record(ctx context.Context, conn *pgx.Conn, c corpus, path, digest, ingester string,
	before, after int, outcome string) {
	_, err := conn.Exec(ctx, `
		INSERT INTO ingestion_run
		  (corpus, source_path, source_digest, ingester, instances_before, instances_after, outcome)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		c.Name, path, digest, ingester, before, after, outcome)
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ingest_run:", err)
	os.Exit(2)
}
