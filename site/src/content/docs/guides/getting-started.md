---
title: Getting Started
description: Install APS and run your first agent profile.
---

## Install

```bash
# Build from source
git clone https://github.com/hop-top/aps.git
cd aps
make build
```

Or download a release binary from [GitHub Releases](https://github.com/hop-top/aps/releases).

## Create a profile

```bash
aps profile new myagent --display-name "My AI Agent" --email "agent@example.com"
```

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
aps myagent -- echo "Hello from agent!"

# Git with profile's own config
aps myagent -- git status

# Interactive shell
aps myagent
```

## Generate documentation

```bash
aps docs
```

Docs will be written to `~/.agents/docs/`.
