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
echo ""
echo -e "${BOLD}Checking referenced test file paths in stories...${RESET}"

missing=0

for story in docs/stories/[0-9]*.md; do
  [ -f "$story" ] || continue

  # Extract backtick-quoted paths ending in .go
  { grep -oE '`[^`]+\.go`' "$story" || true; } | tr -d '`' | while IFS= read -r gopath; do
    [ -z "$gopath" ] && continue
    if [ ! -f "$gopath" ]; then
      echo -e "  ${RED}MISSING${RESET}: $gopath (referenced in $(basename "$story"))"
      # Write to a temp file so the subshell can signal failure
      echo "1" >> /tmp/check-links-missing.$$
    fi
  done
done

if [ -f /tmp/check-links-missing.$$ ]; then
  missing=$(wc -l < /tmp/check-links-missing.$$ | tr -d ' ')
  rm -f /tmp/check-links-missing.$$
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
