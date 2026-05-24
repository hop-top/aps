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

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

"$BIN" message read "$ID" $ACCOUNT_FLAG
