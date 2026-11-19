# Trust ledger evaluation

Reproduce from repo root: `python3 herta/eval/trust/evaluate.py`.
Builds the current CLI once; invokes `trust -json` per file and `check -closed` for HC016 metadata.
Frozen inputs/evidence are read only. No oracle reruns; labels/mechanisms below are from
`herta/eval/closed/summary.md` (sections A/B and verified-path discussion).

Inputs use legacy `build` semantics; migrated models use `build -closed`; corpus uses legacy mode.
A classify clause alone does not install Policy.Classify in a legacy build.
Both raw input and faithful migration versions of all 20 programs are covered;
121 corpus `.he` models are covered (not nested pilot workspaces or mutation variants).

## Counts

Each cell is **unique origin obligations / propagated per-op occurrences / programs flagged**.
Unique keys are (file, origin, kind, detail). Required/informational is retained in JSON output;
transient-honest is required only for nonidempotent origins. Counts are warnings, not bugs.

| Kind | inputs (legacy) | migrated (closed) | corpus (legacy) |
|---|---:|---:|---:|
| opaque-handler | 45 / 45 / 20 | 45 / 67 / 20 | 283 / 447 / 121 |
| transient-honest | 15 / 15 / 10 | 15 / 25 / 10 | 28 / 29 / 27 |
| legacy-tag-path | 45 / 45 / 20 | 30 / 42 / 15 | 283 / 447 / 121 |
| nested-retry-amplification | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| same-serialize-key | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| unresolved-callee | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |

| Group | Programs | Ops | Required origin obligations | Informational origin obligations |
|---|---:|---:|---:|---:|
| inputs (legacy) | 20 | 45 | 100 | 5 |
| migrated (closed) | 20 | 45 | 85 | 5 |
| corpus (legacy) | 121 | 283 | 566 | 28 |

## All closed-evaluation programs

Non-trivial kinds below exclude opaque-handler, and list **local origins**, not merely
an unrelated warning propagated onto the op. `none` is a miss for that location.

| Program | Input local non-trivial assumptions | Migrated local non-trivial assumptions | Input HC016 origins |
|---|---|---|---|
| H1/t1 | main::charge: legacy-tag-path, transient-honest | main::charge: transient-honest | main::charge |
| H1/t2 | main::refund: legacy-tag-path | main::refund: legacy-tag-path | none |
| H1/t3 | main::charge: legacy-tag-path, transient-honest; main::checkout: legacy-tag-path; main::reserve: legacy-tag-path, transient-honest | main::charge: transient-honest; main::checkout: legacy-tag-path; main::reserve: transient-honest | main::charge, main::reserve |
| H1/t4 | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::move_funds: legacy-tag-path; main::transfer: legacy-tag-path | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::move_funds: legacy-tag-path; main::transfer: legacy-tag-path | none |
| H2/t1 | main::charge: legacy-tag-path, transient-honest | main::charge: transient-honest | main::charge |
| H2/t2 | main::refund: legacy-tag-path | main::refund: legacy-tag-path | none |
| H2/t3 | main::charge: legacy-tag-path, transient-honest; main::checkout: legacy-tag-path; main::reserve: legacy-tag-path, transient-honest | main::charge: transient-honest; main::checkout: legacy-tag-path; main::reserve: transient-honest | main::charge, main::reserve |
| H2/t4 | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | none |
| H3/t1 | main::charge: legacy-tag-path, transient-honest | main::charge: transient-honest | main::charge |
| H3/t2 | main::refund: legacy-tag-path | main::refund: legacy-tag-path | none |
| H3/t3 | main::charge: legacy-tag-path, transient-honest; main::checkout: legacy-tag-path; main::reserve: legacy-tag-path, transient-honest | main::charge: transient-honest; main::checkout: legacy-tag-path; main::reserve: transient-honest | main::charge, main::reserve |
| H3/t4 | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | none |
| R/t1 | main::charge: legacy-tag-path, transient-honest | main::charge: transient-honest | main::charge |
| R/t2 | main::charge: legacy-tag-path; main::refund: legacy-tag-path | main::charge: legacy-tag-path; main::refund: legacy-tag-path | none |
| R/t3 | main::charge: legacy-tag-path, transient-honest; main::checkout: legacy-tag-path; main::reserve: legacy-tag-path, transient-honest | main::charge: transient-honest; main::checkout: legacy-tag-path; main::reserve: transient-honest | main::charge, main::reserve |
| R/t4 | main::charge: legacy-tag-path; main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | main::charge: legacy-tag-path; main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | none |
| R2/t1 | main::charge: legacy-tag-path, transient-honest | main::charge: transient-honest | main::charge |
| R2/t2 | main::charge: legacy-tag-path; main::refund: legacy-tag-path | main::charge: legacy-tag-path; main::refund: legacy-tag-path | none |
| R2/t3 | main::charge: legacy-tag-path, transient-honest; main::checkout: legacy-tag-path; main::reserve: legacy-tag-path, transient-honest | main::charge: transient-honest; main::checkout: legacy-tag-path; main::reserve: transient-honest | main::charge, main::reserve |
| R2/t4 | main::charge: legacy-tag-path; main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | main::charge: legacy-tag-path; main::credit: legacy-tag-path; main::debit: legacy-tag-path; main::transfer: legacy-tag-path | none |

## Known failures: location is not diagnosis

| Baseline oracle2 failure | Relevant origin | Input kinds at origin | Migrated kinds at origin | Causal assessment |
|---|---|---|---|---|
| H2/t1 | main::charge | legacy-tag-path, transient-honest | transient-honest | Broad trust obligation at charge; NOT a diagnosis of joined-error arm ordering (HC010 does that). |
| R/t1 | main::charge | legacy-tag-path, transient-honest | transient-honest | Broad trust obligation at charge; NOT a diagnosis of joined-error arm ordering (HC010 does that). |
| R/t3 | main::reserve | legacy-tag-path, transient-honest | transient-honest | MISS: reserve is flagged informationally, but missing retry-on-Uncertain is not detected. |
| R/t4 | main::transfer | legacy-tag-path | legacy-tag-path | MISS: dynamic transfer/account key collision is hidden in Go key values; no HC001/same-serialize-key here. |

## HC016 coverage

15 input op-level HC016 sites in 10 programs; 15/15 have a non-trivial assumption at the exact origin.
The per-program table lists every HC016 origin. All have legacy-tag-path; retry-on-Transient
ones also have transient-honest (required for charge, informational for reserve). This exposes
missing classification authority, not proof that the original behavior is buggy.
Migrated HC016 sites: 0.

## Limitations / misses

- All handlers remain opaque, so strict mode rejects every evaluated program; this is a trust
  boundary, not a useful binary bug classifier. Every legacy op is also conservatively tagged.
- Non-trivial coverage is deliberately not called recall: generic obligations can mark both
  correct and buggy classifiers. No new causal diagnosis of the four failures is claimed.
- R/t3's transient-honest warning does not point to its missing uncertain retry; R2/t3 has
  similar obligations but passes. The ledger cannot infer required application behavior.
- R/t4 retains a generic legacy-tag-path flag, but no matching key hazard flag: distinct symbolic
  key names hide the concrete `transfer:A` collision. Closed runtime ErrKeyReacquired is the
  matching guard, not this ledger.
- Raw service/helper calls in input t3/t4 are opaque; migration's declared calls expose their
  dependencies. No inferred edge through arbitrary Go method dispatch is claimed.
- Corpus counts are an inventory of frozen models, not oracle verification or classifier honesty proofs.
- JSON is available by rerunning CLI per file; summary intentionally stores no copies of frozen models.
