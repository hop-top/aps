#!/usr/bin/env bash
# Calendar delete-event via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_SEND_NOTIFICATIONS
#
# Backend difference vs gcalcli: gam deletes by exact event id.
# CAL_EVENT_ID must be the Google event id (from list-events).
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "delete-event"
: "${BIN:?aps_init_backend did not set BIN}"

USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
SEND="${CAL_SEND_NOTIFICATIONS:-true}"

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

ARGS=(deleteevent "$EVENT_ID" doit)
[ "$SEND" = "false" ] && ARGS+=(notifyattendees false)

"$BIN" calendar "$CALENDAR" "${ARGS[@]}"
