#!/usr/bin/env bash
# Read an email message via himalaya
# Env: APS_EMAIL_ACCOUNT
# Input (env): EMAIL_ID
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "read"
: "${BIN:?aps_init_backend did not set BIN}"

ID="${EMAIL_ID:?missing EMAIL_ID}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCOUNT_FLAG=()
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG=(-a "$ACCOUNT")

"$BIN" message read "$ID" ${ACCOUNT_FLAG+"${ACCOUNT_FLAG[@]}"}
