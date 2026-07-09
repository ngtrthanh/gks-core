// Command replay_d8 executes the D8 benchmark runs end-to-end through the
// persisted execution layer Ê (WP-3): fixture event traces are recorded as
// world_event rows (idempotently, deterministic event ids), then replayed
// from the database through the D1.4 transition rules to e_machine states,
// transition_log entries and persisted verdict rows.
//
//	replay_d8 [-run run1|run2|all]
//
// Run 1 (26 U.S.C. §121): a qualifying sale event; g1 activates the exclusion
// permission n1 -> verdict 'compliant'.
// Run 2 (ISO 9001 §8.7): nonconformity + concession-record PWR exercise; the
// operand GRD is appended to K̂ (S-Exercise), n2a is suspended with
// pwr_instance recorded, n2b stays conditional on OT-1.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/coord"
	"computable-governance/compiler/internal/machine"
	"computable-governance/compiler/internal/registry"
)

const (
	idN1  = "a1000000-0000-4000-8000-000000000001" // Run 1 NRM
	idN2a = "2a000000-0000-4000-8000-000000000002" // Run 2 NRM identify & control
	idN2b = "2b000000-0000-4000-8000-000000000002" // Run 2 NRM take_action (OT-1)
	idN2c = "2c000000-0000-4000-8000-000000000002" // Run 2 NRM re-verify
	idN2d = "2d000000-0000-4000-8000-000000000002" // Run 2 NRM retain documented info
	idP1  = "b1000000-0000-4000-8000-000000000002" // Run 2 PWR concession
)

type fixture struct {
	run      string
	subjects []string
	events   []machine.Event
}

// fixtures returns the D8 event traces. Event ids are deterministic so the
// declarations are idempotent (ON CONFLICT DO NOTHING) and replays reproduce
// identical machines and K̂ extensions (I8).
func fixtures() []fixture {
	return []fixture{
		{
			run:      "d8-run1",
			subjects: []string{idN1},
			events: []machine.Event{
				{
					ID: machine.DetUUID("d8-run1|sale"), Type: "fact",
					Agent: "taxpayer:individual",
					At:    time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
					Payload: map[string]any{
						"predicates": map[string]any{"classified": true},
						"vars":       map[string]any{"realized_gain": 200_000, "aggregate_use_years": 3},
					},
				},
			},
		},
		{
			run:      "d8-run2",
			subjects: []string{idN2a, idN2b, idN2c, idN2d},
			events: []machine.Event{
				{
					ID: machine.DetUUID("d8-run2|nonconformity-detected"), Type: "fact",
					Agent: "org:inspection",
					At:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
					Payload: map[string]any{
						"predicates": map[string]any{
							"conforms_to_requirements":                false,
							"designated_concession_authority":         true,
							"performed:identify_and_control":          true,
							"performed:retain_documented_information": true,
						},
					},
				},
				{
					ID: machine.DetUUID("d8-run2|concession-record"), Type: "pwr-exercise",
					Agent: "person:concession-authority",
					At:    time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
					Payload: map[string]any{
						"pwr":        idP1,
						"exercise":   "concession-record",
						"predicates": map[string]any{"concession_granted": true},
					},
				},
				{
					ID: machine.DetUUID("d8-run2|corrected"), Type: "fact",
					Agent: "org:production",
					At:    time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
					Payload: map[string]any{
						"predicates": map[string]any{
							"corrected":           true,
							"performed:re_verify": true,
						},
					},
				},
			},
		},
	}
}

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
	which := flag.String("run", "all", "which D8 run to replay: run1 | run2 | all")
	atText := flag.String("at-text", "", "eval coordinate t_text (RFC3339; default now)")
	atFact := flag.String("at-fact", "", "eval coordinate t_fact (RFC3339; default now)")
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

	store := &machine.PGStore{Conn: conn, Ctx: ctx}

	kernelRows, err := store.LoadKernel(tText, tFact)
	if err != nil {
		log.Fatalf("load kernel: %v", err)
	}
	view := machine.BuildView(kernelRows)
	reg, err := registry.SnapshotAt(ctx, conn, tFact)
	if err != nil {
		log.Fatalf("load registry: %v", err)
	}
	fmt.Printf("K-hat view: %d norms, %d guards, %d powers; registry %d token(s)\n",
		len(view.Norms), len(view.Guards), len(view.Powers), len(reg))

	for _, f := range fixtures() {
		if *which != "all" && "d8-"+*which != f.run {
			continue
		}

		// 1. Declare the trace (idempotent).
		ids := make([]string, 0, len(f.events))
		for _, ev := range f.events {
			if err := store.InsertEvent(ev); err != nil {
				log.Fatalf("%s: %v", f.run, err)
			}
			ids = append(ids, ev.ID.String())
		}

		// 2. Replay FROM the database (world_event is the source of truth).
		events, err := store.LoadEvents(ids)
		if err != nil {
			log.Fatalf("%s: %v", f.run, err)
		}

		eng := &machine.Engine{
			Store: store, Run: f.run,
			TText: tText, TFact: tFact,
			Registry: reg,
			Subjects: f.subjects,
		}
		res, err := eng.Replay(view, events)
		if err != nil {
			log.Fatalf("%s: replay: %v", f.run, err)
		}

		fmt.Printf("\n=== %s ===\n", f.run)
		fmt.Printf(" events replayed   : %d\n", res.Events)
		fmt.Printf(" K-hat extensions  : %d %v\n", len(res.Created), res.Created)
		fmt.Printf(" transitions       : %d\n", res.Transitions)
		for _, v := range res.Verdicts {
			cond := ""
			if v.ConditionalOn != nil {
				cond = "  conditional_on=" + *v.ConditionalOn
			}
			fmt.Printf(" verdict %s : %-12s%s\n", v.Subject, v.Result, cond)
		}
	}
}
