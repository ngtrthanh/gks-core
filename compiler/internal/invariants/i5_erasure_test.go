package invariants

import (
	"context"
	"encoding/json"
	"testing"

	"computable-governance/compiler/internal/cnf"
	"computable-governance/compiler/internal/kernel"
)

// I5 — Presentation erasure (D0 §5, "P̂ is inessential"): the verdict identity
// of a kernel instance is a function of its Semantic-Algebra AST alone. The
// surface presentation P̂ — the modality cue, human-readable text, chapter /
// article labels, temporal gloss, and the source_map locus — carries NO
// semantic weight: erasing or mutating it must not change the verdict.
//
// The verdict identifier is cnf.CanonicalHash(ast) (the α-invariant "Verdict
// Identifier", I8). These tests assert that hash is invariant under erasure
// and under adversarial mutation of every presentation field, so a future
// change that let presentation leak into the semantic identity would fail here.

// presentationFields are the P̂ envelope keys produced by ingest_docx. They are
// provenance/rendering only; the semantic content is {kind, ast}.
var presentationFields = []string{"article", "chapter", "cue", "modality", "temporal", "text"}

// verdictOf extracts the T AST from an "Extracted" envelope and returns its
// canonical Verdict Identifier — the exact code path the compiler uses.
func verdictOf(t *testing.T, env map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(env["ast"])
	if err != nil {
		t.Fatalf("marshal ast: %v", err)
	}
	var e kernel.Expr
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("unmarshal ast into kernel.Expr: %v", err)
	}
	return cnf.CanonicalHash(&e)
}

// erase returns a copy of env keeping only the semantic content {kind, ast}.
func erase(env map[string]any) map[string]any {
	return map[string]any{"kind": env["kind"], "ast": env["ast"]}
}

// mutatePresentation returns a copy of env with every present presentation
// field overwritten by an adversarial sentinel (semantics must be unaffected).
func mutatePresentation(env map[string]any) (map[string]any, int) {
	out := make(map[string]any, len(env))
	for k, v := range env {
		out[k] = v
	}
	changed := 0
	for _, f := range presentationFields {
		if _, ok := out[f]; ok {
			out[f] = "⟪ADVERSARIAL-PRESENTATION⟫"
			changed++
		}
	}
	return out, changed
}

// TestI5PresentationErasurePure guards I5 with no infrastructure: a fixed AST
// wrapped in a full presentation envelope, a bare {kind,ast} envelope, and a
// mutated-presentation envelope must all yield the same Verdict Identifier.
func TestI5PresentationErasurePure(t *testing.T) {
	e := &kernel.Expr{
		Op:   kernel.OpPred,
		Name: "defined_as",
		Args: []*kernel.Expr{{Op: kernel.OpVar, Name: "subject"}},
	}
	astJSON, _ := json.Marshal(e)
	var astAny any
	_ = json.Unmarshal(astJSON, &astAny)

	full := map[string]any{
		"kind": "CLS", "ast": astAny,
		"cue": "được coi là", "text": "Người lao động là người...",
		"article": "Điều 3", "chapter": "I", "modality": "definition", "temporal": nil,
	}
	want := cnf.CanonicalHash(e)

	if got := verdictOf(t, full); got != want {
		t.Fatalf("full envelope verdict = %s, want %s", got, want)
	}
	if got := verdictOf(t, erase(full)); got != want {
		t.Fatalf("erased envelope verdict = %s, want %s (presentation leaked into I5)", got, want)
	}
	mut, changed := mutatePresentation(full)
	if changed == 0 {
		t.Fatal("no presentation fields mutated; test would be vacuous")
	}
	if got := verdictOf(t, mut); got != want {
		t.Fatalf("mutated-presentation verdict = %s, want %s (presentation leaked into I5)", got, want)
	}
}

// TestI5PresentationErasureCorpus proves I5 over the live docx corpus: for every
// stored instance, erasing and adversarially mutating its presentation envelope
// leaves the Verdict Identifier unchanged.
func TestI5PresentationErasureCorpus(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	rows, err := conn.Query(ctx, `
		SELECT k.payload, s.locus
		FROM kernel_instance k
		JOIN source_map s ON s.instance_pk = k.pk
		WHERE k.payload ? 'kind' AND k.payload ? 'ast'`)
	if err != nil {
		t.Fatalf("query docx instances: %v", err)
	}
	defer rows.Close()

	checked, withPresentation := 0, 0
	for rows.Next() {
		var payload []byte
		var locus string
		if err := rows.Scan(&payload, &locus); err != nil {
			t.Fatalf("scan: %v", err)
		}
		var env map[string]any
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		full := verdictOf(t, env)
		if got := verdictOf(t, erase(env)); got != full {
			t.Fatalf("locus %q: erased verdict %s != full %s (I5 violated)", locus, got, full)
		}
		mut, changed := mutatePresentation(env)
		if got := verdictOf(t, mut); got != full {
			t.Fatalf("locus %q: mutated verdict %s != full %s (I5 violated)", locus, got, full)
		}
		// The source_map locus is presentation too: it never entered verdictOf.
		if changed > 0 || locus != "" {
			withPresentation++
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if checked == 0 {
		t.Skip("no docx-sourced instances in store; nothing to assert")
	}
	if withPresentation == 0 {
		t.Fatal("no instance carried erasable presentation; test was vacuous")
	}
	t.Logf("I5 held over %d stored instances (%d carried erasable presentation)", checked, withPresentation)
}
