# Frozen composition evaluations

- `summary.md`: 121-site, label-blind contract-transcription evaluation.
- `pilot/summary.md`: unmodified agent programs, frozen escapes, oracle cross-checks.
- `corpus/`, `pilot/models/`: frozen models; do not edit after scoring.
- `freeze.json`, `models.sha256`, and their pilot equivalents: provenance and integrity.
- `herta`: frozen Linux amd64 checker executable; both scorers verify its hash.
- `inputs/`: original corpus facts, transcription rules, and hand labels, copied
  only after the label-blind evaluation was completed.

From this directory, run `python3 score.py` and `python3 pilot/score.py` to
reproduce scores against captured inputs. These scripts never write models.
The corpus scorer's only packaging change is preferring bundled `inputs/`
over the original external paths. The builders are historical provenance,
not commands to run over frozen models; they need the original workspaces.

The nested `go.mod` keeps evidence Go source out of the parent module's
`go test ./...`. The capped runner discovers only `herta/testdata/cases`,
not this directory. Vendored pilot `.he` fixtures are data, not added tests.

The full external evaluation exceeds 2 MB, so the repository copy omits
the optional top-level `sources/` pinned-source copies. They remain in
`~/work/eval` and `~/work/herta-composition-eval.tar.gz`. All models,
transcription rationales, results, captured pilot source/oracle evidence,
and replay inputs are retained in the repository copy. Source omission
does not affect either scorer. No checker implementation was changed.

Local `.gitattributes` disables newline normalization for this frozen data:
captured CSV evidence uses CRLF and must keep its recorded SHA-256 bytes.
