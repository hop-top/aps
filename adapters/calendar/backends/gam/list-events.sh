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
set -euo pipefail

GAM="${GAM_BIN:-gam}"
USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
CALENDAR="${CAL_CALENDAR:-primary}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

if [ "$CALENDAR" = "primary" ]; then
  CALENDAR="$USER"
fi

"$GAM" calendar "$CALENDAR" showevents \
  timemin "$START" timemax "$END"
