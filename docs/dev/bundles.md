# Capability Bundles

Bundles are named presets that group capabilities, scope rules, env vars, binary wiring, and services.
A profile declares a bundle and gets everything in one line — no per-item enumeration.

## Using Bundles

Declare in `profile.yaml` under `capabilities`:

```yaml
capabilities:
  - bundle:developer
  - bundle:mobile
  - github          # individual capabilities still work alongside bundles
```

Multiple bundles compose; their scope rules union-merge into the effective profile scope.

## Built-in Bundles

| Bundle | Capabilities | Use Case |
|--------|-------------|----------|
| `developer` | git, github, bash, docker | General development agent |
| `reader` | bash (read-only), git:read | Read-only research agent |
| `mobile` | mobile-pairing, messenger | Mobile-connected agent |
| `ops` | docker, ssh, bash, webhooks | Infrastructure / ops agent |
| `comms` | messenger, webhooks, a2a | Communication-focused agent |
| `agntcy` | agntcy-identity, agntcy-trust, agntcy-directory, agntcy-observability | Full AGNTCY stack |

## CLI Reference

```bash
# List all bundles (built-in + user-defined)
aps bundle list

# Show full definition
aps bundle show developer

# Show fully merged definition (with extends resolved)
aps bundle show developer-strict --resolved

# Edit a bundle (opens $EDITOR; creates user override if built-in)
aps bundle edit developer

# Create a new bundle
aps bundle create my-bundle

# Delete a user-defined bundle (built-ins: override only, not deletable)
aps bundle delete my-bundle

# Validate a bundle file
aps bundle validate developer.yaml
```

## Bundle Format

```yaml
name: developer
description: "Standard developer agent"
version: "1.0"
extends: ""           # optional; one level deep only

capabilities:
  - git
  - github
  - bash

scope:
  operations:
    - git:read
    - git:write
    - shell:run
  file_patterns:
    - "**"
  networks:
    - "github.com"

env:
  GIT_AUTHOR_NAME: "${PROFILE_DISPLAY_NAME}"
  GIT_AUTHOR_EMAIL: "${PROFILE_EMAIL}"

services:
  - name: webhooks
    adapter: webhooks
    start: always       # always | on-demand

requires:
  - binary: git
    missing: error
    command: "git -c user.email=${PROFILE_EMAIL}"
  - binary: gh
    missing: warn
    message: "GitHub CLI not found"
  - binary: docker
    missing: skip
  - binary: curl
    blocked: true
    message: "Use the http capability instead"
  - binary: claude
    missing: warn
    command: "claude --dangerously-skip-permissions --profile ${PROFILE_ID}"
    deny_flags:
      - "--allow-all"
    deny_policy: strip  # strip | error

runtime_overrides:
  claude:
    env:
      CLAUDE_PROFILE: "${PROFILE_ID}"
```

### Template Variables

| Variable | Value |
|----------|-------|
| `${PROFILE_ID}` | Profile identifier |
| `${PROFILE_DISPLAY_NAME}` | Profile display name |
| `${PROFILE_EMAIL}` | Profile email |
| `${PROFILE_CONFIG_DIR}` | Profile config directory |
| `${PROFILE_DATA_DIR}` | Profile data directory |

## Binary Policies

| Policy | Effect |
|--------|--------|
| `error` | Profile fails to start |
| `warn` | Profile starts; warning logged; capability still declared |
| `skip` | Capability, its scope entries, and env vars dropped silently; shown in `aps profile status` |
| `blocked: true` | All invocations of this binary are prevented |

`deny_flags` from multiple bundles union-merge globally — they block the flag on that binary
regardless of which bundle triggered the invocation.

## Scope Merging

Bundle scope uses **union** (most permissive wins), opposite of squad/workspace scope which
intersects. When a bundle and profile both declare `file_patterns`, the union is used — a bundle
can only expand permissions, never restrict them. See [scope.md](scope.md) for intersection logic
that applies between profiles, squads, and workspaces.

## Bundle Inheritance

Use `extends` to inherit from another bundle:

```yaml
name: developer-strict
extends: developer       # one level deep only

scope:
  file_patterns:
    - "src/**"           # replaces developer's "**"

requires:
  - binary: docker
    missing: error       # replaces developer's "skip"
```

Rules:
- One level deep — a bundle cannot extend a bundle that itself extends another
- Built-in bundles can be extended; result is a user-defined bundle
- Child fields replace entire parent fields (not merged at sub-key level)

## User Overrides

Place a file at `~/.config/aps/bundles/<name>.yaml` to replace the built-in bundle of that name
entirely. Edit workflow: `aps bundle edit developer` creates the override file and opens `$EDITOR`.

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/bundle/types.go` | `Bundle`, `BinaryRequirement`, `ServiceEntry` types |
| `internal/core/bundle/loader.go` | Embedded built-ins + user override loading |
| `internal/core/bundle/resolver.go` | Evaluate requires, merge scope, inject env |
| `internal/core/bundle/registry.go` | List, get, validate bundles |
| `assets/bundles/*.yaml` | Embedded built-in bundle definitions |
| `internal/cli/bundle/` | All CLI commands |
