---
title: CLI Reference
description: Complete command reference for the aps CLI.
---

## Global flags and commands

The commonly used global flags, then every top-level command with its subcommands. This
listing is generated from the same command tree the binary dispatches, so it tracks the code. The
[full generated reference](https://github.com/hop-top/aps/blob/next/docs/cli/commands.md)
lists every subcommand and flag.

`aps --help-all` lists every global flag, including ones not shown below such as `--dry-run`,
`--confirm` and `--output`, and `aps <command> --help` is the full reference for one command.

`--profile` has no default. The `APS_PROFILE` environment variable stands in for it in two
places only: `aps workspace` commands (`create`, `join`, `leave`, `remove`, `send`,
`ctx set|get|list|delete`) and the policy principal id (`--profile`, then `APS_PROFILE`,
then `$USER`). `status`, `run`, `profile show` and the other commands do not read it.

Destructive commands (`profile delete`, `session delete`, `session terminate`,
`squad delete`, `squad members remove`, `workspace remove`, `workspace ctx delete`,
`capability delete`, `bundle delete`, `adapter detach`, `adapter revoke`,
`directory delete`) need `--confirm-token <token>`; `--yes` and `--force` do not replace it.
Run the command once without it: nothing changes, it exits with code 5, and the error
prints the token. Run it again with the token.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/climd", "--top"],
    text=True))
]]] -->
```
      --format string                  Output format (csv, human, json, table, text, yaml) (default "table")
  -h, --help                           help for aps
      --help-all[=true|false]          Show all commands and global flags
      --help-instance[=true|false]     Show only instance commands
      --help-interact[=true|false]     Show only interact commands
      --help-management[=true|false]   Show only management commands
      --help-organize[=true|false]     Show only organize commands
      --help-pipelines[=true|false]    Show only pipelines commands
      --help-security[=true|false]     Show only security commands
      --instance string                backend instance to target (defaults to config)
      --no-color[=true|false]          Disable ANSI color
      --no-hints[=true|false]          Suppress next-step hints after command output
      --no-redact[=true|false]         disable redaction of secrets/PII in logs and output (DEBUG ONLY)
      --offline[=true|false]           Disable network access
  -p, --profile string                 profile id (no default; workspace commands and the policy principal id fall back to $APS_PROFILE)
      --quiet[=true|false]             Suppress non-essential output
  -V, --verbose count                  Increase log verbosity (-V=debug, -VV=trace)
      --workspace string               workspace id (defaults to active workspace)
```

| Command | Description | Subcommands |
| --- | --- | --- |
| `aps a2a` | Manage A2A (Agent-to-Agent) protocol operations | `card`, `server`, `tasks`, `toggle` |
| `aps acp` | Manage ACP (Agent Client Protocol) server | `server`, `toggle` |
| `aps action` | Manage and execute profile actions | `list`, `run`, `show` |
| `aps adapter` | Manage adapters (messengers, protocols, mobile, desktop) | `approve`, `attach`, `channels`, `create`, `detach`, `exec`, `link`, `list`, `logs`, `messenger`, `pair`, `pending`, `permissions`, `presence`, `reject`, `revoke`, `start`, `status`, `stop`, `test` |
| `aps alias` | Manage command aliases | `add`, `delete`, `list`, `shell` |
| `aps bundle` | Manage capability bundles | `create`, `delete`, `edit`, `list`, `show`, `validate` |
| `aps capability` | Manage capabilities (tools, configs, dotfiles) | `adopt`, `delete`, `disable`, `enable`, `install`, `link`, `list`, `patterns`, `show`, `watch` |
| `aps chat` | Chat with a profile-backed assistant |  |
| `aps completion` | Generate completion script | `fig`, `usage` |
| `aps config` | Inspect aps configuration | `path`, `paths` |
| `aps contact` | Manage contacts via adapter | `add`, `delete`, `find`, `list`, `note`, `show`, `update` |
| `aps directory` | Manage AGNTCY Directory registration and discovery | `delete`, `discover`, `register`, `show` |
| `aps docs` | Generate documentation |  |
| `aps env` | Output environment variables for configured capabilities |  |
| `aps identity` | Manage DID-based agent identity | `badge`, `init`, `show`, `verify` |
| `aps listen` | Subscribe to bus topics for a profile and print events as JSONL |  |
| `aps migrate` | Migrate legacy configurations to new formats | `messengers` |
| `aps observability` | Manage OpenTelemetry observability | `toggle` |
| `aps org` | Inspect the reporting hierarchy across profiles | `check`, `show`, `snapshot` |
| `aps policy` | Manage workspace access policies | `list`, `set`, `show`, `trust` |
| `aps profile` | Manage agent profiles | `capability`, `create`, `delete`, `edit`, `export`, `import`, `list`, `share`, `show`, `status`, `trust`, `workspace` |
| `aps run` | Run a command in a profile context |  |
| `aps serve` | Start protocol server |  |
| `aps service` | Manage profile-facing services | `add`, `conversation`, `routes`, `show`, `start`, `status`, `stop`, `test` |
| `aps session` | Manage sessions | `attach`, `delete`, `detach`, `inspect`, `list`, `logs`, `terminate` |
| `aps skill` | Manage Agent Skills | `install`, `list`, `run`, `show`, `stats`, `suggest`, `validate` |
| `aps squad` | Manage agent squads (topology, membership, scope) | `check`, `create`, `delete`, `list`, `members`, `show` |
| `aps status` | Show aps configuration and runtime status |  |
| `aps toolspec` | Print the aps tool specification (commands, flags, errors, workflows) |  |
| `aps upgrade` | Check for and install updates | `preamble` |
| `aps version` | Print version information |  |
| `aps voice` | Manage voice sessions and the voice backend service | `service`, `start` |
| `aps webhook` | Manage Webhook server | `server`, `toggle` |
| `aps workspace` | Manage multi-agent workspaces | `activity`, `agents`, `archive`, `audit`, `caps`, `conflicts`, `create`, `ctx`, `join`, `leave`, `list`, `members`, `policy`, `remove`, `role`, `send`, `show`, `sync`, `task`, `tasks`, `use` |
<!-- [[[end]]] -->

## Common commands

The sections below cover the commands used most. Their examples are checked against the
command tree by `make lint-docs`.

## aps profile

Manage agent profiles.

```bash
aps profile create <id> [--display-name <s>] [--email <s>] [--force] [--type agent|human] [--reports-to <id>]
aps profile list
aps profile show <id>
aps profile edit <id> [--display-name <s>] [--email <s>]
aps profile delete <id> --confirm-token <token> [--yes] [--force]
aps profile capability add <profile> <capability>
aps profile capability remove <profile> <capability>
```

Other `aps profile` subcommands: `export`, `import`, `share`, `status`, `trust`, and
`workspace`.

## aps run

Run a command under a profile.

```bash
aps run <profile> -- <command> [args...]
aps <profile> -- <command> [args...]   # shorthand, without progress lines
aps <profile>                          # interactive shell
```

The `--` separator is required. `aps run` takes `--env KEY=VALUE` and
`--env-file <path>` (both repeatable) to add variables for one run, and `--idempotency-key`.
The command's exit code becomes the aps exit code. Only the `process` isolation level runs;
see [Isolation Levels](/guides/isolation/).

Profile-scoped external LLM CLIs use the same execution path:

```bash
aps run <profile> -- claude "review this branch"
aps run <profile> -- codex "write tests"
aps run <profile> -- gemini "summarize docs"
aps run <profile> -- opencode "inspect failures"
```

## aps docs

Generate markdown documentation into `docs/` under the aps data directory (`$APS_DATA_PATH`,
else `$XDG_DATA_HOME/aps`, else `~/.local/share/aps`).

```bash
aps docs
```

## aps version

Print version information.

```bash
aps version
aps version --short
```
