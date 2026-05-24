#!/usr/bin/env bash
# Calendar list-calendars via gam (Google Workspace admin)
# Env: APS_EMAIL_FROM (profile user)
# Input (env): none
#
# Backend difference vs gcalcli: gam runs admin-scoped against the
# workspace OAuth client, so it lists calendars the gam project has
# domain-wide delegation to read for the target user — not just
# calendars the user themselves subscribes to.
set -euo pipefail

GAM="${GAM_BIN:-gam}"
USER="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"

"$GAM" user "$USER" show calendars
