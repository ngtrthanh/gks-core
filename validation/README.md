# validation/ — Reproducibility & Canonical Normal Form (CNF) Export

This directory hosts the reproducibility harnesses (D0 §8.1–8.2): deterministic
**Canonical Normal Form (CNF)** export of verdicts and inter-compiler agreement
metrics (Fleiss' κ ≥ 0.70, verdict agreement ≥ 0.90).

## Canonical Normal Form

CNF is the deterministic, byte-stable serialization of a verdict / kernel state
used for exact syntactic diffing across independent compiler runs (Dimension 1:
Reproducibility). Two conforming compilers must emit identical CNF for the same
`⟨corpus, event-trace, ⟨t_text, t_fact⟩⟩` input.

Requirements for the export:
- reads go through `kernel_instance_at(t_text, t_fact)` (temporal discipline, G4);
- keys sorted; no map-iteration nondeterminism; UTC timestamps in RFC3339;
- append-only source (Invariant I2) guarantees stable inputs.

## Status

The CNF exporter exists: `compiler/cmd/cnf_export` (α-renames store UUIDs to
content-ordered sequential ids so exports are comparable across independent
compilers; byte-stable digest verified across runs). Sealing/verification:
`compiler/cmd/seal_export`, `compiler/cmd/verify_seal`.

Still missing here (WP-8): the κ / verdict-agreement comparison driver over N
independent CNF exports, and the constitutional floor assertions (κ ≥ 0.70,
VA ≥ 0.90). Note the exports also depend on ingestion-time `t_text`/`t_fact`
coordinates: independent compilers must derive coordinates from the corpus
text, not wall-clock ingest time, before cross-compiler byte-comparison is
meaningful.
