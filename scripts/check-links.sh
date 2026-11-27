#!/usr/bin/env bash
# Validate documentation links and referenced test files.
#
# 1. Runs lychee on all markdown files under docs/ to check URL and
#    relative-path links.
# 2. Parses story files for backtick-quoted .go file paths and verifies
#    they exist on disk.
#
# Usage:
#   ./scripts/check-links.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Colors (disabled when not a terminal)
if [ -t 1 ]; then
  GREEN='\033[0;32m' RED='\033[0;31m' CYAN='\033[0;36m'
  BOLD='\033[1m' RESET='\033[0m'
else
  GREEN='' RED='' CYAN='' BOLD='' RESET=''
fi

errors=0

# --- Part 1: Lychee markdown link check ---
#
# External hosts (notably github.com blob pages) intermittently answer 503
# for URLs that are fine seconds later. lychee already retries 5xx/429/timeouts
# itself (max_retries, exponential retry_wait_time), but that backoff is too
# short to outlast the burst. So: run once; if only http(s) links failed,
# re-check just those URLs in a fresh lychee run after a growing wait. A link
# is reported broken only if it fails every attempt. Permanent failures
# (missing local files, bad config) fail at once, without waiting.
#
# LINK_CHECK_RETRY_WAITS: space-separated seconds to wait before each re-check
# (default "20 40 80": 140s of waiting at most). Empty disables re-checks.
read -ra retry_waits <<< "${LINK_CHECK_RETRY_WAITS-20 40 80}"

work_dir="$(mktemp -d)"
missing_file="$work_dir/missing"
trap 'rm -rf "$work_dir"' EXIT

# print_failures <json>: one line per failed link with its source location.
print_failures() {
  jq -r '(.error_map, .timeout_map // {}) | to_entries[]
         | .key as $f | .value[]
         | "  \(.url)  [\(.status.text // "failed")]  (\($f):\(.span.line // 0))"' "$1"
}

# failed_urls <json>: unique failed URLs, one per line.
failed_urls() {
  jq -r '[(.error_map, .timeout_map // {})[][].url] | unique[]' "$1"
}

check_docs_links() {
  local json="$work_dir/lychee.json" rc=0 attempt=0 wait_s urls recheck
  lychee --config .lychee.toml --no-progress --format json \
    --output "$json" 'docs/**/*.md' >/dev/null || rc=$?
  [ "$rc" -eq 0 ] && return 0

  # Anything but "links failed" (exit 2) is a runtime/config error: no retry.
  if [ "$rc" -ne 2 ] || ! jq -e . "$json" >/dev/null 2>&1; then
    echo "lychee exited with status $rc" >&2
    return 1
  fi

  urls="$(failed_urls "$json")"
  if printf '%s\n' "$urls" | grep -qvE '^https?://'; then
    print_failures "$json" >&2
    return 1 # a non-http failure (e.g. missing file) will not heal
  fi

  for wait_s in ${retry_waits[@]+"${retry_waits[@]}"}; do
    attempt=$((attempt + 1))
    echo -e "${CYAN}$(printf '%s\n' "$urls" | wc -l | tr -d ' ') link(s) failed; re-checking in ${wait_s}s (re-check ${attempt}/${#retry_waits[@]})...${RESET}"
    sleep "$wait_s"
    recheck="$work_dir/recheck.md"
    printf '%s\n' "$urls" | sed 's/.*/<&>/' > "$recheck"
    rc=0
    lychee --config .lychee.toml --no-progress --format json \
      --output "$json" "$recheck" >/dev/null || rc=$?
    [ "$rc" -eq 0 ] && return 0
    if [ "$rc" -ne 2 ] || ! jq -e . "$json" >/dev/null 2>&1; then
      echo "lychee exited with status $rc" >&2
      return 1
    fi
    urls="$(failed_urls "$json")"
  done

  echo "Still failing after $((${#retry_waits[@]} + 1)) attempt(s):" >&2
  printf '%s\n' "$urls" | sed 's/^/  /' >&2
  return 1
}

echo -e "${BOLD}Checking markdown links with lychee...${RESET}"
if command -v lychee >/dev/null 2>&1; then
  if ! check_docs_links; then
    echo -e "${RED}Lychee found broken links.${RESET}"
    errors=$((errors + 1))
  else
    echo -e "${GREEN}All markdown links OK.${RESET}"
  fi
else
  echo -e "${CYAN}lychee not installed — skipping URL checks (run 'mise install' to add it).${RESET}"
fi

# --- Part 2: Verify referenced test file paths exist ---
#
# Only paths inside a story's "## Tests" section are checked. Elsewhere a
# story may legitimately name a file that does not exist here: out-of-scope
# work, a path in another repo, or a brace-glob standing for several files.
# Those are prose, not references that must resolve. Paths rooted at a
# sibling repo are skipped even inside that section.
echo ""
echo -e "${BOLD}Checking referenced test file paths in stories...${RESET}"

for story in docs/stories/[0-9]*.md; do
  [ -f "$story" ] || continue

  # Emit only the lines within the "## Tests" section: start at that heading,
  # stop at the next heading of the same or higher level.
  tests_section="$(awk '
    /^##[^#]/ { in_tests = ($0 ~ /^##[[:space:]]+Tests[[:space:]]*$/) ? 1 : 0; next }
    in_tests  { print }
  ' "$story")"

  [ -n "$tests_section" ] || continue

  printf '%s\n' "$tests_section" \
    | { grep -oE '`[^`]+\.go`' || true; } \
    | tr -d '`' \
    | while IFS= read -r gopath; do
        [ -z "$gopath" ] && continue
        # Paths rooted at a sibling repo (kit/, tlc/, wsm/, cxr/, upgrade/)
        # describe files outside this checkout and cannot be verified here.
        case "$gopath" in
          kit/*|tlc/*|wsm/*|cxr/*|upgrade/*) continue ;;
        esac
        if [ ! -f "$gopath" ]; then
          echo -e "  ${RED}MISSING${RESET}: $gopath (referenced in $(basename "$story"))"
          echo "1" >> "$missing_file"
        fi
      done
done

if [ -s "$missing_file" ]; then
  missing=$(wc -l < "$missing_file" | tr -d ' ')
  echo -e "${RED}${missing} referenced test file(s) not found.${RESET}"
  errors=$((errors + missing))
else
  echo -e "${GREEN}All referenced test files exist.${RESET}"
fi

# --- Summary ---
echo ""
if [ "$errors" -gt 0 ]; then
  echo -e "${RED}${BOLD}Documentation check failed with ${errors} error(s).${RESET}"
  exit 1
else
  echo -e "${GREEN}${BOLD}All documentation checks passed.${RESET}"
fi
