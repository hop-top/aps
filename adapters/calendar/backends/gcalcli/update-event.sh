#!/usr/bin/env bash
# Calendar update-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, plus any of
#              CAL_SUMMARY, CAL_START, CAL_END, CAL_ALL_DAY,
#              CAL_TRANSPARENCY, CAL_ATTENDEES, CAL_LOCATION,
#              CAL_RECURRENCE, CAL_DESCRIPTION
#
# gcalcli's `edit` is interactive — it walks fields one by one and
# expects a TTY. To run non-interactively we invoke it with a
# pre-cooked field-set via stdin. Fields not passed retain prior
# values. The event is located by SUMMARY-match search on EVENT_ID
# since gcalcli has no first-class event-id lookup. For exact-id
# updates against a known event, use the gam backend.
set -euo pipefail

GCALCLI="${GCALCLI_BIN:-gcalcli}"
CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

# Build the interactive edit response stream.
# gcalcli edit prompts: Title, Location, Description, When, Duration,
# Reminder, Color — empty line keeps current.
INPUT=""
INPUT+="${CAL_SUMMARY:-}\n"
INPUT+="${CAL_LOCATION:-}\n"
INPUT+="${CAL_DESCRIPTION:-}\n"
INPUT+="${CAL_START:-}\n"
INPUT+="${CAL_END:-}\n"
INPUT+="\n\n"

printf "%b" "$INPUT" | \
  "$GCALCLI" "${CAL_FLAG[@]}" edit "$EVENT_ID"
