#!/usr/bin/env bash
# Calendar free-busy via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): CAL_EMAILS (comma-separated), CAL_START, CAL_END
#
# Backend difference vs gcalcli: gam free-busy is admin-scoped.
# It uses the gam project's domain-wide delegation, so attendee
# coverage equals the gam project's Calendar API scope across the
# workspace. For attendees outside the workspace the entry returns
# only the free/busy bands the external calendar exposes publicly.
# This is intentionally different from gcalcli, which is per-user.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "free-busy"

USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
EMAILS="${CAL_EMAILS:?missing CAL_EMAILS}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

# gam's calendar info verb returns free/busy windows for a user
# given a time range. Iterate attendees.
IFS=',' read -ra EMAIL_LIST <<< "$EMAILS"
failures=0
for email in "${EMAIL_LIST[@]}"; do
  trimmed="$(aps_trim_attendee "$email")"
  echo "=== $trimmed ==="
  if ! "$BIN" user "$USER" show calendar "$trimmed" freebusy timemin "$START" timemax "$END"; then
    echo "free-busy: gam failed for $trimmed (no admin access, network error, or DWD scope missing)" >&2
    failures=$((failures + 1))
  fi
done

if [ "$failures" -gt 0 ]; then
  if [ "$failures" -eq "${#EMAIL_LIST[@]}" ]; then
    exit 1
  fi
  exit 2
fi
