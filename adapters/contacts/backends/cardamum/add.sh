#!/usr/bin/env bash
# Add a contact via cardamum (vCard format)
# Input: CONTACT_EMAIL (required), CONTACT_NAME, CONTACT_ORG,
#        CONTACT_PHONE, CONTACT_NOTE
#        CONTACT_ADDRESSBOOK (override)
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "add"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"
EMAIL="${CONTACT_EMAIL:?missing CONTACT_EMAIL}"
NAME="${CONTACT_NAME:-$EMAIL}"
ORG="${CONTACT_ORG:-}"
PHONE="${CONTACT_PHONE:-}"
NOTE="${CONTACT_NOTE:-}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

# Build vCard
VCARD="BEGIN:VCARD
VERSION:3.0
FN:$NAME
EMAIL:$EMAIL"

[ -n "$ORG" ] && VCARD="$VCARD
ORG:$ORG"
[ -n "$PHONE" ] && VCARD="$VCARD
TEL:$PHONE"
[ -n "$NOTE" ] && VCARD="$VCARD
NOTE:$NOTE"

VCARD="$VCARD
END:VCARD"

echo "$VCARD" | "$BIN" cards create "$ABOOK" \
  $ACCT_FLAG 2>/dev/null
