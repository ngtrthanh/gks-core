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

Scaffolded. The CNF exporter and κ/agreement harness are not yet implemented;
they will be added as `compiler/cmd/export_cnf` and a comparison driver here.
