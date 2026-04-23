# Adapters

Adapters are integration points that connect APS profiles to external systems — messengers, protocols, mobile clients, desktop apps, and custom subprocesses.

> **Terminology note:** Adapters were previously called "devices" (renamed in T-0001..T-0006).

## Types

| Type | Description |
|------|-------------|
| `messenger` | Connects to a messaging platform (Telegram, Discord, Slack, etc.) |
| `protocol` | Implements an agent communication protocol (A2A, ACP, webhook) |
| `mobile` | Mobile client pairing via QR code and WebSocket |
| `desktop` | Desktop application integration |
| `sense` | Input/sensor adapter |
| `actuator` | Output/action adapter |

## Loading Strategies

| Strategy | Description |
|----------|-------------|
| `subprocess` | Runs as a child process managed by APS |
| `script` | Executes a script on demand |
| `builtin` | Native Go implementation inside APS |

## Scope

Adapters are either `global` (shared across all profiles) or `profile`-scoped (tied to a specific profile).

## CLI Reference

### Managing Adapters

```bash
# List all adapters
aps adapter list

# Create an adapter
aps adapter create <name> --type <type> [--strategy subprocess] [--scope profile]

# Show adapter status
aps adapter status <name>

# View logs
aps adapter logs <name> [--tail 50] [--follow]

# Start / stop
aps adapter start <name>
aps adapter stop <name> [--force]
```

### Linking Adapters to Profiles

```bash
# Link adapter to current profile
aps adapter link <name>

# Unlink adapter from profile
aps adapter unlink <name>
```

### Export / Import

Adapters can be exported to YAML for sharing or backup.

```bash
# Export adapter to file
aps adapter export <name> --output adapter.yaml

# Export to stdout
aps adapter export <name> -o -

# Import adapter from file
aps adapter import adapter.yaml
```

**Export format** (`adapter.yaml`):
```yaml
adapter:
  name: my-telegram-bot
  type: messenger
  strategy: subprocess
  scope: profile
  # ...
manifest:
  api_version: v1
  kind: Adapter
  name: my-telegram-bot
  # ...
exported_at: "2026-03-08T12:00:00Z"
```

## Mobile Adapter Pairing

Mobile adapters use a QR-code-based pairing flow over WebSocket.

```bash
# Start pairing server and generate QR code
aps adapter pair <name> [--expires 14d] [--capabilities run:stateless,monitor:sessions]

# List pending (unapproved) mobile adapters
aps adapter pending

# Approve a pending mobile adapter
aps adapter approve <adapter-id>

# Reject a pending mobile adapter
aps adapter reject <adapter-id>

# Revoke a mobile adapter
aps adapter revoke <adapter-id>
```

### Approval Flow

By default, mobile adapters require approval before they can connect. The flow:

1. Run `aps adapter pair` on the host — displays a QR code
2. Scan the QR code on the mobile device
3. Device appears in `aps adapter pending`
4. Run `aps adapter approve <id>` to allow connection

## Workspace Attachment

Adapters can be attached to workspaces with role-based access.

```bash
# Attach adapter to workspace
aps adapter attach <adapter-id> --workspace <workspace-id> --role viewer

# Detach adapter from workspace
aps adapter detach <adapter-id> --workspace <workspace-id>

# View workspace adapter presence
aps adapter presence --workspace <workspace-id>

# View adapter links
aps adapter links

# Set permissions for attached adapter
aps adapter set-permissions <adapter-id> --workspace <workspace-id> --role collaborator
```

**Roles:**
| Role | Permissions |
|------|------------|
| `owner` | Full access: read, write, execute, manage, sync |
| `collaborator` | Operational: read, write, execute, sync |
| `viewer` | Read-only: read, sync |

## Adapter Manifest

Adapters can be described by a manifest file (`adapter.yaml`):

```yaml
api_version: v1
kind: Adapter
name: my-messenger
type: messenger
strategy: subprocess
description: "Telegram integration for billing profile"
config:
  token_env: TELEGRAM_BOT_TOKEN
  polling_interval: 5s
```

## Script Adapter Execution

Script-strategy adapters don't run persistently. Instead,
they define **actions** in their manifest and execute
backend scripts on demand via `aps adapter exec`.

### CLI

```bash
# Execute an action on a script adapter
aps adapter exec <adapter> <action> [flags]

# With profile (resolves From address from profile.email)
aps adapter exec email send --profile noor \
  --input to=user@example.com \
  --input subject="Hello" \
  --input body="Message"

# With explicit From (no profile needed)
aps adapter exec email send --from ops@company.com \
  --input to=user@example.com \
  --input subject="Hello" \
  --input body="Message"

# List inbox
aps adapter exec email list --profile noor

# Read a message
aps adapter exec email read --profile noor --input id=7131

# Reply
aps adapter exec email reply --profile noor \
  --input id=7131 --input body="Thanks!"
```

### Manifest with Actions

Script adapters define actions under `config.actions`:

```yaml
api_version: v1
kind: adapter
name: email
type: messenger
strategy: script
config:
  backend: himalaya
  account: ideacrafters
  actions:
    - name: send
      script: backends/{{backend}}/send.sh
      input:
        - name: to
          required: true
        - name: subject
          required: true
        - name: body
          required: true
    - name: list
      script: backends/{{backend}}/list.sh
      input:
        - name: limit
          default: "10"
```

`{{backend}}` is replaced with `config.backend` at runtime.

### Backend Scripts

Each action maps to a shell script. Scripts receive:

| Env var | Source | Description |
|---------|--------|-------------|
| `APS_EMAIL_FROM` | Profile email | From address |
| `APS_EMAIL_ACCOUNT` | `config.account` | Backend account name |
| `EMAIL_<INPUT>` | `--input` flags | Uppercased input names |

Scripts are in `adapters/<name>/backends/<backend>/`.

### Writing a New Backend

1. Create `backends/<name>/` under the adapter dir
2. Add one script per action (send.sh, list.sh, etc.)
3. Scripts must be executable, use `#!/usr/bin/env bash`
4. Read inputs from `EMAIL_*` env vars
5. Read From address from `APS_EMAIL_FROM`
6. Output to stdout (JSON preferred for list/read)
7. Exit 0 on success, non-zero on failure

### Email Resolution

`aps adapter exec` resolves the sender address:

1. `--from` flag (highest priority)
2. `profile.email` field in profile.yaml
3. Error if neither available

The `profile.email` field is set via `aps profile new --email`
or by editing profile.yaml directly.

## Profile Email Field

Profiles now have a top-level `email` field in profile.yaml:

```yaml
id: noor
display_name: Noor
email: noor@example.com
```

Set during creation:
```bash
aps profile new noor --display-name Noor --email noor@example.com
```

The email is also available as `${PROFILE_EMAIL}` in
bundle template variables.

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/adapter/types.go` | Core types: `Adapter`, `AdapterType`, `AdapterManifest`, `AdapterState` |
| `internal/core/adapter/manager.go` | CRUD and lifecycle: `CreateAdapter`, `StartAdapter`, `StopAdapter`, `LinkAdapter` |
| `internal/core/adapter/script_exec.go` | Script adapter action dispatch: `ExecAction`, script resolution, env building |
| `internal/core/adapter/export.go` | YAML export/import: `ExportToYAML`, `ImportFromYAML` |
| `internal/core/adapter/mobile/` | Mobile pairing: `Registry`, `AdapterServer`, `TokenManager` |
| `internal/cli/adapter/exec.go` | `aps adapter exec` CLI command |
| `internal/cli/adapter/` | All CLI commands |
| `adapters/email/` | Email adapter manifest + himalaya backend scripts |
