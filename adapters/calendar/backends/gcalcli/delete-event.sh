#!/usr/bin/env bash
# Calendar delete-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_SEND_NOTIFICATIONS
#
# gcalcli matches events by title-search rather than opaque event id.
# To prevent silent wrong-event deletion (a typo in CAL_EVENT_ID would
# otherwise quietly delete whichever event happens to be the first
# title match), this script pre-queries with `gcalcli search` and
# refuses unless the query matches exactly one event. For id-precise
# deletion against a known Google event id, use the gam backend.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "delete-event"

CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
SEND="${CAL_SEND_NOTIFICATIONS:-true}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

# Pre-query: count matches. `gcalcli search` emits one event per
# non-blank line; count those to refuse 0-match (typo) and N-match
# (ambiguous) cases.
matches=$("$BIN" "${CAL_FLAG[@]}" search "$EVENT_ID" 2>/dev/null \
  | grep -cE '^[[:space:]]*[A-Z][a-z]{2}\b' || true)

if [ "$matches" -eq 0 ]; then
  echo "delete-event: no event matches '$EVENT_ID' on calendar '$CALENDAR'" >&2
  # Exit 65 = EX_DATAERR per BSD sysexits — input was syntactically
  # valid but did not resolve to a real event.
  exit 65
fi
if [ "$matches" -gt 1 ]; then
  echo "delete-event: '$EVENT_ID' matches $matches events on calendar '$CALENDAR'; refusing ambiguous delete (use the gam backend for id-precise deletion)" >&2
  exit 65
fi

ARGS=(delete "$EVENT_ID")
[ "$SEND" = "false" ] && ARGS+=(--nonotifications)

"$BIN" "${CAL_FLAG[@]}" "${ARGS[@]}"
