#!/usr/bin/env bash
# Email reply via himalaya
# Env: APS_EMAIL_FROM, APS_EMAIL_ACCOUNT
# Input (env): EMAIL_ID, EMAIL_BODY
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "reply"
: "${BIN:?aps_init_backend did not set BIN}"

FROM="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
ID="${EMAIL_ID:?missing EMAIL_ID}"
BODY="${EMAIL_BODY:?missing EMAIL_BODY}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
# Both invocations below must carry it — building the template from
# one account and sending from another would silently misroute.
ACCOUNT_FLAG=()
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG=(-a "$ACCOUNT")

TEMPLATE=$("$BIN" template reply "$ID" \
  -H "From:$FROM" ${ACCOUNT_FLAG+"${ACCOUNT_FLAG[@]}"})

# Replace the empty body placeholder with actual body
# Template has the quoted original after a blank line
HEADER=$(echo "$TEMPLATE" | sed '/^$/q')
QUOTED=$(echo "$TEMPLATE" | sed '1,/^$/d')

"$BIN" template send ${ACCOUNT_FLAG+"${ACCOUNT_FLAG[@]}"} <<EOF
$HEADER

$BODY

$QUOTED
EOF
