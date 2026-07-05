// Command ingest_benchmark constructs and persists the Layer-K instances of
// D8 Run 1: 26 U.S.C. §121 — exclusion of gain from sale of a principal
// residence, together with its §121(b)(3) anti-stacking exception.
//
// It builds four kernel instances and inserts them into the live PostgreSQL
// database (docker-compose service `db`, host port 5435) with bitemporal
// TSTZRANGE bounds of [now, infinity):
//
//	c1  CLS  — classify the subject property as a "principal_residence"
//	           (owned & used ≥ 2 of the last 5 years).
//	n1  NRM  — the taxpayer's permission to exclude the gain from gross income.
//	g1  GRD  — enabling guard: when c1 holds and gain ≤ the §121(b)(1) cap,
//	           activate n1 (priority 100).
//	g2  GRD  — anti-stacking exception §121(b)(3): if ≥ 1 prior §121 sale
//	           occurred within the last 2 years, defeat n1 (priority 200).
//
// NOTE: this program performs INSERTs. It is provided for inspection and is
// intended to be run explicitly by the operator (e.g. `go run
// ./cmd/ingest_benchmark`) once the `db` service is up.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/kernel"
)

// Stable instance identities for the benchmark (reproducible across runs).
const (
	idN1 = "a1000000-0000-4000-8000-000000000001" // NRM  §121(a) gain exclusion
	idC1 = "c1000000-0000-4000-8000-000000000001" // CLS  principal-residence classification
	idG1 = "d1000000-0000-4000-8000-000000000001" // GRD  enabling guard
	idG2 = "d2000000-0000-4000-8000-000000000001" // GRD  §121(b)(3) anti-stacking exception
)

// section121Cap is the single-filer exclusion ceiling, §121(b)(1).
const section121Cap = 250_000

// buildInstances materializes the four D8 Run 1 kernel instances valid from
// `now` onward in both temporal dimensions.
func buildInstances(now time.Time) ([]kernel.KernelInstance, error) {
	// c1: principal-residence test, §121(a). Owned & used as a principal
	// residence within the 5-year lookback, aggregating ≥ 2 years of use.
	c1 := kernel.CLSPayload{
		Entity:     "subject_property",
		ClassToken: "principal_residence",
		Condition: kernel.And(
			kernel.Within("P5Y", "", kernel.Pred("owned_and_used_as_residence",
				kernel.Var("property"), kernel.Var("taxpayer"))),
			kernel.Cmp(kernel.CmpGE, kernel.Var("aggregate_use_years"), kernel.LitInt(2)),
		),
	}

	// n1: the exclusion itself — a permission (+/P) held by the taxpayer.
	n1 := kernel.NRMPayload{
		Bearer:       "taxpayer:individual",
		Counterparty: "us_treasury:irs",
		Act:          "exclude_gain_from_gross_income",
		Sign:         "+",
		Force:        "P",
	}

	// g1: enabling guard — property classified as principal_residence and the
	// realized gain is within the single-filer cap ⇒ activate n1.
	g1 := kernel.GRDPayload{
		Condition: kernel.And(
			kernel.Pred("classified", kernel.Var("property"), kernel.LitStr("principal_residence")),
			kernel.Cmp(kernel.CmpLE, kernel.Var("realized_gain"), kernel.LitInt(section121Cap)),
		),
		Body:     []string{idN1},
		Priority: 100,
	}

	// g2: anti-stacking exception §121(b)(3) — if the taxpayer applied §121 to
	// any other sale within the last 2 years (bounded count ≥ 1), defeat n1.
	g2 := kernel.GRDPayload{
		Condition: kernel.CountCmp(
			"taxpayer_prior_sales",
			kernel.Within("P2Y", "", kernel.Pred("section121_exclusion_applied", kernel.Var("sale"))),
			kernel.CmpGE, 1,
		),
		Defeats:  []string{idN1},
		Priority: 200,
	}

	validity := kernel.Since(now) // [now, infinity) in both text and fact time

	type spec struct {
		id string
		c  kernel.Constructor
		p  any
	}
	specs := []spec{
		{idC1, kernel.CLS, c1},
		{idN1, kernel.NRM, n1},
		{idG1, kernel.GRD, g1},
		{idG2, kernel.GRD, g2},
	}

	out := make([]kernel.KernelInstance, 0, len(specs))
	for _, s := range specs {
		uid, err := kernel.ParseUUID(s.id)
		if err != nil {
			return nil, fmt.Errorf("parse uuid %s: %w", s.id, err)
		}
		raw, err := json.Marshal(s.p)
		if err != nil {
			return nil, fmt.Errorf("marshal %s payload: %w", s.c, err)
		}
		out = append(out, kernel.KernelInstance{
			InstanceID:  uid,
			Constructor: s.c,
			Payload:     kernel.JSONB(raw),
			TText:       validity,
			TFact:       validity,
		})
	}
	return out, nil
}

// connString builds the DSN, honoring DATABASE_URL or standard PG* env vars,
// defaulting to the docker-compose `db` service on host port 5435.
func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "e_writer"),
		getenv("PGPASSWORD", "e_writer_dev"),
		getenv("PGHOST", "127.0.0.1"),
		getenv("PGPORT", "5435"),
		getenv("PGDATABASE", "governance"),
	)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

const insertSQL = `
INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)`

func main() {
	instances, err := buildInstances(time.Now().UTC())
	if err != nil {
		log.Fatalf("build instances: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, ki := range instances {
		tText, err := ki.TText.Value()
		if err != nil {
			log.Fatalf("t_text literal for %s: %v", ki.InstanceID, err)
		}
		tFact, err := ki.TFact.Value()
		if err != nil {
			log.Fatalf("t_fact literal for %s: %v", ki.InstanceID, err)
		}
		if _, err := tx.Exec(ctx, insertSQL,
			ki.InstanceID.String(),
			string(ki.Constructor),
			string(ki.Payload),
			tText,
			tFact,
		); err != nil {
			log.Fatalf("insert %s %s: %v", ki.Constructor, ki.InstanceID, err)
		}
		log.Printf("ingested %-3s %s", ki.Constructor, ki.InstanceID)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	log.Printf("D8 Run 1 (26 U.S.C. §121) ingested: %d instances", len(instances))
}
