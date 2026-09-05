# Contributing to HertaSDK

Thanks for your interest. HertaSDK v0.1.0 is an installable Go module
(`go get github.com/arahe-dev/hertasdk@v0.1.0`) with a frozen V0 execution
contract — see [README.md](README.md), [ROADMAP.md](ROADMAP.md), and
[CHANGELOG.md](CHANGELOG.md).

## What is useful right now

- **Real-world consumer evidence.** Runnable workloads against the public
  package ("here is how my service declares its contracts") are the most
  valuable input — they gate every future contract change.
- **Documentation.** Typo fixes, clarifications, and better explanations are
  welcome. Keep the tone technical and precise; avoid marketing language.
  Ground claims in the frozen v0.1.0 API; do not invent features.
- **Benchmark corpus.** Additional benchmarks are welcome when they come with
  methodology — no performance numbers without it.

## What is not useful

- Contract redesigns without consumer evidence. The V0 contract is frozen;
  Queue, priority admission, dynamic resource profiles, distributed
  coordination, durability, and transports are explicitly out of scope until a
  real consumer proves they are needed (see "Explicit non-goals" in
  [ROADMAP.md](ROADMAP.md)).
- Parallel runtime reimplementations. The frozen runtime is authoritative;
  behavioral changes will not be merged.
- Feature requests for distributed coordination, durability, transports, or
  other out-of-scope concerns (see "What Herta is NOT" in the README).

## How to contribute

1. **Issues first.** For anything beyond a typo fix, open an issue describing
   the problem, the workload, and what you expected. Use the provided issue
   templates.
2. **Small, focused PRs.** One concern per PR. Docs PRs should keep both
   English quality and link validity.
3. **Do not change execution semantics.** Runtime changes must preserve the
   frozen V0 contract; `go test -count=1 ./...` and
   `go test -race -count=1 ./...` must pass.

## Style

- Go code samples must compile against the released v0.1.0 API.
- Diagrams are hand-maintained SVG with light and dark variants; Mermaid is
  not used for README diagrams.
- Keep claims factual and verifiable. No adoption claims, no benchmarks
  without methodology, no superlatives.

## Code of conduct

By participating, you agree to uphold the [Code of Conduct](CODE_OF_CONDUCT.md).
