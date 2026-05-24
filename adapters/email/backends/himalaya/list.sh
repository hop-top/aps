#!/usr/bin/env bash
# List inbox envelopes via himalaya
# Env: APS_EMAIL_ACCOUNT
# Input (env): EMAIL_LIMIT, EMAIL_FOLDER
set -euo pipefail

# shellcheck source=../../../_lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
aps_init_backend "list"
: "${BIN:?aps_init_backend did not set BIN}"

ACCOUNT="${APS_EMAIL_ACCOUNT:-}"
FOLDER="${EMAIL_FOLDER:-INBOX}"

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

"$BIN" envelope list -f "$FOLDER" $ACCOUNT_FLAG -o json \
  2>/dev/null | head -n "${EMAIL_LIMIT:-10}"
