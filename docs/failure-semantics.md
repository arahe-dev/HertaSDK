# Failure semantics

Retry decisions depend on **both** what happened (`Outcome`) and whether
repeating is safe (`Effect`). Herta refuses to guess either one.

## Classification

Every attempt ends in exactly one outcome:

| Outcome | Retry? |
|---|---|
| `Success` | Done. No retry. |
| `Transient` | Retryable, if the policy allows and `Effect` permits. |
| `Permanent` | Never retried. |
| `Throttled` | **Not retried in V0.** Herta has no backoff/jitter semantics, so an immediate retry would only hammer the resource that just refused us. `RetryPolicy.On[Throttled] = true` is rejected at construction. |
| `Uncertain` | Retryable **only** if `Effect` is `Pure` or `Idempotent`. Never for `NonIdempotent`. |

Since v0.2.0 the retry policy must only request what the contract implements:
`On` entries for `Throttled`, `Success`, `Permanent`, or unrecognized
`Outcome` values are construction errors (`ErrInvalidRetryPolicy`), not
silently ignored settings. The actionable entries in V0 are `Transient`,
and `Uncertain` where the `Effect` permits it.

Classification is the **handler's** (or its provider adapter's) job — the
code closest to the external system tags errors with `Fail(outcome, err)`.
Anything untagged is unclassified — and unclassified Go errors are treated
conservatively: **no speculative retry**.

## Retry safety

The retry table is the heart of the contract:

|  | Pure | Idempotent | NonIdempotent |
|---|---|---|---|
| Transient | retry ok | retry ok | retry only if proven unexecuted |
| Uncertain | retry ok | retry ok | **invalid contract** |
| Permanent / Throttled | no blind retry | no blind retry | no blind retry |

`Effect = NonIdempotent` + `Outcome = Uncertain` + retry configured for
`Uncertain` is rejected at construction, before serving.

## Uncertain semantics

`Uncertain` means: the attempt may or may not have executed. The external
operation may already have consumed quota, charged money, sent a message, or
mutated state.

Retrying an uncertain non-idempotent operation is not a transport decision —
it is a business-safety decision. The contract declares it unsafe, so the
runtime refuses to build that operation at all. Fix the contract (remove the
retry, prove idempotence, or make execution observable) instead of hoping.

## NonIdempotent restrictions

- A `NonIdempotent` operation may still retry `Transient` failures **only**
  when the runtime can prove the attempt did not execute (e.g. rejected before
  dispatch, failed before any side effect).
- Timeouts on non-idempotent operations classify as `Uncertain` unless the
  provider proves otherwise — a timed-out call may have completed server-side.
- When in doubt, the runtime does not retry. Conservative is correct.

## Caller cancellation

`ctx` cancellation by the caller is not a failure of the operation — it is a
withdrawal of interest:

- The attempt stops (cooperatively) and no retry follows.
- Waiting operations (admission, key, resources) wake with a cancellation
  error without consuming capacity.
- Cancellation never classifies as `Transient`.

## Plain error conservative behavior

A plain Go `error` with no classification metadata is **unclassified**:

- It does not map to `Transient` on optimism.
- It does not trigger retries the contract did not explicitly allow.
- It surfaces to the caller with its message intact.

The burden is on providers and operation authors to declare meaning.
Silence means "do not guess."
