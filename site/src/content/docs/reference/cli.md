---
title: CLI Reference
description: Complete command reference for the aps CLI.
---

## Global flags

```
--profile string   Profile to use (overrides APS_PROFILE env)
--debug            Enable debug logging
--help             Show help
```

## aps profile

Manage agent profiles.

```bash
aps profile new <name> [--display-name <s>] [--email <s>] [--isolation-level <l>]
aps profile list
aps profile show <name>
aps profile delete <name>
aps profile cap add <name> <capability>
```

## aps run

Run a command under a profile.

```bash
aps <profile> -- <command> [args...]
aps <profile>              # interactive shell
```

## aps wallet

Manage non-custodial wallets attached to profiles (requires `cap:payment`).

```bash
aps wallet create --network base
aps wallet show
aps wallet balance
```

## aps docs

Generate documentation to `~/.agents/docs/`.

```bash
aps docs
```

## aps version

Print version information.

```bash
aps version
```
