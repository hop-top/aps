#!/usr/bin/env bash
# Calendar list-events via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_START, CAL_END
set -euo pipefail

GCALCLI="${GCALCLI_BIN:-gcalcli}"
CALENDAR="${CAL_CALENDAR:-primary}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

"$GCALCLI" "${CAL_FLAG[@]}" agenda "$START" "$END" --details all
