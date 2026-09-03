# Resource arbitration

The defining feature of HertaSDK: heterogeneous operations share named finite
budgets, and the runtime arbitrates fairly instead of letting each subsystem
invent its own semaphore.

## Shared local resources

A resource is a named capacity with a weight system:

```text
renderer   capacity = 8
db-write   capacity = 4
model-call capacity = 2
```

Operations declare claims: `{ renderer: 1 }`, `{ db-write: 2 }`. The runtime
admits an operation only when every claimed budget has room, acquiring in
deterministic order with rollback on partial failure.

The point is sharing: the render path, the event ingester, and the catalogue
writer can all draw from `db-write`. Capacity planning happens once, in one
place, instead of three pools sized by three guesses.

## Weighted capacity

Not all work costs the same. A full-page render may claim weight 2 while a
thumbnail claims weight 1; a bulk catalogue replace may claim weight 3 on
`db-write` while a single event claims 1. Weights are declared in the policy
and validated at construction (positive, finite, within budget).

## Heterogeneous operations consuming the same resource

```text
Events ─────────┐
                ├→  db-write (capacity 4)
Catalogue ──────┘

Render ×20 ──→  renderer (capacity 8)  →  peak inside: exactly 8
```

Measured in the internal reference implementation: 20 concurrent render calls
against capacity 8 produced an observed peak of exactly 8 inside the provider.
No more, no fewer — arbitration is exact, not approximate.

## Admission under contention

When a budget is full, the operation's admission policy decides:

- `Wait`: queue without holding downstream capacity. Fair wake-up on release
  or shutdown.
- `Reject`: return `Throttled` immediately. The caller degrades, sheds, or
  retries at its own layer — Herta does not retry throttled work blindly.

## Per-process limitation

Herta V0 arbitrates **within one process**. It does **not** coordinate
resources globally across processes, hosts, or containers.

- Two processes each running a renderer budget of 8 can hold 16 total.
- If you need global limits, enforce them at the shared provider (database
  admission, upstream rate limits) — Herta stays the local layer.
- No distributed semaphore, no consensus, no network. That is intentional:
  Herta remains local until evidence proves a distributed layer is necessary.

See [execution-model.md](execution-model.md) for acquisition ordering and
[concepts.md](concepts.md) for the `Resource` primitive.
