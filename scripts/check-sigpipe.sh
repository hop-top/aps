#!/usr/bin/env bash
# Fail when a shell script that runs under `set -o pipefail` pipes a
# writer into a reader that can exit before consuming all input.
#
# Such a reader (head, grep -q, sed ...q, awk ...exit, read) lets the
# writer die of SIGPIPE once the data exceeds a pipe buffer; pipefail
# turns that into a failed pipeline (exit 141). Under `set -e` it
# aborts the script; in an `if`/`!`/`&&` condition it reads as a false
# "no match" and the script carries on down the wrong branch.
#
# Scanned: *.sh files that mention pipefail, plus everything under
# adapters/ (the action scripts inherit pipefail and the sourced
# _lib.sh runs under it). Logical lines are joined across a trailing
# `\` or `|` first, so a reader on the next line is still seen.
#
# Fix by giving the reader a file instead of a pipe:
#   grep -q PAT <<<"$var"          instead of   printf '%s\n' "$var" | grep -q PAT
#   first=${var%%$'\n'*}           instead of   printf '%s\n' "$var" | head -1
# or use a reader that drains its input (grep -c, sed -n 1p).
#
# Opt-out for a reader known to see bounded input: put
#   # sigpipe:allow <reason>
# on the same line or the line directly above.
#
# Usage: ./scripts/check-sigpipe.sh [root-dir]

set -euo pipefail

ROOT_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$ROOT_DIR"

files=$(find . \( -path ./.git -o -path ./hops -o -path ./vendor -o -path ./node_modules \) -prune -o \
  -type f -name '*.sh' -print | sort)

targets=""
for f in $files; do
  case "$f" in
    ./adapters/*) targets="$targets $f" ;;
    *) grep -q 'pipefail' "$f" && targets="$targets $f" ;;
  esac
done
[ -n "$targets" ] || exit 0

# shellcheck disable=SC2086
hits=$(awk '
  function flush(   stripped) {
    if (buf == "") return
    stripped = buf
    if (buf !~ /^[[:space:]]*#/ && buf !~ /sigpipe:allow/ && prevline !~ /sigpipe:allow/ &&
        stripped ~ /\|[[:space:]]*(head([[:space:]]|$)|read([[:space:]]|$)|grep[[:space:]]+(-[A-Za-z-]+[[:space:]]+)*(-[A-Za-z]*q[A-Za-z]*|--quiet|--silent|-[A-Za-z]*m[[:space:]]*[0-9]+)([[:space:]]|$)|sed[[:space:]][^|]*([\/; {]q[\x27"; }]|\x27q\x27)|awk[[:space:]][^|]*exit)/)
      printf "%s:%d: %s\n", FILENAME, start, buf
    prevline = last
    buf = ""
  }
  FNR == 1 { flush(); prevline = "" }
  {
    last = $0
    if (buf == "") start = FNR
    line = $0
    sub(/[[:space:]]+$/, "", line)
    if (line ~ /\\$/) { sub(/\\$/, "", line); buf = buf line " "; next }
    if (line ~ /\|$/ && line !~ /\|\|$/) { buf = buf line " "; next }
    buf = buf line
    flush()
  }
  END { flush() }
' $targets)

if [ -n "$hits" ]; then
  echo "check-sigpipe: writer piped into an early-exit reader under pipefail:" >&2
  echo "$hits" | sed 's/^/  /' >&2
  echo "Use a here-string or a draining reader; see the header of scripts/check-sigpipe.sh." >&2
  exit 1
fi
