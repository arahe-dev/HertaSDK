# Closed-world Herta pilot evaluation

Target: `devin/herta-closed-world`, SDK/compiler `ff3f185ba7c4df825791420b73508d659ca0a432`.
20 logical Herta programs: H1/H2/H3 = agents 1/2/3; R = original reference;
R2 = corrected reference. All work occurred in copies; the pilot was not edited.

## Method, freeze, and interpretation

- Mechanical rules are in `migration-rules.md`, written before checks or oracles.
- Frozen at **2026-11-19T17:38:13.248039+00:00**: all 20 migrated `.he` files,
  five behavior-neutral context-alias Go files, migration diffs, original source
  snapshots, and exact oracle inputs: 167 hashed files total.
  `freeze.json` and `models.sha256` remain unchanged. All test event timestamps
  are after the freeze. No migration was changed in response to results.
- Snapshot sources: agents' `bundle-herta`; R's `oracle2-workspace`; R2's
  `workspace`. Original `oracle_test.go` and added `harder_test.go` are from
  R's retained workspace, byte-equal to R2's; original oracle is also verified
  byte-equal to `pilot/oracle`. Test selection mirrors pilot scoring, but suites
  compile separately so own/oracle results are unambiguous.
- Fresh baseline used retained generated Go plus the **new SDK replace**.
  All 20 own suites and all 20 original oracles pass; oracle2 fails exactly
  H2/t1, R/t1, R/t3, R/t4, reproducing the pilot labels.
- `P` = pass; `U` = failure with ErrUndeclaredCall; `K` = failure with
  ErrKeyReacquired; `O` = other behavioral failure. No compilation failure,
  watchdog expiry, or unrelated test failure occurred.
- **BLOCKED*** means the real CLI build exited 1; the three runtime columns
  are explicitly supplemental diagnostic-bypass executions using the same
  `gogen.GenerateClosed` without Check. They are NOT successful CLI builds.
  This is necessary to exercise faithful buggy contracts rather than silently
  repair their behavior. CLI-accepted rows use actual CLI build output.
- Minimal migration adds classify only to retrying operations and calls only
  for actual nested edges; nonretrying classification/propagation is unchanged.
  Context selectors are not supported in classify syntax, so explicit Go
  aliases are a harness accommodation, not a claim of pure `.he`-only migration.
  No original Go/helper/backend/test source was changed.

## A. Unmodified `.he` in closed mode

| Program | check -closed | CLI build | Own | Oracle | Oracle2 |
| --- | --- | --- | --- | --- | --- |
| H1/t1 | HC016 (charge) | BLOCKED* | P | P | P |
| H1/t2 | — | P | P | P | P |
| H1/t3 | HC016 (charge + reserve) | BLOCKED* | U | U | U |
| H1/t4 | — | P | U | U | U |
| H2/t1 | HC016 (charge) | BLOCKED* | P | P | O |
| H2/t2 | — | P | P | P | P |
| H2/t3 | HC016 (charge + reserve) | BLOCKED* | U | U | U |
| H2/t4 | — | P | U | U | U |
| H3/t1 | HC016 (charge) | BLOCKED* | P | P | P |
| H3/t2 | — | P | P | P | P |
| H3/t3 | HC016 (charge + reserve) | BLOCKED* | U | U | U |
| H3/t4 | — | P | U | U | U |
| R/t1 | HC016 (charge) | BLOCKED* | P | P | O |
| R/t2 | — | P | P | P | P |
| R/t3 | HC016 (charge + reserve) | BLOCKED* | U | U | U |
| R/t4 | — | P | U | U | U |
| R2/t1 | HC016 (charge) | BLOCKED* | P | P | P |
| R2/t2 | — | P | P | P | P |
| R2/t3 | HC016 (charge + reserve) | BLOCKED* | U | U | U |
| R2/t4 | — | P | U | U | U |


Ten programs are rejected for HC016 (15 op-level diagnostics); the ten others
build normally. All five t3 and all five t4 programs surface hidden nested
calls as ErrUndeclaredCall. t3 requires the labeled diagnostic bypass to reach
runtime; t4 does not. No ErrKeyReacquired appears yet: undeclared edges reject
the calls before lock admission. H2/t1 and R/t1 still fail only the joined-error
oracle2 case; other t1 and all t2 suites pass in supplemental execution.

These missing-contract diagnostics and guards are migration requirements,
not evidence that all affected original programs were behaviorally buggy.

## B. Frozen faithful migration

| Program | check -closed | CLI build | Own | Oracle | Oracle2 |
| --- | --- | --- | --- | --- | --- |
| H1/t1 | — | P | P | P | P |
| H1/t2 | — | P | P | P | P |
| H1/t3 | HC010 (charge + reserve) | BLOCKED* | P | P | P |
| H1/t4 | — | P | P | P | P |
| H2/t1 | HC010 (charge) | BLOCKED* | P | P | O |
| H2/t2 | — | P | P | P | P |
| H2/t3 | HC010 (charge + reserve) | BLOCKED* | P | P | P |
| H2/t4 | — | P | P | P | P |
| H3/t1 | — | P | P | P | P |
| H3/t2 | — | P | P | P | P |
| H3/t3 | HC010 (charge) | BLOCKED* | P | P | P |
| H3/t4 | — | P | P | P | P |
| R/t1 | HC010 (charge) | BLOCKED* | P | P | O |
| R/t2 | — | P | P | P | P |
| R/t3 | HC010 (charge + reserve) | BLOCKED* | P | P | O |
| R/t4 | — | P | P | P | K |
| R2/t1 | — | P | P | P | P |
| R2/t2 | — | P | P | P | P |
| R2/t3 | HC010 (charge + reserve) | BLOCKED* | P | P | P |
| R2/t4 | — | P | P | P | P |


All HC016 disappear. Seven programs retain HC010 (11 op-level diagnostics),
so making every faithful model pass the checker would require a behavior-changing
repair, contrary to faithful migration. Thirteen build normally. Supplemental
execution of all 20 preserves **20/20 own-test passes and 20/20 original-oracle
passes**; oracle2 has the same four failing programs as baseline. No new
own/original-oracle regression was observed. The seven CLI blocks still impose
real integration cost, including four oracle2-passing programs.

### Program-level confusion table (static flags vs baseline oracle2)

| Static result | Oracle2 fails (4) | Oracle2 passes (16) |
| --- | --- | --- |
| Flagged (7) | 3: H2/t1, R/t1, R/t3 | 4: H1/t3, H2/t3, H3/t3, R2/t3 |
| Silent (13) | 1: R/t4 | 12 |

Nominal TP=3, FP=4, FN=1, TN=12: precision 3/7 (42.9%), recall 3/4 (75%),
oracle-relative false-positive rate 4/16 (25%). **Do not interpret that as
three correct causal diagnoses:** R/t3's flag is for joined-error ordering,
not the missing uncertain retry that its oracle2 test actually fails. The four
oracle-relative FPs likewise have order hazards outside this oracle's t3
coverage; an oracle pass is not proof those warnings are spurious.

Mechanism-sensitive accounting: the checker diagnoses both t1 order failures;
it does not diagnose R/t3's missing retry-on-uncertain; R/t4 is caught dynamically
by key reacquisition, not statically. Thus 2/4 known failing mechanisms receive
matching static diagnoses, and one more receives a matching runtime guard.

### Why R/t3 and R/t4 fail (verified paths and controls)

**R/t3:** `inputs/R/t3/shop.he:6` retries reserve on transient only. Checkout's
handler directly invokes `svc.reserve.Do` at line 12; Service.Checkout reaches
the governed checkout. `glue.go:15-26` maps ErrTimeout to Uncertain, so an
idempotent inventory timeout receives only one attempt. Fresh baseline and B
both fail `TestOracleHardCheckout/uncertain_inventory` and
`uncertain_inventory_exhausted` with ReserveCalls=1 and Charges=0. Changing only
classifier order in C leaves this failure intact. As a disconfirming control,
R2/t3 has the same classifier order but retries transient **and uncertain** and
passes. HC010 on R/t3 is therefore not the causal diagnosis of this oracle bug.

**R/t4:** `inputs/R/t4/glue.go:72` gives transfer the key `"transfer:" + from`.
`shop.he:14` holds it while the handler calls Debit then Credit (lines 17-18).
Debit/Credit serialize on raw user (lines 6,10). Oracle2's from=`A`,
to=`transfer:A` therefore makes credit reacquire the ancestor's `transfer:A` key
on the same runtime. Fresh baseline waits until its 400ms caller deadline:
DebitCalls=1, CreditCalls=0, A=90. B returns **ErrKeyReacquired,
main::transfer -> main::credit**, immediately, with the same partial debit.
The guard prevents a hang; it does not roll back or make the transfer correct.
R2/t4's prefixed account keys pass the same oracle pair, and H3/t4 passes until
the explicit same-key mutation. This is an actual key-value collision, not
proof that distinct symbolic key parameter names are statically disjoint.

## C. Authority and runtime guard demonstrations

Only classifier arm order changed for these variants; original Go classifiers
are byte-identical to the frozen inputs. All seven now pass check and CLI build.

| Order-only variant | B oracle2 | C oracle2 | Interpretation |
| --- | --- | --- | --- |
| H1/t3 | P | P | already passed this oracle |
| H2/t1 | O | P | fixed joined-charge failure |
| H2/t3 | P | P | already passed this oracle |
| H3/t3 | P | P | already passed this oracle |
| R/t1 | O | P | fixed joined-charge failure |
| R/t3 | O | O | missing uncertain retry remains |
| R2/t3 | P | P | already passed this oracle |

H2/t1 and R/t1 switch from two charges/two calls/success to the required one
charge/one call/Uncertain, despite their unchanged Go helpers still tagging the
joined error Transient. Four t3 programs remain passing. R/t3 still fails:
classifier authority cannot add a retry outcome absent from the policy.

| Program | Mutation | Check | CLI build | Sentinel assertion | Elapsed | Oracle2 |
| --- | --- | --- | --- | --- | --- | --- |
| H3/t3 | Drop checkout -> reserve | HC010 | BLOCKED* | PASS: ErrUndeclaredCall | 31.493µs | U |
| H3/t4 | Drop transfer -> debit | — | P | PASS: ErrUndeclaredCall | 30.812µs | U |
| H3/t4 | Nested debit uses held from_key | — | P | PASS: ErrKeyReacquired | 31.8µs | K |

Focused tests use errors.Is, require Permanent, and bound completion to 200ms
under a 250ms caller timeout. All three pass in approximately 31 microseconds;
the ordinary oracle2 tests fail with the expected guard. The t3 drop variant
retains its unrelated HC010, so its runtime test uses the labeled bypass.
The same-key variant inserts a debit with `UserKey: from_key` before the existing
transfer body; original Go files are untouched. It remains statically silent
because this servicePtr field edge/key value is opaque, but the guard sees it.

## D. Visible bypass surfaces

| Surface | Observed in production paths? | Interpretation |
| --- | --- | --- |
| context.Background replacing handler ctx | No, across all 20 | Calls use supplied ctx; tests use Background at legitimate root entry points. |
| Goroutines launched inside handlers/helpers | No, across all 20 | Concurrency is in tests, not detached governed work. |
| Direct backend Charge/Reserve/Refund/Debit/Credit calls | Yes, every task | Expected leaf handler/helper implementation, under its governed op; runtime cannot authorize an arbitrary direct backend call as a nested op. |
| Opaque servicePtr helper/field .Do | Yes, every t3/t4 | Static extraction misses these; calls declarations plus context guards cover the executed nested paths. |
| H2/t4 Credit after transfer.Do returns | Yes | Credit is a separate governed root call, not a transfer callee. Migration correctly declares only transfer -> debit. |

`bypass-scan.json` contains source locations. No observed Background/goroutine
bypass explains the results. The absence is source-scoped, not a sandbox proof:
arbitrary host Go can deliberately discard ctx, forge an op name, or invoke the
backend directly. No such intentional bypass was found in these service paths.

## Reproduction and caveats

From the repository root (Go 1.26), point HERTA_SDK to a checkout containing the
target runtime/compiler. Existing captured logs are reused for baseline/A/B;
to re-execute everything, copy this artifact directory to a persistent scratch
directory and delete only its generated bin/, logs/, workspaces/, results.json
and results.csv first (retain freeze/models/inputs/oracles).

```sh
HERTA_SDK="$PWD" python3 herta/eval/closed/evaluate.py run
python3 herta/eval/closed/report.py
```

External original artifact path: `/home/ubuntu/work/closed-eval`. Repository
copy omits only generated binaries/workspaces and Python caches. A local go.mod
isolates evidence Go files from root `go test ./...`. Local attributes preserve
CSV bytes and exempt frozen .he/diff whitespace rather than rewriting evidence.
Frozen original inputs,
models/diffs/aliases, exact oracles, variants, scripts, results, and compressed
full command logs are retained under 2 MB. `results.csv` has one row per command,
with exit code, diagnostic codes, guards, named failed tests and a log path.
No transcriptions were revised after oracle execution. A harness-only logging
bug initially treated a quoted Go import as a JSON test event; it was fixed,
existing completed observations were recovered, and only pending work resumed.
No checker/SDK fix or post-result model repair was made.

One execution per suite (-count=1), no fresh race/repetition study: these results
are not reliability estimates. Original oracle2 repeats/race logs were consulted
only as corroboration; tables above report fresh executions. No codegen unused-
parameter failure occurred in these final programs. Guard errors in ordinary
tests are identified by messages; the focused mutations additionally assert
sentinel identity directly. Behavioral preservation is demonstrated for the
retained test inputs, not proven for all joined/pre-tagged/context error chains.
