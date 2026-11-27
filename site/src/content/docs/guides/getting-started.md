---
title: Getting Started
description: Install APS and run your first agent profile.
---

## Install

```bash
# Homebrew (macOS, Linux)
brew install hop-top/tap/aps

# Go 1.27.4 or later
go install hop.top/aps/cmd/aps@latest

# Build from source; writes bin/aps
git clone https://github.com/hop-top/aps.git
cd aps
make build
```

Or download a release binary from [GitHub Releases](https://github.com/hop-top/aps/releases).
Release binaries cover Linux and macOS on amd64 and arm64, and Windows on amd64.

## Create a profile

```bash
aps profile create myagent --display-name "My AI Agent" --email "agent@example.com"
```

This writes `profile.yaml`, `secrets.env`, `gitconfig`, `notes.md`, and an `actions/`
directory under `profiles/myagent/` in the aps data directory. List the profiles with
`aps profile list`.

## Choose an isolation level

```bash
# Process isolation is the default and the only implemented level.
# A profile with no isolation block runs at `process`: a child process as
# your user, with the profile's environment and no sandbox.
#
# `platform` and `container` can be set in profile.yaml but `aps run`
# fails for them; see the Isolation Levels guide.
```

## Run a command under a profile

```bash
# Run any command under the profile's environment
aps run myagent -- echo "Hello from agent!"

# Git with profile's own config
aps run myagent -- git config --get user.name

# Shorthand: profile first
aps myagent -- git status

# Interactive shell as the profile
aps myagent
```

`aps run` prints `[exec]` and `[exit]` progress lines on stderr; the profile-first form
and `--quiet` drop them.

## Generate documentation

```bash
aps docs
```

Markdown files describing the CLI, profiles, isolation, sessions, webhooks, and security
are written to `docs/` under the aps data directory (`$APS_DATA_PATH`, else
`$XDG_DATA_HOME/aps`, else `~/.local/share/aps`).
