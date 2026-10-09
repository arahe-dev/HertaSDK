# Release candidate vs v0.2.0: verification run r2

Run 2026-10-10 against recon at `5e771af` (including the fast-reject flag, the refusal yield and the nil-`*Failure` fix), on a quiet machine: go1.27.1 windows/amd64, i7-13650HX, 20 threads. Raw data is in this directory.

## Verdict

recon is **correct**: every gate below passes, mutation score is 100%, and the real consumer behaves identically.

It is **better than v0.2.0 on every measured axis** where both follow the same contract. The one cell where recon looks worse is keyed `Reject`, and that is the deliberate Δ1 fix: v0.2.0 blocks on a held key instead of refusing.

## 1. Correctness

| Gate | Result |
|---|---|
| v0.2.0's own test files, byte-identical, run against recon | pass under `-race` |
| recon's suite (75 tests) | pass under `-race` (×3), **100% statement coverage** |
| Differential against the reference model, 100,000 scenarios | pass (308 s) |
| Invariants under load (30 rounds) and chaos (100 rounds) | pass (2,131 s and 168 s) |
| Mirai-shaped soak, 5 min per implementation | pass |
| mirai-backend's suite with recon in place of its fork (`-race`, ×2) | identical to v0.2.0; only DB and live-bridge tests skip, as with v0.2.0 |
| Mutation testing | **100%: 394/394 killed**, 15 proven equivalent (v0.2.0: 238/238) |
| `go vet`, gofmt, every lab package (`-race`) | clean, all pass |

Logs: [`correctness/`](correctness/). Mutation detail: `lab/mutate/results/recon/summary.md`.

## 2. Per call: benchmarks

10 rounds, alternating which implementation runs first. All 72 comparisons are better, at p < 0.001 ([`bench.md`](bench.md)).

| Path | v0.2.0 | recon |
|---|---:|---:|
| `Do`, uncontended | 893 ns · 464 B · 10 allocs | **59 ns · 0 B · 0 allocs** |
| `Do`, with timeout | 1,465 ns · 14 allocs | **350 ns · 4 allocs** (= `context.WithTimeout`) |
| `Do`, keyed | 1,107 ns · 13 allocs | **103 ns · 0 allocs** |
| Contention (1 / 16 CPUs) | 1,402 / 1,627 ns | **63 / 829 ns** |
| Keyed serialization (1 / 16 CPUs) | 1,687 / 1,313 ns | **108 / 479 ns** |
| Multi-resource (1 / 16 CPUs) | 1,534 / 1,782 ns | **88 / 938 ns** |

## 3. Under load: design of experiments

### 32-run resolution-IV screen

10 factors × implementation × 3 replicates, realistic driver ([`screen-quick/`](screen-quick/REPORT.md)).

| Response | v0.2.0 | recon | Change [95% CI] |
|---|---:|---:|---|
| throughput | 243k/s | 417k/s | ×1.72 [1.69, 1.74] |
| goodput | 155k/s | 198k/s | ×1.28 [1.27, 1.30] |
| latency p50 / p99 / p99.9 | 46.8 µs / 889 µs / 1.92 ms | 28.5 µs / 346 µs / 741 µs | ×0.61 / ×0.39 / ×0.39 |
| admission wait p99 | 582 µs | 114 µs | ×0.20 |
| allocs, bytes per call | 14.7 · 723 B | 2.6 · 177 B | −12.1 · −545 B |
| peak extra goroutines | 534 | 2.6 | −531 |
| drain | 118 µs | 75 µs | ×0.64 |
| capacity / key violations | 0 / 0 | 0 / 0 | same |
| error rate | 19.7% | 30.5% | worse (see below) |

### Full factorial: clients × admission × keys

5 replicates, goodput ratio recon / v0.2.0 ([`reject-realistic/`](reject-realistic/REPORT.md), [`reject-spin/`](reject-spin/REPORT.md)).

| Cell | Realistic driver | Original spin driver | p99, v0.2.0 → recon (realistic) |
|---|---:|---:|---|
| Reject · 8 clients | 2.25× | 2.19× | 31 → 8 µs |
| Reject · 8 · keys | 1.97× | 1.84× | 74 → 61 µs |
| **Reject · 256 clients** (r1: 0.47×) | **1.63×** | **7.35×** | 1,335 → 73 µs |
| Reject · 256 · keys | 0.33× | 0.36× | Δ1: v0.2.0 never refuses here |
| Wait · 8 / 8 · keys | 1.57× / 1.82× | 1.59× / 1.68× | 58 → 45 / 71 → 33 µs |
| Wait · 256 / 256 · keys | 1.75× / 1.51× | 1.71× / 1.48× | 1,384 → 481 / 2,441 → 954 µs |

**Why error rate and keyed `Reject` read "worse".** v0.2.0's keyed `Reject` blocks on a held key (the Δ1 bug), so it almost never refuses: about 13 refusals/s, against 1.4 M/s for recon. Every recon refusal there is a correct "key held" refusal. Like for like, recon's keyed `Wait` is about 1.9× v0.2.0's keyed "Reject" ([`r1/reject-probe/`](../r1/reject-probe/README.md)).

## 4. Industrial tier

| Test | v0.2.0 | recon |
|---|---:|---:|
| goroutines per `Do` (contended / uncontended) | 1 / 1 | **0 / 0** |
| per parked waiter: heap · stack · allocs | 848 B · 7.7 KB · 14.9 | **157 B · 226 B · 2.0** |
| 1M unique keys: keyed calls/s | 0.71 M | **3.03 M** |
| 1M unique keys: peak extra heap | 41.5 MB | **6.0 MB** |
| 1M unique keys: retained after | 1.4 MB | **0** |
| drain under load, p50 / max | 2.6 / 4.1 ms | **2.2 / 2.7 ms** |
| 10× spike: backlog cleared | 101.5 ms | 101.1 ms |
| 10× spike: peak extra goroutines | 2,142 | **104** |
| 60 s soak: db-write p99, early → late | 3.98 → 4.11 ms | 3.92 → 4.11 ms |
| 60 s soak: heap, goroutines | 73.8 MB, 24 | 73.8 MB, **13** |

**`Reject` under load, successful calls/s** (`TestRejectUnderLoad`, full tier, 16 → 1,024 callers):

| Callers | v0.2.0 | recon |
|---|---:|---:|
| server shape | 1.08–1.16 M | **1.44–1.68 M** |
| yielding | 0.65–0.95 M | **1.63–2.04 M** |
| spinning | 69k–910k | **0.97–1.63 M** (×14.7 at 1,024 callers) |

## 5. Size

| | v0.2.0 | recon |
|---|---:|---:|
| non-test source | 1,050 lines + `golang.org/x/sync/semaphore` | 1,080 lines, standard library only |
| tests | 25 | 75 |
| statement coverage | 93.6% | 100% |

## Limits of this verification

- Everything ran on one Windows machine. Linux and other CPUs are untested.
- The 26 Postgres-backed consumer tests and one live-bridge test skip, because neither a database nor a running bridge is available here.
- The render provider is still a mock, and none of this is production traffic.
