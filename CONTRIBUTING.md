# Contributing to HertaSDK

Thanks for your interest. HertaSDK is a stable (v1.0) installable Go
module (`go get github.com/arahe-dev/hertasdk@v1.0.0`); its
execution contract is [docs/contract.md](docs/contract.md). See
[README.md](README.md), [ROADMAP.md](ROADMAP.md), and
[CHANGELOG.md](CHANGELOG.md).

## What is useful right now

- **Real-world consumer evidence.** Runnable workloads against the public
  package ("here is how my service declares its contracts") are the most
  valuable input — they gate every future contract change.
- **Documentation.** Typo fixes, clarifications, and better explanations are
  welcome. Keep the tone technical and precise; avoid marketing language.
  Ground claims in the current API and docs/contract.md; do not invent
  features.
- **Benchmark corpus.** Additional benchmarks are welcome when they come with
  methodology — no performance numbers without it.

## What is not useful

- Contract redesigns without consumer evidence. The contract changes only
  with evidence; Queue, priority admission, dynamic resource profiles, distributed
  coordination, durability, and transports are explicitly out of scope until a
  real consumer proves they are needed (see "Explicit non-goals" in
  [ROADMAP.md](ROADMAP.md)).
- Parallel runtime reimplementations. Behavioral changes must come with a
  contract change and the lab evidence for it.
- Feature requests for distributed coordination, durability, transports, or
  other out-of-scope concerns (see "What Herta is NOT" in the README).

## How to contribute

1. **Issues first.** For anything beyond a typo fix, open an issue describing
   the problem, the workload, and what you expected. Use the provided issue
   templates.
2. **Small, focused PRs.** One concern per PR. Docs PRs should keep both
   English quality and link validity.
3. **Do not change execution semantics silently.** Runtime changes must
   preserve [docs/contract.md](docs/contract.md), or change it explicitly.
   `go test -race -count=1 .` must pass with 100% coverage, and so must the
   lab's quick tier (`cd lab && go test -race -count=1 ./...`).

## Style

- Go code samples must compile against the current `hertasdk` API.
- Diagrams are hand-maintained SVG with light and dark variants, or Mermaid
  flowcharts; keep both readable in GitHub light and dark mode.
- Keep claims factual and verifiable. No adoption claims, no benchmarks
  without methodology, no superlatives.

## Code of conduct

By participating, you agree to uphold the [Code of Conduct](CODE_OF_CONDUCT.md).
