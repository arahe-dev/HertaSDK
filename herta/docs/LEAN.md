# Lean lessons for Herta: checked contracts, explicit trust, measured need

## Scope and evidence

This report connects the agent pilot to a small executable contract model and a
transitive trust ledger. It does **not** establish that Herta improves agent work.
The implementation is on the closed-world Herta branch, not the original main SDK.

Evidence used here:
- The supplied Round-2 report, `REPORT.md`, sections 1–9, at local attachment
  `/home/ubuntu/attachments/45ea2f3d-51fa-4993-a1c8-9df4a2fc4005/REPORT.md`.
- Research source inventory in `~/notes/lean-research.md`; original URLs appear inline below.
- Main-SDK source audit in `~/notes/hertasdk-semantics.md`, at `01a3d671`.
- Formal model commit `5d97156b9676d167447355c2cc6f555d9a8a95c9`.
- Trust ledger commit `863a5465ebd9b381fbc054d9d8f6e5d3efc5d4c7`.
- Committed [trust evaluation](../eval/trust/summary.md) and its reproducing script.

Local notes and the attachment are provenance, not portable repository fixtures.
Pilot counts below are taken from that report; the pilot was not rerun here.
SDK line references distinguish the original main snapshot from this branch.
The ranked lessons are engineering judgments backed by cited evidence, not causal estimates.

## 1. What the pilot actually showed

The supplied report calls this **Round 2**; do not conflate it with the earlier
six-agent pilot or with the later 20-program closed-world evaluation.
Round 2 had 30 agents, 120 agent×task cells, 1,200 normal oracle runs and 120 race runs.
A task needed 10/10 normal passes, a race pass and successful generation.
The independent units for arm comparisons were agents, not repeated oracle runs.

| Arm | Agents | Full pass | 95% Wilson interval | r1 pass | r2/r3/r4 each | Normal passes | Race passes |
|---|---:|---:|---:|---:|---:|---:|---:|
| Go SDK | 10 | 2/10 | 5.7–51.0% | 2/10 | 10/10 | 320/400 | 32/40 |
| Herta | 10 | 1/10 | 1.8–40.4% | 1/10 | 10/10 | 310/400 | 31/40 |
| Herta + composition checks | 10 | 4/10 | 16.8–68.7% | 4/10 | 10/10 | 340/400 | 34/40 |

Overall: 7/30 fully passing, 97/120 task cells passing, 970/1,200 normal passes.
There were no flaky task cells and no stored `DATA RACE` reports.
All **23/23 failing task cells** were the same r1 deadline subcase: 8 Go, 9 Herta, 6 checked.
All 253 failing executions (23×11) had the same one-second deadline assertions.
These are **not 23 observed double charges**, nor caller-deadline overruns.

The pristine composition checker emitted zero diagnostics on 80 final Herta files.
Against the oracle: TP=0, FP=0, FN=15, TN=65; precision is undefined, not 100%.
The four-vs-two full-pass difference is not statistically resolved or attributable to checking.
Intermediate logs had seven failed check targets but did not retain diagnostic bodies.
The report therefore measures final-source detection, not historical bug prevention.

| Round-2 cost proxy (median) | Go | Herta | Checked Herta |
|---|---:|---:|---:|
| Non-test added + deleted lines | 214.5 | 329.5 | 363.5 |
| Non-generated non-test lines | 214.5 | 189 | 261.5 |
| Added test lines | 452.5 | 424 | 404 |
| Logged make targets | 6 | 6 | 12 |

Generated surface explains part of the 1.54×/1.69× total-diff ratios; target nesting
prevents treating counts as independent engineering iterations (report §7).
Session elapsed times are lifecycle proxies, not measurements of active coding speed.

### Four distinct deadline notions, then tag precedence

| Notion | SDK meaning | Consequence |
|---|---|---|
| Per-attempt timeout | Fresh `context.WithTimeout` for each handler invocation | 300ms is not a total retry-loop budget |
| Caller deadline | Absolute bound inherited by each attempt context | Last attempt can have only 100ms left |
| Caller cancellation | Caller context ends independently of a timer | Not evidence that the attempt's own timeout fired |
| Own-deadline promotion | Error plus expired attempt context while caller still live | Default classification becomes Uncertain, with legacy exceptions below |

Current [attempt.go](../../attempt.go):48–61 creates the child context and computes
`expired = err != nil && actx.Err() == context.DeadlineExceeded && ctx.Err() == nil`.
Original main implements the same predicate at `attempt.go:34–43`.
An already ended caller is checked before every attempt (`attempt.go:10–13`).
There is no independent kill of a handler: a noncooperative handler can return late.
A nil-error late success is accepted (`attempt.go:19–21`), not discarded.

Legacy [outcome.go](../../outcome.go):106–114 trusts the Failure found by `tagged` first.
`tagged` uses `errors.AsType`; a typed-nil Failure result is treated as untagged (63–68).
Only absent a tag does `expired && !errors.Is(err, context.Canceled)` promote to Uncertain.
Thus an explicit Transient tag beats own-deadline promotion; so can a Permanent tag.
`Fail` preserves an existing tag (`outcome.go:70–86`); outer wrapping is not a new verdict.
Caller cancellation is never *promoted*, but legacy tags can still classify its error.
The next loop iteration checks the caller before invoking another backend call.
These distinctions matter more than the ambiguous phrase “the timeout was respected.”

The report's verified r1 chain is specific, and includes checks against alternatives:
1. Oracle `r1/oracle_test.go:78–103` installs a cooperative wait for `ctx.Done()`.
2. Each failing host calls the SDK operation/backend with a 300ms attempt timeout.
3. Its Go classifier tags the attempt deadline Transient; legacy tag precedence wins.
4. Four invocations end near 300, 600, 900 and 1,000ms under a one-second caller.
5. The oracle demands one call and ≤550ms, so `deadline calls=4` fails.
6. The backend returns from the wait hook before its application increment
   (`r1/backend.go:67–83`): the measured fixture applies no capture on this path.

The unchanged runtime files and all stored failing logs were inspected by that report.
An ignoring-context hook, data race or deadline overrun is not supported by this evidence.
Four failing checked runs even declared DeadlineExceeded→Transient in metadata;
the prototype did not make that metadata authoritative over the handwritten classifier.

### What the new theorems resolve—and do not

**T3_default** gives one call and exactly 300ms, hence ≤550ms, with untagged errors.
The oracle's rule is exactly the **SDK-default promotion result** for this fixture;
it does not require reinterpreting Timeout as a 300ms end-to-end budget.
**T3_per_attempt** gives four calls and exactly 1,000ms with legacy Transient tagging.
Both readings use per-attempt timeouts; the classification differs.

**T2** supplies an applies-then-times-out backend yielding two applied effects.
This makes the unsafe classification a genuine possible double-application mechanism,
not just a style disagreement. It is a different backend from the pilot wait fixture:
the theorem does not retroactively turn the 23 measured failures into double charges.

**T5** captures the preexisting closed-world fix in `attempt.go:23–39`:
with Policy.Classify installed, caller errors return first; own-expiry forces Uncertain
without calling the classifier. A nonidempotent op not retrying Uncertain cannot retry it.
This structurally removes **this own-deadline retry path**, for any backend/classifier.
It does not eliminate dishonest Transient replies before timeout or nested amplification.
A `classify` clause only selects this execution path when compiled with `build -closed`.

## 2. Lean 1→4: timeline and design changes

Dates and architecture come from the [reference history](https://lean-lang.org/doc/reference/latest/Introduction/),
the [2015 system description](https://leodemoura.github.io/files/lean_cade25.pdf),
the [HoTT library paper](https://florisvandoorn.com/papers/leanhott.pdf), and
the [Lean 4 system paper](https://lean-lang.org/papers/lean4.pdf).
The HoTT paper describes Lean 3 development in 2016; the public 3.0 release was 2017-01-20.
Early version boundaries are not a claim that every subsystem changed on release day.

| Version / year | Kernel | Elaboration / unification | Automation / metaprogramming | Runtime / compiler | Library | Break / reason |
|---|---|---|---|---|---|---|
| Lean 1, 2013–15; 0.1 in 2014 | Small C++ dependent-type kernel; configurable logic | Higher-order unification, coercions, overloading, typeclasses | Early Lua and tactic infrastructure; SMT-inspired automation goal | C++ implementation; JS bindings | Early standard mathematical development | Experimental infrastructure, not today's self-hosted platform |
| Lean 2, ~2015–16 | Standard proof-irrelevant mode plus distinct HoTT mode | Powerful higher-order search/backtracking | Tactics, custom notation and expanding libraries | C++ generation of the system | HoTT: 30,400 proof/spec lines + 3,600 comment lines | Full HO unification caused timeouts and misleading errors; incompatible logical modes |
| Lean 3, development 2016; 3.0 in 2017 | Standard dependent type theory; HoTT mode dropped | Restricted unification, improved predictability | Lean `meta` programs, tactic monad, notation and commands | C++ core plus VM executing Lean metaprograms | Mathlib founded July 2017; >1M lines by end of generation | Proof-erasing VM and pattern matching relying on axiom K did not support prior HoTT mode |
| Lean 4, rewrite 2018; 4.0 in 2023 | Small checking core retained, system rewritten | Extensible elaboration; tabled typeclass search | Hygienic macros; extensible parser, elaborator, tactics, pretty printer | Mostly self-hosted; RC/FBIP; native compilation through C | Mathlib port; modern Std/Mathlib | Not source compatible with Lean 3; rewrite targeted extensibility and performance |

Quantitative checkpoints (historical snapshots, not current universal constants):
- First commit: 2013-07-15; Lean 0.1: 2014-06-16 ([history](https://lean-lang.org/doc/reference/latest/Introduction/)).
- Early kernel: about 6,000 C++ lines plus ~700 for inductive families, as summarized
  in [Lean 1–3 history](https://invarians.org/wiki/Lean_theorem_prover/History/Lean_1_to_Lean_3).
- Lean 2 HoTT's 30,400 lines were comparable to Coq HoTT's 29,800 in the
  [HoTT paper](https://florisvandoorn.com/papers/leanhott.pdf), not proof of feature equivalence.
- April 2018 rewrite: Mathlib ~55K lines; October 2020: Lean 4 compiled Lean 4;
  January 2021 prerelease: Mathlib ~450K; September 2023: ~1.1M ported lines
  ([de Moura's FLoC account](https://leodemoura.github.io/static/floc26/)).
- [Port-status snapshot](https://leanprover-community.github.io/mathlib-port-status/):
  3,112 files and 1,053,081 lines, completion announced 2023-07-16; mathport assisted.
- Lean 4.0 released 2023-09-08, with roughly 90% self-hosting reported in the
  [reference history](https://lean-lang.org/doc/reference/latest/Introduction/).
- The [FLoC talk](https://leodemoura.github.io/static/floc26/) reports 280K+ theorems,
  2.4M+ lines, 750+ contributors, 1,500+ typeclasses and 20K+ instances for Mathlib.
  These speaker-reported ecosystem counts are not measurements made in this PR.

## 3. Primitives, trust boundaries and ranked success factors

### What is checked

Lean's kernel checks terms in dependent type theory: types can depend on values,
functions can return proofs, and a theorem is a term inhabiting its proposition.
Universe levels stratify types; inductive definitions supply constructors and recursors.
Pattern matching and friendly notation elaborate into this smaller core language.
Quotients support reasoning modulo an equivalence relation.
The standard audited dependencies are `propext` (propositional extensionality),
`Classical.choice`, and `Quot.sound` (quotient soundness); these are not claims
that every Lean theorem uses all three, nor all mandatory kernel primitives.
See [axioms](https://lean-lang.org/doc/reference/latest/Axioms/).

A tactic is a program constructing a proof term; it is not itself a kernel inference rule.
The elaborator fills implicit arguments, solves type constraints, resolves instances
and translates syntax. Its search may fail or be buggy without legitimizing a bad
proof term: the kernel must still accept the result, absent explicit trust escapes.
Lean 4 exposes these extension points to ordinary Lean programs
([system paper](https://lean-lang.org/papers/lean4.pdf)).

### Trust escapes are different, not interchangeable

| Mechanism | Boundary | What an audit should say |
|---|---|---|
| `sorry` | Inserts `sorryAx`, inhabiting an unproved goal | Incomplete proof; dependencies propagate |
| Custom `axiom` | Adds an assumed proposition | Valid relative to that assumption, not a proof of it |
| `partial` | Allows executable recursion without totality proof; opaque logical interface | Not a license to prove False; execution may diverge |
| `unsafe` | Permits code outside safe logical checking | Cannot simply be used as a safe proof; separate operational trust |
| `@[implemented_by]` / `@[extern]` | Compiled implementation can differ from logical definition | Runtime agreement is additional trust, not established by ordinary reduction |
| `native_decide` | Uses compiled evaluation to assert a Boolean result | Enlarges trust to compiler/runtime and reached substituted/foreign implementations |

Sources: [proof validation](https://lean-lang.org/doc/reference/latest/ValidatingProofs/),
[recursive definitions](https://lean-lang.org/doc/reference/latest/Definitions/Recursive-Definitions/),
and [decision-procedure taxonomy](https://invarians.org/wiki/Lean_theorem_prover/System/Decision_procedures).
Native-evaluation axiom names vary with version; inspect `#print axioms`, not a tactic's name.
The research notes mention `Lean.ofReduceBool`; this package permits no such extra axiom.
Plain `decide` computes a Decidable instance through checked reduction; our concrete
witnesses use that route, not native evaluation. Neither route validates the specification.
`#print axioms` audits logical dependencies; it is not a complete runtime supply-chain audit.

### Ranked lessons (ranked for transfer to Herta)

1. **Small checking core plus untrusted automation (de Bruijn principle).**
   The 2015 kernel size and Lean 4's elaborator/plugin architecture show the separation
   ([2015](https://leodemoura.github.io/files/lean_cade25.pdf), [2021](https://lean-lang.org/papers/lean4.pdf)).
   Herta should own a small contract checker, not attempt to trust arbitrary generated Go.
2. **Auditable, transitive trust.** `#print axioms` exposes precisely which logical
   assumptions a theorem uses ([validation](https://lean-lang.org/doc/reference/latest/ValidatingProofs/)).
   This is the direct precedent for our ledger; its opacity warnings are not proofs.
3. **Interactive goal state.** Goals and hypotheses give immediate local feedback;
   tactics transform that state, and the kernel checks the resulting term
   ([FLoC demonstration](https://leodemoura.github.io/static/floc26/)).
   Transfer the actionable obligation, not just a red/green CLI exit code.
4. **Self-extensibility.** Lean 3's ICFP 2017 metaprogramming framework put tactics in
   Lean; the [FLoC retrospective](https://leodemoura.github.io/static/floc26/) connects this
   to library automation including `linarith`, `ring`, `omega` and community linters.
   Lean 4 extends the same idea across the frontend and compiler extension points
   ([paper](https://lean-lang.org/papers/lean4.pdf)); “everything extensible” does not mean no trusted core.
5. **A unified library and community.** Mathlib's shared conventions and reusable
   theorems compound; the contributor/theorem counts above are evidence of scale,
   not evidence that collecting unrelated snippets would work equally well.
   Herta's analogue needs vetted contracts and linters, not just classifier examples.
6. **Performance makes checked work practical.** Reference counting, borrowed references,
   and reuse enable destructive updates when values are unshared: “functional but in-place.”
   See [Counting Immutable Beans](https://arxiv.org/abs/1908.05647) and the
   [Lean 4 C backend](https://lean-lang.org/papers/lean4.pdf). This is not a Herta speed benchmark.
7. **Willingness to break, with migration machinery.** Dropping HoTT and constraining
   unification traded generality for the intended workload; the 1.05M-line Mathlib port
   shows both the cost and feasibility of a later rewrite with tools/community support.
   Herta has no comparable installed base that would justify such a migration budget today.
8. **AI writes, kernel checks.** The [FLoC talk](https://leodemoura.github.io/static/floc26/)
   cites AlphaProof's 2024 IMO silver-level result and DeepSeek Prover-V2's 88.9% miniF2F.
   Treat those as task-specific reported results, not general software reliability rates.
   AI can search for terms cheaply without becoming the authority on their validity.
   A wrong formalized requirement can still be proved: human specification review remains.

## 4. Lean concept → Herta analogue

“Implemented” below means this branch has the named artifact, not that Go is verified.

| Lean concept | Herta analogue | Status in this PR / reason |
|---|---|---|
| Small kernel | Small executable contract semantics | **Implemented:** `herta/formal/Herta.lean`; not a verified SDK extraction |
| Axioms / `#print axioms` | Transitive raw-Go trust ledger | **Implemented:** `herta trust`, theorem tags, origin and shortest call path |
| `sorry` | `opaque-handler` | **Implemented analogy:** body remains unchecked; unlike Lean, this is not an actual logical axiom |
| `native_decide` / `implemented_by` | `legacy-tag-path`, host implementation authority | **Implemented warning:** tags can override default semantics; not logical equivalence to Lean's mechanisms |
| `termination_by` | Deadline budget as a decreasing measure | **Proposed:** wall-time budgets require cooperative cancellation; present model recurses on finite attempts |
| `decide` | Finite exhaustive contract checks / counterexample witnesses | **Implemented in Lean:** concrete T2/T3/T4 witnesses; broader CLI exploration proposed |
| Restricted Lean 3 HO unification | Keep checker decidable, local and predictable | **Design constraint:** reject unrestricted Go effect inference; retain explicit contracts |
| Mathlib | Shared vetted classifier/contract library | **Proposed:** only after repeated independently observed needs |
| Infoview | Agent-facing obligation state | **Partly implemented:** deterministic text/JSON and paths; interactive repair UI proposed |
| Tactics / macros | User-authored contract lints | **Proposed:** plugins must not silently enlarge the trusted contract core |
| Proof of program refinement | Verified lowering / Go correspondence | **Not implemented:** no proof that generated Go refines the Lean interpreter |
| General dependent-type language | Replacement for Herta's contract DSL | **Rejected for now:** no measured need or agent benefit justifies the scope |

### Bend/Bend2 analogy (brief; the Bend session owns depth)

Original Bend's HVM/interaction-combinator lineage illustrates automatic parallelism
from a small set of primitives: the runtime, not the user, owns scheduling.
The useful analogy is semantic ownership, not “parallel evaluation solves retry safety.”
Bend2's dependently typed direction adds another intent/proof connection, historically
discussed alongside HVM. Do not freeze that history into a current backend claim:
the [current upstream](https://github.com/bendlang/bend) explicitly says Bend 1 programs
and HVM do not carry over, and the [Bend2 repository](https://github.com/HigherOrderCO/Bend2)
redirects there. No Bend performance, soundness or Herta integration result is claimed here.

## 5. Results of this PR

### Formal artifact and audits

Lean 4.36.0 is pinned; core/Std only, no Mathlib or external dependencies.
The model has 305 Lean lines; the initial complete check took 2s, compilation 595ms.
Its time unit is Nat milliseconds; the backend returns duration/application/error data.
Counters track calls, applied effects and elapsed time, with both classification paths.

| Theorem | Checked result |
|---|---|
| T1 | Arbitrary backend: nonidempotent applied effects ≤1 under explicit Transient honesty |
| T1_assumption_necessary | Deadline hygiene alone permits two effects from dishonest early Transient replies |
| T2 | Legacy own-deadline Transient tag admits an applies-then-times-out witness with two effects |
| T3_per_attempt | 300ms, max four, 1,000ms caller: four calls, exactly 1,000ms with transient tag |
| T3_default | Same fixture, default classification: one call, 300ms ≤550ms |
| T4_key | Any deadline: a synchronously held same key cannot be acquired by nested Wait |
| T4_composition | Pure and idempotent outer retries each admit >1 nested nonidempotent applications |
| T5 | Closed path: any own-expired erroneous nonidempotent attempt cannot retry when Uncertain is not retried |

T1's hypothesis is the informal raw-Go “axiom”: every Transient-classified attempt
applied no effect. It is an explicit parameter in Lean, **not a new Lean axiom**.
T1's proof needs honesty; retained deadline-hygiene/no-Uncertain hypotheses are not sufficient alone.
T5 is local to own-expiry, not whole-run safety against every dishonest classifier.

Recorded successful `check.sh` output from the formal commit, duplicate Lake audit lines omitted:
```text
ℹ [2/4] Built Herta (595ms)
Build completed successfully (4 jobs).
'Herta.observation_applied_le_one' depends on axioms: [propext, Quot.sound]
'Herta.retry_nonidempotent' depends on axioms: [propext]
'Herta.loop_safety' depends on axioms: [propext, Quot.sound]
'Herta.T1' depends on axioms: [propext, Quot.sound]
'Herta.T2' depends on axioms: [propext]
'Herta.T1_assumption_necessary' depends on axioms: [propext]
'Herta.T3_per_attempt' depends on axioms: [propext]
'Herta.T3_default' depends on axioms: [propext]
'Herta.T4_key' depends on axioms: [propext, Classical.choice, Quot.sound]
'Herta.T4_composition' depends on axioms: [propext]
'Herta.T5' depends on axioms: [propext]
'Herta.execution_edges' depends on axioms: [propext]
'Herta.T5_witness_blocked' depends on axioms: [propext]
PASS: all theorem audits present; only standard axioms
check.sh elapsed: 2s
```

Reproduce with `cd herta/formal && ./check.sh` (elan's lake on PATH).
The script replays the source even on cached builds and rejects `sorryAx` or other axioms.
This documentation-only step did not change/re-run those proofs.

Model limits: finite caller deadline, one counted application per backend invocation,
abstract error tags rather than error chains, no independent explicit cancellation,
invalid enum values, scheduling, shutdown, resource queues or goroutine nontermination.
Cooperative hangs end on context expiry; other responses can return late.
Backoff is a parameterized extension; zero matches the SDK's lack of backoff.
Held keys are stable during nested Wait; closed runtime guards may reject immediately.
The nested effect witness aggregates inner applications, not a second full interpreter.
These are contract proofs, not kernel-checked statements about running Go binaries.

### Trust ledger evaluation

Run `python3 herta/eval/trust/evaluate.py` from the repository root.
It builds the CLI and covers 20 original inputs, 20 migrated models and 121 corpus files.
Inputs/corpus use legacy mode; migrated models use `trust -closed`.
Each cell: **unique origin obligations / propagated per-op occurrences / programs flagged**.

| Kind | Inputs (20 programs, 45 ops) | Migrated (20, 45) | Corpus (121, 283) |
|---|---:|---:|---:|
| opaque-handler | 45 / 45 / 20 | 45 / 67 / 20 | 283 / 447 / 121 |
| transient-honest | 15 / 15 / 10 | 15 / 25 / 10 | 28 / 29 / 27 |
| legacy-tag-path | 45 / 45 / 20 | 30 / 42 / 15 | 283 / 447 / 121 |
| nested-retry-amplification | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| same-serialize-key | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |
| unresolved-callee | 0 / 0 / 0 | 0 / 0 / 0 | 0 / 0 / 0 |

Opaque-handler and legacy-tag-path flag **100% of original programs/ops**;
they are not discriminative bug detectors. Closed lowering reduces legacy obligations
but still leaves 15/20 migrated programs with at least one legacy operation.
Only transient-honest is selective among nonzero kinds in the original set: 10/20 programs.
It is required for nonidempotent origins and informational otherwise.
Every nonempty evaluated program fails strict because all retain required opaque-handler.
Zero key/amplification counts describe this frozen inventory, not an absence theorem.

| Known oracle2 failure | Location flagged? | What the ledger actually establishes |
|---|---|---|
| H2/t1 | `charge`: transient-honest; legacy-path in input | Broad obligation, not joined-error ordering diagnosis (HC010 addresses that) |
| R/t1 | `charge`: same as above | Same limitation |
| R/t3 | `reserve`: informational transient-honest | **Miss:** missing retry-on-Uncertain is not detected |
| R/t4 | `transfer`: generic legacy-path | **Miss:** concrete dynamic `transfer:A` collision gets no key-hazard flag |

All 15 input HC016 sites across 10 programs have nontrivial obligations at the exact origin;
migrated HC016 count is zero. Generic legacy coverage alone does not diagnose a bug.
Arbitrary Go methods can hide calls; propagation follows visible/declared edges only.
This is **coverage, not diagnosis**; nothing here shows a measured agent benefit yet.

Recorded Step-2 checks: `go test -v ./herta/...` passed; checker goldens **30/0/0**,
trust goldens **6/0/0** (passed/failed/skipped). Runner caps remain 200 cases,
5s/case and 60s total. No new CI workload or runtime semantics change was introduced.

## 6. What should go into Herta next: prove the need first

The order is deliberate: gather discriminating data before expanding the language.
Each proposal has a rejection condition as well as a falsifiable experiment.

| Order | Concrete next step | Experiment / evidence required to proceed |
|---|---|---|
| 1 | Make timeout/Uncertain/cancellation requirements explicit in task contracts | Rerun a preregistered SDK baseline with both safe no-effect and apply-before-timeout backends; if clarified prompts remove failures without tooling, do not credit a new checker |
| 2 | Test authoritative closed classification before new syntax | Run identical backends/policies through legacy and closed codegen, including wrapped/joined errors and caller cancellation; require the T5 path to block own-expiry retry and preserve intended pre-timeout retries; any disagreement blocks rollout |
| 3 | Prototype selective, actionable trust obligations | Randomize agents to check-only vs check+ledger on held-out tasks; record causal repairs, false leads, time and generated/authored diffs; if generic opacity warnings dominate without better held-out outcomes, keep ledger optional |
| 4 | Add bounded model/Go differential traces | Enumerate small durations, retry sets and context boundaries; compare calls/outcomes/time/effects against deterministic Go fixtures; minimize mismatches before extending proofs; no mismatch means no evidence for a semantic rewrite |
| 5 | Prototype runtime key provenance/disjointness contracts in SDK | Use blind transcriptions plus R/t4 and adversarial concrete keys; measure exact collision detection and false positives; if declarations do not reveal collisions, prefer existing runtime guards to speculative static alias inference |
| 6 | Test total deadline budgets as explicit contracts | Collect real time-bounded polling sites; compare fixed attempts with an SDK budget prototype under cooperative and noncooperative handlers; require reduced deadline violations without losing required progress before adding a decreasing-budget DSL construct |
| 7 | Build a small vetted classifier/contract library | Independently review recurring contracts across unrelated services, inject dishonest Transient and joined-error mutations, measure reuse and escaped effects; no repeated reuse or mutation resistance means no Mathlib-scale investment |
| 8 | Trial agent-facing incremental obligation output | Compare existing JSON with goal-style deltas on the same repair tasks; measure correct first repairs and token/tool cost; no improvement means no infoview-style UI or plugin framework yet |

Keep cases bounded and preserve the capped runner; use a separate explicitly budgeted
experiment for agent outcomes rather than quietly enlarging normal CI.
Report confidence intervals over independent sessions, not ten correlated oracle repeats.
Retain misses and correct-program controls; lock the oracle before seeing submissions.
The near-term deliverable is a smaller, more accountable trust boundary—not a general
proof language, automatic Go verifier, or unmeasured claim of safer agents.
