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

GCALCLI="${GCALCLI_BIN:-gcalcli}"
EMAILS="${CAL_EMAILS:?missing CAL_EMAILS}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

IFS=',' read -ra EMAIL_LIST <<< "$EMAILS"
for email in "${EMAIL_LIST[@]}"; do
  trimmed="${email// /}"
  echo "=== $trimmed ==="
  "$GCALCLI" --calendar "$trimmed" agenda "$START" "$END" \
    --details all || \
    echo "(no access or no events for $trimmed)"
done
