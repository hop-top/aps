#!/usr/bin/env bash
# Calendar delete-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_SEND_NOTIFICATIONS
#
# gcalcli matches events by title-search rather than opaque event id.
# Passing CAL_EVENT_ID literally means: search for an event whose
# title or generated id matches that string and delete the first hit.
# For id-precise deletion against a known Google event id, use the
# gam backend.
set -euo pipefail

GCALCLI="${GCALCLI_BIN:-gcalcli}"
CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
SEND="${CAL_SEND_NOTIFICATIONS:-true}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

ARGS=(delete "$EVENT_ID")
[ "$SEND" = "false" ] && ARGS+=(--nonotifications)

"$GCALCLI" "${CAL_FLAG[@]}" "${ARGS[@]}"
