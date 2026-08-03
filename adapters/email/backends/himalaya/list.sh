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

# Array, not a string: "-a $ACCOUNT" expanded unquoted word-splits on
# an account name containing a space. An empty array expands to
# nothing under `set -u`, so the unset case still omits the flag.
ACCOUNT_FLAG=()
[ -n "$ACCOUNT" ] && ACCOUNT_FLAG=(-a "$ACCOUNT")

# Capture, then truncate via a here-string rather than a pipe.
#
# Any pipe into `head` fails under `set -o pipefail` with 141
# (SIGPIPE) once the input exceeds the limit: head exits after N
# lines and the writer is signalled. That is the COMMON case — an
# inbox nearly always holds more messages than requested — so the
# pipeline form made a normal truncating list look like a failure.
# A here-string gives head a file to read, leaving no writer to
# signal, and unlike `|| true` it does not also mask real errors.
#
# Backend stderr is deliberately not discarded: a listing that fails
# for a real reason (bad folder, auth expired) must say so rather than
# surfacing as an empty result.
LISTING="$("$BIN" envelope list -f "$FOLDER" ${ACCOUNT_FLAG+"${ACCOUNT_FLAG[@]}"} -o json)"
head -n "${EMAIL_LIMIT:-10}" <<< "$LISTING"
