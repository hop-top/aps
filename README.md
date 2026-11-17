# aps

Agent Profile System: run commands and agent workflows under isolated, local-first profiles.

Prerelease: versions stay on the `0.x` alpha channel until 1.0, and a minor bump can carry breaking changes ([CHANGELOG.md](CHANGELOG.md)).

[![Release](https://img.shields.io/github/v/release/hop-top/aps?include_prereleases&sort=semver)](https://github.com/hop-top/aps/releases)
[![CI](https://github.com/hop-top/aps/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/hop-top/aps/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/hop-top/aps/branch/main/graph/badge.svg)](https://codecov.io/gh/hop-top/aps)
[![12-factor AI-CLI conformance](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/hop-top/aps/main/.12fc.json)](e2e/conformance/README.md)

## Table of contents

- [Demo](#demo)
- [Why](#why)
- [Quick start](#quick-start)
- [Install](#install)
- [Usage](#usage)
- [Commands](#commands)
- [Flags](#flags)
- [Configuration](#configuration)
- [Exit codes](#exit-codes)
- [Agent usage](#agent-usage)
- [Troubleshooting](#troubleshooting)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

## Demo

```console
$ aps profile create reviewer --display-name "Code Reviewer" --email reviewer@example.com
Profile 'reviewer' created successfully.
$ aps run reviewer -- git config --get user.name
[exec] git
Code Reviewer
[exit] git ok
```

## Why

Running several agents, or one agent under several identities, means a different set of
credentials, a different git identity, and a different blast radius per task. Exporting
variables by hand puts the wrong token in the wrong tool. aps keeps each identity as a
profile: a directory holding `profile.yaml`, `secrets.env`, a `gitconfig`, notes, and
actions. `aps run reviewer -- git status` starts any command inside that environment at the
isolation level the profile asks for. The rest of the CLI manages profiles and the surfaces
that feed work into them: messengers, webhooks, ticket queues, agent protocols, voice.

**Why use it**

- One static binary. The base install needs no daemon, container, or background service.
- Local-first. Profiles are YAML files on disk, and `--offline` disables every network call.
- Secrets, git identity, and profile variables reach the child process as environment
  variables; secrets and PII are redacted from logs and output unless you pass `--no-redact`.
- Three isolation levels: process, platform (a sandbox user account on macOS and Linux), and
  container (Docker), with a configurable fallback when the requested level is unavailable.
- Every command speaks `--format json`, `yaml`, or `csv`, honours `--dry-run` and
  `--confirm`, and exits with canonical codes; `aps toolspec` describes the whole surface for
  tool runners. Graded 12/12 against the 12-factor AI-CLI contract.

**When not to use it**

- Platform isolation needs passwordless sudo and an OpenSSH server (macOS 10.15 or later
  with the Xcode Command Line Tools; Linux kernel 3.10 or later with `useradd` and
  `setfacl`), and container isolation needs a running Docker daemon. Without them a profile
  falls back to process isolation, or fails when it sets `strict: true`. On Windows only
  process isolation is available; the platform sandbox there is documented as planned.
- Data lives under `~/.local/share/aps`, `~/.config/aps`, `~/.local/state/aps`, and
  `~/.cache/aps` on every OS, including macOS and Windows. There is no OS-native layout.
- `aps service add` records a message service on the aps side only. It does not create the
  Slack app, Discord gateway client, Telegram poller, Twilio webhook, or public tunnel that
  delivers messages to it.

## Quick start

```sh
brew install hop-top/tap/aps
aps profile create reviewer --display-name "Code Reviewer" --email reviewer@example.com
aps run reviewer -- git config --get user.name
```

The last command prints `Code Reviewer` between the `[exec] git` and `[exit] git ok`
progress lines. `aps reviewer -- echo hi` is shorthand for `aps run`, `aps reviewer` opens a
shell as that profile, and `aps` with no arguments opens the interactive TUI.

## Install

Prebuilt binaries need nothing else. Building from source or `go install` needs Go 1.27.4
or later. Platform and container isolation have their own requirements, listed in
[docs/user/isolation.md](docs/user/isolation.md).

| Path | Command |
|---|---|
| Homebrew (macOS, Linux) | `brew install hop-top/tap/aps` |
| Scoop (Windows) | `scoop bucket add hop-top https://github.com/hop-top/scoop-bucket`, then `scoop install aps` |
| Go | `go install hop.top/aps/cmd/aps@latest`, with `$(go env GOPATH)/bin` on `PATH` |
| Prebuilt binary | Download from [Releases](https://github.com/hop-top/aps/releases) and put `aps` on `PATH` |
| Source | `git clone https://github.com/hop-top/aps.git && cd aps && make build` builds `bin/aps`; `make build-vault` adds the opt-in OpenBao secrets backend |

Supported platforms: Linux and macOS on amd64 and arm64, Windows on amd64. Release archives
are named after the version, OS, and architecture (`.tar.gz`, or `.zip` on Windows); the
`aps-vault` archives hold the same binary built with the OpenBao backend.

Verify a download: every release ships `checksums.txt` with SHA-256 sums and an SBOM per
archive. Compare `shasum -a 256` of the archive with its line in `checksums.txt` before
extracting.

Shell completion for bash, zsh, fish, and PowerShell comes from `aps completion`:

```sh
aps completion zsh > "${fpath[1]}/_aps"    # zsh; bash: source <(aps completion bash)
```

`eval "$(aps alias shell)"` defines one shell alias per profile, so `reviewer echo hi` runs
`aps reviewer echo hi`. `aps upgrade` checks for a newer release and installs it after a
prompt; `--auto` skips the prompt.

## Usage

List profiles. Every list and show command takes `--format json`, `yaml`, or `csv`:

```console
$ aps profile list
ID        DISPLAY NAME   ROLES  CAPABILITIES  WORKSPACE  EMAIL                 SECRETS  DID    COLOR  AVATAR
reviewer  Code Reviewer                                  reviewer@example.com  true     false
```

Run a command with the profile's environment plus one-off variables:

```console
$ aps run reviewer --env GREETING=hello -- sh -c 'echo "$GREETING from $APS_PROFILE_ID"'
[exec] sh
hello from reviewer
[exit] sh ok
```

Keys in the profile's `secrets.env` reach the command as environment variables, so an
external CLI such as `claude`, `codex`, `gemini`, or `opencode` picks up the profile's API
keys without aps printing their values:

```console
$ aps run reviewer -- sh -c 'test -n "$GITHUB_TOKEN" && echo "GITHUB_TOKEN is set"'
[exec] sh
GITHUB_TOKEN is set
[exit] sh ok
```

The profile-first form drops the progress lines; with no command it opens a shell as the
profile:

```console
$ aps reviewer -- echo hi
hi
```

Inspect a profile. `aps profile export reviewer --manifest-format agentco` renders the same
record as an `AGENTS.md` role manifest (identity only, never secrets), and
`aps profile import AGENTS.md` creates a profile from one:

```console
$ aps profile show reviewer
id: reviewer
display_name: Code Reviewer
email: reviewer@example.com
preferences:
    shell: /bin/zsh
git:
    enabled: true
isolation:
    level: process
    strict: false
    fallback: false


Modules:
- Secrets: present
```

## Commands

`aps --help` groups commands by intent (interact, organize, pipelines, security, instance);
`--help-all` adds the management group, and `-h` after any command prints its flags. The
generated reference for every command and flag is
[docs/cli/commands.md](docs/cli/commands.md).

| Command | Purpose |
|---|---|
| `run` | Run a command in a profile context |
| `profile` | Manage agent profiles |
| `session` | Manage sessions |
| `chat` | Chat with a profile-backed assistant |
| `serve` | Start protocol server |
| `listen` | Subscribe to bus topics for a profile and print events as JSONL |
| `voice` | Manage voice sessions and the voice backend service |
| `capability` | Manage capabilities (tools, configs, dotfiles) |
| `bundle` | Manage capability bundles |
| `workspace` | Manage multi-agent workspaces |
| `squad` | Manage agent squads (topology, membership, scope) |
| `org` | Inspect the reporting hierarchy across profiles |
| `contact` | Manage contacts via adapter |
| `action` | Manage and execute profile actions |
| `adapter` | Manage adapters (messengers, protocols, mobile, desktop) |
| `service` | Manage profile-facing services |
| `webhook` | Manage Webhook server |
| `skill` | Manage Agent Skills |
| `a2a` | Manage A2A (Agent-to-Agent) protocol operations |
| `acp` | Manage ACP (Agent Client Protocol) server |
| `directory` | Manage AGNTCY Directory registration and discovery |
| `identity` | Manage DID-based agent identity |
| `policy` | Manage workspace access policies |
| `observability` | Manage OpenTelemetry observability |
| `status` | Show aps configuration and runtime status |
| `config` | Inspect aps configuration |
| `docs` | Generate documentation |
| `toolspec` | Print the aps tool specification (commands, flags, errors, workflows) |
| `alias` | Manage command aliases |
| `completion` | Generate completion script |
| `env` | Output environment variables for configured capabilities |
| `migrate` | Migrate legacy configurations to new formats |
| `upgrade` | Check for and install updates |
| `version` | Print version information |

Sessions: `aps session list` (filters `--status`, `--tier`, `--type`; voice sessions are
`--type voice`), `inspect`, `logs`, `attach`, `detach`, `terminate`, and `delete`. Attach
uses tmux; for a platform-sandboxed session it tunnels over SSH into the sandbox user, for a
container session into the container.

Capabilities: `aps capability install`, `list`, `show`, `link`, `adopt` (move a file or
directory under aps and symlink it back), `watch` (symlink an external file in), `enable`
and `disable` per profile, and `delete`. `aps capability link copilot` or `link windsurf`
without `--target` resolves to that tool's conventional path. `eval "$(aps env)"` exports a
path variable per installed capability.

### Adapters

Profiles connect to external systems through adapters, grouped by kind:

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/adaptermd", "kinds-list"],
    text=True))
]]] -->
- **Actuator**: Robotics, hardware
- **Desktop**: Desktop applications
- **Messenger**: Telegram, Slack, etc.
- **Mobile**: Mobile devices (via QR linking)
- **Protocol**: A2A, ACP, webhooks, WebSockets
- **Scheduler**: Calendars, schedulers, reminders
- **Sense**: Camera, microphone
<!-- [[[end]]] -->

Message services bind one chat platform (Telegram, Slack, Microsoft Teams, Discord, SMS,
WhatsApp, or email) to a profile action: `aps service add` records the service, `aps serve`
or `aps service start` exposes its webhook, and aps validates each request, normalises the
event, routes it by sender, and runs the action. Every routed message and delivered reply is
stored as a conversation turn, and each action run receives the newest prior turns of the
same session; `aps service conversation list` and `show` read them back. Platform
comparison: [docs/MESSENGERS_OVERVIEW.md](docs/MESSENGERS_OVERVIEW.md); setup and testing:
[docs/user/messengers.md](docs/user/messengers.md); how conversation and session identity
are derived: [docs/dev/message-conversation-policy.md](docs/dev/message-conversation-policy.md);
adapter internals and manifests: [docs/dev/adapters.md](docs/dev/adapters.md).

## Flags

Global flags apply to every command; `aps --help-all` lists all of them. The ones to reach
for first:

| Flag | Default | Effect |
|---|---|---|
| `--format` | `table` | Output format: `csv`, `human`, `json`, `table`, `text`, `yaml` |
| `-p`, `--profile` | active profile | Profile id; `APS_PROFILE` sets the default |
| `--workspace` | active workspace | Workspace id |
| `--offline` | off | Disable all network calls |
| `--dry-run` | off | Preview side effects without applying them, on commands that support it |
| `--confirm` | `prompt` on a TTY, `no` otherwise | Confirmation policy for destructive commands: `auto`, `yes`, `no`, `prompt` |
| `--max-ops` | `0` (unlimited) | Cap mutating operations per invocation |
| `--quiet` | off | Suppress non-essential output |
| `--no-color` | off | Disable ANSI colour |
| `-V`, `--verbose` | off | Increase log verbosity: `-V` debug, `-VV` trace |

Per-command flags come from `-h`. `run` takes `--env KEY=VALUE` (repeatable, later wins),
`--env-file` (dotenv file, repeatable), and `--idempotency-key`. Every state-changing
subcommand takes `-n`, `--note`, an audit note recorded with the bus event and visible to
the policy engine; some policies reject destructive operations without one
([docs/cli/reference.md](docs/cli/reference.md)).

## Configuration

The user config file is `~/.config/aps/config.yaml` (`$XDG_CONFIG_HOME/aps/config.yaml`).
A project can add `.aps.yaml`, `.aps/config.yaml`, or `.hop/aps/config.yaml`; aps reads
the current directory's markers, walks up to the project root, then the user file, then
`/etc/aps/config.yaml`. `aps config paths` prints that chain, and `-c key=value` overrides
one value for one invocation.

```yaml
prefix: APS
isolation:
  default_level: process
  fallback_enabled: true
capability_sources:
  - ~/shared/capabilities
```

| Field | Default | Effect |
|---|---|---|
| `prefix` | `APS` | Prefix for the variables injected into sessions |
| `isolation.default_level` | `process` | Default isolation level for profiles that do not set one |
| `isolation.fallback_enabled` | `true` | Allow degraded-mode operation if the preferred isolation is unavailable |
| `capability_sources` | `[]` | Additional directories to search for capability definitions |

`secrets.backend` selects where secrets live: `file` (the profile's `secrets.env`, mode
0600, the default), `env`, `keyring`, `onepassword`, `openbao` (needs the `aps-vault`
build), `infisical`, or `ghsecrets`; the keys each backend needs are listed under Secret
Store Backends in [docs/user/messengers.md](docs/user/messengers.md). Per-profile isolation
lives in `profile.yaml` (`isolation.level`, `strict`, `fallback`, and the `platform` and
`container` blocks): [docs/user/isolation.md](docs/user/isolation.md).

Environment variables aps reads:

| Variable | Effect |
|---|---|
| `APS_DATA_PATH` | Data directory; overrides `$XDG_DATA_HOME/aps` and `~/.local/share/aps` |
| `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME` | Roots for config, state, and cache, each with an `aps` subdirectory |
| `APS_PROFILE` | Default profile when `--profile` is not passed |
| `APS_BUS_TOKEN` (or `BUS_TOKEN`), `APS_BUS_ADDR` | Event-bus auth token and address; without a token the bus is disabled and a warning is printed |
| `APS_NO_BUS_WARN` | Suppress that warning |
| `APS_ADAPTER_DISABLE_PREFIX_VALIDATION` | Set to `1` to accept adapter manifests whose `env_prefix` is not a valid variable name while migrating them |

Variables the child process receives, with the prefix from `prefix`: `APS_PROFILE_ID`,
`APS_PROFILE_DIR`, `APS_PROFILE_YAML`, `APS_PROFILE_SECRETS`, `APS_PROFILE_DOCS_DIR`, every
key in `secrets.env`, `GIT_CONFIG_GLOBAL` pointing at the profile's `gitconfig` when git is
enabled, `GIT_SSH_COMMAND` when an SSH key is configured, a path variable per linked
capability, then the values from `--env-file` and `--env`.

### Directory structure

| Purpose | Path |
|---|---|
| Data: profiles, sessions, adapters, capabilities, generated docs | `$APS_DATA_PATH`, else `$XDG_DATA_HOME/aps`, else `~/.local/share/aps` |
| Config | `$XDG_CONFIG_HOME/aps`, else `~/.config/aps` |
| State | `$XDG_STATE_HOME/aps`, else `~/.local/state/aps` |
| Cache | `$XDG_CACHE_HOME/aps`, else `~/.cache/aps` |

Inside the data directory:

```
profiles/reviewer/
  profile.yaml         identity, preferences, git and isolation settings
  secrets.env          KEY=VALUE secrets, mode 0600
  gitconfig            used through GIT_CONFIG_GLOBAL
  notes.md             body of the exported AGENTS.md manifest
  actions/             profile action scripts
sessions/registry.db   session registry
capabilities/          installed capabilities
keys/admin_priv        admin key for attaching to sandboxed sessions
docs/                  output of aps docs
```

Older releases kept profiles under `~/.agents/profiles` and sessions in a `registry.json`;
see [Troubleshooting](#troubleshooting).

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Generic error, including unknown flags, missing arguments, and a missing `--` on `aps run` |
| 3 | Not found: the profile, session, or capability does not exist |
| 4 | Conflict: the record already exists, or a policy denied the change |
| 5 | Unauthorized: a credential or auth check failed |
| 64 | Rate limited: the `--max-ops` budget was exceeded |
| child's code | `aps run` and `aps action run` propagate the command's exit code |

Error output carries the same class as a prefix (`NOT_FOUND`, `CONFLICT`, `GENERIC`), and
`--format json` renders it as a structured envelope.

## Agent usage

Every command accepts `--format json` (also `yaml` and `csv`) and never prompts off a TTY:
`--confirm` defaults to `no` there, and `profile create` asks for missing fields only when
stdin is a terminal.

```console
$ aps profile list --format json
[
  {
    "id": "reviewer",
    "display_name": "Code Reviewer",
    "roles": "",
    "capabilities": "",
    "workspace": "",
    "email": "reviewer@example.com",
    "has_secrets": true,
    "has_identity": false,
    "color": "",
    "avatar": ""
  }
]
```

- `--dry-run` previews commands that support it; the ones that cannot (`run`, `chat`,
  `docs`) say why in `-h`.
- `--max-ops` caps mutating operations per invocation, and `--idempotency-key` on `run`,
  `action run`, and `profile import` replays the recorded result for a repeated key.
- `--progress-format json`, the default alongside `--format json`, emits the exec and exit
  progress envelope as JSONL.
- `aps toolspec --format json` prints every command, flag, error pattern, and workflow;
  `aps status --format json` reports the active profile, workspace, version, and bus state.
- `aps listen --profile reviewer` streams bus events as JSONL (needs `APS_BUS_TOKEN`).

Exit codes are listed above. A cheatsheet for scripts and LLMs is
[docs/cheatsheet-agent.md](docs/cheatsheet-agent.md); [AGENTS.md](AGENTS.md) covers working
on the aps codebase itself.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `warn: bus auth: BUS_TOKEN or APS_BUS_TOKEN not set; bus disabled` on every command | No event-bus token in the environment | Set `APS_BUS_TOKEN` to enable the bus (`aps listen`, event publishing), or `APS_NO_BUS_WARN=1` to silence the warning |
| `aps: command not found` after `go install` | `$(go env GOPATH)/bin` is not on `PATH` | Add it in your shell profile |
| `go install` fails with module or toolchain errors | Go older than 1.27.4 | Upgrade Go |
| Profiles from an older install are missing | Data moved from `~/.agents/profiles` to `~/.local/share/aps/profiles` | Copy the profile directories over, or `export APS_DATA_PATH="$HOME/.agents"` |
| `GENERIC: missing '--' separator` | `aps run` needs `--` between the profile and the command | `aps run reviewer -- git status` |
| `aps voice start` reports the backend unreachable | The voice backend daemon is not running | Run `aps voice service start` first |
| `aps service add` prints `config_valid: false` and exits 1 | A required option for that service type is missing, for example `--provider` and `--from` for `sms` | Add the listed options; nothing was written |

More: the troubleshooting section of
[docs/user/getting-started/install.md](docs/user/getting-started/install.md),
[docs/MESSENGER_SETUP_QUICK_REF.md](docs/MESSENGER_SETUP_QUICK_REF.md), and the platform
pages linked below.

## Documentation

- [SKILL.md](SKILL.md) routes by intent ("I want to..."); [docs/INDEX.md](docs/INDEX.md)
  lists everything; [DEVELOPING.md](DEVELOPING.md) is for changing aps.
- Getting started: [install](docs/user/getting-started/install.md),
  [tutorial](docs/user/getting-started/tutorial.md), [glossary](docs/glossary.md).
- CLI: [reference](docs/cli/reference.md) (global flags, exit codes, `--note` inventory,
  export formats), [commands](docs/cli/commands.md) (generated),
  [human cheatsheet](docs/cheatsheet-human.md), [agent cheatsheet](docs/cheatsheet-agent.md),
  [redaction](docs/cli/redaction.md).
- Isolation: [levels and requirements](docs/user/isolation.md),
  [macOS](docs/dev/platforms/macos/overview.md), [Linux](docs/dev/platforms/linux/overview.md),
  [container](docs/dev/platforms/container/overview.md),
  [migrating a profile from process to platform or container](docs/dev/implementation/guides/migration-guide.md),
  [benchmarks](docs/dev/testing/performance-benchmarks.md).
- Messaging and tickets: [overview](docs/MESSENGERS_OVERVIEW.md),
  [quick reference](docs/MESSENGER_SETUP_QUICK_REF.md),
  [message services](docs/user/messengers.md), [conversations](docs/user/conversations.md),
  [tickets](docs/user/tickets.md), [Telegram](docs/TELEGRAM_SETUP.md),
  [Discord](docs/DISCORD_SETUP.md).
- Protocols and remote access: [A2A quickstart](docs/user/a2a-quickstart.md),
  [A2A examples](docs/user/a2a-examples.md), [ACP quickstart](docs/user/acp-quickstart.md),
  [remote access](docs/user/remote-access.md).
- Extending: [adapters](docs/dev/adapters.md), [skills](docs/user/skills/README.md) with an
  [example skill](examples/skills/hello-world/SKILL.md),
  [capability bundles](docs/dev/bundles.md), [squads](docs/dev/squads.md),
  [voice](docs/dev/voice.md), [policies](docs/policies.md),
  [configuration](docs/dev/configuration.md).
- Architecture: [architecture.md](docs/architecture.md), [ADRs](docs/adr/),
  [developer notebook](docs/dev/readme.md),
  [agent guide to the codebase](docs/agent/README.md),
  [security audit](docs/dev/security/security-audit.md),
  [release notes](docs/dev/operations/releases/release-notes.md).
- Generated locally: `aps docs` writes `README.md`, `CLI.md`, `PROFILES.md`,
  `ISOLATION.md`, `SESSIONS.md`, `EXAMPLES.md`, `WEBHOOKS.md`, `SECURITY.md`, and
  `SPEC.md` under the data directory's `docs/`. `EXAMPLES.md` walks through GitHub issue
  triage, an OpenAI chat assistant, a data-processing pipeline, multi-environment
  deployment, a Reddit bot, git workflow automation, webhook integration, a database
  migration, a custom shell environment, and running tests under different environments.

Changelog: [CHANGELOG.md](CHANGELOG.md), maintained by release-please.

## Contributing

Pull requests are welcome; read [CONTRIBUTING.md](CONTRIBUTING.md) first (fork, feature
branch, Conventional Commits, one concern per PR). Questions and bug reports go to
[GitHub issues](https://github.com/hop-top/aps/issues); templates exist for bug reports and
for platform-adapter PRs.

Dev setup: `mise install` (pinned Go, goreleaser, golangci-lint, act, lychee), then
`make build` for `bin/aps`; `make help` lists every target. Before a PR run `make test` and
`make lint`, which is what CI runs on Ubuntu, macOS, and Windows: unit, internal-package, and
end-to-end Go tests; golangci-lint; the generated-docs staleness check; and the lychee link
check. Tables between cog markers, including the adapter list above, are generated from
code: change the source, run `make docs-gen`, and commit both.

Releases are automated by release-please on `main`: merging its release PR tags `aps/v` plus
the version, publishes the GitHub Release and the CHANGELOG entry, and GoReleaser uploads
the archives and updates the Homebrew tap and Scoop bucket. Versions stay on the prerelease
channel until 1.0. Details: [RELEASING.md](RELEASING.md).

### Test

```sh
make test                   # Go tests plus the GitHub workflows locally via act
make test-unit              # tests/unit
make test-e2e               # tests/e2e
make test-stories           # tests linked from docs/stories/
make docker-test-e2e-user   # user-journey tests in a Docker Linux environment
```

Test layout, the direct `go test` forms, the Docker targets, and story-linked tests:
[docs/dev/testing/running-tests.md](docs/dev/testing/running-tests.md).

## Security

Report vulnerabilities privately by email to the address in [SECURITY.md](SECURITY.md); do
not open a public issue. Reports are acknowledged within 48 hours, assessed within a week,
and fixed on a best-effort basis within 30 days; only the latest release receives security
fixes. How secrets and PII are kept out of logs and output:
[docs/cli/redaction.md](docs/cli/redaction.md); audit:
[docs/dev/security/security-audit.md](docs/dev/security/security-audit.md).

## License

[MIT](LICENSE). Copyright (c) 2026 Idea Crafters LLC and jadb. Release archives include the
license and an SBOM.
