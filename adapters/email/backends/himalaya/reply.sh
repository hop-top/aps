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

# Replace the empty body placeholder with actual body
# Template has the quoted original after a blank line
HEADER=$(echo "$TEMPLATE" | sed '/^$/q')
QUOTED=$(echo "$TEMPLATE" | sed '1,/^$/d')

himalaya template send $ACCOUNT_FLAG <<EOF
$HEADER

$BODY

$QUOTED
EOF
