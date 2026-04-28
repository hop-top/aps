#!/usr/bin/env bash
# Update contact via cardamum
# Input: CONTACT_ID (required), CONTACT_NAME, CONTACT_EMAIL,
#        CONTACT_ORG, CONTACT_PHONE, CONTACT_NOTE
set -euo pipefail

CARDAMUM="${CARDAMUM_BIN:-$HOME/.cargo/bin/cardamum}"
ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
ID="${CONTACT_ID:?missing CONTACT_ID}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

# Read current card, apply updates, write back
CURRENT=$("$CARDAMUM" cards read "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null)

# Apply field updates via sed
[ -n "${CONTACT_NAME:-}" ] && \
  CURRENT=$(echo "$CURRENT" | sed "s/^FN:.*/FN:$CONTACT_NAME/")
[ -n "${CONTACT_EMAIL:-}" ] && \
  CURRENT=$(echo "$CURRENT" | sed "s/^EMAIL:.*/EMAIL:$CONTACT_EMAIL/")
[ -n "${CONTACT_ORG:-}" ] && \
  CURRENT=$(echo "$CURRENT" | sed "s/^ORG:.*/ORG:$CONTACT_ORG/")
[ -n "${CONTACT_PHONE:-}" ] && \
  CURRENT=$(echo "$CURRENT" | sed "s/^TEL:.*/TEL:$CONTACT_PHONE/")
[ -n "${CONTACT_NOTE:-}" ] && \
  CURRENT=$(echo "$CURRENT" | sed "s/^NOTE:.*/NOTE:$CONTACT_NOTE/")

echo "$CURRENT" | "$CARDAMUM" cards update "$ABOOK" "$ID" \
  $ACCT_FLAG 2>/dev/null
