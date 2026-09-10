# Isolation levels

aps runs every command under a profile at one of three isolation levels. The level is a
profile setting; the global config sets the default and whether a profile may fall back
when its level is unavailable. `aps docs` also generates a longer guide as `ISOLATION.md`
under the data directory.

## Levels

| Level | What it does | Platforms |
|---|---|---|
| `process` (default) | Runs the command in a child process with the profile's environment injected: secrets, git config, SSH key, capability paths | Linux, macOS, Windows |
| `platform` | Runs the command as a dedicated sandbox user account, created with `dscl` on macOS and `useradd` on Linux, with ACL-based access control; sessions attach over SSH | macOS, Linux ([Windows](../dev/platforms/windows/overview.md) is documented as planned) |
| `container` | Runs the command in a Docker container built from a generated Dockerfile, with volume mounts, a network setting, and resource limits; sessions attach over SSH into the container | Anywhere Docker runs |

## Requirements

macOS platform isolation: macOS 10.15 (Catalina) or later, administrator (sudo) access, the
Xcode Command Line Tools (for `dscl` and `launchctl`), an OpenSSH server
(`sudo systemsetup -setremotelogin on`), and an admin key pair. Setup:
[macOS overview](../dev/platforms/macos/overview.md).

Linux platform isolation: kernel 3.10 or later (user namespaces), passwordless sudo, an
OpenSSH server, and the `unshare`, `useradd`, `setfacl`, and `sudo` commands; `tmux` is
optional for session management. Setup: [Linux overview](../dev/platforms/linux/overview.md).

Container isolation: Docker installed and running, the Docker CLI on `PATH`, a reachable
daemon, disk space for images, and rights to manage containers. Setup:
[container overview](../dev/platforms/container/overview.md).

## Configuration

Global default in `~/.config/aps/config.yaml`:

```yaml
isolation:
  default_level: process      # process | platform | container
  fallback_enabled: true      # allow a lower level when the requested one is unavailable
```

Per profile in `profile.yaml`:

```yaml
isolation:
  level: container            # process | platform | container
  strict: false               # true: fail instead of falling back
  fallback: true              # allow falling back to a lower level
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
and `fallback: false`. The `platform` and `container` blocks are read only for their level.

## Fallback

When the requested level is unavailable (no Docker, no sudo, no SSH server) and `fallback`
is on, aps degrades to the next lower level: container to platform to process. With
`fallback: false` the run fails instead, and `strict: true` fails whenever the exact level
is unavailable, even with fallback on. `isolation.fallback_enabled: false` in the global
config turns fallback off for every profile.

## Sessions

`aps session attach` connects to the tmux server behind a session: directly for process
sessions, over SSH into the sandbox user for platform sessions (using the admin private key
at `keys/admin_priv` under the data directory), and over SSH into the container for
container sessions. `--mode view` attaches read-only. `aps session logs` captures the tmux
pane, or `docker logs` for container sessions. Session records live in
`sessions/registry.db` under the data directory; older releases wrote `registry.json`.

## Performance

Summary of [performance-benchmarks.md](../dev/testing/performance-benchmarks.md). The first
platform run includes creating the user account; later runs reuse it.

| Level | Setup time, first / subsequent | Execution overhead | Memory overhead |
|---|---|---|---|
| Process | 0 ms | under 5 ms | about 10 MB |
| Platform, macOS | 150-300 ms / under 50 ms | 10-30 ms | about 50 MB |
| Platform, Linux | 200-400 ms / under 50 ms | 15-40 ms | about 60 MB |
| Container | 2-5 s cold / 100-500 ms warm | 20-50 ms | 100-200 MB |

## Further reading

- [Migrating a profile from process to platform or container](../dev/implementation/guides/migration-guide.md)
- [Container implementation](../dev/platforms/container/container-implementation.md),
  [Linux implementation](../dev/platforms/linux/linux-implementation.md),
  [macOS implementation](../dev/platforms/macos/macos-implementation.md)
- [Security audit](../dev/security/security-audit.md)
- [Configuration and data locations](../dev/configuration.md)
