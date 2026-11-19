#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"
log=$(mktemp)
trap 'rm -f "$log"' EXIT
start=$SECONDS
lake build 2>&1 | tee "$log"
# Recheck source so cached lake output cannot hide an axiom dependency.
lake env lean Herta.lean 2>&1 | tee -a "$log"
python3 - "$log" <<'PY'
import re
import sys
text = open(sys.argv[1]).read()
if 'sorryAx' in text:
    sys.exit('FAIL: sorryAx dependency')
audits = re.findall(r"'([^']+)' depends on axioms: \[([^]]*)\]", text, re.S)
clean = re.findall(r"'([^']+)' does not depend on any axioms", text)
allowed = {'propext', 'Classical.choice', 'Quot.sound'}
names = set(clean)
for name, axioms in audits:
    names.add(name)
    bad = set(axioms.replace(',', ' ').split()) - allowed
    if bad:
        sys.exit(f'FAIL: {name}: unexpected axioms {sorted(bad)}')
expected = {'T1', 'T1_assumption_necessary', 'T2', 'T3_per_attempt',
            'T3_default', 'T4_key', 'T4_composition', 'T5'}
if not {'Herta.' + n for n in expected} <= names:
    sys.exit('FAIL: missing theorem axiom audits')
print('PASS: all theorem audits present; only standard axioms')
PY
printf 'check.sh elapsed: %ss\n' "$((SECONDS - start))"
