// Package registry loads the versioned, semantically-inert parameter space R
// into exact evaluator Values, and provides the token-renaming helpers that
// witness Invariant I4 (registry inertness: a bijective renaming of tokens
// preserves all verdicts).
//
// R is INSERT-only and versioned (never mutated in place). A "snapshot" is the
// version of each token in force at a read coordinate: the highest version
// whose recorded_at ≤ the coordinate.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/evaluator"
	"computable-governance/compiler/internal/kernel"
)

// Querier is satisfied by *pgx.Conn and *pgxpool.Pool.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type ratJSON struct {
	Rat string `json:"rat"`
}

// Snapshot returns the latest version of every token (snapshot at "now").
func Snapshot(ctx context.Context, q Querier) (map[string]evaluator.Value, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (token) token, value
		FROM registry ORDER BY token, version DESC`)
	if err != nil {
		return nil, err
	}
	return scan(rows)
}

// SnapshotAt returns the version of each token in force at coordinate `at`:
// the highest version whose recorded_at ≤ at (versioned lookup, I6/I4).
func SnapshotAt(ctx context.Context, q Querier, at time.Time) (map[string]evaluator.Value, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (token) token, value
		FROM registry WHERE recorded_at <= $1
		ORDER BY token, version DESC`, at)
	if err != nil {
		return nil, err
	}
	return scan(rows)
}

func scan(rows pgx.Rows) (map[string]evaluator.Value, error) {
	defer rows.Close()
	out := map[string]evaluator.Value{}
	for rows.Next() {
		var token string
		var raw []byte
		if err := rows.Scan(&token, &raw); err != nil {
			return nil, err
		}
		v, err := decodeValue(token, raw)
		if err != nil {
			return nil, err
		}
		out[token] = v
	}
	return out, rows.Err()
}

// decodeValue maps a registry JSON value to an exact evaluator Value. A
// {"rat":"..."} becomes a KRat (no float64, I8); other scalars map by kind.
func decodeValue(token string, raw []byte) (evaluator.Value, error) {
	var r ratJSON
	if json.Unmarshal(raw, &r) == nil && r.Rat != "" {
		q, ok := new(big.Rat).SetString(r.Rat)
		if !ok {
			return evaluator.Value{}, fmt.Errorf("registry %s: bad rational %q", token, r.Rat)
		}
		return evaluator.VRat(q), nil
	}
	var scalar any
	if err := json.Unmarshal(raw, &scalar); err != nil {
		return evaluator.Value{}, fmt.Errorf("registry %s: %w", token, err)
	}
	switch v := scalar.(type) {
	case string:
		return evaluator.VStr(v), nil
	case bool:
		return evaluator.VBool(v), nil
	case float64:
		// JSON numbers are integer-valued in R; promote exactly.
		return evaluator.VInt(int64(v)), nil
	}
	return evaluator.Value{}, fmt.Errorf("registry %s: unsupported value shape", token)
}

// RenameTokens applies a token bijection σ to a registry snapshot's keys.
func RenameTokens(reg map[string]evaluator.Value, sigma map[string]string) map[string]evaluator.Value {
	out := make(map[string]evaluator.Value, len(reg))
	for k, v := range reg {
		if nk, ok := sigma[k]; ok {
			out[nk] = v
		} else {
			out[k] = v
		}
	}
	return out
}

// RenameLookups returns a copy of e with every OpLookup token renamed through
// σ (the AST-side half of an I4 bijective rename). The input is not mutated.
func RenameLookups(e *kernel.Expr, sigma map[string]string) *kernel.Expr {
	if e == nil {
		return nil
	}
	n := *e
	if e.Op == kernel.OpLookup {
		if nn, ok := sigma[e.Name]; ok {
			n.Name = nn
		}
	}
	if len(e.Args) > 0 {
		n.Args = make([]*kernel.Expr, len(e.Args))
		for i, a := range e.Args {
			n.Args[i] = RenameLookups(a, sigma)
		}
	}
	if e.Count != nil {
		c := *e.Count
		c.Where = RenameLookups(e.Count.Where, sigma)
		n.Count = &c
	}
	if e.Window != nil {
		w := *e.Window
		w.Body = RenameLookups(e.Window.Body, sigma)
		n.Window = &w
	}
	return &n
}
