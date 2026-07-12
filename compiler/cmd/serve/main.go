// Command serve is the READ-ONLY web console for the K̂ store: a single-binary
// HTTP server (stdlib + pgx) that embeds its UI and exposes the kernel over a
// small JSON API. Every read is anchored to explicit bitemporal coordinates
// ⟨t_text, t_fact⟩ (WP-4 temporal read discipline, I6): the coordinates arrive
// as query parameters and default to "now" at the HTTP boundary — evaluation
// itself never calls time.Now() (I8).
//
// The console issues only SELECTs (I1/I2: reading is never writing) and runs
// as the least-privilege role by default.
//
//	serve [-addr :8787]
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"computable-governance/compiler/internal/coord"
	"computable-governance/compiler/internal/refgraph"
)

//go:embed index.html
var ui embed.FS

// connString honors DATABASE_URL / PG* env vars, defaulting to the compose
// `db` service on localhost:5435 (same convention as every other command).
func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("PGUSER", "e_writer"),
		getenv("PGPASSWORD", "e_writer_dev"),
		getenv("PGHOST", "localhost"),
		getenv("PGPORT", "5435"),
		getenv("PGDATABASE", "governance"),
	)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type server struct {
	pool *pgxpool.Pool
}

func main() {
	addr := flag.String("addr", ":8787", "listen address")
	flag.Parse()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString())
	if err != nil {
		log.Fatalf("serve: connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("serve: ping: %v", err)
	}

	s := &server{pool: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/meta", s.meta)
	mux.HandleFunc("GET /api/instances", s.instances)
	mux.HandleFunc("GET /api/instance", s.instance)
	mux.HandleFunc("GET /api/verdicts", s.verdicts)
	mux.HandleFunc("GET /api/impact", s.impact)
	mux.HandleFunc("GET /api/iris", s.iris)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page, _ := ui.ReadFile("index.html")
		_, _ = w.Write(page)
	})

	log.Printf("serve: kernel console on http://localhost%s (store %s)", *addr, getenv("PGHOST", "localhost")+":"+getenv("PGPORT", "5435"))
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// coords parses ?tt=&tf= (RFC3339; empty = now UTC).
func coords(r *http.Request) (tt, tf time.Time, err error) {
	return coord.Parse(r.URL.Query().Get("tt"), r.URL.Query().Get("tf"))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("serve: encode: %v", err)
	}
}

func httpErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// meta reports the store census at the read coordinates plus journal totals.
func (s *server) meta(w http.ResponseWriter, r *http.Request) {
	tt, tf, err := coords(r)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()

	type ccount struct {
		Constructor string `json:"constructor"`
		Count       int    `json:"count"`
	}
	out := struct {
		TT            time.Time `json:"tt"`
		TF            time.Time `json:"tf"`
		AtCoords      int       `json:"at_coords"`
		TotalRows     int       `json:"total_rows"`
		ByConstructor []ccount  `json:"by_constructor"`
		Verdicts      int       `json:"verdicts"`
		Events        int       `json:"events"`
		Machines      int       `json:"machines"`
		IngestionRuns int       `json:"ingestion_runs"`
	}{TT: tt, TF: tf, ByConstructor: []ccount{}}

	rows, err := s.pool.Query(ctx, `
		SELECT constructor::text, count(*)::int
		FROM kernel_instance_at($1, $2)
		GROUP BY constructor ORDER BY constructor`, tt, tf)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	for rows.Next() {
		var c ccount
		if err := rows.Scan(&c.Constructor, &c.Count); err != nil {
			rows.Close()
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		out.AtCoords += c.Count
		out.ByConstructor = append(out.ByConstructor, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}

	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM kernel_instance),
		       (SELECT count(*) FROM verdict),
		       (SELECT count(*) FROM world_event),
		       (SELECT count(*) FROM e_machine),
		       (SELECT count(*) FROM ingestion_run)`).
		Scan(&out.TotalRows, &out.Verdicts, &out.Events, &out.Machines, &out.IngestionRuns)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, out)
}

// instanceRow is one ledger line: a kernel row valid at the read coordinates,
// joined with its source locus (I9).
type instanceRow struct {
	PK          int64           `json:"pk"`
	InstanceID  string          `json:"instance_id"`
	Constructor string          `json:"constructor"`
	Payload     json.RawMessage `json:"payload"`
	TText       string          `json:"t_text"`
	TFact       string          `json:"t_fact"`
	Locus       *string         `json:"locus"`
	Kind        *string         `json:"kind"`
}

// instances lists kernel rows valid at ⟨tt, tf⟩, filterable by constructor and
// a free-text probe over locus + payload.
func (s *server) instances(w http.ResponseWriter, r *http.Request) {
	tt, tf, err := coords(r)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	rows, err := s.pool.Query(r.Context(), `
		SELECT k.pk, k.instance_id::text, k.constructor::text, k.payload,
		       k.t_text::text, k.t_fact::text, s.locus, s.kind,
		       count(*) OVER ()::int AS total
		FROM kernel_instance_at($1, $2) k
		LEFT JOIN source_map s ON s.instance_pk = k.pk
		WHERE ($3 = '' OR k.constructor::text = $3)
		  AND ($4 = '' OR s.locus ILIKE '%'||$4||'%' OR k.payload::text ILIKE '%'||$4||'%')
		ORDER BY s.locus NULLS LAST, k.pk
		LIMIT $5 OFFSET $6`,
		tt, tf, q.Get("c"), q.Get("q"), limit, offset)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	total := 0
	items := []instanceRow{}
	for rows.Next() {
		var it instanceRow
		if err := rows.Scan(&it.PK, &it.InstanceID, &it.Constructor, &it.Payload,
			&it.TText, &it.TFact, &it.Locus, &it.Kind, &total); err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"total": total, "items": items})
}

// instance returns every bitemporal slice of one logical instance_id (the
// append-only history), its loci, and the REF rows that mention it.
func (s *server) instance(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		httpErr(w, http.StatusBadRequest, fmt.Errorf("id is required"))
		return
	}
	ctx := r.Context()

	slices := []instanceRow{}
	rows, err := s.pool.Query(ctx, `
		SELECT k.pk, k.instance_id::text, k.constructor::text, k.payload,
		       k.t_text::text, k.t_fact::text, s.locus, s.kind
		FROM kernel_instance k
		LEFT JOIN source_map s ON s.instance_pk = k.pk
		WHERE k.instance_id = $1::uuid
		ORDER BY lower(k.t_text), k.pk`, id)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	for rows.Next() {
		var it instanceRow
		if err := rows.Scan(&it.PK, &it.InstanceID, &it.Constructor, &it.Payload,
			&it.TText, &it.TFact, &it.Locus, &it.Kind); err != nil {
			rows.Close()
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		slices = append(slices, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}

	// REF rows that mention this instance id anywhere in their payload
	// (source or target side of the designation).
	mentions := []instanceRow{}
	rows, err = s.pool.Query(ctx, `
		SELECT k.pk, k.instance_id::text, k.constructor::text, k.payload,
		       k.t_text::text, k.t_fact::text, s.locus, s.kind
		FROM kernel_instance k
		LEFT JOIN source_map s ON s.instance_pk = k.pk
		WHERE k.constructor = 'REF' AND k.payload::text LIKE '%'||$1||'%'
		ORDER BY k.pk`, id)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	for rows.Next() {
		var it instanceRow
		if err := rows.Scan(&it.PK, &it.InstanceID, &it.Constructor, &it.Payload,
			&it.TText, &it.TFact, &it.Locus, &it.Kind); err != nil {
			rows.Close()
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		mentions = append(mentions, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"slices": slices, "refs": mentions})
}

// verdicts lists the persisted Ê verdicts (newest first) with their
// evaluation coordinates — a verdict without coordinates is meaningless (I6).
func (s *server) verdicts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT v.id, v.run, v.subject_instance::text, v.result::text,
		       v.conditional_on, v.eval_t_text, v.eval_t_fact, v.recorded_at,
		       m.state::text
		FROM verdict v
		LEFT JOIN e_machine m ON m.subject_instance = v.subject_instance
		ORDER BY v.id DESC LIMIT 200`)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type verdict struct {
		ID            int64     `json:"id"`
		Run           string    `json:"run"`
		Subject       string    `json:"subject"`
		Result        string    `json:"result"`
		ConditionalOn *string   `json:"conditional_on"`
		EvalTText     time.Time `json:"eval_t_text"`
		EvalTFact     time.Time `json:"eval_t_fact"`
		RecordedAt    time.Time `json:"recorded_at"`
		MachineState  *string   `json:"machine_state"`
	}
	items := []verdict{}
	for rows.Next() {
		var v verdict
		if err := rows.Scan(&v.ID, &v.Run, &v.Subject, &v.Result, &v.ConditionalOn,
			&v.EvalTText, &v.EvalTFact, &v.RecordedAt, &v.MachineState); err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"items": items})
}

// impact answers the Monday-Morning query at the read coordinates: which
// instances transitively REFERENCE the target IRI?
func (s *server) impact(w http.ResponseWriter, r *http.Request) {
	tt, tf, err := coords(r)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	iri := r.URL.Query().Get("iri")
	if iri == "" {
		httpErr(w, http.StatusBadRequest, fmt.Errorf("iri is required"))
		return
	}
	nodes, err := refgraph.Impact(r.Context(), s.pool, iri, tt, tf)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	if nodes == nil {
		nodes = []refgraph.Node{}
	}
	writeJSON(w, map[string]any{"target": iri, "tt": tt, "tf": tf, "nodes": nodes})
}

// iris lists the distinct REF designation targets valid at the coordinates —
// the vocabulary of the impact view.
func (s *server) iris(w http.ResponseWriter, r *http.Request) {
	tt, tf, err := coords(r)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT DISTINCT payload->>'target_iri'
		FROM kernel_instance_at($1, $2)
		WHERE constructor = 'REF' AND payload ? 'target_iri'
		ORDER BY 1`, tt, tf)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var iri *string
		if err := rows.Scan(&iri); err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		if iri != nil && *iri != "" {
			out = append(out, *iri)
		}
	}
	if err := rows.Err(); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"iris": out})
}
