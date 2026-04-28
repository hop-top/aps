#!/usr/bin/env bash
# Show contact detail via cardamum
# Input: CONTACT_ID
set -euo pipefail

CARDAMUM="${CARDAMUM_BIN:-$HOME/.cargo/bin/cardamum}"
ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
ID="${CONTACT_ID:?missing CONTACT_ID}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

"$CARDAMUM" cards read "$ABOOK" "$ID" $ACCT_FLAG --json \
  2>/dev/null
