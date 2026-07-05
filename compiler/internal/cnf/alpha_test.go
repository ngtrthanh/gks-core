package cnf

import (
	"bytes"
	"testing"
)

// store builds the same two-record corpus (an NRM and a GRD that defeats it)
// under an arbitrary UUID assignment and input order.
func store(nrmID, grdID string, grdFirst bool) []Record {
	nrm := Record{
		InstanceID:  nrmID,
		Constructor: "NRM",
		Locus:       "§8.7.1",
		TText:       "[2026-01-01,)", TFact: "[2026-01-01,)",
		Payload: map[string]any{"bearer": "org", "act": "control-nonconformity", "force": "O"},
	}
	grd := Record{
		InstanceID:  grdID,
		Constructor: "GRD",
		Locus:       "§8.7.1(d)",
		TText:       "[2026-01-01,)", TFact: "[2026-01-01,)",
		Payload: map[string]any{
			"priority": float64(10),
			"defeats":  []any{nrmID}, // cross-reference that must be α-rewritten
		},
	}
	if grdFirst {
		return []Record{grd, nrm}
	}
	return []Record{nrm, grd}
}

func export(recs []Record) []byte {
	out, _, _ := AlphaRename(recs)
	var buf bytes.Buffer
	for _, r := range out {
		buf.WriteString(r.InstanceID + " " + r.Constructor + " ")
		buf.Write(canonicalMarshal(r.Payload))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// Two independent compilers assign different UUIDs in different row orders;
// the α-renamed export must be byte-identical (WP-5 / validation program).
func TestAlphaRenameIsIdentityAndOrderIndependent(t *testing.T) {
	a := export(store("aaaaaaaa-0000-4000-8000-000000000001", "aaaaaaaa-0000-4000-8000-000000000002", false))
	b := export(store("ffffffff-0000-4000-8000-00000000000f", "00000000-0000-4000-8000-000000000001", true))
	if !bytes.Equal(a, b) {
		t.Fatalf("exports differ across UUID assignments:\nA:\n%sB:\n%s", a, b)
	}
	if bytes.Contains(a, []byte("aaaaaaaa")) {
		t.Fatal("raw store UUID leaked into the export")
	}
}

func TestAlphaRenameRewritesReferences(t *testing.T) {
	out, rename, ambiguous := AlphaRename(store("aaaaaaaa-0000-4000-8000-000000000001", "aaaaaaaa-0000-4000-8000-000000000002", false))
	if ambiguous != 0 {
		t.Fatalf("distinct content keys reported ambiguous: %d", ambiguous)
	}
	var grd Record
	for _, r := range out {
		if r.Constructor == "GRD" {
			grd = r
		}
	}
	defeats := grd.Payload.(map[string]any)["defeats"].([]any)
	want := rename["aaaaaaaa-0000-4000-8000-000000000001"]
	if len(defeats) != 1 || defeats[0] != want {
		t.Fatalf("GRD defeats not rewritten: got %v, want [%s]", defeats, want)
	}
}

// Content-key collisions must be surfaced, and their fallback order must be
// deterministic for one store.
func TestAlphaRenameFlagsAmbiguity(t *testing.T) {
	dup := func(id string) Record {
		return Record{InstanceID: id, Constructor: "CLS", Locus: "x",
			TText: "[2026-01-01,)", TFact: "[2026-01-01,)",
			Payload: map[string]any{"entity": "e", "class_token": "c"}}
	}
	_, _, ambiguous := AlphaRename([]Record{dup("aaaaaaaa-0000-4000-8000-000000000001"), dup("aaaaaaaa-0000-4000-8000-000000000002")})
	if ambiguous != 2 {
		t.Fatalf("expected both colliding records flagged, got %d", ambiguous)
	}
}
