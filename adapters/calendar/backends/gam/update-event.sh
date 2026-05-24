#!/usr/bin/env bash
# Calendar update-event via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, plus any of
#              CAL_SUMMARY, CAL_START, CAL_END, CAL_ALL_DAY,
#              CAL_TRANSPARENCY, CAL_ATTENDEES, CAL_LOCATION,
#              CAL_RECURRENCE, CAL_DESCRIPTION
#
# Backend difference vs gcalcli: gam updates by exact event id
# (not title search), so CAL_EVENT_ID must be a real Google event
# id obtained from list-events. Only fields explicitly passed
# are sent in the patch.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "update-event"

USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

ARGS=(updateevent "$EVENT_ID")

[ -n "${CAL_SUMMARY:-}" ] && ARGS+=(summary "$CAL_SUMMARY")
[ -n "${CAL_LOCATION:-}" ] && ARGS+=(location "$CAL_LOCATION")
[ -n "${CAL_DESCRIPTION:-}" ] && ARGS+=(description "$CAL_DESCRIPTION")
[ -n "${CAL_RECURRENCE:-}" ] && ARGS+=(rrule "$CAL_RECURRENCE")
[ -n "${CAL_TRANSPARENCY:-}" ] && \
  ARGS+=(transparency "$CAL_TRANSPARENCY")

if [ -n "${CAL_START:-}" ] && [ -n "${CAL_END:-}" ]; then
  if [ "${CAL_ALL_DAY:-false}" = "true" ]; then
    ARGS+=(start allday "$CAL_START" end allday "$CAL_END")
  else
    ARGS+=(start time "$CAL_START" end time "$CAL_END")
  fi
fi

if [ -n "${CAL_ATTENDEES:-}" ]; then
  IFS=',' read -ra ATTS <<< "$CAL_ATTENDEES"
  for a in "${ATTS[@]}"; do
    ARGS+=(attendee "$(aps_trim_attendee "$a")")
  done
fi

"$BIN" calendar "$CALENDAR" "${ARGS[@]}"
