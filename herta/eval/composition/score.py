#!/usr/bin/env python3
"""Score the frozen, label-blind Herta corpus; never writes model files."""
import collections
import csv
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parent
CHECKER = ROOT / "herta"
KNOWN = re.compile(r'^\d+:\d+: op "[^"]+": retrying uncertain outcomes is unsafe for effect nonidempotent$')
HC = re.compile(r'^\d+:\d+: (HC\d{3}) path ')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rate(n, d):
    return f"{n}/{d} ({100*n/d:.1f}%)" if d else "0/0 (n/a)"


def main():
    start = time.monotonic()
    freeze = json.loads((ROOT / "freeze.json").read_text())
    assert freeze["labels_opened"] is False
    assert sha(CHECKER) == freeze["checker_sha256"], "Checker changed since corpus freeze"
    assert sha(ROOT / "models.sha256") == freeze["model_manifest_sha256"], "Manifest changed"
    facts_path = ROOT / "inputs/corpus-facts.json"
    if not facts_path.exists():
        facts_path = ROOT.parent / "corpus-facts.json"
    facts = json.loads(facts_path.read_text())
    expected_ids = {r["id"] for r in facts}
    assert len(facts) == len(expected_ids) == 121
    assert {p.stem for p in (ROOT / "corpus").glob("*.he")} == expected_ids
    manifest = {}
    for line in (ROOT / "models.sha256").read_text().splitlines():
        digest, relative = line.split("  ", 1)
        assert sha(ROOT / relative) == digest, f"Frozen model changed: {relative}"
        manifest[Path(relative).stem] = digest
    assert set(manifest) == expected_ids
    print("Frozen model hashes verified: 121/121 unchanged.")

    # Labels are read only after verifying that all frozen models exist.
    labels_path = ROOT / "inputs/labels.csv"
    if not labels_path.exists():
        labels_path = ROOT.parent / "analyzer" / "labels.csv"
    with labels_path.open(newline="") as file:
        label_rows = list(csv.DictReader(file))
    labels = {r["id"]: r for r in label_rows}
    assert len(labels) == len(label_rows) and set(labels) == expected_ids
    assert all(r["label"] in {"TP-bug", "TP-benign", "FP", "Unclear"} for r in labels.values())
    assert sum(r["label"] == "TP-bug" for r in labels.values()) == 19

    results, runs = [], []
    instances = collections.Counter()
    for fact in facts:
        if time.monotonic() - start > 60:
            raise RuntimeError("Evaluation exceeded 60s total cap")
        ident = fact["id"]
        model = ROOT / "corpus" / (ident + ".he")
        run = subprocess.run([str(CHECKER), "check", str(model)], capture_output=True, text=True, timeout=5)
        assert run.returncode in (0, 1) and not run.stderr, (ident, run.returncode, run.stderr)
        codes = []
        for line in run.stdout.splitlines():
            match = HC.match(line)
            if match:
                codes.append(match.group(1))
            elif KNOWN.fullmatch(line):
                codes.append("LEGACY_UNSAFE_RETRY")
            else:
                raise RuntimeError(f"Parse error or unknown diagnostic for {ident}: {line}")
        assert bool(codes) == (run.returncode == 1), (ident, run.stdout)
        instances.update(codes)
        reason = next((line.removeprefix("// inexpressible: ") for line in model.read_text().splitlines()
                       if line.startswith("// inexpressible: ")), "")
        label = labels[ident]
        assert label["check"] == fact["check"]
        results.append(dict(id=ident, check=fact["check"], label=label["label"],
                            flagged=bool(codes), codes=";".join(sorted(set(codes))), inexpressible=reason))
        runs.append(dict(id=ident, exit=run.returncode, stdout=run.stdout, stderr=run.stderr))
    with (ROOT / "results.csv").open("w", newline="") as file:
        writer = csv.DictWriter(file, fieldnames=["id", "check", "label", "flagged", "codes", "inexpressible"])
        writer.writeheader()
        writer.writerows(results)
    (ROOT / "diagnostics.json").write_text(json.dumps(runs, indent=2) + "\n")

    report = ["# Herta composition checker: frozen real-repo evaluation", "",
              f"- Checker: `{freeze['checker_commit']}` (no checker fixes during evaluation).",
              "- 121/121 models validated: exit 0 or 1, only HC/known diagnostics, no parse errors or crashes.",
              f"- Models frozen before labels were opened: `{freeze['frozen_at_utc']}`.",
              f"- Corpus manifest SHA-256: `{freeze['model_manifest_sha256']}`; verified unchanged when scoring.",
              f"- Label file SHA-256: `{sha(labels_path)}`.",
              "- No model was edited after label access or scoring. review-decisions.tsv was never opened.",
              "- Flagged means any diagnostic from the checker, not just the analyzer site's matching diagnostic family.",
              "- False-positive rate = flagged (TP-benign + FP) / all (TP-benign + FP); Unclear is excluded.",
              "", "## Results", "",
              "| Analyzer check | Sites | Bugs caught | Non-bug flags / non-bugs (FPR) | Unclear flagged / total | Inexpressible |",
              "| --- | ---: | ---: | ---: | ---: | ---: |"]
    for check in ["C1", "C2", "C3", "C4", "Overall"]:
        group = [r for r in results if check == "Overall" or r["check"] == check]
        bugs = [r for r in group if r["label"] == "TP-bug"]
        nonbugs = [r for r in group if r["label"] in {"TP-benign", "FP"}]
        unclear = [r for r in group if r["label"] == "Unclear"]
        report.append(f"| {check} | {len(group)} | {rate(sum(r['flagged'] for r in bugs), len(bugs))} | "
                      f"{rate(sum(r['flagged'] for r in nonbugs), len(nonbugs))} | "
                      f"{sum(r['flagged'] for r in unclear)}/{len(unclear)} | {sum(bool(r['inexpressible']) for r in group)} |")

    bug_groups = collections.defaultdict(list)
    for row in results:
        if row["label"] == "TP-bug":
            bug_groups[labels[row["id"]]["subtype"]].append(row)
    report += ["", "## Distinct mechanisms, not independent bug counts", "",
               "Grouped by the hand labels' subtype (a mechanism taxonomy, not a claim of independent root causes):", "",
               "| Mechanism | Bug sites caught / total |", "| --- | ---: |"]
    for subtype, group in sorted(bug_groups.items()):
        report.append(f"| {subtype} | {sum(r['flagged'] for r in group)}/{len(group)} |")
    mechanisms_caught = sum(any(r["flagged"] for r in group) for group in bug_groups.values())
    report += ["", f"Mechanism coverage: **{mechanisms_caught}/{len(bug_groups)}** (a mechanism is caught if any of its sites is flagged).",
               "The 11 C2 bug rows all represent the same b2 exhausted-auth-budget flaw, not 11 independent flaws. "
               "Collapsing only those 11 rows gives 9 site/flaw groups; subtype grouping additionally combines the two task-replay "
               "sites and the two recursive-RLock sites.", "", "## Diagnostic counts", "",
               "Site counts are unique models; instances include repeated diagnostics on different call paths.", "",
               "| Code | Sites | Instances | Bug sites | Non-bug sites | Unclear sites |",
               "| --- | ---: | ---: | ---: | ---: | ---: |"]
    for code in sorted(instances):
        group = [r for r in results if code in r["codes"].split(";")]
        report.append(f"| {code} | {len(group)} | {instances[code]} | "
                      f"{sum(r['label']=='TP-bug' for r in group)} | "
                      f"{sum(r['label'] in {'TP-benign','FP'} for r in group)} | "
                      f"{sum(r['label']=='Unclear' for r in group)} |")
    report += ["", "LEGACY_UNSAFE_RETRY is a scoring name for the existing uncoded local nonidempotent/uncertain diagnostic; "
               "it is not a new HC code. Codes absent from this table did not fire.", "", "## Missed bugs", ""]
    missed = [r for r in results if r["label"] == "TP-bug" and not r["flagged"]]
    if not missed:
        report.append("None: all 19 labeled bug sites were flagged.")
    for row in missed:
        report.append(f"- `{row['id']}`: {row['inexpressible'] or 'Expressible model accepted; inspect diagnostics.json and the frozen model for the missing rule.'}")

    report += ["", "## Flags on non-bug rows", ""]
    for row in results:
        if row["flagged"] and row["label"] in {"TP-benign", "FP"}:
            report.append(f"- `{row['id']}` ({row['label']}): `{row['codes']}`. {labels[row['id']]['one-line reason']}")
    report += ["", "## Unclear rows (not treated as bugs or false positives)", ""]
    for row in results:
        if row["label"] == "Unclear":
            report.append(f"- `{row['id']}`: {'flagged: ' + row['codes'] if row['flagged'] else 'not flagged'}"
                          + ("; inexpressible: " + row["inexpressible"] if row["inexpressible"] else "") + ".")

    report += ["", "## Scope and limitations", "",
               "This measures checking of human-transcribed contracts, not source-level bug discovery. Facts already supply "
               "branch reachability, retry-stop behavior, ownership, and lock identity that the checker itself does not infer from Go.",
               "No budget overrides, invented dedupe keys, or intentionally understated effect declarations were used to fit labels. "
               "Effects and edges were chosen before label access; false positives are retained unchanged.",
               "Pinned sources were consulted for actual attempt bounds (rclone default 10, Kubernetes backoff 4/5, Loki reads 5), "
               "transport retry behavior, and multipart allocation. Source copies and per-model rationale are in sources/ and transcription.json.",
               "The certificate model covers the actual selected five-attempt backoff; the subsequent separate infinite poll is not "
               "represented with an invented finite bound. C2 auth models project the credential-recovery path established by facts, "
               "not unspecified surrounding user mutations. Abort-on-inner-failure recovery is kept separate per transcription rule 6.",
               "Time-driven/unbounded retry policies without a fixed attempt cap, unknown effect contracts, and unverified incoming "
               "context paths use pure placeholders with an inexpressible header. Pure placeholders remain in all denominators; "
               "acceptance is not evidence that the original behavior was proven safe.", "", "### Inexpressible models", ""]
    for row in results:
        if row["inexpressible"]:
            report.append(f"- `{row['id']}`: {row['inexpressible']}")
    report += ["", "## Checker changes during evaluation", "", "None. No crashes or parse failures required a checker fix.", ""]
    (ROOT / "summary.md").write_text("\n".join(report))
    caught = sum(r["label"] == "TP-bug" and r["flagged"] for r in results)
    nonbugs = [r for r in results if r["label"] in {"TP-benign", "FP"}]
    unclear = [r for r in results if r["label"] == "Unclear"]
    print("Models checked: 121/121; parse errors: 0; crashes: 0.")
    print(f"Bugs caught: {caught}/19; distinct subtype mechanisms: {mechanisms_caught}/{len(bug_groups)}.")
    print(f"Non-bug flags (TP-benign+FP): {rate(sum(r['flagged'] for r in nonbugs),len(nonbugs))}.")
    print(f"Unclear flags: {sum(r['flagged'] for r in unclear)}/{len(unclear)}.")
    print(f"Inexpressible models: {sum(bool(r['inexpressible']) for r in results)}/121.")
    print(f"Missed bugs: {len(missed)}.")
    print("Wrote results.csv, diagnostics.json, summary.md.")


if __name__ == "__main__":
    main()
