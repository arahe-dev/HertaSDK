# Why spinning `Reject` callers starve: root cause

The measurements below were taken on 2026-10-10 with Go 1.27.1 on windows/amd64 and 20 logical CPUs (`GOMAXPROCS=20`). The setup was capacity 4, a no-op handler, and callers in a closed loop that retry a refused call at once ("spin"). Hold times are measured with the TSC clock, which adds no scheduling points. Scheduler latency is the runtime's own `/sched/latencies` histogram. Files: `root_test.go.txt`, `block_test.go.txt`, `defense_test.go.txt`, `rejlat_test.go.txt`.

## The chain

1. **Spinning callers never block.** A refused `Do` returns in about 50 ns and the loop calls again. The goroutine gives up its processor only when the runtime forces it to: `forcePreemptNS = 10ms` (`runtime/proc.go:6679`).
2. **Sometimes the forced preemption lands while the goroutine holds something:**
   - a capacity unit (handler running, release not yet done), or
   - the budget's mutex (inside `tryAcquire` / `release`, about 20 ns).

   In both cases a normal hold is 100–500 ns.
3. **A preempted goroutine goes to the global run queue** (`goschedImpl` → `globrunqput`, `proc.go:4322`). A processor reads that queue only when its local queue is empty, or once every 61 scheduling ticks (`proc.go:3458`).
4. **Goroutines woken from a mutex take a fast lane.** `semrelease1` → `readyWithTime` → `ready(…, next=true)` puts them in the waking processor's `runnext` slot. So spinners that parked on the budget mutex keep cycling locally, while the preempted holder waits in the global queue behind about 256 runnable goroutines.
5. **Capacity 4 means four stuck holders stop all progress.** Everyone else spins on refusals until a holder is rescheduled, which takes hundreds of milliseconds.

## Evidence

**256 spinning callers:**

| Measure | v0.2 | recon |
|---|---:|---:|
| successful calls/s | 48,879 | 2,061–3,319 |
| hold time p50 | 300 ns | 100 ns |
| hold time p99.9 / max | 38 / 119 ms | 268–549 / 412–809 ms |
| share of hold time in holds > 1 ms | 99.1% | 99.8–100% |
| scheduler latency p99 | 117 ms | 100–940 ms |
| fairness across callers (Jain) | 0.39 | 0.02–0.09 |
| GC cycles/s | 6 | 1–2 |

Holds of 100 ns that occasionally become 100–800 ms mean the holder was descheduled; nothing on that path blocks. GC is ruled out (6 cycles/s or fewer).

**Who waits, from the block profile (recon, spin, 256 callers):**
- 93% of blocked time is in `tryAcquire`, on the budget mutex.
- 7% is in `release`.
- The mutex profile charges the waits to unlocks in the same two places, so it is mutex-holder preemption as well as unit-holder preemption.

**Scheduler latency scales with runnable goroutines per processor** (recon, spin, 256 callers), as a round-robin model predicts:

| GOMAXPROCS | callers per P | scheduler latency p99 |
|---:|---:|---:|
| 4 | 64 | 940 ms |
| 10 | 25 | 470 ms |
| 20 | 12 | 100–335 ms |

**The collapse starts when callers outnumber `GOMAXPROCS`.** Below that, every goroutine has its own processor and nothing waits in a queue (`starve_test.go.txt`).

**Why v0.2 suffers less, but still suffers** (49k successful calls/s spinning against 1.1M yielding): a v0.2 refusal costs about 2 µs and 11 allocations, and every call starts a watcher goroutine. That makes many more scheduling events, which give stuck holders more chances to run. This is plausible but not isolated: GC is ruled out, and the watcher's exact contribution was not measured.

## The defense: yield on a refusal while saturated (`yield-on-full.patch`)

In `tryAcquire`, when the budget's full flag is set, call `runtime.Gosched()` before returning `false`.
- The refused caller leaves its processor. It joins the same global FIFO queue as preempted holders, so holders get scheduled within one short round, and spinners no longer sit on a processor for 10 ms.
- It passes recon's full test suite under `-race`.

| Shape | recon | with yield-on-full | |
|---|---:|---:|---|
| spin, 256 callers | 3.3k/s | **935k/s** | ×283; fairness 0.06 → 1.00; scheduler p99 335 → 1.3 ms |
| spin, 16 callers | 1.24 M/s | 1.70 M/s | ×1.37 |
| yielding callers, 256 | 1.09 M/s | 1.64 M/s | ×1.50 |
| yielding callers, 16 | 1.71 M/s | 2.33 M/s | ×1.36 |
| server shape, 64 / 256 / 1,024 in flight | 1.58 / 1.42 / 1.52 M/s | 1.61 / 1.59 / 1.58 M/s | ×1.02–1.12 |

For comparison, v0.2 spinning at 256 callers manages 49–181k/s. With the defense, recon does about 5–19× better.

**The cost is paid only by refused calls while the budget is full:**
- One refusal on an idle machine: 48 ns → 151 ns.
- Server shape, refusal latency p50: about 1 µs → 3–5 µs.
- Server shape, refusal latency p99 at 1,024 in flight: 159 µs → 486 µs. The refused caller waits up to one scheduling round before it learns the answer.
- Successful calls and the uncontended path are unchanged.

## Verdict

**The root cause is lock-holder preemption under Go's scheduler, made severe by its two-lane run queues.** A preempted holder goes to the global FIFO queue, while mutex-woken spinners take the per-processor `runnext` fast lane. It is not a recon logic bug. Any user-space admission control in Go has it when callers busy-retry, and v0.2 has it too (49k vs 1.1M).

**Why recon was hit harder:** its refusal is 29× cheaper, so spinners spend more of their time contending and holders are preempted more often.

**A defense is cheap and fits Herta's own principle.** The principle is "waiting work must not hoard scarce resources"; a caller that busy-retries a refusal is waiting, and CPU is the scarce resource it hoards. Yielding on a refusal while saturated:
- removes the collapse (×283) and makes every shape faster;
- costs only the refused calls: a few microseconds at p50, under 0.5 ms at p99.

**Recommendation:** adopt yield-on-full in recon and record it in CONTRACT §4.1 as "a Reject refusal against a saturated budget yields the processor once before returning". Keep the caller guidance (back off after a refusal) and v1.0 `Backoff`, because yielding mitigates the problem but does not replace backpressure.
