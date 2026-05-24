#!/usr/bin/env bash
# Show contact detail via cardamum
# Input: CONTACT_ID
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "show"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
ID="${CONTACT_ID:?missing CONTACT_ID}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

"$BIN" cards read "$ABOOK" "$ID" $ACCT_FLAG --json \
  2>/dev/null
