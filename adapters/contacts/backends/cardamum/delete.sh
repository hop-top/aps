#!/usr/bin/env bash
# Delete a contact via cardamum
# Env: CONTACTS_ACCOUNT, CONTACTS_ADDRESSBOOK
# Input: CONTACT_ID, CONTACT_ADDRESSBOOK (override)
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "delete"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"
ID="${CONTACT_ID:?missing CONTACT_ID}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCT_FLAG=()
[ -n "$ACCOUNT" ] && ACCT_FLAG=(-a "$ACCOUNT")

"$BIN" cards delete "$ABOOK" "$ID" ${ACCT_FLAG+"${ACCT_FLAG[@]}"} \
