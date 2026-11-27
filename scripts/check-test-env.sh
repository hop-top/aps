#!/usr/bin/env bash
# Fail when test code mutates process-wide location env without automatic
# restoration.
#
# Flagged in *_test.go and internal/testing/*.go:
#   - os.Setenv / os.Unsetenv on HOME, USERPROFILE, XDG_*_HOME, APS_DATA_PATH,
#     AppData, LocalAppData
#   - the broken restore idiom `defer os.Setenv(X, os.Getenv(X))` for ANY
#     variable (the Getenv argument is evaluated after the overwrite, so the
#     original value is never restored)
#
# Use t.Setenv / b.Setenv instead (t.Setenv(k, "") for empty; t.Setenv then
# os.Unsetenv is NOT exempt, so prefer empty-string where the code under test
# treats empty as unset).
#
# Opt-out for a legitimate site (e.g. a test of the env sandbox itself): put
#   // envguard:allow <reason>
# on the same line or the line directly above. Internal/testguard non-test
# code (the TestMain sandbox) is not scanned.
#
# Usage: ./scripts/check-test-env.sh [root-dir]

set -euo pipefail

ROOT_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$ROOT_DIR"

VARS='HOME|USERPROFILE|XDG_[A-Z_]+|APS_DATA_PATH|AppData|LocalAppData'
LOC="os\.(Setenv|Unsetenv)\(\"(${VARS})\""
BROKEN='defer[[:space:]]+os\.Setenv\(([^,]+),[[:space:]]*os\.Getenv\(([^)]*)\)\)'

files=$(find . \( -path ./.git -o -path ./hops -o -path ./vendor -o -path ./node_modules \) -prune -o \
  -type f \( -name '*_test.go' -o -path './internal/testing/*.go' \) -print)

hits=$(
  [ -z "$files" ] || echo "$files" | env LOC="$LOC" BROKEN="$BROKEN" xargs awk '
    BEGIN { loc = ENVIRON["LOC"]; broken = ENVIRON["BROKEN"] }
    FNR == 1 { prev = "" }
    {
      line = $0
      if ((line ~ loc || line ~ broken) && line !~ /envguard:allow/ && prev !~ /envguard:allow/)
        printf "%s:%d: %s\n", FILENAME, FNR, line
      prev = line
    }'
)

if [ -n "$hits" ]; then
  echo "test env guard: use t.Setenv/b.Setenv (auto-restored) instead of os.Setenv/os.Unsetenv:" >&2
  echo "$hits" >&2
  echo "(opt-out for a legitimate site: '// envguard:allow <reason>')" >&2
  exit 1
fi
echo "test env guard: ok"
