# Roadmap

HertaSDK v0.2.0 is implemented, extracted, consumer-validated, and frozen.
No dates are promised — each future gate below must actually pass before the
next begins.

## Where things stand

- **Go V0:** implemented and frozen at v0.1.0; API safety hardened at v0.2.0.
- **Proof suite:** race-clean (22 execution/proof tests plus the heavy race
  hammer; Go race detector enforced in Linux CI).
- **Consumer validation:** render, event-ingestion, and catalogue-replacement
  workloads validated before extraction. CatalogueReplace landed without Herta
  semantic changes.
- **Public module:** `github.com/arahe-dev/hertasdk`, installable via
  `go get github.com/arahe-dev/hertasdk@v0.2.0`.
- **Quickstart:** runnable at `examples/quickstart`
  (`go run ./examples/quickstart`).
- **Benchmarks:** baseline corpus included
  (`BenchmarkContention`, `BenchmarkKeyedSerialization`,
  `BenchmarkMultiResource`, `BenchmarkRetryOverhead`).

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

## Future

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
