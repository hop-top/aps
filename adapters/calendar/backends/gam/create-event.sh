#!/usr/bin/env bash
# Calendar create-event via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): CAL_CALENDAR, CAL_SUMMARY, CAL_START, CAL_END,
#              CAL_ALL_DAY, CAL_TRANSPARENCY, CAL_ATTENDEES,
#              CAL_LOCATION, CAL_RECURRENCE, CAL_DESCRIPTION
#
# Backend difference vs gcalcli: gam writes the event with the
# admin service account's identity (organizer = USER but
# bookkeeping happens via the gam project). Notifications are
# sent by default; pass sendnotifications false in the calling
# script if you need silent inserts.
set -euo pipefail

GAM="${GAM_BIN:-gam}"
USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
SUMMARY="${CAL_SUMMARY:?missing CAL_SUMMARY}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

ARGS=(addevent summary "$SUMMARY")

if [ "${CAL_ALL_DAY:-false}" = "true" ]; then
  ARGS+=(start allday "$START" end allday "$END")
else
  ARGS+=(start time "$START" end time "$END")
fi

[ -n "${CAL_LOCATION:-}" ] && ARGS+=(location "$CAL_LOCATION")
[ -n "${CAL_DESCRIPTION:-}" ] && ARGS+=(description "$CAL_DESCRIPTION")
[ -n "${CAL_RECURRENCE:-}" ] && ARGS+=(rrule "$CAL_RECURRENCE")
[ -n "${CAL_TRANSPARENCY:-}" ] && \
  ARGS+=(transparency "$CAL_TRANSPARENCY")

if [ -n "${CAL_ATTENDEES:-}" ]; then
  IFS=',' read -ra ATTS <<< "$CAL_ATTENDEES"
  for a in "${ATTS[@]}"; do
    ARGS+=(attendee "${a// /}")
  done
fi

"$GAM" calendar "$CALENDAR" "${ARGS[@]}"
