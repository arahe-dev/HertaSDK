# Herta conformance cases

Golden-file conformance cases for the hertasdk execution contract
(`docs/contract.md`). Each case ports one or more of HertaSDK's Go tests
(`*_test.go`) so the suite can diff Herta-compiled behavior against the SDK
oracle.

Layout follows the capped runner (`lang/hertatest`):

```
herta/conformance/cases/<slug>/
    input.herta     # the Herta program under test (required)
    expected.txt    # golden stdout of `herta check input.herta` (required)
    expected.exit   # present when the check must fail (optional)
    scenario.yaml   # behavioral scenario + expectations for the
                    # conformance driver (runtime-semantics cases only)
```

Caps enforced by the runner: 200 cases, 5 s per case, 60 s total, 64 KiB
per file. Keep the suite small and deterministic: gates and counts, never
millisecond assertions.

## Case groups

| group | cases | covers |
|---|---|---|
| `construct-*` | 5 | construction-time validation as compile errors (contract §3) |
| `admission-*` | 7 | budgets, admission modes, cancellation (§4) |
| `key-*` | 3 | SerializeKey serialization (§4.2) |
| `retry-*`, `attempt-*`, `caller-*`, `failure-*` | 4 | execution and retry (§5) |
| `panic-*` | 1 | panic semantics (§5.1, Δ7) |
| `shutdown-*` | 1 | shutdown and drain (§6, Δ1/Δ2) |

Every `scenario.yaml` carries a `source:` line naming the Go test(s) it
ports, so a coverage diff is mechanical.

## scenario.yaml format

`input.herta` declares the runtimes, ops and handlers. `scenario.yaml`
holds the original case spec: the runtime driver script (`steps:` — calls,
spawns, gates, contexts, shutdown, direct key/unit holds) and the
`expected:` observations a correct backend must produce (per-call results
with error sentinel and outcome, handler call counts, concurrency peaks,
FIFO start orders, final stats). The step vocabulary and observation keys
are documented inside each file; the conformance driver (separate PR)
executes them against hertasdk first and compiled Herta later.

## Ported vs. unportable

Three Go tests have no Herta surface — the grammar cannot express them;
they are covered by hertasdk's Go suite instead:

- `TestContractValidationIsDeterministic` / out-of-range `Outcome` values:
  `retry on` accepts only the five outcome keywords.
- `TestContractUnknownAdmissionWaits` (`AdmissionMode(99)`): `admission`
  accepts only `wait|reject`.
- `TestDelta5RetryTableIsSnapshotted`: mutating a caller-supplied `On` map
  has no language analog (the retry set is a literal).
- `TestEmptyKeyMeansNoSerialization`: `serialize` requires a named
  parameter; "empty key" collapses to "no `serialize` clause", covered by
  the `unkeyed` op in `key-serialization`.

Also intentionally not ported: allocation/goroutine budgets
(`budget_test.go`, §10, Δ8, Δ10), race-detector hammers and
snapshot-ordering proofs (`race_test.go`, `proof_test.go`,
`TestContractStatsSnapshotOrdered`, `TestContractStatsSnapshotsAreConsistent`),
and latency benchmarks — properties a language-level diff cannot observe
cheaply. They stay Go-side tests.
