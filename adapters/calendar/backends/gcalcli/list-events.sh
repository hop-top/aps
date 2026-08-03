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

# Expanded as ${CAL_FLAG+...} at each use: bash 3.2 (the system
# bash on macOS) treats "${EMPTY[@]}" as an unbound variable under
# `set -u`, so the default primary-calendar path — which leaves this
# array empty — aborted before reaching gcalcli.
CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

"$BIN" ${CAL_FLAG+"${CAL_FLAG[@]}"} agenda "$START" "$END" --details all
