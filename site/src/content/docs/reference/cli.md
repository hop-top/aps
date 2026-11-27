---
title: CLI Reference
description: Complete command reference for the aps CLI.
---

## Global flags

```
-p, --profile string   Profile id (defaults to active profile)
--format string        Output format: csv, human, json, table, text, yaml (default table)
--dry-run              Preview side effects, on commands that support it
--confirm string       Confirmation policy for destructive commands: auto, yes, no, prompt
--offline              Disable network access
--quiet                Suppress non-essential output
-V, --verbose          Increase log verbosity (-V debug, -VV trace)
-h, --help             Show help
```

`aps --help-all` lists every global flag, and `aps docs` generates the full command
reference.

## aps profile

Manage agent profiles.

```bash
aps profile create [id] [--display-name <s>] [--email <s>] [--force] [--type agent|human] [--reports-to <id>]
aps profile list
aps profile show [id]
aps profile edit [id] [--display-name <s>] [--email <s>]
aps profile delete <id> [--yes --confirm-token <token>] [--force]
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
