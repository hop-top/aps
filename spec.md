# APS (Agent Profile System) — Specification

## 1) Purpose

APS is a local-first Agent Profile System that enables running commands and agent workflows under isolated profiles containing:

- Identity (Git, GitHub, Reddit, Twitter/X, etc.)
- Credentials (tokens/keys)
- Capability flags (allowed actions/operations)
- Persona traits (tone, verbosity, risk tolerance)
- Runtime preferences (timezone, language, concurrency)
- Optional SSH identity selection per profile
- Event-driven execution via on-demand webhooks

APS provides:

- A scriptable CLI interface
- A first-class TUI interface
- A shared core engine used by both

---

## 2) Goals

- Simple, fast profile creation and management
- Process-scoped profile injection (no global shell state modifications required)
- Predictable filesystem contract for profiles, actions, templates, and docs
- Secrets handling that is safe by default
- Usable by agents written in any language
- Secure webhook-triggered action dispatch
- Every TUI action must be callable via CLI

---

## 3) Non-Goals

- Secrets vault integration
- OAuth flows and external account creation
- OS-level sandboxing
- Long-running daemon requirements by default

---

## 4) Implementation Requirements

### 4.1 Language & Packaging

- APS must be implemented as a single Go application.
- APS must compile to one binary named `aps`.
- Subcommands must be implemented internally (no libexec-style dispatching).

### 4.2 Architecture

APS must be structured as:

- **Core Engine** (shared internal package):
  - profile loading
  - secrets parsing
  - environment injection
  - action discovery and execution
  - command execution
  - webhook server and dispatch logic
  - docs generation

- **CLI Frontend**:
  - subcommands and flags
  - scriptable outputs and exit codes

- **TUI Frontend**:
  - interactive profile selection
  - interactive action selection and execution
  - log/output rendering
  - must call the Core Engine APIs (no duplicated execution logic)

### 4.3 Libraries

CLI framework:

- `cobra` or equivalent is acceptable

TUI/UX libraries (recommended):

- Charmbracelet Bubble Tea
- Charmbracelet Bubbles
- Charmbracelet Lip Gloss
- Charmbracelet Huh (optional)

Config parsing:

- YAML parsing for `profile.yaml`

Secrets parsing:

- dotenv parsing for `secrets.env`

HTTP server:

- Go `net/http`

Cryptography:

- Go `crypto/hmac`, `crypto/sha256`

Embedded assets:

- Go `embed`

---

## 5) Filesystem Layout

APS state must live under:

- `$HOME/.agents`

Structure:

```txt
~/.agents/
  profiles/
    <profile-id>/
      profile.yaml
      secrets.env
      gitconfig
      ssh.key
      actions/
        <action-id>.sh
        <action-id>.js
        <action-id>.py
      actions.yaml
      notes.md
  templates/
    profile.yaml.tpl
    secrets.env.tpl
    gitconfig.tpl
    notes.md.tpl
    actions.yaml.tpl
    actions/
      sample-action.sh.tpl
  docs/
    README.md
    SPEC.md
    CLI.md
    PROFILES.md
    SECURITY.md
    EXAMPLES.md
    WEBHOOKS.md
```

APS must embed templates and doc sources within the binary and write them out using `aps docs`.

---

## 6) Profile Contract

### 6.1 Profile Directory

A profile is stored at:

- `~/.agents/profiles/<profile-id>/`

A profile is considered valid when:

- `profile.yaml` exists
- `id` in `profile.yaml` matches `<profile-id>`

### 6.2 profile.yaml Schema

Required keys:

- `id` (string)
- `display_name` (string)

Optional keys:

- `persona` (map)
- `capabilities` (list of strings)
- `accounts` (map)
- `preferences` (map)
- `limits` (map)
- `git` (map)
- `ssh` (map)
- `webhooks` (map)

Example:

```yaml
id: agent-a
display_name: "Agent A"

persona:
  tone: "concise"
  style: "technical"
  risk: "low"

capabilities:
  - git
  - github
  - reddit
  - twitter
  - webhooks

accounts:
  github:
    username: "agent-a-handle"
  twitter:
    username: "agent_a"
  reddit:
    username: "agent_a_research"

preferences:
  language: "en"
  timezone: "America/Montreal"

limits:
  max_concurrency: 2
  max_runtime_minutes: 30

git:
  enabled: true

ssh:
  enabled: false
  key_path: "~/.ssh/agent-a_ed25519"

webhooks:
  enabled: true
  allowed_events:
    - "github.push"
    - "github.issue_comment.created"
```

---

## 7) Secrets Contract

### 7.1 secrets.env

`secrets.env` is dotenv format.

- Lines must be `KEY=VALUE` or `KEY="VALUE"`
- APS must parse this file and inject values into the environment of executed commands
- APS must never print raw secret values

Example:

```bash
GITHUB_TOKEN="..."
WEBHOOK_SHARED_SECRET="..."
```

### 7.2 Permissions

APS must:

- Create `secrets.env` with permissions `0600`
- Warn if permissions are more permissive than `0600`

### 7.3 Redaction Rules

When displaying secrets:

- show keys only
- values always appear as `***redacted***`

---

## 8) Modules

### 8.1 Git Module

If `gitconfig` exists in the profile directory, APS must inject:

- `GIT_CONFIG_GLOBAL=<profile-dir>/gitconfig`

### 8.2 SSH Module

If `ssh.key` exists in the profile directory and SSH is enabled, APS may inject:

- `GIT_SSH_COMMAND=ssh -i <profile-dir>/ssh.key -F /dev/null`

If `ssh.key` is missing while SSH is enabled:

- APS must warn and continue without injecting SSH

---

## 9) Environment Injection Contract

Whenever APS runs a command or an action within a profile, it must inject:

- `<PREFIX>_PROFILE_ID=<profile-id>`
- `<PREFIX>_PROFILE_DIR=<absolute profile dir>`
- `<PREFIX>_PROFILE_YAML=<absolute path to profile.yaml>`
- `<PREFIX>_PROFILE_SECRETS=<absolute path to secrets.env (even if missing)>`
- `<PREFIX>_PROFILE_DOCS_DIR=<absolute docs dir>`

The default `<PREFIX>` is `APS`. This can be overridden in the global configuration file at `$XDG_CONFIG_HOME/aps/config.yaml`.

APS must preserve the parent environment and then apply overrides for injected variables and secrets.

---

## 10) Actions

### 10.1 Actions Directory

A profile may contain actions under:

- `~/.agents/profiles/<profile-id>/actions/`

Action IDs map to scripts:

- `actions/<action-id>.sh`
- `actions/<action-id>.js`
- `actions/<action-id>.py`

### 10.2 Action Manifest (Optional)

A profile may contain:

- `actions.yaml`

Example:

```yaml
actions:
  - id: github-push
    title: "Handle GitHub push"
    entrypoint: "github-push.sh"
    accepts_stdin: true
  - id: triage-comment
    title: "Triage issue comment"
    entrypoint: "triage-comment.py"
    accepts_stdin: true
```

If `actions.yaml` exists:

- APS must use it to list actions and provide friendly titles/descriptions.

If `actions.yaml` is absent:

- APS must discover actions by scanning files in `actions/`.

### 10.3 Action Execution Rules

When executing an action, APS must:

- execute via the same environment injection rules as `aps run`
- pass payload to the action via stdin when provided
- attach stdio for interactive scripts unless explicitly disabled by flags
- return action exit code

---

## 11) Command Execution

APS must execute commands using Go process execution:

- `exec.Command`
- attached stdio:
  - stdin from parent (default)
  - stdout to parent (default)
  - stderr to parent (default)

APS must return the exact exit code from the invoked process.

---

## 12) CLI Commands

APS must implement the following CLI surface:

### 12.1 aps

Usage:

- `aps`

Behavior:

- Launch the TUI by default when called without arguments.

### 12.2 aps help

Usage:

- `aps help`
- `aps help <command>`

### 12.3 aps profile list

Usage:

- `aps profile list`

Output:

- one profile id per line

### 12.4 aps profile new

Usage:

- `aps profile new <profile-id> [flags]`

Flags:

- `--display-name "<name>"`
- `--email "<email>"`
- `--github "<username>"`
- `--reddit "<username>"`
- `--twitter "<username>"`
- `--force`

Requirements:

- Create profile directory
- Write `profile.yaml` from template
- Write `secrets.env` from template, chmod `0600`
- Write `gitconfig` if `--email` is provided
- Refuse overwrite unless `--force` is provided

### 12.5 aps profile show

Usage:

- `aps profile show <profile-id>`

Requirements:

- Print `profile.yaml`
- Print detected modules:
  - secrets present?
  - gitconfig present?
  - ssh.key present?
  - actions present?
- Print secret keys (redacted values)

### 12.6 aps run

Usage:

- `aps run <profile-id> -- <command> [args...]`

Requirements:

- Fail if profile does not exist
- Fail if `--` separator is missing
- Fail if command is missing after `--`
- Inject environment and run the command
- Exit with the command’s exit code

### 12.7 aps action list

Usage:

- `aps action list <profile-id>`

Output:

- list of available actions by id
- optionally include title if available

### 12.8 aps action show

Usage:

- `aps action show <profile-id> <action-id>`

Output:

- resolved entrypoint path
- whether action accepts stdin
- detected runtime type (sh/js/py)

### 12.9 aps action run

Usage:

- `aps action run <profile-id> <action-id> [flags]`

Flags:

- `--payload-file <path>` (send file bytes to stdin)
- `--payload-stdin` (read stdin and forward to action)
- `--dry-run` (do not execute; print resolution only)

Requirements:

- Resolve action script path
- Inject profile environment
- Execute action
- Exit with action exit code

### 12.10 aps docs

Usage:

- `aps docs`

Requirements:

- Write embedded docs into `~/.agents/docs/`
- Must generate:
  - `README.md`
  - `SPEC.md`
  - `CLI.md`
  - `PROFILES.md`
  - `SECURITY.md`
  - `EXAMPLES.md`
  - `WEBHOOKS.md`

---

## 13) TUI Requirements

The TUI must be launched by running:

- `aps`

Minimum required flows:

- Profile selection screen
- Profile details screen
- Action list screen for selected profile
- Action execution screen with output/log view

The TUI must execute actions and commands exclusively through the Core Engine APIs.

---

## 14) Webhook Support

### 14.1 Command: aps webhook serve

APS must run an HTTP webhook server on demand.

Usage:

- `aps webhook serve [flags]`

Flags:

- `--addr <ip:port>` (default: `127.0.0.1:8080`)
- `--profile <profile-id>`
- `--secret <shared-secret>`
- `--event-map <event=profile:action>` (repeatable)
- `--allow-event <event>` (repeatable)
- `--dry-run`

Examples:

```bash
aps webhook serve --addr 127.0.0.1:8080 --profile agent-a
```

```bash
aps webhook serve \
  --addr 127.0.0.1:8080 \
  --event-map github.push=agent-a:github-push \
  --event-map github.issue_comment.created=agent-b:triage-comment
```

### 14.2 Endpoints

Required:

- `POST /webhook`

Optional:

- `GET /healthz`

### 14.3 Event Identification

APS must determine the event type using:

- Header: `X-APS-Event: <event>`

Event naming convention:

- `provider.event_name`

Examples:

- `github.push`
- `github.issue_comment.created`
- `custom.build.requested`

### 14.4 Signature Validation

If `--secret` is configured, APS must validate an HMAC SHA256 signature:

- Header: `X-APS-Signature: sha256=<hex>`

Signature input:

- raw request body bytes

On invalid signature:

- respond `401 Unauthorized`

### 14.5 Allowlist Enforcement

If one or more `--allow-event` values are provided:

- deny any event not in the allowlist

### 14.6 Event Routing

APS must route webhook events to a profile + action using `--event-map`.

Map syntax:

- `<event>=<profile-id>:<action-id>`

If `--event-map` is provided and no mapping exists for the incoming event:

- respond `404 Not Found`

### 14.7 Dispatch Rules

On successful routing, APS must:

- execute the resolved action using the same injection rules as `aps action run`
- pass the webhook request body to the action’s stdin
- export webhook environment variables:

  - `APS_WEBHOOK_EVENT`
  - `APS_WEBHOOK_DELIVERY_ID`
  - `APS_WEBHOOK_SOURCE_IP`

APS should set `APS_WEBHOOK_DELIVERY_ID` from:

- `X-Request-Id` header if present
- otherwise a generated UUID

### 14.8 Responses

APS must respond with JSON.

Success response:

- status `200 OK`
- fields:
  - `delivery_id`
  - `event`
  - `profile`
  - `action`
  - `status` (`executed` or `dry_run`)

Failure response:

- appropriate `4xx/5xx`
- fields:
  - `delivery_id` (if available)
  - `event` (if available)
  - `error`

### 14.9 Logging Rules

APS must log:

- event type
- delivery id
- selected profile/action
- execution result

APS must not log:

- secrets values
- authorization headers

---

## 15) Documentation Requirements

- No single generated doc file may exceed 500 lines.
- Docs must be split by topic into separate files.
- `WEBHOOKS.md` must include:
  - webhook request examples
  - signature generation instructions
  - event mapping examples
  - sample action scripts

---

## 16) Prerequisites

Build-time:

- Go toolchain

Runtime (optional depending on usage):

- `git`
- `ssh`
- `ssh-keygen`
- `curl`
- `jq`

---

## 17) Testing Plan

### 17.1 Profiles

- `aps profile new agent-a --display-name "Agent A" --email a@x.com --github agent-a`
- `aps profile list`
- `aps profile show agent-a`
- `aps run agent-a -- env` includes required injected variables

### 17.2 Actions

- `aps action list agent-a`
- `aps action run agent-a <action-id> --payload-file payload.json`

### 17.3 Webhooks

Start server:

```bash
aps webhook serve --addr 127.0.0.1:8080 --dry-run \
  --event-map custom.build.requested=agent-a:sample-action
```

Send event:

```bash
curl -X POST http://127.0.0.1:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-APS-Event: custom.build.requested" \
  -d '{"hello":"world"}'
```

---

## 18) Deliverables

- `aps` Go binary implementing:
  - CLI commands
  - TUI mode
  - profile management
  - run and action execution
  - docs generation
  - webhook server
- Embedded templates and docs
- Documentation generated under `~/.agents/docs/`
- Sample action template under `~/.agents/templates/actions/`