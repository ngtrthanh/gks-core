// Command verify_db is a READ-ONLY verification reporter for the Knowledge-Theory
// (K-hat) IR store. It connects to the live PostgreSQL database, retrieves the
// D8 Run 1 (26 U.S.C. §121) benchmark instances, and prints a structured console
// report showing each instance's identity, TIX bitemporal boundaries, and the
// pretty-printed JSONB AST payload.
//
// It issues only SELECTs and never mutates the database.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

type record struct {
	ID          string
	Constructor string
	TText       string
	TFact       string
	Payload     []byte
}

// connString honors DATABASE_URL / PG* env vars, defaulting to the compose
// `db` service on localhost:5435.
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

// prettyJSON indents raw JSONB bytes WITHOUT HTML-escaping, so operators like
// `>=` and `<=` render literally (json.Indent preserves the source bytes).
func prettyJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "  ", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

const query = `
SELECT instance_id::text, constructor::text, t_text::text, t_fact::text, payload
FROM kernel_instance_at(now(), now())
ORDER BY constructor, instance_id`

func main() {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, query)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var recs []record
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.ID, &r.Constructor, &r.TText, &r.TFact, &r.Payload); err != nil {
			log.Fatalf("scan: %v", err)
		}
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	bar := strings.Repeat("=", 78)
	fmt.Println(bar)
	fmt.Println("  GOVERNANCE COMPUTING — READ-ONLY VERIFICATION REPORT")
	fmt.Println("  Benchmark: D8 Run 1 — 26 U.S.C. §121 (principal-residence gain exclusion)")
	fmt.Printf("  kernel_instance rows retrieved: %d\n", len(recs))
	fmt.Println(bar)

	for i, r := range recs {
		var meta struct {
			Priority *int `json:"priority"`
		}
		_ = json.Unmarshal(r.Payload, &meta)

		fmt.Printf("\n[%d] %s  %s\n", i+1, r.Constructor, r.ID)
		fmt.Printf("    t_text : %s\n", r.TText)
		fmt.Printf("    t_fact : %s\n", r.TFact)

		if r.Constructor == "GRD" && meta.Priority != nil && *meta.Priority == 200 {
			fmt.Println(strings.Repeat("-", 78))
			fmt.Println("    >>> HIGHLIGHT: anti-stacking exception g2 — §121(b)(3), priority 200")
			fmt.Println("    >>> Verify below: bounded quantifier (count >= 1) over a temporal")
			fmt.Println("    >>>               window (within P2Y), defeating the NRM exclusion.")
			fmt.Println(strings.Repeat("-", 78))
		}

		fmt.Printf("    payload (AST):\n  %s\n", prettyJSON(r.Payload))
	}

	fmt.Println("\n" + bar)
	fmt.Println("  END OF REPORT (no rows mutated)")
	fmt.Println(bar)
}
