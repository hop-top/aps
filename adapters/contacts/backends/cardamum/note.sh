#!/usr/bin/env bash
# Append note to contact via cardamum
# Input: CONTACT_ID, CONTACT_TEXT
set -euo pipefail

CARDAMUM="${CARDAMUM_BIN:-$HOME/.cargo/bin/cardamum}"
ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
ID="${CONTACT_ID:?missing CONTACT_ID}"
TEXT="${CONTACT_TEXT:?missing CONTACT_TEXT}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

# Read current, append to NOTE field
CURRENT=$("$CARDAMUM" cards read "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null)

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if echo "$CURRENT" | grep -q "^NOTE:"; then
  CURRENT=$(echo "$CURRENT" | \
    sed "s/^NOTE:\(.*\)/NOTE:\1 | [$TIMESTAMP] $TEXT/")
else
  CURRENT=$(echo "$CURRENT" | \
    sed "s/^END:VCARD/NOTE:[$TIMESTAMP] $TEXT\nEND:VCARD/")
fi

echo "$CURRENT" | "$CARDAMUM" cards update "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null
