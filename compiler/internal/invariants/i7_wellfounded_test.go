package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"computable-governance/compiler/internal/kernel"
	"computable-governance/compiler/internal/validation"
)

// I7 — Stratified reflection / well-foundedness (D0 §6; D1.5 T8). The Lean
// obligation strata_wellFounded is still `sorry`, but I7 has concrete
// operational content that IS checkable over the live store: the reflection
// (REF) graph must be well-founded, i.e. acyclic — no instance may transitively
// reference itself, or the reflection strata would not be ℕ-indexed / <-well-
// founded and evaluation of the closure could fail to terminate.
//
// refReachSQL walks REF edges (payload {source -> target_iri}) and reports how
// many nodes are reachable from themselves. UNION dedups the frontier so the
// walk terminates even if a cycle exists; a nonzero count is a genuine I7
// violation (a cyclic reflection graph).
const refReachSQL = `
WITH RECURSIVE refs AS (
    SELECT payload->>'source'     AS s,
           payload->>'target_iri' AS t
    FROM kernel_instance
    WHERE constructor = 'REF'
      AND payload ? 'source' AND payload ? 'target_iri'
),
reach(origin, node, depth) AS (
    SELECT s, t, 1 FROM refs
  UNION
    SELECT rc.origin, r.t, rc.depth + 1
    FROM reach rc JOIN refs r ON r.s = rc.node
    WHERE rc.depth < 10000
)
SELECT (SELECT count(*) FROM refs) AS ref_edges,
       (SELECT count(*) FROM reach WHERE origin = node) AS cyclic_nodes`

// TestI7ReflectionWellFounded asserts the stored REF graph is acyclic.
func TestI7ReflectionWellFounded(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	var edges, cyclic int
	if err := conn.QueryRow(ctx, refReachSQL).Scan(&edges, &cyclic); err != nil {
		t.Fatalf("reflection-graph query: %v", err)
	}
	if edges == 0 {
		t.Skip("no REF edges in store; reflection graph vacuously well-founded")
	}
	if cyclic != 0 {
		t.Fatalf("REF reflection graph has %d node(s) reachable from themselves — not well-founded (I7 / D1.5 T8)", cyclic)
	}
	t.Logf("I7: reflection graph well-founded (acyclic) over %d REF edge(s)", edges)
}

// TestStoreWideFalsificationClean proves that every instance ALREADY in the
// store lies inside the frozen kernel: its constructor is in the closed basis B
// (I3, no 8th constructor) and any T AST it carries uses only sub-Turing
// operators (I1/I7). Equivalently: replaying the falsification screen (the same
// gate cmd/falsify applies to candidate inputs) over the whole live store emits
// zero FALSIFICATION-CANDIDATEs.
func TestStoreWideFalsificationClean(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	rows, err := conn.Query(ctx, `SELECT pk, constructor, payload FROM kernel_instance`)
	if err != nil {
		t.Fatalf("query store: %v", err)
	}
	defer rows.Close()

	checked, withAST := 0, 0
	for rows.Next() {
		var pk int64
		var constructor string
		var payload []byte
		if err := rows.Scan(&pk, &constructor, &payload); err != nil {
			t.Fatalf("scan: %v", err)
		}

		var env map[string]json.RawMessage
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Fatalf("pk=%d: unmarshal payload: %v", pk, err)
		}
		var ast *kernel.Expr
		if raw, ok := env["ast"]; ok {
			var e kernel.Expr
			if err := json.Unmarshal(raw, &e); err != nil {
				t.Fatalf("pk=%d: unmarshal ast: %v", pk, err)
			}
			ast = &e
			withAST++
		}

		if f := validation.Screen(fmt.Sprintf("pk=%d", pk), constructor, ast); f != nil {
			t.Fatalf("stored instance is NOT falsification-clean: %s", f.Error())
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if checked == 0 {
		t.Skip("empty store; nothing to assert")
	}
	if withAST == 0 {
		t.Fatal("no stored instance carried a T AST; the sub-Turing check was vacuous")
	}
	t.Logf("store-wide falsification-clean: %d instances (%d with a T AST) within B and sub-Turing T", checked, withAST)
}
