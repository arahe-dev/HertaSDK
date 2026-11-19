# Executable SDK contract model

See [Lean lessons for Herta](../docs/LEAN.md) for the pilot evidence, design mapping,
theorem results, trust-ledger evaluation and proposed experiments.

Lean 4.36.0 (the installed stable release), core/Std only, no external dependencies.
Run `./check.sh` from this directory (elan's `lake` must be on PATH).

`Herta.lean` models discrete milliseconds, backend applications/errors, both
legacy handler tags and closed-world Policy.Classify, per-attempt/caller
deadlines, bounded retries and counters. Comments cite the Go source at
`ffe32e6`. This is a model of the contract, not a verified Go extraction.

- T1: nonidempotent safety under the explicit Transient/no-effect trust assumption.
- T1_assumption_necessary: honest deadline handling alone does not imply safety.
- T2: legacy own-deadline Transient tags can duplicate an applied effect.
- T3_per_attempt: 300ms x max-4 with a 1000ms caller deadline gives 4 calls/1000ms.
- T3_default: default Uncertain promotion gives 1 call/300ms, within oracle 550ms.
- T4_key: a synchronously held same key cannot be reacquired before any deadline.
- T4_composition: a Pure/Idempotent outer retry can duplicate a nested inner effect.
- T5: closed-world own-expiry overrides any classifier, preventing that retry.
- T6: with dispatch tracking, any classifier is safe if the adapter dispatches before applying.

## Boundaries and trust

The backend's `applied` flag means one application per invocation, including
effects performed before cancellation. Arbitrary raw Go behavior is not proven
honest: T1 takes honesty as a hypothesis, never as a new Lean axiom. `hangs`
models cooperative context termination; other replies model noncooperative late
returns. Classifiers see attempt index and raw error; legacy `some` represents
the first effective Failure tag. Error chains/identity, invalid enum integers,
explicit cancellation separate from caller deadline, panics, shutdown, resource
queues, scheduling races and actual goroutine nontermination are abstracted out.
Caller deadlines are finite, absolute; no-deadline behavior can be approximated
by a deadline larger than the execution. Backoff is an optional extension:
use zero for SDK behavior. Elapsed time includes elapsed waits/returns, not CPU
overhead. Effects are counted as applications, not deduplicated state changes.

Keys are modeled separately as a stable held-key set during synchronous nested
Wait, with Nat standing for a nonempty key string. Reject fails immediately in
Go; closed-world key-reacquisition guards can fail immediately too: T4_key is
the underlying legacy Wait behavior, not a claim that guarded closed calls hang.
The composition witness aggregates one successful inner application into each
outer backend reply; it does not execute a second full inner interpreter.

The check script forces proof replay even on cached builds, prints every theorem's
axiom audit, and permits only `propext`, `Classical.choice`, `Quot.sound`.
