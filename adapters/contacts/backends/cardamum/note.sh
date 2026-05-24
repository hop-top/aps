#!/usr/bin/env bash
# Append note to contact via cardamum
# Input: CONTACT_ID, CONTACT_TEXT
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "note"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
ID="${CONTACT_ID:?missing CONTACT_ID}"
TEXT="${CONTACT_TEXT:?missing CONTACT_TEXT}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

# Read current, append to NOTE field
CURRENT=$("$BIN" cards read "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null)

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if echo "$CURRENT" | grep -q "^NOTE:"; then
  CURRENT=$(echo "$CURRENT" | \
    sed "s/^NOTE:\(.*\)/NOTE:\1 | [$TIMESTAMP] $TEXT/")
else
  CURRENT=$(echo "$CURRENT" | \
    sed "s/^END:VCARD/NOTE:[$TIMESTAMP] $TEXT\nEND:VCARD/")
fi

echo "$CURRENT" | "$BIN" cards update "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null
