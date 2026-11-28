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
# STUB_KIND=file makes the failure a missing local file instead; STUB_KIND=site
# makes the failing link live in a docs-site page. Every call's arguments are
# appended to $STUB_DIR/args.
cat > "$work/bin/lychee" <<'STUB'
#!/usr/bin/env bash
out=""
printf '%s\n' "$*" >> "$STUB_DIR/args"
while [ $# -gt 0 ]; do
  case "$1" in --output) out="$2"; shift ;; esac
  shift
done
n=$(( $(cat "$STUB_DIR/calls" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$STUB_DIR/calls"
fail=false
if [ "${STUB_FAIL_CALLS:-0}" = always ] || [ "$n" -le "${STUB_FAIL_CALLS:-0}" ]; then fail=true; fi
if $fail && [ "${STUB_KIND:-}" = file-many ]; then
  # One missing local file sorted ahead of thousands of http failures: the
  # url list handed to the non-http check exceeds a pipe buffer.
  {
    printf '{"error_map":{"docs/a.md":[{"url":"file:///nope/missing.md","status":{"text":"Cannot find file","code":404},"span":{"line":1}}'
    for i in $(seq 1 4000); do
      printf ',{"url":"https://example.com/page/%05d","status":{"text":"503","code":503},"span":{"line":2}}' "$i"
    done
    printf ']},"timeout_map":{}}\n'
  } > "$out"
  exit 2
fi
if $fail; then
  url="https://github.com/example/repo/blob/main/x.md"
  src="docs/a.md"
  [ "${STUB_KIND:-}" = file ] && url="file:///nope/missing.md"
  [ "${STUB_KIND:-}" = site ] && src="site/src/content/docs/guides/x.md"
  printf '{"error_map":{"'"$src"'":[{"url":"%s","status":{"text":"Rejected status code: 503 Service Unavailable","code":503},"span":{"line":3}}]},"timeout_map":{}}\n' "$url" > "$out"
  exit 2
fi
printf '{"error_map":{},"timeout_map":{}}\n' > "$out"
exit 0
STUB
chmod +x "$work/bin/lychee"

fails=0
# run_case <name> <expected-exit> <expected-calls> <fail-calls> [kind]
# SITE_DIR (env) points the script at a fixture site; default is the real one.
run_case() {
  local name="$1" want_rc="$2" want_calls="$3" fail_calls="$4" kind="${5:-}" rc=0 calls
  local d="$work/$name"
  mkdir -p "$d"
  PATH="$work/bin:$PATH" STUB_DIR="$d" STUB_FAIL_CALLS="$fail_calls" STUB_KIND="$kind" \
    LINK_CHECK_RETRY_WAITS="${WAITS-0 0 0}" SITE_DIR="${SITE_DIR:-site}" \
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
# A permanent failure must not be retried however long the failed-url list is.
run_case permanent-file-large 1 1 always file-many
WAITS="" run_case retry-disabled 1 1 always

# --- docs-site checks, against fixture sites ---

# make_site <name>: fixture site with an index, two pages, a sidebar config.
make_site() {
  local s="$work/$1"
  mkdir -p "$s/src/content/docs/guides" "$s/src/content/docs/reference" "$s/public"
  : > "$s/public/logo.svg"
  printf '# Home\n\n[go](/guides/one/) [ref](/reference/)\n' > "$s/src/content/docs/index.mdx"
  # shellcheck disable=SC2016 # backticks are literal markdown
  printf '# One\n\n[two](/guides/two/#part) [home](/) [logo](/logo.svg)\n`[code](/not/checked/)`\n```\n[fenced](/not/checked/)\n```\n' > "$s/src/content/docs/guides/one.md"
  printf '# Two\n\n<a href="/guides/one/">one</a> [ext](https://example.com/x)\n' > "$s/src/content/docs/guides/two.md"
  printf '# Ref\n' > "$s/src/content/docs/reference/index.md"
  cat > "$s/astro.config.mjs" <<'CFG'
export default { sidebar: [
  { label: 'G', items: [ { label: 'One', slug: 'guides/one' }, { label: 'Two', slug: "guides/two" } ] },
  { label: 'R', items: [ { autogenerate: { directory: 'reference' } } ] },
] };
CFG
}

make_site site-ok
SITE_DIR="$work/site-ok" run_case site-routes-ok 0 1 0
if grep -q 'All site routes and sidebar slugs resolve' "$work/site-routes-ok/out"; then
  echo "PASS site-routes-ok reports success"
else
  echo "FAIL site-routes-ok lacks success line"; fails=$((fails + 1))
fi

# Both docs and site pages go to one lychee run; no per-input second run.
if [ "$(wc -l < "$work/site-routes-ok/args" | tr -d ' ')" = 1 ] \
  && grep -q "docs/\*\*/\*.md" "$work/site-routes-ok/args" \
  && grep -q "site-ok/src/content/docs/\*\*/\*.md" "$work/site-routes-ok/args" \
  && grep -q "site-ok/src/content/docs/\*\*/\*.mdx" "$work/site-routes-ok/args"; then
  echo "PASS docs and site pages share one lychee run"
else
  echo "FAIL lychee not invoked once with docs + site inputs:"; sed 's/^/    /' "$work/site-routes-ok/args"
  fails=$((fails + 1))
fi

# expect_out <case> <pattern>: output of <case> must contain <pattern>.
expect_out() {
  if grep -qF -- "$2" "$work/$1/out"; then
    echo "PASS $1 output names '$2'"
  else
    echo "FAIL $1 output lacks '$2'"; sed 's/^/    /' "$work/$1/out"; fails=$((fails + 1))
  fi
}

# Broken root-relative link: fails at once (no retries), naming file + link.
make_site site-badroute
printf '[bad](/guides/missing/) [badidx](/nope)\n' >> "$work/site-badroute/src/content/docs/guides/two.md"
SITE_DIR="$work/site-badroute" run_case site-route-broken 1 1 0
expect_out site-route-broken '/guides/missing/'
expect_out site-route-broken 'site-badroute/src/content/docs/guides/two.md:'
# A .md-suffixed route is not a Starlight route.
make_site site-mdroute
printf '[bad](/guides/two.md)\n' >> "$work/site-mdroute/src/content/docs/guides/one.md"
SITE_DIR="$work/site-mdroute" run_case site-route-md-suffix 1 1 0
expect_out site-route-md-suffix '/guides/two.md'
# A directory route without an index page is broken.
make_site site-dirroute
mkdir -p "$work/site-dirroute/src/content/docs/empty"
printf '[bad](/empty/)\n' >> "$work/site-dirroute/src/content/docs/guides/one.md"
SITE_DIR="$work/site-dirroute" run_case site-route-no-index 1 1 0
expect_out site-route-no-index '/empty/'

# Sidebar slug naming a missing page.
make_site site-badslug
sed -i.bak "s#guides/two#guides/gone#" "$work/site-badslug/astro.config.mjs" && rm "$work/site-badslug/astro.config.mjs.bak"
SITE_DIR="$work/site-badslug" run_case site-slug-missing 1 1 0
expect_out site-slug-missing 'guides/gone'
# Sidebar autogenerate directory with no pages.
make_site site-baddir
sed -i.bak "s#directory: 'reference'#directory: 'ghost'#" "$work/site-baddir/astro.config.mjs" && rm "$work/site-baddir/astro.config.mjs.bak"
SITE_DIR="$work/site-baddir" run_case site-dir-missing 1 1 0
expect_out site-dir-missing 'ghost'

# External links in site pages ride the same re-check: transient failure of
# a site link heals on re-check; persistent failure is retried then reported.
SITE_DIR="$work/site-ok" run_case site-external-transient 0 2 1 site
SITE_DIR="$work/site-ok" run_case site-external-persistent 1 4 always site
expect_out site-external-persistent 'github.com/example/repo/blob/main/x.md'
expect_out site-external-persistent 'site/src/content/docs/guides/x.md'
# The re-checks are lychee runs over the failed-URL list only, not over the
# docs/site inputs again.
if [ "$(grep -c 'docs/\*\*' "$work/site-external-persistent/args")" = 1 ]; then
  echo "PASS re-checks do not re-run the full inputs"
else
  echo "FAIL full inputs re-run during re-check:"; sed 's/^/    /' "$work/site-external-persistent/args"
  fails=$((fails + 1))
fi

if grep -q 'github.com/example/repo/blob/main/x.md' "$work/persistent/out"; then
  echo "PASS persistent reports failing link"
else
  echo "FAIL persistent output lacks failing link"
  fails=$((fails + 1))
fi

[ "$fails" -eq 0 ] || { echo "$fails test(s) failed"; exit 1; }
echo "all check-links tests passed"
