# Go, Bend/Bend2 and Lean in Herta: what was built and what it measured

Three studies (Go primitives, [Bend/Bend2](https://github.com/arahe-dev/HertaSDK/blob/main/docs/research/bend.md),
[Lean 1–4](LEAN.md)) plus the closed-world evaluation and the 30-agent pilot
converged on three steps. This change implements each one and measures it
against an equally typed Go SDK baseline before keeping any syntax.

| Lesson | Source | What Herta now does |
|---|---|---|
| Values in a signature do not say whether a remote write happened; an error is not evidence | Go (errors, `context`, cooperative cancellation) | `hertasdk.Dispatch` marks the point a request can leave the process; after it, `Transient` ("not applied") is downgraded to `Uncertain` |
| Keep effectful structure executable, use one-use capabilities only for governed actions | Bend (interaction nets keep the program structure; Bend2 affine use) | `call op(args)` in a handler body is executed through a generated plan; the edge the checker walks is the call that runs |
| Small trusted kernel, explicit and named assumptions | Lean (kernel vs elaborator, `#print axioms`) | Theorem T6 replaces "the classifier never lies" with "the adapter dispatches before applying" |

## 1. Dispatch evidence (SDK, plain Go API)

`Policy.TrackDispatch` gives each attempt a state on its context.
`Dispatch(ctx)` marks that attempt and every tracked ancestor attempt. On
failure, the verdict from either path (legacy tag or closed `Classify`) is
corrected by evidence:

- `Transient` after a dispatch becomes `Uncertain`, which a `NonIdempotent`
  operation never retries;
- a dispatch by a `NonIdempotent` descendant forbids an enclosing
  `Idempotent`/`Pure` attempt from retrying `Uncertain` (the T4_composition
  hazard: an outer retry replaying an inner charge).

The adapter's obligation shrinks from "label every error honestly" to "call
`Dispatch` before the first byte can leave". That obligation is still
unchecked: the matrix row "adapter sends before dispatch point" double-commits
on every arm. Cost on an uncontended call (Go 1.27.4, linux/amd64, 3 × 200k):
plain 129–183 ns, 112 B, 2 allocs; tracked 222–279 ns, 184 B, 4 allocs.

## 2. Executable call plans (Herta)

`call name(args)` was already a checker edge (HC001–HC013 walk it) but was
emitted verbatim, so it was not valid Go and the code that ran could differ
from the graph that was checked. Now, in a runtime whose handlers use `call`,
`herta build` emits `<Runtime>Plan` with one field per operation and
`New<Runtime>Plan(rt)`; each `call` lowers to `p.<Op>.Do(ctx, <Input>{...})`
with positional arguments mapped to the callee's parameters (an arity
mismatch or unknown callee fails the build). A statement that is only a call
returns the callee's error unchanged. Closed `Calls` comes from the same
edges, so the declared, checked and executed plans are one list.

`evidence dispatch` sets `TrackDispatch`; a handler statement `dispatch`
lowers to `hertasdk.Dispatch(ctx)`. Because the plan is visible, `herta build`
also turns on tracking for every operation that can reach a tracked one.
This is the only measured place where Herta beats the same Go API: in Go the
caller must remember `TrackDispatch` on every enclosing operation.

## 3. Measured matrix

`go test -race ./herta/eval/synthesis` runs every scenario on a fresh runtime
against a fake provider whose committed writes are ground truth. Expectations
were written into the test before the first run; all 25 cells matched.

| scenario | plain Go SDK | typed Go SDK | typed Go SDK + caller annotation | Herta, no evidence clause | Herta plan, evidence dispatch |
|---|---|---|---|---|---|
| pre-send refusal | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success |
| lost ack, honest adapter | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain |
| lost ack, adapter says not sent | 2 commit(s), Success | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 2 commit(s), Success | 1 commit(s), Uncertain |
| nested: checkout -> publish, mislabel | 2 commit(s), Success | 2 commit(s), Success | 1 commit(s), Uncertain | 2 commit(s), Success | 1 commit(s), Uncertain |
| adapter sends before dispatch point | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success |

Readings:

- **The boundary study's bug is fixed without trusting the label.** A lost ack
  reported as "not sent" commits twice in plain Go and in closed Herta without
  the evidence clause; tracked Go and the Herta plan commit once and return
  `Uncertain` for reconciliation.
- **Nested plans are where syntax helps.** Typed Go without the caller
  annotation still double-commits through the outer retry; Herta infers the
  annotation from the plan. Typed Go with the annotation is equal to Herta.
- **Pre-send refusals still retry.** Tracking does not make transient-before-
  dispatch failures permanent.
- **Not fixed:** an adapter that applies before `Dispatch`, rollback of an
  applied effect, calls made on an unrelated context, opaque host helpers, and
  every finding of the pilot (which showed no measurable Herta benefit and is
  not rerun here).

## 4. Formal result

`herta/formal/Herta.lean` adds `Policy.trackDispatch`, `Reply.dispatched` and
`evidence`. T6: for a nonidempotent tracked operation, any classifier and any
backend satisfying `DispatchBeforeApply` (applied ⇒ dispatched) gives at most
one application. `T6_witness` evaluates the mislabel trace: 2 applications
untracked, 1 tracked, 2 when the backend applies before dispatch. `check.sh`
audits T6 and T6_witness for standard axioms only. Ancestor marking (the
nested row) is tested in Go, not modeled.

## Reproduce

```sh
go test -race -count=1 . ./herta/...
SYNTHESIS_TABLE=/tmp/t.md go test -count=1 -run Matrix ./herta/eval/synthesis
go test -run '^$' -bench Publish -benchtime=200000x -count=3 ./herta/eval/synthesis
(cd herta/formal && ./check.sh)
```
