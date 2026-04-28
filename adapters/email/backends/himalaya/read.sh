#!/usr/bin/env bash
# Read an email message via himalaya
# Env: APS_EMAIL_ACCOUNT
# Input (env): EMAIL_ID
set -euo pipefail

ID="${EMAIL_ID:?missing EMAIL_ID}"
ACCOUNT="${APS_EMAIL_ACCOUNT:-}"

ACCOUNT_FLAG=""
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG="-a $ACCOUNT"

himalaya message read "$ID" $ACCOUNT_FLAG
