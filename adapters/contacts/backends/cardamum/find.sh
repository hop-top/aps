#!/usr/bin/env bash
# Search contacts via cardamum (list + grep)
# Input: CONTACT_QUERY
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "find"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACTS_ADDRESSBOOK:-default}"
QUERY="${CONTACT_QUERY:?missing CONTACT_QUERY}"

ACCT_FLAG=""
[ -n "$ACCOUNT" ] && ACCT_FLAG="-a $ACCOUNT"

"$BIN" cards list "$ABOOK" $ACCT_FLAG --json \
  2>/dev/null | grep -i "$QUERY"
