#!/usr/bin/env bash
# T-1291 — automated check that every state-changing aps subcommand
# exposes --note in its --help output. Reads the inventory from the
# array below; prints PASS/FAIL per command and exits non-zero on any
# missing flag.
#
# The inventory matches the final list documented in the T-1291 PR.
# Add new state-changing subcommands here when introduced.

set -u

APS_BIN="${APS_BIN:-./aps_t1291}"
if [[ ! -x "$APS_BIN" ]]; then
  echo "build aps binary first: rtk proxy go build -buildvcs=false -o ./aps_t1291 ./cmd/aps/" >&2
  exit 2
fi

# Each entry is a single space-separated subcommand path.
SUBCOMMANDS=(
  # Profile (identity, highest stakes)
  "profile create"
  "profile edit"
  "profile delete"
  "profile import"
  "profile capability add"
  "profile capability remove"
  "profile workspace set"

  # Identity
  "identity init"

  # Sessions
  "session attach"
  "session detach"
  "session delete"
  "session terminate"

  # Workspaces + context
  "workspace create"
  "workspace remove"
  "workspace archive"
  "workspace join"
  "workspace leave"
  "workspace role"
  "workspace use"
  "workspace send"
  "workspace sync"
  "workspace ctx set"
  "workspace ctx delete"
  "workspace policy"
  "policy set"

  # Capabilities + bundles
  "capability adopt"
  "capability watch"
  "capability link"
  "capability delete"
  "capability install"
  "capability enable"
  "capability disable"
  "bundle create"
  "bundle edit"
  "bundle delete"

  # Multi-agent
  "squad create"
  "squad delete"
  "squad members add"
  "squad members remove"
  "adapter create"
  "adapter attach"
  "adapter detach"
  "adapter link add"
  "adapter link delete"
  "adapter approve"
  "adapter reject"
  "adapter revoke"
  "adapter pair"
  "adapter start"
  "adapter stop"

  # AGNTCY
  "directory register"
  "directory delete"
)

fail=0
for sub in "${SUBCOMMANDS[@]}"; do
  # shellcheck disable=SC2086
  out="$($APS_BIN $sub --help 2>&1)"
  if echo "$out" | grep -qE -- "--note"; then
    echo "PASS  $sub"
  else
    echo "FAIL  $sub  (--note not registered)"
    fail=$((fail + 1))
  fi
done

if [[ $fail -gt 0 ]]; then
  echo
  echo "FAIL — $fail subcommand(s) missing --note"
  exit 1
fi

echo
echo "PASS — all ${#SUBCOMMANDS[@]} subcommands expose --note"
