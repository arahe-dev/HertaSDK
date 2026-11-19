#!/usr/bin/env python3
"""Capture pilot evidence and freeze contract projections before checker access."""
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import shutil

ROOT = Path(__file__).resolve().parent
OUT = ROOT / "pilot"
PILOT = Path.home() / "work/pilot"
RESULTS = PILOT / "results"
RUNS = ["go-1", "go-2", "go-3", "herta-1", "herta-2", "herta-3",
        "ref-go", "ref-herta", "ref2-go", "ref2-herta"]
assert not (OUT / "freeze.json").exists(), "Never overwrite frozen transcriptions"
for folder in ["models", "evidence", "unmodified", "sanity"]:
    (OUT / folder).mkdir(parents=True, exist_ok=True)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


inventory = []


def capture(source, relative, category, **metadata):
    target = OUT / relative
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, target)
    item = dict(path=relative, source=str(source), sha256=sha(target), category=category, **metadata)
    inventory.append(item)
    return item


for run in RUNS:
    folder = "workspace" if run.startswith("ref2-") else "oracle2-workspace"
    base = RESULTS / run / folder
    for task in ["t1", "t2", "t3", "t4"]:
        for name in ["service.go", "glue.go", "backend.go", "shop.he"]:
            source = base / task / name
            if source.exists():
                capture(source, f"evidence/{run}/{task}/{name}.txt", "source", run=run, task=task)
        for oracle in ["oracle", "oracle2"]:
            for suffix in ["result.json", "race.log"]:
                source = RESULTS / run / f"{oracle}-{task}-{suffix}"
                if source.exists():
                    capture(source, f"evidence/{run}/{source.name}", "oracle", run=run, task=task)
    if "herta" not in run:
        continue
    for source in sorted((RESULTS / run).rglob("*.he")):
        relative = source.relative_to(RESULTS)
        vendor = "hertasdk" in source.parts
        if not vendor and source.parent.name not in {"t1", "t2", "t3", "t4"}:
            raise ValueError(source)
        category = "sanity" if vendor else "unmodified"
        task = source.parent.name if not vendor else ""
        primary = ("bundle-herta" in source.parts if not run.startswith("ref") else folder in source.parts)
        capture(source, str(Path(category) / relative), category, run=run, task=task,
                primary=primary and not vendor)

for name in ["summary.csv", "oracle2_summary.csv", "README.md", "oracle2-controls.json"]:
    capture(RESULTS / name, "evidence/" + name, "oracle-summary")
for task in ["t1", "t4"]:
    capture(PILOT / "oracle2" / task / "harder_test.go", f"evidence/oracle2/{task}/harder_test.go.txt", "oracle-source")
capture(Path.home() / "attachments/4c0cf229-ca4a-487b-9163-9a8bac35690c/herta-agent-pilot.md",
        "evidence/herta-agent-pilot.md", "write-up")
for run in ["herta-2"]:
    for name in ["score.log", "supplied-policy.diff"]:
        capture(RESULTS / run / name, f"evidence/{run}/{name}", "intermediate-search")

models = []


def model(ident, body, mechanism, note, run="", expected=None, source=""):
    relative = "models/" + ident + ".he"
    path = OUT / relative
    path.write_text(f"// {note}\n" + body)
    models.append(dict(id=ident, path=relative, mechanism=mechanism, note=note, run=run,
                       expected_flag=expected, source=source, sha256=sha(path)))


def op(name, effect="nonidempotent", clauses="", params=""):
    return (f"op main::{name}({params}) -> u64 {{ effect {effect} {clauses} }}\n"
            f"handler main::{name}({params}) -> (u64, error) {{ return 0, nil }}\n")


runtime = "runtime main { resource db capacity 8 }\n"
bad_order = {"go-1", "go-2", "go-3", "herta-2", "ref-go", "ref-herta"}
for run in RUNS:
    source = f"evidence/{run}/t1/" + ("glue.go.txt" if "herta" in run else "service.go.txt")
    text = (OUT / source).read_text()
    first_timeout = text.index("case errors.Is(err, ErrTimeout)")
    first_unavailable = text.index("case errors.Is(err, ErrUnavailable)")
    assert (first_unavailable < first_timeout) == (run in bad_order)
    if run == "herta-1":
        arms = [("ErrTimeout", "uncertain"), ("ContextDeadlineExceeded", "uncertain"),
                ("ErrDeclined", "permanent"), ("ContextCanceled", "permanent"), ("ErrUnavailable", "transient")]
    elif run == "herta-3":
        arms = [("ErrTimeout", "uncertain"), ("ContextCanceled", "permanent"),
                ("ContextDeadlineExceeded", "permanent"), ("ErrUnavailable", "transient")]
    else:
        arms = [("ErrUnavailable", "transient"), ("ErrTimeout", "uncertain")]
        if run not in bad_order:
            arms.reverse()
        if run == "go-2":
            arms += [("ErrDeclined", "permanent")]
        if run == "go-3":
            arms += [("ContextDeadlineExceeded", "uncertain"), ("ContextCanceled", "uncertain")]
        if run == "herta-2":
            arms += [("ContextDeadlineExceeded", "permanent"), ("ContextCanceled", "permanent")]
    arms.append(("_", "permanent"))
    clause = "classify { " + ", ".join(f"{error} => {outcome}" for error, outcome in arms) + " }"
    note = (f"Actual {run} t1 first-match classification order; see {source}. "
            "Nil-success branch omitted; context symbols normalized. Untagged context returns use nominal permanent; "
            "SDK deadline promotion is dynamic and outside this classification projection.")
    model("t1-" + run, runtime + "error ErrTimeout ambiguous\n" +
          op("charge", clauses="timeout 500ms retry max 3 on { transient } " + clause),
          "joined-unavailable-timeout", note, run=run, expected=run in bad_order, source=source)

model("t1-timeout-transient-trap", runtime + "error ErrTimeout ambiguous\n" +
      op("charge", clauses="retry max 3 on { transient } classify { ErrTimeout => transient, _ => permanent }"),
      "timeout-transient", "Task t1 trap: timeout may have committed, but is marked transient.", expected=True)
model("declared-nonidempotent-uncertain", runtime + op("charge", clauses="retry max 3 on { uncertain }"),
      "declared-unsafe-retry", "Section 3 already-caught baseline: declared nonidempotent retry on uncertain.", expected=True)
model("t4-herta-2-intermediate-fallback", runtime +
      op("transfer", clauses="serialize key user calls debit", params="user: string") +
      op("debit", clauses="serialize key user", params="user: string"),
      "nested-key", "Trap fallback, NOT recovered intermediate source. Supplied herta-2 final snapshots and retained score logs "
      "contain only the final NUL-prefixed solution; no intermediate revision or agent transcript is retained there.",
      run="herta-2-intermediate", expected=True, source="evidence/herta-2/t4/glue.go.txt")
model("idempotent-wrapper-trap", runtime +
      op("checkout", effect="idempotent", clauses="retry max 3 on { uncertain } calls charge") + op("charge"),
      "idempotent-wrapper", "Probe's opaque wrapper effect and retry, exposed through calls charge.", expected=True)
model("empty-serialize-key", runtime +
      "op main::entry() -> u64 { effect nonidempotent }\n"
      "handler main::entry() -> (u64, error) { call refund(\"\") return 0, nil }\n" +
      op("refund", clauses="serialize key order_key", params="order_key: string"),
      "empty-key", "Section 3 empty-key trap: concrete empty argument disables runtime serialization; no nonempty key type/check.", expected=False)
for name, source, mechanism, expected in [
    ("probe-traps", PILOT / "probes/pr9/composition/traps.he", "combined-probe", True),
    ("codegen-unused-param", PILOT / "probes/pr9/unused/unused.he", "codegen-unused-param", False),
]:
    target = OUT / "models" / (name + ".he")
    shutil.copy2(source, target)
    models.append(dict(id=name, path=str(target.relative_to(OUT)), mechanism=mechanism,
                       note="Exact unmodified retained probe.", run="probe", expected_flag=expected,
                       source=str(source), sha256=sha(target)))

(OUT / "inventory.json").write_text(json.dumps(inventory, indent=2) + "\n")
(OUT / "transcription.json").write_text(json.dumps(models, indent=2) + "\n")
manifest = "".join(f"{m['sha256']}  {m['path']}\n" for m in models)
(OUT / "models.sha256").write_text(manifest)
(OUT / "freeze.json").write_text(json.dumps(dict(
    frozen_at=datetime.now(timezone.utc).isoformat(), checker_sha256=sha(ROOT / "herta"),
    checker_commit="e9741d23d663ea1ad8e708a5f0e88393424b4416",
    checker_run_on_transcriptions=False, model_count=len(models),
    manifest_sha256=sha(OUT / "models.sha256"), transcription_sha256=sha(OUT / "transcription.json"),
    inventory_sha256=sha(OUT / "inventory.json")), indent=2) + "\n")
print(f"Frozen {len(models)} pilot models before checking; captured {len(inventory)} evidence/input files.")
print(f"Primary final programs: {sum(i.get('primary', False) for i in inventory)}; "
      f"physical final copies: {sum(i['category'] == 'unmodified' for i in inventory)}; "
      f"separate sanity inputs: {sum(i['category'] == 'sanity' for i in inventory)}.")
