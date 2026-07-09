// Command simulate_case runs the D8 Run 1 execution-layer simulation for
// 26 U.S.C. §121. It pulls the four Layer-K instances from the live database,
// feeds a mock "Monday morning" fact scenario into the evaluator, resolves the
// NRM against its guards by defeasible priority, and prints the verdict trace.
//
// This program only SELECTs from the database; it does not mutate Layer K.
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

	"computable-governance/compiler/internal/coord"
	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
	"computable-governance/compiler/internal/registry"
)

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

type instance struct {
	ID          string
	Constructor string
	Payload     []byte
}

// §121 (D8 Run 1) instance identities — scope the simulation to this benchmark
// so it is deterministic regardless of other ingested corpora in the store.
const (
	id121N1 = "a1000000-0000-4000-8000-000000000001"
	id121C1 = "c1000000-0000-4000-8000-000000000001"
	id121G1 = "d1000000-0000-4000-8000-000000000001"
	id121G2 = "d2000000-0000-4000-8000-000000000001"
)

func main() {
	atText := flag.String("at-text", "", "read coordinate t_text (RFC3339; default now)")
	atFact := flag.String("at-fact", "", "read coordinate t_fact (RFC3339; default now)")
	flag.Parse()
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

	rows, err := conn.Query(ctx,
		`SELECT instance_id::text, constructor::text, payload FROM kernel_instance_at($1, $2) WHERE instance_id::text = ANY($3)`,
		tText, tFact, []string{id121N1, id121C1, id121G1, id121G2})
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	var (
		normID  string
		clsCond *kernel.Expr
		guards  []evaluator.Guard
	)
	for rows.Next() {
		var in instance
		if err := rows.Scan(&in.ID, &in.Constructor, &in.Payload); err != nil {
			log.Fatalf("scan: %v", err)
		}
		switch in.Constructor {
		case "NRM":
			normID = in.ID
		case "CLS":
			var p kernel.CLSPayload
			if err := json.Unmarshal(in.Payload, &p); err != nil {
				log.Fatalf("unmarshal CLS: %v", err)
			}
			clsCond = p.Condition
		case "GRD":
			var p kernel.GRDPayload
			if err := json.Unmarshal(in.Payload, &p); err != nil {
				log.Fatalf("unmarshal GRD: %v", err)
			}
			guards = append(guards, evaluator.Guard{
				ID:        in.ID,
				Priority:  p.Priority,
				Condition: p.Condition,
				Body:      p.Body,
				Defeats:   p.Defeats,
			})
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}
	if normID == "" || len(guards) < 2 {
		log.Fatalf("expected 1 NRM and >=2 GRDs; got norm=%q guards=%d (is the benchmark ingested?)", normID, len(guards))
	}

	// --- MOCK SCENARIO: "Monday morning" -----------------------------------
	// * owned/used as principal residence for 3 years  -> passes c1
	// * realized gain $150,000                          -> passes g1 (<= $250k)
	// * a prior §121 sale 18 months ago                 -> triggers g2 (within 2y)
	now := tFact
	reg, err := registry.Snapshot(ctx, conn)
	if err != nil {
		log.Fatalf("load registry: %v", err)
	}
	env := evaluator.Environment{
		Now:      now,
		Registry: reg,
		Vars: map[string]evaluator.Value{
			"aggregate_use_years": evaluator.VInt(3),
			"realized_gain":       evaluator.VInt(150_000),
		},
		Predicates: map[string]bool{
			"classified":                  true,
			"owned_and_used_as_residence": true,
		},
		Domains: map[string][]evaluator.Fact{
			"taxpayer_prior_sales": {
				{Name: "section121_exclusion_applied", Time: now.AddDate(0, -18, 0)},
			},
		},
	}

	bar := "──────────────────────────────────────────────────────────────────────────"
	fmt.Println(bar)
	fmt.Println(" LAYER E — EXECUTION SIMULATION :: 26 U.S.C. §121 (D8 Run 1)")
	fmt.Println(bar)
	fmt.Println(" Scenario (Monday morning):")
	fmt.Println("   • principal-residence use : 3 years")
	fmt.Println("   • realized gain           : $150,000")
	fmt.Println("   • prior §121 sale         : 18 months ago (inside the 2-year window)")
	fmt.Printf("   • evaluation time (now)   : %s\n", now.Format(time.RFC3339))
	fmt.Println(bar)

	// Classification (c1) feeds the "classified" predicate consumed by g1.
	if clsCond != nil {
		v, err := evaluator.Eval(clsCond, env)
		if err != nil {
			log.Fatalf("eval CLS c1: %v", err)
		}
		fmt.Printf(" c1  CLS principal_residence test .......... %s\n", verdictBool(v.Bool()))
	}

	fmt.Println("\n Guard evaluation (independent):")
	for _, g := range guards {
		v, err := evaluator.Eval(g.Condition, env)
		if err != nil {
			log.Fatalf("eval guard %s: %v", g.ID, err)
		}
		rel := "—"
		switch {
		case contains(g.Defeats, normID):
			rel = "defeats n1"
		case contains(g.Body, normID):
			rel = "activates n1"
		}
		fmt.Printf("   [prio %3d] %s  cond=%-5v  (%s)\n", g.Priority, short(g.ID), v.Bool(), rel)
	}

	// --- Defeasible priority resolution ------------------------------------
	res, err := evaluator.Resolve(normID, guards, env)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}

	fmt.Println("\n Resolution trace (highest priority first):")
	for _, s := range res.Steps {
		marker := "  "
		if s.Decisive {
			marker = "▶ "
		}
		fmt.Printf(" %s[prio %3d] %s  cond=%-5v  relation=%-10s%s\n",
			marker, s.Priority, short(s.GuardID), s.CondMet, s.Relation,
			decisiveNote(s.Decisive))
	}

	fmt.Println(bar)
	fmt.Printf(" NORM n1 = %s\n", short(normID))
	fmt.Printf(" VERDICT : %s\n", res.Verdict)
	if res.Verdict == "DEFEATED" {
		fmt.Println(" REASON  : anti-stacking exception g2 (§121(b)(3)) fired at priority 200,")
		fmt.Println("           defeating the gain-exclusion norm despite g1 activating it.")
	}
	fmt.Println(bar)
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func short(id string) string {
	if len(id) >= 8 {
		return id[:8] + "…"
	}
	return id
}

func verdictBool(b bool) string {
	if b {
		return "PASS"
	}
	return "FAIL"
}

func decisiveNote(d bool) string {
	if d {
		return "  <-- DECISIVE"
	}
	return ""
}
