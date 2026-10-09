# v0.2-recon vs v0.2.0: results

Run `r1`, 2026-10-09, go1.27.1 windows/amd64, i7-13650HX (20 threads), on an otherwise idle machine. Raw data is in [`r1/`](r1/).

## Conclusion

**recon is a drop-in replacement for v0.2.0.**
- It behaves the same everywhere except the 11 documented fixes.
- It is faster and leaner in every measured regime but one.
- That one regime is a narrow, understood weakness with a small fix.

| Question | Answer | Evidence |
|---|---|---|
| Same behavior? | **Yes**, apart from CONTRACT §8 (Δ1–Δ11) | v0.2's 25 tests pass byte-for-byte unchanged; 100k-scenario differential vs reference model; mirai-backend suite identical with recon swapped in |
| Faster? | **Yes, 2–17× per call** | 72/72 benchmark comparisons better, all p < 0.001 |
| Lighter? | **Yes** | 0 vs 10 allocs per call; 0 vs 1 goroutine per call; 159 B vs 860 B heap per waiter |
| Better under load? | **Yes, except a tight retry loop on `Reject`** | DoE: throughput ×2.2, p99.9 ×0.49, fairness up; see the weakness below |
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

## The one weakness: `Reject` under a tight retry loop

**Finding.** Full factorial over clients × admission × keys, 5 replicates (`r1/reject-interaction/`). With `Reject` and 256 clients that retry *immediately*, recon's goodput is **0.47×** v0.2's. Every `Wait` cell favors recon: 1.38–1.72× goodput and up to 2.6× better p99.

**Mechanism** (confirmed by probe and mutex profile; `scratchpad/rejectprobe`):
- recon rejects 2.3× faster (3.8 M/s vs 1.7 M/s).
- Every rejection takes the budget's mutex to learn "no room".
- 98% of lock contention is in `admit → tryAcquire`, which starves the `release` calls of the successful work.

v0.2 is slower per rejection, and that slowness acts as accidental backoff.

**Scope.** It only occurs with zero backoff.
- Give a rejected client any pause (≤ 0.6 ms here, limited by the Windows clock) and recon's goodput is **3–10× v0.2's** (0.44–0.82 M/s vs 0.07–0.13 M/s).
- Real `Reject` callers return 429/503 to a remote client, so the tight loop is a lab artifact. It is still a real throughput cliff.

**Fix (v0.3, ~10 lines).** Add a lock-free fast-reject: keep an atomic mirror of in-use units and queue length, and let `tryAcquire` return `false` without locking when the request cannot fit. The mutex is then taken only by calls that can actually be granted. Re-run `r1/reject-interaction` to confirm.

Related: `err_rate` reads "worse" in every DoE table. That is this effect plus Δ1 (keyed `Reject` now fails fast instead of blocking). Use goodput, not `err_rate`, to judge `Reject` workloads.

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
