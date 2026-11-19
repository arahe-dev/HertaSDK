# hertatest — capped golden-file runner

One command runs the whole Herta conformance suite:

```sh
go run ./herta/cmd/hertatest -cases herta/testdata/cases -exec selftest
```

`selftest` echoes each `input.he` and is what CI uses until the real
checker lands. Point `-exec` at any command template once it exists:

```sh
hertatest -exec "go run ./herta/cmd/herta check {file} {args}"
```

## Case layout

`herta/testdata/cases/<name>/`:

| file | required | meaning |
|---|---|---|
| `input.he` | yes | program under test |
| `expected.golden` | yes | expected stdout of the command |
| `expected.exit` | no | expected non-zero exit code |
| `args.txt` | no | extra args appended to the command |
| `skip` | no | presence skips the case |

## Caps

The suite stays small by construction. `Discover` fails the run when any cap
is exceeded — raising them is a deliberate edit, not something iterations do
quietly:

- 200 cases (`-max-cases`)
- 5 s per case (`-case-timeout`)
- 60 s total (`-timeout`)
- 64 KiB per case file
- 1 MiB captured output per case

## Updating goldens

`expected.golden` is compared byte-for-byte to stdout. Regenerate by running the
exec command on `input.he` and saving the output — or copy the `--- got ---`
block out of a failure diff.
