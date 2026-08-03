#!/usr/bin/env bash
# Append note to contact via cardamum
# Env: CONTACTS_ACCOUNT, CONTACTS_ADDRESSBOOK
# Input: CONTACT_ID, CONTACT_TEXT, CONTACT_ADDRESSBOOK (override)
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "note"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${CONTACTS_ACCOUNT:-}"
ABOOK="${CONTACT_ADDRESSBOOK:-${CONTACTS_ADDRESSBOOK:-default}}"
ID="${CONTACT_ID:?missing CONTACT_ID}"
TEXT="${CONTACT_TEXT:?missing CONTACT_TEXT}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCT_FLAG=()
[ -n "$ACCOUNT" ] && ACCT_FLAG=(-a "$ACCOUNT")

# Read current, append to NOTE field
CURRENT=$("$BIN" cards read "$ABOOK" "$ID" \
  ${ACCT_FLAG+"${ACCT_FLAG[@]}"})

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# The note text is escaped for the sed replacement side: an unescaped
# "/" ends the substitution early and an unescaped "&" expands to the
# whole matched line. Both are ordinary in note text (URLs, "R&D").
ESCAPED_TEXT="$(aps_sed_replacement "$TEXT")"

if printf '%s\n' "$CURRENT" | grep -q "^NOTE:"; then
  CURRENT=$(printf '%s\n' "$CURRENT" | \
    sed "s/^NOTE:\(.*\)/NOTE:\1 | [$TIMESTAMP] $ESCAPED_TEXT/")
else
  # Insert before END:VCARD with awk rather than sed. A "\n" inside a
  # sed replacement is not expanded by BSD sed (macOS), which wrote a
  # literal "n" and fused the note onto END:VCARD. awk also takes the
  # text as a -v variable instead of interpolating it into a script,
  # so this branch needs no escaping at all.
  CURRENT=$(printf '%s\n' "$CURRENT" | \
    awk -v note="NOTE:[$TIMESTAMP] $TEXT" '/^END:VCARD/{print note} {print}')
fi

echo "$CURRENT" | "$BIN" cards update "$ABOOK" "$ID" \
  ${ACCT_FLAG+"${ACCT_FLAG[@]}"}
