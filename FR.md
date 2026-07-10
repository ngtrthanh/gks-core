# FINAL REVIEW PROMPT — GKS Core Constitutional & Scientific Audit (Phase-1 Exit Review)

Attention Fable

You are acting as the **final external reviewer** before this project is frozen as the canonical **Phase-1 reference implementation**.

Your role is **not** to help improve the project.

Your role is to determine whether the project deserves to exist as a scientific contribution.

Assume the authors are competent and honest, but assume nothing else.

Treat every statement as guilty until proven by evidence contained in the repository.

Repository: https://github.com/ngtrthanh/gks-core

---

# Review Philosophy

Conduct the review as if preparing a confidential report for a research council deciding whether Phase-1 should be accepted.

Do NOT be encouraging.

Do NOT optimize for politeness.

Do NOT generate generic code-review comments.

Assume publication-quality scrutiny.

Reject unsupported claims.

Demand evidence for every assertion.

Prefer identifying one fatal flaw over one hundred cosmetic issues.

---

# Review Objective

Determine whether the repository actually demonstrates a reproducible computational governance kernel, or merely implements an interesting software framework.

The burden of proof lies entirely on the repository.

---

# Review Order

Proceed strictly in this order.

Do not skip steps.

---

## Part I — Repository Integrity

Verify:

* repository organization
* documentation consistency
* tags
* release history
* reproducibility
* build process
* CI
* deterministic outputs

Questions:

Can another researcher reproduce the published results?

Does every claim have an executable artifact?

Are there undocumented assumptions?

Are generated artifacts committed accidentally?

---

## Part II — Specification Consistency

Read every specification document.

Verify:

* D0
* D1
* D1.1
* README
* architectural documents
* constitutional rulings
* migration history

Search for:

* contradictions
* outdated terminology
* definitions changing across documents
* symbols used before definition
* hidden assumptions
* circular definitions

Produce a complete inconsistency table.

---

## Part III — Kernel Minimality

Ignore implementation.

Look only at the theory.

Determine whether every constructor is indispensable.

For each constructor:

NRM

CLS

PWR

GRD

REF

VAL

Attempt to eliminate it.

Can the remaining constructors simulate it?

If yes:

the basis is not minimal.

If no:

explain why.

Do NOT accept author's claims.

Perform independent reasoning.

---

## Part IV — Operational Semantics

Verify that operational semantics precisely implement the specification.

Look for:

undefined transitions

implicit behavior

dead transitions

silent assumptions

unreachable states

state explosion

incorrect temporal semantics

Check every state transition.

---

## Part V — Formal Properties

Verify evidence for:

I1

I2

I3

I4

I5

I6

I7

I8

I9

For each invariant answer:

Implemented?

Tested?

Mechanized?

Proved?

Only claimed?

Provide a table.

---

## Part VI — Soundness

Attempt to break the kernel.

Construct counterexamples.

Examples:

conflicting norms

nested exceptions

recursive references

future amendments

retroactive amendments

cyclic references

multiple authorities

duplicate identifiers

temporal overlap

authority revocation

Explain whether behavior remains deterministic.

---

## Part VII — Completeness

Identify governance constructs that cannot currently be represented.

Examples:

delegation

multi-party authority

probabilistic rules

resource constraints

quantitative obligations

meta-rules

exception priorities

jurisdiction switching

Determine whether these are outside scope or actual weaknesses.

---

## Part VIII — Engineering Review

Review code quality.

Ignore formatting.

Evaluate:

architecture

modularity

coupling

maintainability

determinism

testing

migration strategy

performance assumptions

panic paths

error handling

unsafe assumptions

technical debt

Estimate long-term maintainability.

---

## Part IX — Mathematical Review

Determine whether mathematical claims are justified.

Specifically examine:

constructor algebra

term algebra

minimality

computability

decidability

closure

termination

soundness

completeness

Identify every statement that is stronger than the evidence supports.

---

## Part X — Scientific Novelty

Compare the kernel against:

LegalRuleML

LKIF

SBVR

Drools

Prolog

Datalog

Answer Set Programming

Defeasible Logic

Rule engines

Knowledge graphs

Business rule systems

Separate:

engineering novelty

mathematical novelty

scientific novelty

Do not accept novelty simply because terminology differs.

---

## Part XI — Independent Confirmation

Determine whether repository evidence satisfies scientific confirmation.

Specifically evaluate:

single implementation bias

test oracle bias

shared implementation assumptions

missing independent compiler

missing independent annotation

missing external replication

Determine how much confidence these limitations reduce.

---

## Part XII — Hidden Risks

Look for risks the authors probably overlooked.

Examples:

specification drift

future extensibility

over-generalization

under-specification

semantic ambiguity

versioning issues

migration hazards

ontology drift

silent assumptions

implementation-defined behavior

Provide a ranked list.

---

# Required Output

Produce exactly these sections.

## Executive Summary

Maximum two pages.

---

## Acceptance Decision

One of only:

ACCEPT

ACCEPT WITH MINOR RESERVATIONS

ACCEPT WITH MAJOR RESERVATIONS

REVISE

REJECT

---

## Evidence Matrix

Every major claim.

Evidence.

Confidence.

Missing evidence.

---

## Fatal Findings

Only issues capable of invalidating scientific claims.

---

## Major Findings

Important but non-fatal.

---

## Minor Findings

Editorial and engineering issues.

---

## Constitutional Compliance

Does implementation faithfully implement constitutional rulings?

Yes/No.

Explain.

---

## Scientific Credibility

Evaluate:

Honesty of claims

Scope discipline

Evidence quality

Reproducibility

Novelty

Scientific rigor

Score each from 0–10 with justification.

---

## Technical Debt

Estimate future maintenance burden.

---

## Research Debt

List open research problems.

Separate:

Engineering

Mathematics

Scientific validation

---

## Recommendation for Phase-2

State exactly what should happen next.

Do NOT recommend feature additions unless required for scientific validation.

---

# Review Constraints

* Assume nothing without evidence.
* Never reward ambition.
* Never penalize modesty.
* Distinguish implementation quality from scientific validity.
* Distinguish conjecture from theorem.
* Distinguish tested behavior from proved behavior.
* Distinguish reproducibility from correctness.
* Quote repository evidence whenever possible.
* Every criticism must reference concrete evidence.
* Every compliment must reference concrete evidence.
* If evidence is missing, explicitly state "Evidence not found."

The final report should read like the final review submitted to a top-tier systems or formal methods conference deciding whether GKS Core Phase-1 qualifies as a reproducible scientific contribution.
