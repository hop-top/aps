# Isolation Levels

What the APS isolation setting does today.

## Summary

A profile has an `isolation.level` setting with three accepted values: `process`, `platform`,
and `container`. Only `process` is implemented.

| Level | Status |
|-------|--------|
| `process` (default) | Implemented |
| `platform` | Not wired into `aps run`; the command fails |
| `container` | Not wired into `aps run`; the command fails |

## Process Isolation (Default)

**Status**: Implemented
**Platform Support**: Linux, macOS, Windows

The command runs as a child process of `aps`, as the invoking user. The child receives the
parent environment plus the profile's injected variables.

**What it does:**
- Injects profile environment variables, profile secrets, git config, and SSH key settings
- Keeps each profile's secrets out of runs of other profiles

**What it does not do:**
- Restrict filesystem access: the command can read and write anything your user can
- Restrict network access
- Drop privileges or separate users
- Remove variables already present in the environment `aps` was started from

Process isolation is environment separation, not a sandbox. Do not use it to contain
untrusted code.

```yaml
isolation:
  level: process
```

## Platform and Container Levels

**Status**: Not wired into `aps run`

`level: platform` and `level: container` are accepted when a profile loads (a container
profile also needs `container.image`; an unknown level is rejected). When you run a command
or an action under such a profile, APS stops with an error and runs nothing:

```
platform isolation not yet implemented
container isolation not yet implemented
```

This applies to `aps run`, `aps <profile> -- <command>`, `aps action run`, the webhook
server, and background action jobs. Some other paths ignore the level instead of failing:
`aps serve` (agent-protocol run endpoints) and messenger and ticket routing run the profile's
action as a plain host process, with no sandbox and no error.

Adapter code for a macOS and Linux sandbox user and for Docker exists in the source tree,
with its own tests, but no command calls it. Do not rely on either level to isolate agents
from each other or from your files.

```yaml
isolation:
  level: platform        # or container
  platform:
    name: "sandbox-name"
  container:
    image: "ubuntu:22.04"
```

## Configuration

### Global Configuration

`~/.config/aps/config.yaml`:

```yaml
prefix: APS
isolation:
  default_level: process      # process | platform | container
  fallback_enabled: true
```

Both keys are parsed and saved. `aps run` does not read them.

### Profile Configuration

`profile.yaml`:

```yaml
id: myagent
display_name: "My AI Agent"

isolation:
  level: process              # process | platform | container
  strict: false
  fallback: false
```

Only `level` is read when running a command. A profile with no `level` runs at `process`.
`strict`, `fallback`, and the global `default_level` and `fallback_enabled` are stored and
not applied.

## Fallback

Fallback is not applied. A `platform` or `container` profile fails with the error above
whatever `strict` and `fallback` say, and never degrades to a lower level. The source tree
has a resolver for a container to platform to process chain, with `strict` and the two
`fallback` switches deciding whether it may degrade, plus unit tests for it, but no command
calls it. Setting `strict: true` does not make a profile safer.

## Troubleshooting

### "platform isolation not yet implemented" / "container isolation not yet implemented"

**Cause**: the profile sets a level that `aps run` does not implement.

**Solution**: set `level: process` (or remove the `isolation` block). There is no way to
get a sandbox from `aps run` today.

### "invalid isolation level"

**Cause**: `isolation.level` is not one of `process`, `platform`, `container`.

### "container isolation requires an image"

**Cause**: `level: container` without `isolation.container.image`.

## Related Documentation

- [Profiles](PROFILES.md) - Profile management
- [Sessions](SESSIONS.md) - Session management
- [Security](SECURITY.md) - Security notes
- [Examples](EXAMPLES.md) - Practical examples
