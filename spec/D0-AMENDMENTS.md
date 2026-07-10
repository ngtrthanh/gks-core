# D0 Amendment Protocol & Amendment Log

**Purpose.** `Computable Governance.md` (D0 v1.1) is labelled *PERMANENTLY FROZEN*,
yet the program has already changed three of its constitutional objects (TIX, Force,
and — by the exit review — the status of Hypothesis 1). A frozen document with no
amendment mechanism cannot be honestly revised; the Phase-1 exit review (F2/F3)
correctly found that amending it by fiat invalidates every acceptance issued under
it. This file defines the *only* valid way to change D0, and records the changes.

> **Precedence.** D0 v1.1 (`Computable Governance.md`) is preserved verbatim as the
> historical baseline. Where this log's ratified amendments conflict with v1.1, the
> amended constitution (**D0 v1.2**) governs. v1.1 is never edited in place.

---

## 1. Amendment Protocol

An amendment is valid iff it records, in this file, all of:

1. **ID & date** — `Axx`, UTC date.
2. **Target** — the exact D0 v1.1 clause(s) changed.
3. **Change** — old text → new text (or "struck").
4. **Rationale** — why, with citation to evidence (a ruling, a review finding, or a
   falsification datum).
5. **Falsification linkage** — if the amendment reduces or removes anything D0
   declared *irreducible* or *frozen*, it MUST be cross-logged as a falsification
   datum against the affected Hypothesis (D0 §9), not silently absorbed.
6. **Authority** — the ratifying party. An amendment issued by the same authority
   that benefits from it (e.g. self-acceptance) is **not** valid for
   acceptance-grade claims; those additionally require the independent gate of
   D0 §10.2.
7. **Version bump** — increment D0 minor version; list the amendment here.

Amendments are append-only. Superseding an amendment requires a new amendment.

---

## 2. D0 v1.2 — Ratified Amendment Set (2026-07-10)

Basis: Agent-0 rulings 2026-07-09 (engineering) + Phase-1 exit-review findings.
Authority: internal (Agent-0). **These amendments are constitutionally valid for
*specification* purposes but do NOT confer Phase-1 acceptance**, which remains gated
on D0 §10.2 independent verification (see `PROGRESS.md`, `AGENT-0-DECISIONS.md`).

### A01 — Basis reduced to six constructors  ·  *(with falsification linkage)*
- **Target:** D0 §3.1, §4.1; Hypothesis 1.
- **Change:** `B = {NRM,CLS,PWR,GRD,REF,VAL,TIX}` (7) → `B = {NRM,CLS,PWR,GRD,REF,VAL}` (6). TIX is reclassified as the bitemporal index τ(x) (metadata), realized columnar as `t_text`/`t_fact`.
- **Rationale:** Agent-0 Ruling 1 (an address over an object is not another object class).
- **Falsification linkage:** D0 Hypothesis 1 ("a fixed basis of **seven irreducible** constructors") is thereby **partially falsified** — a constructor declared irreducible was reduced to metadata. Logged as falsification datum **FD-1** (D0 §9). Whether *six* is minimal is the open C1 question, which now trends against a fixed minimal basis.

### A02 — NRM Force deprecated; obligation-only kernel
- **Target:** D0 §4.1 (NRM signature × Force); D1.1 Axiom 1.2.
- **Change:** the `Force ∈ {O,P,F}` component of NRM is struck from constitutional semantics. Permission = absence of an NRM; prohibition = GRD. Stored `force` values are legacy-readable only.
- **Rationale:** Agent-0 Ruling 2 (minimality — one executable branch, two dormant).
- **Falsification linkage:** a second reduction of a frozen signature; logged as **FD-2**. Reinforces the caution on Hypothesis 1/2.

### A03 — Ê lifecycle alphabet corrected  *(fixes review M4)*
- **Target:** D1.1 Axiom 1.3 — Σ.
- **Change:** Σ (v1.1: `{inforce, suspended, violated, discharged, terminated}`) → the **implemented** alphabet `{proposed, in-force, suspended, violated, discharged, extinguished}` (DB enum `e_state`, D8). "terminated" → "extinguished"; "proposed" added as the pre-activation state.
- **Rationale:** the store and D8 use this alphabet; v1.1 was neither implemented nor documented as amended. D0 §9.1.4 makes an un-modeled Ê state a falsification trigger — this amendment models it explicitly rather than leaving the gap.

### A04 — Verdict vocabulary specified  *(fixes review M5)*
- **Target:** D1.4 §4 (verdict as θ-projection, previously unspecified as a vocabulary).
- **Change:** the verdict vocabulary is fixed to `{compliant, violated, conditional, inapplicable, defeated}` with these **defaults**, which any conforming implementation MUST reproduce: (a) an unguarded norm with no activating guard is **in force by default**; (b) `INACTIVE` (activating guard present but unmet) → `inapplicable`; (c) `DEFEATED` (higher-priority guard suppressed it) → `defeated` (distinct from `inapplicable`, preserving provenance); (d) open-texture boundary → `conditional`.
- **Rationale:** Agent-0 Ruling 3 + review M5 (the default was documented only in code comments; an independent implementation would fork here first). This is the **verdict contract** prerequisite for the confirmation experiment.

### A05 — I9 is a total single-valued map, not a bijection  *(fixes review finding-8)*
- **Target:** D0 §6-I9 wording ("bijection").
- **Change:** I9 = every kernel row has exactly one source_map row (total, single-valued, `UNIQUE(instance_pk)`). It is **not** a bijection: many instances may map to one source locus (D8 Run 2 maps four instances to one clause).
- **Rationale:** the implementation and benchmark are many-to-one; the v1.1 "bijection" claim was never true of the design.

---

## 3. Falsification Data Log (cross-referenced from D0 §9)

| ID | Datum | Hypothesis affected | Status |
| --- | --- | --- | --- |
| FD-1 | TIX reduced from constructor to metadata (7→6) | Hyp-1 (seven irreducible constructors) | **partial falsification** |
| FD-2 | Force reduced (obligation-only; prohibition→GRD) | Hyp-1 / Hyp-2 (minimality) | **partial falsification** |

These are recorded, not absorbed. Hypothesis 1 *as originally frozen* is refuted;
the live claim is the weaker "six constructors suffice for the tested corpora",
whose minimality (C1) is open and, on current evidence, trending false.
