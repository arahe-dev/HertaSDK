# Roadmap

HertaSDK 1.0-next, the v1.0 candidate, is the runtime rebuilt from first
principles against a written contract, and checked against v0.2.0 by a
comparison lab. No dates are promised — each future gate below must actually
pass before the next begins.

## Where things stand

- **1.0-next:** package `hertasdk`, zero dependencies, Go 1.26+, twelve
  v0.2 defects fixed (see [CHANGELOG.md](CHANGELOG.md)); being hardened
  before it is locked as v1.0.
- **Contract:** normative in [docs/contract.md](docs/contract.md).
- **Proof suite:** 75 tests, 100% statement coverage, race-clean; v0.2.0's
  own suite passes against `hertasdk`; 100% mutation score; Go race detector
  enforced in CI.
- **Lab:** v0.2.0 frozen in `lab/v02`, with differential (100,000
  scenarios), chaos, industrial and design-of-experiments suites
  ([lab/results/r2](lab/results/r2/REPORT.md)).
- **Consumer validation:** render, event-ingestion, and catalogue-replacement
  workloads validated before extraction. CatalogueReplace landed without Herta
  semantic changes.
- **Public module:** `github.com/arahe-dev/hertasdk`; `@v1.0.0-next` is
  1.0-next (a pre-release, so `@latest` is still `@v0.2.0`).
- **Quickstart:** runnable at `examples/quickstart`
  (`go run ./examples/quickstart`).
- **Benchmarks:** `BenchmarkContention`, `BenchmarkKeyedSerialization`,
  `BenchmarkMultiResource`, `BenchmarkRetryOverhead` and the uncontended
  `BenchmarkDoUncontended{,Timeout,Keyed}`, compared against v0.2.0 with
  published methodology.

## Completed

- [x] Go V0 implemented
- [x] Race-clean proof suite
- [x] Render workload validation
- [x] Events workload validation
- [x] CatalogueReplace third-consumer validation
- [x] Stable public Go extraction
- [x] Runnable quickstart
- [x] Baseline benchmarks
- [x] v0.1.0
- [x] v0.2.0 API safety hardening (Effect must be declared; explicit enum/retry-policy validation)
- [x] First-principles reconstruction against a written contract
- [x] Comparison lab against v0.2.0 (differential, chaos, mutation, DoE, industrial)
- [x] Zero dependencies; package renamed `hertasdk` (1.0-next)

## Future

- [ ] Harden 1.0-next and lock it as **v1.0**, after a real consumer's
  database-backed suite passes on `hertasdk` (so far Postgres-backed
  consumer tests have only been skipped, not run)
- [ ] Additional real-world consumers
- [ ] Improve benchmark corpus when evidence warrants
- [ ] Evaluate Rust/Tower prototype
- [ ] Evaluate language-neutral spec only after multi-language evidence

## Explicit non-goals

- Queue without consumer evidence
- Distributed Herta
- Durable workflow execution
- Transport ownership

No speculative distributed features: no global resource coordination, no
durable scheduling, no transport, no service mesh behavior. Herta remains
local until evidence proves a distributed layer is necessary. See
[docs/architecture.md](docs/architecture.md).
