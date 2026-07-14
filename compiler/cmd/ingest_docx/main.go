// Command ingest_docx performs UNSUPERVISED ingestion of an unstructured legal
// .docx into the Knowledge-Theory (K-hat) store. It:
//
//  1. extracts paragraphs (stdlib, internal/docx),
//  2. tracks structure (Chương / Điều),
//  3. maps clauses to kernel constructors by purely structural heuristics:
//     "<subject> là ..."          -> CLS  (definition/classification)
//     "không được" / "cấm"        -> GRD  (prohibition/condition)
//     "có quyền" / "được quyền"    -> PWR  (power/right)
//     "phải" / "nghĩa vụ"/"trách nhiệm" -> NRM (obligation)
//     temporal cues ("thời hạn", "không quá 30 ngày", ...) attach a Window node,
//  4. infers the document DOMAIN from term frequencies (no manual labeling),
//  5. inserts every extracted instance with t_text = t_fact = [now, infinity).
//
// The AST — not a manual label — carries the semantics (e.g. Window nodes for
// temporal limitations).
package main

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/docx"
	"computable-governance/compiler/internal/kernel"
)

// Extracted is the JSONB payload of an autonomously-ingested instance. It
// records provenance plus the AST whose structure defines the norm.
type Extracted struct {
	Kind     string       `json:"kind"`
	Chapter  string       `json:"chapter,omitempty"`
	Article  string       `json:"article,omitempty"`
	Modality string       `json:"modality"`
	Cue      string       `json:"cue,omitempty"`
	Temporal string       `json:"temporal,omitempty"`
	Text     string       `json:"text"`
	AST      *kernel.Expr `json:"ast"`
}

var (
	reChapter   = regexp.MustCompile(`^Chương\s+[IVXLCDM0-9]+`)
	reArticle   = regexp.MustCompile(`^Điều\s+(\d+)\s*\.?\s*(.*)$`)
	reDefined   = regexp.MustCompile(`^(?:\d+\.\s*)?(.{2,60}?)\s+là\s+\p{L}`)
	reDuration  = regexp.MustCompile(`(\d{1,4})\s*(ngày làm việc|ngày|tháng|năm|tuần|giờ)`)
	reEffective = regexp.MustCompile(`ngày\s+(\d{1,2})\s+tháng\s+(\d{1,2})\s+năm\s+(\d{4})`)

	// Statutory cross-references: "quy định tại [điểm …] [khoản …] Điều N[, M
	// và K] [của Bộ luật/Luật <tên>]". The article list becomes REF edges; the
	// optional "của …" tail switches the target document (absent or "… này"
	// keeps the citing document).
	reCitation = regexp.MustCompile(`(?i)(?:quy định tại|quy định ở|theo)\s+` +
		`((?:điểm\s+[\p{L}\d]{1,3}\s+)?(?:(?:các\s+)?khoản\s+\d+(?:\s*(?:,|và|hoặc)\s*\d+)*\s+)?` +
		`(?:các\s+)?điều\s+\d+(?:\s*(?:,|và|hoặc)\s*\d+)*)` +
		`(\s+của\s+(?:bộ\s+luật|luật)\s+[^,;.()\n]{0,60})?`)
	reArticleNums = regexp.MustCompile(`(?i)điều\s+(\d+(?:\s*(?:,|và|hoặc)\s*\d+)*)`)
	reNum         = regexp.MustCompile(`\d+`)
)

var (
	prohibitionCues = []string{"không được", "nghiêm cấm", "bị cấm", "cấm "}
	rightCues       = []string{"có quyền", "được quyền", "có thể"}
	obligationCues  = []string{"phải ", "có nghĩa vụ", "có trách nhiệm", "chịu trách nhiệm"}
	temporalCues    = []string{"thời hạn", "trong vòng", "không quá", "kể từ ngày", "chậm nhất", "ít nhất", "tối đa", "tối thiểu"}
)

// domainSignals maps candidate domains to characteristic Vietnamese terms.
var domainSignals = map[string][]string{
	"Labour / Employment Law": {"lao động", "hợp đồng lao động", "tiền lương", "làm thêm giờ", "người sử dụng lao động", "nghỉ hằng năm", "kỷ luật lao động", "công đoàn", "an toàn lao động"},
	"Taxation":                {"thuế", "người nộp thuế", "thu nhập chịu thuế", "khấu trừ", "hoàn thuế", "hóa đơn"},
	"Enterprise / Commercial": {"doanh nghiệp", "cổ đông", "vốn điều lệ", "hội đồng quản trị", "hợp đồng thương mại"},
	"Criminal Law":            {"tội phạm", "hình phạt", "phạt tù", "truy cứu trách nhiệm hình sự", "bị cáo"},
}

// RefExtracted is the JSONB payload of an extracted citation edge. The three
// REFPayload fields (source, target_iri, mode) drive refgraph traversal;
// article/text are provenance for the console.
type RefExtracted struct {
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	TargetIRI string `json:"target_iri"`
	Mode      string `json:"mode"`
	Article   string `json:"article,omitempty"`
	Text      string `json:"text,omitempty"`
}

type refEdge struct {
	ext  RefExtracted
	para int
	span int
}

// nameStops terminate a cited document name: the regex tail cannot lookahead,
// so clause text following the name ("… của Bộ luật Lao động và có đủ 15
// năm …") is cut here at the first connective.
var nameStops = []string{" và ", " thì ", " khi ", " nếu ", " được ", " bị ", " mà ", " theo ", " trong "}

// iriToken normalizes a document name to an IRI segment ("Bộ luật Dân sự" ->
// "Bộ-luật-Dân-sự").
func iriToken(name string) string {
	for _, s := range nameStops {
		if i := strings.Index(name, s); i > 0 {
			name = name[:i]
		}
	}
	return strings.Join(strings.Fields(strings.TrimSpace(name)), "-")
}

func articleIRI(docToken, num string) string {
	return fmt.Sprintf("urn:vn:%s:Đ%s", docToken, num)
}

// extractRefs finds statutory citations in one paragraph. source is the citing
// article's IRI; each cited article number yields one edge. seen dedups edges
// document-wide on (source, target, mode).
func extractRefs(p, docToken, article string, para int, seen map[string]bool) []refEdge {
	srcSeg := "preamble"
	if m := reNum.FindString(article); m != "" {
		srcSeg = "Đ" + m
	}
	source := fmt.Sprintf("urn:vn:%s:%s", docToken, srcSeg)
	mode := "cite"
	if strings.Contains(strings.ToLower(p), "sửa đổi, bổ sung") {
		mode = "amend"
	}

	var out []refEdge
	for _, m := range reCitation.FindAllStringSubmatch(p, -1) {
		listPart, tail := m[1], m[2]
		targetDoc := docToken
		if tail != "" && !strings.Contains(strings.ToLower(tail), "này") {
			name := strings.TrimPrefix(strings.TrimSpace(tail), "của ")
			targetDoc = iriToken(name)
		}
		am := reArticleNums.FindStringSubmatch(listPart)
		if am == nil {
			continue
		}
		for _, n := range reNum.FindAllString(am[1], -1) {
			target := articleIRI(targetDoc, n)
			if target == source {
				continue
			}
			key := source + "→" + target + "|" + mode
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, refEdge{
				ext: RefExtracted{
					Kind: "REF", Source: source, TargetIRI: target, Mode: mode,
					Article: article, Text: truncate(m[0], 200),
				},
				para: para,
				span: len([]rune(m[0])),
			})
		}
	}
	return out
}

func containsAny(hay string, cues []string) (string, bool) {
	for _, c := range cues {
		if strings.Contains(hay, c) {
			return strings.TrimSpace(c), true
		}
	}
	return "", false
}

// detectDuration converts the first "<n> <unit>" temporal phrase into an ISO-8601
// duration (e.g. "30 ngày" -> "P30D", "12 tháng" -> "P12M").
func detectDuration(low string) (string, bool) {
	m := reDuration.FindStringSubmatch(low)
	if m == nil {
		return "", false
	}
	n, _ := strconv.Atoi(m[1])
	switch {
	case strings.HasPrefix(m[2], "ngày"):
		return fmt.Sprintf("P%dD", n), true
	case m[2] == "tuần":
		return fmt.Sprintf("P%dW", n), true
	case m[2] == "tháng":
		return fmt.Sprintf("P%dM", n), true
	case m[2] == "năm":
		return fmt.Sprintf("P%dY", n), true
	case m[2] == "giờ":
		return fmt.Sprintf("PT%dH", n), true
	}
	return "", false
}

// classify applies the structural heuristics, returning constructor, modality,
// matched cue, and (if temporal) an ISO duration. ok=false means "no norm here".
func classify(text string) (c kernel.Constructor, modality, cue, iso string, ok bool) {
	low := strings.ToLower(text)
	_, temporal := containsAny(low, temporalCues)
	if temporal {
		iso, _ = detectDuration(low)
	}
	switch {
	case reDefined.MatchString(text):
		return kernel.CLS, "definition", "là", iso, true
	}
	if cue, hit := containsAny(low, prohibitionCues); hit {
		return kernel.GRD, "prohibition", cue, iso, true
	}
	if cue, hit := containsAny(low, rightCues); hit {
		return kernel.PWR, "right", cue, iso, true
	}
	if cue, hit := containsAny(low, obligationCues); hit {
		return kernel.NRM, "obligation", cue, iso, true
	}
	return "", "", "", "", false
}

func modalityPredicate(modality string) string {
	switch modality {
	case "definition":
		return "defined_as"
	case "prohibition":
		return "prohibited"
	case "right":
		return "empowered"
	default:
		return "obligated"
	}
}

// buildAST returns the condition AST; temporal clauses are wrapped in a Window.
func buildAST(modality, iso string) *kernel.Expr {
	base := kernel.Pred(modalityPredicate(modality), kernel.Var("subject"))
	if iso != "" {
		return kernel.Within(iso, "", base)
	}
	return base
}

// effectiveDate derives the corpus's promulgated effective date from a line
// containing "hiệu lực" plus a "ngày D tháng M năm YYYY" date; falls back to a
// declared epoch (the Labour Code took effect 2021-01-01).
func effectiveDate(paras []string) time.Time {
	fallback := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range paras {
		if !strings.Contains(strings.ToLower(p), "hiệu lực") {
			continue
		}
		m := reEffective.FindStringSubmatch(p)
		if m == nil {
			continue
		}
		d, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[3])
		if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 && y >= 1900 {
			return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
		}
	}
	return fallback
}

func detUUID(s string) string {
	h := sha1.Sum([]byte(s))
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func inferDomain(fullLower string) (string, map[string]int, int) {
	type scored struct {
		name  string
		score int
	}
	var ranked []scored
	best := map[string]int{}
	for name, terms := range domainSignals {
		s := 0
		for _, t := range terms {
			s += strings.Count(fullLower, t)
		}
		ranked = append(ranked, scored{name, s})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	winner := ranked[0].name
	for _, t := range domainSignals[winner] {
		if c := strings.Count(fullLower, t); c > 0 {
			best[t] = c
		}
	}
	return winner, best, ranked[0].score
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

const insertSQL = `
INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)
RETURNING pk`

// I9: every kernel row gets exactly one source-map row, in the same tx.
const sourceMapSQL = `
INSERT INTO source_map (instance_pk, locus, kind, span)
VALUES ($1, $2, $3, int4range(0, $4))`

func base(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}

// backfillSQL maps an already-ingested instance (matched by its deterministic
// UUID) to its source locus, inserting only where no source_map row exists.
const backfillSQL = `
INSERT INTO source_map (instance_pk, locus, kind, span)
SELECT k.pk, $2, $3, int4range(0, $4)
FROM kernel_instance k
LEFT JOIN source_map s ON s.instance_pk = k.pk
WHERE k.instance_id = $1::uuid AND s.id IS NULL`

// insertRefSQL is idempotent on instance_id: re-running ref extraction over an
// already-ingested corpus inserts only edges not yet present (the EXCLUDE
// constraint would otherwise reject the overlapping t_text slice).
const insertRefSQL = `
INSERT INTO kernel_instance (instance_id, constructor, payload, t_text, t_fact)
SELECT $1::uuid, 'REF', $2::jsonb, $3::tstzrange, $4::tstzrange
WHERE NOT EXISTS (SELECT 1 FROM kernel_instance WHERE instance_id = $1::uuid)
RETURNING pk`

func main() {
	backfill := flag.Bool("backfill-sourcemap", false,
		"do not ingest; only insert missing source_map rows for instances previously ingested from this corpus (matched by deterministic UUID)")
	refsOnly := flag.Bool("refs-only", false,
		"do not ingest clauses; only extract statutory citations and insert missing REF edges (idempotent)")
	flag.Parse()
	path := "../data/125_VBHN-VPQH_672381.docx"
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	paras, err := docx.Paragraphs(path)
	if err != nil {
		log.Fatalf("extract: %v", err)
	}

	// Domain inference over the full text (no manual labeling).
	fullLower := strings.ToLower(strings.Join(paras, "\n"))
	domain, evidence, score := inferDomain(fullLower)

	// Document token for source-map loci: the corpus file's base name.
	docToken := strings.TrimSuffix(strings.TrimSuffix(base(path), ".docx"), ".DOCX")

	// Structural walk + heuristic extraction.
	type item struct {
		id    string
		c     kernel.Constructor
		ext   Extracted
		locus string
		kind  string
		span  int
	}
	var items []item
	var refs []refEdge
	seenRef := map[string]bool{}
	var chapter, article string
	counts := map[kernel.Constructor]int{}
	temporalCount := 0

	for i, p := range paras {
		uKind := "clause"
		switch {
		case reChapter.MatchString(p):
			chapter = p
			continue
		case reArticle.MatchString(p):
			m := reArticle.FindStringSubmatch(p)
			article = "Điều " + m[1]
			uKind = "article"
			// fall through: the article heading itself may be a norm (e.g. a
			// prohibition heading), so we still classify it below.
		}

		// Citations occur in any paragraph, classified as a norm or not.
		refs = append(refs, extractRefs(p, docToken, article, i, seenRef)...)

		c, modality, cue, iso, ok := classify(p)
		if !ok {
			continue
		}
		if iso != "" {
			temporalCount++
		}
		ext := Extracted{
			Kind:     string(c),
			Chapter:  chapter,
			Article:  article,
			Modality: modality,
			Cue:      cue,
			Temporal: iso,
			Text:     truncate(p, 400),
			AST:      buildAST(modality, iso),
		}
		id := detUUID(fmt.Sprintf("%s|%d|%s", article, i, p))
		art := article
		if art == "" {
			art = "preamble"
		}
		items = append(items, item{
			id: id, c: c, ext: ext,
			locus: fmt.Sprintf("%s : %s ¶%d", docToken, art, i),
			kind:  uKind,
			span:  len([]rune(p)),
		})
		counts[c]++
	}

	// ---- Report: system-identified domain ----
	bar := strings.Repeat("═", 74)
	fmt.Println(bar)
	fmt.Println(" PHASE 4.1 — AUTONOMOUS KERNEL ACQUISITION (DOCX → K-hat)")
	fmt.Println(bar)
	fmt.Printf(" Source            : %s\n", path)
	fmt.Printf(" Paragraphs parsed : %d\n", len(paras))
	fmt.Printf(" SYSTEM-IDENTIFIED DOMAIN : %s  (signal score %d)\n", domain, score)
	fmt.Print(" Evidence terms    : ")
	printEvidence(evidence)
	fmt.Println(bar)
	internalRefs, externalRefs, amendRefs := 0, 0, 0
	for _, e := range refs {
		if strings.Contains(e.ext.TargetIRI, docToken) {
			internalRefs++
		} else {
			externalRefs++
		}
		if e.ext.Mode == "amend" {
			amendRefs++
		}
	}
	fmt.Printf(" Extracted instances: %d\n", len(items))
	fmt.Printf("   NRM (obligation) : %d\n", counts[kernel.NRM])
	fmt.Printf("   GRD (prohibition): %d\n", counts[kernel.GRD])
	fmt.Printf("   PWR (right)      : %d\n", counts[kernel.PWR])
	fmt.Printf("   CLS (definition) : %d\n", counts[kernel.CLS])
	fmt.Printf("   with Window (temporal): %d\n", temporalCount)
	fmt.Printf(" Citation edges (REF): %d  (internal %d, cross-document %d, amend-mode %d)\n",
		len(refs), internalRefs, externalRefs, amendRefs)
	for i, e := range refs {
		if i >= 5 {
			break
		}
		fmt.Printf("   • %s —%s→ %s  (%q)\n", e.ext.Source, e.ext.Mode, e.ext.TargetIRI, e.ext.Text)
	}
	fmt.Println(bar)

	// ---- Insert into DB ----
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	// Corpus-derived coordinate (WP-8, I8): parse the promulgated effective
	// date ("có hiệu lực kể từ ngày …") from the text; fall back to a declared
	// epoch. Deterministic → reproducible CNF across compilers (no time.Now()).
	epoch := effectiveDate(paras)
	validity, err := kernel.Since(epoch).Value()
	if err != nil {
		log.Fatalf("range: %v", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted := 0
	if *backfill {
		for _, it := range items {
			tag, err := tx.Exec(ctx, backfillSQL, it.id, it.locus, it.kind, it.span)
			if err != nil {
				log.Fatalf("backfill source_map %s: %v", it.id, err)
			}
			inserted += int(tag.RowsAffected())
		}
	} else if !*refsOnly {
		for _, it := range items {
			raw, err := json.Marshal(it.ext)
			if err != nil {
				log.Fatalf("marshal: %v", err)
			}
			var pk int64
			if err := tx.QueryRow(ctx, insertSQL, it.id, string(it.c), string(raw), validity, validity).Scan(&pk); err != nil {
				log.Fatalf("insert %s (%s): %v", it.id, it.c, err)
			}
			if _, err := tx.Exec(ctx, sourceMapSQL, pk, it.locus, it.kind, it.span); err != nil {
				log.Fatalf("source_map %s: %v", it.id, err)
			}
			inserted++
		}
	}

	refInserted, refSkipped := 0, 0
	if !*backfill {
		for _, e := range refs {
			raw, err := json.Marshal(e.ext)
			if err != nil {
				log.Fatalf("marshal ref: %v", err)
			}
			id := detUUID("REF|" + e.ext.Source + "|" + e.ext.TargetIRI + "|" + e.ext.Mode)
			var pk int64
			err = tx.QueryRow(ctx, insertRefSQL, id, string(raw), validity, validity).Scan(&pk)
			if errors.Is(err, pgx.ErrNoRows) {
				refSkipped++ // edge already present from an earlier run
				continue
			}
			if err != nil {
				log.Fatalf("insert REF %s→%s: %v", e.ext.Source, e.ext.TargetIRI, err)
			}
			art := e.ext.Article
			if art == "" {
				art = "preamble"
			}
			locus := fmt.Sprintf("%s : %s ¶%d ⇒ %s", docToken, art, e.para, e.ext.TargetIRI)
			if _, err := tx.Exec(ctx, sourceMapSQL, pk, locus, "citation", e.span); err != nil {
				log.Fatalf("source_map REF %s: %v", id, err)
			}
			refInserted++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	if *backfill {
		fmt.Printf(" Backfilled %d source_map rows (existing instances, I9).\n", inserted)
		return
	}
	if !*refsOnly {
		fmt.Printf(" Ingested %d instances into kernel_instance with [now, infinity).\n", inserted)
	}
	fmt.Printf(" REF edges: %d inserted, %d already present (idempotent).\n", refInserted, refSkipped)

	// ---- Show a couple of temporal (Window) exemplars ----
	fmt.Println(bar)
	fmt.Println(" Sample temporal (Window) extractions:")
	shown := 0
	for _, it := range items {
		if it.ext.Temporal == "" {
			continue
		}
		astJSON, _ := json.Marshal(it.ext.AST)
		fmt.Printf("  • [%s %s] %s\n    dur=%s ast=%s\n",
			it.c, it.ext.Article, truncate(it.ext.Text, 90), it.ext.Temporal, astJSON)
		if shown++; shown >= 3 {
			break
		}
	}
	fmt.Println(bar)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func printEvidence(ev map[string]int) {
	type kv struct {
		k string
		v int
	}
	var xs []kv
	for k, v := range ev {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].v > xs[j].v })
	parts := make([]string, 0, len(xs))
	for i, x := range xs {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%q×%d", x.k, x.v))
	}
	fmt.Println(strings.Join(parts, ", "))
}
