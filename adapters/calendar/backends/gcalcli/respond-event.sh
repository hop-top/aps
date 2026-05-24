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

EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
RESPONSE="${CAL_RESPONSE:?missing CAL_RESPONSE}"

case "$RESPONSE" in
  accepted|declined|tentative) ;;
  *)
    echo "respond-event: invalid response '$RESPONSE'" \
      "(expected accepted|declined|tentative)" >&2
    exit 2
    ;;
esac

echo "respond-event: not supported by gcalcli; switch backend to" \
  "gam or use the Google Calendar web UI (event $EVENT_ID)" >&2
exit 2
