// Package coord parses the operator-supplied bitemporal read coordinates
// ⟨t_text, t_fact⟩ used by every reading command (WP-4, temporal read
// discipline). Coordinates are INPUTS at the CLI boundary; evaluation itself
// never calls time.Now() (I8) — it receives these as parameters.
package coord

import (
	"fmt"
	"time"
)

// Parse interprets the --at-text / --at-fact flag values. An empty value
// defaults to the current instant (UTC). Non-empty values must be RFC3339.
func Parse(atText, atFact string) (tText, tFact time.Time, err error) {
	tText, err = one(atText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("coord: --at-text: %w", err)
	}
	tFact, err = one(atFact)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("coord: --at-fact: %w", err)
	}
	return tText, tFact, nil
}

func one(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not RFC3339: %w", s, err)
	}
	return t.UTC(), nil
}
