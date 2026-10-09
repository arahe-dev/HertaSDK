#!/usr/bin/env bash
# Reproduces a quiet-machine study of orig (v0.2.0) against recon.
#
#   bash lab/results/run.sh [stage...]      stages: bench doe industrial soak all
#
# Env: RUN (run id, default quiet-1), SOAK (twin soak, default 10m),
# BENCH_COUNT (default 10), DUR / WARMUP / REPS (DoE windows).
# Run on an otherwise idle machine: every stage measures time.
set -euo pipefail
cd "$(dirname "$0")/../.."

RUN=${RUN:-quiet-1}
OUT=lab/results/$RUN
mkdir -p "$OUT"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# bench: the v0.2 benchmarks plus recon's uncontended ones, run directly on
# each package (no adapter). The uncontended file joins orig's package
# through a build overlay, so the root tree stays untouched. Binaries
# alternate run by run, so machine drift spreads over both.
bench() {
  local abs; abs=$(pwd -W 2>/dev/null || pwd)
  printf '{"Replace":{"%s/perf_bench_test.go":"%s/recon/perf_bench_test.go"}}\n' "$abs" "$abs" > "$TMP/overlay.json"
  go test -c -overlay "$TMP/overlay.json" -o "$TMP/orig.test" .
  go test -c -o "$TMP/recon.test" ./recon
  : > "$OUT/bench-orig.txt"; : > "$OUT/bench-recon.txt"
  for i in $(seq 1 "${BENCH_COUNT:-10}"); do
    local order="orig recon"; (( i % 2 )) || order="recon orig"
    for im in $order; do
      "$TMP/$im.test" -test.run '^$' -test.bench . -test.benchmem -test.cpu 1,4,16 -test.count 1 \
        | grep '^Benchmark' >> "$OUT/bench-$im.txt"
    done
    echo "bench round $i done" >&2
  done
  go run ./lab/cmd/benchdiff -base orig="$OUT/bench-orig.txt" -cand recon="$OUT/bench-recon.txt" \
    -csv "$OUT/bench.csv" > "$OUT/bench.md"
  { echo "# orig"; cat "$OUT/bench-orig.txt"; echo; echo "# recon"; cat "$OUT/bench-recon.txt"; } > "$OUT/bench.txt"
}

# doe: screen, the full factorial on the screen's active factors, sweeps,
# and two A/B studies (A/A noise calibration, and the Mirai-shaped workload).
doe() {
  local w=(-dur "${DUR:-2s}" -warmup "${WARMUP:-500ms}" -q)
  go build -o "$TMP/doe" ./lab/cmd/doe
  "$TMP/doe" -design screen -runs64 -reps "${REPS:-3}" "${w[@]}" -out "$OUT/screen"
  "$TMP/doe" -design full -from "$OUT/screen/summary.json" -reps "${REPS:-3}" "${w[@]}" -out "$OUT/full"
  "$TMP/doe" -design sweep -reps 5 "${w[@]}" -out "$OUT/sweep"
  "$TMP/doe" -design ab -impls a=recon,b=recon -reps 20 "${w[@]}" -out "$OUT/aa-recon"
  "$TMP/doe" -design ab -impls a=orig,b=orig -reps 20 "${w[@]}" -out "$OUT/aa-orig"
  "$TMP/doe" -design ab -reps 20 "${w[@]}" -out "$OUT/ab"
  for d in screen full sweep aa-recon aa-orig ab; do
    tail -n +2 "$OUT/$d/runs.csv" | sed "s/^/$d,/"
  done | { echo "study,$(head -1 "$OUT/screen/runs.csv")"; cat; } > "$OUT/runs.csv"
  for d in screen full; do
    tail -n +2 "$OUT/$d/effects.csv" | sed "s/^/$d,/"
  done | { echo "study,$(head -1 "$OUT/screen/effects.csv")"; cat; } > "$OUT/effects.csv"
}

# industrial: spike, drain under load, 1e6 keys, 10k parked waiters,
# goroutines per Do. Each test appends its numbers to industrial.jsonl.
industrial() {
  : > "$OUT/industrial.jsonl"
  LAB_LEVEL=full LAB_OUT="$OUT/industrial.jsonl" go test -count=1 -v -timeout 30m ./lab/industrial \
    | tee "$OUT/industrial.log" | grep -E '^(=== RUN|--- |\s+\w+_test.go|ok|FAIL)'
}

# soak: the Mirai twin held for $SOAK per implementation.
soak() {
  LAB_LEVEL=full LAB_SOAK="${SOAK:-10m}" go test -count=1 -v -timeout 60m -run '^TestMiraiTwin$' ./lab/system \
    | tee "$OUT/soak.log" | grep -E '^(--- |ok|FAIL)|twin_test'
}

# sampled runs a stage while recording whole-machine CPU use every 5s to
# cpu-<stage>.csv (Windows typeperf), so a report can show how quiet the
# machine really was.
sampled() {
  local pid=
  if command -v typeperf >/dev/null; then
    typeperf '\Processor(_Total)\% Processor Time' -si 5 -o "$OUT/cpu-$1.csv" -y >/dev/null 2>&1 & pid=$!
  fi
  "$1"
  [[ -z $pid ]] || kill "$pid" 2>/dev/null || true
}

stages=("$@"); (( ${#stages[@]} )) || stages=(all)
for s in "${stages[@]}"; do
  case $s in
    all) for t in bench doe industrial soak; do sampled "$t"; done ;;
    bench|doe|industrial|soak) sampled "$s" ;;
    *) echo "unknown stage $s" >&2; exit 2 ;;
  esac
done
