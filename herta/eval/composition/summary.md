# Herta composition checker: frozen real-repo evaluation

- Checker: `e9741d23d663ea1ad8e708a5f0e88393424b4416` (no checker fixes during evaluation).
- 121/121 models validated: exit 0 or 1, only HC/known diagnostics, no parse errors or crashes.
- Models frozen before labels were opened: `2026-11-19T16:59:58.768249+00:00`.
- Corpus manifest SHA-256: `2f6f8d142d59eba789cfa43ebc52de27bfd2229cfc5861306d87849ae5b408b8`; verified unchanged when scoring.
- Label file SHA-256: `5e3ee32ddc90e7e123bb22fd099a38e5dcfe8ec5619b30604ad70d7a23fa3dc5`.
- No model was edited after label access or scoring. review-decisions.tsv was never opened.
- Flagged means any diagnostic from the checker, not just the analyzer site's matching diagnostic family.
- False-positive rate = flagged (TP-benign + FP) / all (TP-benign + FP); Unclear is excluded.

## Results

| Analyzer check | Sites | Bugs caught | Non-bug flags / non-bugs (FPR) | Unclear flagged / total | Inexpressible |
| --- | ---: | ---: | ---: | ---: | ---: |
| C1 | 30 | 4/4 (100.0%) | 1/23 (4.3%) | 0/3 | 11 |
| C2 | 33 | 11/11 (100.0%) | 3/22 (13.6%) | 0/0 | 5 |
| C3 | 28 | 1/1 (100.0%) | 0/24 (0.0%) | 0/3 | 3 |
| C4 | 30 | 3/3 (100.0%) | 0/27 (0.0%) | 0/0 | 1 |
| Overall | 121 | 19/19 (100.0%) | 4/96 (4.2%) | 0/6 | 20 |

## Distinct mechanisms, not independent bug counts

Grouped by the hand labels' subtype (a mechanism taxonomy, not a claim of independent root causes):

| Mechanism | Bug sites caught / total |
| --- | ---: |
| ambiguous-copy-replay | 1/1 |
| ambiguous-task-replay | 2/2 |
| caller-cancel-not-propagated | 1/1 |
| csr-allocation-replay | 1/1 |
| exhausted-auth-budget-repeated | 11/11 |
| mutex-self-deadlock | 1/1 |
| recursive-read-writer-starvation | 2/2 |

Mechanism coverage: **7/7** (a mechanism is caught if any of its sites is flagged).
The 11 C2 bug rows all represent the same b2 exhausted-auth-budget flaw, not 11 independent flaws. Collapsing only those 11 rows gives 9 site/flaw groups; subtype grouping additionally combines the two task-replay sites and the two recursive-RLock sites.

## Diagnostic counts

Site counts are unique models; instances include repeated diagnostics on different call paths.

| Code | Sites | Instances | Bug sites | Non-bug sites | Unclear sites |
| --- | ---: | ---: | ---: | ---: | ---: |
| HC002 | 3 | 3 | 3 | 0 | 0 |
| HC005 | 7 | 10 | 4 | 3 | 0 |
| HC006 | 12 | 13 | 11 | 1 | 0 |
| HC008 | 1 | 1 | 1 | 0 | 0 |
| LEGACY_UNSAFE_RETRY | 7 | 8 | 4 | 3 | 0 |

LEGACY_UNSAFE_RETRY is a scoring name for the existing uncoded local nonidempotent/uncertain diagnostic; it is not a new HC code. Codes absent from this table did not fire.

## Missed bugs

None: all 19 labeled bug sites were flagged.

## Flags on non-bug rows

- `C1-0a578406485b-1` (TP-benign): `HC005;LEGACY_UNSAFE_RETRY`. UploadSessionStartContext is called with nil content and an empty start arg (2070-2072); retries may allocate unused sessions but no file is committed until the later finish operation.
- `C2-111ecc19364a-1` (TP-benign): `HC006`. iscsi_util.go 326 uses 5 attach rounds, each calling waitForPathToExist 420 with deviceDiscoveryTimeout=30; helper 148-176 checks up to30 times: 5*30=150 local stat/glob probes per portal, deliberate discovery (scanner's6 is off by one).
- `C2-37fcde000258-1` (TP-benign): `HC005;LEGACY_UNSAFE_RETRY`. batch-handlers.go 1161 retries the batch (default3); replicateObjectWithMultipart (bucket-replication.go 1637-1679) retries create and abort up to3 each; default3*3=9 per operation, deliberate bounded multipart recovery; scanner's inner4 is wrong.
- `C2-b57fafc140ac-1` (TP-benign): `HC005;LEGACY_UNSAFE_RETRY`. StartFromSource409 repeats batch (default3); copyWithMultipartfromSource212-246 defers up to3 aborts only after failed upload; default3*3=9 abort attempts for cleanup; scanner's inner4 is off by one.

## Unclear rows (not treated as bugs or false positives)

- `C1-2247ea143023-1`: not flagged; inexpressible: Datasource-specific sample dedupe and re-evaluation determinism are not established; cannot truthfully choose one effect for the polymorphic writer..
- `C1-3d89636795c7-1`: not flagged; inexpressible: Concrete provider effect/idempotence and conflict-after-commit behavior are unknown; method name CreateRoute cannot establish an effect contract..
- `C1-4d53c3ab1046-1`: not flagged; inexpressible: drainBody writes to io.Discard. Opt-in arbitrary-verb transport has no established concrete nonidempotent caller; cannot assign its effect truthfully..
- `C3-034276477d8d-1`: not flagged; inexpressible: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path..
- `C3-059c3b6c94a6-1`: not flagged; inexpressible: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path..
- `C3-07b2a60bd673-1`: not flagged; inexpressible: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path..

## Scope and limitations

This measures checking of human-transcribed contracts, not source-level bug discovery. Facts already supply branch reachability, retry-stop behavior, ownership, and lock identity that the checker itself does not infer from Go.
No budget overrides, invented dedupe keys, or intentionally understated effect declarations were used to fit labels. Effects and edges were chosen before label access; false positives are retained unchanged.
Pinned sources were consulted for actual attempt bounds (rclone default 10, Kubernetes backoff 4/5, Loki reads 5), transport retry behavior, and multipart allocation. Source copies and per-model rationale are in sources/ and transcription.json.
The certificate model covers the actual selected five-attempt backoff; the subsequent separate infinite poll is not represented with an invented finite bound. C2 auth models project the credential-recovery path established by facts, not unspecified surrounding user mutations. Abort-on-inner-failure recovery is kept separate per transcription rule 6.
Time-driven/unbounded retry policies without a fixed attempt cap, unknown effect contracts, and unverified incoming context paths use pure placeholders with an inexpressible header. Pure placeholders remain in all denominators; acceptance is not evidence that the original behavior was proven safe.

### Inexpressible models

- `C1-2247ea143023-1`: Datasource-specific sample dedupe and re-evaluation determinism are not established; cannot truthfully choose one effect for the polymorphic writer.
- `C1-2527bf0d5bc0-1`: Startup/periodic bootstrap reconciliation has a context/time-controlled poll, not a supplied finite attempt bound; Herta only has retry max N. Desired-state relist/delete semantics are otherwise idempotent.
- `C1-28bb8e1c677e-1`: Only cache/local credential mutation and telemetry at the identified site; no remote IAM mutation to model.
- `C1-2b11ddbc47ad-1`: Only in-process discovery registry mutations; no remote mutation, governed retry, lock, or context behavior in the supplied facts.
- `C1-2e51a05dfcc9-1`: Repeated controller reconciliation has no finite retry-attempt bound in the facts; Herta cannot encode that policy. Delete and finalizer reconciliation are explicitly idempotent.
- `C1-2f277c230bcf-1`: Pinned idempotency.go uses PollUntilContextTimeout, not a fixed attempt cap; Herta cannot encode this timed retry exactly. GET-before-Create replay safety is established.
- `C1-39c1239aa801-1`: The 10-minute periodic StorageVersion reconciliation is not a finite retry policy. Fixed-name desired-state replacement is otherwise idempotent.
- `C1-3d89636795c7-1`: Concrete provider effect/idempotence and conflict-after-commit behavior are unknown; method name CreateRoute cannot establish an effect contract.
- `C1-40d1d017f6e5-1`: Only intentional stderr logging before retry is identified; no replay of an external mutation in the facts.
- `C1-4d53c3ab1046-1`: drainBody writes to io.Discard. Opt-in arbitrary-verb transport has no established concrete nonidempotent caller; cannot assign its effect truthfully.
- `C1-4ea745d6eef3-1`: Leader acquisition/renewal is context/deadline-driven polling without a finite attempt bound in facts; Herta retry max N cannot express it. Refetch and resourceVersion updates are replay-safe.
- `C2-437671419da0-1`: Snapshot polling has a 30s time bound, not a finite inner attempt count. Outer conflict bound is 4, but no truthful numeric product can be encoded.
- `C2-47cbf4a83c5f-1`: Allocation polling has a 30s time bound with no fixed inner attempt cap; outer conflict bound 4 does not establish a numeric product.
- `C2-861e7b839dc8-1`: Watch shares a 2min context with 10s readiness polling; neither time bound establishes an exact attempt count.
- `C2-862ec1dc2340-1`: Mutually sequential fast/slow health modes share cancellation; there is no finite attempt product to encode in retry max N.
- `C2-ae6f138273d8-1`: 30s startup polling has no exact outer attempt bound; inner DefaultBackoff=4 alone cannot establish a product.
- `C3-034276477d8d-1`: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path.
- `C3-059c3b6c94a6-1`: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path.
- `C3-07b2a60bd673-1`: Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path.
- `C4-0ba54e39c62f-1`: HTTP verb/effect unspecified; no lock or singleflight behavior to express. Pure placeholder is not a claim about arbitrary HTTP methods.

## Checker changes during evaluation

None. No crashes or parse failures required a checker fix.
