#!/usr/bin/env bash
# Regenerate the committed `.12fc.json` conformance badge from real
# measurements only.
#
# The badge on the default branch is the source of truth. It is
# rebuilt from:
#
#   - verify-no-leak + verify-stories, executed here against
#     e2e/stories/                              -> F1, F2, F9, F12
#   - e2e/conformance/verdicts/*.json written
#     by `make 12fcc-grade`                     -> F1, F3-F8, F10, F11
#
# A factor with no measurement stays `skip` and is NOT counted as a
# pass: the badge reports measured factors only, so an unmeasured
# factor can never inflate the score. A failing leaf or verdict marks
# its factors `fail`; an ungradable verdict aborts (measurement itself
# broke — re-run `make 12fcc-grade`). Verdict and colour rules come
# from `kit conformance badge`, never from this script.
#
# Requirements:
#   - KIT_BIN: path to a kit binary shipping the `conformance` command
#     group (build from hop.top/kit cmd/kit). Defaults to `kit`.
#
# Usage (after `make 12fcc-record` + `make 12fcc-grade`):
#   KIT_BIN=/path/to/kit make 12fcc-badge
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONF="$REPO_ROOT/e2e/conformance"
KIT_BIN="${KIT_BIN:-kit}"
STORY_PATHS="e2e/stories"

# The probe requires real help output (not just exit 0) so a wrong
# binary can never masquerade as a conformance-capable kit. Captured
# into a variable rather than piped: `grep -q` under pipefail would
# SIGPIPE the probe on success.
probe="$("$KIT_BIN" conformance badge --help 2>/dev/null || true)"
case "$probe" in
  *--matrix*) ;;
  *)
    echo "ERROR: '$KIT_BIN' does not ship 'conformance badge'. Set KIT_BIN to a kit binary built from hop.top/kit cmd/kit." >&2
    exit 1
    ;;
esac

TMP="$(mktemp -d "${TMPDIR:-/tmp}/aps-12fcc-badge.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

# Run a verify leaf and map kit's exit contract (0 clean / 2 findings)
# to a factor status. Any other exit means measurement broke: abort
# rather than guess.
run_leaf() {
  local leaf="$1" rc
  set +e
  (cd "$REPO_ROOT" && "$KIT_BIN" conformance "$leaf" \
    --paths="$STORY_PATHS" --format=json) >"$TMP/$leaf.json" 2>&1
  rc=$?
  set -e
  case "$rc" in
    0) echo pass ;;
    2) echo fail ;;
    *)
      echo "ERROR: '$KIT_BIN conformance $leaf' exited $rc (want 0 or 2):" >&2
      cat "$TMP/$leaf.json" >&2
      exit 1
      ;;
  esac
}

echo "==> verify-no-leak ($STORY_PATHS)"
NO_LEAK_STATUS="$(run_leaf verify-no-leak)"
echo "    $NO_LEAK_STATUS"

echo "==> verify-stories ($STORY_PATHS)"
STORIES_STATUS="$(run_leaf verify-stories)"
echo "    $STORIES_STATUS"

echo "==> assembling per-factor matrix from verdicts/"
python3 - "$CONF/verdicts" "$TMP/matrix.json" "$NO_LEAK_STATUS" "$STORIES_STATUS" <<'PYEOF'
import json, os, sys

vdir, out, no_leak, stories = sys.argv[1:5]

FACTORS = [
    (1,  "Capability Introspection", "must"),
    (2,  "Intent Clarity",           "must"),
    (3,  "Structured I/O",           "must"),
    (4,  "Corrective Error Model",   "must"),
    (5,  "Explicit Contracts",       "must"),
    (6,  "Previewability",           "must"),
    (7,  "Idempotency",              "must"),
    (8,  "State Transparency",       "must"),
    (9,  "Contextual Guidance",      "should"),
    (10, "Delegation Safety",        "must"),
    (11, "Exit Code Semantics",      "must"),
    (12, "Evolution Guarantees",     "must"),
]

status = {n: "skip" for n, _, _ in FACTORS}
evidence = {n: "" for n, _, _ in FACTORS}

# Story-graded factors.
for n in (1, 2, 9, 12):
    status[n] = stories
    evidence[n] = "verify-stories " + ("clean" if stories == "pass" else "reported findings")
status[10] = no_leak
evidence[10] = "verify-no-leak " + ("clean" if no_leak == "pass" else "reported findings")

names = sorted(f for f in os.listdir(vdir) if f.endswith(".json")) if os.path.isdir(vdir) else []
if not names:
    sys.exit(f"ERROR: no verdicts in {vdir}; run `make 12fcc-grade` first")

for name in names:
    sid = name[: -len(".json")]
    with open(os.path.join(vdir, name)) as f:
        v = json.load(f)
    for facet in v.get("facets", []):
        n, s = facet.get("factor"), facet.get("status")
        if n not in status:
            sys.exit(f"ERROR: {name}: facet for unknown factor {n!r}")
        if s not in ("pass", "fail"):
            sys.exit(
                f"ERROR: {name}: F{n} is {s!r} — measurement broke; re-run `make 12fcc-grade`"
            )
        if s == "fail":
            status[n] = "fail"
            evidence[n] = f"tier-3 verdict fail ({sid})"
        elif status[n] == "skip":
            status[n] = "pass"
            evidence[n] = "tier-3 cassette verdicts"

matrix = {
    "schemaVersion": 1,
    "factors": [
        {"n": n, "name": name, "tier": tier,
         "status": status[n], "evidence": evidence[n]}
        for n, name, tier in FACTORS
    ],
}
with open(out, "w") as f:
    json.dump(matrix, f, indent=2)

for n, name, tier in FACTORS:
    print(f"    F{n:<3} {status[n]:<5} {name}")
PYEOF

echo "==> rendering badge (.12fc.json)"
(cd "$REPO_ROOT" && "$KIT_BIN" conformance badge \
  --matrix="$TMP/matrix.json" --output=".12fc.json" >/dev/null)
python3 - "$REPO_ROOT/.12fc.json" <<'PYEOF'
import json, sys
with open(sys.argv[1]) as f:
    b = json.load(f)
print(f"    {b['label']}: {b['message']} ({b['color']})")
PYEOF
