# HertaSDK execution contract (v1.0-next)

This document is the normative contract of package `hertasdk`. It was written to reconstruct HertaSDK v0.2.0 from first principles, and that history still shapes it:

- the **exported API** is v0.2.0's, under the package name `hertasdk`;
- the **behavior** is v0.2.0's, except for the numbered deltas in §8 (each one a confirmed v0.2 defect);
- the **internals are designed to carry v1.0** (§9) without restructuring.

When this document and v0.2 disagree, this document wins only where §8 says so. Otherwise v0.2's observable behavior is the oracle. v0.2.0 is kept, frozen, in `lab/v02` for that comparison.

## 1. Axioms

**A1. Business logic is ordinary.**
A handler is `func(ctx, I) (O, error)`. Herta never computes, stores or interprets results.

**A2. Absence of semantics has no semantics.**
An undeclared Effect, an out-of-range enum, or an untagged error never grants a permission. The conservative reading always wins.

**A3. Scarce capacity is never held by work that is not executing**, except as stated in §4.3.

**A4. Unsafe contracts fail at construction, not under load.**

**A5. Cost is proportional to contention.**
An uncontended call pays only for what it uses:
- no goroutines;
- no allocations without a timeout;
- only the `context.WithTimeout` cost when a timeout is set.

## 2. Vocabulary (exported, identical to v0.2.0)

| Identifier | Shape |
|---|---|
| `Runtime` | `NewRuntime(specs ...ResourceSpec) (*Runtime, error)`, `Shutdown(ctx) error`, `Stats() Stats` |
| `ResourceSpec` | `{Name string; Capacity int64}` |
| `Resource` | exported type with `String() string`; internals unexported |
| `KeyLocker` | exported type; internals unexported |
| `Operation[I,O]` | `NewOperation(rt, name, handler, policy) (*Operation[I,O], error)`, `Do(ctx, I) (O, error)` |
| `Handler[I,O]` | `func(context.Context, I) (O, error)` |
| `Policy[I]` | `{Effect; Resources []Requirement; Admission AdmissionMode; Timeout time.Duration; SerializeKey func(I) string; Retry RetryPolicy}` |
| `Requirement` | `{Name string; Units int64}` |
| `RetryPolicy` | `{MaxAttempts int; On map[Outcome]bool}` |
| `Effect` | `EffectUnknown`(0), `Pure`, `Idempotent`, `NonIdempotent`, with `String()` |
| `AdmissionMode` | `Wait`(0), `Reject` |
| `Outcome` | `Success`(0), `Transient`, `Permanent`, `Throttled`, `Uncertain`, with `String()` |
| `Failure` | `{Outcome Outcome; Err error}`, `Error()`, `Unwrap()` |
| `Fail`, `OutcomeOf` | `Fail(Outcome, error) error`, `OutcomeOf(error) Outcome` |
| `Stats` | `{Admitted, Rejected, Started, Finished, Retried uint64}` |
| Sentinels | `ErrUnsafeRetry`, `ErrEffectRequired`, `ErrInvalidEffect`, `ErrInvalidRetryPolicy`, `ErrInvalidOutcome`, `ErrOverloaded`, `ErrRuntimeStopping` |

These rules carry over from v0.2, with the rename:
- The package name is `hertasdk` (v0.2: `execution`), and error message text uses the `hertasdk:` prefix (v0.2: `execution:`).
- v0.2's own test files (`execution_test.go`, `proof_test.go`, `bench_test.go`) pass against `hertasdk` with only their package line renamed. Until the rename they passed **byte-for-byte unchanged** (lab/results/r2/correctness).

## 3. Construction (`NewRuntime`, `NewOperation`)

### 3.1 `NewRuntime` rejects
- no resources at all (Amendment 3);
- an empty resource name;
- a duplicate resource name;
- a capacity ≤ 0.

### 3.2 `NewOperation` rejects

| Error | Cases |
|---|---|
| plain error | `rt` is nil; `handler` is nil; `name` is empty; an unknown resource; Units ≤ 0; Units > capacity; the same resource listed twice |
| `ErrEffectRequired` | `Effect` is `EffectUnknown` |
| `ErrInvalidEffect` | `Effect` is out of range |
| `ErrInvalidRetryPolicy` | `On[Throttled]`, `On[Success]` or `On[Permanent]` is true; an `On` key that is out of range |
| `ErrUnsafeRetry` | `Effect` is `NonIdempotent` and `On[Uncertain]` is true |

Order of checks: when several checks fail, the error reported must match v0.2. For any input where the cause would otherwise depend on map iteration order, report `ErrInvalidRetryPolicy` before `ErrUnsafeRetry`. Validation is deterministic: the same input always gives the same error.

### 3.3 Snapshot at construction
Construction snapshots the policy:
- resource claims are resolved to budget pointers and sorted by name;
- `Retry.On` is copied into an immutable table (Δ5).

Mutating the caller's `Policy`, its slice or its map afterwards has no effect.

### 3.4 Normalization (same as v0.2)
- `MaxAttempts` ≤ 1 means exactly 1 attempt.
- `Timeout` ≤ 0 means no per-attempt deadline.
- Any `Admission` other than `Reject` behaves as `Wait`.

## 4. Admission (`Do`, before the handler runs)

`Do` goes through these steps in this exact order:

1. **Gate.** If the runtime is stopping, return `ErrRuntimeStopping`, untagged, so `OutcomeOf` is `Permanent`. Count `Rejected`. Otherwise count `Admitted`. From here on the call counts as in flight for drain.
2. **Key** (if `SerializeKey` is set and returns a non-empty string). If the caller's context is already done, return `ctx.Err()`, uncounted, in either mode (Amendment 1). Otherwise acquire per-key mutual exclusion.
   - `Wait`: block until the key is free, the caller's context is done, or shutdown begins (Amendment 4).
   - `Reject`: if the key is held, fail immediately with `Fail(Throttled, ErrOverloaded)` and count `Rejected` (**Δ1**, Amendment 5).
   - A key waiter holds **no** resource units.
3. **Resources**, in ascending name order. For each claim:
   - `Wait`: if the caller's context is already done, return `ctx.Err()` (Amendment 1). Otherwise block until the units are granted, the caller's context is done, or shutdown begins (Amendment 4).
   - `Reject`: try to acquire. If it isn't granted, fail with `Fail(Throttled, ErrOverloaded)` and count `Rejected`. This is v0.2 behavior, kept: `Rejected` counts both stopping and overload.

   Any failure releases every unit and key already acquired, in reverse order, before returning.

   **Wait errors.** If the caller's context is done, return the caller's `ctx.Err()`. Otherwise, if shutdown woke the waiter, return `ErrRuntimeStopping` (**Δ6**).
4. **Start.** Count `Started` only when the handler is about to run for the first time (**Δ9**).

### 4.1 Budget semantics (identical to `x/sync/semaphore`, made explicit)
- Every budget is a weighted semaphore with a **strict FIFO** waiter queue.
- A request is granted immediately only if it fits **and no one is queued**. Otherwise `Wait` enqueues it and `Reject` fails.
- On release, waiters are granted from the head of the queue for as long as each one fits. A waiter that doesn't fit blocks everyone behind it. This head-of-line behavior is intended: it prevents heavy requests from starving.
- A waiter that cancels is removed. If it was at the head, the waiters behind it are re-examined.
- **A `Reject` refusal yields the processor once (`runtime.Gosched`) before returning.** This applies whether a budget or a key refused the call. It changes no result, outcome or count. It exists because a caller that retries a refusal at once is waiting, and must not hoard the CPU the holders need to finish (A3; evidence in `lab/results/r1/reject-probe/ROOT-CAUSE.md`).

### 4.2 Keys
- Keys give per-key mutual exclusion with FIFO hand-off.
- Entries are reference-counted and reclaimed when the last holder or waiter leaves.
- Idle keys cost zero memory.

### 4.3 Multi-resource holding (v0.2 behavior, kept)
While waiting for a later resource in name order, an operation **holds** the earlier ones it has already acquired.

- This is deadlock-free, because every operation acquires in the same global order.
- It is the one stated exception to A3.
- v1.0 may replace it with all-or-nothing acquisition (§9).

## 5. Execution and retry

For attempt n = 1 … `MaxAttempts`:

1. If the caller's context is done, stop. The return value is defined in §5.2.
2. Build the attempt context:
   - with a timeout set, `attemptCtx = WithTimeout(callerCtx, Timeout)`, cancelled on **every** exit path, including panics (**Δ7**);
   - otherwise, `attemptCtx = callerCtx`.
3. Run the handler. If it returns a nil error, return its result. This holds even if a deadline passed: a late result is returned, not discarded. That is v0.2 behavior, kept.
4. Classify the error with **one** function, used for both the retry decision and the error returned (**Δ3**):
   - A `*Failure` anywhere in the chain: use its outcome. The first (outermost) `*Failure` in the `errors.As` walk wins (Amendment 2).
   - Untagged and `errors.Is(err, context.Canceled)`: `Permanent`.
   - Untagged and the **attempt** deadline expired while the caller's context is still live: `Uncertain`, and the error returned is `Fail(Uncertain, err)` (**Δ3/Δ4**).
   - Untagged and the caller's own context ended: `Permanent`, never retried. The caller's error is returned (§5.2).
   - Any other untagged error: `Permanent`.
5. Retry only if the outcome is allowed by the retry table **and** safe for the Effect:
   - `Transient` is safe for every Effect. The adapter vouches that the attempt did not execute; the runtime trusts it.
   - `Uncertain` is safe for every Effect except `NonIdempotent`.
   - Everything else is never retried.

   Each retry counts in `Retried`. There is no backoff in v0.2.
6. When attempts run out, or the outcome is not retryable, return the classified error from step 4.

### 5.1 Panics
A panicking handler releases its attempt context, its resources (in reverse order) and its key exactly once, and marks the call finished. The panic then propagates unchanged, including `panic(nil)`.

### 5.2 Caller context ends
Once the caller's context is done:
- **Before or between attempts:** `Do` returns `ctx.Err()`, untagged.
- **During an attempt:** the handler's own error is returned, classified as in §5 step 4. Caller-driven endings are `Permanent` and are never retried.

## 6. Shutdown and drain

`Shutdown(ctx)`:
1. **Stop admitting.** This is atomic with the gate in §4 step 1: no call can be admitted after `Shutdown` has observed "stopping".
2. **Wake** every waiter (key or resource). Each fails per §4 step 3.
3. **Drain.** Return `nil` once every admitted call has finished. If `ctx` ends first, return `ctx.Err()`; draining continues.

Further rules:
- `Shutdown` is idempotent and safe to call concurrently.
- Repeated calls with a timeout create **no** extra goroutines (**Δ8**).
- When `Shutdown` returns `nil`, `Stats().Finished == Stats().Admitted` must already hold (**Δ2**).
- If the runtime is idle and the context has already expired, `Shutdown` returns `nil` (drained beats expired).

## 7. Stats
- Every counter is atomic and monotonic.
- `Admitted` counts gate passes.
- `Rejected` counts stopping refusals plus `Reject` overloads (v0.2 semantics).
- `Started` counts handler entries (first attempts only).
- `Finished` counts calls that have fully exited `Do` after admission, including withdrawals.
- `Retried` counts attempts beyond the first.

## 8. Intentional deltas from v0.2.0 (the only allowed behavior differences)

| Δ | v0.2.0 | recon |
|---|---|---|
| 1 | `Reject` with a held key blocks | fails fast with `Fail(Throttled, ErrOverloaded)` |
| 2 | `wg.Done()` runs before `finished++`, so `Finished` can lag after `Shutdown()==nil` | `Finished` is counted before the drain signal |
| 3 | `classify` (retry decision) and `finish` (error returned) disagree, so a plain error after the attempt deadline is reported `Uncertain` without being retried | one classification feeds both; Pure/Idempotent with `On[Uncertain]` retries it |
| 4 | a parent deadline is reported 2 different ways depending on `Timeout`/`On` | caller-ended attempts are always `Permanent`, never `Uncertain`-promoted and never retried |
| 5 | `Retry.On` is kept by reference: post-construction mutation changes behavior and races | snapshotted at construction |
| 6 | `admissionError` reports `ErrRuntimeStopping` even when the caller's context caused the wake-up | the caller's `ctx.Err()` takes precedence |
| 7 | the attempt context is not cancelled when the handler panics | cancelled on every path |
| 8 | each timed-out `Shutdown` parks a goroutine until drain | no per-call goroutines |
| 9 | `Started` counts calls whose handler never ran (pre-cancelled, no resources) | counted only when the handler runs |
| 10 | one goroutine plus about 10 allocations per `Do` | 0 goroutines; allocations per §10 |

| 11 | a `Reject` overload is reported as `ErrRuntimeStopping` (still counted in `Rejected`) when `Shutdown` began after the call passed the gate | reported as `Fail(Throttled, ErrOverloaded)`, per §4 step 3 (Amendment 6) |
| 12 | a nil `*Failure` returned as an error (`var f *Failure; return v, f`) panics in `OutcomeOf`, `Do`, `Error` and `Unwrap` | read as untagged: `Permanent`, never retried; `Error` and `Unwrap` are nil-safe |

Any other observed difference is a **bug in recon**, unless it is nondeterministic in v0.2 (for example, which waiter wins a tie under `Shutdown`).

## 9. Designed for v1.0 (internal architecture requirements)

recon must make these v1.0 features additive, not a rewrite:

- **Own budget semaphore**, with no `golang.org/x/sync` dependency. It exposes exact counts of waiters and units in use, which v1.0 needs for `Snapshot()`, `MaxWaiting` and `MaxWait`, and keeps room for all-or-nothing multi-resource acquisition.
- **One admission path** (key, then resources), parameterized by mode, so that a bounded `Wait` (with `MaxWait` and `MaxWaiting`) slots in.
- **One classifier, one retry table** (a bitset over `Outcome`), so that `OutcomeSet`, `Backoff` and retryable `Throttled` slot in.
- **Centralized error construction**, so the v1.0 typed `Error` (with `Op`, `Outcome` and `Attempts`) slots in.
- **Lock-free admission/drain state**: one atomic word holding the stopping bit and the in-flight count.
- **No per-call goroutines**; context propagation via `context.AfterFunc` only where a waiter actually blocks.
- **Package layout:** a handful of files, each owning one concern; no file over about 250 lines.

## 10. Performance budgets (recon, measured on the reference machine)

| Path | Budget |
|---|---|
| `Do`, uncontended, 1 resource, `Timeout=0`, `context.Background()` | **0 allocs**, ≤ 150 ns |
| `Do`, uncontended, 1 resource, `Timeout>0` | ≤ allocs of a bare `context.WithTimeout` + 0 |
| `Do`, uncontended, keyed | ≤ 1 alloc (amortized 0 with entry pooling) |
| Waiter (blocked acquire) | ≤ 2 allocs, ≤ 256 B excluding the goroutine stack |
| Idle key | 0 B after release |
| Goroutines created per `Do` | 0 |
| Retry attempt overhead | 0 allocs beyond the attempt context |

## 11. Amendments

Each amendment resolves a point where this document was silent, ambiguous or wrong about v0.2. Each one follows v0.2, except Amendment 6, which only names a v0.2 race. Each cites the evidence and the test that pins it.

**Amendment 1. A done caller takes no key and waits for no units** (§4 steps 2 and 3).
- **Rule:** if the caller's context is already done when admission reaches the key (in either mode) or a `Wait` claim, `Do` returns `ctx.Err()` untagged. It holds nothing, and `Rejected` is not counted. A `Reject` claim does not consult the context, so an unkeyed `Reject` call with a done context still reports overload when the units are taken.
- **Why:** v0.2 acquires keys and `Wait` units through `x/sync/semaphore.Weighted.Acquire`. That call fails first on a done context, even when the semaphore is free (v0.22.0, `semaphore.go`: "We prefer to fail even if we could acquire"). `TryAcquire`, used for `Reject` units, never looks at a context. A3 points the same way: a caller that has given up must not hold capacity.
- **Gap closed:** before this amendment the contract left a pre-cancelled keyed `Reject` call undefined. recon answered `Fail(Throttled, ErrOverloaded)` and counted `Rejected`, where v0.2 answers `context.Canceled`.
- **Pinned by:** `contract_test.go` `TestContractDoneCallerTakesNothing`; by `lab/assume` `TestDoneCallerTakesNothing`, where orig passes the same expectations; and by `lab/system` `TestDifferential`, whose generator now draws pre-cancelled keyed `Reject` calls. With the recon fix reverted, the differential fails within 30 steps.

**Amendment 2. The outermost `*Failure` wins** (§5 step 4).
- v0.2's `OutcomeOf` and `classify` use `errors.As`, which stops at the first `*Failure` in the chain.
- For `&Failure{Transient, &Failure{Uncertain, e}}`, both orig and recon return `Transient`.
- `Fail` never nests: re-tagging an error that already carries a `*Failure` returns it unchanged. So only a hand-built chain can reach this case.

**Amendment 3. `NewRuntime()` with no specs is rejected** (§3.1).
- The error is "execution: runtime requires at least one resource".
- The v0.2 suite requires this, and recon runs that suite unchanged.

**Amendment 4. "Shutdown begins" withdraws only calls that are queued** (§4 steps 2 and 3, §6 step 2).
- A call that passed the gate before `Shutdown` still takes a key or units that are free when it reaches them, and runs.
- A waiter that was granted before the withdrawal keeps its grant.
- v0.2 is nondeterministic here: its per-call watcher goroutine cancels the admission context at some point after `Shutdown`. recon takes the reading v0.2 shows whenever its watcher is late.
- The lab model marks these interleavings as ambiguous (`model.ErrAmbiguous`), and the generators skip them.

**Amendment 5. A `Reject` call refused because its key is held counts in `Rejected`** (§7, Δ1).
- §7 counts "`Reject` overloads", and this refusal is one.
- This is `model.KeyRejectCounted`. It is pinned by `lab/assume` (deltas ledger, Δ1) and `delta_test.go` `TestDelta1RejectFailsFastOnHeldKey`.

**Amendment 6. Δ11 names a v0.2 race; it adds no new behavior** (§8).
- v0.2 passes every admission failure through `admissionError`. Once `Shutdown` has begun, that function rewrites a `Reject` overload into `ErrRuntimeStopping`, and `Rejected` is still counted.
- Reaching that state requires `Shutdown` to land between the gate and `TryAcquire`, which is a race. The last clause of §8 already covered it; Δ11 names it so that the ledger can account for it.
- The lab measures it: the trace reconciliation in `lab/system` `TestChaos` logs "N overloads were reported as ErrRuntimeStopping" for orig and requires 0 for recon.

**Note (no amendment needed).** `ErrInvalidOutcome` is never returned by orig or recon. `Fail` normalizes invalid outcomes to `Permanent` instead. The sentinel exists only for API parity.
