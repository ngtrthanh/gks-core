// Command impact runs the REF-traversal "Monday-Morning" query (WP-4): every
// instance transitively referencing a target IRI, valid at the read
// coordinates. READ-ONLY.
//
//	impact <target_iri> [--at-text RFC3339] [--at-fact RFC3339]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/coord"
	"computable-governance/compiler/internal/refgraph"
)

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "e_writer"), getenv("PGPASSWORD", "e_writer_dev"),
		getenv("PGHOST", "localhost"), getenv("PGPORT", "5435"),
		getenv("PGDATABASE", "governance"))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	atText := flag.String("at-text", "", "read coordinate t_text (RFC3339; default now)")
	atFact := flag.String("at-fact", "", "read coordinate t_fact (RFC3339; default now)")
	flag.Parse()

	target := flag.Arg(0)
	if target == "" {
		log.Fatal("usage: impact <target_iri> [--at-text RFC3339] [--at-fact RFC3339]")
	}
	tText, tFact, err := coord.Parse(*atText, *atFact)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	nodes, err := refgraph.Impact(ctx, conn, target, tText, tFact)
	if err != nil {
		log.Fatalf("impact: %v", err)
	}

	fmt.Printf("IMPACT of %q  @ t_text=%s t_fact=%s\n", target,
		tText.Format("2006-01-02"), tFact.Format("2006-01-02"))
	if len(nodes) == 0 {
		fmt.Println("  (no instances reference this target at these coordinates)")
		return
	}
	for _, n := range nodes {
		fmt.Printf("  depth %d  %s\n", n.Depth, n.IRI)
	}
	fmt.Printf("total impacted: %d\n", len(nodes))
}
