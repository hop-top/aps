#!/usr/bin/env bash
# List inbox envelopes via himalaya
# Env: APS_EMAIL_ACCOUNT
# Input (env): EMAIL_LIMIT, EMAIL_FOLDER
set -euo pipefail

ACCOUNT="${APS_EMAIL_ACCOUNT:-}"
FOLDER="${EMAIL_FOLDER:-INBOX}"

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

himalaya envelope list -f "$FOLDER" $ACCOUNT_FLAG -o json \
  2>/dev/null | head -n "${EMAIL_LIMIT:-10}"
