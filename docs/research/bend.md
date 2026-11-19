# Bend and Bend2: what transfers to Herta

**Decision:** Prototype a small, executable representation of governed calls in
Go before changing Herta's syntax. Make it the runtime's source of truth for
call order and retries. Let Go implement the leaves, not the orchestration
being checked. Freeze the law independently, and make the adapter's evidence
an explicit assumption. Do not adopt HVM, a GPU scheduler, or a general proof
language on the strength of this study.

The 30-agent pilot did not show a measurable language benefit. It mostly
measured agreement on a contested deadline rule and did not exercise the rule
checker against the Go code that actually ran. The subsequent
[closed-world evaluation][closed] is stronger but still limited: two of four
known failing mechanisms had matching static diagnoses; one more was caught
only by a runtime key guard, **after** the debit; one missing retry remained
undetected. Four out of the 16 oracle-passing programs were also blocked by
the static checker; those warnings may be real hazards outside oracle
coverage. A stricter declaration is not yet a checked workflow.

## Two different Bends

| | Original Bend, pinned at 0.2.38 | Bend2, pinned at 2.0.40 |
| --- | --- | --- |
| Main bet | Turn parallel structure in functional code into runnable interaction nets. | Use an affine dependent type discipline for ownership and proofs; make pure fork/join explicit. |
| Source primitives | Lambdas/applications, ADTs/matches, recursion; `bend` sugar builds recursive generators and `fold` consumes recursive data. | Affine binders, erased `-`, reusable `+` restricted to `Data`; dependent functions/pairs and equality; paired calls and an explicit GPU `!` mark. |
| Executed | Compile to HVM2 nets; HVM2 uses local rewrites, with Rust, C and CUDA paths. | Compile source to a flat C evaluator for native execution, or JS; native forks have disjoint result slots and join when all results arrive. **Not** an HVM net backend. |
| Promise and limit | No explicit source threading, but a dependent chain remains sequential; the toolchain had an optional type system by this release, not Bend2's proof gate. | Pure branches can be independent but need balanced work for the native scheduler; the balance is not checked. The JS fork runs sequentially. IO effects use a separate request/continuation path. |

**Original Bend's enabling representation.** Lafont-style nets connect nodes
by ports. When two principal ports meet, a rule rewrites only that active pair
and reconnects its external wires. Disjoint active pairs can reduce without
choosing one global evaluation order. HVM2 extends the constructors,
duplicators and erasers with pragmatic nodes for numbers, operations, matching
and references. Bend's source may use a variable repeatedly; the compiler
inserts duplicators or erasures to make usage explicit in the net. It encodes
application and lambda with different orientations of the same node. A
recursive-priority pass marks selected binary branches for distribution.
This buys schedulable *pure reductions*, not independence of arbitrary Go
effects or speedup for a serial algorithm. See [Bend's lowering][lowering],
[its priority pass][priority], and [HVM2][hvm2].

**Lineage, not a version-number analogy.** [HVM1][hvm1] was the earlier
interaction-net prototype; original Bend actually pinned *HVM2 2.0.22*.
[HVM3][hvm3] documents an intermediate direction based on Taelin's
[Interaction Calculus][ic]: affine variables, global substitution, explicit
labelled duplication and superposition. Equal labels at a dup/sup
interaction select corresponding branches; unequal labels distribute.
HVM3's stated aim to target Bend is not evidence that today's Bend2 uses it.
The [August 2026 rewrite][rewrite] became [Bend 2.0.0][launch] in September;
the old [Bend2 repo redirects][redirect] to the current Bend repo. The
[BendRT paper][bendrt] explicitly says the native evaluator has no interaction
nets. In other words, the transferable idea survived replacement of the VM.

**Bend2's enabling restriction.** A live value is affine (use at most once;
discarding is allowed) unless the binder asks for reuse and the kind permits
it. `Type` values such as arrays, handles and closures are not freely
copyable. This gives the compiler enough information to pass ownership and
reclaim or reuse a consumed node; permitted sharing can use reference counts.
The native compiler lowers a tail call to a jump, a dependent call to a
continuation, and a fork to children plus a join. It uses a fixed placement
scheme instead of work stealing, so equal-sized splits matter. IO operations
are sequenced by their effect machinery, not by interaction reduction; even
with affine handles, external services can contend on shared state. See the
[guide][guide] and sections 3–9 of [BendRT][bendrt]. The authors' hardware
benchmarks are not independent evidence, and their paper explicitly leaves
native generated code/runtime correctness outside its proof.

**Bend2's enabling proof boundary.** A `law` names a proposition about the
application definition; a matching `def` inhabits that type. Case splits,
recursive proof terms, rewriting and definitional equality can discharge it.
This is stronger than recording that some unrelated proof ran. The gate
rejects open claims and refuses unsafe/foreign dependencies outside its
supported trusted base. The ordinary fast check is not the
independent kernel: `--verdict` translates the checked book and rechecks it
with BendTT, whose Lean source proves results about the core checker's
consistency and live evaluation. The [BendTT paper][bendtt] distinguishes
those results from the unverified surface-to-kernel translation and kernel
parser. Rechecking a generated body does not establish that it faithfully
translates the source law, nor that native code or real IO implement the
model. The project can emit the translated book for inspection.

## Mechanism, not imitation

| System | What the abstraction owns | What it cannot infer | Herta consequence |
| --- | --- | --- | --- |
| Go | Goroutines, typed channels and `select` compose blocking work; the [memory model][go-mem] specifies synchronization edges. The [compatibility promise][go-compat] makes small primitives safe to build on. | Domain effects, admission limits, retries or whether two channels ultimately drive the same device. An idiomatic goroutine can still leak or deadlock. | Keep host Go, its integration story, and a tiny stable vocabulary. Herta must own the modeled execution path, not compete as another goroutine syntax. |
| Bend / HVM2 | Representation makes independent reductions available to the engine. | Speedup without parallel work; application meaning. | Put the governed dependency in executable plan data, not a `calls` list detached from execution. Dynamic keys and external effects still need checks. |
| Bend2 | Affine pure dataflow makes fork ownership visible. The gate checks a declared law against a definition. | Equal work, honest foreign code and whether the stated law captures the user's intent. | Use explicit sequencing and resolved ownership; check required laws against the same plan the runtime executes. Keep provider assumptions visible. |
| Lean 4 | Elaboration and tactics generate terms; a [kernel checks the terms][lean-ref]. | Whether a proposition captures an informal request or an axiom describes a real system. | Separate authoring from validation. Start with transparent checks over a small plan algebra; only import a theorem prover if it buys assurance a simpler checker cannot. |

Go's enabling choice (per its [concurrency talk][go-talk]) is a few direct
coordination primitives, rather than a promise to infer parallelism. Bend
goes the other direction for pure computation; Bend2 explicitly spells the
fork and restricts what it may touch. Lean checks a proposition, not a user
story. Herta should adopt the shared principle: **the mechanism owns exactly
the behavior it claims, and exposes anything it merely trusts.**

## Reproduced counterexamples

**Original:** Building the locked Rust sources at [0.2.38][bend1] and
lowering upstream's tree-sum example emitted four `&!` marks: the two
recursive calls in the generated tree builder and the two in the tree fold.
Running the same program with the HVM 2.0.22 Rust evaluator returned
`16744448`, the sum of 1 through 65535 modulo \(2^{24}\). That demonstrates
the source-to-IR path, not hardware parallel scaling.

**Bend2:** The [fixture][fixture] enumerates every case in a three-state
charge/retry model (not applied, effect unknown, applied). The law is not a
second copy of the policy: it explicitly states the required decision per
outcome, using the actual policy function. Both the fast checker and the
independent Lean-backed verdict produced the same result on all four cases:

| Variant | Fast check | Kernel verdict | What it establishes |
| --- | --- | --- | --- |
| Policy and fixed laws agree | Pass | Pass | Baseline proves the finite cases. |
| Enable retry after an unknown charge; leave laws alone | Reject | Reject before kernel execution | The gate catches this change to the modeled implementation. |
| Keep bad policy and change the unknown law to expect retry | Pass | Pass | A stronger checker cannot protect an edited or incorrect requirement. |
| Replace the proof entry with an unrelated well-typed def | Reject | Reject before kernel execution | At this pin the named proof entry must import the sibling law file. It is not a repository-wide rule unless CI pins the entrypoint. |

**HertaSDK:** The [Go experiment][adapter] exercised the current SDK with an
unchanged contract: non-idempotent, at most two attempts, retry only on
`Transient`. An in-memory backend separately records actual charges so the
oracle does not take the adapter's error tag on faith. No network or clocks
are involved.

| First attempt | Adapter says | Calls | Charges | SDK result |
| --- | --- | ---: | ---: | --- |
| Rejected before charging | Transient | 2 | 1 | Success |
| Charge applied, acknowledgement lost | Uncertain | 1 | 1 | Uncertain |
| Same lost acknowledgement, wrong adapter classification | Transient | 2 | 2 | Success |

The last result is not a new SDK defect: the [documented contract][sdk] trusts
the adapter's evidence. It is a concrete witness against claiming “safe
retry” end to end based on a verified decision table. Conversely, always
refusing a retry would fail the first control, and a blanket ban on
`Uncertain` retries would be wrong for an idempotent reservation that must
retry. The experiments deliberately use different executables: the Bend proof
does **not** prove conformance of the Go SDK. This is a deterministic
mechanism check, not a measurement of bug frequency or agent productivity.

## Smallest Herta experiment worth building

1. **Own the calls.** Start with a Go-constructed, finite `Call` /
   `Then` / outcome-branch plan. The interpreter dispatches each leaf and
   owns retries, with explicit caller deadline vs attempt deadline and a
   named total-attempt limit per governed call. Initially do not hold a
   key or resource claim across a composite step, and do not allow outer
   retries to replay a completed non-idempotent child. A legacy Go handler
   can still run, but the verdict must say its effects are opaque. Do not
   claim whole-program closure: Go packages or adapters with raw backend
   access can bypass the plan. Separate those capabilities by construction
   within the prototype; report uncontrolled calls as uncovered.
2. **Require evidence at the boundary.** “Not applied” is an adapter
   statement backed by specific provider evidence (e.g. local rejection
   before dispatch), not just a generic error or timeout. Otherwise use
   “unknown”; after a non-idempotent unknown, return it or reconcile.
   A Go struct or unexported constructor prevents accidental misuse, not
   a dishonest adapter. Test adapter evidence against an independent
   backend trace. Give operators the decision and reason, not just an
   error code. Do not call an in-process dedupe key exactly-once delivery.
3. **Freeze the policy and try to break the actual executor.** Before
   implementations are run, fix the oracle for charge-after-lost-ack,
   retry-on-unknown for an idempotent reserve, shared key aliasing,
   joined-error priority, and caller/attempt deadline boundaries. Include
   benign neighbors, not just known bugs. If an alias is resolvable from
   inputs, test that rejection happens **before** any debit; if it depends
   on a prior response, surface partial effects rather than claim rollback.
   Only try an explicit parallel operator once workloads show value;
   disjoint Go values alone say nothing about two backend writes, runtime
   capacity, or fairness.
4. **Use an ablation to decide about syntax.** Compare ordinary Go/SDK,
   Go building the enforced plan, and any Herta notation lowering to
   the identical plan. Score causal bug coverage, benign blocks, escapes
   or opaque dependencies, and migration cost; score blind agent attempts
   on a fresh workload instead of growing the pilot task count. Advance
   the Go prototype only if it catches cases the baseline misses without
   blocking the paired valid cases. Advance syntax only if it adds value
   over the same Go API. A finite model checker or Lean proof can come later
   for a named law with an explicit model-to-runtime conformance boundary.

## Reproduction

Run on Linux amd64 with Rust/Cargo for original Bend; Bun 1.3.13 for Bend2;
Lean 4.34.0 (`lean` and `leanc` on PATH, or the exact toolchain installed
through Elan) for `--verdict`; Go 1.26+ for the SDK experiment. The SDK
experiment was run with Go 1.27.4. No native Bend2 speed, GPU behavior or
compiler-correctness claims were tested.

```sh
git clone https://github.com/bendlang/bend /tmp/bendlang-study
git -C /tmp/bendlang-study worktree add --detach /tmp/bend1-study 58372e05fb96ec228cd09310f6a6418f1967a48f
cargo build --locked --manifest-path /tmp/bend1-study/Cargo.toml
cargo install hvm --version 2.0.22 --locked --root /tmp/hvm2-study
(
  cd /tmp/bend1-study
  target/debug/bend gen-hvm examples/parallel_sum.bend | grep -F '&!'
  target/debug/bend --hvm-bin /tmp/hvm2-study/bin/hvm run-rs examples/parallel_sum.bend
)
git -C /tmp/bendlang-study checkout --detach 0e2f38f759cc830849e489307f3d256a7d148288
bash lab/bend/replay.sh /tmp/bendlang-study
bash lab/bend/replay.sh /tmp/bendlang-study --verdict
(cd lab && go test -race -v -count=1 ./bend)
```

[closed]: https://github.com/arahe-dev/HertaSDK/blob/ffe32e6/herta/eval/closed/summary.md
[bend1]: https://github.com/bendlang/bend/tree/58372e05fb96ec228cd09310f6a6418f1967a48f
[lowering]: https://github.com/bendlang/bend/blob/58372e05fb96ec228cd09310f6a6418f1967a48f/docs/compilation-and-readback.md
[priority]: https://github.com/bendlang/bend/blob/58372e05fb96ec228cd09310f6a6418f1967a48f/src/hvm/add_recursive_priority.rs
[hvm1]: https://github.com/HigherOrderCO/HVM1
[hvm2]: https://github.com/HigherOrderCO/HVM2/tree/654276018084b8f44a22b562dd68ab18583bfb5b
[hvm3]: https://github.com/HigherOrderCO/HVM3/tree/fba2e9c82faf6e2f019c9ecea94c32f19a8b7820
[ic]: https://github.com/VictorTaelin/Interaction-Calculus/tree/e9f0185682ba8b41205691efd4f58491dc405a53
[rewrite]: https://github.com/bendlang/bend/commit/c36597a29cf137e629cf472c811755008f69c42e
[launch]: https://github.com/bendlang/bend/commit/a7e0600eda901d791aebcfff4fd6edb2780b4ea4
[redirect]: https://github.com/HigherOrderCO/Bend2/blob/35cf1cb7782a7e7a68335707ed5929b91b89a7c7/README.md
[guide]: https://github.com/bendlang/bend/blob/0e2f38f759cc830849e489307f3d256a7d148288/guide/GUIDE.md
[bendrt]: https://github.com/bendlang/bend/blob/0e2f38f759cc830849e489307f3d256a7d148288/paper/BendRT.pdf
[bendtt]: https://github.com/bendlang/bend/blob/0e2f38f759cc830849e489307f3d256a7d148288/paper/BendTT.pdf
[go-talk]: https://go.dev/talks/2012/concurrency.slide
[go-mem]: https://go.dev/ref/mem
[go-compat]: https://go.dev/doc/go1compat
[lean-ref]: https://lean-lang.org/doc/reference/latest/Elaboration-and-Compilation/
[fixture]: ../../lab/bend/testdata/retry/
[adapter]: ../../lab/bend/boundary_test.go
[sdk]: ../failure-semantics.md
