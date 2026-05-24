#!/usr/bin/env bash
# Calendar list-events via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_START, CAL_END
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "list-events"
: "${BIN:?aps_init_backend did not set BIN}"

CALENDAR="${CAL_CALENDAR:-primary}"
START="${CAL_START:?missing CAL_START}"
END="${CAL_END:?missing CAL_END}"

CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

"$BIN" "${CAL_FLAG[@]}" agenda "$START" "$END" --details all
