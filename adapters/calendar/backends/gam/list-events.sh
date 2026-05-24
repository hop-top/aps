#!/usr/bin/env bash
# Calendar list-events via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): CAL_CALENDAR, CAL_START, CAL_END
#
# Backend difference vs gcalcli: when CAL_CALENDAR is "primary",
# gam resolves it to the profile user's primary calendar id;
# otherwise CAL_CALENDAR is taken as a calendar id (e.g.
# "team-x@group.calendar.google.com") that the gam admin project
# has access to.
#
# Output: `formatjson` emits newline-delimited JSON. gam's default
# human-formatted text would silently produce garbage when piped
# through jq downstream.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "list-events"
: "${BIN:?aps_init_backend did not set BIN}"

USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

"$BIN" calendar "$CALENDAR" showevents \
  timemin "$START" timemax "$END" formatjson
