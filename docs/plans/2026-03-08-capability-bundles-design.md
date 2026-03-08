# Capability Bundles — Design

**Date:** 2026-03-08
**Status:** Draft

---

## Problem

APS profiles declare capabilities as an itemized list. Every profile must enumerate each permission, scope rule, env var, and service individually. This is verbose, error-prone, and a poor first-run experience.

---

## Solution

**Bundles** are named presets that group capabilities together. A profile declares a bundle and gets all its scope rules, env vars, binary wiring, and services in one line.

```yaml
# profile.yaml
capabilities:
  - bundle:developer
  - bundle:mobile-pairing
  - github          # individual capabilities still work
```

---

## Bundle File Format

Bundles are YAML files. Built-ins ship embedded in the APS binary (`go:embed`). Users override by name in `~/.config/aps/bundles/`.

A user-defined file at `~/.config/aps/bundles/developer.yaml` replaces the built-in `developer` bundle entirely.

### Full Schema

```yaml
name: developer
description: "Standard developer agent — git, GitHub, shell access"
version: "1.0"

# Capabilities this bundle activates
capabilities:
  - git
  - github
  - bash
  - docker

# Scope rules merged (union) into the profile's scope
scope:
  operations:
    - git:read
    - git:write
    - git:push
    - github:pr
    - github:issues
    - shell:run
    - docker:build
    - docker:run
  file_patterns:
    - "**"
  networks:
    - "github.com"
    - "api.github.com"

# Env vars injected into every session using this bundle
env:
  GIT_AUTHOR_NAME: "${PROFILE_DISPLAY_NAME}"
  GIT_AUTHOR_EMAIL: "${PROFILE_EMAIL}"

# Services started with the profile or on demand
services:
  - name: webhooks
    adapter: webhooks
    start: always       # always | on-demand

# Binary requirements checked at profile start
requires:
  - binary: git
    missing: error
    command: "git -c user.email=${PROFILE_EMAIL} -c user.name=${PROFILE_DISPLAY_NAME}"

  - binary: gh
    missing: warn
    message: "GitHub CLI not found — some github:* operations may fail"

  - binary: docker
    missing: skip       # drops docker scope/env silently; shown in profile status

  - binary: curl
    blocked: true
    message: "Use the http capability instead of curl directly"

  - binary: claude
    missing: warn
    command: "claude --dangerously-skip-permissions --profile ${PROFILE_ID}"
    deny_flags:
      - "--allow-all"
      - "--no-sandbox"
    deny_policy: strip  # strip | error

# Runtime-specific overrides (merged on top of base config)
runtime_overrides:
  claude:
    env:
      CLAUDE_PROFILE: "${PROFILE_ID}"
  gemini:
    env:
      GEMINI_PROFILE: "${PROFILE_ID}"
```

---

## Scope Merging

Bundle scope merges with the profile's own scope using **union** (most permissive wins). This is the opposite of squad/workspace scope, which intersects. Capabilities add permissions — they don't restrict them.

| Source | file_patterns | Result |
|--------|--------------|--------|
| Profile | `src/**` | |
| Bundle | `**` | `**` |

| Source | operations | Result |
|--------|-----------|--------|
| Profile | `git:read` | |
| Bundle | `git:read, git:write` | `git:read, git:write` |

---

## Binary Entries

Each `requires` entry describes one binary:

| Field | Type | Description |
|-------|------|-------------|
| `binary` | string | Executable name checked on `$PATH` |
| `missing` | `skip \| warn \| error` | Policy when binary not found |
| `blocked` | bool | Prevent all invocations of this binary |
| `command` | string | Base invocation override (supports template vars) |
| `deny_flags` | []string | Flags stripped or blocked if agent appends them |
| `deny_policy` | `strip \| error` | What to do when a denied flag is used |
| `message` | string | Shown in `aps profile status` |

### Command Override Behavior

The `command` field sets the base invocation. Flags the agent adds are appended after:

```
bundle command:   claude --dangerously-skip-permissions --profile my-agent
agent appends:    --verbose
resolved:         claude --dangerously-skip-permissions --profile my-agent --verbose
```

Template variables available in `command` and `env`:

| Variable | Value |
|----------|-------|
| `${PROFILE_ID}` | Profile identifier |
| `${PROFILE_DISPLAY_NAME}` | Profile display name |
| `${PROFILE_EMAIL}` | Profile email |
| `${PROFILE_CONFIG_DIR}` | Profile config directory |
| `${PROFILE_DATA_DIR}` | Profile data directory |

### Missing Binary Policies

| Policy | Effect |
|--------|--------|
| `error` | Profile fails to start |
| `warn` | Profile starts; warning logged; capability still declared |
| `skip` | Capability, its scope entries, and its env vars are dropped; shown in `aps profile status` |

---

## Profile Status Output

`aps profile status` surfaces every binary's resolved state:

```
Profile: my-agent
Bundle:  developer

  ✓ git        active
  ✓ github     active
  ✓ bash       active
  ✗ docker     skipped  (binary not found)
  ✗ curl       blocked  (use http capability instead)

Scope: 10 operations, files: **, networks: github.com api.github.com

Services:
  webhooks     on-demand
```

`aps profile status --verbose` lists the full resolved scope rules, including which entries came from which bundle.

---

## Runtime Overrides

Bundles are runtime-agnostic by default. The `runtime_overrides` block layers additional config on top when APS detects a specific runtime (`claude`, `gemini`, `codex`, etc.). Detection uses the SmartPatterns system already in `internal/core/capability/registry.go`.

---

## Built-in Bundles

Proposed initial set:

| Bundle | Capabilities | Use case |
|--------|-------------|----------|
| `developer` | git, github, bash, docker | General development agent |
| `reader` | bash (read-only), git:read | Read-only research agent |
| `mobile` | mobile-pairing, messenger | Mobile-connected agent |
| `ops` | docker, ssh, bash, webhooks | Infrastructure/ops agent |
| `comms` | messenger, webhooks, a2a | Communication-focused agent |
| `agntcy` | agntcy-identity, agntcy-trust, agntcy-directory, agntcy-observability | Full AGNTCY stack |

---

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/bundle/types.go` | `Bundle`, `BinaryRequirement`, `ServiceEntry` types |
| `internal/core/bundle/loader.go` | Embedded built-ins + user override loading |
| `internal/core/bundle/resolver.go` | Evaluate requires, merge scope, inject env |
| `internal/core/bundle/registry.go` | List, get, validate bundles |
| `assets/bundles/*.yaml` | Embedded built-in bundle definitions |
| `internal/cli/bundle/` | `aps bundle list`, `aps bundle show`, `aps bundle edit` |

---

## CLI

```bash
# List all bundles (built-in and user-defined)
aps bundle list

# Show a bundle's full definition
aps bundle show developer

# Edit a bundle (opens in $EDITOR; creates user override if built-in)
aps bundle edit developer

# Create a new bundle
aps bundle create my-bundle

# Delete a user-defined bundle (built-ins cannot be deleted, only overridden)
aps bundle delete my-bundle

# Validate a bundle file
aps bundle validate developer.yaml
```

---

## Open Questions

- Should a bundle be able to extend another bundle (`extends: developer`)?
- Should `deny_flags` apply to all binaries or only the ones declared in the bundle?
