#!/usr/bin/env bash
# Calendar list-calendars via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): none
set -euo pipefail

GCALCLI="${GCALCLI_BIN:-gcalcli}"

"$GCALCLI" list
