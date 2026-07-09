package invariants

import (
	"context"
	"sort"
	"testing"

	"computable-governance/compiler/internal/kernel"
)

// Registry Law (D0 §7, Θ(1) basis growth): the constructor basis B is FIXED at
// six; ingesting more corpora — and structurally different normative domains —
// never grows it. The store already spans heterogeneous domains (Vietnamese
// labour statute, ISO 9001 quality standard, US federal tax code §121,
// KPI/policy). This test asserts that (a) several distinct domains are present,
// (b) every domain's constructor set is a subset of the closed basis B, and
// (c) the union across ALL domains is still ≤ 6 — i.e. adding domains added no
// new constructor (the basis is Θ(1), not O(domains)).
func TestRegistryLawBoundedBasisAcrossDomains(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()

	rows, err := conn.Query(ctx, `
		SELECT DISTINCT split_part(s.locus, ':', 1) AS domain, k.constructor::text
		FROM kernel_instance k
		JOIN source_map s ON s.instance_pk = k.pk`)
	if err != nil {
		t.Fatalf("query domains: %v", err)
	}
	defer rows.Close()

	perDomain := map[string]map[string]bool{}
	union := map[string]bool{}
	for rows.Next() {
		var domain, ctor string
		if err := rows.Scan(&domain, &ctor); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if perDomain[domain] == nil {
			perDomain[domain] = map[string]bool{}
		}
		perDomain[domain][ctor] = true
		union[ctor] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if len(perDomain) < 3 {
		t.Skipf("only %d domain(s) in store; need ≥3 heterogeneous domains to exercise the Registry Law", len(perDomain))
	}

	// (b) every domain uses only constructors in the closed basis B.
	for domain, ctors := range perDomain {
		for c := range ctors {
			if !kernel.Constructor(c).Valid() {
				t.Fatalf("domain %q uses constructor %q outside the closed basis B (Registry Law violated)", domain, c)
			}
		}
	}

	// (c) the union across all domains never exceeds |B| = 6.
	basis := make([]string, 0, len(union))
	for c := range union {
		basis = append(basis, c)
	}
	sort.Strings(basis)
	if len(basis) > 6 {
		t.Fatalf("basis grew to %d constructors %v across %d domains — Θ(1) Registry Law violated", len(basis), basis, len(perDomain))
	}

	domains := make([]string, 0, len(perDomain))
	for d := range perDomain {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	t.Logf("Registry Law held across %d domains %v: basis is %v (|B|=%d ≤ 6, Θ(1))",
		len(perDomain), domains, basis, len(basis))
}
