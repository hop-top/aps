#!/usr/bin/env bash
# Email send via himalaya
# Env: APS_EMAIL_FROM, APS_EMAIL_ACCOUNT
# Input (env): EMAIL_TO, EMAIL_SUBJECT, EMAIL_BODY, EMAIL_CC
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "send"
: "${BIN:?aps_init_backend did not set BIN}"

FROM="${APS_EMAIL_FROM:?missing APS_EMAIL_FROM}"
TO="${EMAIL_TO:?missing EMAIL_TO}"
SUBJECT="${EMAIL_SUBJECT:?missing EMAIL_SUBJECT}"
BODY="${EMAIL_BODY:?missing EMAIL_BODY}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space, passing a truncated account plus
# a stray positional arg. An empty array expands to nothing under
# `set -u`, so the unset case still omits the flag entirely.
ACCOUNT_FLAG=()
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG=(-a "$ACCOUNT")

CC_HEADER=""
[ -n "${EMAIL_CC:-}" ] && CC_HEADER="Cc: $EMAIL_CC"

"$BIN" template send ${ACCOUNT_FLAG+"${ACCOUNT_FLAG[@]}"} <<EOF
From: $FROM
To: $TO
${CC_HEADER:+$CC_HEADER
}Subject: $SUBJECT

$BODY
EOF
