# Herta language specification (v0.1)

Herta is a small declarative language for describing **governed operations**:
units of work that consume shared local resources, declare their side-effect
class, and say what failure means. Herta source compiles to Go that calls
[`hertasdk`](https://github.com/arahe-dev/HertaSDK); the compiler checks the
contracts as types, so a contract that `hertasdk.NewRuntime` /
`hertasdk.NewOperation` would reject at run time is a **compile error** in
Herta.

This document is the normative spec for the grammar (`lang` package parser)
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
- A `handler` for an undeclared `op` is an error.
- Handler signature must equal the op signature plus `error`.
- `fail <outcome>` accepts `transient|uncertain|permanent|throttled`
  (`success` is a compile error — you cannot fail successfully).

## 4. What is not checked

Herta does not check handler bodies beyond arity and the `fail` form.
Everything else — serialization bounds, allocation budgets, drain behavior —
is runtime semantics owned by `hertasdk`, not the language. The conformance
harness (separate PR) ports hertasdk's test suite as Herta programs to keep
the two aligned.

## 5. Golden files

`lang/testdata/` holds one `.he` input and `.golden` expected output per
case (parse tree dump or diagnostic list). `go test ./lang/...` compares
byte-for-byte; `-update` regenerates. One command, no fixtures elsewhere.
