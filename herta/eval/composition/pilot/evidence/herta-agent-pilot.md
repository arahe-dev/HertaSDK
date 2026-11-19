# Herta vs Go+hertasdk: agent benchmark pilot (n=3 per arm)

Toolchain: arahe-dev/HertaSDK PR #9 (Go codegen, stacked on #6 checker). Advice only; nothing was pushed to the repo.

## Bottom line
1. **On these tasks the compiler made no difference to correctness.** All 6 agents (3 Go, 3 Herta) passed all 4 hidden-oracle tasks, 10/10 repeats plus a `-race` run. None fell into the planted traps.
2. **The compiler did not catch the bugs that matter here.** The Herta agents' wrong turns were a nested-key deadlock (herta-2 found it with its own test) and generated code that didn't compile. The checker missed both. The PR #9 checker has no composition analysis: an op whose handler calls another op is opaque Go to it.
3. **Herta cost more.** The Herta arm took about 1.5x the time to report done, made 2.4x the non-test diff (most of it generated Go) and added 1.4x the test lines. 2 of 3 Herta agents hit a codegen bug that the Go arm cannot hit.
4. **A harder oracle written after the runs** found a classification bug in 4 of 6 runs: an error that joins `ErrUnavailable` and `ErrTimeout` was retried, so the charge went through twice. It affected 3/3 Go and 1/3 Herta runs. The checker played no part in that difference (see §4).
5. **Implication for scope:** Herta's value only shows up if the checker models **op-to-op composition**: call edges, key reacquisition, retry amplification and effect propagation, plus error classification as a declared, total mapping. Per-op policy checks that hertasdk already does at `NewOperation` add friction and nothing else.

## 1. Method
- **Tasks** (identical fixtures, specs and smoke tests in both arms. The only difference is how ops are written):
  - **t1 Charge:** retry ≤3 when safe; a timeout may mean the charge was applied; never double charge. *Trap:* treating timeout as transient.
  - **t2 Refund(orderID int64):** serialize per order and run different orders in parallel. Every int64 (0, negative, min, max) needs a distinct non-empty key, because an empty key turns serialization off. *Trap:* using the raw or zero key.
  - **t3 Checkout:** reserve inventory (idempotent, retryable), then call the supplied `Charge` once; never charge after a failed reserve. *Trap:* one retrying checkout op wrapping the charge.
  - **t4 Transfer(from,to):** serialize by source account using the supplied Debit/Credit; IDs are opaque, `""` included. *Trap:* nested ops reacquiring a colliding key, which self-deadlocks.
- **Arms:**
  - **Go:** write `hertasdk.NewOperation`/`Policy` by hand.
  - **Herta:** declare ops in `.he`, then `herta build`. Agents were told not to write `NewOperation`/`NewRuntime` by hand.
- **Fairness checks before the runs:**
  - Byte-identical SDK and fixtures across arms.
  - TODO ops have empty policy blocks; Go stubs don't name other tasks' ops.
  - No oracle or reference code in the bundles; both bundles compile with smoke tests failing.
- **Validation:** the reference solution passes 4/4 in both arms. The trap solution passes 1/4 in both arms: the oracle catches t1, t3 and t4. An unsafe retry declaration is rejected by both `NewOperation` and `herta check`.
- **Agents:** 6 independent Devin sessions (same model and mode), the same prompt apart from arm instructions, started at the same time, no shared state. Go: 23ac636e, d9751c70, b197d18b. Herta: a1c06268, 0327b77f, bea1f255.
- **Scoring:** hidden oracle per task, run 10x plus once with `-race`. Also scored: test lines added (non-smoke test files), non-test lines changed (adds + deletes, generated Go included), and edits to supplied op policies.

## 2. Results

| Run | Oracle (pre-registered) | Oracle2 (post hoc) | Test lines added | Non-test lines changed | Time to "done" |
|---|---|---|---|---|---|
| go-1 | 4/4 | 3/4 (t1) | 146 | 134 | ~2.2 min |
| go-2 | 4/4 | 3/4 (t1) | 165 | 154 | ~1.9 min |
| go-3 | 4/4 | 3/4 (t1) | 215 | 196 | ~2.8 min |
| herta-1 | 4/4 | 4/4 | 308 | 434 | ~3.3 min |
| herta-2 | 4/4 | 3/4 (t1) | 135 | 335 | ~3.5 min |
| herta-3 | 4/4 | 4/4 | 282 | 375 | ~4.0 min |
| **Go mean** | 12/12 | 9/12 | 175 | 161 | ~2.3 min |
| **Herta mean** | 12/12 | 10/12 | 242 | 381 | ~3.6 min |

Time is from the session start to the agent's final report. Iterations could only be counted at the top level, because the children delegated part of their shell work, so these are lower bounds:

| Run | Failed build/test cycles seen (cause) |
|---|---|
| go-1 | 1 (own t1 test) |
| go-2 | 0 |
| go-3 | 0 |
| herta-1 | 2 (codegen compile error ×2) |
| herta-2 | 2 (t4 deadlock in own test; t3 .he edit) |
| herta-3 | 1 (codegen compile error) |

**Design choices per task:**
- **t1:** all six retry only `ErrUnavailable` and map `ErrTimeout` to uncertain.
- **t2:** all six use a decimal key (`"order:"+id`, `"refund:"+id`, or plain).
- **t3:** all six use a separate retrying reserve op, then call the existing charge once.
- **t4:** the designs split into three:
  - prefixed runtime keys (go-1, herta-3);
  - a separate context-aware lock (go-2, go-3);
  - ordered two-account locks (herta-1);
  - a NUL-prefixed source lock with credit outside the lock (herta-2). This passes the oracle but serializes only the debit step.

## 3. Contract-bug escapes (where each bug was caught)

| Bug | Go: caught by | Herta: caught by |
|---|---|---|
| nonidempotent + retry uncertain (declared) | `NewOperation` at startup | `herta check` at compile time |
| timeout classified as transient (handler code) | nothing static; oracle only | nothing static; oracle only |
| joined Unavailable+Timeout retried | nothing; oracle2 caught 3/3 | nothing; oracle2 caught 1/3 |
| nested op reacquires same key (deadlock) | nothing; own tests | nothing; own test (herta-2) |
| idempotent op wrapping nonidempotent call + retry | nothing | **accepted** by check and build (probe) |
| empty serialize key turns serialization off | nothing | nothing |

The probe for the last rows (in `~/pilot/probes/pr9`): a `.he` handler that calls `debit.Do` under the same `serialize key`, and an `idempotent` op with `retry on {uncertain}` that calls a nonidempotent `charge.Do`. Both pass `herta check` and `herta build`. `checkHandler` only looks at `fail` statements, and `rg 'reacquir|amplif|callgraph'` in `herta/lang` finds nothing.

**Toolchain defect found by the agents:** codegen unpacks every parameter into a local (`gogen.go:143-145`). Any parameter the handler body does not use, including one used only as a `serialize key`, therefore fails to compile with `declared and not used`. 2/3 Herta agents hit this, and each worked around it by hand.

## 4. Interpretation
- **No correctness gain from Herta on this suite.** The pre-registered result is a ceiling effect: 24/24 cells passed. Strong agents read the spec and avoided every trap in both arms, so the suite is too easy to separate the arms.
- **The Herta lead on oracle2 (2/3 vs 0/3 on the joined error) is not a compiler effect.** herta-1 and herta-3 happened to check `ErrTimeout` first, and the language has nothing that pushed them to do that. herta-2 failed the same way the Go runs did. Oracle2 was also written after reading the submissions, and the original references failed it too, so treat it as a probe and not a score.
- **Herta costs:** ~1.5x time, ~2.4x non-test diff, more test code, and a new class of failure (codegen). For agents, every extra layer (`.he` → generate → Go glue) is one more surface to edit and regenerate. This is the CI and test-growth worry Abdur raised.
- **Where the escapes actually came from:**
  - error classification: how raw errors map to transient/uncertain/permanent;
  - key construction: empty or colliding keys;
  - op-to-op composition: nested keys, retry wrapping, effects.

  Herta's design aims at all three, but none of them is modeled today.

## 5. What this says about Herta's scope
Priority order, based on where the bugs escaped:
1. **Composition-aware checking:**
   - make op-to-op calls first-class (`call debit(...)` and not `debit.Do` in raw Go);
   - build the call graph;
   - reject reacquiring the same or an overlapping key (key-namespace typing);
   - propagate effects (an op is at least as non-idempotent as its callees);
   - bound retry amplification across layers.

   This is the only check category in this pilot that neither the SDK nor tests caught by construction.
2. **Total, ordered error classification as a declaration:** for example, `classify { ErrTimeout => uncertain; ErrUnavailable => transient; _ => permanent }`. The checker should enforce precedence (uncertain wins on a joined error) and exhaustiveness. This is the bug that escaped in 4/6 runs.
3. **Key types:** a `key` type that cannot be empty and carries a namespace, so `serialize key` on a raw user string, or two ops with colliding keyspaces, are type errors.
4. **Cut agent friction before adding features:**
   - fix the unused-param codegen bug;
   - make `herta build` part of `go test` (or check generated code in) so that one command reaches green;
   - emit diagnostics that name the policy fix.

   The cost gap here is mostly glue and generated code.
5. **Do not spend effort on per-op policy checks.** `NewOperation` already enforces them at startup, with equal results in this pilot.

## 6. Limitations
- **n=3 per arm, one model, one mode, 4 tasks.** None of the differences is statistically meaningful. The pre-registered result is a ceiling.
- **Iteration counts are top-level only and undercounted**, because the children delegated part of their work. Use them only as direction.
- **Oracle2 was written after the runs** and partly from reading the submissions, so it is biased toward the bugs that were present.
- **The specs were explicit** ("timeout may mean applied", "empty key disables"). Real tasks would be vaguer, and that is likely where static checks help more. This was not tested.
- **The Herta arm is on an early toolchain (PR #9)** with a known codegen bug. Line counts include generated Go.
- **Tasks were short and single-session.** The "hundreds of iterations bloat CI" effect was not measured.

## 7. Next run (to make this decisive)
- **Harder tasks:**
  - vague specs;
  - deeper composition (3-level nesting, cross-package);
  - multi-step edits to an existing codebase, where an agent changes one op's effect and a caller breaks.
- **More data:** n≥10 per arm, at least 2 models, and keep oracle2 cases pre-registered.
- **A third arm:** Herta with prototype composition checks (call edges + key reacquire + effect propagation). Measure whether it converts oracle-only catches into compile errors.
- **Longer runs:** 50–100 edit iterations per arm, tracking test-suite and CI time growth.

Artifacts on the benchmark VM: `~/pilot` (bundles, oracle, oracle2, score.sh, results/summary.csv, probes/pr9).
