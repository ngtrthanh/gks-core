# Track D — Clause-Level Extraction Depth Study

Reads stored docx source text only (no re-ingestion). Two independent
classifiers (A: definition-first/"có quyền"; B: prohibition-first/any
"quyền") are run at paragraph and clause granularity.

## Extraction yield

- paragraphs (stored units): **392**
- clauses after khoản/điểm segmentation: **392**  (×1.00)
- multi-modal paragraphs (clauses span ≥2 constructors): **0** (0.0%)

## Inter-classifier agreement (Fleiss' κ)

| granularity | shared units | κ(A,B) |
|---|---|---|
| paragraph | 392 | 0.8380 |
| clause | 392 | 0.8380 |

- Δκ (clause − paragraph): **+0.0000** — no material change
- Track B baseline (stored-vs-independent, paragraph): κ=0.7877

## Findings

1. **The corpus is already clause-atomic.** Every stored unit begins with
   its own khoản/điểm marker ("1.", "a)", …); `ingest_docx` already extracts
   one instance per enumerated point. Structural segmentation therefore
   recovers **0** further units — extraction depth is already maximal at
   the structural level, and no stored unit conflates ≥2 modalities.
2. **The Track B gap was in the *stored* assignments, not the text.** Two
   fresh independent classifiers on the same units agree at κ=0.8380, above
   the 0.7877 stored-vs-independent baseline: much of Track B's disagreement
   traces to the older ingester rules frozen in the store, not to genuine
   textual ambiguity.
3. **Improvement lever = cue modelling, not splitting.** Remaining
   disagreement is *semantic* (ambiguous cues inside atomic clauses), so the
   next gain is richer modality detection, not finer segmentation.
