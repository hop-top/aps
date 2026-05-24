#!/usr/bin/env bash
# Calendar update-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID
#
# UNSUPPORTED: gcalcli's `edit` is an interactive TTY walk — it
# prompts the user field by field, and the prompt order has shifted
# between gcalcli versions. The previous implementation piped fields
# via stdin, which silently wrote values to whichever field slot was
# next in the current build's prompt order: a corrupted event with
# exit 0 was the typical failure mode.
#
# Use the gam backend for non-interactive updates; gam patches by
# event id without an interactive prompt.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "update-event"

EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"

echo "update-event: not supported by gcalcli; switch backend to" \
  "gam for non-interactive field updates (event $EVENT_ID)" >&2
# Exit 2 = "backend cannot perform this action"; distinct from 64
# (bad input) and 127 (gcalcli not installed).
exit 2
