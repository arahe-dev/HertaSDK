# Mutation report: v0.2-recon

396 mutants. Score = killed / (killed + survived); equivalents are excluded, compile mutants are not counted. Raw counts equivalents as survivors.

| operator | mutants | compile | killed | survived | equivalent | score | raw |
|---|---:|---:|---:|---:|---:|---:|---:|
| cond-negate | 67 | 0 | 66 | 1 | 0 | 98.5% | 98.5% |
| cond-boundary | 10 | 0 | 8 | 0 | 2 | 100.0% | 80.0% |
| cond-logic | 18 | 0 | 18 | 0 | 0 | 100.0% | 100.0% |
| cond-not | 14 | 0 | 14 | 0 | 0 | 100.0% | 100.0% |
| stmt-delete | 108 | 6 | 96 | 1 | 5 | 99.0% | 94.1% |
| stmt-swap | 33 | 0 | 20 | 1 | 12 | 95.2% | 60.6% |
| bool-flip | 22 | 0 | 22 | 0 | 0 | 100.0% | 100.0% |
| err-nil | 40 | 1 | 39 | 0 | 0 | 100.0% | 100.0% |
| int-off | 74 | 9 | 52 | 0 | 13 | 100.0% | 80.0% |
| defer-delete | 5 | 2 | 3 | 0 | 0 | 100.0% | 100.0% |
| defer-inline | 5 | 0 | 5 | 0 | 0 | 100.0% | 100.0% |
| **all** | 396 | 18 | 343 | 3 | 32 | 99.1% | 90.7% |

## Kill cascade

| stage | command | kills | score if the cascade stopped here |
|---|---|---:|---:|
| own | `go test -count=1 -short -failfast -vet=off -timeout=60s ./recon/` | 330 | 95.4% |
| lab | `LAB_IMPLS=recon go test -count=1 -failfast -vet=off -timeout=150s -skip ^(TestClockResolution|TestTimerResolution|TestContextCosts|TestGoroutineCost|TestAtomicVsLocks|TestGoroutineSettling|TestGCTailLatency)$ ./lab/assume ./lab/model ./lab/system ./lab/mutate` | 7 | 97.4% |
| race | `go test -race -count=2 -short -failfast -vet=off -timeout=150s ./recon/` | 5 | 98.8% |
| race-lab | `LAB_IMPLS=recon go test -race -count=3 -failfast -vet=off -timeout=150s -run ^TestGapCancelStorm$ ./lab/mutate` | 1 | 99.1% |

## Tests that kill most

| stage | test | mutants killed |
|---|---|---:|
| own | TestBudgetUncontendedAllocatesNothing | 49 |
| own | TestBudgetBlockedWaiter | 38 |
| own | TestBudgetRetryAddsNoAllocations | 35 |
| own | TestContractRejectedCountsOverloadAndStopping | 22 |
| own | TestShutdownWakesAdmissionWaiters | 20 |
| own | TestConstructionValidationTable | 18 |
| own | TestBudgetKeyedAllocations | 15 |
| own | TestBudgetNoGoroutinesPerDo | 15 |
| own | TestContractEveryAttemptGetsAFreshDeadline | 13 |
| own | TestNewOperationValidation | 13 |
| own | TestContractDoneCallerTakesNothing | 12 |
| own | TestContractShutdownWaitsForDrain | 10 |

## Unreproduced failures

These tests failed once on a mutant and passed when re-run, so they killed nothing: they are flaky under the run's load.

| stage: test | mutants |
|---|---:|
| lab: TestKeyReclamation | 2 |
| lab: TestKeyReclamation/recon | 2 |
| lab: TestMiraiTwin | 2 |
| lab: TestMiraiTwin/recon | 2 |

## Survivors (3)

### #95 recon/keys.go:101 `stmt-delete` in (*KeyLocker).handOff

`keyEntries.Put(e)` → ``  
key: `recon/keys.go|(*KeyLocker).handOff|stmt-delete|keyEntries.Put(e)||0`

```diff
--- recon/keys.go
+++ recon/keys.go (mutant)
@@ -100,3 +100,3 @@
 	e.key = ""
-	keyEntries.Put(e)
+	
 }
```

### #303 recon/runtime.go:128 `stmt-swap` in (*Runtime).Stats

`swap` → `out.Started = s.started.Load() ⇄ out.Admitted = s.admitted.Load()`  
key: `recon/runtime.go|(*Runtime).Stats|stmt-swap|swap|out.Started = s.started.Load() ⇄ out.Admitted = s.admitted.Load()|0`

```diff
--- recon/runtime.go
+++ recon/runtime.go (mutant)
@@ -127,4 +127,4 @@
 	out.Finished = s.finished.Load()
-	out.Started = s.started.Load()
-	out.Admitted = s.admitted.Load()
+	out.Admitted = s.admitted.Load()
+	out.Started = s.started.Load()
 	out.Rejected = s.rejected.Load()
```

### #351 recon/validate.go:77 `cond-negate` in retryTable

`<` → `>=`  
key: `recon/validate.go|retryTable|cond-negate|<|>=|0`

```diff
--- recon/validate.go
+++ recon/validate.go (mutant)
@@ -76,3 +76,3 @@
 			set |= setOf(o)
-		case !found || o < bad:
+		case !found || o >= bad:
 			bad, found = o, true
```

## Equivalent (32)

- #67 recon/keys.go:51 `int-off` `1` → `2`: A key waiter's n is never read; only Resource waiters use it.
- #68 recon/keys.go:51 `int-off` `1` → `0`: A key waiter's n is never read; only Resource waiters use it.
- #75 recon/keys.go:59 `stmt-delete` `e = nil` → ``: admit ignores the entry whenever lock reports !ok.
- #83 recon/keys.go:74 `stmt-delete` `<-w.ready` → ``: Redundant since waiters stopped being pooled: w.state is read under the owner's lock, which ordered the owner's write, and the unread token dies with the waiter. Delete the statement.
- #93 recon/keys.go:100 `stmt-delete` `e.key = ""` → ``: lock overwrites e.key on every Get; the reset only lets a pooled entry drop its last key string sooner.
- #100 recon/keys.go:95 `stmt-swap` `swap` → `e.q.remove(w) ⇄ w.wake(granted)`: Both run under kl.mu; the woken goroutine reads only w.state, and re-enters the queue only through kl.mu.
- #123 recon/operation.go:61 `int-off` `1` → `0`: attempts = 0 still runs one attempt: the loop stops when n >= attempts after the first.
- #175 recon/outcome.go:112 `cond-boundary` `<` → `<=`: outcomeSet is a uint8: 1<<8 is 0, so o = 8 is false either way; the bound only guards negative shifts.
- #176 recon/outcome.go:112 `int-off` `8` → `9`: outcomeSet is a uint8: 1<<8 is 0, so o = 8 is false either way; the bound only guards negative shifts.
- #177 recon/outcome.go:112 `int-off` `8` → `7`: Bits 5 to 7 are never set: setOf is only called with valid outcomes.
- #184 recon/resource.go:37 `int-off` `10` → `11`: No exported path yields a *Resource; String exists for API parity and recon's errors format r.name and r.size directly.
- #185 recon/resource.go:37 `int-off` `10` → `9`: No exported path yields a *Resource; String exists for API parity and recon's errors format r.name and r.size directly.
- #226 recon/resource.go:92 `stmt-delete` `<-w.ready` → ``: Redundant since waiters stopped being pooled: w.state is read under the owner's lock, which ordered the owner's write, and the unread token dies with the waiter. Delete the statement.
- #244 recon/resource.go:110 `stmt-swap` `swap` → `r.cur += w.n ⇄ r.q.remove(w)`: remove does not touch w.n or r.cur; both run under r.mu.
- #246 recon/resource.go:111 `stmt-swap` `swap` → `r.q.remove(w) ⇄ w.wake(granted)`: Both run under r.mu; the woken goroutine reads only w.state, and touches the queue only through r.mu.
- #251 recon/resource.go:119 `stmt-swap` `swap` → `r.closed = true ⇄ r.q.closeAll()`: Both run under r.mu; nothing observes closed in between.
- #262 recon/runtime.go:12 `int-off` `62` → `61`: The drained bit only has to sit above any reachable in-flight count; 2^61 calls in flight is unreachable.
- #274 recon/runtime.go:67 `stmt-swap` `swap` → `rt.leave() ⇄ rt.stats.rejected.Add(1)`: Both happen before Do returns ErrRuntimeStopping, and no contract orders Rejected against the drain signal (§6 orders only Finished).
- #296 recon/runtime.go:107 `stmt-swap` `swap` → `rt.keys.close() ⇄ rt.markDrained()`: markDrained acts only with nothing in flight, and then no key has a waiter to withdraw.
- #301 recon/runtime.go:127 `stmt-swap` `swap` → `out.Finished = s.finished.Load() ⇄ out.Started = s.started.Load()`: Admitted is still loaded after both, so neither can exceed it; Finished may legitimately exceed Started (withdrawals).
- #305 recon/runtime.go:129 `stmt-swap` `swap` → `out.Admitted = s.admitted.Load() ⇄ out.Rejected = s.rejected.Load()`: Rejected has no ordering promise; Admitted is still loaded after Started and Finished.
- #307 recon/runtime.go:130 `stmt-swap` `swap` → `out.Rejected = s.rejected.Load() ⇄ out.Retried = s.retried.Load()`: Rejected and Retried have no ordering promise.
- #341 recon/validate.go:71 `int-off` `0` → `1`: The initial bad outcome is overwritten before use: found is false until the first invalid key sets both.
- #342 recon/validate.go:71 `int-off` `0` → `-1`: The initial bad outcome is overwritten before use: found is false until the first invalid key sets both.
- #352 recon/validate.go:77 `cond-boundary` `<` → `<=`: Map keys are distinct, so o == bad never happens.
- #355 recon/validate.go:82 `int-off` `0` → `1`: The set returned beside an error is discarded by NewOperation.
- #360 recon/validate.go:85 `int-off` `0` → `1`: The set returned beside an error is discarded by NewOperation.
- #365 recon/waiter.go:27 `int-off` `1` → `2`: One token is ever sent per waiter; any buffer of at least 1 keeps wake non-blocking.
- #380 recon/waiter.go:77 `stmt-swap` `swap` → `q.tail = w ⇄ q.len++`: Independent writes under the owner's lock.
- #384 recon/waiter.go:92 `stmt-delete` `w.prev, w.next = nil, nil` → ``: A removed waiter is never queued or walked again (waiters are not reused), so its stale links are unreachable.
- #386 recon/waiter.go:92 `stmt-swap` `swap` → `w.prev, w.next = nil, nil ⇄ q.len--`: Independent writes under the owner's lock.
- #396 recon/waiter.go:99 `stmt-swap` `swap` → `q.remove(w) ⇄ w.wake(withdrawn)`: Under the owner's lock; the woken goroutine reads only w.state and needs the lock to touch the queue.
