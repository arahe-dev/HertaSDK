# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0-next] - 2026-10-10

The v1.0 candidate. The runtime was rebuilt from first principles against a
written contract ([docs/contract.md](docs/contract.md)) and checked against
v0.2.0 by a comparison lab ([lab/](lab)). It keeps v0.2.0's API and
behavior except for the fixes listed below, and is laid out so v1.0's
features slot in without a rewrite. It will be hardened before it is locked
as v1.0.

### Changed (breaking)

- The package is named `hertasdk`, matching its import path (was
  `execution`). Error messages use the `hertasdk:` prefix (was
  `execution:`).
- No dependencies: the runtime owns its weighted FIFO semaphore, and
  `golang.org/x/sync` is gone.
- Go 1.26 or later (`errors.AsType`), down from the 1.27.1 directive of
  v0.2.0.

### Fixed

Each fix is a row of the contract's §8 and is pinned by a test.

1. `Reject` with a held key no longer blocks; it fails fast with
   `Fail(Throttled, ErrOverloaded)`.
2. `Stats().Finished` is counted before the drain signal, so it equals
   `Admitted` once `Shutdown` returns `nil`.
3. One classifier drives both the retry decision and the returned error: a
   plain error after the attempt deadline is `Uncertain` in both, and is
   retried where `Uncertain` retries are allowed.
4. An attempt ended by the caller's own deadline or cancellation is always
   `Permanent` and never retried.
5. `RetryPolicy.On` is copied at construction; editing the map later no
   longer changes behavior or races.
6. A waiter whose own context ended gets `ctx.Err()`, not
   `ErrRuntimeStopping`, even if shutdown began meanwhile.
7. The per-attempt context is cancelled on every path, including a panic.
8. A `Shutdown` whose context expires no longer leaves a goroutine behind.
9. `Stats.Started` counts only calls whose handler actually ran.
10. No goroutine per `Do`, and no allocations on the uncontended path.
11. A `Reject` overload is never reported as `ErrRuntimeStopping`.
12. A nil `*Failure` returned as an error no longer panics; it reads as
    unclassified (`Permanent`, never retried).

### Added

- `Reject` refusals yield the processor once, and a full budget refuses
  without taking its lock. Callers that retry a refusal in a tight loop can
  no longer starve the calls holding capacity: with 256 such callers,
  successful calls went from about 3k/s to about 1M/s (v0.2.0: 57–112k/s).
  The cost falls only on refused calls: a refusal takes about 0.16 µs
  instead of 0.05 µs, and under heavy saturation a refused caller may learn
  of its refusal up to one scheduling round later (p99 about 0.6 ms with
  1,024 requests in flight, server-shaped). Successful calls are unaffected.
- [docs/contract.md](docs/contract.md): the normative contract, including
  FIFO and head-of-line semantics, multi-resource holding and the deltas
  from v0.2.0.
- The [lab/](lab) module (not part of the package): v0.2.0 frozen in
  `lab/v02`, plus assumption, model, differential, chaos, industrial,
  design-of-experiments, mutation and real-consumer suites.
- CI runs the package on Go 1.26 and stable, with a 100% coverage gate, and
  runs the lab's quick tier.

### Performance

Measured against v0.2.0 on a quiet machine
([lab/results/r2](lab/results/r2/REPORT.md)):

- uncontended `Do`: 893 ns / 10 allocs → 59 ns / 0 allocs;
- `Do` with a timeout: 1,465 ns / 14 allocs → 350 ns / 4 allocs;
- 0 goroutines per call (was 1); 157 B per waiting call (was 848 B plus a
  7.7 KB stack);
- mixed-load throughput ×1.72, p99 latency ×0.39;
- all 72 benchmark comparisons better.

### Upgrade notes

**Source-breaking:** code written against v0.2.0 refers to the package as
`execution`, and that name no longer exists. Either keep your code and pin
the old name with an import alias:

```go
import execution "github.com/arahe-dev/hertasdk"
```

or rename the uses: `execution.NewRuntime(...)` becomes
`hertasdk.NewRuntime(...)`, and so on for every identifier. The exported
names and signatures are otherwise unchanged.

- If you match error text, the prefix is now `hertasdk:`. Prefer
  `errors.Is` with the exported sentinels.
- Keyed `Reject` callers now receive `Throttled` refusals where v0.2.0
  blocked on the key (fix 1).

## [0.2.0] - 2026-09-05

API safety hardening. One rule drove every change: **if semantics matter,
absence of semantics must not itself have semantics.** No new execution
features; no changes to admission, arbitration, retry execution, timeouts,
or shutdown behavior.

### Changed (behavioral contract)

- `Effect` zero value is now `EffectUnknown`, not `Pure`. A `Policy` that
  omits `Effect` used to silently claim the strongest retry-safety
  semantics; it is now a construction error. Declaring semantics is the
  library's entire thesis — omission must not grant any.

### Added

- Construction-time validation of the declared execution contract:
  - `EffectUnknown` rejected with `ErrEffectRequired`.
  - Out-of-range `Effect` values rejected with `ErrInvalidEffect`.
  - `RetryPolicy.On[Throttled] = true` rejected with
    `ErrInvalidRetryPolicy`: `Throttled` is observable but NOT
    automatically retryable in V0 — Herta has no backoff/jitter semantics,
    so an immediate retry would only hammer the resource that just refused
    us. Docs updated to match (`On` actionability in V0 = `Transient`, and
    safe `Uncertain`).
  - `On` entries for never-retryable outcomes (`Success`, `Permanent`) and
    unrecognized `Outcome` values rejected with `ErrInvalidRetryPolicy`
    instead of being silently ignored.
- `Fail` normalization: `Fail(Success, err)` ("a successful failure") and
  out-of-range Outcome values can no longer manufacture a nonsense
  `Failure` — the verdict is normalized to `Permanent` with the cause
  preserved via `Unwrap`. `Fail(_, nil) == nil` and `OutcomeOf(nil) ==
  Success` unchanged.
- `Effect.String()` for diagnostics.

### Unchanged

- `Wait` stays the `AdmissionMode` zero value (blocking is the
  conservative default; documented).
- `OutcomeOf(nil) == Success`; `Success` stays the `Outcome` zero value
  (natural for a result classification).
- All runtime execution semantics (admission, keying, rollback, timeout,
  retry loop, shutdown/drain, stats).

### Upgrade notes

- Source-compatible except for the intentional break: `Policy{}` /
  `Policy{...}` literals without an explicit `Effect` now fail
  construction with `ErrEffectRequired`. Add the field.

## [0.1.0] - 2026-09-04

Initial public release. Herta V0 is a minimal in-process execution fabric:
one execution model shared by many heterogeneous operations.

### Added

Frozen V0 contract:

- Wait/Reject admission — `Wait` blocks until capacity is available, the
  caller's context is done, or runtime shutdown begins; `Reject` fails fast
  with `ErrOverloaded` (wrapped as `Fail(Throttled, ErrOverloaded)`).
  (Queue admission was descoped.)
- Keyed serialization — operations sharing a `SerializeKey` execute
  one-at-a-time; the key is acquired BEFORE resources, so a queued
  same-key request holds zero resource budget.
- Cooperative per-attempt timeout — a fresh deadline derived from the
  caller's context is propagated to the handler each attempt; handlers
  that ignore their context are not forcibly terminated.
- Effect×Outcome safe-retry validation — semantically unsafe combinations
  (e.g. `NonIdempotent` + retry on `Uncertain`) are rejected at
  construction with `ErrUnsafeRetry`, and the retry loop enforces the
  same contract independently.
- Graceful shutdown + drain — `Shutdown` rejects new work, withdraws work
  still waiting for key/resource admission, and drains work that already
  began executing (executing handlers run to completion).
- Atomic `Stats` — `Admitted`, `Rejected`, `Started`, `Finished`, `Retried`
  counters with no callbacks or hooks.

### Non-goals

- Transport (no HTTP/gRPC wiring; handlers are plain Go functions).
- Distributed execution (single process only).
- Durable queue (no persistence; Queue admission descoped from V0).
- Priorities (all admitted operations are equal).

### Notes

- v0.1.0 is frozen until a real consumer produces evidence the contract
  is insufficient. No model changes without such evidence.
