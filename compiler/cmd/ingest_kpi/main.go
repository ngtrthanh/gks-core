// Command ingest_kpi persists D8 Run 6 (KPI-SEC-03, on-time incident logging
// rate) into the store and evaluates it with EXACT rational arithmetic (WP-7).
//
// It writes:
//   - registry token policy-p11-§4.threshold = 1 (P-11 §4 sets 100%);
//   - v4 (VAL): measure = ratio(on-time logs, total incidents),
//     comparator ≥, target = 0.95 × reg(policy-p11-§4.threshold);
//   - n7 (NRM): the Security Manager's obligation to achieve v4.
//
// Then it reads the registry back, builds the evaluation environment, and
// prints the verdict for a sample quarter — demonstrating that the
// normative-reference-in-a-formula target compiles and evaluates with no
// float64 anywhere (I8).
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

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
	"computable-governance/compiler/internal/registry"
)

const (
	idV4 = "44000000-0000-4000-8000-000000000006" // VAL KPI-SEC-03
	idN7 = "77000000-0000-4000-8000-000000000006" // NRM achieve(v4)
	idR6 = "44000000-0000-4000-8000-0000000000f6" // REF v4 -> P-11 §4

	thresholdToken = "policy-p11-§4.threshold"
)

// ratJSON is the registry storage form of an exact rational: {"rat":"..."}.
type ratJSON struct {
	Rat string `json:"rat"`
}

func kpiVAL() kernel.VALPayload {
	return kernel.VALPayload{
		Function:   "on_time_logging_rate",
		Unit:       "ratio",
		Comparator: kernel.CmpGE,
		Measure: kernel.Ratio(
			kernel.Var("on_time_logs"),
			kernel.Var("total_incidents"),
		),
		Target: kernel.Arith(kernel.ArithMul,
			kernel.LitRat("0.95"),
			kernel.Lookup(thresholdToken)),
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

const insertKernelSQL = `
INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)
ON CONFLICT DO NOTHING
RETURNING pk`

const sourceMapSQL = `
INSERT INTO source_map (instance_pk, locus, kind, span)
VALUES ($1, $2, $3, int4range(0, $4))`

// registrySQL appends a NEW version of a token only when its value differs
// from the current latest — registry R is versioned and INSERT-only (I4), so
// values are never mutated in place. A no-op returns no rows.
const registrySQL = `
INSERT INTO registry (token, version, value)
SELECT $1, COALESCE(MAX(version), 0) + 1, $2::jsonb
FROM registry WHERE token = $1
HAVING (
  SELECT value FROM registry r WHERE r.token = $1 ORDER BY version DESC LIMIT 1
) IS DISTINCT FROM $2::jsonb
RETURNING version`

func main() {
	onTime := flag.Int64("on-time", 96, "on-time incident logs in the quarter")
	total := flag.Int64("total", 100, "total incidents in the quarter")
	flag.Parse()

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	// Corpus-derived coordinate (WP-8, I8): P-11 policy effective epoch
	// (declared, fixed) → reproducible CNF across compilers.
	validity, err := kernel.Since(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)).Value()
	if err != nil {
		log.Fatalf("range: %v", err)
	}

	// --- registry: P-11 §4 threshold = 1 (100%), stored as an exact rational.
	thr, _ := json.Marshal(ratJSON{Rat: "1"})
	var ver int
	switch err := conn.QueryRow(ctx, registrySQL, thresholdToken, string(thr)).Scan(&ver); err {
	case nil:
		fmt.Printf("  registry %s = 1  (version %d)\n", thresholdToken, ver)
	case pgx.ErrNoRows:
		fmt.Printf("  registry %s unchanged\n", thresholdToken)
	default:
		log.Fatalf("registry: %v", err)
	}

	// --- kernel instances: v4 (VAL) and n7 (NRM achieve v4).
	n7 := kernel.NRMPayload{
		Bearer: "security_manager", Counterparty: "org",
		Act: "achieve_kpi_sec_03", Sign: "+", Force: "O", Target: idV4,
	}
	insert := func(id string, c kernel.Constructor, payload any, locus, kind string) {
		raw, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("marshal %s: %v", id, err)
		}
		var pk int64
		err = conn.QueryRow(ctx, insertKernelSQL, id, string(c), string(raw), validity, validity).Scan(&pk)
		if err == pgx.ErrNoRows {
			fmt.Printf("  %s already present; skipped\n", id)
			return
		}
		if err != nil {
			log.Fatalf("insert %s: %v", id, err)
		}
		if _, err := conn.Exec(ctx, sourceMapSQL, pk, locus, kind, len(raw)); err != nil {
			log.Fatalf("source_map %s: %v", id, err)
		}
		fmt.Printf("  ingested %-3s %s  ⟦%s⟧\n", c, id, locus)
	}
	insert(idV4, kernel.VAL, kpiVAL(), "kpi-cat : SEC-03 : row", "row")
	insert(idN7, kernel.NRM, n7, "kpi-cat : SEC-03-owner : row", "row")
	// REF: the KPI's target embeds a normative reference to P-11 §4 (the
	// threshold source) — a cross-corpus designation traversable by `impact`.
	insert(idR6, kernel.REF,
		kernel.REFPayload{Source: idV4, TargetIRI: "policy-p11:§4", Mode: "cite"},
		"kpi-cat : SEC-03-ref : row", "row")

	// --- read the registry back and evaluate (exact rational, no float64).
	reg, err := registry.Snapshot(ctx, conn)
	if err != nil {
		log.Fatalf("load registry: %v", err)
	}
	val := kpiVAL()
	env := evaluator.Environment{
		Registry: reg,
		Vars: map[string]evaluator.Value{
			"on_time_logs":    evaluator.VInt(*onTime),
			"total_incidents": evaluator.VInt(*total),
		},
	}
	measure, err := evaluator.Eval(val.Measure, env)
	if err != nil {
		log.Fatalf("eval measure: %v", err)
	}
	target, err := evaluator.Eval(val.Target, env)
	if err != nil {
		log.Fatalf("eval target: %v", err)
	}
	verdict, err := evaluator.Eval(val.AsExpr(), env)
	if err != nil {
		log.Fatalf("eval VAL: %v", err)
	}

	bar := "──────────────────────────────────────────────────────────────────────────"
	fmt.Println(bar)
	fmt.Printf(" KPI-SEC-03  measure = %d/%d = %s\n", *onTime, *total, measure.String())
	fmt.Printf(" target      0.95 × reg(%s) = %s\n", thresholdToken, target.String())
	fmt.Printf(" comparator  %s\n", val.Comparator)
	fmt.Printf(" VERDICT     %s  (%s)\n", verdictLabel(verdict.Bool()), boolStr(verdict.Bool()))
	fmt.Println(bar)
}

// loadRegistry reads the latest version of every registry token into exact
// evaluator Values. A {"rat":"..."} value becomes a KRat; other JSON scalars
// map to their kinds. (Now provided by internal/registry — WP-6.)
func verdictLabel(ok bool) string {
	if ok {
		return "COMPLIANT"
	}
	return "VIOLATED"
}

func boolStr(ok bool) string {
	if ok {
		return "measure meets target"
	}
	return "measure below target"
}
