# Six-agent benchmark evaluation

Original submissions are extracted under `<run>/bundle-go` or
`<run>/bundle-herta`. Submissions were not edited. Every `score.sh` invocation
works in a separate persistent copy, and repetition/race checks use isolated
copies too.

## Outputs and definitions

- `summary.csv`: original-oracle results, with one additional `design` column.
  `oracle_runs_passed` is the number of successful repetitions out of ten; the
  initial score.sh run and separate race run are not included in that count.
  All original-oracle task results passed all ten repetitions and the race run.
- `changed_nontest_lines`: additions plus deletions against the matching starting
  bundle's task directory, excluding `*_test.go`; generated Go and Herta source
  both count. It is not a count of distinct modified lines.
- `added_test_lines`: total lines in task test files other than smoke_test.go,
  measured before injecting either oracle.
- `<run>/score.log`: original score.sh output.
- `<run>/supplied-policy.diff`: policy-only diff for supplied t3 Charge and t4
  Debit/Credit. Whitespace-only changes are ignored. Changes are reported, not
  counted as failures.
- `<run>/oracle*-count10.jsonl`, `*-race.log`, and `*-result.json`: raw test
  evidence. A failing race invocation can mean an assertion failed; none of
  these invocations reported a data race.
- `oracle2_summary.csv`: harder-oracle repetitions and race results, including
  original references and corrected reference controls.

## Supplied operations

All six preserve the supplied t3 Charge policy. Only these supplied t4 policies
changed:

| Run | Debit/Credit serialization key |
| --- | --- |
| go-1 | raw user -> `account:` + user |
| herta-1 | raw user -> separate account_key parameter, `account:` + user |
| herta-3 | raw user -> separate user_key parameter, `user:` + user |

go-2, go-3, and herta-2 preserve the supplied t4 policies. herta-2 holds its
source-transfer key only around debit and credits outside that lock; the
tested balance/completion assertions pass, but this does not establish that
the entire two-step Transfer handler is serialized.

## oracle2

`../oracle2/` retains each original oracle_test.go byte-for-byte and adds
harder_test.go. `../oracle/` is unchanged. Additional cases cover:

- t1: wrapped timeout; timeout joined to an unrelated transport error; timeout
  joined to ErrUnavailable; caller deadline; independent repeated payments.
- t2: same-order key-wait cancellation without entering the backend; mixed
  signed orders under concurrency.
- t3: uncertain inventory recovery/exhaustion; declined charge without retry;
  concurrent checkout capacity; cancellation before reservation.
- t4: simultaneous opposite-direction transfers, nonzero transfers involving
  the empty account, observed prefix spellings (including NUL-prefixed keys),
  and canceled transfers. Watchdogs bound completion independently of the Go
  test timeout.

All six agents pass ordinary wrapped/joined timeout cases. The four t1 failures
are specifically the synthetic compound error
`errors.Join(ErrUnavailable, ErrTimeout)`. The provided backend applies the
charge whenever errors.Is(err, ErrTimeout); checking ErrUnavailable first then
retries an applied payment. The result is two charges and a nil error. The
timeout-first classifiers in herta-1 and herta-3 pass. Treat this as a separate
robustness probe, not a retroactive failure of the original oracle.

## Reference correction disclosure

The original `../ref-go` and `../ref-herta` do NOT pass oracle2: each passes only
t2. They have three gaps: unavailable-first classification on the compound
error; no uncertain-inventory retry; and an outer transfer key that collides
with the raw destination account `transfer:A`.

The original references were preserved. Corrected controls live in
`ref2-go/workspace` and `ref2-herta/workspace`; `corrections.diff` records their
changes. They prioritize timeout classification, retry uncertain inventory,
and namespace Debit/Credit account keys separately from Transfer keys. Both
controls pass every oracle2 task 10/10 plus a race run. These corrected controls
are not agent submissions and are not included in summary.csv.

## Reproduction

```sh
~/pilot/score.sh ~/pilot/results/go-1/bundle-go go
PILOT_ORACLE_DIR=~/pilot/oracle2 ~/pilot/score.sh ~/pilot/results/go-1/bundle-go go
```

`score.py` now accepts optional PILOT_ORACLE_DIR; without it, the original
oracle and scoring behavior remain the default. Added oracle tests are copied
after submission metrics are measured.
