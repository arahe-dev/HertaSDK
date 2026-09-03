# Contributing to HertaSDK

Thanks for your interest. HertaSDK is currently **pre-alpha**: the execution
model is proven internally and the public Go package is being extracted. There
is no installable module yet, so contributions at this stage are mostly
discussion, specification review, and documentation.

## What is useful right now

- **Design discussion.** Open an issue about the execution model, failure
  semantics, resource arbitration, or the conceptual API. Concrete workloads
  ("here is how my service would declare its contracts") are the most valuable
  input to extraction.
- **Documentation.** Typo fixes, clarifications, and better explanations are
  welcome. Keep the tone technical and precise; avoid marketing language.
- **Extraction validation.** Once the public package lands, real-world consumer
  validation is the gating factor for alpha — see [ROADMAP.md](ROADMAP.md).

## What is not useful yet

- Pull requests implementing the runtime from scratch. The reference
  implementation exists internally and will be extracted deliberately; parallel
  implementations will not be merged.
- Benchmarks against a package that does not exist.
- Feature requests for distributed coordination, durability, transports, or
  other out-of-scope concerns (see "What Herta is NOT" in the README).

## How to contribute

1. **Issues first.** For anything beyond a typo fix, open an issue describing
   the problem, the workload, and what you expected. Use the provided issue
   templates.
2. **Small, focused PRs.** One concern per PR. Docs PRs should keep both
   English quality and link validity.
3. **No implementation source yet.** Do not submit files under future package
   paths (`herta/`, `pkg/`, `internal/`). There is intentionally no source tree
   in this repository.

## Style

- Go code samples are conceptual only and must be labeled as such until the
  public API is released.
- Diagrams are hand-maintained SVG with light and dark variants; Mermaid is
  not used for README diagrams.
- Keep claims factual and verifiable. No adoption claims, no benchmarks
  without methodology, no superlatives.

## Code of conduct

By participating, you agree to uphold the [Code of Conduct](CODE_OF_CONDUCT.md).
