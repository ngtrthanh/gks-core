// Command cnf_export dumps the entire kernel_instance store in Canonical Normal
// Form (NDJSON) for exact syntactic diffing / reproducibility (Invariant I8).
//
// For every row it canonicalizes any embedded T AST (any nested object carrying
// an "op" field), key-sorts the payload, and emits a deterministic record with
// a per-row SHA-256 `cnf_hash`. Records are sorted by instance_id, and a single
// export-wide digest is printed as the store's reproducibility fingerprint.
//
// Reads go through kernel_instance_at(now(), now()) (temporal discipline, G4).
// Output path is argv[1] (default "dump.cnf").
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/jackc/pgx/v5"

	"computable-governance/compiler/internal/cnf"
	"computable-governance/compiler/internal/kernel"
)

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

// marshalNoEscape encodes deterministically (map keys sorted by encoding/json)
// without HTML-escaping.
func marshalNoEscape(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// canonicalizeValue walks arbitrary decoded JSON and replaces every AST node
// (any object with an "op" field) with its Canonical Normal Form.
func canonicalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if _, isAST := t["op"]; isAST {
			raw, err := json.Marshal(t)
			if err == nil {
				var e kernel.Expr
				if json.Unmarshal(raw, &e) == nil {
					var m map[string]any
					if json.Unmarshal(cnf.CanonicalJSON(&e), &m) == nil {
						return m
					}
				}
			}
			return t
		}
		for k, val := range t {
			t[k] = canonicalizeValue(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = canonicalizeValue(val)
		}
		return t
	default:
		return v
	}
}

type record struct {
	InstanceID  string          `json:"instance_id"`
	Constructor string          `json:"constructor"`
	TText       string          `json:"t_text"`
	TFact       string          `json:"t_fact"`
	CNFHash     string          `json:"cnf_hash"`
	Payload     json.RawMessage `json:"payload"`
}

func main() {
	outPath := "dump.cnf"
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT instance_id::text, constructor::text, t_text::text, t_fact::text, payload
		FROM kernel_instance_at(now(), now())`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}

	var recs []record
	for rows.Next() {
		var id, ctor, tt, tf string
		var payload []byte
		if err := rows.Scan(&id, &ctor, &tt, &tf, &payload); err != nil {
			log.Fatalf("scan: %v", err)
		}
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			log.Fatalf("decode payload %s: %v", id, err)
		}
		canonical := marshalNoEscape(canonicalizeValue(decoded))
		recs = append(recs, record{
			InstanceID:  id,
			Constructor: ctor,
			TText:       tt,
			TFact:       tf,
			CNFHash:     hashHex(canonical),
			Payload:     json.RawMessage(canonical),
		})
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	// Deterministic order.
	sort.Slice(recs, func(i, j int) bool { return recs[i].InstanceID < recs[j].InstanceID })

	if err := os.MkdirAll(dir(outPath), 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create %s: %v", outPath, err)
	}
	defer f.Close()

	digest := sha256.New()
	for _, r := range recs {
		line := marshalNoEscape(r)
		if _, err := f.Write(append(line, '\n')); err != nil {
			log.Fatalf("write: %v", err)
		}
		digest.Write([]byte(r.CNFHash))
	}

	fmt.Printf("CNF export: %d records -> %s\n", len(recs), outPath)
	fmt.Printf("export digest (SHA-256): %s\n", hex.EncodeToString(digest.Sum(nil)))
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func dir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
