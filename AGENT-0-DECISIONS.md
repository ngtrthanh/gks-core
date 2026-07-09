# AGENT-0 DECISION REQUEST — three constitutional rulings — **RESOLVED**

**From:** Kiro (Engineering) · **To:** Agent 0 (constitutional authority)
**Date filed:** 2026-07-09 UTC · **Ruled:** 2026-07-09 UTC · **Implemented:** 2026-07-09 UTC

> ## RULING RECEIVED — implemented in full
>
> | Decision | Ruling | Implementation |
> | --- | --- | --- |
> | 1 — TIX | **Option A**: TIX is the temporal index, not a constructor; |B|=6 | D1.1 Def 3.1 → 6 constructors + τ metadata; README I3; enum already 6 (migration 0002); unused `TIXPayload` removed. |
> | 2 — NRM Force | **Option B**: obligation-only; permission = absence of NRM; prohibition = GRD | `NRMPayload.Force` marked DEPRECATED (legacy read only); D1.1 NRM relation drops ×Force; machine logic unchanged. |
> | 3 — Verdict | **Option B**: `DEFEATED` is a first-class verdict distinct from `INAPPLICABLE` | `e_verdict` enum + `defeated` (migration 0005); resolver DEFEATED → `defeated`; `machine_test` distinguishes it. |
>
> All `AGENT-0-DECISION-*` code markers have been removed. `go build/vet/test`
> green; live DB enum updated. The record below is retained for provenance.

---

The three items were flagged during the G-review (`handoff.md` §6, `handoff2.md`
§AGENT-0, `handoff3.md` §5). Each is stated as: **tension → question → options
(with trade-offs) → the behavior at time of filing → advisory recommendation.**
The ruling adopted the recommendation on all three.

---

## Decision 1 — Is `TIX` a constructor, or a columnar coordinate? (review G6)

**Tension.** `spec/D1.1` defines the basis with **seven** summands
`B = {NRM,CLS,PWR,GRD,REF,VAL,TIX}` and `Inst ≜ Σ_{c∈B} c` (Def. 3.1–3.2), and
`README` §I3 lists TIX in B. But `db/migrations/0002_hardening.sql` **dropped
`TIX` from the `kernel_constructor` enum** — bitemporality is realized *columnar*
as `t_text`/`t_fact` (the map `τ(x) ∈ TIX` of I6), not as an instantiable row.
`handoff.md` §6 had said "leave the enum as-is, pending"; the migration and that
note now **conflict**.

**Question.** Is TIX (a) a first-class instantiable constructor (|B| = 7), or
(b) the bitemporal *index* τ every instance carries (|B| = 6, bitemporality
structural)?

**Options.**
- **(A) Ratify columnar (|B| = 6).** Amend `D1.1` so TIX is the index set of τ
  (Def. 3.2), not a summand of `Inst`. Matches the store, the evaluator, and
  every test (Registry Law, store-wide screen all assume 6). *Cost:* a spec edit;
  `README` I3 wording updated.
- **(B) Restore TIX as the 7th constructor.** Re-add the enum label; permit TIX
  rows. *Cost:* re-opens append-only/RBAC surface for a constructor with no
  operational role; contradicts the columnar I6 realization; would break the
  "|B| = 6" tests.
- **(C) Dual reading.** Keep TIX in the spec algebra, columnar in the impl, and
  document the isomorphism `Inst×TIX ≅ (6-constructor row)+(t_text,t_fact)`.

**Current provisional behavior.** Enum has 6 labels (migration 0002); all code
and tests treat |B| = 6.

**Advisory recommendation:** **(A)**. The columnar realization is already the
one true implementation of I6, and confirmation criteria (§9.2, minimality C1)
are cleaner with a 6-constructor basis. This needs a `D1.1` amendment, which is
constitutional — hence the request.

---

## Decision 2 — `NRMPayload.Force` O|P|F trichotomy (review G5)

**Tension.** `NRM` carries a `Force` field intended as an
Obligation | Permission | proHibition trichotomy. Only the **S-Violate**
transition branches on it, and only for positive obligations
(`internal/machine/machine.go:455`, `// AGENT-0-DECISION-2`). No permission/
prohibition transition logic exists. Meanwhile prohibition cues in the corpus
("nghiêm cấm", "không được") are currently classified as **GRD**, not NRM-with-
Force-F.

**Question.** Is deontic modality a `Force` attribute on NRM, or is prohibition
realized by GRD and permission by the *absence* of obligation — making `Force`
partly redundant? If it is an attribute, what are the P/F transition semantics?

**Options.**
- **(A) Keep the trichotomy, add P/F semantics.** Permission ⇒ no violation is
  ever raised; prohibition ⇒ violation on the prohibited act. *Cost:* new machine
  branches; must reconcile with GRD-realized prohibition (double representation).
- **(B) Collapse Force to O only.** Prohibition ⇒ GRD; permission ⇒ no NRM.
  Deprecate the P/F values. *Cost:* a spec/type note; simplest, matches the
  ingester's current GRD-for-prohibition behavior.
- **(C) Status quo (dormant field).** Keep the field, no P/F logic, until a
  corpus demonstrably needs it.

**Current provisional behavior.** (C) — `Force` exists; only O is acted on.

**Advisory recommendation:** **(B) or (C)**. The corpus evidence (prohibition →
GRD) suggests Force=F is redundant with GRD; leaning (B) if Agent 0 wants a clean
type, else (C) to defer. Avoid (A) unless a corpus forces distinct P/F verdicts —
which would itself be a notable finding.

---

## Decision 3 — Resolver-state → verdict mapping (review G3)

**Tension.** The defeasible resolver yields π₃-internal states
`IN_FORCE / DEFEATED / INACTIVE`. The Ê machine maps them provisionally
(`internal/machine/machine.go:370,442`, `// AGENT-0-DECISION-3`):
`IN_FORCE → compliant`, `DEFEATED → guard-suppressed (machine stays in-force,
verdict inapplicable)`, `INACTIVE → inapplicable`.

**Question.** Is collapsing `DEFEATED` and `INACTIVE` both to *inapplicable*
constitutionally correct, or should `DEFEATED` (an obligation actively
suppressed by a guard) be a distinct, auditable verdict — or `compliant`?

**Options.**
- **(A) Ratify the current 3→{compliant, inapplicable} mapping.** *Cost:* none;
  simplest.
- **(B) Surface `DEFEATED` as its own verdict value** (defeated ≠ inapplicable),
  for audit transparency of *why* no obligation fired. *Cost:* widen the verdict
  vocabulary; downstream consumers updated.
- **(C) `DEFEATED → compliant`** (the guard lawfully suppressed the obligation,
  so there is no violation). *Cost:* conflates "suppressed" with "satisfied".

**Current provisional behavior.** (A), documented at the cited lines and in
`machine_test.go:132`.

**Advisory recommendation:** **(A) now, (B) if auditability becomes a
requirement.** Distinguishing DEFEATED is valuable for explainability but is a
verdict-vocabulary change (constitutional).

---

## What engineering will do meanwhile

- Hold the provisional behavior above; keep `go build/vet/test ./...` green.
- Not amend `internal/kernel`, the enum, the resolver mapping, or `D1.1` until a
  ruling lands.
- On a ruling, implement it as a numbered migration + spec edit + tests, and
  remove the corresponding `AGENT-0-DECISION-*` marker.


---

## FINAL CONSTITUTIONAL VERDICT (Agent-0, 2026-07-10)

> **Phase 1 is accepted.**
>
> **The remaining work is no longer implementation work. It is scientific
> confirmation.**

**Effect.** The implementation program (Phases 0–3 + the four-dimensional rigor
build-out) is constitutionally **accepted**. `gks-core` stands as the accepted
reference realization of the kernel ⟨B, T⟩: six constructors, nine invariants
I1–I9 (each tested and/or CI-proved), all eight D1.5 theorem obligations T1–T8
mechanized in Lean, the Ê execution layer, deterministic CNF + seal, temporal
reads, registry snapshots, exact-rational VAL, the validation/interop/
falsification harnesses, and the continuous-ingestion control plane.

**What remains is science, not code.** Per D0 §9.2, *confirmation* is earned by
- **independent compilation convergence** at the verdict stratum — a genuinely,
  organizationally *separate* second implementation converging on the CNF/verdict
  contract; and
- resolution of the **C1 (minimality)** question — whether ⟨B, T⟩ is the smallest
  spanning basis.

These are empirical/research obligations. The falsification campaign (D0 §9.1)
stands permanently open; no criterion has triggered. Engineering does not "close"
confirmation by writing more code — it is earned by independent replication and
scientific result. Kernel status remains *provisional-but-accepted* until then.
