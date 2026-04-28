#!/usr/bin/env bash
# List all contacts via cardamum
# Env: CONTACTS_ACCOUNT, CONTACTS_ADDRESSBOOK
# Input: CONTACT_ADDRESSBOOK (override)
set -euo pipefail

CARDAMUM="${CARDAMUM_BIN:-$HOME/.cargo/bin/cardamum}"
ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

"$CARDAMUM" cards list "$ABOOK" $ACCT_FLAG --json 2>/dev/null
