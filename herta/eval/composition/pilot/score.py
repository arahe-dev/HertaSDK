#!/usr/bin/env python3
"""Check frozen pilot projections and captured unmodified programs, never models."""
import collections
import csv
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parent
CHECKER = ROOT.parent / "herta"
BASELINE = Path.home() / "work/pilot/probes/pr9/herta"


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check(path, checker=CHECKER):
    p = subprocess.run([str(checker), "check", str(path)], capture_output=True, text=True, timeout=5)
    return dict(exit=p.returncode, stdout=p.stdout, stderr=p.stderr)


def codes(result, strict=True):
    found = []
    for line in (result["stdout"] + result["stderr"]).splitlines():
        m = re.match(r"^\d+:\d+: (HC\d{3}) path ", line)
        if m:
            found.append(m[1])
        elif re.fullmatch(r'\d+:\d+: op "[^"]+": retrying uncertain outcomes is unsafe for effect nonidempotent', line):
            found.append("LEGACY_UNSAFE_RETRY")
        elif strict:
            raise AssertionError(f"Unexpected diagnostic: {line}")
        else:
            found.append("PREEXISTING_FIXTURE_DIAGNOSTIC")
    if strict:
        assert result["exit"] in (0, 1) and not result["stderr"], result
        assert bool(found) == (result["exit"] == 1), result
    return ";".join(sorted(set(found)))


freeze = json.loads((ROOT / "freeze.json").read_text())
assert sha(CHECKER) == freeze["checker_sha256"]
assert sha(ROOT / "models.sha256") == freeze["manifest_sha256"]
assert sha(ROOT / "transcription.json") == freeze["transcription_sha256"]
assert sha(ROOT / "inventory.json") == freeze["inventory_sha256"]
models = json.loads((ROOT / "transcription.json").read_text())
inventory = json.loads((ROOT / "inventory.json").read_text())
for item in models + inventory:
    assert sha(ROOT / item["path"]) == item["sha256"], item["path"]
assert len(models) == freeze["model_count"] == 17
print("Frozen pilot models verified: 17/17 unchanged; checker and all 565 captured inputs unchanged.")

with (ROOT / "evidence/oracle2_summary.csv").open(newline="") as file:
    oracle2 = {(r["run"], r["task"]): r for r in csv.DictReader(file)}
with (ROOT / "evidence/summary.csv").open(newline="") as file:
    oracle = {(r["run"], r["task"]): r for r in csv.DictReader(file)}

rows, diagnostics = [], []


def record(item, cohort, result, verdict, baseline_equal=""):
    strict = cohort != "sanity"
    run, task = item.get("run", ""), item.get("task", "t1" if item["id"].startswith("t1-") else "")
    o1 = oracle.get((run, task), {}).get("oracle(pass/fail)", "not recorded")
    o2 = oracle2.get((run, task), {}).get("oracle2", "not applicable")
    rows.append(dict(cohort=cohort, id=item.get("id", item["path"]), path=item["path"],
                     run=run, task=task, primary=item.get("primary", False), exit=result["exit"],
                     flagged=bool(result["stdout"] or result["stderr"]), codes=codes(result, strict),
                     oracle=o1, oracle2=o2, verdict=verdict, baseline_equal=baseline_equal))
    diagnostics.append(dict(path=item["path"], cohort=cohort, **result))
    return rows[-1]


primary, final_copies = [], []
for item in inventory:
    if item["category"] != "unmodified":
        continue
    result = check(ROOT / item["path"])
    assert result["exit"] == 0 and not result["stdout"] and not result["stderr"], result
    bug = oracle2[(item["run"], item["task"])]["oracle2"] == "fail"
    row = record({"id": item["path"], **item}, "unmodified", result,
                 "known oracle2 bug not caught" if bug else "no diagnostic; oracle2 passes")
    final_copies.append(row)
    if item["primary"]:
        primary.append(row)
assert len(primary) == 20 and len(final_copies) == 32
assert len({(r["run"], r["task"]) for r in primary}) == 20
for row in primary:
    hashes = {i["sha256"] for i in inventory if i["category"] == "unmodified" and
              i["run"] == row["run"] and i["task"] == row["task"]}
    assert len(hashes) == 1, "Final submission differs from oracle2 copy"
print("Unmodified final programs: 20 logical / 32 physical copies; 0 diagnostics; 0 false positives.")

baseline_path = ROOT / "sanity-baseline.json"
baseline = json.loads(baseline_path.read_text()) if baseline_path.exists() else {}
baseline_sha = sha(BASELINE) if BASELINE.exists() else baseline.get("checker_sha256")
cached = baseline.get("by_input_sha256", {})
sanity = []
for item in inventory:
    if item["category"] != "sanity":
        continue
    result = check(ROOT / item["path"])
    if item["sha256"] not in cached:
        assert BASELINE.exists(), "Provide the original PR9 checker for first sanity evaluation"
        cached[item["sha256"]] = check(ROOT / item["path"], BASELINE)
    before = cached[item["sha256"]]
    assert result == before, (item["path"], before, result)
    sanity.append(record({"id": item["path"], **item}, "sanity", result,
                         "intentional vendored fixture; identical to original PR9 checker", True))
baseline_path.write_text(json.dumps(dict(checker_sha256=baseline_sha, by_input_sha256=cached), indent=2) + "\n")
assert len(sanity) == 296 and len(cached) == 36
sanity_flags = sum(r["flagged"] for r in sanity)
print(f"Vendored sanity: 296 physical / 36 unique contents; {sanity_flags} diagnostic-bearing fixture copies; 296/296 identical to PR9 baseline.")

scored = []
for item in models:
    result = check(ROOT / item["path"])
    code = codes(result)
    flagged = bool(code)
    assert flagged == item["expected_flag"], (item["id"], item["expected_flag"], result)
    verdict = "true bug caught" if flagged else "out of scope; not caught" if item["mechanism"] in {
        "empty-key", "codegen-unused-param"} else "safe classification; no diagnostic"
    scored.append(record(item, "transcription", result, verdict))
class_rows = [r for r in scored if r["run"] in {m["run"] for m in models if m["mechanism"] == "joined-unavailable-timeout"}]
assert len(class_rows) == 10
for r in class_rows:
    assert r["flagged"] == (oracle2[(r["run"], "t1")]["oracle2"] == "fail"), r
agents = [r for r in class_rows if not r["run"].startswith("ref")]
agent_flags = sorted(r["run"] for r in agents if r["flagged"])
assert agent_flags == ["go-1", "go-2", "go-3", "herta-2"]
print("Classification agent flags exactly oracle2: go-1, go-2, go-3, herta-2; herta-1 and herta-3 silent.")
print("References: ref-go/ref-herta flagged; ref2-go/ref2-herta silent (all match oracle2).")
with (ROOT / "results.csv").open("w", newline="") as file:
    writer = csv.DictWriter(file, fieldnames=list(rows[0]))
    writer.writeheader()
    writer.writerows(rows)
(ROOT / "diagnostics.json").write_text(json.dumps(diagnostics, indent=2) + "\n")

by_id = {r["id"]: r for r in scored}
nonbugs = sum(r["oracle2"] == "pass" for r in primary)
summary = f"""# Herta composition checker: agent-pilot evaluation

- Checker: `{freeze['checker_commit']}`; unchanged from the prototype. No checker fixes.
- Transcriptions frozen before checking: `{freeze['frozen_at']}`.
- Frozen manifest SHA-256: `{freeze['manifest_sha256']}`; all 17 models remain unchanged.
- Evidence inventory includes SHA-256 and original path for every captured input. `diagnostics.json` records every checker output, including all sanity-fixture diagnostics; `results.csv` includes all 345 checks.

## A. Unmodified real programs: false positives

Checked all t1–t4 final `.he` files from herta-1/2/3, ref-herta, and ref2-herta: **20 logical programs, 32 physical files**. The 12 oracle2-workspace mirrors of agent submissions are byte-identical and checked separately, not double-counted. Original agent submissions are the primary copies; the retained oracle2 workspace supplies ref-herta, and workspace supplies ref2-herta.

**Every unmodified check exited 0 with no diagnostics.** Consequently there are no diagnostics to mark true-bug versus false-positive. File flag rate: 0/20. False-positive rate among oracle2-passing program files: **0/{nonbugs} (0.0%)**. Agent-only: 0/11 oracle2-passing files (0/12 file flag rate).

All 12 agent task files pass the original oracle. Oracle2 fails herta-2/t1 and ref-herta/t1,t3,t4; these four known-bug programs are also accepted, not false positives. Original reference-oracle results were not retained in results/summary.csv, so are not invented here. Corrected ref2-herta passes all four oracle2 tasks. Oracle evidence was cross-checked from the captured summaries and per-task result/race logs.

These programs classify errors in raw Go helpers, not `classify` blocks, so classification diagnostics cannot fire unmodified. Most nested calls also go through opaque `servicePtr` methods; the checker does not perform whole-Go-program method/field dataflow. Zero false positives is **not** evidence of source-level bug discovery or proof that accepted programs are safe.

### Separate vendored sanity set

Excluded all vendored hertasdk files from the primary denominator. Checked **296 physical `.he` fixture/example copies, 36 distinct contents**. {sanity_flags} copies emit diagnostics; **296/296 outputs and exit codes match the original PR9 checker byte-for-byte**. These include intentionally invalid parser/checker fixtures, not agent bugs or false positives. Full individual outputs are in diagnostics.json, with original paths/hashes in inventory.json and baseline outputs/hash in sanity-baseline.json. No new sanity regression or crash was observed.

## B. Frozen escaped-bug transcriptions

Contracts expose logic that lives in Go. They are projections, not unchanged agent programs. Every t1 model includes the task author's supplied `error ErrTimeout ambiguous`, nonidempotent charging, three transient-only attempts, and the actual ordered classifier. Nil-success handling is omitted; context names are normalized. Untagged context errors have nominal permanent classification; SDK deadline promotion is dynamic and is not modeled. This does not affect the ErrUnavailable+ErrTimeout comparison.

### Actual t1 classifier order and oracle2 comparison

| Run | Checker diagnostics | Oracle2 t1 | Agreement |
| --- | --- | --- | --- |
"""
for r in class_rows:
    summary += f"| {r['run']} | {r['codes'] or 'none'} | {r['oracle2']} | yes |\n"
summary += """
**Agent flags are exactly go-1, go-2, go-3, herta-2; herta-1 and herta-3 remain silent.** All 6/6 agent predictions match oracle2. Original references (both Go and Herta) are flagged; corrected references are silent, giving 10/10 comparisons across all source variants. The reference pairs are controls, not additional agent runs.

Unavailable-first classifiers produce HC010 because joined errors can match both arms. In the actual oracle2 scenario, t1 Service.Charge dispatches the governed charge handler, the backend applies whenever errors.Is(err, ErrTimeout), the handler checks ErrUnavailable first, and the transient-only retry policy replays it. The retained race logs report `uncertain charge repeated/misreported: err=<nil> outcome=Success charges=2 calls=2`. As a disconfirming control, timeout-first herta-1/herta-3 and both ref2 variants pass the same joined-error oracle. Ordered transcription and positive/negative controls match this causal chain; no hypothetical source reachability is substituted for oracle evidence.

### Per §3 bug class

| Class | Caught? | Diagnostic / evidence |
| --- | --- | --- |
"""
for title, ident, explanation in [
    ("Declared nonidempotent + retry uncertain (already caught baseline)", "declared-nonidempotent-uncertain", "local policy baseline"),
    ("Timeout classified transient", "t1-timeout-transient-trap", "task's t1 trap, not observed in final six submissions"),
    ("Joined ErrUnavailable+ErrTimeout retried", "t1-go-1", "4/6 agents; original references also flagged"),
    ("Nested op reacquires same key", "t4-herta-2-intermediate-fallback", "explicitly labeled fallback trap; not recovered agent source"),
    ("Idempotent wrapper + nonidempotent call + retry", "idempotent-wrapper-trap", "declared edge exposes the probe mechanism"),
    ("Empty serialize key disables serialization", "empty-serialize-key", "out of scope: no nonempty-key/value analysis"),
    ("Codegen unused-parameter defect", "codegen-unused-param", "out of scope: static check does not compile generated Go"),
]:
    r = by_id[ident]
    summary += f"| {title} | {'yes' if r['flagged'] else '**not caught**'} | {r['codes'] or 'none'}; {explanation} |\n"
summary += f"""
Of the five originally uncaught semantic classes in §3, **4/5 are caught in these contract/trap projections**; the empty-key class is not. The already-caught local unsafe retry remains caught. The separately listed codegen defect is also not caught. These are class-level demonstrations, not independent real-source detections.

The exact retained `probes/pr9/composition/traps.he` is additionally frozen and checked unchanged: **{by_id['probe-traps']['codes']}**. This exercises whitespace-tolerant typed `.Do` extraction as well as the explicit-clause models.

### herta-2 intermediate recovery limitation

Inspected both retained final t4 workspaces, score.log, supplied-policy.diff, and the supplied pilot artifact/log inventory. No intermediate revision, git history, or agent editing transcript is retained in these snapshots. Used the user-authorized same-key nested-call trap fallback, clearly labeled in the model and transcription.json. This is **not** a claim that the intermediate implementation was recovered or that the final herta-2 transfer deadlocks. The final implementation uses a NUL-prefixed transfer key around debit and credits outside that lock; it passes oracle2. Oracle completion does not establish full two-step Transfer serialization.

## Reproduction and limits

Run `python3 pilot/score.py` from the evaluation directory; it verifies checker/model/evidence hashes before checking. The original PR9 baseline is needed only to generate missing sanity-baseline entries; the frozen captured baseline allows replay without the pilot checkout. `build_pilot.py` is the evidence/transcription builder and deliberately refuses to overwrite an existing freeze. It requires the original pilot paths and is not needed to score captured inputs.

No transcriptions were changed after checker scoring. Empty-key and codegen defects are explicitly not caught. No runtime compilation, new oracle runs, or arbitrary Go alias/value inference was added. The oracle outcomes quoted here are the retained pilot evidence, not fresh executions. No UI or visual verification is applicable.
"""
(ROOT / "summary.md").write_text(summary)
print("Escaped semantic classes caught: 4/5; local unsafe retry remains caught; empty key and codegen unused-param not caught.")
print("Wrote pilot/results.csv (345 rows), diagnostics.json, summary.md; no checker changes.")
