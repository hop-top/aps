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
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "respond-event"
: "${BIN:?aps_init_backend did not set BIN}"

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
# Capture-then-check so a gam failure (no DWD scope, event doesn't
# exist, network error) surfaces with the real error instead of
# being swallowed into a misleading "not an attendee" diagnosis.
event_info=""
if ! event_info="$("$BIN" calendar "$CALENDAR" info event "$EVENT_ID" 2>&1)"; then
  echo "respond-event: gam info event failed for '$EVENT_ID' on calendar '$CALENDAR':" >&2
  printf '%s\n' "$event_info" >&2
  exit 1
fi

# gam emits attendee data on lines like `Attendees:` (header) followed
# by per-attendee blocks with `email:` (lowercase) keys. The header
# match is loose (Attendees / Attendee, case-insensitive) so minor
# format drift between gam releases doesn't silently break the check.
attendees=$(printf '%s\n' "$event_info" \
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
