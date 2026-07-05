package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBuildInstancesSerialize verifies the D8 Run 1 instances build and that
// every payload serializes to a JSON object (satisfying the DB payload_is_object
// check) and every temporal bound renders to a tstzrange literal. It does NOT
// touch the database.
func TestBuildInstancesSerialize(t *testing.T) {
	insts, err := buildInstances(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build instances: %v", err)
	}
	if len(insts) != 4 {
		t.Fatalf("expected 4 instances, got %d", len(insts))
	}
	for _, ki := range insts {
		if !ki.Constructor.Valid() {
			t.Errorf("invalid constructor %q", ki.Constructor)
		}
		var obj map[string]any
		if err := json.Unmarshal(ki.Payload, &obj); err != nil {
			t.Errorf("%s %s: payload is not a JSON object: %v", ki.Constructor, ki.InstanceID, err)
			continue
		}
		tt, err := ki.TText.Value()
		if err != nil {
			t.Errorf("%s: t_text literal: %v", ki.InstanceID, err)
		}
		if _, ok := tt.(string); !ok {
			t.Errorf("%s: t_text is not a string literal", ki.InstanceID)
		}
		// Emit the mapping for inspection under `go test -v`.
		pretty, _ := json.MarshalIndent(obj, "", "  ")
		t.Logf("\n%s %s  t_text=%v\npayload=%s", ki.Constructor, ki.InstanceID, tt, pretty)
	}
}
