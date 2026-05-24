#!/usr/bin/env bash
# Calendar list-calendars via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): none
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "list-calendars"
: "${BIN:?aps_init_backend did not set BIN}"

"$BIN" list
