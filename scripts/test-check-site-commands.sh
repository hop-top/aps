#!/usr/bin/env bash
# Tests for scripts/check-site-commands.sh against a fixture command tree
# (APS_COMMAND_PATHS), so no Go toolchain is needed.
#
# Usage:
#   ./scripts/test-check-site-commands.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/check-site-commands.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Fixture tree: "<path>\t<flags>", as `climd --paths` prints it.
printf 'aps\t--help --profile -h -p\naps profile\t--help --profile -h -p\naps profile create\t--display-name --force --help --profile -h -n -p --note\naps profile list\t--help --profile -h -p\naps run\t--env --help --profile -h -p\n' > "$work/tree"

fails=0
# run_case <name> <expected-exit> <page-body> [expected-output-substring]
run_case() {
  local name="$1" want_rc="$2" body="$3" want_out="${4:-}" rc=0 d="$work/$1"
  mkdir -p "$d/docs"
  printf '%s\n' "$body" > "$d/docs/page.md"
  APS_COMMAND_PATHS="$work/tree" SITE_CONTENT_DIR="$d/docs" bash "$SCRIPT" > "$d/out" 2>&1 || rc=$?
  if [ "$rc" -ne "$want_rc" ]; then
    echo "FAIL $name: exit $rc (want $want_rc)"; sed 's/^/    /' "$d/out"; fails=$((fails + 1))
  elif [ -n "$want_out" ] && ! grep -qF -- "$want_out" "$d/out"; then
    echo "FAIL $name: output lacks '$want_out'"; sed 's/^/    /' "$d/out"; fails=$((fails + 1))
  else
    echo "PASS $name (exit $rc)"
  fi
}

# shellcheck disable=SC2016 # backticks are literal markdown
fence() { printf '```%s\n%s\n```' "$1" "$2"; }

run_case valid 0 "$(fence bash 'aps profile create myagent --display-name "A B" --force -n note
aps profile list
aps run myagent -- git status --not-an-aps-flag
aps myagent -- git status
aps myagent
aps <profile> -- cmd
aps profile create [id] [--display-name <s>] [--force]
$ aps profile list   # trailing comment --bogus
aps --help')"
run_case unknown-command 1 "$(fence bash 'aps profile new myagent')" "unknown subcommand 'new' of 'aps profile'"
run_case unknown-top-level 1 "$(fence bash 'aps wallet balance')" "unknown command 'aps wallet'"
run_case unknown-flag 1 "$(fence bash 'aps profile create x --debug')" "no flag --debug"
run_case unknown-short-flag 1 "$(fence bash 'aps profile list -z')" "no flag -z"
run_case flag-of-other-command 1 "$(fence bash 'aps profile list --display-name x')" "no flag --display-name"
run_case flag-after-separator-ignored 0 "$(fence bash 'aps run p -- tool --anything -x')"
run_case optional-syntax-checked 1 "$(fence bash 'aps profile create [id] [--debug <s>]')" "no flag --debug"
run_case skip-comment 0 "$(fence bash 'aps profile new x   # docs-check: skip')"
run_case non-shell-fence-ignored 0 "$(fence console '$ aps profile new x')"
# shellcheck disable=SC2016 # backticks are literal markdown
run_case prose-ignored 0 'Use `aps profile new x` or: aps wallet'
run_case location-reported 1 "$(printf 'x\n\n%s' "$(fence bash 'aps wallet balance')")" "page.md:4"

[ "$fails" -eq 0 ] || { echo "$fails test(s) failed"; exit 1; }
echo "all check-site-commands tests passed"
