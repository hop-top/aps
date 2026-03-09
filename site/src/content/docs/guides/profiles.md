---
title: Profiles
description: Understanding APS profiles — identity, config, and capabilities.
---

## What is a profile?

A profile is an isolated identity for an agent. It carries:

- Its own environment variables and shell config
- Its own git identity (`user.name`, `user.email`)
- Its own credential store
- Optional capability bundles (e.g. `cap:payment`)

Profiles are stored under `$XDG_DATA_HOME/aps/profiles/` (default: `~/.local/share/aps/profiles/`).

## Create

```bash
aps profile new <name> [flags]

Flags:
  --display-name string   Human-readable name
  --email string          Git author email
  --isolation-level       process | platform | container (default: process)
```

## List

```bash
aps profile list
```

## Show

```bash
aps profile show <name>
```

## Delete

```bash
aps profile delete <name>
```

## Capability bundles

Add capabilities to a profile:

```bash
aps profile cap add <name> cap:payment --network base
```

See [reference/capabilities](/reference/capabilities/) for available bundles.
