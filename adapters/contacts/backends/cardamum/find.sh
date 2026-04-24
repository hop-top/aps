#!/usr/bin/env bash
# Search contacts via cardamum (list + grep)
# Input: CONTACT_QUERY
set -euo pipefail

CARDAMUM="${CARDAMUM_BIN:-$HOME/.cargo/bin/cardamum}"
ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
QUERY="${CONTACT_QUERY:?missing CONTACT_QUERY}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

"$CARDAMUM" cards list "$ABOOK" $ACCT_FLAG --json \
  2>/dev/null | grep -i "$QUERY"
