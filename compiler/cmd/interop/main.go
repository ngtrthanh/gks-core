// Command interop computes REAL inter-compiler agreement (Track B) over the
// live labour-code corpus, rather than the synthetic fixtures used by
// `make validate`.
//
// Compiler 1 = the assignments already in the store (the ingest_docx heuristic).
// Compiler 2 = an INDEPENDENT re-classification of the same source text with a
// deliberately different cue set and precedence (implemented here). Fleiss' κ
// is computed over the two assignments per source_map locus.
//
// CAVEAT (honest): a single team maintains both classifiers, so this measures
// robustness-to-rule-choice, not true organizational independence. A genuinely
// separate compiler is required for a constitutional κ claim; this makes the
// harness operate on real corpus output and yields a real number.
//
//	interop [--at-text RFC3339] [--at-fact RFC3339] [outdir]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/coord"
	"computable-governance/compiler/internal/validation"
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

var reDef2 = regexp.MustCompile(`^(?:\d+\.\s*)?(.{2,60}?)\s+là\s+\p{L}`)

func anyOf(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

// classify2 is an INDEPENDENT constructor classifier: precedence
// prohibition → right (any "quyền") → definition → obligation, which differs
// from the ingester's definition-first, "có quyền"-only rules — so the two
// disagree on genuinely ambiguous paragraphs.
func classify2(text string) string {
	low := strings.ToLower(text)
	switch {
	case anyOf(low, "nghiêm cấm", "không được", "bị cấm", "cấm "):
		return "GRD"
	case strings.Contains(low, "quyền"):
		return "PWR"
	case reDef2.MatchString(text):
		return "CLS"
	case anyOf(low, "phải ", "nghĩa vụ", "trách nhiệm"):
		return "NRM"
	default:
		return "NRM"
	}
}

func main() {
	atText := flag.String("at-text", "", "read coordinate t_text (RFC3339; default now)")
	atFact := flag.String("at-fact", "", "read coordinate t_fact (RFC3339; default now)")
	flag.Parse()
	tt, tf, err := coord.Parse(*atText, *atFact)
	if err != nil {
		log.Fatal(err)
	}
	outdir := "../validation/interop"
	if a := flag.Arg(0); a != "" {
		outdir = a
	}
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	// Docx-sourced rows carry the extraction envelope (payload ? 'kind').
	rows, err := conn.Query(ctx, `
		SELECT s.locus, k.constructor::text, k.payload->>'text'
		FROM kernel_instance_at($1,$2) k
		JOIN source_map s ON s.instance_pk = k.pk
		WHERE k.payload ? 'kind' AND k.payload ? 'text'`, tt, tf)
	if err != nil {
		log.Fatalf("query: %v", err)
	}

	c1 := map[string]string{}
	c2 := map[string]string{}
	type dis struct{ locus, a, b, text string }
	var disagreements []dis
	f1, _ := os.Create(filepath.Join(outdir, "compiler1.cnf"))
	f2, _ := os.Create(filepath.Join(outdir, "compiler2.cnf"))
	defer f1.Close()
	defer f2.Close()

	for rows.Next() {
		var locus, ctor, text string
		if err := rows.Scan(&locus, &ctor, &text); err != nil {
			log.Fatalf("scan: %v", err)
		}
		alt := classify2(text)
		c1[locus] = ctor
		c2[locus] = alt
		writeLine(f1, locus, ctor)
		writeLine(f2, locus, alt)
		if ctor != alt {
			disagreements = append(disagreements, dis{locus, ctor, alt, text})
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	ratings := validation.Align([]map[string]string{c1, c2})
	kappa, n := validation.FleissKappa(ratings)

	// Report.
	var b strings.Builder
	fmt.Fprintf(&b, "# Inter-Compiler Agreement — REAL corpus (Track B)\n\n")
	fmt.Fprintf(&b, "Corpus: live labour-code store (docx-sourced instances).\n\n")
	fmt.Fprintf(&b, "- shared loci compared: **%d**\n", n)
	fmt.Fprintf(&b, "- disagreements: **%d**\n", len(disagreements))
	fmt.Fprintf(&b, "- Fleiss' κ (constructor assignment): **%.4f**  (floor 0.70: %s)\n\n",
		kappa, meets(kappa, 0.70))
	fmt.Fprintf(&b, "Compiler 1 = stored ingest_docx assignment; Compiler 2 = independent\n")
	fmt.Fprintf(&b, "re-classifier (prohibition→right→definition→obligation precedence).\n\n")
	fmt.Fprintf(&b, "**Caveat:** one team maintains both classifiers — this measures\n")
	fmt.Fprintf(&b, "robustness to rule choice, not organizational independence. A genuinely\n")
	fmt.Fprintf(&b, "separate compiler is still required for a constitutional κ claim.\n\n")
	sort.Slice(disagreements, func(i, j int) bool { return disagreements[i].locus < disagreements[j].locus })
	fmt.Fprintf(&b, "## Sample disagreements (first 10)\n\n")
	for i, d := range disagreements {
		if i >= 10 {
			break
		}
		t := d.text
		if len([]rune(t)) > 90 {
			t = string([]rune(t)[:90]) + "…"
		}
		fmt.Fprintf(&b, "- `%s`  c1=%s c2=%s — %s\n", d.locus, d.a, d.b, t)
	}
	_ = os.WriteFile(filepath.Join(outdir, "REPORT.md"), []byte(b.String()), 0o644)

	fmt.Printf("interop: %d shared loci, %d disagreements, Fleiss' κ = %.4f (floor 0.70: %s)\n",
		n, len(disagreements), kappa, meets(kappa, 0.70))
	fmt.Printf("wrote %s/{compiler1.cnf,compiler2.cnf,REPORT.md}\n", outdir)
}

func writeLine(f *os.File, locus, ctor string) {
	line, _ := json.Marshal(map[string]any{"locus": locus, "constructor": ctor, "payload": map[string]any{}})
	f.Write(append(line, '\n'))
}

func meets(v, floor float64) string {
	if v >= floor {
		return "MET"
	}
	return "BELOW"
}
