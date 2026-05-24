#!/usr/bin/env bash
# Calendar list-calendars via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): none
#
# Backend difference vs gcalcli: gam runs admin-scoped against the
# workspace OAuth client, so it lists calendars the gam project has
# domain-wide delegation to read for the target user — not just
# calendars the user themselves subscribes to.
#
# Output: `formatjson` emits newline-delimited JSON. gam's default
# human-formatted text would silently produce garbage when piped
# through jq downstream.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "list-calendars"

USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"

"$BIN" user "$USER" show calendars formatjson
