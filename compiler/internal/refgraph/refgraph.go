// Package refgraph answers the "Monday-Morning" impact query (WP-4): given a
// target IRI, which kernel instances transitively REFERENCE it, valid at the
// bitemporal read coordinates ⟨t_text, t_fact⟩?
//
// Edges are REF instances (constructor 'REF', payload {source, target_iri,
// mode}); the closure is computed by a recursive CTE over the REF rows that
// kernel_instance_at(tt, tf) admits — so the impacted set is a function of the
// eval coordinates (temporal read discipline, G4/I6).
package refgraph

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Querier is satisfied by *pgx.Conn and *pgxpool.Pool.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Node is one impacted instance: the referencing source IRI and its minimum
// reference distance from the target.
type Node struct {
	IRI   string
	Depth int
}

// impactSQL walks REF edges backwards from the target: an instance is impacted
// if it references the target, or references something impacted. UNION (not
// UNION ALL) dedups the working set, so cyclic REF graphs terminate.
const impactSQL = `
WITH RECURSIVE refs AS (
    SELECT payload->>'source'     AS source,
           payload->>'target_iri' AS target_iri
    FROM kernel_instance_at($1, $2)
    WHERE constructor = 'REF'
      AND payload ? 'source' AND payload ? 'target_iri'
),
closure(iri, depth) AS (
    SELECT $3::text, 0
  UNION
    SELECT r.source, c.depth + 1
    FROM refs r JOIN closure c ON r.target_iri = c.iri
)
SELECT iri, MIN(depth) AS depth
FROM closure
WHERE iri <> $3::text
GROUP BY iri
ORDER BY depth, iri`

// Impact returns every instance transitively referencing target, valid at
// ⟨tt, tf⟩, ordered by (depth, iri).
func Impact(ctx context.Context, q Querier, target string, tt, tf time.Time) ([]Node, error) {
	rows, err := q.Query(ctx, impactSQL, tt, tf, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.IRI, &n.Depth); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
