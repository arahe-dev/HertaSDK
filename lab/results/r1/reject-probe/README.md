# Reject under load: root-cause probe

This is the evidence behind REPORT.md, "The one weakness". It ran on 2026-10-10 on the same quiet machine as `r1`.

## Files
- `full-flag-fast-reject.patch` — the proposed change to `recon/resource.go` (variant C). It passes recon's full suite under `-race`.
- `*.go.txt` and `impl_tmpl.txt` — the harness, stored as text so the module still builds. To run it, put these files in a scratch module next to copies of `recon/` named `v0`, `va`, `vb` and `vc`, then run `go test -run 'TestStorm|TestStarvation|TestServerShape' -bench BenchmarkReject`. Module setup: `replace github.com/arahe-dev/hertasdk => <repo>`, `GOFLAGS=-mod=mod`.

## Variants

| Name | Change |
|---|---|
| `recon` | as committed |
| A | `tryAcquire` checks an atomic copy of "units free" before taking the lock, updated on every change |
| B | A, plus the gate and counters padded onto separate cache lines |
| C | an atomic "full" flag, stored only when saturation changes; `tryAcquire` returns false without the lock while it is set |

## Measurements

Capacity is 4 and the handler does no work.

### Cost of one rejection (single goroutine)

| | ns per rejection | allocs |
|---|---:|---:|
| v0.2 | 2,030 | 11 |
| recon | 70 | 1 |
| C | 60 | 1 |

### Uncontended and contended `Do` (C vs recon)
- 1 CPU: 59.3 vs 60.2 ns uncontended, 90 vs 90 ns multi-resource, which is noise.
- 16 CPUs: within 3% on Contention and MultiResource; KeyedSerialization is +8%, inside run-to-run noise.

### Server shape: one goroutine per request, which exits after a 429 (successful calls/s)

| In flight | v0.2 | recon | C |
|---:|---:|---:|---:|
| 16 | 1,071k | 1,635k | 1,666k |
| 64 | 1,068k | 1,597k | 1,587k |
| 256 | 1,046k | 1,535k | 1,591k |
| 1,024 | 977k | 1,571k | 1,577k |

### Closed loop: rejected callers retry inside the process (successful calls/s)

| Retry | Clients | v0.2 | recon | C |
|---|---:|---:|---:|---:|
| immediately | 20 | 862k | 1,260k | 1,256k |
| immediately | 64 | 439k | 539k | 45k |
| immediately | 256 | 181k | 75k | 26k |
| after `runtime.Gosched()` | 20 | 1,279k | 1,399k | 1,601k |
| after `runtime.Gosched()` | 64 | 984k | 812k | 1,291k |
| after `runtime.Gosched()` | 256 | 982k | 519k | 999k |

The immediate-retry rows vary up to about 2× between runs. An earlier run gave 330k, 284k and 47k at 64 clients. The pattern was the same in every run.

In the immediate-retry case, successful calls wait 1–25 ms at p99. A successful call should take well under 0.5 ms, which is below this machine's clock resolution. Those waits are Go scheduler time slices.
