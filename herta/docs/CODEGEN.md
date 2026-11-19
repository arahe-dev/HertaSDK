# Herta → Go code generation (v0.1)

`herta build [-pkg name] <file.he>` parses and contract-checks the program
(see SPEC.md §3), then writes a Go source file to stdout. A program that
fails `herta check` fails `herta build` identically — codegen only runs on
checked programs.

The generated file calls [`hertasdk`](https://github.com/arahe-dev/hertasdk)
directly. It compiles inside whatever package the user puts it in
(`-pkg` selects the package clause) and relies on that package for the
Go names handler bodies reference (result types, helper functions).

## Mapping

| Herta | Generated Go |
|---|---|
| `runtime main { resource db capacity 16 }` | `func NewMainRuntime() (*hertasdk.Runtime, error)` calling `hertasdk.NewRuntime` with the `ResourceSpec`s |
| `op main::charge(user_id: string, cents: u64) -> receipt` | `type MainChargeInput struct { UserId string; Cents uint64 }` plus `func NewMainCharge(rt *hertasdk.Runtime) (*hertasdk.Operation[MainChargeInput, receipt], error)` |
| op clauses | the matching `hertasdk.Policy` fields: `effect`, `admission`, `resources`, `timeout`, `serialize key`, `retry` |
| `handler` | `func mainChargeHandler(ctx context.Context, in MainChargeInput) (receipt, error)` — params are unpacked to local variables, then the body is emitted |

Names are the runtime and op identifiers title-cased per
underscore-separated part (`main::keyed_hold` → `NewMainKeyedHold`). Herta
primitive types map to Go (`u64` → `uint64`, `i32` → `int32`, `f64` →
`float64`, `bytes` → `[]byte`); every other type name passes through
unchanged so it resolves in the user's package.

## Handler bodies

Body statements are opaque text preserved by the parser, emitted verbatim.
The one rewrite is `fail <outcome>, <expr>` — at statement level or inside
an inline block — which becomes

```go
return *new(T), hertasdk.Fail(hertasdk.<Outcome>, <expr>)
```

i.e. exactly the classified-error return the spec gives `fail`. Everything
else in a body is ordinary Go that the surrounding package must make valid.

## Example

`herta/examples/gen/` is the compiled form of `herta/examples/charge.he`:

- `charge_gen.go` — output of `herta build -pkg gen herta/examples/charge.he`
- `support.go` — the host side the body expects (`receipt`, `do_charge`)
- `gen_test.go` — runs the compiled op against hertasdk end to end

## Tests

Generated output is golden-tested in the capped runner format:
`herta/testdata/codegen/<case>/` holds `input.he`, `expected.golden`
(the emitted file), and `args.txt` (`-pkg testpkg`). Run it with

```sh
go run ./herta/cmd/hertatest -cases herta/testdata/codegen \
    -exec "/path/to/herta build {args} {file}"
```
