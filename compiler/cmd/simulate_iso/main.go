// Command simulate_iso simulates the ISO 9001:2015 §8.7 concession workflow
// against the live database (READ-ONLY). It:
//
//  1. detects a nonconforming output (c2),
//  2. hits the open-texture boundary token OT-1 in n2b and emits a CONDITIONAL
//     verdict rather than a definite one,
//  3. simulates the concession authority (c3) exercising the power p1, which
//     dynamically introduces the operand GRD (defeats = [n2a]),
//  4. resolves n2a and shows it DEFEATED (nonconformity control suspended).
//
// The PWR exercise is an in-memory simulated transition; nothing is written to
// the database.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
)

const (
	idN2a = "2a000000-0000-4000-8000-000000000002"
	idC2  = "c2000000-0000-4000-8000-000000000002"
	idN2b = "2b000000-0000-4000-8000-000000000002"
	idC3  = "c3000000-0000-4000-8000-000000000002"
	idP1  = "b1000000-0000-4000-8000-000000000002"
)

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "e_writer"), getenv("PGPASSWORD", "e_writer_dev"),
		getenv("PGHOST", "localhost"), getenv("PGPORT", "5435"),
		getenv("PGDATABASE", "governance"),
	)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func short(id string) string {
	if len(id) >= 8 {
		return id[:8] + "…"
	}
	return id
}

const bar = "──────────────────────────────────────────────────────────────────────────"

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	// READ-ONLY load of the four instances we need.
	payloads := map[string][]byte{}
	rows, err := conn.Query(ctx,
		`SELECT instance_id::text, payload FROM kernel_instance_at(now(), now()) WHERE instance_id::text = ANY($1)`,
		[]string{idN2a, idC2, idN2b, idP1, idC3})
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	for rows.Next() {
		var id string
		var p []byte
		if err := rows.Scan(&id, &p); err != nil {
			log.Fatalf("scan: %v", err)
		}
		payloads[id] = p
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}
	if len(payloads) < 5 {
		log.Fatalf("expected 5 ISO instances; got %d (is Run 2 ingested?)", len(payloads))
	}

	var c2 kernel.CLSPayload
	var n2b kernel.NRMPayload
	var p1 kernel.PWRPayload
	must(json.Unmarshal(payloads[idC2], &c2), "decode c2")
	must(json.Unmarshal(payloads[idN2b], &n2b), "decode n2b")
	must(json.Unmarshal(payloads[idP1], &p1), "decode p1")

	// Base environment: the output fails to conform (=> nonconforming); it has
	// not yet been corrected; the actor is a designated concession authority; no
	// concession has been granted yet.
	env := evaluator.Environment{
		Now: time.Now().UTC(),
		Predicates: map[string]bool{
			"conforms_to_requirements":        false,
			"corrected":                       false,
			"designated_concession_authority": true,
			"concession_granted":              false,
		},
		Vars:    map[string]evaluator.Value{},
		Domains: map[string][]evaluator.Fact{},
	}

	fmt.Println(bar)
	fmt.Println(" LAYER E — ISO 9001:2015 §8.7 CONCESSION SIMULATION (D8 Run 2)")
	fmt.Println(bar)

	// --- STEP 1: nonconformity detected ------------------------------------
	nc, err := evaluator.Eval(c2.Condition, env)
	if err != nil {
		log.Fatalf("eval c2: %v", err)
	}
	fmt.Println(" [1] INITIAL STATE")
	fmt.Printf("     c2  output classified nonconforming ......... %v\n", nc.Bool())
	fmt.Printf("     n2a control obligation (%s) ............ IN_FORCE (default)\n", short(idN2a))
	fmt.Println("         guards currently defeating n2a: none")

	// --- STEP 2: open-texture boundary → conditional verdict ---------------
	fmt.Println("\n [2] EVALUATE n2b \"take appropriate action\"")
	_, otErr := evaluator.Eval(n2b.Qualifier, env)
	if !evaluator.IsBoundary(otErr) {
		log.Fatalf("expected an open-texture boundary signal at OT-1, got: %v", otErr)
	}
	fmt.Printf("     hit boundary token: %s (%q)\n", n2b.Qualifier.Name, n2b.Qualifier.Label)
	fmt.Println("     VERDICT: CONDITIONAL — open texture unresolved; deferred to adjudication")
	fmt.Printf("     (evaluator signal: %v)\n", otErr)

	// --- STEP 3: concession authority exercises PWR p1 ---------------------
	fmt.Println("\n [3] PWR EXERCISE — concession authority (c3) exercises p1")
	authorized := env.Predicates["designated_concession_authority"]
	fmt.Printf("     holder authorized (c3) .................. %v\n", authorized)
	fmt.Printf("     effect=%q  event=%q\n", p1.Effect, p1.Event)

	fmt.Println("     defeats list BEFORE exercise: []  (n2a has no active defeater)")

	if p1.Operand == nil {
		log.Fatalf("p1 has no operand GRD")
	}
	// Simulated transition: the exercise (a) grants the concession and
	// (b) instantiates the operand GRD into the active guard set.
	env.Predicates["concession_granted"] = true
	concession := evaluator.Guard{
		ID:        "g:concession(p1-exercise)",
		Priority:  p1.Operand.Priority,
		Condition: p1.Operand.Condition,
		Body:      p1.Operand.Body,
		Defeats:   p1.Operand.Defeats,
	}
	fmt.Printf("     >>> concession GRD instantiated: id=%s priority=%d\n", concession.ID, concession.Priority)
	fmt.Printf("     defeats list AFTER exercise:  %v   <-- dynamically added\n", concession.Defeats)
	fmt.Println("     env fact set: concession_granted = true")

	// --- STEP 4: resolve n2a with the new guard ----------------------------
	fmt.Println("\n [4] RESOLUTION — n2a against the post-exercise guard set")
	res, err := evaluator.Resolve(idN2a, []evaluator.Guard{concession}, env)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}
	for _, s := range res.Steps {
		marker := "  "
		if s.Decisive {
			marker = "▶ "
		}
		note := ""
		if s.Decisive {
			note = "  <-- DECISIVE"
		}
		fmt.Printf("   %s[prio %3d] %s  cond=%-5v relation=%-10s%s\n",
			marker, s.Priority, s.GuardID, s.CondMet, s.Relation, note)
	}

	fmt.Println(bar)
	fmt.Printf(" NORM n2a = %s\n", short(idN2a))
	fmt.Printf(" VERDICT  : %s\n", res.Verdict)
	if res.Verdict == "DEFEATED" {
		fmt.Println(" RESULT   : nonconformity-control obligation SUSPENDED by concession")
		fmt.Println("            (§8.7.1(d) acceptance under concession by authorized person).")
		fmt.Println(" NOTE     : n2b remains a CONDITIONAL verdict (OT-1 \"appropriate\" unresolved).")
	}
	fmt.Println(bar)
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("%s: %v", ctx, err)
	}
}
