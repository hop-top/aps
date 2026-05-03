# APS CLI Reference

Complete reference for all APS CLI commands.

## Overview

```bash
aps [command] [flags]
```

Run `aps` without arguments to launch the TUI.

## Commands

### `aps`

Launch the TUI (Terminal User Interface).

```bash
aps
```

### `aps help`

Get help for any command.

```bash
aps help
aps help profile
aps help run
```

## Profile Commands

### `aps profile list`

List all available profiles.

```bash
aps profile list
```

**Output:**
```
myagent
test-agent
dev-agent
```

### `aps profile create`

Create a new profile.

```bash
aps profile create <profile-id> [flags]
```

**Flags:**

- `--display-name <name>` - Human-readable name for the profile
- `--email <email>` - Email for git config
- `--github <username>` - GitHub username
- `--reddit <username>` - Reddit username
- `--twitter <username>` - Twitter/X username
- `--force` - Overwrite existing profile

**Examples:**

```bash
# Minimal profile
aps profile create myagent

# Profile with display name
aps profile create myagent --display-name "My AI Agent"

# Profile with git config
aps profile create myagent --display-name "Agent" --email "agent@example.com"

# Profile with social accounts
aps profile openai-agent \
  --display-name "OpenAI Agent" \
  --email "openai@example.com" \
  --github "openai-bot"

# Overwrite existing profile
aps profile create myagent --force
```

**What gets created:**

- `~/.agents/profiles/<id>/profile.yaml`
- `~/.agents/profiles/<id>/secrets.env` (chmod 0600)
- `~/.agents/profiles/<id>/notes.md`
- `~/.agents/profiles/<id>/gitconfig` (if email provided)
- `~/.agents/profiles/<id>/actions/` directory

### `aps profile show`

Display profile details.

```bash
aps profile show <profile-id>
```

**Output:**

```
id: myagent
display_name: My AI Agent
preferences:
  shell: /bin/zsh
modules:
  gitconfig: present
  secrets.env: present (2 keys)
  actions: 3 actions found
```

Secret values are redacted (shown as `***redacted***`).

## Run Commands

### `aps run`

Run a command under a profile.

```bash
aps run <profile-id> -- <command> [args...]
```

**Important:** The `--` separator is required.

**Examples:**

```bash
# Run echo command
aps run myagent -- echo "Hello!"

# Run shell command
aps run myagent -- ls -la

# Run git with profile's git config
aps run myagent -- git status

# Run multiple commands with shell
aps run myagent -- sh -c "cd /tmp && ls"

# Run a Node.js script
aps run myagent -- node script.js

# Run Python with profile's environment
aps run myagent -- python main.py
```

### Shorthand Execution

You can skip the `run` subcommand:

```bash
# These are equivalent
aps run myagent -- echo "Hello"
aps myagent -- echo "Hello"
```

**Important:** If you don't provide a command after `--`, APS starts an interactive shell session:

```bash
# Starts shell configured in profile (defaults to your login shell)
aps myagent
```

## Action Commands

### `aps action list`

List available actions for a profile.

```bash
aps action list <profile-id>
```

**Output:**

```
hello-world.sh
greet-user.py
process-data.js
```

If `actions.yaml` exists, titles are shown:

```
hello-world.sh - Say hello
greet-user.py - Greet the current user
process-data.js - Process JSON data
```

### `aps action show`

Show action details.

```bash
aps action show <profile-id> <action-id>
```

**Output:**

```
action: hello-world.sh
type: sh
path: /home/user/.agents/profiles/myagent/actions/hello-world.sh
accepts_stdin: true
```

### `aps action run`

Run an action script.

```bash
aps action run <profile-id> <action-id> [flags]
```

**Flags:**

- `--payload-file <path>` - Send file contents to stdin
- `--payload-stdin` - Read stdin and forward to action
- `--dry-run` - Don't execute, just show what would happen

**Examples:**

```bash
# Run action without input
aps action run myagent hello-world.sh

# Run action with file input
aps action run myagent process-data.js --payload-file data.json

# Run action with stdin input
echo '{"name": "John"}' | aps action run myagent greet.py --payload-stdin

# Dry run (don't execute)
aps action run myagent hello-world.sh --dry-run
```

## Capability Commands

### `aps capability list`

List all capabilities (builtin + external).

```bash
aps cap list [--profile <id>] [--json]
```

**Flags:**

- `--profile <id>` - Filter to capabilities on a specific profile
- `--json` - JSON output

**Output:**
```
Capabilities

NAME              KIND      TYPE        LINKS  PROFILES
a2a               builtin   --          --     work, dev
webhooks          builtin   --          --     work
my-vim-config     external  managed     2      work
windsurf-agent    external  reference   1      dev

4 capabilities (2 builtin, 2 external)
```

### `aps capability show`

Show capability details.

```bash
aps cap show <name>
```

### `aps capability install`

Install a capability from a source directory.

```bash
aps cap install <source> --name <name>
```

### `aps capability link`

Symlink a capability to a target path.

```bash
aps cap link <name> [--target <path>]
```

Supports smart linking: `aps cap link my-cap windsurf` resolves
to the Windsurf default path.

### `aps capability delete`

Delete a capability.

```bash
aps cap delete <name> [--force]
```

Warns about active links unless `--force` is provided.

### `aps capability adopt`

Move a file/dir into APS and symlink back.

```bash
aps cap adopt <path> --name <name>
```

### `aps capability watch`

Watch an external file (symlink into APS).

```bash
aps cap watch <path> --name <name>
aps cap watch --tool <tool> --name <name>
```

### `aps capability patterns list`

Show smart patterns and builtin capabilities.

```bash
aps cap patterns list
```

### `aps capability enable`

Enable a capability on a profile.

```bash
aps cap enable <profile> <capability>
```

### `aps capability disable`

Disable a capability on a profile.

```bash
aps cap disable <profile> <capability>
```

### `aps profile capability add`

Add a capability to a profile.

```bash
aps profile capability add <profile> <capability>
```

### `aps profile capability remove`

Remove a capability from a profile.

```bash
aps profile capability remove <profile> <capability>
```

## Session Commands

### `aps session list`

List active and recent sessions.

```bash
aps session list [flags]
```

**Flags:**

- `--profile <id>` - Filter sessions by profile ID
- `--status <status>` - Filter sessions by status (active, inactive, errored)
- `--tier <tier>` - Filter sessions by tier (basic, standard, premium)

**Examples:**

```bash
# List all sessions
aps session list

# List sessions for a specific profile
aps session list --profile myagent

# List active sessions
aps session list --status active

# List premium tier sessions
aps session list --tier premium

# Combine filters
aps session list --profile myagent --status active --tier premium
```

**Output:**
```
ID          PROFILE   PID   STATUS   TIER      CREATED              LAST SEEN
session-123 myagent   12345 active   basic     2026-01-21 10:30:00  10:35:00
session-124 prod      12346 active   standard  2026-01-21 10:32:00  10:34:00
session-125 dev       0     inactive basic     2026-01-21 09:00:00  09:15:00
```

### `aps session attach`

Attach to a running session.

```bash
aps session attach <session-id>
```

**Note:** Full attach functionality is coming soon. Currently displays session information.

### `aps session detach`

Detach from a running session.

```bash
aps session detach <session-id>
```

**Note:** Full detach functionality is coming soon. Currently displays session information.

## Webhook Commands

### `aps webhook serve`

Start webhook server for event-driven execution.

```bash
aps webhook serve [flags]
```

**Flags:**

- `--addr <ip:port>` - Bind address (default: 127.0.0.1:8080)
- `--profile <profile-id>` - Default profile
- `--secret <secret>` - Shared secret for signature validation
- `--event-map <event=profile:action>` - Map events to profile:action (repeatable)
- `--allow-event <event>` - Allow only specific events (repeatable)
- `--dry-run` - Dry run mode

**Examples:**

```bash
# Simple server
aps webhook serve --addr 127.0.0.1:8080

# With event mapping
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --event-map github.push=myagent:deploy.sh \
  --event-map github.issue_comment.created=myagent:respond.py

# With signature validation
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --secret "my-secret-key" \
  --event-map build.requested=myagent:build.sh

# With event allowlist
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --allow-event github.push \
  --allow-event github.issue_comment.created

# Dry run (don't execute actions)
aps webhook serve --dry-run
```

**Endpoints:**

- `POST /webhook` - Receive webhook events
- `GET /healthz` - Health check (optional)

**Request Headers:**

- `X-APS-Event: <event>` - Event type (e.g., `github.push`)
- `X-APS-Signature: sha256=<hex>` - HMAC signature (if secret configured)

**Response:**

Success (200):
```json
{
  "delivery_id": "uuid",
  "event": "github.push",
  "profile": "myagent",
  "action": "deploy.sh",
  "status": "executed"
}
```

## Utility Commands

### `aps docs`

Generate documentation to `~/.agents/docs/`.

```bash
aps docs
```

**Output:**
```
Generated /home/user/.agents/docs/README.md
Generated /home/user/.agents/docs/CLI.md
Generated /home/user/.agents/docs/PROFILES.md
Generated /home/user/.agents/docs/ISOLATION.md
Generated /home/user/.agents/docs/SESSIONS.md
Generated /home/user/.agents/docs/SECURITY.md
Generated /home/user/.agents/docs/EXAMPLES.md
Generated /home/user/.agents/docs/WEBHOOKS.md
```

### `aps completion`

Generate shell completion scripts.

```bash
aps completion bash
aps completion zsh
aps completion fish
aps completion powershell
```

**Install:**

```bash
# Zsh
echo 'source <(aps completion zsh)' >> ~/.zshrc

# Bash
echo 'source <(aps completion bash)' >> ~/.bashrc
```

### `aps alias`

Generate shell aliases for all profiles.

```bash
aps alias
```

**Output:**
```
alias myagent='aps myagent'
alias test-agent='aps test-agent'
```

**Install:**

```bash
echo 'eval "$(aps alias)"' >> ~/.zshrc
```

Then you can run:
```bash
myagent echo "Hello!"
```

## Global Flags

None currently supported. All configuration is done via:
- Command-line flags
- Global config file (`~/.config/aps/config.yaml`)
- Profile configuration (`~/.agents/profiles/<id>/profile.yaml`)

## Exit Codes

- `0` - Success
- `1` - General error
- `>1` - Exit code from executed command/action

## Environment Variables

APS respects:

- `XDG_CONFIG_HOME` - Override config directory (default: `~/.config`)
- `HOME` - User home directory (for `~/.agents` and `~/.config`)

## Profile Environment Variables

When running commands/actions, APS injects these environment variables:

- `<PREFIX>_PROFILE_ID` - Profile ID (default prefix: `APS`)
- `<PREFIX>_PROFILE_DIR` - Path to profile directory
- `<PREFIX>_PROFILE_YAML` - Path to profile.yaml
- `<PREFIX>_PROFILE_SECRETS` - Path to secrets.env
- `<PREFIX>_PROFILE_DOCS_DIR` - Path to docs directory
- `GIT_CONFIG_GLOBAL` - Path to gitconfig (if module enabled)
- `GIT_SSH_COMMAND` - SSH command (if SSH module enabled)
- `APS_WEBHOOK_EVENT` - Webhook event type (if triggered by webhook)
- `APS_WEBHOOK_DELIVERY_ID` - Webhook delivery ID (if triggered by webhook)
- `APS_WEBHOOK_SOURCE_IP` - Webhook source IP (if triggered by webhook)

Plus all variables from `secrets.env`.

## Examples Directory

Use templates from `~/.agents/templates/` to bootstrap new profiles and actions.
