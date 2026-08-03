#!/usr/bin/env bash
# Calendar delete-event via gcalcli
# Env: APS_EMAIL_ACCOUNT (optional override)
# Input (env): CAL_CALENDAR, CAL_EVENT_ID, CAL_SEND_NOTIFICATIONS
#
# gcalcli matches events by title-search rather than opaque event id.
# To prevent silent wrong-event deletion (a typo in CAL_EVENT_ID would
# otherwise quietly delete whichever event happens to be the first
# title match), this script pre-queries with `gcalcli search` and
# refuses unless the query matches exactly one event. For id-precise
# deletion against a known Google event id, use the gam backend.
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "delete-event"
: "${BIN:?aps_init_backend did not set BIN}"

CALENDAR="${CAL_CALENDAR:-primary}"
EVENT_ID="${CAL_EVENT_ID:?missing CAL_EVENT_ID}"
SEND="${CAL_SEND_NOTIFICATIONS:-true}"

# Expanded as ${CAL_FLAG+...} at each use: bash 3.2 (the system
# bash on macOS) treats "${EMPTY[@]}" as an unbound variable under
# `set -u`, so the default primary-calendar path — which leaves this
# array empty — aborted before reaching gcalcli.
CAL_FLAG=()
[ "$CALENDAR" != "primary" ] && CAL_FLAG=(--calendar "$CALENDAR")

# Pre-query in two steps so a gcalcli failure (expired OAuth, network
# error, calendar permission revoked) surfaces as its own error
# instead of being swallowed into a misleading "no event matches"
# diagnosis. The previous one-shot pipeline used `| grep -cE ... || true`
# which converted any tool failure into matches=0 → exit 65.
search_output=""
if ! search_output="$("$BIN" ${CAL_FLAG+"${CAL_FLAG[@]}"} search "$EVENT_ID" 2>&1)"; then
  echo "delete-event: gcalcli search failed for '$EVENT_ID' on calendar '$CALENDAR':" >&2
  printf '%s\n' "$search_output" >&2
  # Forward gcalcli's exit class via 1 (general failure) so the caller
  # can tell auth/network errors apart from 65 (data-error / wrong id).
  exit 1
fi

# `gcalcli search` emits one event per non-blank line beginning with a
# weekday name (e.g., "Mon May 22 ..."). Count those to refuse 0-match
# (typo) and N-match (ambiguous) cases.
matches=$(printf '%s\n' "$search_output" \
  | grep -cE '^[[:space:]]*[A-Z][a-z]{2}\b' || true)

if [ "$matches" -eq 0 ]; then
  echo "delete-event: no event matches '$EVENT_ID' on calendar '$CALENDAR'" >&2
  # Exit 65 = EX_DATAERR per BSD sysexits — input was syntactically
  # valid but did not resolve to a real event.
  exit 65
fi
if [ "$matches" -gt 1 ]; then
  echo "delete-event: '$EVENT_ID' matches $matches events on calendar '$CALENDAR'; refusing ambiguous delete (use the gam backend for id-precise deletion)" >&2
  exit 65
fi

ARGS=(delete "$EVENT_ID")
[ "$SEND" = "false" ] && ARGS+=(--nonotifications)

"$BIN" ${CAL_FLAG+"${CAL_FLAG[@]}"} "${ARGS[@]}"
