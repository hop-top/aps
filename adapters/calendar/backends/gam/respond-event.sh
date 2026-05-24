#!/usr/bin/env bash
# Calendar respond-event via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user, attendee whose status changes)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_RESPONSE, CAL_COMMENT
#
# Backend difference vs gcalcli: gam can patch the attendee's
# responseStatus on the event because it acts admin-scoped on behalf
# of USER. CAL_EVENT_ID must be the Google event id.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "respond-event"

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

# Pre-check that USER is on the event's attendee list. gam's
# updateevent silently creates a new attendee entry if the email
# isn't already present, which is rarely what "respond" implies.
# Detect by listing the event's attendees and grepping for USER.
attendees=$("$BIN" calendar "$CALENDAR" info event "$EVENT_ID" 2>/dev/null \
  | grep -iE '^[[:space:]]*Attendees?:|^[[:space:]]*email:' || true)

if ! printf '%s\n' "$attendees" | grep -qiF "$USER"; then
  echo "respond-event: $USER is not an attendee on event $EVENT_ID (calendar $CALENDAR)" >&2
  # Exit 65 = EX_DATAERR — input was syntactically valid but did not
  # match a real attendee relationship.
  exit 65
fi

ARGS=(updateevent "$EVENT_ID"
  attendee "$USER" responsestatus "$RESPONSE")
[ -n "${CAL_COMMENT:-}" ] && ARGS+=(comment "$CAL_COMMENT")

"$BIN" calendar "$CALENDAR" "${ARGS[@]}"
