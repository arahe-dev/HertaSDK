# Transcription rules: real-repo site -> Herta contract model

Goal: write the contract a careful author would *truthfully* declare for the code at the site, then let the checker decide. Never write a contract to make the checker fire or stay silent; transcribe only facts stated in `facts` (and the pinned source if needed).

1. One `op` per logical operation in the path (outer retrying function, each callee that performs a remote/side effect, each auth/refresh step, each lock holder).
2. effect: `nonidempotent` if a replay can create a second externally visible effect (POST that creates, fresh generated name, copy, task creation, append) and no request key/precondition/refetch is visible; `idempotent` for desired-state writes, refetch-and-reconcile, preconditioned/resourceVersion updates, Get-before-Create, fixed-ID writes; `pure` for reads, local mutation, header/query construction, logging, hashing.
3. If a request key / dedupe token / precondition is visible, declare `dedupe key <param>`.
4. retry: `retry max N on {classes}` with N = actual attempt bound (CallNoRetry => no retry clause; unbounded/configurable => the default stated in facts). Classes: retrying unknown/network/5xx/timeout after possible commit => include `uncertain`; retrying only clearly-not-applied errors (401, 429 pre-send, conflict) => `transient`.
5. If an inner retrying step's exhausted failure is still returned as retryable to the outer loop, the outer op `calls` the inner op (amplification is the product).
6. If the inner failure aborts the outer policy (short-circuits), model the inner call as not nested under the outer retry (separate ops, no `calls` edge from the retrying op).
7. Contexts: callee with its own TODO/Background context => `detached`; if facts show explicit lifecycle ownership (cleanup, shutdown, background job, shared load) => `detached for <owner>`. Callee timeout values as `timeout`.
8. Locks: each mutex => `lock <name>` (`shared` for RLock) on the op that holds it while calling. Calls made from go-routines, timer/deferred/registered callbacks, or after Unlock are NOT call edges under the lock (model as a separate op not called by the lock holder).
9. Sites where no remote effect / retry / lock / context exists in Herta terms (pure local work) => write the minimal model anyway (pure op) so it counts in the denominator.
10. Header comment in every model: id, link, one line of facts used. Do NOT include the label.
