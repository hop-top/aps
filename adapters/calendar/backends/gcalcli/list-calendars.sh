#!/usr/bin/env bash
# Calendar list-calendars via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): none
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "list-calendars"

"$BIN" list
