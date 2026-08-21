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
echo -e "${BOLD}Checking markdown links with lychee...${RESET}"
if command -v lychee >/dev/null 2>&1; then
  if ! lychee --config .lychee.toml 'docs/**/*.md'; then
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

missing_file="$(mktemp)"
trap 'rm -f "$missing_file"' EXIT

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
