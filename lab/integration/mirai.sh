#!/usr/bin/env bash
# Real-consumer integration test: is v0.2-recon a drop-in replacement for
# the v0.2.0 fork that mirai-backend vendors in internal/execution?
#
# It copies the consumer twice (never touching the original), swaps the
# second copy's internal/execution non-test files for recon's (the fork's
# own execution_test.go stays, so v0.2's suite also runs against recon),
# builds both offline, runs the whole test suite COUNT times on each, and
# prints the package/test matrix. Postgres-backed tests skip without
# MIRAI_TEST_DATABASE_URL; the matrix lists them with their reason.
# Finally it injects probe/*_probe_test.go into both copies to show what
# the consumer itself observes of the CONTRACT §8 deltas.
#
# Usage: bash lab/integration/mirai.sh
# Env:   MIRAI  consumer checkout, read only (default /c/mirai/mirai-backend)
#        WORK   scratch for the copies and results (default $TMPDIR/herta-mirai)
#        COUNT  go test -count (default 3: repeats expose flaky differences)
#        RACE   1 adds -race (needs cgo and a C compiler)
# Out:   $WORK/matrix.md, $WORK/{base,recon}.json (go test -json),
#        $WORK/{base,recon}.probe (the probe/ tests injected into the consumer)
# Exit:  non-zero if recon fails to build, or fails a package or test that
#        the v0.2.0 baseline passes.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
herta=$(cd "$here/../.." && pwd)
MIRAI=${MIRAI:-/c/mirai/mirai-backend}
WORK=${WORK:-${TMPDIR:-/tmp}/herta-mirai}
COUNT=${COUNT:-3}
race=$([ "${RACE:-0}" = 1 ] && echo -race || true)
export GOFLAGS=-mod=mod GOPROXY=off # offline: the module cache must already hold every dependency

copy() { # $1=destination
  rm -rf "$1" && mkdir -p "$1"
  (cd "$MIRAI" && tar --exclude=./.git -cf - .) | (cd "$1" && tar -xf -)
}

copy "$WORK/base"
copy "$WORK/recon"
exec_dir=$WORK/recon/internal/execution
find "$exec_dir" -maxdepth 1 -name '*.go' ! -name '*_test.go' -delete
find "$herta/recon" -maxdepth 1 -name '*.go' ! -name '*_test.go' -exec cp {} "$exec_dir/" \;
imports=$(cd "$WORK/recon" && go list -f '{{join .Imports " "}}' ./internal/execution)
case " $imports " in *" golang.org/x/sync/"*) echo "recon must not import golang.org/x/sync: $imports" >&2; exit 1 ;; esac
echo "swapped in recon: $(cd "$exec_dir" && ls ./*.go | grep -v _test | tr '\n' ' ')"

status=0
for t in base recon; do
  echo "== $t: build, vet internal/execution, test -count=$COUNT $race"
  (cd "$WORK/$t" && go build ./... && go vet ./internal/execution/) || { echo "$t does not build" >&2; exit 1; }
  start=$(date +%s)
  (cd "$WORK/$t" && go test -count="$COUNT" $race -json ./... >"$WORK/$t.json" 2>"$WORK/$t.stderr") || true
  echo "   $(($(date +%s) - start))s"
done

(cd "$herta" && go run ./lab/integration/matrix -names "v0.2.0 fork,recon" \
  -trim github.com/mirai/mirai-backend/ "$WORK/base.json" "$WORK/recon.json") >"$WORK/matrix.md" || status=1

# Probes: what the consumer observes of the §8 deltas through its own
# error mapping. They pass on both implementations; the table shows how
# the observations differ.
{
  printf '\n### Consumer probes (%s)\n\n| probe | v0.2.0 fork | recon |\n|---|---|---|\n' "$(ls "$here"/probe)"
  for t in base recon; do
    cp "$here"/probe/*_probe_test.go "$WORK/$t/internal/events/"
    (cd "$WORK/$t" && go test -tags mirai_probe -run '^TestProbe' -count=1 -v ./internal/events/) >"$WORK/$t.probe" 2>&1 ||
      { echo "probe failed on $t" >&2; status=1; }
  done
  paste -d'\t' <(grep -o 'PROBE .*' "$WORK/base.probe") <(grep -o 'PROBE .*' "$WORK/recon.probe") |
    sed -E 's/^PROBE ([^:]*): ([^\t]*)\tPROBE [^:]*: (.*)$/| \1 | \2 | \3 |/'
} >>"$WORK/matrix.md"
cat "$WORK/matrix.md"
exit $status
