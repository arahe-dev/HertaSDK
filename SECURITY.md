# Security Policy

HertaSDK is a Go module (`github.com/arahe-dev/hertasdk`).
This policy covers the repository, the module, and the supply chain around
them.

## Supported versions

| Version | Supported |
|---|---|
| v1.0.0-next | Supported: the v1.0 candidate |
| v0.2.0 | Supported until v1.0 is tagged |
| v0.1.0 | Unsupported: an omitted `Effect` silently meant `Pure` |

Security fixes land on `main` and are noted in the next release.

## Reporting a vulnerability

Email **araheemimami@gmail.com** with:

- A description of the issue and its potential impact.
- Steps to reproduce, or the document/diagram affected (for repository issues).
- Any suggested mitigation, if known.

What to expect:

- Acknowledgment within **5 business days**.
- A private fix or correction before any public disclosure, where the issue
  affects the module or future implementation guidance.
- Credit in the fix notes if you want it.

Please do not open a public issue for a suspected vulnerability — use email
first.

## Scope notes

- HertaSDK is an in-process runtime. Its threat model concerns local
  resource exhaustion, contract misuse, and unsafe retry semantics — not
  network attack surface. There is deliberately no network code in scope.
- Supply-chain hygiene: the module has no dependencies beyond the Go
  standard library (Go 1.26 or later), so there is no `go.sum`. The lab
  module, which does not ship, depends only on `golang.org/x/sync` for its
  frozen v0.2.0 reference.
