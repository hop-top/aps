#!/usr/bin/env bash
# Email send via himalaya
# Env: APS_EMAIL_FROM, APS_EMAIL_ACCOUNT
# Input (env): EMAIL_TO, EMAIL_SUBJECT, EMAIL_BODY, EMAIL_CC
set -euo pipefail

FROM="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
TO="${EMAIL_TO:?missing EMAIL_TO}"
SUBJECT="${EMAIL_SUBJECT:?missing EMAIL_SUBJECT}"
BODY="${EMAIL_BODY:?missing EMAIL_BODY}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

CC_HEADER=""
[ -n "${EMAIL_CC:-}" ] && CC_HEADER="Cc: $EMAIL_CC"

himalaya template send $ACCOUNT_FLAG <<EOF
From: $FROM
To: $TO
${CC_HEADER:+$CC_HEADER
}Subject: $SUBJECT

$BODY
EOF
