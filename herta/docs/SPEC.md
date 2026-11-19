# Herta language specification (v0.1)

Herta is a small declarative language for describing **governed operations**:
units of work that consume shared local resources, declare their side-effect
class, and say what failure means. Herta source compiles to Go that calls
[`hertasdk`](https://github.com/arahe-dev/HertaSDK); the compiler checks the
contracts as types, so a contract that `hertasdk.NewRuntime` /
`hertasdk.NewOperation` would reject at run time is a **compile error** in
Herta.

This document is the normative spec for the grammar (the `lang` package parser)
and the contract checker. It covers the language up to the contract boundary;
Go code generation is specified separately.

## 1. Source files

A program is one or more `.he` files. A file is a sequence of declarations:

- `resource` declarations — named capacity budgets for a runtime;
- `op` declarations — an operation's name, signature, and contract;
- `handler` declarations — the body an operation executes (this PR only
  requires that a handler exists and is well-formed; bodies are checked
  against the operation's contract).

Keywords: `runtime`, `resource`, `op`, `handler`, `contract`, `effect`,
`admission`, `resources`, `timeout`, `serialize`, `retry`, `on`, `pure`,
`idempotent`, `nonidempotent`, `wait`, `reject`, `transient`, `uncertain`,
`permanent`, `throttled`, `success`, `return`, `fail`, `with`, `capacity`,
`units`, `key`, `fn`.

Comments: `//` to end of line. Identifiers: `[A-Za-z_][A-Za-z0-9_]*`.
String literals: `"..."` with Go-style escapes. Integers: decimal, or a
duration literal (`<int>` followed by `ns|us|ms|s|m|h`).

## 2. Declarations

### 2.1 `runtime`

```herta
runtime main {
    resource db      capacity 16
    resource network capacity 4
}
```

Declares a named runtime and its resources. A program may declare several
runtimes; each `op` names the runtime it belongs to.

### 2.2 `op`

```herta
op main::charge(user_id: string, cents: u64) -> receipt {
    effect nonidempotent
    admission reject
    resources { db units 1, network units 1 }
    timeout 500ms
    serialize key user_id
    retry max 3 on { transient }
}
```

Grammar:

```
op        = "op" <runtime> "::" <name> "(" params ")" "->" <type> block
params    = [ <name> ":" <type> { "," <name> ":" <type> } ]
block     = "{" clause* "}"
clause    = effect | admission | resources | timeout | serialize | retry
effect    = "effect" ("pure" | "idempotent" | "nonidempotent")
admission = "admission" ("wait" | "reject")
resources = "resources" "{" <res> "units" <int> { "," <res> "units" <int> } "}"
timeout   = "timeout" <duration>
serialize = "serialize" "key" <param-name>
retry     = "retry" "max" <int> "on" "{" outcome { "," outcome } "}"
outcome   = "transient" | "uncertain" | "permanent" | "throttled" | "success"
```

Clauses may appear in any order; each at most once.

### 2.3 `handler`

```herta
handler main::charge(user_id: string, cents: u64) -> (receipt, error) {
    ...
}
```

A handler binds a Go-callable body to an `op`. Its parameter list and result
must match the op's signature plus a trailing `error`. Handler bodies are
ordinary statements; the only contract-relevant form is `fail <outcome>,
<expr>`, which returns a `hertasdk.Fail(outcome, err)`-classified error.

## 3. Contract checking (compile-time)

The checker rejects a program with an error for each of these — the exact
analogs of `hertasdk`'s construction-time rejection rules (contract §3.1/3.2),
lifted to compile time:

**Runtime checks**
- A runtime with **no resources** is an error.
- An empty resource name is impossible (grammar enforces identifiers).
- A **duplicate resource name** in a runtime is an error.
- `capacity` must be **> 0**.

**Operation checks** (in declaration order; the first error per op is
reported, matching hertasdk's deterministic order — retry-policy errors
before unsafe-retry errors):
- An `op` naming an **undeclared runtime** is an error.
- Missing `effect` clause is an error (`ErrEffectRequired` analog).
- An `effect` value outside `pure|idempotent|nonidempotent` is a parse
  error.
- A `resources` entry naming a resource **not declared** on the op's
  runtime is an error.
- `units` must be **> 0** and **≤ that resource's capacity**.
- The same resource may not appear **twice** in one `resources` clause.
- A `serialize key` naming a parameter **not in the signature** is an
  error; the parameter's type must be `string` (the keyed-mutex key).
- `admission` values outside `wait|reject` are a parse error.
- `timeout` must be **> 0**.
- `retry max` must be **≥ 1**.
- A `retry on` set containing `success`, `permanent`, or `throttled` is
  an `ErrInvalidRetryPolicy` analog.
- `retry on { uncertain }` combined with `effect nonidempotent` is an
  `ErrUnsafeRetry` analog.

**Handler checks**
- An `op` with **no handler** is an error — an operation cannot be
  constructed without its body.
- A `handler` for an undeclared `op`, or a second handler for the same
  `op`, is an error.
- Handler signature must equal the op signature plus `error`.
- `fail <outcome>` accepts `transient|uncertain|permanent|throttled`
  (`success` is a compile error — you cannot fail successfully), wherever
  the `fail` statement appears in the body.

## 4. Composition prototype (checker only)

The prototype additionally accepts these op-block clauses, in any order:

```
calls a, b
lock ledger
lock cache shared
dedupe key request_id
detached
detached for cleanup // also shutdown or background
budget 20
classify { ErrCommitted => uncertain, ErrBusy => transient, _ => permanent }
```

`lock` may repeat for different locks; other clauses occur at most once.
`dedupe key` must name an input parameter and asserts replay safety; it does
not implement storage. `budget` is a positive int64 attempt-product limit;
only the outermost op's override applies (default 10). Retry `max` continues
to mean total attempts, not additional retries. Classifier arms are ordered,
first match wins, with `_` matching anything; allowed classes are the existing
outcomes. A top-level `error ErrCommitted ambiguous` declares that the error
may occur after commit. Classification and dedupe assertions are checker
metadata, not a generated runtime implementation.

Handler `call debit(user)` records a positional call edge. Raw Go calls on
an op-typed parameter, e.g. `debit: debitOp` followed by
`debit.Do(ctx, MainDebitInput{User: user})`, also record edges, tolerating
arbitrary whitespace and multiline calls. `calls debit` supplies edges for
opaque bodies. Callees resolve in the caller's runtime. Calls are conservative:
branches are all considered reachable, with no general Go type/alias analysis.
Quoted strings and comments do not create edges. Explicit `call` syntax is
checker-only; the Go generator does not lower it.

Serialize namespaces are **runtime + key parameter name** by default. Simple
identifier forwarding in positional `call` arguments or Go input struct fields
(case/underscore-insensitive field names) preserves a caller parameter's
namespace even if the callee parameter is renamed. Reentering the identical
op is always unsafe. No prefix-expression inference or arbitrary Go dataflow
is attempted. Named locks are process-wide. Resources are runtime-scoped;
units are summed along a nested call path, not across sequential siblings.

Every op is checked as a possible entry point. Effective effects propagate
through calls. Ancestor deadlines propagate until an explicitly detached
boundary. Every call is assumed to pass a potentially cancellable context;
thus any unowned, non-pure detached callee is rejected even when the caller
has no declared timeout. Owned detached operations opt out of inherited
deadlines, but do not release ancestor locks or resource units.

Composition diagnostics include a call path and remediation:

| Code | Rejection |
| --- | --- |
| HC001 | Serialize-key reacquisition |
| HC002 | Lock reacquisition (shared/shared warns of writer-starvation deadlock) |
| HC003 | Nested resource units exceed capacity |
| HC004 | Pure/idempotent declaration hides a nonidempotent effective effect |
| HC005 | Unsafe retries of propagated nonidempotent effects or classified ambiguous errors |
| HC006 | Retry attempt product exceeds outermost budget |
| HC007 | Callee timeout exceeds an inherited deadline |
| HC008 | Unowned non-pure detached callee drops cancellation |
| HC009 | Classification lacks `_` fallback |
| HC010 | Transient arm precedes uncertain arm (joined-error hazard) |
| HC011 | Ambiguous error or `_` mapped transient with nonidempotent effect |
| HC012 | Call cycle (traversal stops at the repeated op) |
| HC013 | Unresolved callee |
| HC014 | Dedupe key is not an input parameter |
| HC015 | Traversal exceeds 10,000 path visits from an entry point |

The preexisting local nonidempotent/uncertain diagnostic remains unchanged
for compatibility; a valid dedupe assertion suppresses that local rejection.
Cycles also participate in effect propagation before path traversal stops.

## 5. Closed mode (prototype)

`herta check -closed file.he` and `herta build -closed [-pkg name] file.he`
opt into closed-world contracts. The default remains unchanged. Bodies remain
ordinary Go, not a restricted body language. Closed builds set `Policy.Closed`
on every generated operation and snapshot its fully qualified declared callees
from `calls`, checker-only `call` expressions, and typed parameter `.Do` edges.
Opaque helper calls must declare their edges with `calls`.

`classify { ErrTimeout => uncertain, ErrUnavailable => transient, _ => permanent }`
lowers to an authoritative `Policy.Classify` function: ordered `errors.Is`
checks followed by the fallback. Sentinel names are Go identifiers resolved by
the surrounding package; missing names fail Go compilation. First match wins,
including `_`. Error declarations annotate ambiguity; they do not define Go
sentinel values. Helpers return errors, not outcome tags. A helper's or child
operation's `*hertasdk.Failure` tag is ignored by this classifier, but its error
chain remains visible to `errors.Is`. Nil errors remain successful. Caller
cancellation wins; an expired attempt-owned deadline remains Uncertain even
when a classifier would choose Transient or Permanent. Invalid classifier
outcomes (including Success for a non-nil error) normalize to Permanent.
Without `Policy.Classify`, SDK classification is unchanged. An operation
without retries may omit `classify` and retain that legacy classification.

Closed checks add HC016 for **any** retry clause without `classify` and HC018
for direct handler constructions of `hertasdk.Fail`, `&hertasdk.Failure`, or
`fail outcome, error`. Missing classifier fallback remains HC009; HC001–HC015
retain their existing meanings. Helpers are not inspected for outcome tags;
the authoritative classifier makes those tags irrelevant.

Before admission, SDK `.Do` checks the parent frame on the supplied context.
A closed parent's undeclared callee returns a Permanent `ErrUndeclaredCall`
with `parent -> callee` names, without running or acquiring resources. A key
already held by an ancestor on the same runtime returns Permanent
`ErrKeyReacquired` rather than waiting on itself. Empty keys hold no lock;
identical key strings on different runtimes are independent. Frames carry the
operation name, held key, and parent through handler-derived contexts, starting
at a closed operation and continuing through its descendants. Open-only call
trees retain the previous behavior and allocation profile.

This is not a Go sandbox or a whole-program proof. Calls on unrelated contexts,
such as `context.Background()`, are **not seen**. Host code must propagate the
handler context; asynchronous work must finish before the handler returns so
its frame still describes held keys. Callee authorization is by operation name,
not an unforgeable capability. Codegen still exposes ordinary SDK operations.

### Executable plans and dispatch evidence

`call name(args)` executes: in a runtime whose handlers use it, codegen emits
a `<Runtime>Plan` and lowers each call to the callee's `Do` with positional
arguments bound to the callee's parameters. A statement that is only a call
returns the callee's error. The clause `evidence dispatch` sets
`Policy.TrackDispatch`; the handler statement `dispatch` marks the send point
(`hertasdk.Dispatch`). After it, `Transient` is reported and treated as
`Uncertain`. Tracking propagates to every operation that can reach a tracked
one. See [SYNTHESIS.md](SYNTHESIS.md).

## 6. What is not checked

Outside the composition forms above, Herta does not type-check handler Go.
Everything else — serialization bounds, allocation budgets, drain behavior —
is runtime semantics owned by `hertasdk`, not the language. The conformance
harness (separate PR) ports hertasdk's test suite as Herta programs to keep
the two aligned.

## 7. Golden files

`lang/testdata/` (relative to the language root) holds one `.he` input and `.golden` expected output per
case (parse tree dump or diagnostic list). `go test ./lang/...` compares
byte-for-byte; `-update` regenerates. One command, no fixtures elsewhere.
