# Retrycheck experiment

A small, syntax-only `go/analysis` analyzer for three retry smells from the
demand study: three had multiple independent public examples; the agent
fresh-key example did not add a distinct local pattern. The analyzer is not a
production linter, a Herta language feature, or evidence that users will
install a checker. Its separate module keeps the runtime dependency-free.

| Rule | Candidate | What it does not prove |
|---|---|---|
| RK001 | A UUID or named key generator feeds a recognizable idempotency key setter or field inside a retry-looking `for` or an inline callback to a known retry library. | That the loop repeats the same logical operation or sends the key; misses helpers, queues and model-level re-invocation. |
| RK002 | A default `go-retryablehttp` client submits POST/PATCH without a visible key, or a `Do`/`RoundTrip` method explicitly loops over HTTP sends without a visible method/key branch. | That the destination is non-idempotent or that downstream policy, server dedupe, or configuration does not prevent replay. |
| RK003 | A recognizable webhook handler calls an effect-looking function without a recognizable dedupe call in the handler. | That the call mutates state, the dedupe works, or downstream layers cannot absorb replay. |

## Run

From this directory, using Go 1.26+:

```sh
go test ./...
go build -o /tmp/retrycheck ./cmd/retrycheck
/tmp/retrycheck /path/to/checkout
```

The argument is the root of a clean Git checkout. The driver reports JSON
with the scanned-file count and one diagnostic per matching site; it parses
tracked non-test Go source across all build tags and nested modules without
building it. Vendor/fixture/testdata trees and Go-generated files marked with
the standard generated-code comment are excluded. Examples are included and
must be separated from production code during triage. Dependencies are not
loaded, so named string constants, aliasing, wrappers, middleware, and
cross-file flow mostly escape analysis.

`cohort.tsv` fixes the three public repositories per category and their
commits, plus a known-positive retry-policy control excluded from cohort
metrics. The cohort is purposive, not representative. The experiment groups
hits by rule, reviews each against surrounding code, and counts unsure hits
in the denominator: confirmed true positives / all alerts. Zero alerts
yield no precision measurement and say nothing about recall.
