#!/usr/bin/env bash
# Calendar respond-event via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user, attendee whose status changes)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_RESPONSE, CAL_COMMENT
#
# Backend difference vs gcalcli: gam can patch the attendee's
# responseStatus on the event because it acts admin-scoped on behalf
# of USER. CAL_EVENT_ID must be the Google event id.
set -euo pipefail

GAM="${GAM_BIN:-gam}"
USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
RESPONSE="${CAL_RESPONSE:?missing CAL_RESPONSE}"

case "$RESPONSE" in
  accepted|declined|tentative) ;;
  *)
    echo "respond-event: invalid response '$RESPONSE'" \
      "(expected accepted|declined|tentative)" >&2
    # EX_USAGE per BSD sysexits — caller passed an invalid argument.
    exit 64
    ;;
esac

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

ARGS=(updateevent "$EVENT_ID"
  attendee "$USER" responsestatus "$RESPONSE")
[ -n "${CAL_COMMENT:-}" ] && ARGS+=(comment "$CAL_COMMENT")

"$GAM" calendar "$CALENDAR" "${ARGS[@]}"
