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
# Process isolation (default, fastest)
aps profile new myagent

# Platform isolation — macOS/Linux user-level sandbox
aps profile new myagent --isolation-level platform

# Container isolation — strongest, requires Docker
aps profile new myagent --isolation-level container
```

## Run a command under a profile

```bash
# Run any command isolated under the profile
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
