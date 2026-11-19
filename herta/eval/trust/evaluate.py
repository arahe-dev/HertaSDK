#!/usr/bin/env python3
"""Rebuild CLI and inventory frozen inputs without changing frozen evidence."""
import collections
import json
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[3]
OUT = Path(__file__).resolve().parent
KINDS = ["opaque-handler", "transient-honest", "legacy-tag-path",
         "nested-retry-amplification", "same-serialize-key", "unresolved-callee"]


def execute(args):
    return subprocess.run(args, cwd=ROOT, text=True, capture_output=True, timeout=30)


def evaluate(binary, directory, closed):
    rows = []
    for path in sorted(directory.rglob("*.he")):
        cmd = [str(binary), "trust", "-json"]
        if closed:
            cmd.append("-closed")
        result = execute(cmd + [str(path)])
        if result.returncode:
            raise RuntimeError(f"trust failed: {path}: {result.stdout}{result.stderr}")
        report = json.loads(result.stdout)
        direct = {(a["origin"], a["kind"], a["detail"]): a
                  for op in report for a in op["assumptions"]}
        check = execute([str(binary), "check", "-closed", str(path)])
        if check.returncode not in (0, 1):
            raise RuntimeError(f"check failed: {path}: {check.stderr}")
        hc016 = sorted(set(re.findall(r"HC016 path ([^:]+::[^:]+):", check.stdout)))
        rows.append({"file": str(path.relative_to(directory)), "ops": len(report),
                     "occurrences": collections.Counter(a["kind"] for op in report for a in op["assumptions"]),
                     "direct": list(direct.values()), "hc016": hc016})
    return rows


def local_nontrivial(row, op):
    return sorted({a["kind"] for a in row["direct"]
                   if a["origin"] == op and a["kind"] != "opaque-handler"})


def main():
    with tempfile.TemporaryDirectory(prefix="herta-trust-") as temp:
        binary = Path(temp) / "herta"
        build = execute(["go", "build", "-o", str(binary), "./herta/cmd/herta"])
        if build.returncode:
            raise RuntimeError(build.stdout + build.stderr)
        groups = {
            "inputs (legacy)": evaluate(binary, ROOT / "herta/eval/closed/inputs", False),
            "migrated (closed)": evaluate(binary, ROOT / "herta/eval/closed/models", True),
            "corpus (legacy)": evaluate(binary, ROOT / "herta/eval/composition/corpus", False),
        }
    assert [len(rows) for rows in groups.values()] == [20, 20, 121], "frozen inventory changed"
    lines = ["# Trust ledger evaluation", "",
             "Reproduce from repo root: `python3 herta/eval/trust/evaluate.py`.",
             "Builds the current CLI once; invokes `trust -json` per file and `check -closed` for HC016 metadata.",
             "Frozen inputs/evidence are read only. No oracle reruns; labels/mechanisms below are from",
             "`herta/eval/closed/summary.md` (sections A/B and verified-path discussion).", "",
             "Inputs use legacy `build` semantics; migrated models use `build -closed`; corpus uses legacy mode.",
             "A classify clause alone does not install Policy.Classify in a legacy build.",
             "Both raw input and faithful migration versions of all 20 programs are covered;",
             "121 corpus `.he` models are covered (not nested pilot workspaces or mutation variants).", "",
             "## Counts", "",
             "Each cell is **unique origin obligations / propagated per-op occurrences / programs flagged**.",
             "Unique keys are (file, origin, kind, detail). Required/informational is retained in JSON output;",
             "transient-honest is required only for nonidempotent origins. Counts are warnings, not bugs.", "",
             "| Kind | inputs (legacy) | migrated (closed) | corpus (legacy) |",
             "|---|---:|---:|---:|"]
    for kind in KINDS:
        cells = []
        for rows in groups.values():
            direct = sum(sum(a["kind"] == kind for a in r["direct"]) for r in rows)
            occurrences = sum(r["occurrences"][kind] for r in rows)
            programs = sum(any(a["kind"] == kind for a in r["direct"]) for r in rows)
            cells.append(f"{direct} / {occurrences} / {programs}")
        lines.append(f"| {kind} | " + " | ".join(cells) + " |")
    lines += ["", "| Group | Programs | Ops | Required origin obligations | Informational origin obligations |",
              "|---|---:|---:|---:|---:|"]
    for group, rows in groups.items():
        required = sum(sum(a["required"] for a in r["direct"]) for r in rows)
        info = sum(sum(not a["required"] for a in r["direct"]) for r in rows)
        lines.append(f"| {group} | {len(rows)} | {sum(r['ops'] for r in rows)} | {required} | {info} |")
    lines += ["", "## All closed-evaluation programs", "",
              "Non-trivial kinds below exclude opaque-handler, and list **local origins**, not merely",
              "an unrelated warning propagated onto the op. `none` is a miss for that location.", "",
              "| Program | Input local non-trivial assumptions | Migrated local non-trivial assumptions | Input HC016 origins |",
              "|---|---|---|---|"]
    inputs = {r["file"].removesuffix("/shop.he"): r for r in groups["inputs (legacy)"]}
    models = {r["file"].removesuffix("/shop.he"): r for r in groups["migrated (closed)"]}

    def locations(row):
        return "; ".join(f"{origin}: {', '.join(local_nontrivial(row, origin))}"
                         for origin in sorted({a["origin"] for a in row["direct"]})
                         if local_nontrivial(row, origin)) or "none"

    for name, row in inputs.items():
        lines.append(f"| {name} | {locations(row)} | {locations(models[name])} | {', '.join(row['hc016']) or 'none'} |")
    lines += ["", "## Known failures: location is not diagnosis", "",
              "| Baseline oracle2 failure | Relevant origin | Input kinds at origin | Migrated kinds at origin | Causal assessment |",
              "|---|---|---|---|---|"]
    failing = [("H2/t1", "main::charge"), ("R/t1", "main::charge"),
               ("R/t3", "main::reserve"), ("R/t4", "main::transfer")]
    for name, op in failing:
        before = ", ".join(local_nontrivial(inputs[name], op)) or "none"
        after = ", ".join(local_nontrivial(models[name], op)) or "none"
        if name.endswith("t1"):
            assessment = "Broad trust obligation at charge; NOT a diagnosis of joined-error arm ordering (HC010 does that)."
        elif name.endswith("t3"):
            assessment = "MISS: reserve is flagged informationally, but missing retry-on-Uncertain is not detected."
        else:
            assessment = "MISS: dynamic transfer/account key collision is hidden in Go key values; no HC001/same-serialize-key here."
        lines.append(f"| {name} | {op} | {before} | {after} | {assessment} |")
    hcrows = [(name, op, local_nontrivial(row, op)) for name, row in inputs.items() for op in row["hc016"]]
    covered = sum(bool(kinds) for _, _, kinds in hcrows)
    lines += ["", "## HC016 coverage", "",
              f"{len(hcrows)} input op-level HC016 sites in {sum(bool(r['hc016']) for r in inputs.values())} programs; "
              f"{covered}/{len(hcrows)} have a non-trivial assumption at the exact origin.",
              "The per-program table lists every HC016 origin. All have legacy-tag-path; retry-on-Transient",
              "ones also have transient-honest (required for charge, informational for reserve). This exposes",
              "missing classification authority, not proof that the original behavior is buggy.",
              f"Migrated HC016 sites: {sum(len(r['hc016']) for r in models.values())}.", "",
              "## Limitations / misses", "",
              "- All handlers remain opaque, so strict mode rejects every evaluated program; this is a trust",
              "  boundary, not a useful binary bug classifier. Every legacy op is also conservatively tagged.",
              "- Non-trivial coverage is deliberately not called recall: generic obligations can mark both",
              "  correct and buggy classifiers. No new causal diagnosis of the four failures is claimed.",
              "- R/t3's transient-honest warning does not point to its missing uncertain retry; R2/t3 has",
              "  similar obligations but passes. The ledger cannot infer required application behavior.",
              "- R/t4 retains a generic legacy-tag-path flag, but no matching key hazard flag: distinct symbolic",
              "  key names hide the concrete `transfer:A` collision. Closed runtime ErrKeyReacquired is the",
              "  matching guard, not this ledger.",
              "- Raw service/helper calls in input t3/t4 are opaque; migration's declared calls expose their",
              "  dependencies. No inferred edge through arbitrary Go method dispatch is claimed.",
              "- Corpus counts are an inventory of frozen models, not oracle verification or classifier honesty proofs.",
              "- JSON is available by rerunning CLI per file; summary intentionally stores no copies of frozen models.", ""]
    summary = "\n".join(lines)
    (OUT / "summary.md").write_text(summary)
    print(summary)


if __name__ == "__main__":
    main()
