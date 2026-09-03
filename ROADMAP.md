# Roadmap

HertaSDK's execution model is designed, implemented, and proven internally.
The public repository and Go API are being extracted. No dates are promised —
each gate below must actually pass before the next begins.

## Where things stand

- **Execution model:** designed and proven in an internal Go V0 reference
  implementation (22 proof tests, ~94% statement coverage, race-clean).
- **Public repository:** this shell — concepts, contracts, and extraction
  criteria. No implementation source.
- **Public API extraction:** in progress. Names, signatures, and package
  layout are not final.

## Extraction gates

Each gate is a real validation workload, not a paperwork step:

- [x] Execution model designed
- [x] Internal Go V0 implemented
- [x] Concurrency/lifecycle proof suite (22 tests, race-clean)
- [x] Capacity behavior proven (20 concurrent / capacity 8 → peak exactly 8)
- [ ] Validate against real Mirai Render workload
- [ ] Validate Events and Catalogue workloads
- [ ] External heterogeneous validation workload (outside the original codebase)
- [ ] Extract stable public Go package from the proven core

## Alpha criteria

The public alpha ships only when all of these hold:

1. The extracted package passes the full proof suite unmodified in behavior.
2. At least two heterogeneous real-world workloads run on the public package
   (not the internal reference).
3. The public API is frozen for the alpha series, with documented Effect /
   Outcome contracts and construction-time validation behavior.
4. Supply-chain hygiene is in place: pinned toolchain, verified `go.sum`,
   minimal dependencies, security policy for the module.

Then:

- [ ] Publish Go alpha (module path, version tag, install instructions)
- [ ] Benchmarks against conventional composition (semaphore + retry-lib
  baselines, with published methodology — no numbers without it)

## Beyond alpha (exploratory, not committed)

- [ ] Evaluate Rust/Tower prototype
- [ ] Consider a language-neutral execution contract specification

## Explicitly out of scope

No speculative distributed features: no global resource coordination, no
durable scheduling, no transport, no service mesh behavior. Herta remains
local until evidence proves a distributed layer is necessary. See
[docs/architecture.md](docs/architecture.md).
