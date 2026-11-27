---
title: Isolation Levels
description: What each isolation setting does today, and what it does not.
---

A profile has an `isolation.level` setting with three accepted values. Only `process` is
implemented. `platform` and `container` are not wired into `aps run`.

| Level | Status |
|-------|--------|
| `process` (default) | Implemented |
| `platform` | Not wired into `aps run` |
| `container` | Not wired into `aps run` |

## Process (default)

Runs the command as a child process of aps, as your user. The child gets your environment
plus the profile's variables (secrets, git config, SSH key, capability paths). There is no
filesystem, network, or privilege boundary, and the child can read anything your user can.

**Best for:** development and trusted workloads.

## Platform and container

You can set `level: platform` or `level: container` in `profile.yaml`; a container profile
also needs `container.image`. aps accepts the setting, but `aps run` and `aps action run`
then fail with `platform isolation not yet implemented` or `container isolation not yet
implemented` and run nothing. The `strict` and `fallback` fields, and the global
`isolation.default_level` and `isolation.fallback_enabled`, are stored but not applied, so
there is no automatic fallback to `process`.

`aps serve` and messenger and ticket routing run a profile's actions as plain host processes
whatever level the profile sets, so those paths give no sandbox either.

Do not rely on either setting to contain an untrusted agent. Adapter code for a macOS and
Linux sandbox user and for Docker exists in the source tree, but no command calls it.

## Comparison

| Feature              | Process | Platform | Container |
|----------------------|---------|----------|-----------|
| Runs under `aps run` | Yes     | No       | No        |
| Filesystem isolation | No      | n/a      | n/a       |
| Network isolation    | No      | n/a      | n/a       |
