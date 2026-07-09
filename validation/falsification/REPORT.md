# Falsification Campaign — held-out corpus (Track C)

Inputs screened: 8

## Per-unit outcome

- OK `labour:Điều-90 (wage ≥ minimum)` → VAL
- OK `labour:Điều-5 (worker has right)` → NRM
- OK `iso:8.7 (nonconforming classification)` → CLS
- OK `policy:concession authority (power)` → PWR
- OK `cross-ref: KPI → P-11 §4` → REF
- **HALT** `ADVERSARIAL: 'for ALL transactions, without bound'` — term uses operator "forall" outside the sub-Turing algebra T — would require unbounded/undecidable expressiveness (I1/I3)
- **HALT** `ADVERSARIAL: proposes an 8th constructor` — constructor "OBLIGATION2" is outside the closed basis B={NRM,CLS,PWR,GRD,REF,VAL} — would require an eighth constructor (I3)
- **HALT** `ADVERSARIAL: fixpoint / self-reference` — term uses operator "fix" outside the sub-Turing algebra T — would require unbounded/undecidable expressiveness (I1/I3)

## Result

- admitted units use **5** distinct constructors: [CLS NRM PWR REF VAL]
- FALSIFICATION-CANDIDATEs halted: **3**
- Registry Law (basis ≤ 6, Θ(1)): **HELD**

The kernel was NOT extended to admit any adversarial input; each was
halted with a FALSIFICATION-CANDIDATE record (I3 Iron Rule preserved).
