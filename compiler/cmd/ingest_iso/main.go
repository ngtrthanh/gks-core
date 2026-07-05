// Command ingest_iso constructs and persists the Layer-K instances of D8 Run 2:
// ISO 9001:2015 Clause 8.7 (Control of nonconforming outputs). It builds eight
// kernel instances — including a PWR (authority/concession) and an open-texture
// boundary token ("appropriate", OT-1) — and inserts them into the live
// PostgreSQL database with bitemporal TSTZRANGE bounds of [now, infinity).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/kernel"
)

// Stable instance identities for the ISO 9001 §8.7 benchmark.
const (
	idN2a = "2a000000-0000-4000-8000-000000000002" // NRM identify & control
	idC2  = "c2000000-0000-4000-8000-000000000002" // CLS output -> nonconforming
	idN2b = "2b000000-0000-4000-8000-000000000002" // NRM take appropriate action (OT-1)
	idN2c = "2c000000-0000-4000-8000-000000000002" // NRM re-verify
	idN2d = "2d000000-0000-4000-8000-000000000002" // NRM retain documented information
	idC3  = "c3000000-0000-4000-8000-000000000002" // CLS person -> concession-authority
	idP1  = "b1000000-0000-4000-8000-000000000002" // PWR concession power
	idG3  = "d3000000-0000-4000-8000-000000000002" // GRD corrected -> re-verify
)

const qms = "quality_management_system"

func buildInstances() []struct {
	id string
	c  kernel.Constructor
	p  any
} {
	// n2a: identify and control nonconforming outputs (§8.7.1).
	n2a := kernel.NRMPayload{
		Bearer: "org", Counterparty: qms,
		Act: "identify_and_control", Sign: "+", Force: "O",
		Target: "output:c2",
	}

	// c2: classify an output as nonconforming (fails to conform to requirements).
	c2 := kernel.CLSPayload{
		Entity: "output", ClassToken: "nonconforming",
		Condition: kernel.Not(kernel.Pred("conforms_to_requirements", kernel.Var("output"))),
	}

	// n2b: take APPROPRIATE action — "appropriate" is open texture, modeled as a
	// boundary token OT-1 that the evaluator must treat as unresolved (D0 §7).
	n2b := kernel.NRMPayload{
		Bearer: "org", Counterparty: qms,
		Act: "take_action", Sign: "+", Force: "O",
		Target:    "nonconformity",
		Qualifier: kernel.Boundary("OT-1", "appropriate"),
	}

	// n2c: re-verify conformity after correction (§8.7.1).
	n2c := kernel.NRMPayload{
		Bearer: "org", Counterparty: qms,
		Act: "re_verify", Sign: "+", Force: "O",
		Target: "corrected_output",
	}

	// n2d: retain documented information (§8.7.2).
	n2d := kernel.NRMPayload{
		Bearer: "org", Counterparty: qms,
		Act: "retain_documented_information", Sign: "+", Force: "O",
		Target: "nonconformity_record",
	}

	// c3: classify a person as the concession authority (§8.7.2).
	c3 := kernel.CLSPayload{
		Entity: "person", ClassToken: "concession_authority",
		Condition: kernel.Pred("designated_concession_authority", kernel.Var("person")),
	}

	// p1: the concession power, held by the concession-authority class (c3).
	// Exercising it (event "concession-record") CREATES a GRD that suspends the
	// identify-and-control obligation n2a for the accepted-under-concession output.
	p1 := kernel.PWRPayload{
		Holder:        idC3,
		Effect:        "create",
		Event:         "concession-record",
		OperandSchema: kernel.Pred("nonconforming_output", kernel.Var("output")),
		Operand: &kernel.GRDPayload{
			Condition: kernel.Pred("concession_granted", kernel.Var("output")),
			Defeats:   []string{idN2a},
			Priority:  150,
		},
	}

	// g3: when the output is corrected, activate the re-verify obligation n2c.
	g3 := kernel.GRDPayload{
		Condition: kernel.Pred("corrected", kernel.Var("output")),
		Body:      []string{idN2c},
		Priority:  100,
	}

	return []struct {
		id string
		c  kernel.Constructor
		p  any
	}{
		{idN2a, kernel.NRM, n2a},
		{idC2, kernel.CLS, c2},
		{idN2b, kernel.NRM, n2b},
		{idN2c, kernel.NRM, n2c},
		{idN2d, kernel.NRM, n2d},
		{idC3, kernel.CLS, c3},
		{idP1, kernel.PWR, p1},
		{idG3, kernel.GRD, g3},
	}
}

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "e_writer"),
		getenv("PGPASSWORD", "e_writer_dev"),
		getenv("PGHOST", "localhost"),
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

// locusFor returns the D8 Run-2 §6 source-map coordinates. D8 lists n2a…n2d
// (8.7.1/8.7.2 clauses) and p1 (8.7.1(c) concession, inline) explicitly; c2,
// c3, g3 anchor to their host clauses.
func locusFor(id string) (locus, kind string) {
	switch id {
	case idN2a, idN2b, idN2c:
		return "iso9001 : 8.7.1", "clause"
	case idN2d, idC3:
		return "iso9001 : 8.7.2", "clause"
	case idP1:
		return "iso9001 : 8.7.1(c)-concession", "inline"
	case idC2:
		return "iso9001 : 8.7.1", "classification"
	case idG3:
		return "iso9001 : 8.7.1", "condition-span"
	}
	return "iso9001 : 8.7", "clause"
}

const insertSQL = `
INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)
RETURNING pk`

// I9: every kernel row gets exactly one source-map row, in the same tx.
const sourceMapSQL = `
INSERT INTO source_map (instance_pk, locus, kind, span)
VALUES ($1, $2, $3, int4range($4, $5))`

// backfillSQL inserts the source-map row for an already-ingested instance,
// only where none exists (I9 backfill; span = stored payload length).
const backfillSQL = `
INSERT INTO source_map (instance_pk, locus, kind, span)
SELECT k.pk, $2, $3, int4range(0, length(k.payload::text))
FROM kernel_instance k
LEFT JOIN source_map s ON s.instance_pk = k.pk
WHERE k.instance_id = $1::uuid AND s.id IS NULL`

func main() {
	backfill := flag.Bool("backfill-sourcemap", false,
		"do not ingest; only insert missing source_map rows for the Run-2 instances")
	flag.Parse()

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	if *backfill {
		n := 0
		for _, id := range []string{idN2a, idC2, idN2b, idN2c, idN2d, idC3, idP1, idG3} {
			locus, kind := locusFor(id)
			tag, err := conn.Exec(ctx, backfillSQL, id, locus, kind)
			if err != nil {
				log.Fatalf("backfill %s: %v", id, err)
			}
			n += int(tag.RowsAffected())
		}
		log.Printf("backfilled %d source_map rows (D8 Run 2, I9)", n)
		return
	}

	now := time.Now().UTC()
	validity, err := kernel.Since(now).Value()
	if err != nil {
		log.Fatalf("range literal: %v", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, s := range buildInstances() {
		raw, err := json.Marshal(s.p)
		if err != nil {
			log.Fatalf("marshal %s: %v", s.c, err)
		}
		var pk int64
		if err := tx.QueryRow(ctx, insertSQL, s.id, string(s.c), string(raw), validity, validity).Scan(&pk); err != nil {
			log.Fatalf("insert %s %s: %v", s.c, s.id, err)
		}
		locus, kind := locusFor(s.id)
		if _, err := tx.Exec(ctx, sourceMapSQL, pk, locus, kind, 0, len(raw)); err != nil {
			log.Fatalf("source_map %s: %v", s.id, err)
		}
		log.Printf("ingested %-3s %s  ⟦%s : %s⟧", s.c, s.id, locus, kind)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	log.Printf("D8 Run 2 (ISO 9001:2015 §8.7) ingested: 8 instances")

	// Read p1 back and print a JSONB excerpt to verify PWR power-schema
	// serialization (Holder, Effect, Event, and the GRD Operand).
	var payload []byte
	if err := conn.QueryRow(ctx,
		`SELECT payload FROM kernel_instance_at(now(), now()) WHERE instance_id = $1::uuid`, idP1,
	).Scan(&payload); err != nil {
		log.Fatalf("fetch p1: %v", err)
	}
	var pretty map[string]any
	if err := json.Unmarshal(payload, &pretty); err != nil {
		log.Fatalf("decode p1: %v", err)
	}
	out, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Printf("\n── p1 (PWR) JSONB excerpt [%s] ─────────────────────────────\n%s\n", idP1, out)
}
