# Mutation report: v0.2-recon

428 mutants. Score = killed / (killed + survived); equivalents are excluded, compile mutants are not counted. Raw counts equivalents as survivors.

| operator | mutants | compile | killed | survived | equivalent | score | raw |
|---|---:|---:|---:|---:|---:|---:|---:|
| cond-negate | 73 | 0 | 73 | 0 | 0 | 100.0% | 100.0% |
| cond-boundary | 12 | 0 | 10 | 0 | 2 | 100.0% | 83.3% |
| cond-logic | 21 | 0 | 21 | 0 | 0 | 100.0% | 100.0% |
| cond-not | 14 | 0 | 14 | 0 | 0 | 100.0% | 100.0% |
| stmt-delete | 116 | 7 | 108 | 0 | 1 | 100.0% | 99.1% |
| stmt-swap | 41 | 0 | 34 | 0 | 7 | 100.0% | 82.9% |
| bool-flip | 23 | 0 | 23 | 0 | 0 | 100.0% | 100.0% |
| err-nil | 40 | 1 | 39 | 0 | 0 | 100.0% | 100.0% |
| int-off | 76 | 9 | 62 | 0 | 5 | 100.0% | 92.5% |
| defer-delete | 6 | 2 | 4 | 0 | 0 | 100.0% | 100.0% |
| defer-inline | 6 | 0 | 6 | 0 | 0 | 100.0% | 100.0% |
| **all** | 428 | 19 | 394 | 0 | 15 | 100.0% | 96.3% |

## Kill cascade

| stage | command | kills | score if the cascade stopped here |
|---|---|---:|---:|
| own | `go test -count=1 -short -failfast -vet=off -timeout=60s ./recon/` | 375 | 95.2% |
| lab | `LAB_IMPLS=recon go test -count=1 -failfast -vet=off -timeout=150s -skip ^(TestClockResolution|TestTimerResolution|TestContextCosts|TestGoroutineCost|TestAtomicVsLocks|TestGoroutineSettling|TestGCTailLatency)$ ./lab/assume ./lab/model ./lab/system ./lab/mutate` | 6 | 96.7% |
| race | `go test -race -count=2 -short -failfast -vet=off -timeout=150s ./recon/` | 12 | 99.7% |
| race-lab | `LAB_IMPLS=recon go test -race -count=3 -failfast -vet=off -timeout=150s -run ^TestGapCancelStorm$ ./lab/mutate` | 1 | 100.0% |

## Tests that kill most

| stage | test | mutants killed |
|---|---|---:|
| own | TestBudgetUncontendedAllocatesNothing | 49 |
| own | TestBudgetBlockedWaiter | 44 |
| own | TestBudgetRetryAddsNoAllocations | 36 |
| own | TestContractRejectedCountsOverloadAndStopping | 23 |
| own | TestContractStatsSnapshotOrdered | 20 |
| own | TestShutdownWakesAdmissionWaiters | 20 |
| own | TestConstructionValidationTable | 18 |
| own | TestBudgetKeyedAllocations | 16 |
| own | TestBudgetNoGoroutinesPerDo | 14 |
| own | TestContractEveryAttemptGetsAFreshDeadline | 13 |
| own | TestNewOperationValidation | 13 |
| own | TestContractDoneCallerTakesNothing | 12 |

## Unreproduced failures

These tests failed once on a mutant and passed when re-run, so they killed nothing: they are flaky under the run's load.

| stage: test | mutants |
|---|---:|
| lab: TestKeyReclamation | 1 |
| lab: TestKeyReclamation/recon | 1 |
| lab: TestMiraiTwin | 1 |
| lab: TestMiraiTwin/recon | 1 |

## Survivors (0)

## Equivalent (15)

- #181 recon/outcome.go:129 `cond-boundary` `<` → `<=`: outcomeSet is a uint8: 1<<8 is 0, so o = 8 is false either way; the bound only guards negative shifts.
- #306 recon/runtime.go:67 `stmt-swap` `swap` → `rt.leave() ⇄ rt.stats.rejected.Add(1)`: Both happen before Do returns ErrRuntimeStopping, and no contract orders Rejected against the drain signal (§6 orders only Finished).
- #333 recon/runtime.go:127 `stmt-swap` `swap` → `out.Finished = s.finished.Load() ⇄ out.Started = s.started.Load()`: Admitted is still loaded after both, so neither can exceed it; Finished may legitimately exceed Started (withdrawals).
- #337 recon/runtime.go:129 `stmt-swap` `swap` → `out.Admitted = s.admitted.Load() ⇄ out.Rejected = s.rejected.Load()`: Rejected has no ordering promise; Admitted is still loaded after Started and Finished.
- #339 recon/runtime.go:130 `stmt-swap` `swap` → `out.Rejected = s.rejected.Load() ⇄ out.Retried = s.retried.Load()`: Rejected and Retried have no ordering promise.
- #373 recon/validate.go:71 `int-off` `0` → `1`: The initial bad outcome is overwritten before use: found is false until the first invalid key sets both.
- #374 recon/validate.go:71 `int-off` `0` → `-1`: The initial bad outcome is overwritten before use: found is false until the first invalid key sets both.
- #384 recon/validate.go:77 `cond-boundary` `<` → `<=`: Map keys are distinct, so o == bad never happens.
- #387 recon/validate.go:82 `int-off` `0` → `1`: The set returned beside an error is discarded by NewOperation.
- #392 recon/validate.go:85 `int-off` `0` → `1`: The set returned beside an error is discarded by NewOperation.
- #397 recon/waiter.go:27 `int-off` `1` → `2`: One token is ever sent per waiter; any buffer of at least 1 keeps wake non-blocking.
- #412 recon/waiter.go:77 `stmt-swap` `swap` → `q.tail = w ⇄ q.len++`: Independent writes under the owner's lock.
- #416 recon/waiter.go:92 `stmt-delete` `w.prev, w.next = nil, nil` → ``: A removed waiter is never queued or walked again (waiters are not reused), so its stale links are unreachable.
- #418 recon/waiter.go:92 `stmt-swap` `swap` → `w.prev, w.next = nil, nil ⇄ q.len--`: Independent writes under the owner's lock.
- #428 recon/waiter.go:99 `stmt-swap` `swap` → `q.remove(w) ⇄ w.wake(withdrawn)`: Under the owner's lock; the woken goroutine reads only w.state and needs the lock to touch the queue.

## Stale triage entries (20)

These verdicts matched no surviving mutant in this run; re-check them.

- `recon/errors.go|(*Runtime).refused|stmt-swap|swap|rt.stats.rejected.Add(1) ⇄ runtime.Gosched()|0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).handOff|stmt-delete|e.key = ""||0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).handOff|stmt-swap|swap|e.q.remove(w) ⇄ w.wake(granted)|0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).leave|stmt-delete|<-w.ready||0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).lock|int-off|1|0|0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).lock|int-off|1|2|0 (triaged equivalent, but killed)`
- `recon/keys.go|(*KeyLocker).lock|stmt-delete|e = nil||0 (triaged equivalent, but killed)`
- `recon/operation.go|NewOperation|int-off|1|0|0 (triaged equivalent, but killed)`
- `recon/outcome.go|(outcomeSet).has|int-off|8|7|0 (triaged equivalent, but killed)`
- `recon/outcome.go|(outcomeSet).has|int-off|8|9|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).String|int-off|10|11|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).String|int-off|10|9|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).close|stmt-swap|swap|r.closed = true ⇄ r.q.closeAll()|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).close|stmt-swap|swap|r.q.closeAll() ⇄ r.sync()|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).grant|stmt-swap|swap|r.cur += w.n ⇄ r.q.remove(w)|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).grant|stmt-swap|swap|r.q.remove(w) ⇄ w.wake(granted)|0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).leave|stmt-delete|<-w.ready||0 (triaged equivalent, but killed)`
- `recon/resource.go|(*Resource).sync|stmt-swap|swap|r.isFull = f ⇄ r.full.Store(f)|0 (triaged equivalent, but killed)`
- `recon/runtime.go|(*Runtime).Shutdown|stmt-swap|swap|rt.keys.close() ⇄ rt.markDrained()|0 (triaged equivalent, but killed)`
- `recon/runtime.go||int-off|62|61|0 (triaged equivalent, but killed)`
