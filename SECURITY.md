# Security Policy

HertaSDK v0.1.0 is a released Go module (`github.com/arahe-dev/hertasdk`).
This policy covers the repository, the module, and the supply chain around
them.

## Supported versions

| Version | Supported |
|---|---|
| v0.1.0 | Supported |

Security fixes are tracked against the latest tagged release.

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
- Supply-chain hygiene: pinned toolchain via `go.mod`, verified `go.sum`,
  minimal dependencies (only `golang.org/x/sync`).
