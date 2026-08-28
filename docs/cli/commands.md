# Aps CLI command reference

Every aps command, flag, and subcommand, rendered from the live cobra
command tree (`internal/tools/climd`). The region between the cog
markers below is generated — do not edit it by hand. Regenerate with
`make docs-gen`; CI's `make docs-check` fails when it drifts from the
command tree.

For global flags, exit codes, and the `--note|-n` inventory, see
[reference.md](reference.md).

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/climd"],
    text=True))
]]] -->
## aps

Agent Profile System CLI

### Synopsis

Agent Profile System CLI

Run aps with no arguments to launch the interactive TUI.

Pass a profile ID to start a session for that profile, or pass a profile
ID followed by a command to run that command under the selected profile.

```
aps [flags]
```

### Options

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
  -h, --help               help for aps
      --help-all           Show all commands including management
      --help-instance      Show only instance commands
      --help-interact      Show only interact commands
      --help-management    Show only management commands
      --help-organize      Show only organize commands
      --help-pipelines     Show only pipelines commands
      --help-security      Show only security commands
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a](#aps-a2a)	 - Manage A2A (Agent-to-Agent) protocol operations
* [aps acp](#aps-acp)	 - Manage ACP (Agent Client Protocol) server
* [aps action](#aps-action)	 - Manage and execute profile actions
* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)
* [aps alias](#aps-alias)	 - Manage command aliases
* [aps bundle](#aps-bundle)	 - Manage capability bundles
* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)
* [aps chat](#aps-chat)	 - Chat with a profile-backed assistant
* [aps completion](#aps-completion)	 - Generate completion script
* [aps config](#aps-config)	 - Inspect aps configuration
* [aps contact](#aps-contact)	 - Manage contacts via adapter
* [aps directory](#aps-directory)	 - Manage AGNTCY Directory registration and discovery
* [aps docs](#aps-docs)	 - Generate documentation
* [aps env](#aps-env)	 - Output environment variables for configured capabilities
* [aps identity](#aps-identity)	 - Manage DID-based agent identity
* [aps listen](#aps-listen)	 - Subscribe to bus topics for a profile and print events as JSONL
* [aps migrate](#aps-migrate)	 - Migrate legacy configurations to new formats
* [aps observability](#aps-observability)	 - Manage OpenTelemetry observability
* [aps org](#aps-org)	 - Inspect the reporting hierarchy across profiles
* [aps policy](#aps-policy)	 - Manage workspace access policies
* [aps profile](#aps-profile)	 - Manage agent profiles
* [aps run](#aps-run)	 - Run a command in a profile context
* [aps serve](#aps-serve)	 - Start protocol server
* [aps service](#aps-service)	 - Manage profile-facing services
* [aps session](#aps-session)	 - Manage sessions
* [aps skill](#aps-skill)	 - Manage Agent Skills
* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)
* [aps status](#aps-status)	 - Show aps configuration and runtime status
* [aps toolspec](#aps-toolspec)	 - Print the aps tool specification (commands, flags, errors, workflows)
* [aps upgrade](#aps-upgrade)	 - Check for and install updates
* [aps version](#aps-version)	 - Print version information
* [aps voice](#aps-voice)	 - Manage voice sessions and the voice backend service
* [aps webhook](#aps-webhook)	 - Manage Webhook server
* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps a2a

Manage A2A (Agent-to-Agent) protocol operations

### Synopsis

Manage A2A (Agent-to-Agent) protocol operations for inter-profile communication.

The a2a command group provides operations for:
- Creating and managing tasks
- Sending messages between profiles
- Subscribing to task updates
- Managing agent cards
- Discovering other agents

### Options

```
  -h, --help   help for a2a
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps a2a card](#aps-a2a-card)	 - Manage A2A agent cards
* [aps a2a server](#aps-a2a-server)	 - Start an A2A server for a profile
* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks
* [aps a2a toggle](#aps-a2a-toggle)	 - Enable or disable A2A for a profile

## aps a2a card

Manage A2A agent cards

### Synopsis

Show local profile cards or fetch remote agent cards.

### Options

```
  -h, --help   help for card
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a](#aps-a2a)	 - Manage A2A (Agent-to-Agent) protocol operations
* [aps a2a card fetch](#aps-a2a-card-fetch)	 - Fetch an Agent Card from a URL
* [aps a2a card show](#aps-a2a-card-show)	 - Show the Agent Card for a profile

## aps a2a card fetch

Fetch an Agent Card from a URL

### Synopsis

Fetch an A2A Agent Card from a remote URL (typically /.well-known/agent-card).

Example:
  aps a2a card fetch --url http://localhost:8081/.well-known/agent-card

```
aps a2a card fetch [flags]
```

### Options

```
  -h, --help         help for fetch
  -u, --url string   Agent Card URL (required)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a card](#aps-a2a-card)	 - Manage A2A agent cards

## aps a2a card show

Show the Agent Card for a profile

### Synopsis

Display the A2A Agent Card for a specified profile.

```
aps a2a card show [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a card](#aps-a2a-card)	 - Manage A2A agent cards

## aps a2a server

Start an A2A server for a profile

### Synopsis

Start an A2A server to expose a profile as an A2A agent.

The server will listen on the address configured in the profile's A2A settings
(default: 127.0.0.1:8081) and serve:
  - A2A JSON-RPC endpoint at /
  - Agent Card at /.well-known/agent-card

Example:
  aps --profile worker a2a server

```
aps a2a server [flags]
```

### Options

```
  -h, --help   help for server
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a](#aps-a2a)	 - Manage A2A (Agent-to-Agent) protocol operations

## aps a2a tasks

Manage A2A tasks

### Synopsis

List, inspect, send, cancel, and subscribe to A2A tasks.

### Options

```
  -h, --help   help for tasks
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a](#aps-a2a)	 - Manage A2A (Agent-to-Agent) protocol operations
* [aps a2a tasks cancel](#aps-a2a-tasks-cancel)	 - Cancel a running A2A task
* [aps a2a tasks list](#aps-a2a-tasks-list)	 - List A2A tasks for a profile
* [aps a2a tasks send](#aps-a2a-tasks-send)	 - Send a message to create or continue an A2A task
* [aps a2a tasks show](#aps-a2a-tasks-show)	 - Show details of a specific A2A task
* [aps a2a tasks stream](#aps-a2a-tasks-stream)	 - Send a message with streaming updates (not yet supported)
* [aps a2a tasks subscribe](#aps-a2a-tasks-subscribe)	 - Subscribe to push notifications for an A2A task

## aps a2a tasks cancel

Cancel a running A2A task

### Synopsis

Cancel a running A2A task on a target profile.

```
aps a2a tasks cancel <task-id> [flags]
```

### Options

```
  -h, --help            help for cancel
  -t, --target string   Target profile ID (required)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a tasks list

List A2A tasks for a profile

### Synopsis

List A2A tasks for the active profile (--profile global) with
optional filtering by status.

```
aps a2a tasks list [flags]
```

### Options

```
  -h, --help            help for list
      --status string   Filter by status (submitted, working, completed, failed, cancelled)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a tasks send

Send a message to create or continue an A2A task

### Synopsis

Send a message to create a new A2A task or continue an existing task.

Example:
  aps a2a tasks send --target worker --message "Deploy application"
  aps a2a tasks send --target worker --task-id <id> --message "Continue deployment"

```
aps a2a tasks send [flags]
```

### Options

```
  -h, --help             help for send
  -m, --message string   Message text (required)
  -t, --target string    Target profile ID (required)
      --task-id string   Existing task ID (optional, creates new if not specified)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a tasks show

Show details of a specific A2A task

### Synopsis

Retrieve detailed information about a specific A2A task including its message history.

```
aps a2a tasks show <task-id> [flags]
```

### Options

```
  -h, --help          help for show
      --history int   Limit message history length
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a tasks stream

Send a message with streaming updates (not yet supported)

### Synopsis

Send a message with streaming updates. This feature requires SDK support for streaming.

```
aps a2a tasks stream [flags]
```

### Options

```
  -h, --help             help for stream
  -m, --message string   Message text
  -t, --target string    Target profile ID
      --task-id string   Existing task ID (optional)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a tasks subscribe

Subscribe to push notifications for an A2A task

### Synopsis

Subscribe to push notifications for task updates via webhook.

Example:
  aps a2a tasks subscribe <task-id> --target worker --webhook http://localhost:9000/hook

```
aps a2a tasks subscribe <task-id> [flags]
```

### Options

```
  -h, --help             help for subscribe
  -t, --target string    Target profile ID (required)
      --webhook string   Webhook URL for push notifications (required)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a tasks](#aps-a2a-tasks)	 - Manage A2A tasks

## aps a2a toggle

Enable or disable A2A for a profile

### Synopsis

Enable or disable A2A (Agent-to-Agent) protocol for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps --profile worker a2a toggle                    # Toggle A2A
  aps --profile worker a2a toggle --enabled=on       # Force enable
  aps --profile worker a2a toggle --enabled=off      # Force disable
  aps --profile worker a2a toggle --protocol=grpc --port=9000

```
aps a2a toggle [flags]
```

### Options

```
      --enabled string    Enable (on), disable (off), or toggle (omit or blank)
  -h, --help              help for toggle
      --host string       Listen host (default "127.0.0.1")
      --port string       Listen port (default "8081")
      --protocol string   Protocol binding (jsonrpc, grpc, http) (default "jsonrpc")
      --url string        Public endpoint URL (defaults to http://{host}:{port})
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps a2a](#aps-a2a)	 - Manage A2A (Agent-to-Agent) protocol operations

## aps acp

Manage ACP (Agent Client Protocol) server

### Synopsis

Manage ACP (Agent Client Protocol) server for editor integrations.

The acp command group provides operations for:
- Starting an ACP server for a profile
- Managing ACP sessions
- Configuring ACP settings

### Options

```
  -h, --help   help for acp
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps acp server](#aps-acp-server)	 - Start an ACP server for a profile
* [aps acp toggle](#aps-acp-toggle)	 - Enable or disable ACP for a profile

## aps acp server

Start an ACP server for a profile

### Synopsis

Start an ACP (Agent Client Protocol) server for a profile.

The server communicates with editor clients via JSON-RPC 2.0 over stdio
or WebSocket transport, based on the profile ACP configuration.

Example:
  aps acp server my-profile

```
aps acp server [profile] [flags]
```

### Options

```
  -h, --help   help for server
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps acp](#aps-acp)	 - Manage ACP (Agent Client Protocol) server

## aps acp toggle

Enable or disable ACP for a profile

### Synopsis

Enable or disable ACP (Agent Client Protocol) for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps --profile worker acp toggle                    # Toggle ACP
  aps --profile worker acp toggle --enabled=on       # Force enable
  aps --profile worker acp toggle --enabled=off      # Force disable

```
aps acp toggle [flags]
```

### Options

```
      --enabled string     Enable (on), disable (off), or toggle (omit or blank)
  -h, --help               help for toggle
      --host string        Listen host for network transports (default "127.0.0.1")
      --port string        Listen port for network transports (default "8088")
      --transport string   Transport (stdio, ws, websocket) (default "stdio")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps acp](#aps-acp)	 - Manage ACP (Agent Client Protocol) server

## aps action

Manage and execute profile actions

### Options

```
  -h, --help   help for action
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps action list](#aps-action-list)	 - List available actions for a profile
* [aps action run](#aps-action-run)	 - Run an action
* [aps action show](#aps-action-show)	 - Show details of a specific action

## aps action list

List available actions for a profile

### Synopsis

List every action defined under the named profile's actions/
directory. Actions are profile-scoped scripts that aps can invoke by
id (sh, py, or js runtime, inferred from the entrypoint extension).

Output respects the global --format flag (table|json|yaml). The local
--type flag filters by inferred runtime (sh, py, js). Empty title
fields render as "(no description)" in the table; structured formats
leave them empty.

Read-only: no profile state is mutated. Idempotent across runs.

```
aps action list [profile] [flags]
```

### Options

```
  -h, --help          help for list
      --type string   Filter by runtime type (sh, py, js)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps action](#aps-action)	 - Manage and execute profile actions

## aps action run

Run an action

### Synopsis

Execute the named action script under the named profile. The
action runtime (sh/py/js) is inferred from its entrypoint extension;
aps invokes the corresponding interpreter and streams the action's
stdout/stderr.

Payload sources are mutually exclusive: --payload-file reads the
named file into the action's stdin; --payload-stdin streams the
current process stdin through. Without either flag aps inherits
stdin directly. The inherited --dry-run global previews the
resolved action path without executing.

Mutating: invokes an opaque user-supplied script whose side effects
aps cannot enumerate. Idempotency is conditional on the action.

```
aps action run [profile] [action] [flags]
```

### Options

```
  -h, --help                  help for run
      --payload-file string   File to send to action stdin
      --payload-stdin         Read stdin and forward to action
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps action](#aps-action)	 - Manage and execute profile actions

## aps action show

Show details of a specific action

### Synopsis

Show the resolved metadata for a single action under the named
profile: id, title, inferred runtime type, on-disk path, and whether
the action declares stdin input. Useful before invoking aps action
run to confirm what will execute.

Read-only: loads the action record from the profile's actions/
directory and prints it. Idempotent.

```
aps action show [profile] [action] [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps action](#aps-action)	 - Manage and execute profile actions

## aps adapter

Manage adapters (messengers, protocols, mobile, desktop)

### Synopsis

Manage aps adapter devices — the external transports
(messengers, protocols, mobile, desktop, sense, actuator) the
runtime can talk to. Records live under $APS_DATA_PATH at either
global or profile scope; subcommands cover the full lifecycle:
create / start / stop / status / logs / list, the link parent
(add | list | delete) for profile binding, mobile pairing
(pair | approve | reject | revoke | pending), workspace device
management (attach | detach | presence | permissions), and
messenger-specific helpers (channels | test).

aps messenger is a type-scoped shorthand for the messenger subset
of this tree.

### Options

```
  -h, --help   help for adapter
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps adapter approve](#aps-adapter-approve)	 - Approve a pending mobile device
* [aps adapter attach](#aps-adapter-attach)	 - Attach a device to a workspace
* [aps adapter channels](#aps-adapter-channels)	 - List known channels for a messenger device
* [aps adapter create](#aps-adapter-create)	 - Create a new device
* [aps adapter detach](#aps-adapter-detach)	 - Detach a device from a workspace
* [aps adapter exec](#aps-adapter-exec)	 - Execute a script-strategy adapter action
* [aps adapter link](#aps-adapter-link)	 - Manage device-profile links (add, list, delete)
* [aps adapter list](#aps-adapter-list)	 - List adapter devices
* [aps adapter logs](#aps-adapter-logs)	 - View device logs
* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)
* [aps adapter pair](#aps-adapter-pair)	 - Generate QR code to pair a mobile device
* [aps adapter pending](#aps-adapter-pending)	 - List mobile devices pending approval
* [aps adapter permissions](#aps-adapter-permissions)	 - Manage device permissions in a workspace
* [aps adapter presence](#aps-adapter-presence)	 - Show device presence in a workspace
* [aps adapter reject](#aps-adapter-reject)	 - Reject a pending mobile device
* [aps adapter revoke](#aps-adapter-revoke)	 - Revoke a paired mobile device
* [aps adapter start](#aps-adapter-start)	 - Start a device
* [aps adapter status](#aps-adapter-status)	 - Show device status
* [aps adapter stop](#aps-adapter-stop)	 - Stop a device
* [aps adapter test](#aps-adapter-test)	 - Test the messenger pipeline

## aps adapter approve

Approve a pending mobile device

### Synopsis

Flip a pending mobile device entry to "approved" in the
profile's adapter registry. The device must have previously
registered via the pairing flow (aps adapter pair) and currently
sit in the pending state — list candidates with aps adapter
pending.

Pass a single <device-id> to approve one entry, or use --all to
approve every pending device under the active profile. The --json
flag emits a structured result; --quiet (inherited from root)
suppresses the success line. --profile is inherited from the root
global and is required.

Mutates the local registry only. Idempotent: re-approving an
already-approved device is a no-op. Dry-run is opted out because
preview would only echo the device ID the user already passed.

```
aps adapter approve <device-id> [flags]
```

### Options

```
      --all           Approve all pending devices
  -h, --help          help for approve
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter attach

Attach a device to a workspace

### Synopsis

Attach a device to a workspace with a specified role.

Roles control what the device can do:
  owner        Full access (read, write, execute, manage, sync)
  collaborator Operational access (read, write, execute, sync)
  viewer       Read-only access (read, sync)

```
aps adapter attach <device-id> [flags]
```

### Options

```
  -h, --help          help for attach
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --role string   Device role: owner, collaborator, viewer (default "viewer")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter channels

List known channels for a messenger device

### Synopsis

Lists channels from existing mappings across all profiles for a messenger device.

```
aps adapter channels <messenger> [flags]
```

### Options

```
  -h, --help   help for channels
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter create

Create a new device

### Synopsis

Create a new aps adapter device entry under the active
profile (or globally when no profile is set). The record lives in
$APS_DATA_PATH/profiles/<profile>/adapters/<name>/ (or the global
equivalent) and registers an external transport — messenger,
protocol, mobile, desktop, sense, or actuator — that aps can talk
to. The record holds the device's type, loading strategy
(subprocess / script / builtin), and a manifest scaffold; it does
NOT start the device — pair that with aps adapter start <name>.

If --type is omitted the command launches an interactive huh prompt
listing the implemented types; --strategy defaults to the type's
canonical strategy when omitted. --json emits a structured result.
--profile is inherited from the root global; no explicit flag is
needed when the active profile is already set.

Mints a new local record. Each invocation creates a fresh entry
(not idempotent); dry-run is opted out because the prospective ID
is exactly the <name> argument.

```
aps adapter create <name> [flags]
```

### Options

```
  -h, --help              help for create
      --json              JSON output
  -n, --note string       Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --strategy string   Loading strategy (subprocess, script, builtin)
      --type string       Device type (messenger, protocol, mobile, desktop, sense, actuator, scheduler)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter detach

Detach a device from a workspace

### Synopsis

Detach a device from a workspace, removing all access.

This is a destructive operation. The device will lose access to the
workspace and any pending offline queue entries will be discarded.
Use --force to skip confirmation.

```
aps adapter detach <device-id> [flags]
```

### Options

```
      --force         Skip confirmation prompt
  -h, --help          help for detach
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter exec

Execute a script-strategy adapter action

### Synopsis

Run an action on a script-strategy adapter.

The adapter's manifest defines available actions and their
scripts. Profile email is resolved from the --profile flag.

Examples:
  aps adapter exec email send --profile noor \
    --input to=user@example.com \
    --input subject="Hello" \
    --input body="Message body"

  aps adapter exec email reply --profile noor \
    --input id=7131 \
    --input body="Thanks!"

  aps adapter exec email list --profile noor

  aps adapter exec email read --profile noor \
    --input id=7131

  # Explicit from address (no profile needed)
  aps adapter exec email send --from ops@company.com \
    --input to=user@example.com \
    --input subject="Hello" \
    --input body="Hi"

```
aps adapter exec <adapter> <action> [flags]
```

### Options

```
      --from string         Explicit From address (overrides profile lookup)
  -h, --help                help for exec
  -i, --input stringArray   Action input as key=value (repeatable)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter link

Manage device-profile links (add, list, delete)

### Synopsis

Manage links between adapter devices and profiles.

A "link" is the relationship between a device and a profile; the device
and profile are the parties.

### Options

```
  -h, --help   help for link
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)
* [aps adapter link add](#aps-adapter-link-add)	 - Link a device to a profile
* [aps adapter link delete](#aps-adapter-link-delete)	 - Unlink a device from a profile
* [aps adapter link list](#aps-adapter-link-list)	 - List messenger-profile links

## aps adapter link add

Link a device to a profile

### Synopsis

Attach an existing adapter device to a profile so the
device participates in that profile's runtime. The link is
recorded on the device record and (for messengers) augmented with
channel-action mappings that route inbound messages to skills.
The companion list/delete subcommands are aps adapter link list
and aps adapter link delete.

For messenger devices, --mapping channel=action populates the
routing table (repeatable); --add-mapping and --remove-mapping
mutate a single mapping on an existing link; --default-action sets
the fallback for unmapped channels. Non-messenger devices ignore
the mapping flags. --json emits the structured outcome.
--profile and --dry-run inherit from the root globals; --profile
is required.

Mutates the local registry. Idempotent: re-linking an already-
linked device is a no-op; messenger mapping flags act on the
existing link when one is already present.

```
aps adapter link add <device> [flags]
```

### Options

```
      --add-mapping string      Add a single channel=action mapping to an existing link
      --default-action string   Set default action for unmapped channels
  -h, --help                    help for add
      --json                    JSON output
      --mapping strings         Channel=Action mapping (messenger devices only, repeatable)
  -n, --note string             Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --remove-mapping string   Remove a mapping by channel ID from an existing link
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter link](#aps-adapter-link)	 - Manage device-profile links (add, list, delete)

## aps adapter link delete

Unlink a device from a profile

### Synopsis

Detach an adapter device from a profile, removing the
profile id from the device's linked-profiles list and publishing
an adapter-unlinked event. For messenger devices this also tears
down the routing entry that mapped channels to skills. The device
record itself is preserved — use aps adapter delete (when
available) to remove the device entirely. Companion read is
aps adapter link list.

The command errors when the device is not currently linked to the
named profile. --json emits the structured outcome.
--profile and --dry-run inherit from the root globals; --profile
is required. --dry-run prints the would-be transition without
mutating anything.

Mutates the local registry only. Not naturally idempotent — a
second invocation against an already-unlinked device returns a
"not linked" error so the operator can distinguish drift.

```
aps adapter link delete <device> [flags]
```

### Options

```
  -h, --help          help for delete
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter link](#aps-adapter-link)	 - Manage device-profile links (add, list, delete)

## aps adapter link list

List messenger-profile links

### Synopsis

Lists all messenger-profile links, optionally filtered by profile or messenger.

```
aps adapter link list [flags]
```

### Options

```
  -h, --help               help for list
      --messenger string   Filter by messenger device name
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter link](#aps-adapter-link)	 - Manage device-profile links (add, list, delete)

## aps adapter list

List adapter devices

### Synopsis

Print one row per adapter device known to aps across both
global and profile scopes, with name, type, runtime status, owning
workspace (profile id or "global"), paired-device count, and
last-seen timestamp. Output respects the root --format flag
(table | json | yaml). Local --type and --status filters select
by adapter type and runtime state respectively; --workspace
inherits from the root global and filters by profile id (or the
literal "global" for global-scope adapters).

Read-only: no state mutation. Idempotent.

```
aps adapter list [flags]
```

### Options

```
  -h, --help            help for list
      --status string   Filter by runtime status (running, stopped, failed, ...)
      --type string     Filter by adapter type (messenger, protocol, mobile, ...)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter logs

View device logs

### Synopsis

Print captured stdout/stderr lines from the named adapter
device's runtime. Defaults to the last 20 lines; --tail <N>
controls the line count, --follow / -f streams new lines as they
arrive (blocks until interrupted), and --since <duration> (e.g.
1h, 30m) restricts to lines emitted within the given window. The
device must exist in the registry; logs are sourced from the
manager's per-adapter log buffer.

Read-only: no state mutation. Idempotent.

```
aps adapter logs <name> [flags]
```

### Options

```
  -f, --follow         Follow log output (stream)
  -h, --help           help for logs
      --since string   Show logs since duration (e.g., 1h, 30m)
      --tail int       Number of lines to show (default 20)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter messenger

Messenger device commands (alias for 'aps device' with messenger context)

### Synopsis

Messenger commands provide shortcuts for common messenger device operations.

These commands are equivalent to their 'aps device' counterparts but
pre-filtered for messenger-type devices.

### Options

```
  -h, --help   help for messenger
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)
* [aps adapter messenger channels](#aps-adapter-messenger-channels)	 - List known channels for a messenger device
* [aps adapter messenger create](#aps-adapter-messenger-create)	 - Create a new device
* [aps adapter messenger link](#aps-adapter-messenger-link)	 - Manage device-profile links (add, list, delete)
* [aps adapter messenger list](#aps-adapter-messenger-list)	 - List messenger adapters
* [aps adapter messenger logs](#aps-adapter-messenger-logs)	 - View device logs
* [aps adapter messenger start](#aps-adapter-messenger-start)	 - Start a device
* [aps adapter messenger status](#aps-adapter-messenger-status)	 - Show device status
* [aps adapter messenger stop](#aps-adapter-messenger-stop)	 - Stop a device
* [aps adapter messenger test](#aps-adapter-messenger-test)	 - Test the messenger pipeline

## aps adapter messenger channels

List known channels for a messenger device

### Synopsis

Lists channels from existing mappings across all profiles for a messenger device.

```
aps adapter messenger channels <messenger> [flags]
```

### Options

```
  -h, --help   help for channels
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger create

Create a new device

### Synopsis

Create a new aps adapter device entry under the active
profile (or globally when no profile is set). The record lives in
$APS_DATA_PATH/profiles/<profile>/adapters/<name>/ (or the global
equivalent) and registers an external transport — messenger,
protocol, mobile, desktop, sense, or actuator — that aps can talk
to. The record holds the device's type, loading strategy
(subprocess / script / builtin), and a manifest scaffold; it does
NOT start the device — pair that with aps adapter start <name>.

If --type is omitted the command launches an interactive huh prompt
listing the implemented types; --strategy defaults to the type's
canonical strategy when omitted. --json emits a structured result.
--profile is inherited from the root global; no explicit flag is
needed when the active profile is already set.

Mints a new local record. Each invocation creates a fresh entry
(not idempotent); dry-run is opted out because the prospective ID
is exactly the <name> argument.

```
aps adapter messenger create <name> [flags]
```

### Options

```
  -h, --help              help for create
      --json              JSON output
  -n, --note string       Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --strategy string   Loading strategy (subprocess, script, builtin)
      --type string       Device type (messenger, protocol, mobile, desktop, sense, actuator, scheduler)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger link

Manage device-profile links (add, list, delete)

### Synopsis

Manage links between adapter devices and profiles.

A "link" is the relationship between a device and a profile; the device
and profile are the parties.

### Options

```
  -h, --help   help for link
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)
* [aps adapter messenger link add](#aps-adapter-messenger-link-add)	 - Link a device to a profile
* [aps adapter messenger link delete](#aps-adapter-messenger-link-delete)	 - Unlink a device from a profile
* [aps adapter messenger link list](#aps-adapter-messenger-link-list)	 - List messenger-profile links

## aps adapter messenger link add

Link a device to a profile

### Synopsis

Attach an existing adapter device to a profile so the
device participates in that profile's runtime. The link is
recorded on the device record and (for messengers) augmented with
channel-action mappings that route inbound messages to skills.
The companion list/delete subcommands are aps adapter link list
and aps adapter link delete.

For messenger devices, --mapping channel=action populates the
routing table (repeatable); --add-mapping and --remove-mapping
mutate a single mapping on an existing link; --default-action sets
the fallback for unmapped channels. Non-messenger devices ignore
the mapping flags. --json emits the structured outcome.
--profile and --dry-run inherit from the root globals; --profile
is required.

Mutates the local registry. Idempotent: re-linking an already-
linked device is a no-op; messenger mapping flags act on the
existing link when one is already present.

```
aps adapter messenger link add <device> [flags]
```

### Options

```
      --add-mapping string      Add a single channel=action mapping to an existing link
      --default-action string   Set default action for unmapped channels
  -h, --help                    help for add
      --json                    JSON output
      --mapping strings         Channel=Action mapping (messenger devices only, repeatable)
  -n, --note string             Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --remove-mapping string   Remove a mapping by channel ID from an existing link
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger link](#aps-adapter-messenger-link)	 - Manage device-profile links (add, list, delete)

## aps adapter messenger link delete

Unlink a device from a profile

### Synopsis

Detach an adapter device from a profile, removing the
profile id from the device's linked-profiles list and publishing
an adapter-unlinked event. For messenger devices this also tears
down the routing entry that mapped channels to skills. The device
record itself is preserved — use aps adapter delete (when
available) to remove the device entirely. Companion read is
aps adapter link list.

The command errors when the device is not currently linked to the
named profile. --json emits the structured outcome.
--profile and --dry-run inherit from the root globals; --profile
is required. --dry-run prints the would-be transition without
mutating anything.

Mutates the local registry only. Not naturally idempotent — a
second invocation against an already-unlinked device returns a
"not linked" error so the operator can distinguish drift.

```
aps adapter messenger link delete <device> [flags]
```

### Options

```
  -h, --help          help for delete
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger link](#aps-adapter-messenger-link)	 - Manage device-profile links (add, list, delete)

## aps adapter messenger link list

List messenger-profile links

### Synopsis

Lists all messenger-profile links, optionally filtered by profile or messenger.

```
aps adapter messenger link list [flags]
```

### Options

```
  -h, --help               help for list
      --messenger string   Filter by messenger device name
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger link](#aps-adapter-messenger-link)	 - Manage device-profile links (add, list, delete)

## aps adapter messenger list

List messenger adapters

### Synopsis

Print one row per messenger-type adapter device with
name, platform (telegram, slack, discord, ...), runtime status,
owning profile, and channel-mapping count. Equivalent to
aps adapter list --type=messenger but type-scoped at the parent
group level. Output respects the root --format flag
(table | json | yaml). The local --platform / --status flags
filter by platform string and runtime state; --profile inherits
from the root global.

Read-only: no state mutation. Idempotent.

```
aps adapter messenger list [flags]
```

### Options

```
  -h, --help              help for list
      --platform string   Filter by messenger platform (telegram, slack, discord, ...)
      --status string     Filter by runtime status (running, stopped, failed, ...)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger logs

View device logs

### Synopsis

Print captured stdout/stderr lines from the named adapter
device's runtime. Defaults to the last 20 lines; --tail <N>
controls the line count, --follow / -f streams new lines as they
arrive (blocks until interrupted), and --since <duration> (e.g.
1h, 30m) restricts to lines emitted within the given window. The
device must exist in the registry; logs are sourced from the
manager's per-adapter log buffer.

Read-only: no state mutation. Idempotent.

```
aps adapter messenger logs <name> [flags]
```

### Options

```
  -f, --follow         Follow log output (stream)
  -h, --help           help for logs
      --since string   Show logs since duration (e.g., 1h, 30m)
      --tail int       Number of lines to show (default 20)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger start

Start a device

### Synopsis

Spawn the runtime for the named adapter device under
the manager. The exact mechanics depend on the device's loading
strategy: subprocess strategies fork the executable named by the
manifest, script strategies invoke the script binary, and builtin
strategies attach the in-process handler. Pair with
aps adapter create <name> to mint a device record first, and
aps adapter stop <name> to terminate.

The command prints a "Starting <name>... running (PID <pid>)"
progress line on a TTY, or emits structured runtime state with
--json. On failure it surfaces the error and — for messenger
devices missing a token — hints the corresponding
aps secrets set <NAME>_TOKEN command. The Already-Running case
returns success.

Mutates local runtime state. Idempotent only in the
already-running sense (no double-spawn). Dry-run is opted out
because previewing would have to bisect the spawn-and-wait path
that is the operation itself.

```
aps adapter messenger start <name> [flags]
```

### Options

```
  -h, --help          help for start
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger status

Show device status

### Synopsis

Print the resolved state of the named adapter device:
type, scope (global vs profile-bound), runtime state, health
(when running), uptime, PID, loading strategy, creation timestamp,
and the list of profiles the device is linked to. Failed-state
devices also show the captured LastError and a hint pointing at
aps adapter logs <name>. --json emits the same fields as a
structured payload; --verbose (root global) is plumbed through
for future expansion. Output respects only the local --json flag
because the human view is hand-rendered with styled badges.

Read-only: no state mutation. Idempotent.

```
aps adapter messenger status <name> [flags]
```

### Options

```
  -h, --help   help for status
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger stop

Stop a device

### Synopsis

Terminate the runtime for the named adapter device.
Default behavior sends SIGTERM and waits for graceful shutdown;
--force escalates to SIGKILL. When the device is linked to one
or more profiles, the command prints the linked-profile list and
prompts for confirmation before stopping (suppressed under --json
or --force).

--json emits the structured outcome instead of the human
"Stopping <name>... stopped" progress line. --dry-run (root
global) prints what would happen — type, current state, PID, and
the profiles that would be unaffected — without sending any
signal.

Mutates local runtime state. Idempotent in the already-stopped
sense; force-stopping a healthy adapter is destructive only for
the in-flight work the device was handling, not for the device
record itself (use aps adapter delete to remove the record).

```
aps adapter messenger stop <name> [flags]
```

### Options

```
      --force         Force stop (SIGKILL)
  -h, --help          help for stop
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter messenger test

Test the messenger pipeline

### Synopsis

Tests the full messenger pipeline: normalize, route, execute, denormalize, send.

```
aps adapter messenger test <messenger> [flags]
```

### Options

```
      --channel string     Channel ID (defaults to first mapped channel)
  -h, --help               help for test
      --json               JSON output
      --message string     Test message content (default "test query")
      --send               Actually deliver the test message
      --timeout duration   Pipeline timeout (default 30s)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter messenger](#aps-adapter-messenger)	 - Messenger device commands (alias for 'aps device' with messenger context)

## aps adapter pair

Generate QR code to pair a mobile device

### Synopsis

Start a device server and display a QR code for mobile device pairing.

The QR code contains connection details that the mobile APS app uses to
establish a WebSocket connection to this profile.

```
aps adapter pair [flags]
```

### Options

```
      --bind-addr string       Bind address (auto-detected if not set)
      --capabilities strings   Device capabilities (default: run:stateless,run:streaming,monitor:sessions)
      --code-only              Show pairing code only, no QR
      --expires string         Device token expiry (e.g., 14d, 30d) (default "14d")
  -h, --help                   help for pair
      --json                   JSON output
      --no-qr                  Skip QR code display (accessibility, screen readers)
  -n, --note string            Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --port int               Server port (default 8443)
      --qr-expires string      QR code expiry (e.g., 15m, 30m) (default "15m")
      --qr-output string       Save QR code as PNG to file
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter pending

List mobile devices pending approval

### Synopsis

Print one row per mobile device that has registered via
the pairing flow (aps adapter pair) but has not yet been approved
or rejected. Each row shows the device id, a time-ago badge for
the pairing request, and a composite "name, os version"
device-info column. The default output is a styled table;
--json emits the structured registry rows.

After listing the command echoes the resolved approve / reject
commands (pre-filled with --profile when set) so the operator can
copy-paste the next action. --profile is inherited from the root
global and scopes the listing to one profile's pending queue when
provided.

Read-only: no state mutation. Idempotent.

```
aps adapter pending [flags]
```

### Options

```
  -h, --help   help for pending
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter permissions

Manage device permissions in a workspace

### Synopsis

Group device-permission operations against a workspace.
The single leaf under this parent is aps adapter permissions set,
which writes or inspects the permission bits a device holds in a
named workspace. Companion read is the device-level
aps adapter status / show.

### Options

```
  -h, --help   help for permissions
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)
* [aps adapter permissions set](#aps-adapter-permissions-set)	 - Set device permissions in a workspace

## aps adapter permissions set

Set device permissions in a workspace

### Synopsis

Set or view permissions for a device in a workspace.

Use --role for quick role-based configuration (covers 90% of cases).
Use fine-grained flags to override individual permissions.

Roles:
  owner        read, write, execute, manage, sync
  collaborator read, write, execute, sync
  viewer       read, sync

Use --show to display current permissions without making changes.

```
aps adapter permissions set <device-id> [flags]
```

### Options

```
      --can-execute      Override: allow execute access
      --can-manage       Override: allow manage access
      --can-write        Override: allow write access
  -h, --help             help for set
      --json             JSON output
      --rate-limit int   Rate limit (requests per minute, 0 = unlimited)
      --role string      Set role: owner, collaborator, viewer
      --show             Show current permissions without changes
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter permissions](#aps-adapter-permissions)	 - Manage device permissions in a workspace

## aps adapter presence

Show device presence in a workspace

### Synopsis

Show the current presence status of all devices linked to a workspace.

Displays device status, last heartbeat, and sync lag information.

```
aps adapter presence [workspace-id] [flags]
```

### Options

```
  -h, --help   help for presence
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter reject

Reject a pending mobile device

### Synopsis

Remove a pending mobile device entry from the profile's
adapter registry, denying the pairing request. The device must
currently sit in the pending state (list candidates with
aps adapter pending); attempting to reject a non-pending entry
returns the registry error verbatim. --profile is inherited from
the root global and is required. --json emits a structured
outcome; --quiet (root global) suppresses the success line.

Mutates the local registry only. Idempotent: re-rejecting an
already-rejected (i.e. removed) device returns a not-found error.
Dry-run is opted out because preview would only echo the device
ID the user already passed.

```
aps adapter reject <device-id> [flags]
```

### Options

```
  -h, --help          help for reject
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter revoke

Revoke a paired mobile device

### Synopsis

Revoke a mobile device's access token, disconnecting it immediately.

The device must re-pair via a new QR code to reconnect.

```
aps adapter revoke [device-id] [flags]
```

### Options

```
      --all           Revoke all devices
      --force         Skip confirmation
  -h, --help          help for revoke
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter start

Start a device

### Synopsis

Spawn the runtime for the named adapter device under
the manager. The exact mechanics depend on the device's loading
strategy: subprocess strategies fork the executable named by the
manifest, script strategies invoke the script binary, and builtin
strategies attach the in-process handler. Pair with
aps adapter create <name> to mint a device record first, and
aps adapter stop <name> to terminate.

The command prints a "Starting <name>... running (PID <pid>)"
progress line on a TTY, or emits structured runtime state with
--json. On failure it surfaces the error and — for messenger
devices missing a token — hints the corresponding
aps secrets set <NAME>_TOKEN command. The Already-Running case
returns success.

Mutates local runtime state. Idempotent only in the
already-running sense (no double-spawn). Dry-run is opted out
because previewing would have to bisect the spawn-and-wait path
that is the operation itself.

```
aps adapter start <name> [flags]
```

### Options

```
  -h, --help          help for start
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter status

Show device status

### Synopsis

Print the resolved state of the named adapter device:
type, scope (global vs profile-bound), runtime state, health
(when running), uptime, PID, loading strategy, creation timestamp,
and the list of profiles the device is linked to. Failed-state
devices also show the captured LastError and a hint pointing at
aps adapter logs <name>. --json emits the same fields as a
structured payload; --verbose (root global) is plumbed through
for future expansion. Output respects only the local --json flag
because the human view is hand-rendered with styled badges.

Read-only: no state mutation. Idempotent.

```
aps adapter status <name> [flags]
```

### Options

```
  -h, --help   help for status
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter stop

Stop a device

### Synopsis

Terminate the runtime for the named adapter device.
Default behavior sends SIGTERM and waits for graceful shutdown;
--force escalates to SIGKILL. When the device is linked to one
or more profiles, the command prints the linked-profile list and
prompts for confirmation before stopping (suppressed under --json
or --force).

--json emits the structured outcome instead of the human
"Stopping <name>... stopped" progress line. --dry-run (root
global) prints what would happen — type, current state, PID, and
the profiles that would be unaffected — without sending any
signal.

Mutates local runtime state. Idempotent in the already-stopped
sense; force-stopping a healthy adapter is destructive only for
the in-flight work the device was handling, not for the device
record itself (use aps adapter delete to remove the record).

```
aps adapter stop <name> [flags]
```

### Options

```
      --force         Force stop (SIGKILL)
  -h, --help          help for stop
      --json          JSON output
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps adapter test

Test the messenger pipeline

### Synopsis

Tests the full messenger pipeline: normalize, route, execute, denormalize, send.

```
aps adapter test <messenger> [flags]
```

### Options

```
      --channel string     Channel ID (defaults to first mapped channel)
  -h, --help               help for test
      --json               JSON output
      --message string     Test message content (default "test query")
      --send               Actually deliver the test message
      --timeout duration   Pipeline timeout (default 30s)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps adapter](#aps-adapter)	 - Manage adapters (messengers, protocols, mobile, desktop)

## aps alias

Manage command aliases

### Synopsis

Manage user-defined command aliases (YAML-backed) and
generate per-profile shell aliases.

Subcommands:
  add | list | remove   Manage YAML aliases at $XDG_CONFIG_HOME/aps/aliases.yaml
  shell                 Print shell-source-able alias lines (legacy `aps alias`)

Note: the bare `aps alias` form previously printed shell aliases.
That behaviour has moved to `aps alias shell`; this notice will be
removed after one release.

```
aps alias [flags]
```

### Options

```
  -h, --help   help for alias
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps alias add](#aps-alias-add)	 - Add or update an alias
* [aps alias delete](#aps-alias-delete)	 - Delete an alias
* [aps alias list](#aps-alias-list)	 - List aliases
* [aps alias shell](#aps-alias-shell)	 - Generate shell aliases for profiles

## aps alias add

Add or update an alias

### Synopsis

Persist a new alias (or replace an existing one) in the YAML store and register a runtime shim so subsequent invocations dispatch to <target>. <target> may include flags, captured as a single shell-quoted string.

```
aps alias add <name> <target...> [flags]
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps alias](#aps-alias)	 - Manage command aliases

## aps alias delete

Delete an alias

### Synopsis

Remove an alias from the YAML store and tear down its runtime shim. Idempotent: deleting a missing alias returns success. The legacy `remove`/`rm` aliases keep working through the deprecation window.

```
aps alias delete <name> [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps alias](#aps-alias)	 - Manage command aliases

## aps alias list

List aliases

### Synopsis

Print the active alias table — both YAML-backed entries and runtime-registered shims — in the active --format. Read-only: no state mutation.

```
aps alias list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps alias](#aps-alias)	 - Manage command aliases

## aps alias shell

Generate shell aliases for profiles

### Synopsis

Generate shell aliases for all available profiles.
Add the following to your shell configuration file:

  eval "$(aps alias shell)"


```
aps alias shell [flags]
```

### Options

```
  -h, --help   help for shell
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps alias](#aps-alias)	 - Manage command aliases

## aps bundle

Manage capability bundles

### Options

```
  -h, --help   help for bundle
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps bundle create](#aps-bundle-create)	 - Scaffold a new bundle file in the user bundle directory
* [aps bundle delete](#aps-bundle-delete)	 - Delete a user-defined bundle (refuses on built-ins)
* [aps bundle edit](#aps-bundle-edit)	 - Open a bundle in $EDITOR; copies built-in to user dir first
* [aps bundle list](#aps-bundle-list)	 - List built-in and user bundles
* [aps bundle show](#aps-bundle-show)	 - Print full bundle definition as YAML
* [aps bundle validate](#aps-bundle-validate)	 - Validate a bundle YAML file and report issues

## aps bundle create

Scaffold a new bundle file in the user bundle directory

### Synopsis

Scaffold a new bundle YAML file at aps/bundles/<name>.yaml
under the user config directory. The scaffold is a minimal stub
with name, empty description, version "1.0", and an empty
capabilities list — ready to be hand-edited or passed to aps
bundle edit. The directory is created on demand.

By default the command refuses to overwrite an existing file at
that path; pass --force to replace it. Pair with aps bundle edit
to open the new file in $EDITOR, or aps bundle validate to check
the YAML once populated. --dry-run is opted out because the
destination is a pure function of the name argument.

```
aps bundle create <name> [flags]
```

### Options

```
      --force         Overwrite existing bundle file
  -h, --help          help for create
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps bundle delete

Delete a user-defined bundle (refuses on built-ins)

### Synopsis

Remove the user bundle file at aps/bundles/<name>.yaml under
the user config directory. The command refuses to touch built-in
bundles — if the named bundle is a built-in with no user override,
it suggests aps bundle edit to create an override first. The user
is prompted for confirmation unless --force is passed.

Destructive: the user bundle YAML file is removed irreversibly.
The destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would only restate the
bundle name. Idempotent on already-absent records — re-running on
a missing user bundle reports the bundle as not found.

```
aps bundle delete <name> [flags]
```

### Options

```
      --force         Skip confirmation prompt
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps bundle edit

Open a bundle in $EDITOR; copies built-in to user dir first

### Synopsis

Open the named bundle in $EDITOR (or vi when $EDITOR is unset).
If no user override exists at aps/bundles/<name>.yaml under the
user config directory, the built-in bundle of the same name is
copied there first so the edit lands in the user-owned override
rather than the kit-shipped source. A "Note: built-in bundle …
copied to …" line surfaces when that copy happens.

The command shells out to the editor and waits for it to exit;
whether the file changed is up to the user, so the effect is
idempotency-conditional. --dry-run is opted out because the
operation is fundamentally an interactive editor session — preview
would either suppress the editor or describe a write the user
hasn't authored yet.

```
aps bundle edit <name> [flags]
```

### Options

```
  -h, --help          help for edit
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps bundle list

List built-in and user bundles

### Synopsis

List every capability bundle aps can resolve — built-in
bundles shipped with the binary plus user-authored bundles under
the user config directory (aps/bundles/<name>.yaml). Each row
reports the bundle name, source ("built-in", "user", or "user
(overrides built-in)"), member capability count, tags, and the
bundle description.

The output respects the global --format flag (table|json|yaml) and
the local filter flags: --tag selects by tag value, --builtin or
--user scope the source (mutually exclusive). Read-only: no state
mutation. Idempotent.

```
aps bundle list [flags]
```

### Options

```
      --builtin      Show only built-in bundles
  -h, --help         help for list
      --tag string   Filter to bundles carrying this tag
      --user         Show only user bundles (incl. overrides)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps bundle show

Print full bundle definition as YAML

### Synopsis

Print the full bundle definition as YAML — name, version,
description, tags, member capabilities, and any extends/inherits
chain. The lookup resolves through the same registry that aps
bundle list uses: user overrides at aps/bundles/<name>.yaml under
the user config directory win over built-ins shipped with the
binary. Pass --resolved to apply the inheritance chain and print
the merged result instead of the raw record.

Read-only: no state mutation. Idempotent.

```
aps bundle show <name> [flags]
```

### Options

```
  -h, --help       help for show
      --resolved   Apply inheritance and print merged result
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps bundle validate

Validate a bundle YAML file and report issues

### Synopsis

Validate a bundle YAML file at the given path. Parses the file
into the bundle struct (reporting YAML syntax errors with file
context), then runs the registry validator against it — required
fields, valid capability references, well-formed inheritance, and
any other invariants the registry enforces. Errors are surfaced
with the file path prefixed for grepability.

Read-only: no state mutation. Idempotent. The <file> argument is a
literal filesystem path and is not resolved through the bundle
name registry — pass aps/bundles/<name>.yaml under the user config
directory when validating a user bundle.

```
aps bundle validate <file> [flags]
```

### Options

```
  -h, --help   help for validate
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps bundle](#aps-bundle)	 - Manage capability bundles

## aps capability

Manage capabilities (tools, configs, dotfiles)

### Options

```
  -h, --help   help for capability
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps capability adopt](#aps-capability-adopt)	 - Adopt an existing file/dir (move to APS + symlink back)
* [aps capability delete](#aps-capability-delete)	 - Delete a capability
* [aps capability disable](#aps-capability-disable)	 - Disable a capability on a profile
* [aps capability enable](#aps-capability-enable)	 - Enable a capability on a profile
* [aps capability install](#aps-capability-install)	 - Install a capability from a source directory or URL
* [aps capability link](#aps-capability-link)	 - Symlink a capability to a target path
* [aps capability list](#aps-capability-list)	 - List all capabilities (builtin + external)
* [aps capability patterns](#aps-capability-patterns)	 - Smart patterns + builtin capabilities
* [aps capability show](#aps-capability-show)	 - Show capability details
* [aps capability watch](#aps-capability-watch)	 - Watch an external file (symlink into APS)

## aps capability adopt

Adopt an existing file/dir (move to APS + symlink back)

### Synopsis

Bring an existing file or directory under aps management by
moving it into $APS_DATA_PATH/capabilities/<name>/ and dropping a
symlink at the original <path> that points back to the new home.
The original location keeps working because the symlink remains
valid, while the canonical copy lives under aps and can be linked
into other profiles or hosts.

Mints a new local record. Not idempotent — re-running with the same
path after adoption either fails or shadows the prior record. Pair
with aps capability link to add more symlinks elsewhere, or aps
capability delete to undo. --dry-run is opted out because the
destination path is a pure function of --name.

```
aps capability adopt <path> --name <name> [flags]
```

### Options

```
  -h, --help          help for adopt
      --name string   Name of capability
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability delete

Delete a capability

### Synopsis

Remove the on-disk capability directory at
$APS_DATA_PATH/capabilities/<name>/ along with any symlinks it
holds. Builtins cannot be deleted — use aps capability disable on a
profile to drop the linkage. If the capability has active links to
external targets the command refuses unless --force is passed, since
deleting an active source leaves dangling symlinks at the targets.

Destructive: the on-disk directory and its symlinks are removed
irreversibly. The destructive-token confirmation flow gates the
apply path, and --dry-run is opted out because preview would only
restate the capability name. Idempotent on already-absent records.

```
aps capability delete <name> [flags]
```

### Options

```
      --force         Skip link warning
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability disable

Disable a capability on a profile

### Synopsis

Drop <capability> from the capabilities list of the named
profile at $APS_DATA_PATH/profiles/<profile>/. Removes only the
profile-level linkage — the underlying capability record under
$APS_DATA_PATH/capabilities/<name>/ is left intact and remains
available to other profiles. Not-enabled pairings are a no-op
that prints a dim notice.

Local write to the profile manifest only; pair with aps capability
enable to restore. Idempotent on the pair. --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.

```
aps capability disable <profile> <capability> [flags]
```

### Options

```
  -h, --help          help for disable
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability enable

Enable a capability on a profile

### Synopsis

Add <capability> to the capabilities list of the named profile at
$APS_DATA_PATH/profiles/<profile>/. The capability must already
exist as a builtin or be installed under $APS_DATA_PATH/capabilities/;
the profile must exist as well. Already-enabled pairings are a no-op
that prints a dim notice and leaves the manifest untouched.

Local write to the profile manifest only; pair with aps capability
disable to undo. Idempotent on the pair. --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.

```
aps capability enable <profile> <capability> [flags]
```

### Options

```
  -h, --help          help for enable
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability install

Install a capability from a source directory or URL

### Synopsis

Copy a capability bundle from the <source> argument into the
user capability tree at $APS_DATA_PATH/capabilities/<name>/. The
source may be a local directory or a URL the underlying installer
recognises; the destination directory name comes from --name. After
copy, the capability is discoverable by aps capability list with
source=external and can be linked into a profile via aps capability
enable <profile> <name>.

Mints a new local record. Each invocation copies into the destination
under --name (not idempotent); preview via --dry-run is opted out
because the destination path is a pure function of the --name flag.

```
aps capability install <source> --name <name> [flags]
```

### Options

```
  -h, --help          help for install
      --name string   Name of the capability
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability link

Symlink a capability to a target path

### Synopsis

Create a symlink at --target pointing back at the capability
stored under $APS_DATA_PATH/capabilities/<name>/. When --target is
omitted, the command consults the smart-pattern registry (see aps
capability patterns list) and, if <name> matches a known tool,
links into that tool's conventional location.

Local write of a single filesystem symlink. Idempotent on the
source/target pair. --dry-run is opted out because the source and
target paths are fully determined by --name and --target; preview
would only echo the inputs. Use --note to attach an audit reason
that flows to the event bus alongside the mutation.

```
aps capability link <name> [--target <path>] [flags]
```

### Options

```
  -h, --help            help for link
  -n, --note string     Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --target string   Target path for symlink
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability list

List all capabilities (builtin + external)

### Synopsis

List every capability discoverable by aps — both kit-shipped
builtins and external capabilities installed under the user's
$APS_DATA_PATH/capabilities/ tree. Each row reports the capability
name, source ("builtin" or "external"), type, on-disk path, tags,
and a description sourced from the capability manifest.

The output respects the global --format flag (table|json|yaml) and
the local filter flags: --tag selects by tag value, --builtin or
--external scope the source (mutually exclusive), and --enabled-on
=<profile> limits to capabilities linked to that profile.

Read-only: no state mutation. Idempotent.

```
aps capability list [flags]
```

### Options

```
      --builtin             Show only builtin capabilities
      --enabled-on string   Show only capabilities linked to the given profile id
      --external            Show only external (installed) capabilities
  -h, --help                help for list
      --tag string          Filter to capabilities carrying this tag
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability patterns

Smart patterns + builtin capabilities

### Options

```
  -h, --help   help for patterns
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)
* [aps capability patterns list](#aps-capability-patterns-list)	 - List smart patterns

## aps capability patterns list

List smart patterns

### Synopsis

List the smart-pattern registry — well-known external tools
(e.g. windsurf, claude) and the conventional file path each tool
expects in a user repository. aps capability link and aps capability
watch consult this registry when --target/--tool is used so callers
can opt into a known mapping instead of spelling out paths by hand.

Read-only: no state mutation. Idempotent. Output respects the
global --format flag (table|json|yaml).

```
aps capability patterns list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability patterns](#aps-capability-patterns)	 - Smart patterns + builtin capabilities

## aps capability show

Show capability details

### Synopsis

Print the full record for a single capability — name, kind
(builtin vs external), type, on-disk path, description, source URL,
install timestamp, and the set of active links the capability holds.
Builtins resolve from the kit-shipped registry; externals load from
$APS_DATA_PATH/capabilities/<name>/. The trailing block lists every
profile that has the capability linked via aps capability enable.

Read-only: no state mutation. Idempotent.

```
aps capability show <name> [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps capability watch

Watch an external file (symlink into APS)

### Synopsis

Install a filesystem watcher that mirrors an externally-managed
file or directory into the aps capability tree. The watcher
symlinks the source into $APS_DATA_PATH/capabilities/<name>/ and
keeps the link in sync as the source changes. The source path comes
from <path> argument or --tool <tool>; --tool consults the
smart-pattern registry so well-known tools resolve to their
conventional default paths in the current working directory.

Long-running local mutation: the watcher persists until the process
exits, rewriting symlinks on every change event. --dry-run is opted
out because there is no batch boundary on which to scope a preview.
Use --note to attach an audit reason that flows to the event bus.

```
aps capability watch <path> --name <name> | --tool <tool> [flags]
```

### Options

```
  -h, --help          help for watch
      --name string   Name of capability
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --tool string   Smart tool name (e.g. windsurf)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps capability](#aps-capability)	 - Manage capabilities (tools, configs, dotfiles)

## aps chat

Chat with a profile-backed assistant

### Synopsis

Open an interactive chat session against the assistant
configured by the given aps profile. The single-profile mode is
the default: aps chat <profile-id> loads the profile, instantiates
its configured engine, and drops the caller into a streaming TUI
that appends each turn to a SessionTypeChat session in the local
registry. The session ID, transcript, and metadata persist across
exits, so the same conversation can be resumed later.

Multi-profile mode activates when --invite is passed: the named
profiles join as additional participants and a round-robin turn
policy selects the active speaker. --invite accepts a
comma-separated list or repeats. --once <prompt> bypasses the TUI
entirely — the prompt is appended as one user turn, the assistant
response is printed to stdout, and the command exits, making
chat scriptable from non-interactive callers. --attach <id> resumes
an existing chat session instead of minting a fresh one;
--no-stream disables token streaming; --model overrides the
profile's configured chat model; --max-auto-turns caps autonomous
turns before returning control to the human.

Each invocation appends a turn to the underlying SessionTypeChat
session — write-local and non-idempotent. --dry-run is opted out
because the operation is an interactive REPL whose effects (LLM
turns, transcript appends) are user-driven and unbounded;
previewing would have to fake the human input that drives them.

```
aps chat <profile-id> [flags]
```

### Options

```
      --attach string        Attach to an existing chat session
      --effort string        Override the reasoning effort (minimal|low|medium|high|xhigh)
  -h, --help                 help for chat
      --invite strings       Invite additional profile IDs (comma-separated or repeatable)
      --max-auto-turns int   Maximum autonomous turns before returning control to the human (default 10)
      --max-tokens int       Override the maximum response tokens
      --model string         Override the configured chat model
      --no-stream            Disable streaming responses
      --once string          Send one prompt, print the assistant response, and exit
      --temperature float    Override the sampling temperature (0-2)
      --verbosity string     Override the response verbosity (low|medium|high)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps completion

Generate completion script

### Synopsis

To load completions:

Bash:
  $ source <(aps completion bash)

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ aps completion zsh > "${fpath[1]}/_aps"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ aps completion fish | source

PowerShell:
  PS> aps completion powershell | Out-String | Invoke-Expression


```
aps completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps config

Inspect aps configuration

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps config path](#aps-config-path)	 - Print the highest-precedence existing aps config file
* [aps config paths](#aps-config-paths)	 - Print the full aps config resolution chain

## aps config path

Print the highest-precedence existing aps config file

### Synopsis

Print the single config file that aps would load.

The resolution chain is walked highest-precedence first (cwd, project,
workspace, user, system, default). The first rung whose file exists is
printed. If no rung exists, the command prints "no config file found
in resolution chain" to stderr and exits 1.

```
aps config path [flags]
```

### Options

```
      --from string   Resolve from this directory instead of os.Getwd()
  -h, --help          help for path
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps config](#aps-config)	 - Inspect aps configuration

## aps config paths

Print the full aps config resolution chain

### Synopsis

Print every rung of the aps config resolution
chain, highest-precedence first. Each entry includes the source label
(cwd|project|workspace|user|system|default), scope, file path, and
whether the file exists on disk.

In text format (default), one path is printed per line. JSON and YAML
emit the full ResolvedPath shape (path, source, scope, exists) so
callers can drive scripts off precedence metadata.

```
aps config paths [flags]
```

### Options

```
      --from string   Resolve from this directory instead of os.Getwd()
  -h, --help          help for paths
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps config](#aps-config)	 - Inspect aps configuration

## aps contact

Manage contacts via adapter

### Synopsis

Manage contacts through the contacts adapter.

Dispatches to the configured contacts adapter backend
(e.g. cardamum for CardDAV).

### Options

```
  -h, --help   help for contact
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps contact add](#aps-contact-add)	 - Add a new contact
* [aps contact delete](#aps-contact-delete)	 - Delete a contact
* [aps contact find](#aps-contact-find)	 - Search contacts
* [aps contact list](#aps-contact-list)	 - List all contacts
* [aps contact note](#aps-contact-note)	 - Append note to contact
* [aps contact show](#aps-contact-show)	 - Show contact detail
* [aps contact update](#aps-contact-update)	 - Update contact fields

## aps contact add

Add a new contact

### Synopsis

Add a new contact to the configured contacts adapter under the
active profile. The email argument is required; optional fields
(--name, --org, --phone, --note, --addressbook) populate the
corresponding vCard properties on the new card.

The contact id is minted by the upstream provider (e.g. cardamum
returns a freshly-issued UID), so --dry-run is opted out — aps
cannot preview an ID the provider has not yet assigned. Each
invocation creates a fresh card; not idempotent. Use aps contact
find to look up an existing card before adding to avoid duplicates.

```
aps contact add <email> [flags]
```

### Options

```
      --addressbook string   Addressbook ID
  -h, --help                 help for add
      --name string          Contact name
      --note string          Note
      --org string           Organization
      --phone string         Phone number
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact delete

Delete a contact

### Synopsis

Delete a contact from the configured contacts adapter by id. The
operation forwards to the upstream addressbook's delete endpoint;
the local effect on aps state is none, but the upstream card is
removed.

Delete-by-id is idempotent at the wire level: repeating the call
with a missing id is a no-op on the provider side. --dry-run is
opted out because the preview would only restate the id argument.
Pair with aps contact find to confirm the target id before
running.

```
aps contact delete <id> [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact find

Search contacts

### Synopsis

Search contacts via the configured contacts adapter. The query
argument is forwarded verbatim to the adapter, which decides what
fields to match (name, email, org, etc.). Output is the raw adapter
response.

The --profile global selects which profile's adapter configuration
handles the search. Read-only: idempotent.

```
aps contact find <query> [flags]
```

### Options

```
  -h, --help   help for find
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact list

List all contacts

### Synopsis

List every contact reachable through the configured contacts
adapter (e.g. cardamum/CardDAV) under the active profile. The
adapter is invoked once per call and the resulting cards are
projected into a uniform row shape: ID, Name (FN), Email, Org,
Phone, Addressbook.

Filters: --addressbook scopes to a single addressbook ID; --org
filters by ORG value; --has-email keeps cards that have at least
one EMAIL property. Output respects the local --format flag
(table|json|yaml). The --profile global selects which profile's
adapter configuration is used.

Read-only: queries the upstream addressbook; no contact state is
mutated.

```
aps contact list [flags]
```

### Options

```
      --addressbook string   Addressbook ID
      --has-email            Filter on whether the contact has an email
  -h, --help                 help for list
      --org string           Filter to a single ORG value
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact note

Append note to contact

### Synopsis

Append a note to an existing contact identified by id via the
configured contacts adapter. The remaining positional arguments are
joined with single spaces into the note text. Each call appends a
fresh entry — not idempotent.

The note is stored in the upstream addressbook's NOTE field (or
adapter-specific equivalent). --dry-run is opted out because the
preview would only restate the text the user already typed.

```
aps contact note <id> <text> [flags]
```

### Options

```
  -h, --help   help for note
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact show

Show contact detail

### Synopsis

Show the full vCard body for a single contact identified by id
via the configured contacts adapter. Output is the raw adapter
response (typically vCard text) rather than the projected row shape
used by aps contact list. The --profile global selects which
profile's adapter configuration handles the request.

Read-only: queries the upstream addressbook; idempotent.

```
aps contact show <id> [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps contact update

Update contact fields

### Synopsis

Update one or more vCard fields on an existing contact via the
configured contacts adapter. The id argument identifies the card;
only flags that are explicitly passed are applied (--name, --email,
--org, --phone, --note). Unset flags leave the existing value
unchanged.

Idempotent at the field level: repeating the call with the same
payload converges. --dry-run is opted out because previewing the
diff would require the same provider round-trip that performs the
update.

```
aps contact update <id> [flags]
```

### Options

```
      --email string   Email address
  -h, --help           help for update
      --name string    Contact name
      --note string    Note
      --org string     Organization
      --phone string   Phone number
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps contact](#aps-contact)	 - Manage contacts via adapter

## aps directory

Manage AGNTCY Directory registration and discovery

### Synopsis

Manage agent registration and discovery via the AGNTCY Directory.

Register profiles to make them discoverable by other agents,
discover agents by capability, and manage directory records.

### Options

```
  -h, --help   help for directory
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps directory delete](#aps-directory-delete)	 - Remove a profile from the AGNTCY Directory
* [aps directory discover](#aps-directory-discover)	 - Discover agents by capability in the AGNTCY Directory
* [aps directory register](#aps-directory-register)	 - Register a profile with the AGNTCY Directory
* [aps directory show](#aps-directory-show)	 - Show a profile's OASF record

## aps directory delete

Remove a profile from the AGNTCY Directory

### Synopsis

Remove an agent profile's record from the AGNTCY Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory delete

```
aps directory delete [flags]
```

### Options

```
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps directory](#aps-directory)	 - Manage AGNTCY Directory registration and discovery

## aps directory discover

Discover agents by capability in the AGNTCY Directory

### Synopsis

Search the AGNTCY Directory for agents matching a capability query.

Example:
  aps directory discover --capability "invoice-processing"
  aps dir discover --capability a2a --endpoint https://dir.example.com
  aps --instance prod directory discover --capability a2a

```
aps directory discover [flags]
```

### Options

```
      --capability string   Capability to search for (required)
      --endpoint string     Directory endpoint URL (default: https://dir.agntcy.org)
  -h, --help                help for discover
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps directory](#aps-directory)	 - Manage AGNTCY Directory registration and discovery

## aps directory register

Register a profile with the AGNTCY Directory

### Synopsis

Register an agent profile with the AGNTCY Directory service.

Generates an OASF record from the profile and pushes it to the Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory register

```
aps directory register [flags]
```

### Options

```
  -h, --help          help for register
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps directory](#aps-directory)	 - Manage AGNTCY Directory registration and discovery

## aps directory show

Show a profile's OASF record

### Synopsis

Display the OASF record for a profile as it would appear in the Directory.

Profile is supplied via the tool-level --profile global:
  aps --profile worker directory show

```
aps directory show [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps directory](#aps-directory)	 - Manage AGNTCY Directory registration and discovery

## aps docs

Generate documentation

### Synopsis

Generate a markdown documentation tree under
<agents-dir>/docs/ describing every aps command, flag, and
subcommand. The agents dir resolves via core.GetAgentsDir (default:
$APS_DATA_PATH/agents).

The output is a deterministic projection of the live cobra command
tree, so rerunning overwrites existing files with identical content
when the binary hasn't changed. --dry-run is opted out because the
preview would have to render the same output and discard it instead
of writing it.

Idempotent overwrite: safe to rerun. No network calls.

```
aps docs [flags]
```

### Options

```
  -h, --help   help for docs
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps env

Output environment variables for configured capabilities

### Synopsis

Outputs shell commands to export environment variables for all installed capabilities.
Usage: eval $(aps env)

```
aps env [flags]
```

### Options

```
  -h, --help   help for env
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps identity

Manage DID-based agent identity

### Synopsis

Manage decentralized identifiers (DIDs) and verifiable credentials for agent profiles.

Initialize identity to generate a DID and Ed25519 key pair. Issue and verify
badges (Verifiable Credentials) to attest agent capabilities.

### Options

```
  -h, --help   help for identity
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps identity badge](#aps-identity-badge)	 - Manage agent badges (Verifiable Credentials)
* [aps identity init](#aps-identity-init)	 - Initialize identity for a profile
* [aps identity show](#aps-identity-show)	 - Show identity for a profile
* [aps identity verify](#aps-identity-verify)	 - Verify and resolve a DID

## aps identity badge

Manage agent badges (Verifiable Credentials)

### Synopsis

Issue and verify agent badges (Verifiable Credentials) for capability attestation.

### Options

```
  -h, --help   help for badge
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity](#aps-identity)	 - Manage DID-based agent identity
* [aps identity badge issue](#aps-identity-badge-issue)	 - Issue a badge for a capability
* [aps identity badge verify](#aps-identity-badge-verify)	 - Verify a badge file

## aps identity badge issue

Issue a badge for a capability

### Synopsis

Issue a signed Verifiable Credential attesting an agent's capability.

Profile is supplied via the tool-level --profile global:
  aps --profile worker identity badge issue --capability invoice-processing

```
aps identity badge issue [flags]
```

### Options

```
      --capability string   Capability to attest (required)
  -h, --help                help for issue
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity badge](#aps-identity-badge)	 - Manage agent badges (Verifiable Credentials)

## aps identity badge verify

Verify a badge file

### Synopsis

Verify a badge (Verifiable Credential) file's signature and contents.

Example:
  aps identity badge verify ~/.agents/profiles/worker/badges/invoice-processing.json

```
aps identity badge verify <badge-file> [flags]
```

### Options

```
  -h, --help   help for verify
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity badge](#aps-identity-badge)	 - Manage agent badges (Verifiable Credentials)

## aps identity init

Initialize identity for a profile

### Synopsis

Generate a DID and Ed25519 key pair for a profile.

Supported DID methods:
  did:key  — Self-describing, no network required (default)
  did:web  — Web-based, requires hosting a DID document

Profile is supplied via the tool-level --profile global:
  aps --profile worker identity init
  aps --profile worker id init --method did:web

```
aps identity init [flags]
```

### Options

```
  -h, --help            help for init
      --method string   DID method (did:key, did:web) (default "did:key")
  -n, --note string     Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity](#aps-identity)	 - Manage DID-based agent identity

## aps identity show

Show identity for a profile

### Synopsis

Display the DID and identity configuration for a profile.

Profile is supplied via the tool-level --profile global:
  aps --profile worker identity show

```
aps identity show [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity](#aps-identity)	 - Manage DID-based agent identity

## aps identity verify

Verify and resolve a DID

### Synopsis

Resolve a DID to its DID Document and display verification details.

Example:
  aps identity verify did:key:z6Mk...

```
aps identity verify <did> [flags]
```

### Options

```
  -h, --help   help for verify
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps identity](#aps-identity)	 - Manage DID-based agent identity

## aps listen

Subscribe to bus topics for a profile and print events as JSONL

### Synopsis

Long-running daemon that subscribes to configured bus topic patterns
for the given profile and prints every matching event as JSONL on stdout.

Honors SIGTERM/SIGINT for graceful shutdown — in-flight async forwarders
flush before exit (see drainBus in internal/cli/bus.go).

Examples:
  aps listen --profile noor
  aps listen --profile noor --topics "aps.#,tlc.task.*"
  aps listen --profile noor --exit-after-events 5

```
aps listen [flags]
```

### Options

```
      --exit-after-events int   Exit cleanly after N events received (0 = run until signal)
  -h, --help                    help for listen
      --topics string           Comma-separated topic patterns to subscribe to (kit/bus glob: * = one segment, # = trailing) (default "aps.#,tlc.#")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps migrate

Migrate legacy configurations to new formats

### Options

```
  -h, --help   help for migrate
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps migrate messengers](#aps-migrate-messengers)	 - Migrate messengers to adapter framework

## aps migrate messengers

Migrate messengers to adapter framework

### Synopsis

Migrate legacy messengers from ~/.aps/messengers into the unified adapter framework.

Discovers every directory with a manifest.yaml under ~/.aps/messengers and
converts each into the adapter shape under $APS_DATA_PATH. --only takes a
comma-separated allowlist; --backup snapshots the source tree under
xdg.DataDir("aps")/backups/messengers-<YYYYMMDD>/ before any mutation;
--dry-run previews the plan without touching disk. Idempotent: re-running
after a successful migration is a no-op when no legacy entries remain.

```
aps migrate messengers [flags]
```

### Options

```
      --backup        Create backup before migration
  -h, --help          help for messengers
      --only string   Migrate only specified messengers (comma-separated)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps migrate](#aps-migrate)	 - Migrate legacy configurations to new formats

## aps observability

Manage OpenTelemetry observability

### Synopsis

Manage OpenTelemetry observability for agent profiles.

Enables distributed tracing and metrics export via OpenTelemetry.
Supports OTLP (gRPC) and stdout exporters.

### Options

```
  -h, --help   help for observability
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps observability toggle](#aps-observability-toggle)	 - Enable or disable observability for a profile

## aps observability toggle

Enable or disable observability for a profile

### Synopsis

Enable or disable OpenTelemetry observability for a profile.

Without --enabled flag, toggles the current state.
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps observability toggle --profile worker                          # Toggle
  aps observability toggle --profile worker --enabled=on             # Force enable
  aps observability toggle --profile worker --enabled=off            # Force disable
  aps o11y toggle --profile worker --exporter=otlp --endpoint=localhost:4317
  aps otel toggle --profile worker --exporter=stdout --sampling-rate=0.5

```
aps observability toggle [flags]
```

### Options

```
      --enabled string        Enable (on), disable (off), or toggle (omit)
      --endpoint string       OTLP collector endpoint (e.g. localhost:4317)
      --exporter string       Exporter type (otlp, stdout, none) (default "stdout")
  -h, --help                  help for toggle
      --sampling-rate float   Trace sampling rate (0.0–1.0) (default 1)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps observability](#aps-observability)	 - Manage OpenTelemetry observability

## aps org

Inspect the reporting hierarchy across profiles

### Options

```
  -h, --help   help for org
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps org check](#aps-org-check)	 - Validate the reporting hierarchy for consistency
* [aps org show](#aps-org-show)	 - Show a profile's management chain and reports
* [aps org snapshot](#aps-org-snapshot)	 - Capture a point-in-time organigram

## aps org check

Validate the reporting hierarchy for consistency

### Synopsis

Load every profile on disk and validate the reporting
hierarchy: cycles, dangling reports_to references, self-references,
unknown type values, and profile.yaml files that fail to load
(surfaced as unloadable_profile findings instead of being silently
skipped). Each finding contributes a row with its KIND, the affected
PROFILES, and a human-readable MESSAGE.

The command exits non-zero when any finding exists and zero on a
clean hierarchy, so it can gate CI. Output respects the global
--format flag (table|json|yaml).

Read-only: no state mutation. Idempotent.

```
aps org check [flags]
```

### Options

```
  -h, --help   help for check
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps org](#aps-org)	 - Inspect the reporting hierarchy across profiles

## aps org show

Show a profile's management chain and reports

### Synopsis

Show where a profile sits in the reporting hierarchy: its
management chain up to the root (RELATION self, then manager rows in
walk order), its direct reports, and its transitive reports. Each
row carries the profile id, display name, effective type, and the
communication CHANNELS derivable from its configuration (a2a, acp,
email, webhooks).

--depth N limits how many levels of transitive reports are included;
the default 0 means unlimited. Direct reports always render.

Cyclic reports_to data never hangs: the partial chain is printed and
the command exits non-zero with a clear cycle error. Unknown profile
ids are an error. Output respects the global --format flag
(table|json|yaml).

Read-only: no state mutation. Idempotent.

```
aps org show <profile-id> [flags]
```

### Options

```
      --depth int   Levels of transitive reports to include (0 = unlimited)
  -h, --help        help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps org](#aps-org)	 - Inspect the reporting hierarchy across profiles

## aps org snapshot

Capture a point-in-time organigram

### Synopsis

Capture a point-in-time organigram of the reporting
hierarchy. The scope defaults to every profile on disk (--all);
--root <profile-id> limits it to a profile and its transitive
reports, and --squad <id> to the profiles whose squads list contains
the given squad id. The scope flags are mutually exclusive.

The rendering is selected with the LOCAL --snapshot-format flag
(json|yaml|tree|mermaid, default tree); the global --format flag is
ignored here because snapshots support formats the global enum does
not. json/yaml emit a {captured_at, scope, nodes, edges} document
with nodes and edges sorted by id; tree emits an ASCII forest (roots
first, reports indented, humans marked); mermaid emits a flowchart TD
with manager --> report edges. tree and mermaid carry no timestamp,
so identical state yields identical bytes.

The global --output <file> flag writes the rendering atomically
(temp file + rename) instead of stdout ("-" or empty means stdout).
Cyclic reports_to data still renders — traversals
are visited-set guarded — but the command exits non-zero with a
cycle error.

Read-only: no profile state is mutated; --output only writes the
operator-directed local file. Idempotent.

```
aps org snapshot [flags]
```

### Options

```
      --all                      Snapshot every profile on disk (default scope)
  -h, --help                     help for snapshot
      --root string              Snapshot a profile and its transitive reports
      --snapshot-format string   Snapshot rendering (json|yaml|tree|mermaid) (default "tree")
      --squad string             Snapshot the members of a squad
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps org](#aps-org)	 - Inspect the reporting hierarchy across profiles

## aps policy

Manage workspace access policies

### Synopsis

Manage access control policies for workspaces.

Policies control which devices can access a workspace and what
they can do. Modes:
  allow-all   All linked devices have access (default)
  allow-list  Only specified devices have access
  deny-list   All devices except specified ones have access

### Options

```
  -h, --help   help for policy
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps policy list](#aps-policy-list)	 - List workspace policies
* [aps policy set](#aps-policy-set)	 - Set workspace access policy
* [aps policy show](#aps-policy-show)	 - Show effective workspace policy
* [aps policy trust](#aps-policy-trust)	 - Manage inbound trust verification policy

## aps policy list

List workspace policies

### Synopsis

List the multidevice policy attached to the given workspace.

Reads the policy via multidevice.LoadPolicy(workspace-id) and renders the
mode plus per-device allow/deny rows as a styled table on a TTY. --json
emits the raw policy.Policy struct for structured callers. Read-only: no
state mutation, no network calls. Missing or unparseable policies return
the underlying error rather than an empty table.

```
aps policy list <workspace-id> [flags]
```

### Options

```
  -h, --help   help for list
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy](#aps-policy)	 - Manage workspace access policies

## aps policy set

Set workspace access policy

### Synopsis

Set the access control mode for a workspace.

Modes:
  allow-all   All linked devices have access (default)
  allow-list  Only specified devices have access
  deny-list   All devices except specified ones have access

Use --add-allow / --remove-allow to manage the allow list.
Use --add-deny / --remove-deny to manage the deny list.

Workspace is supplied via the tool-level --workspace global:
  aps --workspace <id> policy set --mode allow-list

```
aps policy set [flags]
```

### Options

```
      --add-allow strings      Add devices to allow list
      --add-deny strings       Add devices to deny list
  -h, --help                   help for set
      --json                   JSON output
      --mode string            Policy mode: allow-all, allow-list, deny-list
  -n, --note string            Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --remove-allow strings   Remove devices from allow list
      --remove-deny strings    Remove devices from deny list
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy](#aps-policy)	 - Manage workspace access policies

## aps policy show

Show effective workspace policy

### Synopsis

Show the effective access policy for a workspace, including
the mode and device lists.

Workspace is supplied via the tool-level --workspace global:
  aps --workspace <id> policy show

```
aps policy show [flags]
```

### Options

```
  -h, --help   help for show
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy](#aps-policy)	 - Manage workspace access policies

## aps policy trust

Manage inbound trust verification policy

### Synopsis

Configure trust verification for inbound A2A requests using DID-based identity.

### Options

```
  -h, --help   help for trust
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy](#aps-policy)	 - Manage workspace access policies
* [aps policy trust set](#aps-policy-trust-set)	 - Set trust verification policy for a profile
* [aps policy trust show](#aps-policy-trust-show)	 - Show trust verification policy for a profile

## aps policy trust set

Set trust verification policy for a profile

### Synopsis

Configure trust verification for inbound A2A requests.

When require-identity is set, the A2A server will reject requests
without a valid X-Agent-DID header. Optionally restrict to specific
DIDs via --allowed-issuers.

Profile is supplied via the tool-level --profile global:
  aps --profile worker policy trust set --require-identity
  aps --profile worker policy trust set --require-identity --allowed-issuers "did:key:z6Mk..."
  aps --profile worker policy trust set --require-identity=false

```
aps policy trust set [flags]
```

### Options

```
      --allowed-issuers string   Comma-separated list of allowed DIDs
  -h, --help                     help for set
      --require-identity         Require X-Agent-DID header on inbound requests
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy trust](#aps-policy-trust)	 - Manage inbound trust verification policy

## aps policy trust show

Show trust verification policy for a profile

### Synopsis

Display the current trust verification policy configuration.

Profile is supplied via the tool-level --profile global:
  aps --profile worker policy trust show

```
aps policy trust show [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps policy trust](#aps-policy-trust)	 - Manage inbound trust verification policy

## aps profile

Manage agent profiles

### Synopsis

Create, list, and inspect agent profiles.

### Options

```
  -h, --help   help for profile
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps profile capability](#aps-profile-capability)	 - Manage capabilities on a profile
* [aps profile create](#aps-profile-create)	 - Create a new profile
* [aps profile delete](#aps-profile-delete)	 - Delete a profile
* [aps profile edit](#aps-profile-edit)	 - Edit fields on an existing profile
* [aps profile export](#aps-profile-export)	 - Export a profile record
* [aps profile import](#aps-profile-import)	 - Import a shared profile bundle or an agent role manifest
* [aps profile list](#aps-profile-list)	 - List all available profiles
* [aps profile share](#aps-profile-share)	 - Export a shareable profile bundle
* [aps profile show](#aps-profile-show)	 - Show profile details
* [aps profile status](#aps-profile-status)	 - Show bundle resolution status for a profile
* [aps profile trust](#aps-profile-trust)	 - Show trust scores and history for a profile
* [aps profile workspace](#aps-profile-workspace)	 - Manage workspace link for a profile

## aps profile capability

Manage capabilities on a profile

### Options

```
  -h, --help   help for capability
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles
* [aps profile capability add](#aps-profile-capability-add)	 - Add a capability to a profile
* [aps profile capability remove](#aps-profile-capability-remove)	 - Remove a capability from a profile

## aps profile capability add

Add a capability to a profile

### Synopsis

Attach a capability to a profile's capability list. The
capability argument must resolve in either the builtin registry or
the external capabilities tree under
$APS_DATA_PATH/capabilities/; unknown names are rejected before
any state changes.

Mutating: writes profile.yaml and emits a ProfileUpdated bus event
with the --note metadata attached. Idempotent at the field level
(re-adding an already-attached capability is a no-op).

```
aps profile capability add <profile> <capability> [flags]
```

### Options

```
  -h, --help          help for add
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile capability](#aps-profile-capability)	 - Manage capabilities on a profile

## aps profile capability remove

Remove a capability from a profile

### Synopsis

Detach a capability from a profile's capability list. The
capability files on disk are not removed — only the link on the
profile record is dropped, so other profiles with the same
capability are unaffected.

Mutating: writes profile.yaml and emits a ProfileUpdated bus event
with the --note metadata attached. Idempotent: removing a
capability that isn't on the profile is a no-op.

```
aps profile capability remove <profile> <capability> [flags]
```

### Options

```
  -h, --help          help for remove
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile capability](#aps-profile-capability)	 - Manage capabilities on a profile

## aps profile create

Create a new profile

### Synopsis

Create a new agent profile under
$APS_DATA_PATH/profiles/<id>/. The id argument doubles as the
directory name; display name, email, avatar URL, and color hex
default to interactive prompts when omitted and stdin is a TTY.
Pass --force to overwrite an existing profile directory.

The --type flag sets the profile type discriminator (agent or
human; empty means agent). The --reports-to flag links the profile
into the reporting hierarchy: the target must be an existing
profile, and self-references or reporting cycles are rejected
before anything is written.

The --auto-avatar / --auto-color flags generate deterministic
values from the profile id (avatar via the configured provider,
default dicebear; color from a fixed palette hash). Provider knobs
(--avatar-provider, --avatar-style, --avatar-size, --avatar-format)
override per-call config. ProfileDefaults config drives the auto
behavior when the flags are unset.

Mutating: writes the profile.yaml record and (optionally) seeds the
git config block. Emits a ProfileCreated bus event with the --note
metadata attached. Not idempotent — each call creates a fresh
record (use --force to replace).

```
aps profile create [id] [flags]
```

### Options

```
      --auto-avatar              Generate a deterministic avatar via the configured provider (overrides config)
      --auto-color               Generate a deterministic palette color (overrides config)
      --avatar string            URL or local path to profile image
      --avatar-format string     Avatar format: svg, png, webp (provider-dependent)
      --avatar-provider string   Avatar provider name (default: kit/avatar's default — dicebear)
      --avatar-size int          Avatar size in pixels (0 = provider default)
      --avatar-style string      Provider-specific style (e.g. dicebear: shapes, bottts, identicon)
      --color string             Hex color (e.g. #3b82f6) for UI rendering
      --display-name string      Display name for the profile
      --email string             Email for profile and git config
      --force                    Overwrite existing profile
  -h, --help                     help for create
  -n, --note string              Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --reports-to string        Profile id this profile reports to (must exist; no self-reference or cycles)
      --type string              Profile type: agent or human (empty means agent)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile delete

Delete a profile

### Synopsis

Delete a profile directory under $APS_DATA_PATH/profiles/<id>/
along with its profile.yaml, secrets, and any side-car files.
Interactive confirmation prompts unless --yes is set or stdin is
not a TTY.

Destructive: irreversible without a prior aps profile share
export. Blocked when the profile has active sessions in the
session store, or when other profiles report to it (their ids are
listed); --force overrides both guards and removes the profile
anyway, leaving orphaned sessions to fail on next lookup and
inbound reports_to links dangling. A ProfileDeleted bus event
fires with the --note metadata attached.

```
aps profile delete <id> [flags]
```

### Options

```
      --force         Delete even if there are active sessions (orphans them — they keep running but lose profile context) or inbound reports_to links (they are left dangling)
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
  -y, --yes           Skip interactive confirmation
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile edit

Edit fields on an existing profile

### Synopsis

Update display name, email, avatar, color, type, or reports-to on an
existing profile.

Only flags that are explicitly passed are applied; unset flags leave the
existing value unchanged. To clear a field, pass the flag with an empty
string (e.g. --avatar "").

--type is validated write-strict (agent or human). --reports-to must
name an existing profile and may not introduce a self-reference or a
reporting cycle; the would-be graph is checked before saving.

The --auto-avatar / --auto-color flags generate a deterministic value
from the profile id and overwrite the existing value when set.

```
aps profile edit [id] [flags]
```

### Options

```
      --auto-avatar              Generate and apply a deterministic avatar via the configured provider
      --auto-color               Generate and apply a deterministic palette color
      --avatar string            URL or local path to profile image (pass empty string to clear)
      --avatar-format string     Avatar format for --auto-avatar
      --avatar-provider string   Avatar provider name for --auto-avatar
      --avatar-size int          Avatar size in pixels for --auto-avatar
      --avatar-style string      Provider-specific style for --auto-avatar
      --color string             Hex color (e.g. #3b82f6) for UI rendering (pass empty string to clear)
      --display-name string      Display name for the profile
      --email string             Email for profile and git config
  -h, --help                     help for edit
  -n, --note string              Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --reports-to string        Profile id this profile reports to (pass empty string to clear; must exist; no self-reference or cycles)
      --type string              Profile type: agent or human (pass empty string to clear back to the agent default)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile export

Export a profile record

### Synopsis

Export a profile to stdout (or --out <path>). The default
output is the native profile.yaml record. --manifest-format agentco
renders an agent role manifest (AGENTS.md — YAML frontmatter + markdown
body): name from the display name, slug from the profile id,
skills from the linked capability shortnames, and the body from
notes.md (the same file manifest import writes).

The agentco format exports identity only — secrets, isolation
config, gitconfig, knowledge references, and machine-specific
paths are never included.

Read-only: loads profile state; writes only to stdout or the
--out destination. Idempotent.

```
aps profile export <id> [flags]
```

### Options

```
  -h, --help                     help for export
      --manifest-format string   Manifest format: agentco (agent role manifest); omit for native yaml
      --out string               Write to a file instead of stdout
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile import

Import a shared profile bundle or an agent role manifest

### Synopsis

Import a profile bundle previously produced by aps profile
share, or an agent role manifest (AGENTS.md — YAML frontmatter +
markdown body). Dispatch is by extension: a .md argument is
treated as a manifest, anything else as a .aps-profile.yaml
bundle. By default the new profile keeps the source id; pass --id
to rename it (e.g. when the local install already has a profile
with the source id). For bundle imports --force overwrites an
existing profile directory with the same target id; for manifest
imports it overrides the reportsTo check (see below).

Manifest imports map title (falling back to name) to the display
name, slug (falling back to the slugified name) to the profile
id, description and reportsTo to the matching profile fields,
the markdown body to notes.md, and each skills entry to a
capability link when the shortname resolves in the capability
registry — unresolvable shortnames are warned to stderr and
skipped, never failing the import. A reportsTo naming no existing
profile does fail the import, before anything is written — import
the supervising profile first, or pass --force to downgrade it to
a warning and store the value anyway. A reportsTo that would
introduce a self-reference or a reporting cycle always fails the
import; --force never downgrades a cycle. Secrets, isolation, and
machine-specific paths are never taken from a manifest; the
profile receives the normal create-path defaults. --dry-run
prints the resulting profile.yaml plus intended links and skips
without writing anything, and reports the same reportsTo verdict
a real import would.

Mutating: creates $APS_DATA_PATH/profiles/<target-id>/ and emits
both a ProfileCreated bus event and a profile_share_imported
tracking event. The --note metadata is attached to the
ProfileCreated payload. Secrets are NOT imported (they are not
part of the bundle); set them separately after import.

```
aps profile import [bundle|AGENTS.md] [flags]
```

### Options

```
      --force         Overwrite an existing profile (bundle imports); import despite a reportsTo that names no existing profile (manifest imports)
  -h, --help          help for import
      --id string     Override profile ID from bundle
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile list

List all available profiles

### Synopsis

List every agent profile discovered under
$APS_DATA_PATH/profiles/, projected into a row shape that includes
id, display name, roles, capabilities, workspace link, email,
has-secrets, has-identity, color, and avatar.

Filters compose with AND semantics: --capability, --role, --squad,
--tone match against the corresponding slice/string field;
--workspace (kit-shipped persistent global) filters by linked
workspace name; --has-identity / --has-secrets are boolean flags
that scope to profiles that do (or do not) have the given module
attached. Output respects the global --format flag (table|json|
yaml).

Read-only: no profile state is mutated. Idempotent.

```
aps profile list [flags]
```

### Options

```
      --capability string   Filter by capability membership
      --has-identity        Filter to profiles with (true) or without (false) a DID identity
      --has-secrets         Filter to profiles with (true) or without (false) at least one secret
  -h, --help                help for list
      --role string         Filter by role membership (owner, assignee, evaluator, auditor)
      --squad string        Filter by squad membership
      --tone string         Filter by persona tone
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile share

Export a shareable profile bundle

### Synopsis

Export a profile as a portable bundle file that another aps
install can ingest via aps profile import. The default output path
is "<id>.aps-profile.yaml" in the current working directory;
--out overrides the destination.

The bundle includes the profile.yaml plus any side-car files
(secrets are NOT included — the importer must re-supply them per
the profile_share_created tracking event). A profile_share_created
event is emitted on success with the bundle version recorded.

Mutating (filesystem write): creates the bundle file. Pair with
aps profile import on the receiving install.

```
aps profile share [id] [flags]
```

### Options

```
  -h, --help         help for share
      --out string   Output path for the bundle
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile show

Show profile details

### Synopsis

Show the full profile record as YAML: identity fields,
workspace link, capability list (annotated as builtin or external
with description), and the status of optional modules
(secrets present/missing, redacted secret keys). Capability
descriptions are looked up from the builtin registry or the
external capability path.

Read-only: loads $APS_DATA_PATH/profiles/<id>/profile.yaml and
prints a human-friendly render. Idempotent.

```
aps profile show [id] [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile status

Show bundle resolution status for a profile

### Synopsis

Show per-bundle binary resolution for a profile: which required
binaries are active, missing/skipped, or blocked. Bundles are
inferred from the profile's capability list via
core.ExtractBundleNames.

With the inherited --verbose global, also emits each bundle's
resolved scope (operations, file_patterns, networks) and the set
of env-var keys the bundle would inject when activated. Warnings
from the resolver (e.g. version conflicts) surface inline.

Read-only: loads profile.yaml and runs the bundle resolver; no
disk writes. Idempotent up to filesystem drift between calls.

```
aps profile status [id] [flags]
```

### Options

```
  -h, --help   help for status
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile trust

Show trust scores and history for a profile

### Synopsis

Show the trust ledger for a profile: per-domain trust scores
and (with --history) the timestamped entries that produced them.
The trust ledger lives on the profile record and accumulates as
the profile completes evaluated tasks.

Filters: --domain scopes the score block (and history, when
shown) to a single trust domain; without it, every entry in
core.TrustDomains is rendered. --json switches to a structured
emission shaped {roles, scores, history?}; otherwise the styled
table renderer is used.

Read-only: loads profile.yaml and projects the trust ledger.
Idempotent.

```
aps profile trust <profile-id> [flags]
```

### Options

```
      --domain string   Filter by trust domain
  -h, --help            help for trust
      --history         Show trust history entries
      --json            Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles

## aps profile workspace

Manage workspace link for a profile

### Options

```
  -h, --help   help for workspace
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile](#aps-profile)	 - Manage agent profiles
* [aps profile workspace set](#aps-profile-workspace-set)	 - Set workspace link for a profile

## aps profile workspace set

Set workspace link for a profile

### Synopsis

Associate a profile with a workspace by name.

```
aps profile workspace set <profile-id> <workspace-name> [flags]
```

### Options

```
  -h, --help          help for set
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps profile workspace](#aps-profile-workspace)	 - Manage workspace link for a profile

## aps run

Run a command in a profile context

### Synopsis

Spawn an external command under the named profile's resolved
environment (secrets, bundle env vars, profile-scoped settings).
The "--" separator is required; everything before it is parsed by
aps, everything after is the user command and its arguments.

A structured progress envelope (exec start + exit) is emitted on
stdout per kit's JSONL progress contract so agents that read the
stream observe a uniform lifecycle without aps touching the child
process's stdio. The child's exit code is propagated as the aps
exit code on failure.

Mutating: spawns an opaque subprocess whose side effects aps cannot
enumerate; idempotency is conditional on the spawned command.
--dry-run is opted out because previewing a third-party binary
without invoking it is impossible.

```
aps run [profile] -- [command] [args...] [flags]
```

### Options

```
      --env stringArray    Set KEY=VALUE in the child env; repeatable; later --env wins for duplicate keys
      --env-file strings   Read KEY=VALUE entries from a dotenv-style file; repeatable; later files win for duplicate keys
  -h, --help               help for run
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps serve

Start protocol server

### Synopsis

Start Agent Protocol server to expose APS functionality via HTTP API.

```
aps serve [flags]
```

### Options

```
      --addr string         Address to listen on (default "127.0.0.1:8080")
      --auth-token string   Bearer token for authentication (optional)
  -h, --help                help for serve
      --log-level string    Log level (debug, info, warn, error) (default "info")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps service

Manage profile-facing services

### Options

```
  -h, --help   help for service
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps service add](#aps-service-add)	 - Add a profile-facing service
* [aps service conversation](#aps-service-conversation)	 - Inspect recorded message-service conversations
* [aps service routes](#aps-service-routes)	 - Show reachable routes for a persisted service
* [aps service show](#aps-service-show)	 - Show a persisted service
* [aps service start](#aps-service-start)	 - Start APS HTTP serving for a service webhook
* [aps service status](#aps-service-status)	 - Show operator status for a service
* [aps service stop](#aps-service-stop)	 - Show how to stop a foreground service server
* [aps service test](#aps-service-test)	 - Validate service configuration and optionally probe its webhook

## aps service add

Add a profile-facing service

### Synopsis

Add a profile-facing service.

The --type flag accepts canonical service types and adapter aliases. Adapter
aliases are resolved through kit aliasing before APS persists the service.

```
aps service add <service-id> --type <type-or-adapter-alias> --profile <profile-id> [flags]
```

### Options

```
      --adapter string                    Concrete adapter when --type is canonical
      --allowed-channel stringArray       Allowed message channel ID, repeatable
      --allowed-chat stringArray          Allowed Telegram chat ID, repeatable
      --allowed-guild stringArray         Allowed Discord guild ID, repeatable
      --allowed-number stringArray        Allowed phone number, repeatable
      --allowed-sender stringArray        Allowed email sender: exact address or *@domain glob (case-insensitive), repeatable
      --auth-scheme string                Generic webhook auth scheme: bearer, token, hmac-sha256, ed25519, slack-signing-secret (bearer/token read --auth-token-env; hmac-sha256/ed25519 read --signature-secret-env)
      --auth-token-env string             Environment variable holding the generic webhook bearer/token secret
      --bot-user-id string                Slack bot user ID used for mention-only routing
      --contacts string                   Contacts snapshot YAML consulted by --route-table (org:/contact: selectors)
      --dedup-ttl string                  Slack Events API duplicate event retention duration
      --default-action string             Default profile action for routed messages or tickets
      --description string                Human-readable description
      --env stringArray                   Environment binding KEY=VALUE, repeatable; VALUE may be a literal or secret:NAME to resolve NAME from the profile secret store, then the environment
      --events string                     Comma-separated ticket events to receive
      --force                             Overwrite an existing service with the same ID
      --from string                       Sender phone number or provider identity
      --group string                      GitLab group path or ID
  -h, --help                              help for add
      --history-turns int                 Prior conversation turns attached to each routed action run (0 = default 20)
      --jql string                        Jira issue query
      --label stringArray                 Metadata label KEY=VALUE, repeatable
      --language-code string              WhatsApp template language code
      --linear-workspace string           Linear workspace key or ID
      --option stringArray                Raw service option KEY=VALUE, repeatable; escape hatch for options without a flag (named flags win on the same key)
      --phone-number-id string            WhatsApp phone number ID
      --project string                    Ticket adapter project key, ID, or path
      --provider string                   Message provider, such as twilio or whatsapp-cloud
      --receive string                    Message receive mode: polling or webhook
      --reply string                      Reply behavior: text, comment, status, auto, or none
      --require-bot-mention               Require Slack channel messages to mention the bot
      --route-table string                Sender route table YAML for message services (replaces --default-action; relative paths resolve against the services directory)
      --signature-secret-env string       Environment variable holding the generic webhook HMAC secret or Ed25519 public key
      --signing-secret-env string         Slack/WhatsApp provider signing secret environment variable (provider-native signature; for generic webhook auth use --signature-secret-env)
      --site string                       Ticket adapter site or base URL
      --team string                       Linear team key or ID
      --template-name string              WhatsApp template name for business-initiated replies
      --template-required                 Require WhatsApp outbound delivery to use a template
      --type string                       Service type or adapter alias
      --verify-token string               WhatsApp Cloud webhook verification token
      --verify-token-env string           Environment variable holding WhatsApp Cloud verification token
      --webhook-secret-token string       Telegram webhook secret token
      --webhook-secret-token-env string   Environment variable holding Telegram webhook secret token
      --webhook-url string                Public provider webhook URL used for signature validation
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service conversation

Inspect recorded message-service conversations

### Synopsis

Query the message thread history recorded by message services.

Every routed inbound message and every delivered reply is persisted as a
conversation turn keyed by the conversation policy identity
(msgconv:v1:... / msgsess:v1:...). The same turns are attached to routed
action runs as prior_turns.

### Options

```
  -h, --help   help for conversation
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services
* [aps service conversation list](#aps-service-conversation-list)	 - List recorded conversations
* [aps service conversation show](#aps-service-conversation-show)	 - Show the recorded turns of a conversation

## aps service conversation list

List recorded conversations

### Synopsis

List conversations recorded by message services, most recently
active first. Each row reports the conversation ID, owning service,
platform, channel, turn count, last activity time and direction, and a
preview of the last turn's text.

Filters: --service, --platform, --limit. Output respects the global
--format flag (table|json|yaml). Read-only: the store is never created
by this command; a missing store lists nothing. Idempotent.

```
aps service conversation list [flags]
```

### Options

```
  -h, --help              help for list
      --limit int         Maximum conversations to list (0 = all)
      --platform string   Only conversations on this platform (sms, whatsapp, telegram, slack, discord)
      --service string    Only conversations recorded by this service ID
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service conversation](#aps-service-conversation)	 - Inspect recorded message-service conversations

## aps service conversation show

Show the recorded turns of a conversation

### Synopsis

Print the turns recorded for one conversation, oldest first so the
newest turn is last — the same order and shape actions receive in
prior_turns. Each turn carries its direction (inbound/outbound), sender,
text, message ID, profile/action, and the session (thread) key.

--session narrows the conversation to one policy session key (a platform
thread); --limit keeps only the newest N turns. Output respects the
global --format flag (table|json|yaml). Read-only. Idempotent.

```
aps service conversation show <conversation-id> [flags]
```

### Options

```
  -h, --help             help for show
      --limit int        Maximum turns to show, newest kept (default 500)
      --session string   Only turns in this session (thread) key
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service conversation](#aps-service-conversation)	 - Inspect recorded message-service conversations

## aps service routes

Show reachable routes for a persisted service

### Synopsis

Show the HTTP routes that aps serve (or aps service start)
would mount for the named service. Routes are derived by
core.DescribeServiceRuntime from the service's adapter type and
configuration; "routes: none" prints when the adapter exposes no
endpoints.

Read-only: no service state mutation. Idempotent.

```
aps service routes <service-id> [flags]
```

### Options

```
  -h, --help   help for routes
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service show

Show a persisted service

### Synopsis

Show the full configuration for a persisted service: id, type,
backing adapter, owning profile, optional description, and the
adapter-specific option map sorted by key. Message services also
print the inbound auth the validator will enforce (auth: scheme,
header, token/secret env names, timestamp and replay headers, or
"auth: none"). Runtime metadata from core.DescribeServiceRuntime
(receives, executes, replies, maturity) is also printed.

Read-only: loads the service record from the service store and
prints it. Idempotent.

```
aps service show <service-id> [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service start

Start APS HTTP serving for a service webhook

### Synopsis

Start APS HTTP serving for a service webhook.

APS serves message services through the shared protocol server. This command
validates the selected service, reports its reachable webhook URL, starts the
same mounted service routes as aps serve, and runs in the foreground until
interrupted.

```
aps service start <service-id> [flags]
```

### Options

```
      --addr string       Address to listen on (default "127.0.0.1:8080")
      --base-url string   Public base URL used to report reachable webhook URLs (default "http://127.0.0.1:8080")
  -h, --help              help for start
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service status

Show operator status for a service

### Synopsis

Show operator-facing status for a service: id, type, adapter,
profile, lifecycle hint, reachable webhook URL (from --base-url),
runtime receives/replies, delivery health summary, last inbound
and outbound event metadata, the retry policy, and any
configuration validation issues.

The --base-url flag overrides the default http://127.0.0.1:8080
public base used to render reachable URLs. Read-only: no service
or upstream state is touched. Idempotent.

```
aps service status <service-id> [flags]
```

### Options

```
      --base-url string   Public base URL used to report reachable webhook URLs (default "http://127.0.0.1:8080")
  -h, --help              help for status
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service stop

Show how to stop a foreground service server

### Synopsis

Print guidance for stopping the foreground HTTP server that
hosts the named service. The aps service lifecycle is external:
the server runs only while aps service start (or aps serve) is
in the foreground, so stop prints the canonical "interrupt the
process" instruction rather than sending any signal itself.

Read-only: loads the service record to confirm the id exists,
then prints the lifecycle hint. Idempotent.

```
aps service stop <service-id> [flags]
```

### Options

```
  -h, --help   help for stop
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps service test

Validate service configuration and optionally probe its webhook

### Synopsis

Validate the persisted service configuration via
core.ValidateServiceConfig — surfacing any issues or warnings —
and report the reachable webhook URL derived from --base-url. The
command exits non-zero if the config is invalid.

With --probe, additionally POST a synthetic adapter-shaped
payload (signed for telegram/slack/sms/whatsapp where signing
secrets are configured) at the webhook URL and print the response
status and body. The synthetic inbound impersonates the first
configured allowlist entry (allowed_numbers, allowed_chats,
allowed_channels, allowed_guilds) and the service's own channel
identity, so it passes the same allowlist checks real traffic
must pass; probe_sender/probe_channel report what was sent. A 403
therefore means the allowlist rejected the probe identity, not
that the endpoint is down. --timeout bounds the probe round-trip;
the default is 5s.

Read-only on aps state; --probe makes a live outbound HTTP call
each invocation so the kit-level idempotency tag is Conditional.

```
aps service test <service-id> [flags]
```

### Options

```
      --base-url string    Public base URL to probe (default "http://127.0.0.1:8080")
  -h, --help               help for test
      --probe              POST a synthetic message payload to the webhook URL
      --timeout duration   Webhook probe timeout (default 5s)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps service](#aps-service)	 - Manage profile-facing services

## aps session

Manage sessions

### Options

```
  -h, --help   help for session
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps session attach](#aps-session-attach)	 - Attach to a running session
* [aps session delete](#aps-session-delete)	 - Delete a session
* [aps session detach](#aps-session-detach)	 - Detach from a running session
* [aps session inspect](#aps-session-inspect)	 - Inspect a session's details
* [aps session list](#aps-session-list)	 - List active sessions
* [aps session logs](#aps-session-logs)	 - Show session logs (tmux or container capture)
* [aps session terminate](#aps-session-terminate)	 - Terminate a session gracefully

## aps session attach

Attach to a running session

### Synopsis

Attach the current terminal to the tmux server backing a
running session. The session is looked up by ID (or --latest, which
picks the most recently created active session) and must be in
status=active. The command shells out to tmux against the session's
recorded socket; when the session runs in a platform sandbox
(platform_type=macos-darwin or linux-namespace), the attach path
tunnels through ssh into the sandbox user using the admin private
key at $APS_DATA_PATH/keys/admin_priv.

Interactive lifecycle: --mode=view attaches read-only (-r), --mode
=control (the default) attaches read/write. The standard tmux
detach binding (Ctrl-B then D) returns control to the caller.

Idempotency is conditional on mode and current state — repeating
attach against an already-attached session opens an additional
client; switching mode while attached requires detach first. Pair
with aps session detach to disconnect without terminating, or aps
session terminate to stop the session entirely.

```
aps session attach <session-id> [flags]
```

### Options

```
  -h, --help          help for attach
  -l, --latest        Attach to the most recent session
  -m, --mode string   Attachment mode (view|control) (default "control")
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session delete

Delete a session

### Synopsis

Remove a session entry from the registry and tear down the
backing tmux server. Tmux teardown failures (server already gone,
benign races) are logged as warnings but do not abort the
registry unregister — the session is going away regardless. The
user is prompted for confirmation unless --force is passed.

Before the registry write, the command publishes a synchronous
policy pre-persisted event on the kit policy bus with Op=delete
and kind=session; any sync subscriber (the runtime policy engine
wired in PersistentPreRunE) can veto by returning a policy denial.
When the session is bound to a workspace, the workspace ID is
stuffed into request_attrs so the principal resolver can surface
the calling profile's workspace role as principal.role.

Destructive: the session record is removed irreversibly. The
destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would only restate the
session ID. Idempotent on already-absent records. Use --note to
attach an audit reason that flows to the SessionStopped event.

```
aps session delete <session-id> [flags]
```

### Options

```
      --force         Force delete without confirmation
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session detach

Detach from a running session

### Synopsis

Disconnect attached tmux clients from a session without
terminating it. The target session is identified by [session-id]
or, with --all, every active session in the registry is detached
in one pass. The session itself keeps running on the backing tmux
server and can be re-attached later via aps session attach.

Local mutation against the tmux server only — no registry state
changes. Idempotent: an already-detached session stays detached
and the command reports the detach count.

--dry-run is opted out because preview would only restate the
session ID being detached. Pair with aps session attach to
reconnect, or aps session terminate to stop the session entirely.

```
aps session detach [session-id] [flags]
```

### Options

```
  -a, --all           Detach from all sessions
  -h, --help          help for detach
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session inspect

Inspect a session's details

### Synopsis

Print the full record for a single session — ID, owning
profile, profile directory, command, PID, status, tier, created/
last-seen timestamps, tmux socket path, and the propagated
environment map. The environment block runs through the aps
logging redactor before render, so OPENAI_API_KEY, GITHUB_TOKEN,
and other tokens propagated by the session builder are masked in
both the table and JSON outputs.

The default output is a styled property/value table; --json
switches to a JSON dump of the full SessionInfo struct and
--pretty indents that JSON. Read-only: no state mutation.
Idempotent.

```
aps session inspect <session-id> [flags]
```

### Options

```
  -h, --help     help for inspect
      --json     Output in JSON format
      --pretty   Pretty-print JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session list

List active sessions

### Synopsis

List every session known to the local session registry. Each
row reports the session ID, owning profile, status (active,
inactive, errored), workspace ID, type (standard, voice), tier
(basic, standard, premium), creation timestamp, and last-seen
timestamp.

The output respects the global --format flag (table|json|yaml) and
the global --profile and --workspace persistent flags (which filter
to the named profile/workspace). Local filters: --status, --tier,
--type. Empty session Type values (legacy registry entries written
before the field existed) render as "standard" so the TYPE column
stays populated. Read-only: no state mutation. Idempotent.

```
aps session list [flags]
```

### Options

```
  -h, --help            help for list
      --status string   Filter sessions by status (active, inactive, errored)
      --tier string     Filter sessions by tier (basic, standard, premium)
      --type string     Filter sessions by type (standard, voice); default = all
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session logs

Show session logs (tmux or container capture)

### Synopsis

Capture a session's log output and write it to stdout.

Tmux-backed sessions are captured with tmux capture-pane against
the socket recorded in the registry. --tail "all" dumps the entire
buffer, --tail <N> limits to the last N lines, and the default
captures the visible pane plus escape sequences. --follow
re-attaches pipe-pane so new output streams as it lands.

Container-backed sessions (a container id and no tmux socket) are
captured with docker logs, where --follow, --tail and --timestamps
all map to native flags.

--timestamps has no effect on tmux sessions: capture-pane exposes
no timestamp option. The flag is accepted and a warning is printed.

Read-only: no state mutation on the session or the buffer.
Idempotent on the same buffer state. Pair with aps session attach
when you want an interactive terminal rather than a one-shot
capture.

```
aps session logs <session-id> [flags]
```

### Options

```
  -f, --follow        Follow log output
  -h, --help          help for logs
      --tail string   Number of lines to show from the end ("all" for entire buffer)
      --timestamps    Show timestamps
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps session terminate

Terminate a session gracefully

### Synopsis

Tear down a running session and its tmux server. The default
graceful path asks tmux to kill the session (which sends SIGHUP to
the inner shell), waits up to --timeout seconds (default 10) for
the inner PID to exit, and escalates to SIGKILL if the deadline
elapses. With --force the inner PID is SIGKILLed immediately and
tmux is killed without waiting.

Registry status is updated to inactive even when tmux teardown
returns warnings, so the session entry never strands in an
ambiguous in-between state. Pair with aps session delete to remove
the registry entry afterwards, or skip terminate entirely and call
aps session delete which tears down and unregisters in one step.

Destructive: in-flight work in the session is lost. The
destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would have to fake the OS
signal path that defines the operation. Idempotent on the registry
status — terminated sessions stay inactive. Use --note to attach an
audit reason that flows to the SessionStopped event payload.

```
aps session terminate <session-id> [flags]
```

### Options

```
      --force         Force terminate without graceful shutdown
  -h, --help          help for terminate
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --timeout int   Graceful shutdown timeout in seconds (default 10)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps session](#aps-session)	 - Manage sessions

## aps skill

Manage Agent Skills

### Synopsis

Manage Agent Skills - discover, install, run, and configure skills for your profiles.

### Options

```
  -h, --help   help for skill
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps skill install](#aps-skill-install)	 - Install a skill
* [aps skill list](#aps-skill-list)	 - List available skills
* [aps skill run](#aps-skill-run)	 - Run a skill script
* [aps skill show](#aps-skill-show)	 - Show detailed skill information
* [aps skill stats](#aps-skill-stats)	 - Show skill usage statistics
* [aps skill suggest](#aps-skill-suggest)	 - Suggest IDE skill paths to configure
* [aps skill validate](#aps-skill-validate)	 - Validate a skill

## aps skill install

Install a skill

### Synopsis

Install a skill from a local directory or archive.

```
aps skill install <path> [flags]
```

### Options

```
      --global   Install to global skills directory
  -h, --help     help for install
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill list

List available skills

### Synopsis

List all skills available in configured skill directories.

```
aps skill list [flags]
```

### Options

```
  -h, --help            help for list
      --source string   Filter by source label (Profile, Global, User, Claude Code, Cursor, Zed, VS Code, Windsurf)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill run

Run a skill script

### Synopsis

Execute a script from a skill.

```
aps skill run <skill-name> -- <script> [args...] [flags]
```

### Options

```
  -h, --help   help for run
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill show

Show detailed skill information

### Synopsis

Show the full SKILL.md metadata for a single discovered skill:
name, description, license, compatibility tag, arbitrary metadata
fields, on-disk location, the resolving skill source, the list of
scripts under scripts/, the list of references, and a preview of
the instructional body (first 500 chars).

Skill discovery uses the same registry as aps skill list — the
--profile global selects which profile's configured skill sources
are scanned. Read-only: no skill content is modified. Idempotent.

```
aps skill show <skill-name> [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill stats

Show skill usage statistics

### Synopsis

Show recorded skill invocation telemetry for the active
profile: total invocations, completions, and failures across all
skills, then a per-skill breakdown (invocations, completions,
failures, success rate, average duration in ms). The telemetry
store is populated by aps skill run as it tracks invocation,
completion, and failure events.

The --profile global selects which profile's telemetry to read.
Prints "No skill usage recorded yet." when the store is empty.
Read-only: idempotent.

```
aps skill stats [flags]
```

### Options

```
  -h, --help   help for stats
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill suggest

Suggest IDE skill paths to configure

### Synopsis

Detect IDE/TDE skill directories and suggest adding them to configuration.

```
aps skill suggest [flags]
```

### Options

```
  -h, --help   help for suggest
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps skill validate

Validate a skill

### Synopsis

Validate that a skill directory follows the Agent Skills specification.

```
aps skill validate <path> [flags]
```

### Options

```
  -h, --help   help for validate
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps skill](#aps-skill)	 - Manage Agent Skills

## aps squad

Manage agent squads (topology, membership, scope)

### Options

```
  -h, --help   help for squad
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps squad check](#aps-squad-check)	 - Validate squad topology against the 8-item design checklist
* [aps squad create](#aps-squad-create)	 - Create a new squad
* [aps squad delete](#aps-squad-delete)	 - Delete a squad
* [aps squad list](#aps-squad-list)	 - List all squads
* [aps squad members](#aps-squad-members)	 - Manage squad membership
* [aps squad show](#aps-squad-show)	 - Show squad details

## aps squad check

Validate squad topology against the 8-item design checklist

### Synopsis

Validate the current squad topology against the design
checklist defined in the core squad package — invariants drawn
from team-topology theory (e.g. stream-aligned squad density,
platform-squad coverage, enabling-squad scope). Each check
contributes a row to the output with CHECK name, PASS/FAIL
status, and a DETAIL string. The command exits non-zero when any
check fails so it can gate CI.

Read-only: no state mutation. Idempotent. The output respects the
table renderer and works on a TTY (styled table) and non-TTY
(plain tabwriter) alike.

```
aps squad check [flags]
```

### Options

```
  -h, --help   help for check
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)

## aps squad create

Create a new squad

### Synopsis

Create a new squad entry in the local squad store. The ID is
derived from the <name> argument by lower-casing and replacing
spaces with dashes. --type and --domain are required: --type must
be one of the four team-topology values (stream-aligned, enabling,
complicated-subsystem, platform), and --domain names the squad's
domain boundary. --description and --members are optional; the
members slice takes a comma-separated list of profile IDs.

Mints a new local record. Not idempotent — re-running with the
same name on an existing squad fails. --dry-run is opted out
because the resulting record is fully determined by the flags.
Use --note to attach an audit reason that flows to the event bus
alongside the mutation.

```
aps squad create <name> [flags]
```

### Options

```
      --description string   Squad description
      --domain string        Domain boundary
  -h, --help                 help for create
      --members strings      Comma-separated profile IDs
  -n, --note string          Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --type string          Squad type (stream-aligned, enabling, complicated-subsystem, platform)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)

## aps squad delete

Delete a squad

### Synopsis

Remove the named squad from the local squad store. The squad
ID is the slug shown by aps squad list, not the human-readable
name. Member profile records are not touched — deleting a squad
only drops the squad entry and any topology it carried.

Destructive: the squad record is removed irreversibly. The
destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would only restate the
squad ID. Idempotent on the pair. Use --note to attach an audit
reason that flows to the event bus alongside the mutation.

```
aps squad delete <id> [flags]
```

### Options

```
  -h, --help          help for delete
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)

## aps squad list

List all squads

### Synopsis

List every squad in the local squad store. Each row reports
the squad ID, name, team-topology type (stream-aligned, enabling,
complicated-subsystem, platform), domain boundary, member count,
and a truncated member-ID list (first three IDs plus "+N" when the
squad has more).

The output respects the global --format flag (table|json|yaml) and
the local filter flags: --member selects squads containing a given
profile ID, and --role selects by team-topology type. Read-only:
no state mutation. Idempotent.

```
aps squad list [flags]
```

### Options

```
  -h, --help            help for list
      --member string   Filter to squads containing this profile ID
      --role string     Filter by squad type (stream-aligned, enabling, complicated-subsystem, platform)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)

## aps squad members

Manage squad membership

### Options

```
  -h, --help   help for members
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)
* [aps squad members add](#aps-squad-members-add)	 - Add a member to a squad
* [aps squad members remove](#aps-squad-members-remove)	 - Remove a member from a squad

## aps squad members add

Add a member to a squad

### Synopsis

Append a profile to a squad's member list. Both arguments are
positional: <squad-id> is the squad slug (see aps squad list) and
<profile-id> is the profile ID. The squad must exist; the profile
ID is recorded verbatim and is not cross-validated against the
profile store at this layer.

Local write to the squad record only — pair with aps squad members
remove to undo. Not idempotent at the manager level (re-running
returns an error on duplicate membership). --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.

```
aps squad members add <squad-id> <profile-id> [flags]
```

### Options

```
  -h, --help          help for add
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad members](#aps-squad-members)	 - Manage squad membership

## aps squad members remove

Remove a member from a squad

### Synopsis

Drop a profile from a squad's member list. Both arguments are
positional: <squad-id> is the squad slug (see aps squad list) and
<profile-id> is the profile ID to remove. The profile record
itself is untouched — only the squad's membership entry is
dropped.

Destructive at the squad level: the membership entry is removed
irreversibly from the squad store. The destructive-token
confirmation flow gates the apply path, and --dry-run is opted out
because preview would only restate the two positional arguments.
Idempotent on already-absent membership.

```
aps squad members remove <squad-id> <profile-id> [flags]
```

### Options

```
  -h, --help          help for remove
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad members](#aps-squad-members)	 - Manage squad membership

## aps squad show

Show squad details

### Synopsis

Print the full record for a single squad — ID, name,
team-topology type, domain boundary, description, the complete
member ID list (no truncation, unlike aps squad list), and the
created/updated timestamps. The squad is looked up by ID; unknown
IDs return an error from the underlying squad manager.

Read-only: no state mutation. Idempotent.

```
aps squad show <id> [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps squad](#aps-squad)	 - Manage agent squads (topology, membership, scope)

## aps status

Show aps configuration and runtime status

### Synopsis

Show a snapshot of the current aps configuration and runtime state.

Reports the active profile, active workspace, build version, and event-bus
connection state. Read-only: no network calls, no state mutation. Intended
as a quick "is aps wired up" probe for users and agents.

Output respects the --format global (table|json|yaml). Empty profile or
workspace values render as "(none)" to distinguish unset from blank.

```
aps status [flags]
```

### Options

```
  -h, --help   help for status
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps toolspec

Print the aps tool specification (commands, flags, errors, workflows)

### Synopsis

Output a structured ToolSpec describing every aps command, persistent
flag, known error pattern, and canonical workflow. Intended for agent
consumption — point an LLM tool runner at this output to teach it how to
drive aps without parsing --help.

```
aps toolspec [flags]
```

### Options

```
  -h, --help   help for toolspec
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps upgrade

Check for and install updates

### Synopsis

Check for a newer version of aps and optionally install it.

```
aps upgrade [flags]
```

### Options

```
      --auto   Install without prompting
  -h, --help   help for upgrade
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps upgrade preamble](#aps-upgrade-preamble)	 - Print the upgrade preamble fragment for skill files

## aps upgrade preamble

Print the upgrade preamble fragment for skill files

### Synopsis

Print a markdown preamble fragment for embedding in APS skill files.
Agents read this to know how to self-upgrade aps before executing tasks.

```
aps upgrade preamble [flags]
```

### Options

```
      --auto      Emit auto-upgrade (SnoozeNever) variant
  -h, --help      help for preamble
      --install   Write preamble to ~/.config/aps/skills/
      --never     Emit check-only (SnoozeAlways) variant
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps upgrade](#aps-upgrade)	 - Check for and install updates

## aps version

Print version information

### Synopsis

Print detailed version information including build metadata.

```
aps version [flags]
```

### Options

```
  -h, --help    help for version
      --short   Output only the version number
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI

## aps voice

Manage voice sessions and the voice backend service

### Options

```
  -h, --help   help for voice
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps voice service](#aps-voice-service)	 - Control the voice backend service
* [aps voice start](#aps-voice-start)	 - Start a voice session

## aps voice service

Control the voice backend service

### Options

```
  -h, --help   help for service
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps voice](#aps-voice)	 - Manage voice sessions and the voice backend service
* [aps voice service start](#aps-voice-service-start)	 - Start the voice backend service
* [aps voice service status](#aps-voice-service-status)	 - Show voice backend service status
* [aps voice service stop](#aps-voice-service-stop)	 - Stop the voice backend service

## aps voice service start

Start the voice backend service

### Synopsis

Start the long-running voice backend daemon that hosts aps
voice sessions. The daemon owns the audio-capture pipeline and
the realtime STT/TTS provider connections; aps voice start
(per-session command) talks to this daemon.

Idempotent: starting an already-running backend is a no-op
(NewBackendManager().Start returns nil when the process is already
up). --dry-run is opted out because previewing the spawn would
have to bisect the spawn-and-wait path that defines the
operation. Pair with aps voice service stop to terminate and aps
voice service status to inspect.

```
aps voice service start [flags]
```

### Options

```
  -h, --help   help for start
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps voice service](#aps-voice-service)	 - Control the voice backend service

## aps voice service status

Show voice backend service status

### Synopsis

Print the voice backend daemon's running state: "running"
if the manager reports the process is up, "stopped" otherwise.
Used as a quick liveness probe before invoking aps voice start.

Read-only: no daemon state mutation. Idempotent.

```
aps voice service status [flags]
```

### Options

```
  -h, --help   help for status
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps voice service](#aps-voice-service)	 - Control the voice backend service

## aps voice service stop

Stop the voice backend service

### Synopsis

Stop the voice backend daemon by sending SIGTERM and waiting
for graceful cleanup. Any active aps voice sessions are torn down
when the daemon exits; new aps voice start calls will fail until
aps voice service start is run again.

Idempotent: stopping an already-stopped backend is a no-op.
--dry-run is opted out because previewing would have to fake the
OS signal path that defines the operation.

```
aps voice service stop [flags]
```

### Options

```
  -h, --help   help for stop
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps voice service](#aps-voice-service)	 - Control the voice backend service

## aps voice start

Start a voice session

### Synopsis

Register a fresh voice session against the running voice
backend daemon and print the assigned session id. The session
binds to the named profile (via the inherited --profile global,
required) and the channel selected by --channel (web | tui |
telegram | twilio; default web).

Mints a new session record per call (not idempotent). The backend
daemon must already be running (see aps voice service start);
otherwise this call fails with a backend-unreachable error.
--dry-run is opted out because previewing would have to fake the
audio-stream handshake that defines the session.

```
aps voice start [flags]
```

### Options

```
      --channel string   Channel: web | tui | telegram | twilio (default "web")
  -h, --help             help for start
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps voice](#aps-voice)	 - Manage voice sessions and the voice backend service

## aps webhook

Manage Webhook server

### Synopsis

Manage Webhook server for event-driven task execution.

The webhook command group provides operations for:
- Enabling and configuring webhooks for a profile
- Starting a webhook server
- Managing webhook event mappings

### Options

```
  -h, --help   help for webhook
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps webhook server](#aps-webhook-server)	 - Start a webhook server
* [aps webhook toggle](#aps-webhook-toggle)	 - Enable or disable Webhook for a profile

## aps webhook server

Start a webhook server

### Synopsis

Start a webhook server to receive and process webhook events.

The server listens on the specified address and processes incoming webhook
events by mapping them to profile actions.

If --profile is provided and webhooks are not enabled, will auto-enable them.

Examples:
  aps webhook server
  aps webhook server --addr 0.0.0.0:9000
  aps webhook server --profile worker --secret my-secret
  aps webhook server --secret my-secret --event-map github=profile:action

```
aps webhook server [flags]
```

### Options

```
      --addr string           Address to listen on (default "127.0.0.1:8080")
      --allow-event strings   Allowed event types
      --event-map strings     Map event to action (event=profile:action)
  -h, --help                  help for server
      --secret string         Shared secret for HMAC validation
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps webhook](#aps-webhook)	 - Manage Webhook server

## aps webhook toggle

Enable or disable Webhook for a profile

### Synopsis

Enable or disable Webhook server for a profile.

Without --enabled flag, toggles the current state (enables if not configured).
With --enabled=on, forces enable. With --enabled=off, forces disable.

Examples:
  aps webhook toggle --profile worker                    # Toggle Webhook
  aps webhook toggle --profile worker --enabled=on      # Force enable
  aps webhook toggle --profile worker --enabled=off     # Force disable

```
aps webhook toggle [flags]
```

### Options

```
      --enabled string   Enable (on), disable (off), or toggle (omit or blank)
  -h, --help             help for toggle
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps webhook](#aps-webhook)	 - Manage Webhook server

## aps workspace

Manage multi-agent workspaces

### Synopsis

Manage workspaces for multi-agent collaboration.

A workspace is a shared context where agents coordinate, exchange
tasks, share context, and resolve conflicts. Set an active
workspace to avoid repeating its name:

  aps workspace use my-team
  aps workspace members      # uses active workspace

### Options

```
  -h, --help   help for workspace
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps](#aps)	 - Agent Profile System CLI
* [aps workspace activity](#aps-workspace-activity)	 - Show workspace activity log
* [aps workspace agents](#aps-workspace-agents)	 - Find agents by capability
* [aps workspace archive](#aps-workspace-archive)	 - Archive a collaboration workspace
* [aps workspace audit](#aps-workspace-audit)	 - Show audit trail
* [aps workspace caps](#aps-workspace-caps)	 - List capabilities in a workspace
* [aps workspace conflicts](#aps-workspace-conflicts)	 - Manage workspace conflicts
* [aps workspace create](#aps-workspace-create)	 - Create a collaboration workspace
* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context
* [aps workspace join](#aps-workspace-join)	 - Join a collaboration workspace
* [aps workspace leave](#aps-workspace-leave)	 - Leave a collaboration workspace
* [aps workspace list](#aps-workspace-list)	 - List collaboration workspaces
* [aps workspace members](#aps-workspace-members)	 - List workspace members
* [aps workspace policy](#aps-workspace-policy)	 - Manage conflict resolution policy
* [aps workspace remove](#aps-workspace-remove)	 - Remove an agent from the workspace
* [aps workspace role](#aps-workspace-role)	 - Set an agent's role in the workspace
* [aps workspace send](#aps-workspace-send)	 - Send a task to an agent
* [aps workspace show](#aps-workspace-show)	 - Show workspace details
* [aps workspace sync](#aps-workspace-sync)	 - Sync workspace state
* [aps workspace task](#aps-workspace-task)	 - Show task details
* [aps workspace tasks](#aps-workspace-tasks)	 - List tasks in a workspace
* [aps workspace use](#aps-workspace-use)	 - Set active workspace

## aps workspace activity

Show workspace activity log

### Synopsis

Show recent activity events for a workspace.

Events are grouped by type: profile, action, device, workspace, conflict.
Use --follow to tail the log in real time.

```
aps workspace activity <workspace-id> [flags]
```

### Options

```
      --device string    Filter by device ID
      --exclude string   Exclude event type category
  -f, --follow           Follow the activity log (live tail)
  -h, --help             help for activity
      --json             JSON output
  -n, --limit int        Maximum number of events to display (default 50)
      --since string     Show events since duration (e.g. 1h, 30m, 7d) (default "24h")
  -t, --type string      Filter by event type: profile, action, device, workspace, conflict
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace agents

Find agents by capability

### Synopsis

Find agents in a workspace that match a capability or task description.

Use --cap for exact capability matching, or --task for fuzzy matching
against a task description.

```
aps workspace agents [flags]
```

### Options

```
      --cap string    Capability name to match
  -h, --help          help for agents
      --json          Output as JSON
      --task string   Task description for fuzzy matching
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace archive

Archive a collaboration workspace

### Synopsis

Archive a collaboration workspace. Archived workspaces are read-only
and cannot accept new agents or tasks. Use --force to skip confirmation.

```
aps workspace archive [workspace] [flags]
```

### Options

```
      --force         Skip confirmation
  -h, --help          help for archive
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace audit

Show audit trail

### Synopsis

Display the audit trail for a collaboration workspace.

All state-changing operations are recorded: agent joins, task
creation, conflict resolution, context mutations, and more.

Filter with --since, --actor, --event, and --limit.

```
aps workspace audit [workspace] [flags]
```

### Options

```
      --actor string   Filter by actor (agent ID)
      --event string   Filter by event type (supports glob, e.g. task.*)
  -h, --help           help for audit
      --json           Output as JSON
      --limit int      Maximum results (default 25)
      --since string   Show events since duration (e.g. 1h, 24h)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace caps

List capabilities in a workspace

### Synopsis

List all registered capabilities in a collaboration workspace.

Optionally filter by a specific agent using --agent.

```
aps workspace caps [flags]
```

### Options

```
      --agent string   Filter by agent profile ID
  -h, --help           help for caps
      --json           Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace conflicts

Manage workspace conflicts

### Synopsis

Manage conflicts that occur when multiple devices modify
the same workspace resource concurrently.

Conflicts are detected automatically during sync. Use these
commands to list, inspect, and resolve them.

### Options

```
  -h, --help   help for conflicts
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces
* [aps workspace conflicts list](#aps-workspace-conflicts-list)	 - List workspace conflicts
* [aps workspace conflicts resolve](#aps-workspace-conflicts-resolve)	 - Resolve a conflict
* [aps workspace conflicts show](#aps-workspace-conflicts-show)	 - Show conflict details

## aps workspace conflicts list

List workspace conflicts

### Synopsis

List conflicts detected in a workspace.

By default, all detected conflicts are listed (pending, auto-resolved
awaiting review, manual, and fully resolved). Use --unresolved to
narrow to only conflicts still requiring attention.

The --workspace flag is a global (T-0376) and inherits from the active
workspace when not supplied.

```
aps workspace conflicts list [flags]
```

### Options

```
  -h, --help         help for list
      --unresolved   Show only conflicts not yet fully resolved
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace conflicts](#aps-workspace-conflicts)	 - Manage workspace conflicts

## aps workspace conflicts resolve

Resolve a conflict

### Synopsis

Resolve a workspace conflict using the specified strategy.

Strategies:
  lww     Last-write-wins: the most recent event wins
  manual  Choose a specific event as the winner

Use --dry-run to see what would happen without applying changes.

```
aps workspace conflicts resolve <conflict-id> [flags]
```

### Options

```
      --choice string     Event ID to choose as winner (for manual strategy)
      --force             Skip confirmation prompt
  -h, --help              help for resolve
      --json              JSON output
      --strategy string   Resolution strategy: lww, manual (default "lww")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace conflicts](#aps-workspace-conflicts)	 - Manage workspace conflicts

## aps workspace conflicts show

Show conflict details

### Synopsis

Show full details of a conflict including both versions and their values.

```
aps workspace conflicts show <conflict-id> [flags]
```

### Options

```
  -h, --help   help for show
      --json   JSON output
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace conflicts](#aps-workspace-conflicts)	 - Manage workspace conflicts

## aps workspace create

Create a collaboration workspace

### Synopsis

Create a new multi-agent collaboration workspace.

A workspace provides a shared context where agents coordinate,
exchange tasks, and resolve conflicts.

```
aps workspace create <name> [flags]
```

### Options

```
      --description string         Workspace description
  -h, --help                       help for create
      --json                       Output as JSON
  -n, --note string                Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --resolution-policy string   Conflict resolution policy (default "priority")
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace ctx

Manage workspace shared context

### Synopsis

Manage shared context variables in a collaboration workspace.

Context variables are key-value pairs visible to all agents.
Changes are tracked with version history and ACL enforcement.

### Options

```
  -h, --help   help for ctx
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces
* [aps workspace ctx delete](#aps-workspace-ctx-delete)	 - Delete a context variable
* [aps workspace ctx get](#aps-workspace-ctx-get)	 - Get a context variable
* [aps workspace ctx history](#aps-workspace-ctx-history)	 - Show mutation history for a variable
* [aps workspace ctx list](#aps-workspace-ctx-list)	 - List all context variables
* [aps workspace ctx set](#aps-workspace-ctx-set)	 - Set a context variable

## aps workspace ctx delete

Delete a context variable

### Synopsis

Remove a key from the workspace's shared context store and
persist the truncated snapshot. The delete is mediated by the
T-1292 policy gate: the kit pre_persisted topic is published with
Op=delete and kind=workspace_context before the in-memory mutation
fans out, and CEL rules can veto via context.request_attrs (the
variable's visibility and the calling workspace_id are stuffed in
for the principal/role resolver to read).

--workspace inherits from the active workspace, --profile names
the deleting agent, and --note (T-1291) attaches a free-form
rationale that flows into the policy context. Marked
SideEffectDestructiveLocal: a destructive-token confirm gates the
apply path (--force bypasses) and --dry-run is opted out because
preview would only restate the key argument. Deleting a key that
is absent (or private and owned by another profile, per T-1309)
returns a not-found error rather than silently succeeding.

```
aps workspace ctx delete <key> [flags]
```

### Options

```
      --force         Skip confirmation
  -h, --help          help for delete
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context

## aps workspace ctx get

Get a context variable

### Synopsis

Read a single context variable from the active or selected
workspace and print its value to stdout.

The lookup honours the T-1309 visibility filter: when --profile (or
the inherited APS_PROFILE global) is set, private variables owned
by other profiles are hidden and the key reads as not found.
Passing no profile falls through to the raw view used by tooling
and tests. --workspace inherits from the active workspace (T-0376);
--json swaps the bare-value output for the full ContextVariable
record. Companion: aps workspace ctx set, ctx list, ctx history.

Read-only; idempotent.

```
aps workspace ctx get <key> [flags]
```

### Options

```
  -h, --help   help for get
      --json   Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context

## aps workspace ctx history

Show mutation history for a variable

### Synopsis

Print the ordered mutation log for a single context variable
in the active or selected workspace. Each entry records the
version, mutating agent, old value, new value, and timestamp;
values are truncated to 30 characters in the table view (use
--json for the full payload).

--limit N returns the most recent N entries (keeps the tail of
the slice); --workspace inherits from the active workspace
global. The default styled-table output trims to a TTY-aware
column set, and --json emits the raw []ContextMutation slice.
Read-only; idempotent.

```
aps workspace ctx history <key> [flags]
```

### Options

```
  -h, --help        help for history
      --json        Output as JSON
      --limit int   Maximum results (default 25)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context

## aps workspace ctx list

List all context variables

### Synopsis

List context variables in the active or selected workspace.

The --workspace flag is a global (T-0376) and inherits from the
active workspace when not supplied. Use --key-prefix to narrow
the listing to keys starting with a given string.

```
aps workspace ctx list [flags]
```

### Options

```
  -h, --help                help for list
      --key-prefix string   Filter to keys with this prefix (set membership)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context

## aps workspace ctx set

Set a context variable

### Synopsis

Set a context variable in the workspace.

By default variables are workspace-wide ("shared"): visible to every
member, gated by the per-key ACL. Pass --private to scope the
variable to the current profile only — private variables are
invisible to other profiles' workspace ctx get/list (T-1309).

```
aps workspace ctx set <key> <value> [flags]
```

### Options

```
  -h, --help          help for set
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --private       Scope variable to the current profile (default: shared workspace-wide)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace ctx](#aps-workspace-ctx)	 - Manage workspace shared context

## aps workspace join

Join a collaboration workspace

### Synopsis

Join an existing collaboration workspace as a contributor.

You must specify your profile to identify which agent is joining.

```
aps workspace join <workspace> [flags]
```

### Options

```
  -h, --help          help for join
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace leave

Leave a collaboration workspace

### Synopsis

Leave a collaboration workspace. Use --force to skip confirmation.

If no workspace is specified, the active workspace is used.

```
aps workspace leave [workspace] [flags]
```

### Options

```
      --force         Skip confirmation
  -h, --help          help for leave
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace list

List collaboration workspaces

### Synopsis

List all collaboration workspaces with summary information.

```
aps workspace list [flags]
```

### Options

```
      --archived        Filter to archived (true) or non-archived (false) workspaces
  -h, --help            help for list
      --limit int       Maximum results (default 25)
      --member string   Filter to workspaces containing this profile ID
      --owner string    Filter by workspace owner profile ID
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace members

List workspace members

### Synopsis

List all agents that are members of a collaboration workspace.

```
aps workspace members [workspace] [flags]
```

### Options

```
  -h, --help   help for members
      --json   Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace policy

Manage conflict resolution policy

### Synopsis

View or set the conflict resolution policy for a workspace.

Use --set to change the default policy. Use --key with --set to
configure an override for a specific resource key.

Valid strategies:
  priority     Resolve by agent role priority (default)
  keep-first   Keep the earliest write
  keep-last    Keep the most recent write
  rollback     Revert to pre-conflict value
  consensus    Require agent agreement
  voting       Majority vote
  merge        Attempt to merge changes

```
aps workspace policy [workspace] [flags]
```

### Options

```
  -h, --help          help for policy
      --json          Output as JSON
      --key string    Resource key for override (use with --set)
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --set string    Set default resolution strategy
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace remove

Remove an agent from the workspace

### Synopsis

Remove an agent from a collaboration workspace. Only the workspace owner
can perform this action. Use --force to skip confirmation.

```
aps workspace remove <agent> [workspace] [flags]
```

### Options

```
      --force         Skip confirmation
  -h, --help          help for remove
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace role

Set an agent's role in the workspace

### Synopsis

Change an agent's role within a collaboration workspace.

Valid roles: owner, contributor, observer

If role is not provided, an interactive selector is shown.

```
aps workspace role <agent> [role] [workspace] [flags]
```

### Options

```
  -h, --help          help for role
      --json          Output as JSON
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace send

Send a task to an agent

### Synopsis

Send a task to another agent in the workspace.

Input can be provided as:
  --input '{"key": "value"}'   JSON string
  --input @file.json           Read from file
  --input -                    Read from stdin
  --set key=value              Shorthand for simple key-value input

```
aps workspace send <recipient> [flags]
```

### Options

```
      --action string    Task action (required)
  -h, --help             help for send
      --input string     Task input (JSON string, @file, or - for stdin)
      --json             Output as JSON
  -n, --note string      Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
      --set strings      Set key=value input pairs
      --timeout string   Task timeout (e.g. 5m, 1h)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace show

Show workspace details

### Synopsis

Display detailed information about a collaboration workspace.

```
aps workspace show [workspace] [flags]
```

### Options

```
  -h, --help   help for show
      --json   Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace sync

Sync workspace state

### Synopsis

Sync workspace state for the current device.

Normally sync happens automatically when devices connect.
Use this command when:
  - A device was offline and you want to force a sync
  - You suspect sync is behind and want to catch up
  - You want to verify sync status

```
aps workspace sync <workspace-id> [flags]
```

### Options

```
      --device string   Device ID to sync (defaults to current device)
  -h, --help            help for sync
      --json            JSON output
  -n, --note string     Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace task

Show task details

### Synopsis

Display detailed information about a specific inter-agent task.

```
aps workspace task <task-id> [flags]
```

### Options

```
  -h, --help   help for task
      --json   Output as JSON
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace tasks

List tasks in a workspace

### Synopsis

List inter-agent tasks in a collaboration workspace.

```
aps workspace tasks [workspace] [flags]
```

### Options

```
      --agent string    Filter by agent (sender or recipient)
  -h, --help            help for tasks
      --json            Output as JSON
      --limit int       Maximum results (default 25)
      --status string   Filter by status (submitted, working, completed, failed, cancelled)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

## aps workspace use

Set active workspace

### Synopsis

Set the active collaboration workspace.

Once set, other collab commands will use this workspace by default
when no --workspace flag is provided.

```
aps workspace use <workspace> [flags]
```

### Options

```
  -h, --help          help for use
  -n, --note string   Audit note (recorded against the bus event payload and exposed to policy engine via context.note)
```

### Options inherited from parent commands

```
      --format string      Output format (csv, human, json, table, text, yaml) (default "table")
      --instance string    backend instance to target (defaults to config)
      --no-color           Disable ANSI color
      --no-hints           Suppress next-step hints after command output
      --no-redact          disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline string     disable all network calls
  -p, --profile string     profile id (defaults to active profile)
      --quiet              Suppress non-essential output
  -V, --verbose count      Increase log verbosity (-V=debug, -VV=trace)
      --workspace string   workspace id (defaults to active workspace)
```

### SEE ALSO

* [aps workspace](#aps-workspace)	 - Manage multi-agent workspaces

<!-- [[[end]]] -->
