# Mutation report: v0.2.0 (orig)

265 mutants. Score = killed / (killed + survived); equivalents are excluded, compile mutants are not counted. Raw counts equivalents as survivors.

| operator | mutants | compile | killed | survived | equivalent | score | raw |
|---|---:|---:|---:|---:|---:|---:|---:|
| cond-negate | 40 | 0 | 40 | 0 | 0 | 100.0% | 100.0% |
| cond-boundary | 14 | 0 | 11 | 0 | 3 | 100.0% | 78.6% |
| cond-logic | 7 | 0 | 7 | 0 | 0 | 100.0% | 100.0% |
| cond-not | 8 | 0 | 8 | 0 | 0 | 100.0% | 100.0% |
| stmt-delete | 52 | 2 | 49 | 0 | 1 | 100.0% | 98.0% |
| stmt-swap | 14 | 0 | 6 | 0 | 8 | 100.0% | 42.9% |
| bool-flip | 12 | 0 | 8 | 0 | 4 | 100.0% | 66.7% |
| err-nil | 36 | 0 | 36 | 0 | 0 | 100.0% | 100.0% |
| int-off | 68 | 3 | 61 | 0 | 4 | 100.0% | 93.8% |
| defer-delete | 7 | 2 | 5 | 0 | 0 | 100.0% | 100.0% |
| defer-inline | 7 | 0 | 7 | 0 | 0 | 100.0% | 100.0% |
| **all** | 265 | 7 | 238 | 0 | 20 | 100.0% | 92.2% |

## Kill cascade

| stage | command | kills | score if the cascade stopped here |
|---|---|---:|---:|
| own | `go test -count=1 -short -failfast -vet=off -timeout=60s .` | 200 | 84.0% |
| lab | `LAB_IMPLS=orig go test -count=1 -failfast -vet=off -timeout=150s -skip ^(TestClockResolution|TestTimerResolution|TestContextCosts|TestGoroutineCost|TestAtomicVsLocks|TestGoroutineSettling|TestGCTailLatency)$ ./lab/assume ./lab/model ./lab/system ./lab/mutate` | 37 | 99.6% |
| race | `go test -race -count=2 -short -failfast -vet=off -timeout=150s .` | 1 | 100.0% |
| race-lab | `LAB_IMPLS=orig go test -race -count=3 -failfast -vet=off -timeout=150s -run ^TestGapCancelStorm$ ./lab/mutate` | 0 | 100.0% |

## Tests that kill most

| stage | test | mutants killed |
|---|---|---:|
| own | TestCapacityLimitHeldUnderLoad | 25 |
| own | TestNewOperationValidation | 22 |
| lab | TestGapNothingAccumulates | 20 |
| own | TestConstructionValidationTable | 18 |
| lab | TestDifferential | 16 |
| lab | TestGapAcquisitionFollowsNameOrder | 16 |
| own | TestNewRuntimeValidation | 15 |
| own | TestFreshTimeoutPerAttempt | 14 |
| own | TestShutdownWakesAdmissionWaiters | 14 |
| own | TestPanicReleasesEverythingAndPropagates | 13 |
| lab | TestAdmissionDrain | 12 |
| lab | TestKeyReclamation | 12 |

## Unreproduced failures

These tests failed once on a mutant and passed when re-run, so they killed nothing: they are flaky under the run's load.

| stage: test | mutants |
|---|---:|
| lab: TestMiraiTwin | 1 |
| lab: TestMiraiTwin/orig | 1 |

## Survivors (0)

## Equivalent (20)

- #18 keylock.go:54 `stmt-swap` `swap` → `l.sem.Release(1) ⇄ kl.release(l)`: The handler has returned when unlock runs; dropping the ref before releasing the semaphore can only orphan a semaphore nobody else will wait on, never admit two holders.
- #69 operation.go:177 `stmt-delete` `lastDeadlineExpired = false` → ``: The timed path reassigns it every attempt and the untimed path never sets it, so the reset is redundant.
- #90 operation.go:184 `stmt-swap` `swap` → `lastDeadlineExpired = attemptCtx.Err() == context.DeadlineExceeded ⇄ cancel()`: Context errors are sticky: after an expired deadline cancel() leaves Err() == DeadlineExceeded, and before it Err() becomes Canceled, which also compares false. The code comment claiming the order matters is wrong.
- #114 operation.go:252 `stmt-swap` `swap` → `acquired = append(acquired, res) ⇄ units = append(units, req.Units)`: Two independent appends to parallel slices.
- #118 operation.go:242 `stmt-swap` `swap` → `releaseAll() ⇄ op.runtime.stats.rejected.Add(1)`: No contract orders the Rejected increment against the release of earlier claims; both happen before Do returns.
- #127 operation.go:266 `int-off` `1` → `0`: Starting the insertion sort at 0 adds one pass whose inner loop (j > 0) never runs.
- #136 operation.go:267 `cond-boundary` `<` → `<=`: Two requirements share a position only if they name the same resource, which construction rejects.
- #154 policy.go:56 `bool-flip` `true` → `false`: Effect.valid has no callers in v0.2 (dead code).
- #155 policy.go:58 `bool-flip` `false` → `true`: Effect.valid has no callers in v0.2 (dead code).
- #157 policy.go:141 `cond-boundary` `<=` → `<`: MaxAttempts == 1 yields 1 through either branch.
- #159 policy.go:141 `int-off` `1` → `0`: MaxAttempts == 1 yields 1 through either branch.
- #171 runtime.go:57 `int-off` `0` → `1`: Capacity hint of make; the slice grows as needed.
- #175 runtime.go:68 `stmt-swap` `swap` → `resources[s.Name] = newResource(s.Name, s.Capacity) ⇄ order = append(order, s.Name)`: Two independent writes inside construction.
- #190 runtime.go:96 `stmt-swap` `swap` → `rt.wg.Add(1) ⇄ rt.stats.admitted.Add(1)`: Both run under the read lock that Shutdown's write lock excludes; no observer can see the gap.
- #202 runtime.go:103 `stmt-swap` `swap` → `rt.wg.Done() ⇄ rt.stats.finished.Add(1)`: This is CONTRACT Δ2's fix. The Finished lag it removes is a race (about 1 in 20 000 drains), so no deterministic test can demand the defect, and the lab accepts both orders for v0.2.
- #214 runtime.go:143 `stmt-swap` `swap` → `rt.accepting = false ⇄ rt.cancelWait()`: Both run under the write lock; woken waiters test the closed stop channel, not accepting, and begin cannot run until the lock is released.
- #221 runtime.go:193 `int-off` `1` → `0`: Starting the insertion sort at 0 adds one pass whose inner loop (j > 0) never runs.
- #230 runtime.go:194 `cond-boundary` `<` → `<=`: Resource names are unique (duplicates fail construction), so no two compare equal.
- #262 validation.go:128 `bool-flip` `false` → `true`: Belt and braces: the retry table validated at construction never contains Success, Permanent or Throttled, so mayRetry is already false.
- #265 validation.go:139 `bool-flip` `false` → `true`: Belt and braces: an out-of-range Outcome cannot be in the retry table (construction rejects it), so mayRetry is already false.
