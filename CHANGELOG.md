# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
