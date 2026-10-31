# Roadmap

HertaSDK v1.0.0 is the runtime rebuilt from first principles against a
written contract, checked against v0.2.0 by a comparison lab, and proven in
a production pilot. No dates are promised — each future gate below must actually
pass before the next begins.

## Where things stand

- **v1.0.0:** package `hertasdk`, zero dependencies, Go 1.26+, twelve
  v0.2 defects fixed (see [CHANGELOG.md](CHANGELOG.md)). The exported API
  is stable: 1.x releases only add to it.
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
- **Production:** validated in a production pilot.
- **Public module:** `github.com/arahe-dev/hertasdk`; `@latest` is
  `@v1.0.0`.
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
- [x] Zero dependencies; package renamed `hertasdk`
- [x] Validated in a production pilot
- [x] v1.0.0

## Future

- [ ] Additive 1.x features, each only on consumer evidence:
  `Snapshot()`, bounded `Wait` (`MaxWait`, `MaxWaiting`), `Backoff`,
  retryable `Throttled`, an `OutcomeSet` alongside `RetryPolicy.On`, a
  typed `Error` alongside `*Failure`, and opt-in all-or-nothing
  multi-resource acquisition (see [docs/contract.md](docs/contract.md) §9)
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
