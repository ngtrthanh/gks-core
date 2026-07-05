# Governance Computing — Formal Specification and Compiler Repository

This repository is the **Formal Specification and Compiler Repository for Governance
Computing**. It hosts the machine-checkable specification, mechanized proofs, reference
compiler, and validation harnesses for the kernel $\langle B, T \rangle$ frozen in
`D0v5.md` (Constitutional Specification, D0 v1.1).

Phase 0 (kernel discovery, narrative) is closed. This tree is the artifact of **Phase 1 —
The Formal Transition**: every normative claim in D0 is restated here as an axiom,
grammar, inference rule, or proof obligation.

## Repository Layout

| Path | Contents |
| --- | --- |
| `spec/` | Formal mathematical specifications (D1 series): objects, type system, grammar, operational semantics, proof obligations. |
| `mechanization/` | Interactive theorem-prover developments (Lean 4 / mathlib4) discharging the obligations in `spec/D1.5`. |
| `compiler/` | MVP intermediate-representation (IR) database and the four-pass compilation pipeline $\pi = \pi_4 \circ \pi_3 \circ \pi_2 \circ \pi_1$. |
| `validation/` | Reproducibility harnesses and inter-compiler agreement metrics (Fleiss' $\kappa$, verdict agreement). |

## Specification Index (`spec/`)

| Document | Subject | Key invariants |
| --- | --- | --- |
| `D1.1-Mathematical-Objects.md` | Sets, relations, tuple definitions of the 7 constructors | I2, I6 |
| `D1.2-Type-System.md` | Base sorts, roles, typing judgments $\Gamma \vdash e : \tau$ | I3 |
| `D1.3-Grammar.md` | EBNF for the semantic algebra $T$; sub-Turing constraints | I1, I7 |
| `D1.4-Operational-Semantics.md` | Big-/small-step rules for layer $\hat{E}$ | I2, I8 |
| `D1.5-Proof-Obligations.md` | Theorem/conjecture ledger for Lean 4 | I1–I8, Hyp. 2 |

## Foundational Commitments

- **Kernel closure (I3).** The basis $B = \{\text{NRM}, \text{CLS}, \text{PWR}, \text{GRD}, \text{REF}, \text{VAL}, \text{TIX}\}$ and the language $T$ are closed. No eighth constructor.
- **Read-only algebra (I1).** $T$ has empty write-effect by construction (see `spec/D1.3` §4).
- **Single writer (I2).** Only $\hat{E}$ extends $\hat{K}$, append-only.
- **Determinism (I8).** All compilation passes are total deterministic functions.

## Build

Formal artifacts are built and checked via the root `Makefile`:

```sh
make spec            # render/validate the specification documents
make verify          # run Lean 4 mechanization (placeholder)
make test-compiler   # run the reference-compiler test suite (placeholder)
```

## Status

`spec/` — DRAFT. `mechanization/`, `compiler/`, `validation/` — scaffolded, empty.
All proof obligations in `spec/D1.5` are currently **conjectures**.
