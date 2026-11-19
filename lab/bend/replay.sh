#!/usr/bin/env bash
set -euo pipefail

upstream=${1:?usage: bash replay.sh /path/to/bendlang/bend [--verdict]}
mode=${2:---check-only}
case "$mode" in
  --check-only|--verdict) ;;
  *) echo "expected --check-only or --verdict" >&2; exit 2 ;;
esac
pin=0e2f38f759cc830849e489307f3d256a7d148288
test "$(git -C "$upstream" rev-parse HEAD)" = "$pin" || {
  echo "expected upstream checkout at $pin" >&2
  exit 2
}
root=$(dirname -- "${BASH_SOURCE[0]}")
seed="$root/testdata/retry"
tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT

check() {
  local name=$1 want_status=$2 wanted=$3 status=0
  bun "$upstream/bend2/main.ts" "$tmp/$name/PROOF.bend" "$mode" >"$tmp/out" 2>&1 || status=$?
  if [[ $status != "$want_status" ]] || ! grep -Fq -- "$wanted" "$tmp/out"; then
    cat "$tmp/out" >&2
    echo "$name: expected exit $want_status and '$wanted'; got exit $status" >&2
    exit 1
  fi
  printf '%s: exit %s (%s)\n' "$name" "$status" "$wanted"
}

cp -r "$seed" "$tmp/baseline"
check baseline 0 'ALL PROOFS CHECK'

cp -r "$seed" "$tmp/policy-mutation"
sed '/case Unknown{}:/{n;s/False{}/True{}/;}' "$seed/policy.bend" >"$tmp/policy-mutation/policy.bend"
check policy-mutation 1 'charge_stops_on_unknown'

cp -r "$tmp/policy-mutation" "$tmp/law-mutation"
sed '/Policy.Unknown{}/s/False{}/True{}/' "$seed/LAWS.bend" >"$tmp/law-mutation/LAWS.bend"
check law-mutation 0 'ALL PROOFS CHECK'

cp -r "$seed" "$tmp/missing-import"
printf '%s\n' 'def unrelated() -> U32:' '  0' >"$tmp/missing-import/PROOF.bend"
check missing-import 1 'PROOF.bend must import ./LAWS.bend'
