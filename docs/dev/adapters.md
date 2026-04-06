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

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/adapter/types.go` | Core types: `Adapter`, `AdapterType`, `AdapterManifest`, `AdapterState` |
| `internal/core/adapter/manager.go` | CRUD and lifecycle: `CreateAdapter`, `StartAdapter`, `StopAdapter`, `LinkAdapter` |
| `internal/core/adapter/export.go` | YAML export/import: `ExportToYAML`, `ImportFromYAML` |
| `internal/core/adapter/mobile/` | Mobile pairing: `Registry`, `AdapterServer`, `TokenManager` |
| `internal/cli/adapter/` | All CLI commands |
