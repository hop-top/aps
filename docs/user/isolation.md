# Isolation levels

aps has three isolation level settings: `process`, `platform`, and `container`. Only
`process` is implemented. A profile can be set to `platform` or `container`, and the setting
is accepted and stored, but `aps run` and `aps action run` refuse to run under either: they
exit with an error and run nothing. `aps docs` also generates a longer guide as
`ISOLATION.md` under the data directory.

## Levels

| Level | Status | What it does |
|---|---|---|
| `process` (default) | Implemented | Runs the command as a child process of aps, as your user, with your environment plus the profile's variables injected (secrets, git config, SSH key, capability paths). No filesystem, network, or privilege boundary. Linux, macOS, Windows. |
| `platform` | Not wired into `aps run` | Selectable in `profile.yaml`. `aps run`, `aps action run`, and webhook-triggered actions fail with `platform isolation not yet implemented`. Sandbox-user adapters for macOS and Linux exist in the source tree but no command calls them. |
| `container` | Not wired into `aps run` | Selectable in `profile.yaml` (an `image` is required). Same failure: `container isolation not yet implemented`. A Docker adapter exists in the source tree but no command calls it. |

`process` is not a sandbox. The child can read and write anything your user can, and it
inherits every variable in the environment aps was started from. Profile separation under
`process` means each profile injects its own secrets and git identity; it does not stop a
command from reading another profile's files.

### What happens at each level

- `aps run <profile> -- <command>` and `aps <profile> -- <command>`: `process` runs the
  command. `platform` and `container` print the error above and exit 1 without starting the
  command.
- `aps action run`, the webhook server, and background action jobs: same as `aps run`.
- `aps serve` (the agent-protocol run endpoints) and messenger and ticket routing: same as
  `aps run`. A `platform` or `container` profile is refused and its action is not started.
  The run endpoints report a failed run whose `error` is the not-implemented message, and
  messenger and ticket routing report a failed result carrying it.
- An unknown level, or `container` without `container.image`, is rejected when the profile
  loads.

Do not rely on `platform` or `container` to contain an untrusted agent or to keep it away
from your files, credentials, or other profiles. If you need that boundary today, run aps
itself inside a VM, a container, or a separate OS account that you set up.

## Configuration

Global setting in `~/.config/aps/config.yaml` (parsed and saved; `aps run` does not read it):

```yaml
isolation:
  default_level: process      # process | platform | container
  fallback_enabled: true      # intended: allow a lower level; not applied today
```

Per profile in `profile.yaml`. Only `level: process` runs; the other fields are stored and
validated but have no effect on `aps run`:

```yaml
isolation:
  level: container            # process | platform | container (aps run fails for the last two)
  strict: false               # intended: fail instead of falling back; not applied today
  fallback: true              # intended: allow a lower level; not applied today
  platform:
    name: macos-sandbox       # optional sandbox name
  container:
    image: ubuntu:22.04
    packages:
      - git
      - curl
    build_steps:
      - type: shell
        run: npm install -g some-tool
    volumes:
      - /tmp/aps-shared:/workspace
    network: bridge
    resources:
      memory_mb: 1024
      cpu_quota: 100000
```

A profile created by `aps profile create` starts with `level: process`, `strict: false`,
and `fallback: false`. The `platform` and `container` blocks are used only with their own level, and `aps run` does not act on them yet.

## Fallback

Fallback is not applied. `aps run` and `aps action run` read only `isolation.level`: for
`platform` or `container` they fail with the not-implemented error whatever `strict` and
`fallback` say, and they never degrade to a lower level. The profile `strict` and
`fallback` fields and the global `isolation.default_level` and `isolation.fallback_enabled`
settings are parsed and saved, and nothing in the run path reads them. A profile with no
`level` runs at `process`; the global `default_level` does not change that.

The source tree has a fallback resolver (container to platform to process, with `strict` and
the two `fallback` switches deciding whether it may degrade) and unit tests for it, but no
command calls it. Setting `strict: true` does not make a profile safer today.

## Sessions

Session records live in `sessions/registry.db` under the data directory. `aps session
attach` connects to the tmux server behind a session, and `--mode view` attaches read-only.
It has an SSH branch for sessions recorded as macOS or Linux sandbox sessions (using the
admin private key at `keys/admin_priv` under the data directory), but no aps command creates
such a session, because `platform` is not wired into `aps run`; it has no branch for
container sessions. `aps session logs` captures the tmux pane, or `docker logs` for a record
that carries a container id and no tmux socket.

## Further reading

- Adapter design and reference for the unwired levels (not current behaviour):
  [container](../dev/platforms/container/container-implementation.md),
  [Linux](../dev/platforms/linux/linux-implementation.md),
  [macOS](../dev/platforms/macos/macos-implementation.md)
- [Security audit](../dev/security/security-audit.md)
- [Configuration and data locations](../dev/configuration.md)
