#!/usr/bin/env python3
"""Label-blind transcription. Reads facts and pinned source evidence only."""
import hashlib
import json
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROWS = json.loads((HERE.parent / "corpus-facts.json").read_text())
assert len(ROWS) == 121 and len({r["id"] for r in ROWS}) == 121
MODELS = {}


def op(name, effect="pure", clauses="", params="", calls=()):
    contract = "effect " + effect
    if clauses:
        contract += "\n    " + clauses.replace("\n", "\n    ")
    if calls:
        contract += "\n    calls " + ", ".join(calls)
    return (f"op main::{name}({params}) -> u64 {{\n    {contract}\n}}\n"
            f"handler main::{name}({params}) -> (u64, error) {{\n    return 0, nil\n}}\n")


def put(prefix, body, note="", inexpressible=""):
    matches = [r for r in ROWS if r["id"].startswith(prefix)]
    assert len(matches) == 1, (prefix, len(matches))
    key = matches[0]["id"]
    assert key not in MODELS, key
    MODELS[key] = dict(body=body, note=note, inexpressible=inexpressible)


def unsupported(prefix, why):
    put(prefix, op("site"), inexpressible=why)


def retry_path(outer, inner, effect, attempts, classes="uncertain", params="", leaf=""):
    return (op(outer, effect, f"retry max {attempts} on {{ {classes} }}", params, [inner])
            + op(inner, effect, leaf, params))


RCLONE = ("Default 10 total pacer attempts: pinned rclone fs/config.go:196-198; "
          "lib/pacer/pacer.go Call passes p.retries, CallNoRetry passes 1. "
          "Default matches the C2 facts, not an evaluation budget override.")

# C1: replay safety. Effects describe the actual logical operation, never an
# intentionally understated pure/idempotent contract for an effectful wrapper.
put("C1-011467", retry_path("updateNode", "refetchReconcilePatch", "idempotent", 4),
    "Pinned nodeinfomanager.go:56-60 sets updateBackoff.Steps=4. The callee is the logical refetch-and-reconcile update.")
put("C1-03bf5", retry_path("copy", "postCopy", "nonidempotent", 10), RCLONE)
put("C1-043f", op("insertOnce", "nonidempotent", calls=["insert"]) + op("insert", "nonidempotent"),
    "CallNoRetry has one attempt: no retry clause. Insert may create but is not replayed at this site.")
put("C1-04d352", op("rotationBackoff", "nonidempotent", "retry max 5 on { uncertain }", calls=["rotateCerts"])
    + op("rotateCerts", "nonidempotent", calls=["generateKey", "requestCertificate", "waitForCertificate"])
    + op("generateKey") + op("requestCertificate", "nonidempotent") + op("waitForCertificate"),
    "Pinned certificate_manager.go:464-470: 5 attempts in the selected ExponentialBackoff. Lines 471-472 then enter a separate infinite poll, not represented as a fictitious finite retry. uncertain represents failure after CSR commit while waiting.")
put("C1-04ef", retry_path("requestDecompress", "postTask", "nonidempotent", 10), RCLONE)
put("C1-088", op("listCounters", calls=["executeDump"]) + op("executeDump"),
    "Read-only netfilter dump. Execute is not classified by its name.")
for prefix, outer, leaf in [("C1-08af", "listFolders", "listAllFolders"), ("C1-0b99", "about", "getUsage")]:
    put(prefix, op(outer, "idempotent", "retry max 10 on { transient, uncertain }", calls=[leaf, "refreshOrReLogin"])
        + op(leaf)
        + op("refreshOrReLogin", "idempotent", calls=["saveReplacementCredentials"])
        + op("saveReplacementCredentials", "idempotent"),
        RCLONE + " The facts establish 401 recovery; pinned internxt.go:53-76 also retries transport errors. Auth is a recovery step, not an additional retry loop inside the selected read; replacement credentials are desired-state bookkeeping.")
put("C1-08e48", retry_path("requestNewTask", "postTask", "nonidempotent", 10), RCLONE)
put("C1-0a578", retry_path("startSession", "allocateEmptySession", "nonidempotent", 10),
    RCLONE + " Allocating another session is an externally visible allocation even without committed file content; do not silently reclassify it as a read. Pinned dropbox.go:474-500 retries transport errors.")
for prefix, method in [("C1-0a66", "getObjectRange"), ("C1-383", "getObject")]:
    put(prefix, retry_path(method, "readS3", "pure", 5) + op("hashBucketKey"),
        "Pinned s3_storage_client.go:140-142 configures MaxRetries=5; the backoff Ongoing loop bounds actual attempts to 5 (pinned vendored backoff implementation). Hash Write is local pure work.")
put("C1-0b4e", retry_path("dirMove", "renameSameFolder", "idempotent", 10), RCLONE)
put("C1-0b5c", retry_path("cleanupTrash", "clearSameTrash", "idempotent", 10), RCLONE)
put("C1-0ba4", retry_path("uploadUnchecked", "confirmSameBatch", "idempotent", 10), RCLONE)
put("C1-140", op("getPodInfo", calls=["readPod", "extractContainerNames"]) + op("readPod") + op("extractContainerNames"))
put("C1-18b", retry_path("uninstallDriver", "refetchRemoveDriver", "idempotent", 4),
    "Pinned nodeinfomanager.go:56-60 uses 4 attempts; the logical update refetches, filters matching entries, skips clean objects.")
unsupported("C1-2247", "Datasource-specific sample dedupe and re-evaluation determinism are not established; cannot truthfully choose one effect for the polymorphic writer.")
put("C1-224c", retry_path("removeEndpoints", "updateResourceVersion", "idempotent", 4, "transient", "resource_version: string", "dedupe key resource_version"),
    "Pinned instancecount.go:176 uses retry.DefaultBackoff (4 steps, corroborated by C2 facts); resourceVersion is the visible write precondition.")
unsupported("C1-2527", "Startup/periodic bootstrap reconciliation has a context/time-controlled poll, not a supplied finite attempt bound; Herta only has retry max N. Desired-state relist/delete semantics are otherwise idempotent.")
unsupported("C1-28bb", "Only cache/local credential mutation and telemetry at the identified site; no remote IAM mutation to model.")
unsupported("C1-2b11", "Only in-process discovery registry mutations; no remote mutation, governed retry, lock, or context behavior in the supplied facts.")
unsupported("C1-2e51", "Repeated controller reconciliation has no finite retry-attempt bound in the facts; Herta cannot encode that policy. Delete and finalizer reconciliation are explicitly idempotent.")
unsupported("C1-2f277", "Pinned idempotency.go uses PollUntilContextTimeout, not a fixed attempt cap; Herta cannot encode this timed retry exactly. GET-before-Create replay safety is established.")
put("C1-301", op("serveAcceptedConnections", "nonidempotent", calls=["accept", "copyOneConnection"]) + op("accept") + op("copyOneConnection", "nonidempotent"),
    "No retry clause: successful iterations act on distinct connections, not replay the same failed operation.")
unsupported("C1-39c", "The 10-minute periodic StorageVersion reconciliation is not a finite retry policy. Fixed-name desired-state replacement is otherwise idempotent.")
unsupported("C1-3d896", "Concrete provider effect/idempotence and conflict-after-commit behavior are unknown; method name CreateRoute cannot establish an effect contract.")
unsupported("C1-40d1", "Only intentional stderr logging before retry is identified; no replay of an external mutation in the facts.")
unsupported("C1-4d53", "drainBody writes to io.Discard. Opt-in arbitrary-verb transport has no established concrete nonidempotent caller; cannot assign its effect truthfully.")
unsupported("C1-4ea", "Leader acquisition/renewal is context/deadline-driven polling without a finite attempt bound in facts; Herta retry max N cannot express it. Refetch and resourceVersion updates are replay-safe.")

# C2: repeated exhausted recovery stays nested; short-circuited recovery is
# separate by transcription rule 6, even though successful recovery can recur.
for row in ROWS:
    if row["check"] != "C2":
        continue
    ident, facts = row["id"], row["facts"]
    if "authorizeAccount" in facts:
        put(ident, op("requestAuthRecovery", "idempotent", "retry max 10 on { transient, uncertain }", calls=["shouldRetry"])
            + op("shouldRetry", "idempotent", calls=["authorizeAccount"])
            + op("authorizeAccount", "idempotent", "retry max 10 on { uncertain }\nlock authMu", calls=["fetchAndInstallCredentials"])
            + op("fetchAndInstallCredentials", "idempotent"),
            RCLONE + " Projection is the credential-recovery path established in the facts, not unspecified surrounding user mutations. Pinned b2.go:677-696 GETs authorization and replaces current credentials; failed auth remains outer-retryable.")
    elif "ulozto" in facts:
        clauses = "" if "CallNoRetry" in facts else "retry max 10 on { transient }"
        put(ident, op("requestAuthRecovery", "idempotent", clauses)
            + op("authenticate", "idempotent", "retry max 10 on { uncertain }", calls=["installCredentials"])
            + op("installCredentials", "idempotent"),
            RCLONE + " Auth failure short-circuits the outer policy; no calls edge under outer retry, per rule 6. Inner reauth is disabled. Underlying user-request effect is not specified; the modeled operation is credential recovery.")
    elif "linkbox" in facts:
        put(ident, op("requestTokenRecovery", "idempotent", "retry max 10 on { transient }")
            + op("login", "idempotent", calls=["getUnmarshaledResponse"])
            + op("getUnmarshaledResponse", "idempotent", "retry max 10 on { uncertain }"),
            RCLONE + " Failed login aborts outer retry: recovery is separate under rule 6, not a nested call from the retrying op.")

put("C2-10e7", op("createDir", "idempotent", "retry max 10 on { uncertain }")
    + op("findLeafItem", calls=["listAll"]) + op("listAll", clauses="retry max 10 on { uncertain }"),
    RCLONE + " Lookup failure aborts outer retry (rule 6); successful refetch guards against duplicate creation, making logical CreateDir idempotent.")
put("C2-111", op("attachRounds", clauses="retry max 5 on { transient }", calls=["waitForPath"])
    + op("waitForPath", clauses="retry max 30 on { transient }", calls=["statOrGlob"])
    + op("statOrGlob"), "5 attach rounds * 30 local read probes; no invented budget override for intentional discovery.")
put("C2-37fc", op("batch", "nonidempotent", "retry max 3 on { uncertain }", calls=["replicateMultipart"])
    + op("replicateMultipart", "nonidempotent", calls=["createMultipart", "abortMultipart"])
    + op("createMultipart", "nonidempotent", "retry max 3 on { uncertain }")
    + op("abortMultipart", "idempotent", "retry max 3 on { uncertain }"),
    "Facts establish per-operation 3*3=9 attempts, not 3*3*3; create and abort are sequential siblings. Multipart creation allocates a new upload, without a visible request key. Abort desired-state cleanup is idempotent.")
put("C2-42e", op("uploadRecovery", "idempotent", "retry max 10 on { transient }")
    + op("getPosition", clauses="retry max 10 on { uncertain }"),
    RCLONE + " 416 recovery uses the server offset; failed position fetch aborts outer policy, so separate by rule 6.")
for prefix, why in [
    ("C2-437", "Snapshot polling has a 30s time bound, not a finite inner attempt count. Outer conflict bound is 4, but no truthful numeric product can be encoded."),
    ("C2-47cb", "Allocation polling has a 30s time bound with no fixed inner attempt cap; outer conflict bound 4 does not establish a numeric product."),
    ("C2-861", "Watch shares a 2min context with 10s readiness polling; neither time bound establishes an exact attempt count."),
    ("C2-862", "Mutually sequential fast/slow health modes share cancellation; there is no finite attempt product to encode in retry max N."),
    ("C2-ae6", "30s startup polling has no exact outer attempt bound; inner DefaultBackoff=4 alone cannot establish a product."),
]:
    unsupported(prefix, why)
put("C2-b57", op("batch", "nonidempotent", "retry max 3 on { uncertain }", calls=["copyMultipart"])
    + op("copyMultipart", "nonidempotent", calls=["allocateMultipartUpload", "deferredAbort"])
    + op("allocateMultipartUpload", "nonidempotent", "detached for background")
    + op("deferredAbort", "idempotent", "retry max 3 on { uncertain }"),
    "Deferred abort completes before the outer failed upload attempt returns, so its 3 attempts remain nested under the batch's 3 attempts (rule 5). Rule 8 excludes deferred callbacks from held-lock edges, not from this lock-free retry path. Pinned batch-handlers.go:212-245 allocates a fresh multipart upload with Background before copying; the batch job owns it, but no replay precondition/dedupe token is established. Copy/allocation is nonidempotent under rule 2; abort is desired-state cleanup.")
put("C2-b639", op("writeChunk", "idempotent", "retry max 10 on { uncertain }")
    + op("getUploadURL", clauses="retry max 10 on { uncertain }"),
    RCLONE + " Failed URL lookup stops the outer retry (rule 6); successful part uploads reuse/cache the URL. Chunk is a fixed part, not fresh object creation.")

# C3: independent owned lifecycles are not request descendants. All ownership
# declarations below are justified in the corresponding facts, not labels.
def owned(prefix, outer, inner, effect="idempotent", owner="background", timeout="", note=""):
    put(prefix, op(outer, effect, calls=[inner])
        + op(inner, effect, "detached for " + owner + ("\ntimeout " + timeout if timeout else "")), note)

owned("C3-00743", "workerRun", "schedulerLifecycle", "pure", "shutdown",
      note="Explicit schedulerCancel/Wait after thread drain owns shutdown ordering; WaitReady is a read.")
owned("C3-007ab", "watchExpiry", "finish", owner="cleanup", note="Finish cancels owned run and persists outcomes after expiry.")
owned("C3-00828", "onSubscribe", "longLivedWatch", "pure", note="Initialization owns independent long-lived watch, not callback request lifetime.")
owned("C3-009a", "action", "finish", owner="cleanup", note="Async cleanup is a separate op, not a synchronous call under the action.")
# Rule 8 removes the asynchronous edge, even for an explicitly owned cleanup.
MODELS[next(r['id'] for r in ROWS if r['id'].startswith('C3-009a'))]['body'] = op("action", "idempotent") + op("finish", "idempotent", "detached for cleanup")
owned("C3-00a0", "daemonRestore", "checkpointDeadState", note="Startup restore explicitly owns crash repair, not request work.")
put("C3-00dfa", op("operation", "idempotent", calls=["remoteWork"]) + op("remoteWork", "idempotent")
    + op("completionSignal"), "stopCtx derives from caller; runningCtx completion signal does not run the remote effect. No detach.")
put("C3-01a4", op("controllerLoop", "idempotent", calls=["shutdownCloud", "addTaint"])
    + op("shutdownCloud", "idempotent")
    + op("addTaint", "idempotent", "detached", calls=["getNode", "patchDesiredTaint"])
    + op("getNode") + op("patchDesiredTaint", "idempotent"),
    "Loop is cancellation-bearing. Context-free taint helper uses TODO without an explicit separate lifecycle owner; no retry bound is invented.")
put("C3-020b", op("initializeCSINode", "idempotent", "detached", calls=["readAndReconcileCSINode"])
    + op("readAndReconcileCSINode", "idempotent"), "Standalone context-free startup entry, not called from a cancellable operation in the facts.")
put("C3-027", op("openReader", calls=["cachedLength"]) + op("cachedLength"), "Immediate cancellation forbids backend calls; cached-length read is pure.")
owned("C3-02c2", "httpServerRun", "inProcessWatches", "pure", "shutdown", note="Explicit endWatches on ctx.Done and deferred cancellation establish ownership.")
owned("C3-02cb", "gcRun", "cleanupInactiveValues", owner="background", note="Background GC owns per-batch timeout; configured duration is unspecified, not invented.")
owned("C3-031", "rootShutdown", "metricsShutdown", owner="shutdown", timeout="5s")
for prefix in ["C3-034", "C3-059", "C3-07b2"]:
    unsupported(prefix, "Facts establish ctx forwarding in equal, but not the incoming detached chain; closure-scope taint cannot be expressed as a verified call path.")
put("C3-046", op("credentialInitialization", "idempotent", "detached\nretry max 4 on { transient }", calls=["getOrCreateServiceAccount"])
    + op("getOrCreateServiceAccount", "idempotent"), "Context-free credential startup/refresh entry; no established cancellable request caller. Four startup steps are explicit in facts.")
owned("C3-04a4", "maintenance", "mergeShardedRequests", note="Actual callers use Background for startup/finished-batch maintenance.")
owned("C3-05c2", "serviceDrain", "processQueuedBatch", owner="shutdown", note="Queue/state owns multi-request batches while Running or Stopping, not any single task ctx.")
owned("C3-05d7", "workerRun", "schedulerLifecycle", "pure", "shutdown", note="Same explicit schedulerCancel/Wait lifecycle; errgroup inherits scheduler ctx.")
owned("C3-066", "gracefulStop", "cancelRemoteRun", owner="cleanup", note="Separate live force-cancellation channel intentionally cancels the remote run after graceful stop.")
put("C3-06d", op("interactiveConfig", "idempotent", "timeout 10s", calls=["getAuthorizationToken"])
    + op("getAuthorizationToken", "idempotent"), "Authentication inherits the bounded 10s interactive config context; no detach.")
put("C3-075", op("startupPeriodicRepair", "idempotent", "detached", calls=["listServices", "reconcileAllocation"])
    + op("listServices") + op("reconcileAllocation", "idempotent"), "Context-free startup/periodic entry; no request cancellation chain or retry attempt count established.")
put("C3-078", op("successfulAccountUpdate", "idempotent")
    + op("propagationJob", "idempotent", "detached for background", calls=["mutateZanzana"])
    + op("mutateZanzana", "idempotent"), "Success hook launches a separate postcommit background job (rule 8); configured timeout value is not given.")
owned("C3-07c1", "serve", "resolveAndBind", "idempotent", note="Service-listener bind is owned startup; cancellation closes listener and is checked by accept loop.")
owned("C3-082", "restoreStoppedContainer", "cleanupNetwork", owner="cleanup", note="Restore-owned cleanup survives cancellation.")
key = next(r['id'] for r in ROWS if r['id'].startswith('C3-082'))
MODELS[key]['body'] += op("deleteContainer", "idempotent", "detached for cleanup")
put("C3-620", op("reportCrash", "idempotent", calls=["head", "put"])
    + op("head", clauses="timeout 10s") + op("put", "idempotent", "timeout 60s"),
    "HEAD and PUT contexts independently inherit original ctx: sibling edges, not HEAD -> PUT. Object PUT is a desired-state write.")
put("C3-95a", op("diskInfo", calls=["noopBranch", "cacheLoaderBranch"])
    + op("noopBranch", clauses="timeout 5s") + op("cacheLoaderBranch"),
    "Disjoint branches, no child edge from NoOp to cache loader; loader timeout value is not supplied.")
put("C3-b6c", op("refreshRead", clauses="timeout 10s")
    + op("sharedCacheLoad", clauses="detached for background", calls=["tocRead"])
    + op("tocRead", clauses="timeout 10s"),
    "Callback receives a new shared-cache load context, not refreshRead ctx; independent entry points.")

# C4: scopes are specialized to reachable branches. Asynchronous callbacks and
# reacquisition after Unlock are separate operations, with no held-lock edge.
def lock_pair(prefix, outer, inner, lock="mu", shared=False, nested=False, note=""):
    clause = "lock " + lock + (" shared" if shared else "")
    put(prefix, op(outer, clauses=clause, calls=[inner] if nested else []) + op(inner, clauses=clause), note)

lock_pair("C4-002", "clientLock", "monitorLockLoss", note="Monitor runs on a goroutine after Lock returns; no held-lock call edge.")
lock_pair("C4-008727589243-1", "closeAfterGrace", "closeAfterRelease", note="_actualClose Unlocks before downloader close, then restores lock; separate held regions.")
lock_pair("C4-008727589243-2", "actualClose", "ensureAfterRelease", note="_ensure first Unlocks, then defers Lock; no overlapping acquisition.")
put("C4-008727589243-3", op("closeNoSyncWriteBack", clauses="lock mu", calls=["actualCloseNoStore"])
    + op("actualCloseNoStore"), "syncWriteBack=false excludes _store; no edge to an unreachable branch.")
lock_pair("C4-008727589243-4", "closeBeforeEnqueue", "registeredStore", note="Unlock precedes enqueue; store callback is independent writeback work.")
lock_pair("C4-044", "cleanupAttachedConnections", "timerClose", note="Timer callback is independent; immediate branch Unlocks first.")
put("C4-078", op("applianceInfo", clauses="lock tokenMu", calls=["rpcWithToken"])
    + op("rpcWithToken"), "token=* excludes getToken; no edge into tokenMu reacquire.")
lock_pair("C4-08cf", "safePathCloseEmpty", "path", lock="safePathMutex", nested=True,
          note="Reachable repeated Close path holds mutex while synchronously calling Path.")
lock_pair("C4-095", "getOrCreateIndexSet", "asyncInitCleanup", lock="indexSetsMtx", note="Init and cleanup run in goroutine431.")
lock_pair("C4-0b0", "registryGet", "cleaner", lock="mut", note="go cleaner has an independent stack; get releases mut.")
unsupported("C4-0ba", "HTTP verb/effect unspecified; no lock or singleflight behavior to express. Pure placeholder is not a claim about arbitrary HTTP methods.")
lock_pair("C4-0e48", "healBucket", "peerConfigureReplication", lock="siteReplication", shared=True, nested=True,
          note="Both reachable synchronous frames RLock the same embedded RWMutex.")
put("C4-0ed", op("purgeCheck", clauses="lock mkdirMu", calls=["findRootNoCreate"])
    + op("findRootNoCreate"), "create=false returns before mkdir; no edge into mkdirMu reacquire.")
lock_pair("C4-13cd", "healGroups", "iamChangeHook", lock="siteReplication", shared=True, nested=True,
          note="Both reachable synchronous frames RLock the same embedded RWMutex.")
lock_pair("C4-16b", "resolve", "timerRefresh", lock="networkMutex", note="Constructing returned closure does not execute it; timer takes lock on independent stack.")
lock_pair("C4-183", "deviceManagerBeforeRpc", "deviceManagerAfterRpc", lock="deviceManagerMutex", note="Helper releases before plugin RPC and reacquires afterwards; held regions do not nest.")
put("C4-1a24", op("deleteUnsyncedWatcher", clauses="lock cacher", calls=["stopWatcherLocked"])
    + op("stopWatcherLocked"), "Locked helper consumes caller-held lock, does not acquire another lock.")
lock_pair("C4-1b55", "handleAction", "asyncFinish", lock="opMu", note="go Finish executes independently; Handle can return and release opMu.")
lock_pair("C4-1b8", "constructForgetClosure", "invokeForgetClosure", lock="cacher", note="Returning func(bool) does not invoke it; later callback acquires after constructor releases.")
put("C4-1c7", op("postingsBeforeYield", clauses="lock postings")
    + op("postingsYieldRead", clauses="lock postings shared")
    + op("postingsAfterYield", clauses="lock postings"), "Unlock -> RLock/RUnlock -> Lock are separate regions; no nested edge.")
put("C4-1cb", op("read", clauses="lock mu", calls=["readErrAlreadyLocked"]) + op("readErrAlreadyLocked"),
    "readErr(true) cannot execute !locked acquisition branch.")
lock_pair("C4-20b", "isOffline", "initHealthChecks", lock="hMutex", shared=True, note="go initHC is independent, caller immediately returns.")
for prefix, inner in [("C4-240bff32acd5-1", "reconnectWatcher"), ("C4-240bff32acd5-2", "controlSocketChanged"),
                      ("C4-240bff32acd5-3", "handleNodeExit"), ("C4-240bff32acd5-4", "handleReadyEvent")]:
    lock_pair(prefix, "start", inner, note="Acquisition runs on explicitly separate goroutine; no synchronous held-mu call edge.")
lock_pair("C4-2ade", "addState", "receiveLoopMerge", lock="statesMtx", note="go receiveLoop publishes then acquires independently.")
lock_pair("C4-2d92", "constructTranscoder", "invokeTranscoder", lock="transcoderMutex", note="Returned function calls getTranscoder only when later invoked after deferred Unlock.")
lock_pair("C4-2eac", "checkObject", "removeAfterRelease", note="_remove Unlocks before writeback.Remove, then restores Lock; separate regions.")
lock_pair("C4-311", "ensureSynced", "processEventsBatch", lock="cacheLock", note="go processEvents runs independently while ensureSynced can release cacheLock.")


def main():
    assert set(MODELS) == {r["id"] for r in ROWS}, sorted({r["id"] for r in ROWS} - set(MODELS))
    assert not (HERE / "models.sha256").exists(), "Models are frozen; do not rewrite after validation/scoring"
    audit = []
    for row in ROWS:
        model = MODELS[row["id"]]
        header = f"// id: {row['id']}\n// link: {row['link']}\n// facts: {row['facts']}\n"
        if model["note"]:
            header += "// model: " + model["note"] + "\n"
        if model["inexpressible"]:
            header += "// inexpressible: " + model["inexpressible"] + "\n"
        text = header + "runtime main { resource evaluation capacity 1 }\n" + model["body"]
        (HERE / "corpus" / (row["id"] + ".he")).write_text(text)
        audit.append({"id": row["id"], "note": model["note"], "inexpressible": model["inexpressible"],
                      "sha256": hashlib.sha256(text.encode()).hexdigest()})
    (HERE / "transcription.json").write_text(json.dumps(audit, indent=2) + "\n")
    print(f"Wrote {len(audit)} label-blind models; labels have not been opened.")


if __name__ == "__main__":
    main()
