# Related Work & Positioning

The Phase-1 exit review noted that the repository contained **no related-work
positioning** — a fair and serious gap for a scientific artifact. This document
positions ⟨B, T⟩ against the relevant prior art and states plainly what is, and is
not, novel. It is deliberately candid: most of the *constructors* map onto
well-established notions; what is plausibly new is the *integration* and the
*falsificationist protocol*, both of which are engineering-methodological rather
than mathematical.

## 1. The constructor basis is a re-packaging of standard notions

| Constructor | Established notion | Representative prior art |
| --- | --- | --- |
| NRM (directed normative position) | Hohfeldian duty/claim; deontic obligation | Hohfeld (1913); von Wright deontic logic; LegalRuleML deontic norms |
| CLS (constitutive classification) | "counts-as" / constitutive rules | Searle (1995); Jones & Sergot (1996) "counts-as"; InstAL institutional facts |
| PWR (authority operator) | legal/normative power (Hohfeldian power) | Hohfeld; Governatori & Rotolo on institutional power |
| GRD (defeasible override) | defeasible rule + superiority relation | Nute defeasible logic; LegalRuleML `Override`; Regorous/SPINdle |
| REF (typed cross-reference) | citation / amendment edges | Akoma Ntoso references; LKIF; legal citation graphs |
| VAL (quantitative constraint) | arithmetic/threshold constraint | SBVR; ordinary constraint languages; **arguably eliminable** (see §4) |

None of these six is individually novel. The claim that they form a *closed,
minimal spanning basis* (Hypotheses 1–2) is the scientific content — and it is
exactly the claim the exit review found unconfirmed (and, for minimality, trending
false after the TIX and Force reductions).

## 2. Comparable systems the project must be measured against

- **LegalRuleML / LKIF / SBVR** — standardized rule-markup with deontic operators,
  defeasibility, and override. Broader vocabulary than B; no sub-Turing guarantee,
  no bitemporal execution model, no falsificationist protocol.
- **Catala** (Merigoux, Chataing, Protzenko, 2021) — a domain-specific language for
  law with *peer-reviewed, machine-checked default-logic semantics* and real
  deployments (French tax). **This is the most important comparison and the highest
  bar:** Catala already delivers what gks-core's mechanization only claims —
  formally specified, mechanized semantics. gks-core's Lean development is, by the
  corrected D1.5 ledger, not yet at that level.
- **Regorous / SPINdle / defeasible deontic engines** (Governatori et al.) — compliance
  checking by defeasible deontic logic; directly comparable to the GRD/resolver core.
- **Event Calculus / InstAL / ASP** — normative state over time, institutional facts,
  well-understood stratified/stable-model semantics; a natural alternative to the
  bespoke Ê layer and the stratified-reflection (I7) idea.
- **Akoma Ntoso + SQL:2011 bitemporal** — standardized bitemporal legal-text and
  temporal-DB models; the K̂ store's ⟨t_text, t_fact⟩ indexing is a (reasonable)
  re-derivation of these, not a new idea.
- **Datalog / stratified negation** — the "stratified reflection" (I7) and
  Registry-Law (Θ(1) basis) intuitions have direct analogues here.

## 3. What is plausibly novel

1. **The specific integration**: a single small kernel combining defeasible deontic
   resolution + constitutive classification + bitemporal execution + a
   content-addressed canonical form (CNF) with cryptographic sealing.
2. **The falsificationist engineering protocol**: κ agreement floors, byte-deterministic
   sealed exports, an explicit falsification ledger, and a constructor-closure screen.
   This is a *methodology* contribution — a way to run a governance-DSL project as a
   refutable program — more than a mathematical result.

Neither of these has been *validated* yet (that requires the independent replication
of D0 §10.2). They are candidate contributions, not established ones.

## 4. Open comparative questions raised by the review

- **Is VAL eliminable?** The reviewer observes `VAL(φ,u,⊕,t)` reduces to a GRD
  condition using T's comparators over registry lookups (the campaign's own
  "wage ≥ minimum" unit shows the overlap). If so, the basis is ≤ 5 and Hypothesis 2
  is further undermined. This is a concrete, in-tree experiment (WS-G/C1) that has
  not been run.
- **What does ⟨B, T⟩ add over Catala + a bitemporal store?** This is the sharpest
  positioning question and is currently unanswered.

## 5. Bibliographic pointers

Hohfeld 1913 (Yale L.J.); Searle 1995 *Construction of Social Reality*; Jones &
Sergot 1996 (J. IGPL); von Wright 1951 (Mind); Nute 1994 (defeasible logic);
Governatori et al. (Regorous, defeasible deontic); Merigoux/Chataing/Protzenko 2021
(Catala, ICFP); OASIS LegalRuleML; OASIS Akoma Ntoso; Boulanger/LKIF; SQL:2011
temporal; Kowalski & Sergot 1986 (event calculus); Datalog stratified negation.

*(Full citations to be added with the Phase-2 write-up; this file establishes the
positioning the review found missing.)*
