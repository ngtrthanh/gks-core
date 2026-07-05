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
	reChapter  = regexp.MustCompile(`^Chương\s+[IVXLCDM0-9]+`)
	reArticle  = regexp.MustCompile(`^Điều\s+(\d+)\s*\.?\s*(.*)$`)
	reDefined  = regexp.MustCompile(`^(?:\d+\.\s*)?(.{2,60}?)\s+là\s+\p{L}`)
	reDuration = regexp.MustCompile(`(\d{1,4})\s*(ngày làm việc|ngày|tháng|năm|tuần|giờ)`)
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
VALUES ($1::uuid, $2, $3::jsonb, $4::tstzrange, $5::tstzrange)`

func main() {
	path := "../data/125_VBHN-VPQH_672381.docx"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	paras, err := docx.Paragraphs(path)
	if err != nil {
		log.Fatalf("extract: %v", err)
	}

	// Domain inference over the full text (no manual labeling).
	fullLower := strings.ToLower(strings.Join(paras, "\n"))
	domain, evidence, score := inferDomain(fullLower)

	// Structural walk + heuristic extraction.
	type item struct {
		id  string
		c   kernel.Constructor
		ext Extracted
	}
	var items []item
	var chapter, article string
	counts := map[kernel.Constructor]int{}
	temporalCount := 0

	for i, p := range paras {
		switch {
		case reChapter.MatchString(p):
			chapter = p
			continue
		case reArticle.MatchString(p):
			m := reArticle.FindStringSubmatch(p)
			article = "Điều " + m[1]
			// fall through: the article heading itself may be a norm (e.g. a
			// prohibition heading), so we still classify it below.
		}

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
		items = append(items, item{id: id, c: c, ext: ext})
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
	fmt.Printf(" Extracted instances: %d\n", len(items))
	fmt.Printf("   NRM (obligation) : %d\n", counts[kernel.NRM])
	fmt.Printf("   GRD (prohibition): %d\n", counts[kernel.GRD])
	fmt.Printf("   PWR (right)      : %d\n", counts[kernel.PWR])
	fmt.Printf("   CLS (definition) : %d\n", counts[kernel.CLS])
	fmt.Printf("   with Window (temporal): %d\n", temporalCount)
	fmt.Println(bar)

	// ---- Insert into DB ----
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	validity, err := kernel.Since(time.Now().UTC()).Value()
	if err != nil {
		log.Fatalf("range: %v", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted := 0
	for _, it := range items {
		raw, err := json.Marshal(it.ext)
		if err != nil {
			log.Fatalf("marshal: %v", err)
		}
		if _, err := tx.Exec(ctx, insertSQL, it.id, string(it.c), string(raw), validity, validity); err != nil {
			log.Fatalf("insert %s (%s): %v", it.id, it.c, err)
		}
		inserted++
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Printf(" Ingested %d instances into kernel_instance with [now, infinity).\n", inserted)

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
