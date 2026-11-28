#!/usr/bin/env bash
# Validate documentation links and referenced test files.
#
# 1. Runs lychee on all markdown files under docs/ and on the docs-site
#    pages (site/src/content/docs/**/*.md, *.mdx) to check URL and
#    relative-path links, in one run with one shared re-check of failed
#    URLs.
# 2. Checks the docs-site routes: every root-relative link in a site page
#    (`/guides/isolation/`) must resolve to a content page, and every
#    sidebar slug in the Astro config must name an existing page. lychee
#    cannot do this without loosening the docs/ checks (see below).
# 3. Parses story files for backtick-quoted .go file paths and verifies
#    they exist on disk.
#
# Env:
#   SITE_DIR                  site project dir (default: site)
#   LINK_CHECK_RETRY_WAITS    see Part 1
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

SITE_DIR="${SITE_DIR:-site}"
SITE_CONTENT="$SITE_DIR/src/content/docs"

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

# Site pages link to each other with root-relative Starlight routes
# (/guides/isolation/). lychee can resolve those only with --root-dir plus
# --fallback-extensions md,mdx (and --index-files for directory routes), and
# those options are global to a run: they would also make docs/ links like
# `[x](page)` pass when only page.md exists, and --index-files rejects the
# legitimate directory links in docs/. So the routes are checked by Part 2
# below (exact), and lychee gets an empty root dir whose resolved URLs are
# excluded, which makes it skip root-relative links instead of erroring.
noroot_dir="$work_dir/lychee-noroot"
mkdir -p "$noroot_dir"

lychee_inputs=('docs/**/*.md')
if [ -d "$SITE_CONTENT" ]; then
  lychee_inputs+=("$SITE_CONTENT/**/*.md" "$SITE_CONTENT/**/*.mdx")
fi

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
    --root-dir "$noroot_dir" --exclude '/lychee-noroot/' \
    --output "$json" "${lychee_inputs[@]}" >/dev/null || rc=$?
  [ "$rc" -eq 0 ] && return 0

  # Anything but "links failed" (exit 2) is a runtime/config error: no retry.
  if [ "$rc" -ne 2 ] || ! jq -e . "$json" >/dev/null 2>&1; then
    echo "lychee exited with status $rc" >&2
    return 1
  fi

  urls="$(failed_urls "$json")"
  # Keep where each URL was found: the re-check runs lose that.
  first_failures="$work_dir/first-failures"
  print_failures "$json" > "$first_failures"
  # Here-string, not `printf | grep -q`: grep -q stops at the first
  # non-http line and under pipefail the still-writing printf dies of
  # SIGPIPE on a long list, which read as "all http" and retried a
  # permanent failure.
  if grep -qvE '^https?://' <<<"$urls"; then
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
  while IFS= read -r u; do
    [ -n "$u" ] || continue
    # first-failures lines are "  <url>  [<status>]  (<file>:<line>)".
    grep -F -- "  $u  " "$first_failures" >&2 || echo "  $u" >&2
  done <<<"$urls"
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

# --- Part 2: docs-site routes and sidebar slugs ---
#
# Plain file checks: no network, no waiting, so they fail at once.

# route_exists <route>: true when a root-relative link target is a page
# (<route>.md|.mdx, <route>/index.md|.mdx; "/" is the index page) or a
# static asset under <site>/public.
route_exists() {
  local r="$1"
  r="${r%%#*}"
  r="${r%%\?*}"
  r="${r#/}"
  r="${r%/}"
  if [ -z "$r" ]; then
    [ -f "$SITE_CONTENT/index.md" ] || [ -f "$SITE_CONTENT/index.mdx" ]
    return
  fi
  [ -f "$SITE_CONTENT/$r.md" ] || [ -f "$SITE_CONTENT/$r.mdx" ] \
    || [ -f "$SITE_CONTENT/$r/index.md" ] || [ -f "$SITE_CONTENT/$r/index.mdx" ] \
    || [ -f "$SITE_DIR/public/$r" ]
}

# root_links <file>...: "<file>:<line><TAB><route>" for each root-relative
# markdown link or href outside fenced code blocks and inline code spans.
root_links() {
  awk '
    FNR == 1 { fence = 0 }
    /^[[:space:]]*(```|~~~)/ { fence = !fence; next }
    fence { next }
    {
      line = $0
      gsub(/`[^`]*`/, "", line)
      while (match(line, /\]\(\/[^)[:space:]]*\)|href=("|\047)\/[^"\047]*("|\047)/)) {
        m = substr(line, RSTART, RLENGTH)
        line = substr(line, RSTART + RLENGTH)
        sub(/^\]\(/, "", m); sub(/\)$/, "", m)
        sub(/^href=("|\047)/, "", m); sub(/("|\047)$/, "", m)
        if (m ~ /^\/\//) continue
        printf "%s:%d\t%s\n", FILENAME, FNR, m
      }
    }
  ' "$@"
}

check_site_routes() {
  local files bad=0 where route
  files="$(find "$SITE_CONTENT" -type f \( -name '*.md' -o -name '*.mdx' \) | sort)"
  [ -n "$files" ] || return 0
  while IFS=$'\t' read -r where route; do
    [ -n "$where" ] || continue
    if ! route_exists "$route"; then
      echo -e "  ${RED}BROKEN ROUTE${RESET}: $route  ($where)" >&2
      bad=$((bad + 1))
    fi
  done < <(
    # shellcheck disable=SC2086 # word-splitting of the file list is intended
    root_links $files
  )
  [ "$bad" -eq 0 ]
}

# check_sidebar_slugs: every `slug: '...'` in the Astro config must name a
# page; every `autogenerate: { directory: '...' }` must name a content dir
# holding at least one page.
check_sidebar_slugs() {
  local cfg bad=0 slug dir
  for cfg in "$SITE_DIR"/astro.config.*; do
    [ -f "$cfg" ] || continue
    while IFS= read -r slug; do
      [ -n "$slug" ] || continue
      if ! route_exists "/$slug"; then
        echo -e "  ${RED}SIDEBAR SLUG WITHOUT PAGE${RESET}: $slug  ($cfg)" >&2
        bad=$((bad + 1))
      fi
    done < <(grep -oE "slug:[[:space:]]*['\"][^'\"]*['\"]" "$cfg" | sed -E "s/^slug:[[:space:]]*['\"]//; s/['\"]$//")
    while IFS= read -r dir; do
      [ -n "$dir" ] || continue
      if [ -z "$(find "$SITE_CONTENT/$dir" -type f \( -name '*.md' -o -name '*.mdx' \) 2>/dev/null | sed -n 1p)" ]; then
        echo -e "  ${RED}SIDEBAR DIRECTORY WITHOUT PAGES${RESET}: $dir  ($cfg)" >&2
        bad=$((bad + 1))
      fi
    done < <(grep -oE "directory:[[:space:]]*['\"][^'\"]*['\"]" "$cfg" | sed -E "s/^directory:[[:space:]]*['\"]//; s/['\"]$//")
  done
  [ "$bad" -eq 0 ]
}

echo ""
echo -e "${BOLD}Checking docs-site routes and sidebar slugs...${RESET}"
if [ -d "$SITE_CONTENT" ]; then
  site_ok=1
  check_site_routes || site_ok=0
  check_sidebar_slugs || site_ok=0
  if [ "$site_ok" -eq 1 ]; then
    echo -e "${GREEN}All site routes and sidebar slugs resolve.${RESET}"
  else
    echo -e "${RED}Site has broken internal links or sidebar entries.${RESET}"
    errors=$((errors + 1))
  fi
else
  echo -e "${CYAN}No site content at $SITE_CONTENT — skipping.${RESET}"
fi

# --- Part 3: Verify referenced test file paths exist ---
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
