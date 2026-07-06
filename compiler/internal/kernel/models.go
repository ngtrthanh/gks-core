// Package kernel defines the Go domain model for the Knowledge-Theory (K-hat)
// IR: the seven closed kernel constructors and the bitemporal, JSONB-backed
// storage row they persist to (see db/schema.sql and spec/D1.1).
//
// The custom types below implement database/sql's Scanner/Valuer so that
// PostgreSQL `jsonb`, `uuid`, and `tstzrange` columns round-trip with only the
// standard library — no external driver-specific types required.
package kernel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Constructor is the closed basis B of seven irreducible constructors
// (Invariant I3: kernel closure — no eighth constructor).
type Constructor string

const (
	NRM Constructor = "NRM" // directed normative position
	CLS Constructor = "CLS" // constitutive classification
	PWR Constructor = "PWR" // authority operator
	GRD Constructor = "GRD" // defeasible composition / priority
	REF Constructor = "REF" // typed cross-corpus designation
	VAL Constructor = "VAL" // governed quantitative binding
	// NOTE: TIX is intentionally absent. Bitemporality is realized columnar
	// (t_text/t_fact, Invariant I6), not as an instantiable constructor (G6).
)

// Valid reports whether c is one of the seven closed constructors.
func (c Constructor) Valid() bool {
	switch c {
	case NRM, CLS, PWR, GRD, REF, VAL:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// UUID — 128-bit logical instance identity, stdlib-only.
// ---------------------------------------------------------------------------

// UUID mirrors PostgreSQL's `uuid` type.
type UUID [16]byte

// ParseUUID parses the canonical 8-4-4-4-12 hyphenated form.
func ParseUUID(s string) (UUID, error) {
	var u UUID
	clean := strings.ReplaceAll(strings.Trim(s, "{}"), "-", "")
	if len(clean) != 32 {
		return u, fmt.Errorf("kernel: invalid UUID %q", s)
	}
	if _, err := hex.Decode(u[:], []byte(clean)); err != nil {
		return u, fmt.Errorf("kernel: invalid UUID %q: %w", s, err)
	}
	return u, nil
}

// String renders the canonical hyphenated representation.
func (u UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// Scan implements sql.Scanner for `uuid` (accepts 16-byte binary or text).
func (u *UUID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		if len(v) == 16 {
			copy(u[:], v)
			return nil
		}
		p, err := ParseUUID(string(v))
		if err != nil {
			return err
		}
		*u = p
		return nil
	case string:
		p, err := ParseUUID(v)
		if err != nil {
			return err
		}
		*u = p
		return nil
	default:
		return fmt.Errorf("kernel: cannot scan %T into UUID", src)
	}
}

// Value implements driver.Valuer.
func (u UUID) Value() (driver.Value, error) { return u.String(), nil }

// MarshalJSON / UnmarshalJSON render UUID as a JSON string.
func (u UUID) MarshalJSON() ([]byte, error) { return json.Marshal(u.String()) }

func (u *UUID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	p, err := ParseUUID(s)
	if err != nil {
		return err
	}
	*u = p
	return nil
}

// ---------------------------------------------------------------------------
// JSONB — Layer S semantic-algebra AST payload.
// ---------------------------------------------------------------------------

// JSONB is a raw `jsonb` document carried opaquely at the storage boundary.
type JSONB json.RawMessage

// Scan implements sql.Scanner for `jsonb`.
func (j *JSONB) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = append((*j)[:0], v...)
		return nil
	default:
		return fmt.Errorf("kernel: cannot scan %T into JSONB", src)
	}
}

// Value implements driver.Valuer.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON / UnmarshalJSON make JSONB transparent to encoding/json.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSONB) UnmarshalJSON(b []byte) error {
	*j = append((*j)[:0], b...)
	return nil
}

// ---------------------------------------------------------------------------
// TSTZRange — one dimension of the bitemporal TIX index.
// ---------------------------------------------------------------------------

// rangeTimeFormat is accepted by PostgreSQL as a timestamptz literal.
const rangeTimeFormat = time.RFC3339Nano

// tstzInLayouts are the formats PostgreSQL may emit for range bounds.
var tstzInLayouts = []string{
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999Z07",
	"2006-01-02 15:04:05Z07",
	time.RFC3339Nano,
	time.RFC3339,
}

// TSTZRange models PostgreSQL `tstzrange` (half-open by convention: [lower,upper)).
type TSTZRange struct {
	Lower    time.Time
	Upper    time.Time
	LowerInc bool // '[' lower bound inclusive
	UpperInc bool // ']' upper bound inclusive
	LowerInf bool // lower bound unbounded / -infinity
	UpperInf bool // upper bound unbounded / +infinity
	Empty    bool // the empty range
}

// NewClosedOpen builds the common [lower, upper) range.
func NewClosedOpen(lower, upper time.Time) TSTZRange {
	return TSTZRange{Lower: lower, Upper: upper, LowerInc: true}
}

// Since builds [lower, infinity): valid from lower onward.
func Since(lower time.Time) TSTZRange {
	return TSTZRange{Lower: lower, LowerInc: true, UpperInf: true}
}

// Contains reports whether instant t falls within the range.
func (r TSTZRange) Contains(t time.Time) bool {
	if r.Empty {
		return false
	}
	if !r.LowerInf {
		if r.LowerInc {
			if t.Before(r.Lower) {
				return false
			}
		} else if !t.After(r.Lower) {
			return false
		}
	}
	if !r.UpperInf {
		if r.UpperInc {
			if t.After(r.Upper) {
				return false
			}
		} else if !t.Before(r.Upper) {
			return false
		}
	}
	return true
}

func parseBound(s string) (t time.Time, inf bool, err error) {
	s = strings.TrimSpace(strings.Trim(s, `"`))
	if s == "" || s == "infinity" || s == "-infinity" {
		return time.Time{}, true, nil
	}
	for _, layout := range tstzInLayouts {
		if parsed, e := time.Parse(layout, s); e == nil {
			return parsed, false, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("kernel: unparsable tstz bound %q", s)
}

// Scan implements sql.Scanner for `tstzrange`.
func (r *TSTZRange) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("kernel: cannot scan %T into TSTZRange", src)
	}

	s = strings.TrimSpace(s)
	if s == "" || s == "empty" {
		*r = TSTZRange{Empty: true}
		return nil
	}
	if len(s) < 2 {
		return fmt.Errorf("kernel: invalid tstzrange %q", s)
	}

	var out TSTZRange
	out.LowerInc = s[0] == '['
	out.UpperInc = s[len(s)-1] == ']'
	inner := s[1 : len(s)-1]

	// Split on the top-level comma (bounds may be double-quoted).
	comma, inQuote := -1, false
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				comma = i
			}
		}
		if comma >= 0 {
			break
		}
	}
	if comma < 0 {
		return fmt.Errorf("kernel: malformed tstzrange %q", s)
	}

	lo, loInf, err := parseBound(inner[:comma])
	if err != nil {
		return err
	}
	hi, hiInf, err := parseBound(inner[comma+1:])
	if err != nil {
		return err
	}
	out.Lower, out.LowerInf = lo, loInf
	out.Upper, out.UpperInf = hi, hiInf
	*r = out
	return nil
}

// Value implements driver.Valuer, emitting a PostgreSQL range literal.
func (r TSTZRange) Value() (driver.Value, error) {
	if r.Empty {
		return "empty", nil
	}
	lb, ub := "[", ")"
	if !r.LowerInc {
		lb = "("
	}
	if r.UpperInc {
		ub = "]"
	}
	lo := "-infinity"
	if !r.LowerInf {
		lo = `"` + r.Lower.Format(rangeTimeFormat) + `"`
	}
	hi := "infinity"
	if !r.UpperInf {
		hi = `"` + r.Upper.Format(rangeTimeFormat) + `"`
	}
	return fmt.Sprintf("%s%s,%s%s", lb, lo, hi, ub), nil
}

// ---------------------------------------------------------------------------
// KernelInstance — one append-only bitemporal row of a kernel constructor.
// ---------------------------------------------------------------------------

// KernelInstance maps 1:1 to a row of the `kernel_instance` table.
type KernelInstance struct {
	PK          int64       `json:"pk"          db:"pk"`
	InstanceID  UUID        `json:"instance_id" db:"instance_id"`
	Constructor Constructor `json:"constructor" db:"constructor"`
	Payload     JSONB       `json:"payload"     db:"payload"`
	TText       TSTZRange   `json:"t_text"      db:"t_text"`
	TFact       TSTZRange   `json:"t_fact"      db:"t_fact"`
	RecordedAt  time.Time   `json:"recorded_at" db:"recorded_at"`
}

// DecodePayload deserializes an instance's JSONB payload into a typed
// constructor struct (e.g. NRMPayload). Modern generic decode helper.
func DecodePayload[T any](ki KernelInstance) (T, error) {
	var out T
	if len(ki.Payload) == 0 {
		return out, fmt.Errorf("kernel: instance %s has empty payload", ki.InstanceID)
	}
	if err := json.Unmarshal(ki.Payload, &out); err != nil {
		return out, fmt.Errorf("kernel: decode %s payload: %w", ki.Constructor, err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Constructor payload schemas (AST projections stored in JSONB).
// ---------------------------------------------------------------------------

// NRMPayload — directed normative position.
type NRMPayload struct {
	Bearer       string `json:"bearer"`
	Counterparty string `json:"counterparty"`
	Act          string `json:"act"`
	Sign         string `json:"sign"`  // "+" | "-"
	Force        string `json:"force"` // "O" | "P" | "F"
	// Target is the object of the act (e.g. "output:c2"); optional.
	Target string `json:"target,omitempty"`
	// Qualifier is an open-texture constraint on the act (e.g. Boundary "OT-1"
	// for "appropriate"); optional.
	Qualifier *Expr `json:"qualifier,omitempty"`
}

// CLSPayload — constitutive classification. Condition is a T-expression AST.
type CLSPayload struct {
	Entity     string `json:"entity"`
	ClassToken string `json:"class_token"`
	Condition  *Expr  `json:"condition"`
}

// GRDPayload — defeasible guard (spec D1.1/D1.4). When Condition holds, the
// instances in Body are activated; when this guard has higher Priority than a
// conflicting one, the instances in Defeats are suspended/overridden.
type GRDPayload struct {
	Condition *Expr    `json:"condition"`
	Body      []string `json:"body,omitempty"` // instance_ids activated
	Priority  int      `json:"priority"`
	Defeats   []string `json:"defeats,omitempty"` // instance_ids defeated
}

// PWRPayload — authority operator; OperandSchema constrains admissible events,
// and Operand is the kernel instance produced when the power is exercised
// (e.g. a concession GRD that suspends a nonconformity-control obligation).
type PWRPayload struct {
	Holder        string      `json:"holder"`
	OperandSchema *Expr       `json:"operand_schema,omitempty"`
	Operand       *GRDPayload `json:"operand,omitempty"`
	Effect        string      `json:"effect"`
	Event         string      `json:"event"`
}

// VALPayload — governed quantitative binding: the measured quantity satisfies
// `Measure <Comparator> Target`. Both Measure and Target are T-expressions
// evaluated to EXACT rationals (Measure is typically a Ratio; Target may be
// arithmetic over a registry Lookup, e.g. 0.95 × reg(threshold) — the D8 Run 6
// "normative reference embedded in a formula" pattern). No float64 anywhere:
// verdicts over VAL are bit-for-bit reproducible (WP-7, I8).
type VALPayload struct {
	Function   string `json:"function"`   // human label, e.g. "on_time_logging_rate"
	Unit       string `json:"unit"`       // e.g. "ratio", "%"
	Comparator string `json:"comparator"` // one of the T CmpOp tokens
	Measure    *Expr  `json:"measure"`    // the evaluated quantity (rational)
	Target     *Expr  `json:"target"`     // the target quantity (rational)
}

// AsExpr projects a VAL binding to the boolean T-expression `Measure cmp
// Target`, so it evaluates through the ordinary big-step evaluator.
func (v VALPayload) AsExpr() *Expr {
	return &Expr{Op: OpCmp, Cmp: v.Comparator, Args: []*Expr{v.Measure, v.Target}}
}

// REFPayload — typed cross-corpus designation.
type REFPayload struct {
	Source    string `json:"source"`
	TargetIRI string `json:"target_iri"`
	Mode      string `json:"mode"` // cite | amend | derogate | define
}

// TIXPayload — an explicit bitemporal coordinate carried in a payload.
type TIXPayload struct {
	TText TSTZRange `json:"t_text"`
	TFact TSTZRange `json:"t_fact"`
}

// Compile-time interface assertions.
var (
	_ sql.Scanner   = (*UUID)(nil)
	_ driver.Valuer = UUID{}
	_ sql.Scanner   = (*JSONB)(nil)
	_ driver.Valuer = JSONB(nil)
	_ sql.Scanner   = (*TSTZRange)(nil)
	_ driver.Valuer = TSTZRange{}
)
