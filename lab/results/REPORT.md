# v0.2-recon vs v0.2.0: results

Run `r1`, 2026-10-09, go1.27.1 windows/amd64, i7-13650HX (20 threads), on an otherwise idle machine. Raw data is in [`r1/`](r1/).

## Conclusion

**recon is a drop-in replacement for v0.2.0.**
- It behaves the same everywhere except the 11 documented fixes.
- It is faster and leaner in every measured regime where both implementations follow the same contract.
- The one regression found (`Reject` under a retry storm) has been root-caused and fixed.

| Question | Answer | Evidence |
|---|---|---|
| Same behavior? | **Yes**, apart from CONTRACT §8 (Δ1–Δ11) | v0.2's 25 tests pass byte-for-byte unchanged; 100k-scenario differential vs reference model; mirai-backend suite identical with recon swapped in |
| Faster? | **Yes, 2–17× per call** | 72/72 benchmark comparisons better, all p < 0.001 |
| Lighter? | **Yes** | 0 vs 10 allocs per call; 0 vs 1 goroutine per call; 159 B vs 860 B heap per waiter |
| Better under load? | **Yes**, in every factorial cell where both follow the same contract (1.42–2.42×), after the fast-reject fix | DoE: throughput ×2.2, p99.9 ×0.49, fairness up; see the `Reject` section below |
| Stable over time? | **Tie** | 60 s soak each: flat heap, no drift, identical db-write p99 |
| Tests stronger? | **Yes** | own-suite mutation kill 95.4% vs 84.0%; 100% statement coverage vs 93.6% |

## Headline numbers

| Metric | v0.2.0 | recon | Source |
|---|---:|---:|---|
| `Do`, uncontended | 1,050 ns · 464 B · 10 allocs | **63 ns · 0 B · 0 allocs** | `r1/bench.md` |
| `Do`, with timeout | 1,669 ns · 14 allocs | **392 ns · 4 allocs** (= bare `WithTimeout`) | `r1/bench.md` |
| `Do`, keyed | 1,299 ns · 13 allocs | **110 ns · 0 allocs** | `r1/bench.md` |
| Contention, 4 CPUs | 609 ns · 10 allocs | **143 ns · 0 allocs** | `r1/bench.md` |
| Contention, 16 CPUs | 1,893 ns · 12 allocs | **1,002 ns · 1 alloc** | `r1/bench.md` |
| Goroutines per `Do` | 1 | **0** | `r1/industrial.jsonl` |
| Heap per parked waiter | 860 B (+7.3 KB stack) | **159 B** (no extra goroutine) | `r1/industrial.jsonl` |
| 1M unique keys: keyed calls/s | 0.58 M | **2.20 M** | `r1/industrial.jsonl` |
| 1M unique keys: peak extra heap | 40 MB | **5.4 MB** | `r1/industrial.jsonl` |
| 1M unique keys: retained after | 1.5 MB | **0** | `r1/industrial.jsonl` |
| Drain under load, max | 4.6–5.4 ms | **3.0–3.3 ms** | `r1/industrial.jsonl` |
| 10× spike: backlog cleared | 101 ms | 101 ms | `r1/industrial.jsonl` |
| 60 s soak: db-write p99 early→late | 3.92→3.98 ms | 3.85→4.01 ms | `r1/industrial.jsonl` |
| Source size | 1,050 lines + `x/sync` | 1,034 lines, stdlib only | |
| Statement coverage | 93.6% | 100% | |
| Mutation kill (own tests / all) | 84.0% / 100% | 95.4% / 99.1% | `lab/mutate/results/` |

**DoE, 32-run resolution-IV screen** (10 factors × implementation × 3 replicates, quiet; `r1/screen-quick/REPORT.md`):

| | |
|---|---|
| throughput | ×2.23 |
| goodput | ×1.37 |
| p50 / p99 / p99.9 latency | ×0.61 / ×0.67 / ×0.49 |
| admission-wait p99 | ×0.61 |
| peak goroutines | 3,769 → 2 |
| capacity / key violations | 0 for both implementations |

## `Reject` under an in-process retry loop: found, explained, fixed

**Finding.** Full factorial over clients × admission × keys, 5 replicates (`r1/reject-interaction/`). With `Reject` and 256 clients that retry *immediately*, recon's goodput is **0.47×** v0.2's. Every `Wait` cell favors recon: 1.38–1.72× goodput and up to 2.6× better p99.

**Root cause** (follow-up probe, 2026-10-10; evidence in [`r1/reject-probe/`](r1/reject-probe/README.md)). An earlier version of this section blamed lock contention alone and proposed a lock-free fast reject as the fix. That was only half right. There are two mechanisms:

1. **CPU starvation (the main cause, and not a library defect).** A tight loop of retried rejections never blocks.
   - **Onset:** the collapse starts exactly when clients outnumber `GOMAXPROCS` (20). Below that, recon beats v0.2.
   - **What it does:** a call holding a unit that gets preempted waits behind the spinners for Go scheduler slices. Successful calls wait 1–25 ms at p99.
   - **Why recon suffers more:** a v0.2 rejection costs 2,030 ns and 11 allocations, against 70 ns and 1 allocation in recon. v0.2 spends that time allocating and starting goroutines, which accidentally gives holders a chance to run.
   - **The proof:** making rejections even cheaper makes the collapse *worse*. A lock-free fast reject drops to 26–45k successes/s. One `runtime.Gosched()` after each rejection removes it for every implementation.
2. **Lock contention on the reject path (secondary; a real recon issue).** Even with yielding callers, every recon rejection takes the budget's mutex. At 64–256 clients that cuts recon to 0.53–0.83× of v0.2 (519–812k vs 982–984k).

**Where it shows up.** In a server shape, each request is its own goroutine and exits after a 429. There, **neither mechanism appears**: recon delivers about 1.5× v0.2's successful calls at every concurrency from 16 to 1,024. The weakness needs in-process code that retries `Reject` in a loop. The lab's DoE driver is exactly that kind of code, which is why it showed 0.47×.

**Fixed (2026-10-10).** Each step was gated on green tests.

1. **recon: fast reject when full** (`recon/resource.go`, the variant C patch).
   - **How:** an atomic `full` flag is stored only when a budget enters or leaves saturation. `tryAcquire` returns `false` without the lock while it is set.
   - **Cost:** none measurable on the uncontended path (59 vs 60 ns).
   - **Pinned by** `TestResourceFastReject`, plus a `full`-flag invariant now checked in every resource test, including the 16-goroutine churn under `-race`.
2. **lab: `TestRejectUnderLoad`** (`lab/industrial`). Recon must deliver ≥ 0.9× v0.2 for server-shaped and yielding callers. It passed 4 of 4 runs, at 1.46–1.49× (server) and 1.45–2.85× (yield). Spin-loop callers are recorded, not judged.
3. **lab: realistic DoE driver.** Closed-loop callers now yield after a refusal (`Workload.SpinOnRefusal`, recorded in the `on_refusal` column of `runs.csv`); `-spin` restores the old behavior.

**Rejected after measurement.**
- A "units free" hint written on every change: +17% (about 10 ns) on every uncontended call.
- Skipping the key when a claim is full (a pre-check in `admit`): 0.85–1.35×, inconsistent, so reverted. See `peek_test.go.txt`.

**Result.** The same factorial, rerun with the fixed recon and the realistic driver (`r1/reject-interaction-fixed/`):

| Cell | Before | After | p99, v0.2 → recon |
|---|---:|---:|---|
| Reject · 8 clients | 2.55× | 2.42× | 35 → 9 µs |
| Reject · 8 clients · keys | 2.06× | 1.91× | 79 → 74 µs |
| **Reject · 256 clients** | **0.47×** | **1.43×** | **1,352 → 119 µs** |
| Reject · 256 clients · keys | 0.48× | 0.36× | Δ1, explained below |
| Wait (4 cells) | 1.38–1.72× | 1.42–1.75× | recon better in all |

The keyed `Reject` cell compares two different behaviors, not two speeds (`keyed_test.go.txt`, instrumented):
- **v0.2 almost never rejects there:** 13 refusals/s against recon's 1.4 M/s. Its keyed `Reject` blocks on a held key, which is the Δ1 bug, so it is really running `Wait`.
- **Every recon refusal in that cell is "key held"** (about 990k per run, none from a full budget). That is the contract working.
- **Like for like, recon is faster:** recon keyed `Wait` delivers 1.29–1.32 M/s against v0.2's keyed "Reject" at 0.68 M/s, which is 1.9×.

**Spinning callers, root cause and fix.** The root cause is lock-holder preemption under Go's two-lane run queues: preempted holders go to the global queue, while mutex-woken spinners take `runnext`. See [`ROOT-CAUSE.md`](r1/reject-probe/ROOT-CAUSE.md).
- **Fix:** recon now yields the processor once on every `Reject` refusal (`refused()` in `recon/errors.go`; CONTRACT §4.1). Results, outcomes and counts are unchanged.
- **Measured** (spin shape is now judged by `TestRejectUnderLoad`, 3 of 3 runs passing):

| Spinning callers | v0.2 | recon |
|---:|---:|---:|
| 64 | 285k/s | 1.60M/s (5.6×) |
| 256 | 57–112k/s | **0.997–1.13M/s** (up to 17×) |

At 256 callers, fairness goes from 0.33 to 1.00, and scheduler p99 from 134 to 1.3 ms.
- **Cost:** a refusal takes about 157 ns (v0.2: 1,870 ns). Server-shape refusal latency is p50 about 5 µs and p99 up to 0.6 ms at 1,024 in flight. Successful and uncontended calls are unchanged (59 ns).

Callers should still back off after a refusal; yielding is a mitigation, not backpressure. v1.0's `Backoff` gives callers the tool.

Related: `err_rate` reads "worse" in the DoE tables, because recon refuses correctly where v0.2 blocked (Δ1). Use goodput to judge `Reject` workloads.

## Bugs the lab caught in recon (all fixed before these runs)
1. **Pooled waiter channels broke `testing/synctest`** for any user testing with it. Waiters are now per call: 2 allocs, ~160 B.
2. **A caller with an already-cancelled context could take a key or count as Rejected.** It now returns `ctx.Err()`, matching v0.2.
3. **Three mutation-test gaps** (keyed-entry pooling, `Stats` load order, lowest-invalid-outcome reporting). They are listed in `lab/mutate/results/recon/summary.md`, and the tests to close them are specified there but not yet written.

## Not established by this run
- **Linux, other CPUs, containers with CPU limits.** Everything here is one Windows laptop, and Windows' ~0.6 ms clock steps limit sub-millisecond timing. The lab uses the TSC clock to work around this.
- **Postgres-backed consumer tests.** The 26 mirai-backend tests that need a database skipped for both implementations.
- **A real billable provider.** The render path is mock-only.
- **Production traffic** of any kind.

## Reproduce
```bash
RUN=r2 bash lab/results/run.sh bench          # quiet-machine benchmarks (~12 min)
go run ./lab/cmd/doe -quick                    # 32-run screen (~1.5 min)
go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms
LAB_LEVEL=full LAB_SOAK=60s LAB_OUT="$PWD/lab/results/r2/industrial.jsonl" go test -v -timeout 20m ./lab/industrial
```
