#!/usr/bin/env bash
# Record 12fcc conformance cassettes from real aps runs.
#
# For every scenario under e2e/conformance/scenarios/aps/, drives
# `kit conformance harness record` against a freshly built binary:
# each step executes as a real subprocess; exit code, stdout, stderr,
# and duration are captured verbatim into the svc upload layout the
# grading service consumes:
#
#   e2e/conformance/cassettes/<scenario-id>/
#     manifest.yaml
#     story.yaml
#     steps/<step-id>/{result.json,stdout.txt,stderr.txt}
#
# Captures are never edited after the fact; re-running re-records
# everything from scratch. The recorder refuses to proceed when the
# story bytes do not hash to the scenario's declared content_hash.
#
# Isolation is the whole game here: aps manages profiles, secrets, and
# bus credentials, so a cassette recorded against a developer's real
# environment would bake personal data into a committed artifact. Every
# scenario therefore runs with:
#
#   - its own HOME and APS_DATA_PATH under a throwaway work dir, so
#     profile writes never touch the developer's store;
#   - every XDG_* dir redirected inside that work dir;
#   - a scrubbed environment: `env -i` starts from nothing and only an
#     explicit allowlist is passed back in, so APS_*, BUS_TOKEN, and
#     any ambient credential cannot reach the recorded process;
#   - the bus disabled, so no network call is attempted.
#
# scripts/12fcc-scan-cassettes.sh re-checks the committed result for
# leaked secrets and personal paths; run it after every re-record.
#
# Requirements:
#   - KIT_BIN: path to a kit binary shipping `conformance harness
#     record` (build from hop.top/kit cmd/kit). Defaults to `kit`.
#
# Usage: [KIT_BIN=/path/to/kit] scripts/12fcc-record.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONF="$REPO_ROOT/e2e/conformance"
BIN="$REPO_ROOT/bin/aps-12fcc"
KIT_BIN="${KIT_BIN:-kit}"
# Pinned, machine-independent work root. `config paths` echoes the
# absolute paths it probed, so a $TMPDIR-derived root would bake this
# machine's temp directory into the captures and churn the diff on
# every re-record elsewhere. A fixed path under /tmp keeps captures
# byte-stable across machines.
WORK="${APS_12FCC_WORK:-/tmp/aps-12fcc-record}"

# The probe requires real help output (not just exit 0) so a wrong
# binary can never masquerade as a conformance-capable kit.
probe="$("$KIT_BIN" conformance harness record --help 2>/dev/null || true)"
case "$probe" in
  *--scenario*) ;;
  *)
    echo "ERROR: '$KIT_BIN' does not ship 'conformance harness record'. Set KIT_BIN to a kit binary built from hop.top/kit cmd/kit." >&2
    exit 1
    ;;
esac

echo "==> building $BIN"
mkdir -p "$(dirname "$BIN")"
(cd "$REPO_ROOT" && go build -buildvcs=false -o "$BIN" ./cmd/aps)
BINARY_VERSION="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo dev)"

rm -rf "$WORK"
mkdir -p "$WORK"

for scenario in "$CONF"/scenarios/aps/*/*/scenario.yaml; do
  sid="$(basename "$(dirname "$(dirname "$scenario")")")"
  echo "==> recording $sid"
  proj="$WORK/$sid"
  mkdir -p "$proj/home" "$proj/data" "$proj/xdg/config" "$proj/xdg/data" \
           "$proj/xdg/cache" "$proj/xdg/state"
  rm -rf "$CONF/cassettes/$sid"

  # `env -i` drops the ambient environment entirely; only the variables
  # named below reach the recorder and, through it, aps. Nothing from
  # the developer's shell (APS_*, BUS_TOKEN, credentials, personal
  # paths) survives into a capture.
  env -i \
    PATH="/usr/bin:/bin:/usr/sbin:/sbin" \
    HOME="$proj/home" \
    TMPDIR="$proj/tmp" \
    APS_DATA_PATH="$proj/data" \
    XDG_CONFIG_HOME="$proj/xdg/config" \
    XDG_DATA_HOME="$proj/xdg/data" \
    XDG_CACHE_HOME="$proj/xdg/cache" \
    XDG_STATE_HOME="$proj/xdg/state" \
    NO_COLOR=1 \
    "$KIT_BIN" conformance harness record \
      --scenario "$scenario" \
      --binary "$BIN" \
      --binary-version "$BINARY_VERSION" \
      --out "$CONF/cassettes/$sid" \
      --workdir "$proj" \
      --no-hints >/dev/null
done

echo "==> cassettes written to $CONF/cassettes"
echo "==> run scripts/12fcc-scan-cassettes.sh before committing"
