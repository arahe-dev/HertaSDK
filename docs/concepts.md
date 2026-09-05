# Concepts

The six primitives of HertaSDK. One execution model, many heterogeneous
operations — every primitive below exists to make shared in-process execution
explicit and checkable.

> Status: **v0.1.0 — frozen contract.** The names below are the real,
> installed public API (`github.com/arahe-dev/hertasdk`). The V0 execution
> contract will not change without consumer evidence.

## Runtime

The execution authority and lifecycle owner for one process.

- Owns named resource budgets and admission policy.
- Validates operation contracts at construction.
- Arbitrates execution across otherwise-independent subsystems.
- Owns shutdown: reject new work, wake waiters, drain active handlers.

There is exactly one runtime per process in V0. It is local: no network, no
daemon, no cross-process coordination.

## Operation[I, O]

A typed wrapper around a normal Go handler.

- The handler stays ordinary: it computes, stores, returns. Herta never owns
  business logic.
- The wrapper carries a `Policy` declaring resources, admission, timeout,
  retry, and optional keyed serialization.
- Execution goes through the runtime: `Do(ctx, input) -> (output, error)`.

Typing the operation (`I` input, `O` output) keeps heterogeneous operations
distinct at the call site while sharing one arbitration path underneath.

## Resource

A named finite local capacity.

- Examples: `renderer` (capacity 8), `db-write` (shared by events and
  catalogue), `model-call` (quota).
- Weighted: an operation may consume more than one unit.
- Shared: different operations may consume the same resource. This is the
  point — see [resource-arbitration.md](resource-arbitration.md).
- Local: V0 budgets are per process. There is no global coordination.

## Policy

Everything the runtime needs to execute an operation safely:

- **Resources**: which named budgets, what weights. Acquired deterministically
  (fixed order) with rollback after partial acquisition.
- **Admission**: `Wait` (queue for capacity) or `Reject` (fail fast when full).
- **Timeout**: cooperative per-attempt `context.Context` deadline.
- **Retry**: which outcomes may be retried — validated against `Effect` at
  construction. Unsafe combinations are rejected before serving.
- **SerializeBy(key)** (optional): same-key serial, different-key concurrent.
  Key waiters hold no downstream capacity while waiting.

Ordering: key acquisition happens **before** resource acquisition, so a waiter
on a key never holds scarce downstream capacity.

## Effect

Whether repeating the operation is safe:

| Effect | Meaning | Retry posture |
|---|---|---|
| `Pure` | No observable side effects. | Safe to repeat. |
| `Idempotent` | Repeating has the same effect as doing once. | Safe to repeat. |
| `NonIdempotent` | Repeating may change state, spend quota, or charge money. | Repeat only with proof it did not execute. |

Effect is declared by the operation author — the runtime cannot infer it.

## Outcome

What happened during one attempt:

| Outcome | Meaning |
|---|---|
| `Success` | The attempt completed. |
| `Transient` | Failed in a way a retry may fix (e.g. momentary contention). |
| `Permanent` | Failed in a way a retry will not fix. Do not retry. |
| `Throttled` | Rejected by admission or a budget. Back off; do not hammer. |
| `Uncertain` | Unknown whether the operation executed. The dangerous one. |

Retry decisions require **both** Effect and Outcome. `Uncertain` on a
`NonIdempotent` operation with retry configured is an invalid contract —
rejected at construction. See [failure-semantics.md](failure-semantics.md).
