# Herta composition checker: agent-pilot evaluation

- Checker: `e9741d23d663ea1ad8e708a5f0e88393424b4416`; unchanged from the prototype. No checker fixes.
- Transcriptions frozen before checking: `2026-11-19T17:03:57.217309+00:00`.
- Frozen manifest SHA-256: `3bdb39a76c337954cd37080073b483d6c795a48409bd9399ab4781eb34106b12`; all 17 models remain unchanged.
- Evidence inventory includes SHA-256 and original path for every captured input. `diagnostics.json` records every checker output, including all sanity-fixture diagnostics; `results.csv` includes all 345 checks.

## A. Unmodified real programs: false positives

Checked all t1–t4 final `.he` files from herta-1/2/3, ref-herta, and ref2-herta: **20 logical programs, 32 physical files**. The 12 oracle2-workspace mirrors of agent submissions are byte-identical and checked separately, not double-counted. Original agent submissions are the primary copies; the retained oracle2 workspace supplies ref-herta, and workspace supplies ref2-herta.

**Every unmodified check exited 0 with no diagnostics.** Consequently there are no diagnostics to mark true-bug versus false-positive. File flag rate: 0/20. False-positive rate among oracle2-passing program files: **0/16 (0.0%)**. Agent-only: 0/11 oracle2-passing files (0/12 file flag rate).

All 12 agent task files pass the original oracle. Oracle2 fails herta-2/t1 and ref-herta/t1,t3,t4; these four known-bug programs are also accepted, not false positives. Original reference-oracle results were not retained in results/summary.csv, so are not invented here. Corrected ref2-herta passes all four oracle2 tasks. Oracle evidence was cross-checked from the captured summaries and per-task result/race logs.

These programs classify errors in raw Go helpers, not `classify` blocks, so classification diagnostics cannot fire unmodified. Most nested calls also go through opaque `servicePtr` methods; the checker does not perform whole-Go-program method/field dataflow. Zero false positives is **not** evidence of source-level bug discovery or proof that accepted programs are safe.

### Separate vendored sanity set

Excluded all vendored hertasdk files from the primary denominator. Checked **296 physical `.he` fixture/example copies, 36 distinct contents**. 112 copies emit diagnostics; **296/296 outputs and exit codes match the original PR9 checker byte-for-byte**. These include intentionally invalid parser/checker fixtures, not agent bugs or false positives. Full individual outputs are in diagnostics.json, with original paths/hashes in inventory.json and baseline outputs/hash in sanity-baseline.json. No new sanity regression or crash was observed.

## B. Frozen escaped-bug transcriptions

Contracts expose logic that lives in Go. They are projections, not unchanged agent programs. Every t1 model includes the task author's supplied `error ErrTimeout ambiguous`, nonidempotent charging, three transient-only attempts, and the actual ordered classifier. Nil-success handling is omitted; context names are normalized. Untagged context errors have nominal permanent classification; SDK deadline promotion is dynamic and is not modeled. This does not affect the ErrUnavailable+ErrTimeout comparison.

### Actual t1 classifier order and oracle2 comparison

| Run | Checker diagnostics | Oracle2 t1 | Agreement |
| --- | --- | --- | --- |
| go-1 | HC010 | fail | yes |
| go-2 | HC010 | fail | yes |
| go-3 | HC010 | fail | yes |
| herta-1 | none | pass | yes |
| herta-2 | HC010 | fail | yes |
| herta-3 | none | pass | yes |
| ref-go | HC010 | fail | yes |
| ref-herta | HC010 | fail | yes |
| ref2-go | none | pass | yes |
| ref2-herta | none | pass | yes |

**Agent flags are exactly go-1, go-2, go-3, herta-2; herta-1 and herta-3 remain silent.** All 6/6 agent predictions match oracle2. Original references (both Go and Herta) are flagged; corrected references are silent, giving 10/10 comparisons across all source variants. The reference pairs are controls, not additional agent runs.

Unavailable-first classifiers produce HC010 because joined errors can match both arms. In the actual oracle2 scenario, t1 Service.Charge dispatches the governed charge handler, the backend applies whenever errors.Is(err, ErrTimeout), the handler checks ErrUnavailable first, and the transient-only retry policy replays it. The retained race logs report `uncertain charge repeated/misreported: err=<nil> outcome=Success charges=2 calls=2`. As a disconfirming control, timeout-first herta-1/herta-3 and both ref2 variants pass the same joined-error oracle. Ordered transcription and positive/negative controls match this causal chain; no hypothetical source reachability is substituted for oracle evidence.

### Per §3 bug class

| Class | Caught? | Diagnostic / evidence |
| --- | --- | --- |
| Declared nonidempotent + retry uncertain (already caught baseline) | yes | LEGACY_UNSAFE_RETRY; local policy baseline |
| Timeout classified transient | yes | HC005;HC011; task's t1 trap, not observed in final six submissions |
| Joined ErrUnavailable+ErrTimeout retried | yes | HC010; 4/6 agents; original references also flagged |
| Nested op reacquires same key | yes | HC001; explicitly labeled fallback trap; not recovered agent source |
| Idempotent wrapper + nonidempotent call + retry | yes | HC004;HC005; declared edge exposes the probe mechanism |
| Empty serialize key disables serialization | **not caught** | none; out of scope: no nonempty-key/value analysis |
| Codegen unused-parameter defect | **not caught** | none; out of scope: static check does not compile generated Go |

Of the five originally uncaught semantic classes in §3, **4/5 are caught in these contract/trap projections**; the empty-key class is not. The already-caught local unsafe retry remains caught. The separately listed codegen defect is also not caught. These are class-level demonstrations, not independent real-source detections.

The exact retained `probes/pr9/composition/traps.he` is additionally frozen and checked unchanged: **HC001;HC004;HC005**. This exercises whitespace-tolerant typed `.Do` extraction as well as the explicit-clause models.

### herta-2 intermediate recovery limitation

Inspected both retained final t4 workspaces, score.log, supplied-policy.diff, and the supplied pilot artifact/log inventory. No intermediate revision, git history, or agent editing transcript is retained in these snapshots. Used the user-authorized same-key nested-call trap fallback, clearly labeled in the model and transcription.json. This is **not** a claim that the intermediate implementation was recovered or that the final herta-2 transfer deadlocks. The final implementation uses a NUL-prefixed transfer key around debit and credits outside that lock; it passes oracle2. Oracle completion does not establish full two-step Transfer serialization.

## Reproduction and limits

Run `python3 pilot/score.py` from the evaluation directory; it verifies checker/model/evidence hashes before checking. The original PR9 baseline is needed only to generate missing sanity-baseline entries; the frozen captured baseline allows replay without the pilot checkout. `build_pilot.py` is the evidence/transcription builder and deliberately refuses to overwrite an existing freeze. It requires the original pilot paths and is not needed to score captured inputs.

No transcriptions were changed after checker scoring. Empty-key and codegen defects are explicitly not caught. No runtime compilation, new oracle runs, or arbitrary Go alias/value inference was added. The oracle outcomes quoted here are the retained pilot evidence, not fresh executions. No UI or visual verification is applicable.
