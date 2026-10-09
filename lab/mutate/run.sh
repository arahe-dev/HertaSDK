#!/usr/bin/env bash
# Mutation-tests v0.2.0 (orig, the root package) and v0.2-recon with the
# same four-stage kill cascade, then prints the side-by-side score table.
#
#   own  the package's own tests; -short skips wall-clock latency budgets,
#        which are meaningless while other mutants build in parallel
#   lab  the implementation-agnostic lab suites restricted to that
#        implementation (LAB_IMPLS), plus lab/mutate's gap tests
#   race the own tests again under -race: catches mutants that only
#        drop a lock or reorder a publication (needs cgo)
#   race-lab  the lab's cancel storm under -race, for the lock-dropping
#        mutants the own tests never put on two goroutines at once
#
# A test failure kills a mutant only if the stage fails again when re-run
# as is; summary.md lists the failures that did not (flaky tests).
#
# Usage: bash lab/mutate/run.sh [orig|recon|both] [extra "mutate run" flags]
#        e.g. bash lab/mutate/run.sh recon -resume   # after adding a test or stage
#        e.g. bash lab/mutate/run.sh recon -ops stmt-swap -j 4
# Env:   ROOT  module tree to mutate (default: this checkout; point it at a
#              frozen copy when other work is editing the tree)
#        OUT   results directory (default: lab/mutate/results)
#        WORK  scratch for the per-worker module copies (default: $TMPDIR/hertamut)
#        JOBS  parallel workers (default: 6)
# Exit:  non-zero if any mutant survives that equivalent.tsv does not
#        prove equivalent.
set -euo pipefail
# Native Go needs C:/ paths; MSYS would only translate some arguments.
native() { if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf %s "$1"; fi; }
here=$(native "$(cd "$(dirname "$0")" && pwd)")
ROOT=$(native "${ROOT:-$here/../..}")
OUT=$(native "${OUT:-$here/results}")
WORK=$(native "${WORK:-${TMPDIR:-/tmp}/hertamut}")
JOBS=${JOBS:-6}
which=${1:-both}
shift || true

lab="./lab/assume ./lab/model ./lab/system ./lab/mutate"
# The platform assumptions measure Go and the machine, never Herta, so no
# mutant can change them; under the run's load they only add noise.
platform='^(TestClockResolution|TestTimerResolution|TestContextCosts|TestGoroutineCost|TestAtomicVsLocks|TestGoroutineSettling|TestGCTailLatency)$'
mutate() { (cd "$here/../.." && go run ./lab/mutate/cmd/mutate "$@"); }

target() { # $1=name  $2=package dir  $3=title  $4...=extra flags
  local name=$1 dir=$2 title=$3 pkg=./$2/
  shift 3
  [ "$dir" = . ] && pkg=.
  mutate run -root "$ROOT" -pkg "$dir" -title "$title" -triage "$here/equivalent.tsv" \
    -stage "own: go test -count=1 -short -failfast -vet=off -timeout=60s $pkg" \
    -stage "lab: LAB_IMPLS=$name go test -count=1 -failfast -vet=off -timeout=150s -skip $platform $lab" \
    -stage "race: go test -race -count=2 -short -failfast -vet=off -timeout=150s $pkg" \
    -stage "race-lab: LAB_IMPLS=$name go test -race -count=3 -failfast -vet=off -timeout=150s -run ^TestGapCancelStorm$ ./lab/mutate" \
    -out "$OUT/$name" -work "$WORK/$name" -j "$JOBS" "$@"
}

case $which in
orig | recon | both) ;;
*) echo "usage: $0 [orig|recon|both] [flags]" >&2; exit 2 ;;
esac
status=0
if [ "$which" != recon ]; then target orig . "v0.2.0 (orig)" "$@" || status=1; fi
if [ "$which" != orig ]; then target recon recon "v0.2-recon" "$@" || status=1; fi
if [ -f "$OUT/orig/run.json" ] && [ -f "$OUT/recon/run.json" ]; then
  mutate compare "orig=$OUT/orig" "recon=$OUT/recon"
fi
exit $status
