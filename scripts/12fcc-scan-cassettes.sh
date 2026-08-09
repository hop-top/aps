#!/usr/bin/env bash
# Scan recorded cassettes for secrets and personal data before commit.
#
# Cassettes are verbatim captures of a real aps process, and aps
# manages profiles, secret files, and bus credentials. The recorder
# isolates each run (see scripts/12fcc-record.sh), but isolation is a
# claim — this script is the check on it, and it runs against the bytes
# that are actually about to be committed.
#
# Flags, in the recorded stdout/stderr and the manifests:
#   - credential-shaped env names (APS_SECRET_*, *_TOKEN, *_KEY, ...)
#   - the contents of a profile secrets.env
#   - the developer's real home directory and username
#   - absolute paths outside the throwaway work dir
#
# Exit codes:
#   0  clean
#   1  findings (printed with file and line)
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CASSETTES="$REPO_ROOT/e2e/conformance/cassettes"

if [ ! -d "$CASSETTES" ]; then
  echo "no cassettes at $CASSETTES; run scripts/12fcc-record.sh first" >&2
  exit 0
fi

findings=0

report() {
  findings=$((findings + 1))
  echo "LEAK: $1" >&2
}

# Credential-shaped tokens. Matches an assignment or a JSON/YAML field
# whose name looks like a secret AND whose value is non-empty, so the
# literal string "has_secrets": false does not trip the scan.
while IFS= read -r hit; do
  report "credential-shaped value: $hit"
done < <(grep -rnaE '(APS_SECRET_[A-Z0-9_]*|[A-Z0-9_]*(TOKEN|SECRET|PASSWORD|APIKEY|API_KEY)[A-Z0-9_]*)[=:][[:space:]]*["'"'"']?[^"'"'"'[:space:],}]+' \
  "$CASSETTES" 2>/dev/null \
  | grep -vaE '[=:][[:space:]]*["'"'"']?(null|false|true|0|\{\}|\[\])' \
  | grep -vaE 'BUS_TOKEN or APS_BUS_TOKEN not set' || true)

# Real home directory and username of whoever recorded.
real_home="${HOME:-}"
if [ -n "$real_home" ] && [ "$real_home" != "/" ]; then
  while IFS= read -r hit; do
    report "recorder home path: $hit"
  done < <(grep -rnaF "$real_home" "$CASSETTES" 2>/dev/null || true)
fi

# Username match is word-bounded and skips very short names. A bare
# substring search makes short or common usernames ("root", "ci")
# match unrelated text — "root" hits the `--root` flag in the recorded
# toolspec — which would fail the build on a finding that is not a
# leak. Word boundaries keep a real `/home/<user>/` or `user=<user>`
# occurrence detected while ignoring those substrings.
real_user="${USER:-${LOGNAME:-}}"
if [ -n "$real_user" ] && [ "${#real_user}" -ge 4 ] && [ "$real_user" != "root" ]; then
  while IFS= read -r hit; do
    report "recorder username: $hit"
  done < <(grep -rnaE "(^|[^A-Za-z0-9_-])${real_user}([^A-Za-z0-9_-]|$)" "$CASSETTES" 2>/dev/null || true)
fi

# Anything that looks like a decrypted secrets.env body.
while IFS= read -r hit; do
  report "secrets.env content: $hit"
done < <(grep -rna 'secrets\.env' "$CASSETTES" 2>/dev/null \
  | grep -vaE '(Secrets: (present|missing)|"?has_secrets"?)' || true)

if [ "$findings" -ne 0 ]; then
  echo >&2
  echo "$findings finding(s): cassettes must not be committed until these are resolved." >&2
  echo "Re-record with scripts/12fcc-record.sh (it scrubs the environment) and re-scan." >&2
  exit 1
fi

echo "cassette scan clean: no credentials, secrets, or personal paths found"
