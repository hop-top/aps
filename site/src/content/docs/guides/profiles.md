---
title: Profiles
description: Understanding APS profiles — identity, config, and capabilities.
---

## What is a profile?

A profile is a separate identity and environment for an agent. It carries:

- Its own environment variables and shell setting
- Its own git identity (`user.name`, `user.email`), written to a `gitconfig` file
- Its own secrets, in `secrets.env` (mode 0600)
- Optional capabilities (for example `a2a`)

Each profile is a directory under `profiles/` in the aps data directory: `$APS_DATA_PATH`,
else `$XDG_DATA_HOME/aps`, else `~/.local/share/aps`. It holds `profile.yaml`, `secrets.env`,
`gitconfig`, `notes.md`, and an `actions/` directory.

## Create

```bash
aps profile create [id] [flags]

Flags:
  --display-name string   Display name for the profile
  --email string          Email for profile and git config
  --force                 Overwrite existing profile
  --type string           Profile type: agent or human (empty means agent)
  --reports-to string     Profile id this profile reports to (must exist)
  --avatar string         Url or local path to profile image
  --color string          Hex color (e.g. #3b82f6) for UI rendering
  -n, --note string       Audit note recorded with the bus event
```

`--auto-avatar`, `--auto-color`, and the `--avatar-*` provider flags generate those values
from the profile id; see `aps profile create --help`. On a terminal, `create` prompts for
missing fields. Without one, it uses only the flags you pass. Creating an id that already
exists fails with a conflict unless you pass `--force`.

```console
$ aps profile create myagent --display-name "My AI Agent" --email agent@example.com
Profile 'myagent' created successfully.
```

To change a profile later, use `aps profile edit <id>` with the same identity flags.

## List

```bash
aps profile list
```

```console
ID       DISPLAY NAME  ROLES  CAPABILITIES  WORKSPACE  EMAIL              SECRETS  DID    COLOR  AVATAR
myagent  My AI Agent                                   agent@example.com  false    false
```

Filters: `--capability`, `--role`, `--squad`, `--tone`, `--has-secrets`, `--has-identity`.
Add `--format json`, `yaml`, or `csv` for machine-readable output.

## Show

```bash
aps profile show <name>
```

```console
id: myagent
display_name: My AI Agent
email: agent@example.com
preferences:
    shell: /bin/zsh
git:
    enabled: true
isolation:
    level: process
    strict: false
    fallback: false
```

The shell setting follows your environment. Capabilities, when present, are listed after
the isolation block.

## Delete

```bash
aps profile delete <name>
```

`delete` always needs a confirmation token, on a terminal and in a script. Run it once
without one: nothing is deleted, it exits with code 5, and it prints the token (under
`--format json` the error carries it in `message` and `suggested_fix`). Run it again with
`--confirm-token <token>`. The token depends only on the command, not on the profile, so it
is the same every time. On a terminal `delete` then asks a yes/no question; `--yes` skips
that question and does not replace the token. `--force` deletes a profile that still has
active sessions or inbound `reports_to` links.

## Capabilities

Add a capability to a profile, or remove it:

```bash
aps profile capability add <name> <capability>
aps profile capability remove <name> <capability>
```

`aps capability list` shows the names that exist: the built-in ones (`a2a`,
`agent-protocol`, `webhooks`, `agntcy-observability`, `agntcy-directory`, `agntcy-identity`,
`agntcy-trust`, `ibr`) and any you installed with `aps capability install`. Naming one that
does not exist fails with `capability '<name>' does not exist`.

Capability bundles, which group several capabilities, are managed with `aps bundle`
(`list`, `show`, `create`, `edit`, `validate`, `delete`).
