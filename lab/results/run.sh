#!/usr/bin/env bash
# Reproduces a quiet-machine study of orig (v0.2.0, frozen in lab/v02)
# against recon (the root package, hertasdk).
#
#   bash lab/results/run.sh [stage...]      stages: bench doe industrial soak all
#
# Env: RUN (run id, default quiet-1), SOAK (soak per implementation, default
# 10m), BENCH_COUNT (default 10), DUR / WARMUP / REPS (DoE windows).
# Run on an otherwise idle machine: every stage measures time.
set -euo pipefail
cd "$(dirname "$0")/.." # the lab module

RUN=${RUN:-quiet-1}
OUT=results/$RUN
mkdir -p "$OUT"
ABS_OUT=$(cd "$OUT" && (pwd -W 2>/dev/null || pwd)) # tests run in their package dir
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# bench: the v0.2 benchmarks plus the uncontended ones, run directly on
# each package (no adapter). The uncontended file joins lab/v02 through a
# build overlay (with its package line renamed), so neither tree is edited.
# Binaries alternate run by run, so machine drift spreads over both.
bench() {
  local lab tmp
  lab=$(pwd -W 2>/dev/null || pwd)
  tmp=$(cd "$TMP" && (pwd -W 2>/dev/null || pwd))
  sed 's/^package hertasdk$/package execution/' ../perf_bench_test.go > "$TMP/perf_bench_test.go"
  printf '{"Replace":{"%s/v02/perf_bench_test.go":"%s/perf_bench_test.go"}}\n' "$lab" "$tmp" > "$TMP/overlay.json"
  go test -c -overlay "$TMP/overlay.json" -o "$TMP/orig.test" ./v02
  go -C .. test -c -o "$TMP/recon.test" .
  : > "$OUT/bench-orig.txt"; : > "$OUT/bench-recon.txt"
  for i in $(seq 1 "${BENCH_COUNT:-10}"); do
    local order="orig recon"; (( i % 2 )) || order="recon orig"
    for im in $order; do
      "$TMP/$im.test" -test.run '^$' -test.bench . -test.benchmem -test.cpu 1,4,16 -test.count 1 \
        | grep '^Benchmark' >> "$OUT/bench-$im.txt"
    done
    echo "bench round $i done" >&2
  done
  go run ./cmd/benchdiff -base orig="$OUT/bench-orig.txt" -cand recon="$OUT/bench-recon.txt" \
    -csv "$OUT/bench.csv" > "$OUT/bench.md"
  { echo "# orig"; cat "$OUT/bench-orig.txt"; echo; echo "# recon"; cat "$OUT/bench-recon.txt"; } > "$OUT/bench.txt"
}

# doe: screen, the full factorial on the screen's active factors, sweeps,
# and two A/B studies (A/A noise calibration, and the Mirai-shaped workload).
doe() {
  local w=(-dur "${DUR:-2s}" -warmup "${WARMUP:-500ms}" -q)
  go build -o "$TMP/doe" ./cmd/doe
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

# industrial: spike, drain under load, 1e6 keys, parked waiters, goroutines
# per Do, Reject under load, and a short soak. Each test appends its numbers
# to industrial.jsonl.
industrial() {
  : > "$OUT/industrial.jsonl"
  LAB_LEVEL=full LAB_OUT="$ABS_OUT/industrial.jsonl" go test -count=1 -v -timeout 40m ./industrial \
    | tee "$OUT/industrial.log" | grep -E '^(=== RUN|--- |\s+\w+_test.go|ok|FAIL)'
}

# soak: the Mirai-shaped twin held for $SOAK per implementation.
soak() {
  LAB_LEVEL=full LAB_SOAK="${SOAK:-10m}" LAB_OUT="$ABS_OUT/soak.jsonl" \
    go test -count=1 -v -timeout 60m -run '^TestSoak$' ./industrial \
    | tee "$OUT/soak.log" | grep -E '^(--- |ok|FAIL)|soak_test'
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
