#!/usr/bin/env bash
# Calendar free-busy via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_EMAILS (comma-separated), CAL_START, CAL_END
#
# gcalcli does not expose Google Calendar's freeBusy API directly.
# This shells out to `gcalcli agenda` per attendee email, treating
# each as a calendar name. Works when the authenticated user has
# read access to each attendee's calendar (org-shared or explicitly
# delegated). For arbitrary external attendees, the result will be
# empty — use the `gam` backend with admin scope or the upcoming
# CalDAV backend for cross-domain free/busy.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "free-busy"
: "${BIN:?aps_init_backend did not set BIN}"

EMAILS="${CAL_EMAILS:?missing CAL_EMAILS}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

IFS=',' read -ra EMAIL_LIST <<< "$EMAILS"
failures=0
for email in "${EMAIL_LIST[@]}"; do
  trimmed="$(aps_trim_attendee "$email")"
  echo "=== $trimmed ==="
  if ! "$BIN" --calendar "$trimmed" agenda "$START" "$END" --details all; then
    echo "free-busy: gcalcli failed for $trimmed (no access, network error, or auth expired)" >&2
    failures=$((failures + 1))
  fi
done

# Partial-failure signal: 0 = all queries succeeded, 2 = at least one
# attendee was unreachable, 1 = all unreachable. Callers parsing
# stdout should re-check stderr for the failing addresses.
if [ "$failures" -gt 0 ]; then
  if [ "$failures" -eq "${#EMAIL_LIST[@]}" ]; then
    exit 1
  fi
  exit 2
fi
