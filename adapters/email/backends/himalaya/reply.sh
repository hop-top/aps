#!/usr/bin/env bash
# Email reply via himalaya
# Env: APS_EMAIL_FROM, APS_EMAIL_ACCOUNT
# Input (env): EMAIL_ID, EMAIL_BODY
set -euo pipefail

FROM="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
ID="${EMAIL_ID:?missing EMAIL_ID}"
BODY="${EMAIL_BODY:?missing EMAIL_BODY}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

TEMPLATE=$(himalaya template reply "$ID" \
  -H "From:$FROM" $ACCOUNT_FLAG)

# Split template at the first blank line: headers above, quoted
# original below. Done with bash parameter expansion (no pipe) so a
# multi-MB HTML alternative can't trigger SIGPIPE on an early-
# exiting consumer like `sed '/^$/q'`. See T-0332.
SEP=$'\n\n'
HEADER="${TEMPLATE%%"$SEP"*}"
QUOTED="${TEMPLATE#*"$SEP"}"
# If TEMPLATE has no blank line, %% / # leave both equal to TEMPLATE;
# fall back to "no quoted body" rather than duplicating headers.
[ "$QUOTED" = "$TEMPLATE" ] && QUOTED=""

himalaya template send $ACCOUNT_FLAG <<EOF
$HEADER

$BODY

$QUOTED
EOF
