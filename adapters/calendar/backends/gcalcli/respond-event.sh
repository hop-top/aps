#!/usr/bin/env bash
# Calendar respond-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_RESPONSE, CAL_COMMENT
#
# UNSUPPORTED: gcalcli has no verb to respond to an invitation
# (accept/decline/tentative). Google Calendar's responseStatus field
# requires a direct Events.patch API call against the attendee
# sub-object, which gcalcli does not expose. Use the gam backend
# (admin-scoped) or respond via the Google Calendar web UI.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(dirname "$0")/../../../_lib.sh"
aps_init_backend "respond-event"

EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
RESPONSE="${CAL_RESPONSE:?missing CAL_RESPONSE}"

case "$RESPONSE" in
  accepted|declined|tentative) ;;
  *)
    echo "respond-event: invalid response '$RESPONSE'" \
      "(expected accepted|declined|tentative)" >&2
    # EX_USAGE per BSD sysexits — caller passed an invalid argument.
    exit 64
    ;;
esac

echo "respond-event: not supported by gcalcli; switch backend to" \
  "gam or use the Google Calendar web UI (event $EVENT_ID)" >&2
# Exit 2 = "backend cannot perform this action"; distinct from 64
# (bad input) and 127 (gcalcli not installed).
exit 2
