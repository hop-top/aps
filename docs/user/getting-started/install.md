---
title: Installation
---

**What you will learn**: How to install APS on your machine, verify
it works, and understand where it stores data — in about five
minutes.

APS (Agent Profile System) is a single Go binary. No daemon, no
containers, no background services required for the base install.

---

## Prerequisites

### 1. Go

APS builds from source with Go **1.26.1 or later** (matches the
project's `go.mod`).

- [Download and install Go](https://go.dev/doc/install)
- Verify: `go version`

### 2. Git

APS uses git for profile workspaces and capability bundles.

- Any recent git works. Verify: `git --version`

That's it for the base install. Additional prerequisites apply only
if you enable specific features:

| Feature                          | Extra requirement                    |
| -------------------------------- | ------------------------------------ |
| Voice backend                    | system audio libs (see voice docs)   |
| Messenger adapters (inbound)     | a tunnel for webhooks — see below    |
| Remote access (Tailscale / CF)   | see [remote-access.md](../remote-access.md) |

---

## Install APS

### From source with `go install`

```bash
go install hop.top/aps/cmd/aps@latest
```

Ensure `$(go env GOPATH)/bin` is on your `$PATH`.

### From a clone

```bash
git clone https://github.com/hop-top/aps.git
cd aps
go build -o aps ./cmd/aps
# Move it somewhere on PATH:
mv aps ~/.local/bin/     # or /usr/local/bin/
```

### Verify

```bash
aps version
```

Expected output (your version will differ):

```
aps vv0.4.0-alpha.2 (c42cb76)
```

If you see that, APS is installed.

---

## Data directory

APS picks its data directory in this order:

1. `$APS_DATA_PATH` — explicit override
2. `$XDG_DATA_HOME/aps` — if XDG is set
3. `~/.local/share/aps` — default on macOS and Linux

This directory holds profiles, sessions, capabilities, and
workspaces. Inspect it any time:

```bash
ls "$(aps env 2>/dev/null | grep APS_DATA_PATH | cut -d= -f2)" \
  || ls ~/.local/share/aps
```

To pin it somewhere else, export `APS_DATA_PATH` in your shell
profile before running `aps`.

---

## Shell completions (recommended)

APS profiles are referenced by ID everywhere. Completions save a
lot of typing.

```bash
# Zsh
aps completion zsh > "${fpath[1]}/_aps"

# Bash
aps completion bash > /usr/local/etc/bash_completion.d/aps

# Fish
aps completion fish > ~/.config/fish/completions/aps.fish
```

Restart your shell.

---

## Shell aliases (optional but useful)

```bash
aps alias >> ~/.zshrc   # or ~/.bashrc
```

This generates one alias per profile so you can type `ea-alex-goad`
instead of `aps ea-alex-goad`. Re-run after adding profiles.

---

## What you get out of the box

Running `aps` with no arguments launches the interactive TUI. From
the CLI, the main surface areas are:

| Command            | What it does                                       |
| ------------------ | -------------------------------------------------- |
| `aps profile`      | Create, list, inspect agent profiles               |
| `aps session`      | Attach, detach, list, inspect, terminate sessions  |
| `aps run`          | Run a command under a profile's context            |
| `aps serve`        | Start the Agent Protocol HTTP server               |
| `aps a2a`          | A2A protocol operations                            |
| `aps acp`          | ACP (Agent Client Protocol) server                 |
| `aps adapter`      | Messenger, protocol, mobile, desktop adapters      |
| `aps capability`   | Install and link external tools                    |
| `aps bundle`       | Manage capability bundles                          |
| `aps workspace`    | Manage workspaces (linked directories)             |
| `aps voice`        | Voice sessions and backend service                 |
| `aps squad`        | Multi-agent squads                                 |

Every command supports `--help`, `--format json|yaml|table`,
`--quiet`, and `--no-hints`.

---

## Troubleshooting

### `aps: command not found`

`$GOPATH/bin` is not on your `$PATH`. Add it:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Put that in `~/.zshrc` or `~/.bashrc` to make it permanent.

### `go install` fails with module errors

Make sure Go is **1.26.1 or later**. Older versions will fail on
APS's generics and toolchain pins.

```bash
go version
```

### Profile commands work but data persists across users

You probably have `APS_DATA_PATH` set in a shared shell profile.
Unset it or point it at a per-user location.

### Where are my profiles?

```bash
ls ~/.local/share/aps/profiles      # default
# or, if you set APS_DATA_PATH:
ls "$APS_DATA_PATH/profiles"
```

---

## Next

- [Tutorial: your first profile and session](./tutorial.md) — five
  minutes, start to teardown.
- [Remote access](../remote-access.md) — expose a profile over
  Tailscale or Cloudflare Tunnel.
- [Messengers](../messengers.md) — wire profiles to Telegram,
  Discord, Slack, GitHub, or email.
