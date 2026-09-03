# Security Policy

HertaSDK is currently **pre-alpha** with no released package. This policy
covers the repository itself (docs, assets, workflows) and sets expectations
for the future public package.

## Supported versions

There are no supported releases yet. Security fixes will be tracked against
the upcoming public alpha once it exists; until then, advisories do not apply
to any installable artifact because none exists.

| Version | Supported |
|---|---|
| (pre-alpha, unreleased) | N/A — no installable package |

## Reporting a vulnerability

Email **araheemimami@gmail.com** with:

- A description of the issue and its potential impact.
- Steps to reproduce, or the document/diagram affected (for repository issues).
- Any suggested mitigation, if known.

What to expect:

- Acknowledgment within **5 business days**.
- A private fix or correction before any public disclosure, where the issue
  affects future implementation guidance.
- Credit in the fix notes if you want it.

Please do not open a public issue for a suspected vulnerability — use email
first.

## Scope notes

- HertaSDK is an in-process runtime. Its future threat model concerns local
  resource exhaustion, contract misuse, and unsafe retry semantics — not
  network attack surface. There is deliberately no network code in scope.
- Dependency and supply-chain hygiene will be documented when the Go module
  is published (pinned toolchain, `go.sum` verification, minimal
  dependencies).
