#!/usr/bin/env bash
# Calendar create-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_SUMMARY, CAL_START, CAL_END,
#              CAL_ALL_DAY, CAL_TRANSPARENCY, CAL_ATTENDEES,
#              CAL_LOCATION, CAL_RECURRENCE, CAL_DESCRIPTION
set -euo pipefail

GCALCLI="${GCALCLI_BIN:-gcalcli}"
CALENDAR="${CAL_CALENDAR:-primary}"
SUMMARY="${CAL_SUMMARY:?missing CAL_SUMMARY}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

ARGS=(add --title "$SUMMARY" --when "$START" --duration_end "$END")

[ -n "${CAL_LOCATION:-}" ] && ARGS+=(--where "$CAL_LOCATION")
[ -n "${CAL_DESCRIPTION:-}" ] && ARGS+=(--description "$CAL_DESCRIPTION")
[ -n "${CAL_RECURRENCE:-}" ] && ARGS+=(--rrule "$CAL_RECURRENCE")
[ "${CAL_ALL_DAY:-false}" = "true" ] && ARGS+=(--allday)
[ "${CAL_TRANSPARENCY:-opaque}" = "transparent" ] && \
  ARGS+=(--transparency transparent)

if [ -n "${CAL_ATTENDEES:-}" ]; then
  IFS=',' read -ra ATTS <<< "$CAL_ATTENDEES"
  for a in "${ATTS[@]}"; do
    ARGS+=(--email "${a// /}")
  done
fi

"$GCALCLI" "${CAL_FLAG[@]}" "${ARGS[@]}"
