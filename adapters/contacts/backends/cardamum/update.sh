#!/usr/bin/env bash
# Update contact via cardamum
# Env: CONTACTS_ACCOUNT, CONTACTS_ADDRESSBOOK
# Input: CONTACT_ID (required), CONTACT_NAME, CONTACT_EMAIL,
#        CONTACT_ORG, CONTACT_PHONE, CONTACT_NOTE,
#        CONTACT_ADDRESSBOOK (override)
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "update"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"
ID="${CONTACT_ID:?missing CONTACT_ID}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCT_FLAG=()
[ -n "$ACCOUNT" ] && ACCT_FLAG=(-a "$ACCOUNT")

# Read current card, apply updates, write back
CURRENT=$("$BIN" cards read "$ABOOK" "$ID" \
  ${ACCT_FLAG+"${ACCT_FLAG[@]}"})

# Apply field updates via sed.
#
# Every value is routed through aps_sed_replacement: an unescaped "/"
# closes the substitution early (sed then errors on the trailing text
# as a flag) and an unescaped "&" expands to the whole matched line,
# silently corrupting the field with exit 0. Both characters are
# ordinary in these fields — "Eng/Ops", "R&D".
[ -n "${CONTACT_NAME:-}" ] && \
  CURRENT=$(printf '%s\n' "$CURRENT" | sed "s/^FN:.*/FN:$(aps_sed_replacement "$CONTACT_NAME")/")
[ -n "${CONTACT_EMAIL:-}" ] && \
  CURRENT=$(printf '%s\n' "$CURRENT" | sed "s/^EMAIL:.*/EMAIL:$(aps_sed_replacement "$CONTACT_EMAIL")/")
[ -n "${CONTACT_ORG:-}" ] && \
  CURRENT=$(printf '%s\n' "$CURRENT" | sed "s/^ORG:.*/ORG:$(aps_sed_replacement "$CONTACT_ORG")/")
[ -n "${CONTACT_PHONE:-}" ] && \
  CURRENT=$(printf '%s\n' "$CURRENT" | sed "s/^TEL:.*/TEL:$(aps_sed_replacement "$CONTACT_PHONE")/")
[ -n "${CONTACT_NOTE:-}" ] && \
  CURRENT=$(printf '%s\n' "$CURRENT" | sed "s/^NOTE:.*/NOTE:$(aps_sed_replacement "$CONTACT_NOTE")/")

echo "$CURRENT" | "$BIN" cards update "$ABOOK" "$ID" \
  ${ACCT_FLAG+"${ACCT_FLAG[@]}"}
