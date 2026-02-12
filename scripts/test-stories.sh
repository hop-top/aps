#!/usr/bin/env bash
# Run all Go tests linked in docs/stories/*.md
#
# Parses "## Tests" sections from each story file, extracts
# test file paths and function names, then runs them grouped
# by Go package using `go test -run`.
#
# Usage:
#   ./scripts/test-stories.sh              # run all story tests
#   ./scripts/test-stories.sh 001 003      # run tests for specific stories
#   ./scripts/test-stories.sh --list       # list tests without running

set -euo pipefail

STORY_DIR="docs/stories"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT_DIR"

# Colors (disabled when not a terminal)
if [ -t 1 ]; then
  GREEN='\033[0;32m' RED='\033[0;31m' CYAN='\033[0;36m'
  YELLOW='\033[0;33m' BOLD='\033[1m' RESET='\033[0m'
else
  GREEN='' RED='' CYAN='' YELLOW='' BOLD='' RESET=''
fi

list_only=false
story_filter=()

for arg in "$@"; do
  case "$arg" in
    --list|-l) list_only=true ;;
    *) story_filter+=("$arg") ;;
  esac
done

# Collect: package -> "TestA|TestB|TestC"
declare -A pkg_tests
# Track: story -> test count (for summary)
declare -A story_counts

# Convert a test file path (e.g. tests/unit/core/config_test.go) to a Go
# package import path relative to the module (e.g. ./tests/unit/core)
file_to_pkg() {
  local f="$1"
  # Strip leading ./ if present
  f="${f#./}"
  # For internal/* paths, use the directory directly
  # For tests/* paths, same
  local dir
  dir="$(dirname "$f")"
  echo "./${dir}"
}

# Parse a single story file and populate pkg_tests
parse_story() {
  local file="$1"
  local id
  id="$(basename "$file" .md | grep -oE '^[0-9]+')"

  # If filter is set, skip non-matching stories
  if [ ${#story_filter[@]} -gt 0 ]; then
    local match=false
    for fid in "${story_filter[@]}"; do
      # Normalize: strip leading zeros for comparison
      if [ "$((10#$fid))" = "$((10#$id))" ]; then
        match=true
        break
      fi
    done
    if [ "$match" = false ]; then
      return
    fi
  fi

  local in_tests=false
  local count=0

  while IFS= read -r line; do
    # Detect entry into Tests section
    if [[ "$line" =~ ^##[[:space:]]+Tests ]]; then
      in_tests=true
      continue
    fi

    # Exit Tests section on next h2
    if $in_tests && [[ "$line" =~ ^##[[:space:]] ]] && [[ ! "$line" =~ ^###[[:space:]] ]]; then
      in_tests=false
      continue
    fi

    if ! $in_tests; then
      continue
    fi

    # Match lines like: - `tests/e2e/profile_test.go` — `TestProfileLifecycle`, `TestProfileOverwrite`
    if [[ "$line" =~ ^-[[:space:]]+\` ]]; then
      # Extract file path (first backtick-quoted value)
      local test_file
      test_file="$(echo "$line" | sed -n 's/^- *`\([^`]*\)`.*/\1/p')"
      if [ -z "$test_file" ]; then
        continue
      fi

      local pkg
      pkg="$(file_to_pkg "$test_file")"

      # Skip lines with no em dash — they reference a file without specific functions
      if [[ "$line" != *"—"* ]]; then
        continue
      fi

      # Extract test function names (all backtick-quoted values after the em dash)
      local funcs
      funcs="$(echo "$line" | sed 's/^[^—]*— *//' | grep -oE '`[^`]+`' | tr -d '`' | tr '\n' '|')"
      funcs="${funcs%|}" # trim trailing pipe

      if [ -z "$funcs" ]; then
        continue
      fi

      # Merge into pkg_tests
      if [ -n "${pkg_tests[$pkg]+x}" ]; then
        pkg_tests[$pkg]="${pkg_tests[$pkg]}|${funcs}"
      else
        pkg_tests[$pkg]="$funcs"
      fi

      # Count functions
      local n
      n="$(echo "$funcs" | tr '|' '\n' | wc -l | tr -d ' ')"
      count=$((count + n))
    fi
  done < "$file"

  story_counts[$id]=$count
}

# Parse all story files
for story_file in "$STORY_DIR"/[0-9]*.md; do
  [ -f "$story_file" ] || continue
  parse_story "$story_file"
done

# Deduplicate test names within each package
declare -A pkg_tests_dedup
for pkg in "${!pkg_tests[@]}"; do
  deduped="$(echo "${pkg_tests[$pkg]}" | tr '|' '\n' | sort -u | paste -sd'|' -)"
  pkg_tests_dedup[$pkg]="$deduped"
done

# Summary
total_tests=0
total_pkgs="${#pkg_tests_dedup[@]}"
for pkg in "${!pkg_tests_dedup[@]}"; do
  n="$(echo "${pkg_tests_dedup[$pkg]}" | tr '|' '\n' | wc -l | tr -d ' ')"
  total_tests=$((total_tests + n))
done

# Stories with tests
stories_with_tests=0
stories_without_tests=0
for id in "${!story_counts[@]}"; do
  if [ "${story_counts[$id]}" -gt 0 ]; then
    stories_with_tests=$((stories_with_tests + 1))
  else
    stories_without_tests=$((stories_without_tests + 1))
  fi
done

echo -e "${BOLD}User Story Test Runner${RESET}"
echo -e "Stories: ${CYAN}${stories_with_tests}${RESET} with tests, ${YELLOW}${stories_without_tests}${RESET} without"
echo -e "Tests:   ${CYAN}${total_tests}${RESET} functions across ${CYAN}${total_pkgs}${RESET} packages"
echo ""

if $list_only; then
  for pkg in $(echo "${!pkg_tests_dedup[@]}" | tr ' ' '\n' | sort); do
    echo -e "${CYAN}${pkg}${RESET}"
    echo "${pkg_tests_dedup[$pkg]}" | tr '|' '\n' | sort | while read -r fn; do
      echo "  $fn"
    done
  done
  exit 0
fi

# Run tests
passed=0
failed=0
failed_pkgs=()

for pkg in $(echo "${!pkg_tests_dedup[@]}" | tr ' ' '\n' | sort); do
  pattern="^(${pkg_tests_dedup[$pkg]})$"
  echo -e "${BOLD}--- ${pkg}${RESET}"
  echo -e "    -run '${pattern}'"
  if go test -v -run "$pattern" "$pkg" 2>&1 | while IFS= read -r line; do echo "    $line"; done; then
    echo -e "    ${GREEN}PASS${RESET}"
    passed=$((passed + 1))
  else
    echo -e "    ${RED}FAIL${RESET}"
    failed=$((failed + 1))
    failed_pkgs+=("$pkg")
  fi
  echo ""
done

echo -e "${BOLD}=== Results ===${RESET}"
echo -e "Packages: ${GREEN}${passed} passed${RESET}, ${RED}${failed} failed${RESET} / ${total_pkgs} total"

if [ ${#failed_pkgs[@]} -gt 0 ]; then
  echo -e "\n${RED}Failed packages:${RESET}"
  for p in "${failed_pkgs[@]}"; do
    echo -e "  ${RED}${p}${RESET}"
  done
  exit 1
fi
