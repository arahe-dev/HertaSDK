# Trust ledger

See [Lean lessons for Herta](LEAN.md) for the formal model's motivation, results,
evaluation limits and evidence required before expanding the checker.

```sh
go run ./herta/cmd/herta trust [-closed] [-json] [-strict] file.he
```

Like Lean's `#print axioms`, this lists dependencies, not bug verdicts or
proofs. Each JSON entry includes kind, required/informational status, theorem
names from `herta/formal/Herta.lean`, origin operation and a canonical shortest
call path. Text/JSON order is deterministic. A dependency is propagated once
per origin/kind/detail, even through cycles and diamonds.

`-closed` must match `build -closed`: only then are declared `classify` arms
installed as Policy.Classify. Without it **even a classify clause is legacy**.
With it, an operation lacking classify still uses handler tags (and retrying
ones receive HC016 from `check -closed`). Trust analyzes parsed programs even
when contract checks fail; it does not replace `check`.

| Kind | Meaning | Formal link |
|---|---|---|
| opaque-handler | Go expressions/helpers and effect/call declarations remain unchecked | T1, T4_composition |
| transient-honest | Transient attempts must not apply effects; required for nonidempotent, informational otherwise | T1 |
| legacy-tag-path | Go handler tags may override own-deadline promotion | T2, T5 |
| nested-retry-amplification | Pure/idempotent retrying parent reaches a nonidempotent callee | T4_composition |
| same-serialize-key | existing HC001 key-reacquisition diagnostic, also shown as a trust dependency | T4_key |
| unresolved-callee | a visible/declared edge cannot be inspected | T4_composition |

All current handlers retain opaque Go expressions (even a DSL `fail` argument
is opaque), so opaque-handler is always required. An op with no other dependency
prints `trust: opaque-handler only` with required status and theorem links.
Legacy tags are conservatively flagged even on nonretrying operations; this is
a path/trust warning, not a claim that a retry actually occurs. Transient honesty
is listed only with max attempts >1 and retry-on-transient, plus a reachable
Transient arm in closed mode. In legacy mode opaque Go may supply such a tag.

Default exit is 0 regardless of assumptions or contract diagnostics. `-strict`
returns 1 if any required assumption exists: currently every nonempty op set
fails, intentionally. An empty program passes strict and JSON prints `[]`.
Malformed source returns 1; usage/read/write errors return 2, as elsewhere in
the CLI. Informational exit behavior does not disguise inability to read/parse.

Propagation uses declared `calls` and the same visible DSL/typed-Op edges as
codegen (`lang.Callees`). Arbitrary Go service methods can hide nested calls;
opaque-handler records that blind spot. Key analysis reuses HC001, not guesses
that different parameter names imply disjoint runtime key values. No ledger
claim verifies helper error tags, actual external effects, classifier ordering,
missing retries, runtime string-key disjointness, or dynamic behavior. See
`herta/eval/trust/summary.md` for observed coverage and causal misses.
