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

GAM="${GAM_BIN:-gam}"
USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
EMAILS="${CAL_EMAILS:?missing CAL_EMAILS}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

# gam's calendar info verb returns free/busy windows for a user
# given a time range. Iterate attendees.
IFS=',' read -ra EMAIL_LIST <<< "$EMAILS"
for email in "${EMAIL_LIST[@]}"; do
  trimmed="${email// /}"
  echo "=== $trimmed ==="
  "$GAM" user "$USER" show calendar "$trimmed" \
    freebusy timemin "$START" timemax "$END" || \
    echo "(no admin access or no events for $trimmed)"
done
