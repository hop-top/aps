#!/usr/bin/env bash
# Tests for the retry behaviour of scripts/check-links.sh, using a stub
# `lychee` placed first on PATH. No network, no real lychee.
#
# Usage:
#   ./scripts/test-check-links.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/check-links.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/bin"
# Stub: counts calls in $STUB_DIR/calls; the first STUB_FAIL_CALLS calls (or
# all, if "always") write a lychee-style JSON report with one 503 and exit 2.
# STUB_KIND=file makes the failure a missing local file instead.
cat > "$work/bin/lychee" <<'STUB'
#!/usr/bin/env bash
out=""
while [ $# -gt 0 ]; do
  case "$1" in --output) out="$2"; shift ;; esac
  shift
done
n=$(( $(cat "$STUB_DIR/calls" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$STUB_DIR/calls"
fail=false
if [ "${STUB_FAIL_CALLS:-0}" = always ] || [ "$n" -le "${STUB_FAIL_CALLS:-0}" ]; then fail=true; fi
if $fail; then
  url="https://github.com/example/repo/blob/main/x.md"
  [ "${STUB_KIND:-}" = file ] && url="file:///nope/missing.md"
  printf '{"error_map":{"docs/a.md":[{"url":"%s","status":{"text":"Rejected status code: 503 Service Unavailable","code":503},"span":{"line":3}}]},"timeout_map":{}}\n' "$url" > "$out"
  exit 2
fi
printf '{"error_map":{},"timeout_map":{}}\n' > "$out"
exit 0
STUB
chmod +x "$work/bin/lychee"

fails=0
# run_case <name> <expected-exit> <expected-calls> <fail-calls> [kind]
run_case() {
  local name="$1" want_rc="$2" want_calls="$3" fail_calls="$4" kind="${5:-}" rc=0 calls
  local d="$work/$name"
  mkdir -p "$d"
  PATH="$work/bin:$PATH" STUB_DIR="$d" STUB_FAIL_CALLS="$fail_calls" STUB_KIND="$kind" \
    LINK_CHECK_RETRY_WAITS="${WAITS-0 0 0}" \
    bash "$SCRIPT" > "$d/out" 2>&1 || rc=$?
  calls="$(cat "$d/calls" 2>/dev/null || echo 0)"
  if [ "$rc" -eq "$want_rc" ] && [ "$calls" -eq "$want_calls" ]; then
    echo "PASS $name (exit $rc, lychee calls $calls)"
  else
    echo "FAIL $name: exit $rc (want $want_rc), lychee calls $calls (want $want_calls)"
    sed 's/^/    /' "$d/out"
    fails=$((fails + 1))
  fi
}

run_case clean            0 1 0
run_case transient-1      0 2 1
run_case transient-3      0 4 3
run_case persistent       1 4 always
run_case permanent-file   1 1 always file
WAITS="" run_case retry-disabled 1 1 always

if grep -q 'github.com/example/repo/blob/main/x.md' "$work/persistent/out"; then
  echo "PASS persistent reports failing link"
else
  echo "FAIL persistent output lacks failing link"
  fails=$((fails + 1))
fi

[ "$fails" -eq 0 ] || { echo "$fails test(s) failed"; exit 1; }
echo "all check-links tests passed"
