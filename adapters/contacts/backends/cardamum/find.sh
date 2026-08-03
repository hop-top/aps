#!/usr/bin/env bash
# Search contacts via cardamum (list + grep)
# Env: CONTACTS_ACCOUNT, CONTACTS_ADDRESSBOOK
# Input: CONTACT_QUERY, CONTACT_ADDRESSBOOK (override)
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "find"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"
QUERY="${CONTACT_QUERY:?missing CONTACT_QUERY}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCT_FLAG=()
[ -n "$ACCOUNT" ] && ACCT_FLAG=(-a "$ACCOUNT")

# Capture the listing first so a backend failure (auth expired, no
# such addressbook) is distinguishable from an empty search result.
# Backend stderr is deliberately not discarded.
LISTING="$("$BIN" cards list "$ABOOK" ${ACCT_FLAG+"${ACCT_FLAG[@]}"} --json)"

# grep exits 1 when nothing matches. Under `set -e` that aborted the
# script, so "no contact matched" was reported to the caller as a
# failure — indistinguishable from a real error. An empty result is an
# ordinary outcome for a search, so only grep's error status (>1) is
# propagated.
if ! printf '%s\n' "$LISTING" | grep -i -- "$QUERY"; then
  status=$?
  [ "$status" -eq 1 ] && exit 0
  exit "$status"
fi
