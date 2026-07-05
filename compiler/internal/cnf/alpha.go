package cnf

import (
	"fmt"
	"sort"
	"strings"
)

// Record is one exportable kernel row whose payload has already been decoded
// from JSONB (and had embedded T ASTs canonicalized).
type Record struct {
	InstanceID  string // original store UUID; replaced by the α-name on export
	Constructor string
	Locus       string // source_map locus; "" when the row is unmapped
	TText       string
	TFact       string
	Payload     any // decoded JSON value (map[string]any / []any / scalars)
}

// AlphaRename implements the WP-5 identity discipline: store-generated UUIDs
// are α-renamed to sequential ids ("k000001", ...) so that two independent
// compilers ingesting the same corpus emit comparable exports.
//
// Records are ordered by the content key (constructor, locus, payload shape,
// t_text, t_fact), where "shape" is the canonical payload with every embedded
// instance reference masked — the key must not depend on the very UUIDs being
// renamed. Ids are assigned in that order and every reference inside every
// payload (e.g. GRD body/defeats lists) is rewritten through the same map.
//
// ambiguous counts records whose content key collides with another's; their
// relative order falls back to the original UUID, which is deterministic for
// re-exports of one store but NOT comparable across independent compilers.
func AlphaRename(recs []Record) (out []Record, rename map[string]string, ambiguous int) {
	idSet := make(map[string]bool, len(recs))
	for _, r := range recs {
		idSet[r.InstanceID] = true
	}

	type keyed struct {
		Record
		key string
	}
	ks := make([]keyed, len(recs))
	for i, r := range recs {
		shape := hashHex(canonicalMarshal(maskRefs(r.Payload, idSet)))
		ks[i] = keyed{r, strings.Join([]string{r.Constructor, r.Locus, shape, r.TText, r.TFact}, "\x00")}
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].key != ks[j].key {
			return ks[i].key < ks[j].key
		}
		return ks[i].InstanceID < ks[j].InstanceID
	})

	rename = make(map[string]string, len(ks))
	for i, k := range ks {
		rename[k.InstanceID] = fmt.Sprintf("k%06d", i+1)
		if (i > 0 && ks[i-1].key == k.key) || (i+1 < len(ks) && ks[i+1].key == k.key) {
			ambiguous++
		}
	}

	out = make([]Record, len(ks))
	for i, k := range ks {
		r := k.Record
		r.InstanceID = rename[r.InstanceID]
		r.Payload = rewriteRefs(r.Payload, rename)
		out[i] = r
	}
	return out, rename, ambiguous
}

// maskRefs returns a copy of v with every string that is a known instance
// UUID replaced by the placeholder "@ref" (identity-independent shape).
func maskRefs(v any, idSet map[string]bool) any {
	return mapStrings(v, func(s string) string {
		if idSet[s] {
			return "@ref"
		}
		return s
	})
}

// rewriteRefs returns a copy of v with every known instance UUID replaced by
// its α-name.
func rewriteRefs(v any, rename map[string]string) any {
	return mapStrings(v, func(s string) string {
		if n, ok := rename[s]; ok {
			return n
		}
		return s
	})
}

// mapStrings structurally copies decoded JSON, applying f to every string
// (values only — payload object keys are never instance ids).
func mapStrings(v any, f func(string) string) any {
	switch t := v.(type) {
	case string:
		return f(t)
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = mapStrings(val, f)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, val := range t {
			s[i] = mapStrings(val, f)
		}
		return s
	default:
		return v
	}
}
