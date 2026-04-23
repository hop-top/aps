# APS Cheatsheet — Agent

Quick reference for autonomous agents, scripts, and LLMs consuming the APS CLI or API.
Scannable in 30 seconds.

---

## Prerequisites

```bash
aps version                          # verify client
aps serve --addr 127.0.0.1:8080      # start REST API (optional; CLI works standalone)
curl http://127.0.0.1:8080/health    # verify server if using REST
export APS_PROFILE=<profile>         # default profile for CLI calls
```

Config: XDG (`~/.config/aps/`); data: `$APS_DATA_PATH`

---

## Agent Loop Contract

```
1. Orient     →  aps profile list / GET /profiles
2. Discover   →  aps collab agents --cap <cap> / aps directory discover
3. Coordinate →  aps collab use <ws> / aps collab send <agent> --action <act>
4. Execute    →  aps run <profile> -- <cmd> / aps action run <profile> <action>
5. Observe    →  aps session inspect <id> / aps session logs <id>
6. Report     →  aps collab ctx set <key> <result>
```

**DO:** resolve profile + workspace before sending tasks.
**DON'T:** send tasks before joining collab workspace.
**DO:** poll session state before assuming completion.
**DON'T:** use A2A and collab send interchangeably — different protocols.

---

## Profile Resolution

```bash
aps profile list --output json       # enumerate profiles
aps profile show <name>              # full profile detail (caps, workspace link, etc.)
aps profile status <name>            # bundle resolution; check before running

# Env-based targeting
export APS_PROFILE=<name>            # default profile for session
```

---

## Run Commands

```bash
aps run <profile> -- <cmd> [args]    # run cmd under profile env isolation
aps env <profile>                    # emit env vars (source into shell)
```

---

## Adapter Actions

Script adapters dispatch actions to backend scripts.
No LLM needed — pure shell execution.

```bash
# Send email as profile
aps adapter exec email send \
  --profile <id> \
  --input to=<addr> \
  --input subject="<subject>" \
  --input body="<body>"

# List inbox
aps adapter exec email list --profile <id>

# Read message
aps adapter exec email read --profile <id> \
  --input id=<envelope-id>

# Reply
aps adapter exec email reply --profile <id> \
  --input id=<envelope-id> \
  --input body="<reply>"

# Explicit From (no profile lookup)
aps adapter exec email send \
  --from ops@company.com \
  --input to=<addr> \
  --input subject="<subject>" \
  --input body="<body>"
```

Profile email resolved from `profile.yaml` `email` field.
Env vars passed to scripts: `APS_EMAIL_FROM`,
`APS_EMAIL_ACCOUNT`, `EMAIL_<INPUT>`.

---

## Sessions

```bash
aps session list                     # active sessions
aps session inspect <id>             # state, profile, start time
aps session logs <id>                # stdout/stderr capture (tmux)
aps session terminate <id>           # graceful stop
aps session delete <id>              # clean up record
```

States: `active` → `terminated`

---

## Actions

```bash
aps action list <profile>            # discoverable actions for profile
aps action show <profile> <action>   # schema: inputs, outputs, description
aps action run <profile> <action>    # execute; blocks until done
```

---

## Collaboration (Multi-Agent)

Primary coordination path for agent-to-agent work within a shared workspace.

```bash
aps collab use <workspace>           # set active workspace (persists per session)
aps collab list                      # enumerate workspaces
aps collab show <ws>                 # details: members, policy, conflicts
aps collab join <ws>                 # register agent presence
aps collab members                   # list members + roles (uses active ws)
aps collab agents --cap <capability> # discover agents by capability

# Send a task to another agent
aps collab send <recipient> \
  --action <act> \
  --set key=val \
  --timeout 5m                       # optional; default no timeout

# Monitor tasks
aps collab tasks                     # all workspace tasks
aps collab task <id>                 # single task detail + status

# Shared context (broadcast state to all members)
aps collab ctx list                  # all k/v pairs
aps collab ctx set <key> <value>     # write
aps collab ctx get <key>             # read
aps collab ctx history <key>         # mutation log (versioned)

# Conflicts
aps collab conflicts                 # list unresolved conflicts
aps collab resolve <id>              # resolve conflict
aps collab policy show               # active resolution policy
```

---

## A2A Protocol (Inter-Agent, HTTP)

Use when targeting agents by URL rather than workspace membership.

```bash
aps a2a show-card <profile>          # emit agent card (share with remote agents)
aps a2a fetch-card <url>             # fetch remote agent's card

# Task lifecycle
aps a2a send-task <profile> \
  --to <agent-url> \
  --msg "<message>"                  # create or continue task → returns task ID

aps a2a get-task <task-id>           # poll task state
aps a2a list-tasks <profile>         # all tasks for profile
aps a2a subscribe-task <task-id>     # push notification (webhook-based)
aps a2a cancel-task <task-id>        # cancel running task

# Server mode (expose profile as A2A endpoint)
aps a2a server <profile> \
  --addr :8081                       # serve profile over A2A HTTP
```

A2A task states: `submitted` → `working` → `completed` | `failed` | `canceled`

---

## ACP (Editor/Tool Protocol)

```bash
aps acp server <profile> \
  --addr :8082                       # start ACP server for editor integration
aps acp toggle <profile> on          # enable ACP for profile
```

---

## HTTP REST API

Start with `aps serve`; then use REST equivalents.

| CLI | REST |
|-----|------|
| `aps profile list` | `GET /profiles` |
| `aps profile show <n>` | `GET /profiles/{name}` |
| `aps session list` | `GET /sessions` |
| `aps session inspect <id>` | `GET /sessions/{id}` |
| `aps action run <p> <a>` | `POST /profiles/{p}/actions/{a}` |
| `aps collab list` | `GET /workspaces` |
| `aps collab ctx set` | `PUT /workspaces/{ws}/ctx/{key}` |
| `aps a2a send-task` | `POST /a2a/tasks` |
| `aps a2a get-task` | `GET /a2a/tasks/{id}` |

Auth (if `--auth-token` set): `Authorization: Bearer <token>`

---

## Capability & Bundle Discovery

```bash
aps capability list                  # all available capabilities
aps capability show <name>           # detail: type, path, config schema
aps bundle list                      # builtin + user bundles
aps bundle show <name>               # full YAML: capabilities + config
```

---

## Identity & Trust

```bash
aps identity show <profile>          # DID + public key
aps identity verify <did>            # resolve + verify DID
aps identity badge list <profile>    # verifiable credentials (capabilities attested)
aps identity badge verify <badge>    # verify credential authenticity
```

---

## Directory (AGNTCY)

```bash
aps directory show <profile>         # OASF record (machine-readable agent descriptor)
aps directory discover \
  --cap <capability>                 # find agents by capability (federated registry)
```

---

## Policy & Audit

```bash
aps policy show <workspace>          # effective access policy
aps audit <workspace>                # full access audit log
aps collab audit                     # collaboration event trail
```

---

## Error Handling

| Condition | Handling |
|-----------|----------|
| Profile not found | `aps profile list`; check `$APS_DATA_PATH` |
| Session stuck | `aps session inspect <id>`; terminate + clean up |
| A2A task failed | `aps a2a get-task <id>` → check error; `cancel-task` + retry |
| Collab task not delivered | `aps collab task <id>`; check recipient presence with `members` |
| Conflict blocks workspace | `aps collab conflicts`; resolve before sending more tasks |
| Capability missing | `aps capability list`; install with `aps capability install <src>` |
| APS server not reachable | Verify `aps serve` running; check `--addr`; use CLI fallback |
| Auth rejected | Confirm `--auth-token` matches; use `Authorization: Bearer <tok>` |

---

## Profile Data Model (Key Fields)

| Field | Notes |
|-------|-------|
| `name` | unique identifier |
| `workspace` | linked workspace (optional) |
| `capabilities` | list of enabled capabilities |
| `actions` | named executable actions |
| `a2a.enabled` | A2A protocol active |
| `acp.enabled` | ACP (editor) protocol active |
| `webhook.enabled` | webhook server active |
| `identity.did` | decentralized identifier (after `identity init`) |

---

## Session Data Model (Key Fields)

| Field | Notes |
|-------|-------|
| `id` | stable session ID |
| `profile` | owning profile name |
| `state` | `active`, `terminated` |
| `started_at` | ISO 8601 |
| `pid` | OS process ID (if applicable) |
