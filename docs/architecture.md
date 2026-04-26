# Architecture — aps

Last updated: 2026-04-26
Author: $USER

## Purpose

`hop.top/aps` is the Agent Profile System — a local-first
identity, capability, and credential envelope that lets
commands run "as" a profile. Each profile binds: an identity
(email, persona, capabilities), a workspace scope, an
isolation level, and a set of adapters that integrate with
external systems (email, messengers, mobile devices, A2A
peers).

## Context

aps is the identity substrate for the hop-top ecosystem. When
hop, tlc, rux, or any other tool needs to know "who is acting
and with what privileges", they consult aps. Profiles are
local YAML files; the system is designed to work offline and
without a central server.

Cross-references:

- [kit](https://github.com/hop-top/kit/blob/main/docs/architecture.md) — CLI substrate; bus, charm v2, lipgloss
- [cxr](https://github.com/hop-top/cxr/blob/main/docs/architecture.md) — execution framework (subprocess / script / builtin handler routing)
- [upgrade](https://github.com/hop-top/upgrade/blob/main/docs/architecture.md) — version checks
- [wsm](https://github.com/hop-top/wsm/blob/main/docs/architecture.md) — workspace state; aps profiles link to workspaces by name
- [tlc](https://github.com/hop-top/tlc/blob/main/docs/architecture.md) — task ledger; assignee resolution uses aps profile ID
- [a2aproject/a2a-go](https://github.com/a2aproject/a2a-go) — Agent-to-Agent protocol (external)

## Containers

| Container | Path | Role |
|---|---|---|
| `aps` CLI | `cmd/aps/` | Cobra-based CLI |
| `aps serve` HTTP API | `internal/server/` | Optional REST server for profiles, sessions, A2A tasks |
| Adapters | `~/.agents/adapters/` + `~/.agents/profiles/<id>/adapters/` | Per-profile or global adapter binaries |
| Profile registry | `~/.agents/profiles/<id>/profile.yaml` | YAML files; one per profile |
| Session registry | `~/.agents/sessions/registry.json` | Active session tracking |

## Components

### Profile management (`internal/core/profile.go`, 687L)

YAML profile load / save. Enforces the `WorkspaceLink`,
`Persona`, `Capabilities`, and `Scope` isolation contracts.
Profile data is per-machine (`~/.agents/profiles/<id>/`).

### Adapter system (`internal/core/adapter/`)

Multi-strategy dispatch. An adapter has a `type`
(messenger / protocol / mobile / desktop / sense / actuator)
and a `strategy` (`subprocess` / `script` / `builtin`).
Adapters are auto-discovered from `~/.agents/adapters/` and
per-profile dirs. Each declares itself via `manifest.yaml`.

The messenger adapter (`messenger_adapter.go`) ships with a
migration path from legacy messenger configs. Mobile pairing
(`mobile/`) uses a token + QR-code flow with optional
approval, persisted to a per-profile device registry.

### Workspace association (`profile.go:117-121`)

`WorkspaceLink {name, scope, ScopeRules}` ties a profile to a
workspace. The link is lightweight — aps does not enforce
workspace existence; it only stores the reference. wsm owns
workspace state.

### Session engine (`internal/core/session/`)

Sessions move through `active → terminated`. The registry at
`~/.agents/sessions/registry.json` tracks `{session_id:
{profile_id, status, ...}}`. Loose JSON parsing tolerates
corruption — deletes still work even if the file is partially
broken.

### Adapter exec (`internal/cli/adapter/exec.go` + `internal/core/adapter/script_exec.go`)

`aps adapter exec <name> <action>` is the primary integration
point. Scripts receive a curated environment:

- `APS_EMAIL_FROM` resolved from `profile.email` (or `--from` override)
- `APS_EMAIL_ACCOUNT`
- `EMAIL_<INPUT>*` for each `--input k=v`

Default email backend is `himalaya`; scripts handle IMAP / SMTP details.

### Event bus (`internal/events/events.go` + `internal/cli/bus.go`)

Topics:

- `aps.profile.created`, `.updated`, `.deleted`
- `aps.adapter.linked`, `.unlinked`

Wired through kit/bus → dpkms cross-process hub via env vars
(`APS_BUS_ADDR`, `APS_BUS_TOKEN`). If the token is missing,
bus is disabled with a stderr warning; adapter commands still
work (fire-and-forget).

### Isolation layer (`internal/core/isolation/`)

Three levels:

- `process` (default) — environment injection only
- `platform` — Docker
- `container` — Docker + extra constraints

Set per-profile or globally. tmux capture for session logs.

### Capability system (`internal/core/capability/`)

Capability discovery + enforcement. Capabilities are typed
strings declared on profile (`capabilities: []`). Adapters
gate actions on required capabilities.

### Bundle resolution (`internal/core/bundle/`)

Capability + action discovery; binary execution logging.

### Collaboration (`internal/core/collaboration/`)

Multi-agent workspace primitives: conflict resolution, task
dispatch, shared context. Powers `aps collab` subcommand.

### Multi-device (`internal/core/multidevice/`)

Offline queue, sync manager, presence tracking. Foundation
for mobile + desktop device pairing.

## Public surfaces

### CLI

```
aps profile [new\|show\|list\|delete\|update]
aps run <profile> -- <cmd>
aps env <profile>
aps adapter exec <name> <action> [--profile <id>] [--input k=v] [--from email]
aps adapter [create\|start\|stop\|link\|unlink\|list\|status\|logs]
aps session [list\|inspect\|logs\|terminate\|delete]
aps action [list\|show\|run]
aps collab [use\|list\|show\|join\|members\|agents\|send\|tasks\|task\|ctx\|conflicts\|resolve\|policy]
aps a2a [show-card\|fetch-card\|send-task\|get-task\|list-tasks\|subscribe-task\|cancel-task\|server]
aps acp [server\|toggle]
aps capability [list\|show\|install]
aps bundle [list\|show]
aps serve --addr <host:port>
```

### Profile schema

YAML at `~/.agents/profiles/<id>/profile.yaml`:

```yaml
id: noor
display_name: "Noor"
email: "jad+noor@ideacrafters.com"
persona: { tone, style, risk }
capabilities: [...]
accounts: { service: { username } }
preferences: { language, timezone, shell }
workspace: { name, scope, scope_rules }
isolation: { level, strict, fallback }
a2a: { protocol_binding, listen_addr, public_endpoint }
acp: { enabled, transport, listen_addr, port }
mobile: { enabled, port, max_devices, allowed_capabilities }
identity: { did, ... }
squads: [...]
scope: { file_patterns, operations, tools, secrets, networks }
```

File permissions enforced at `0600` (per `SECURITY.md`).

### Adapter manifest

`<adapter-dir>/manifest.yaml`:

```yaml
name: email
type: messenger
strategy: subprocess
backend: himalaya
config:
  actions:
    - name: send
      script: ./scripts/send.sh
    - name: read
      script: ./scripts/read.sh
    - name: list
      script: ./scripts/list.sh
```

### Adapter exec convention

```bash
aps adapter exec email send --profile noor \
  --input to=user@example.com \
  --input subject="Hello"
```

Env passed to script: `APS_EMAIL_FROM`, `EMAIL_TO`,
`EMAIL_SUBJECT`, ... (each `--input k=v` becomes `EMAIL_K=v`).

### REST API (`aps serve`)

| Endpoint | Method | Purpose |
|---|---|---|
| `/profiles` | GET | List |
| `/profiles/{name}` | GET | Detail |
| `/sessions` | GET | List active sessions |
| `/sessions/{id}` | GET | Session detail |
| `/profiles/{p}/actions/{a}` | POST | Run action |
| `/workspaces` | GET | Workspace list |
| `/workspaces/{ws}/ctx/{key}` | PUT | Context update |
| `/a2a/tasks` | POST | Submit A2A task |
| `/a2a/tasks/{id}` | GET | Task status |

Auth: `Authorization: Bearer <token>` if `--auth-token` set.

## Integrations

| Integration | Where | Notes |
|---|---|---|
| `hop.top/kit` | `go.mod` | Bus (kafka-like), charm v2, event publishing |
| `hop.top/cxr` | `internal/core/adapter/manager.go` | Execution routing (subprocess / script / builtin) |
| `hop.top/upgrade` | `go.mod` | Version checker; release.md integration |
| `a2aproject/a2a-go` v0.3.4 | `go.mod` | Agent-to-agent protocol (HTTP + JSON) |
| Workspace | profile YAML `workspace` field | Decoupled — no direct wsm import |
| tlc | implicit | Profile shows caps / workspace / identity; no tlc dep |
| ctxt | per-profile `scope` field | Manages file patterns, tools, secrets, networks |
| Bus (dpkms hub) | kit/bus + env-based auth | Optional; missing token = disabled |

## Build / test / release

### Makefile

| Target | Runs |
|---|---|
| `make all` | build + test + lint |
| `make build` | `go build -ldflags "-X .../version.Version=..."` |
| `make test` | test-go + test-workflows |
| `make test-go` | `go test -v ./...` |
| `make test-unit` | `tests/unit/...` |
| `make test-e2e` | `tests/e2e/...` |
| `make test-stories` | `bash scripts/test-stories.sh` (doc-driven tests) |
| `make test-workflows` | `act push` (run GHA locally) |
| `make lint` | golangci-lint |
| `make lint-docs` | `bash scripts/check-links.sh` |
| `make release` | goreleaser |

### CI (`.github/workflows/`)

`build.yml` (govulncheck + Go 1.26), `ci.yml` (multi-OS test matrix), `coverage.yml` (codecov), `lint.yml` (golangci-lint), `platform-adapter-tests.yml`, `release.yml` (goreleaser on `v*` tags), `security.yml` (gosec weekly), `docker-user-journey.yml` (E2E in Docker).

### Code stats

450 Go files; ~91.5K LOC including tests.

## Architecture decisions

Embedded specs (`internal/core/assets/docs/`):

- [SPEC.md](../internal/core/assets/docs/SPEC.md) — goals, non-goals, library choices (cobra, charm, yaml, dotenv)
- [PROFILES.md](../internal/core/assets/docs/PROFILES.md) — profile creation, YAML schema, capabilities, accounts
- [CLI.md](../internal/core/assets/docs/CLI.md) — full command reference
- [SECURITY.md](../internal/core/assets/docs/SECURITY.md) — secrets management (`0600`), file permissions, best practices
- [ISOLATION.md](../internal/core/assets/docs/ISOLATION.md) — process / platform / container levels
- [SESSIONS.md](../internal/core/assets/docs/SESSIONS.md) — session lifecycle, tmux log capture
- [WEBHOOKS.md](../internal/core/assets/docs/WEBHOOKS.md) — event-driven action dispatch

A2A protocol ADRs at `docs/specs/005-a2a-protocol/adrs/` (9
ADRs on serialisation, transport, ordering, compression).

Plans at `docs/plans/`: integration gaps (agntcy), kit
adoption, CloudEvents bus design, server docs matrix.

## Evolution

### Recent significant changes

- Bus env-based auth (`a63aeea`) — replaces hardcoded dev token
- Connect to dpkms cross-process bus hub (`a05284a`)
- Profile lifecycle events from CLI commands (`1738225`)
- Bus-backed event types and publisher (`a445ad6`)
- Contact adapter + cardamum backend + CLI (`48de285`)
- Email adapter scaffold + exec command (`6fd1125`, `a7d954b`)
- Adopted hop.top/kit + full v2 charm migration (`64ebca0`)
- Lipgloss tables, huh prompts, error consistency (`91f3e88`)
- Go 1.26.1 + CI fixes (`b522729`)
- Adopted hop.top/cxr (`3b83ed1`)
- Integrated hop.top/upgrade (`f2f3e20`)

### Trend

Bus + adapter exec hardening; email / contact adapters; kit /
cxr adoption complete.

## Open questions

1. **Adapter discovery order.** Two scan paths (`~/.agents/adapters/` + per-profile). Tie-break rule when same name registered in both?
2. **Bus fallback.** Missing `APS_BUS_TOKEN` disables bus silently. Should it be louder, or optional-by-design?
3. **Session registry corruption.** Loose JSON parsing tolerates damage. Periodic compaction strategy?
4. **Workspace coupling.** Lightweight (name + scope only). Should aps verify workspace existence at link-time, or stay decoupled?
5. **Mobile device pairing.** QR / token → optional approval → registry. Revocation flow per-device or bulk?
6. **kit / cxr boundary.** cxr handles handler routing (subprocess / script / builtin); kit provides bus + charm v2. No shared executor protocol — is this intentional?
7. **`aps adapter exec` env safety.** `buildScriptEnv()` merges OS environ + profile + inputs. No special isolation for env (process-scoped only). Adequate for secrets in env vars?
