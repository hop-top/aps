# APS Documentation Index

Quick navigation to all APS documentation.

## Getting Started

- [**Messengers Overview**](MESSENGERS_OVERVIEW.md) — Platform comparison, routing architecture, best practices
- [**Adapters**](dev/adapters.md) — What adapters are, types, CLI usage, export/import
- [**Squads**](dev/squads.md) — Squad topologies, contracts, checklist, CLI usage
- [**Configuration**](dev/configuration.md) — XDG directories, config file, profile storage, migration

## Messenger Setup

### Platform Guides
- [**Telegram Setup**](TELEGRAM_SETUP.md) — 2–3 min setup, best for commands and alerts
- [**Discord Setup**](DISCORD_SETUP.md) — 5 min setup, rich features, community use
- [**Messenger Quick Reference**](MESSENGER_SETUP_QUICK_REF.md) — One-liners and troubleshooting for all platforms

### Scripts
- [**Scripts Documentation**](../scripts/README.md) — `setup-telegram.sh`, `setup-messenger.sh`

## Architecture & Design

- [**Squad Topologies Spec**](dev/squad-topologies-spec.md) — Theory: four squad types, three interaction modes, context load
- [**Squads Implementation**](dev/squads.md) — What's built: types, manager, contracts, router, evolution, CLI
- [**Scope System**](dev/scope.md) — Unified scope type, intersection logic, multi-layer resolution
- [**A2A Implementation**](a2a-implementation.md) — Agent-to-agent protocol
- [**ACP Implementation**](acp-implementation.md) — Agent control protocol
- [**Protocol Interface Unification**](protocol-interface-unification.md) — Protocol abstraction layer

## Development Plans

- [**AGNTCY Integration Gap Analysis**](plans/2026-03-01-agntcy-integration-gap-analysis.md)

## Quick Navigation by Task

### Set up a messenger
1. Read [Messengers Overview](MESSENGERS_OVERVIEW.md)
2. Choose: [Telegram](TELEGRAM_SETUP.md) | [Discord](DISCORD_SETUP.md)
3. Run: `./scripts/setup-messenger.sh --type=<platform>`

### Manage adapters
```bash
aps adapter list
aps adapter create <name> --type messenger
aps adapter export <name> --output adapter.yaml
aps adapter import adapter.yaml
```

See [Adapters](dev/adapters.md) for full reference.

### Work with squads
```bash
aps squad list
aps squad create <name> --type stream-aligned --domain <domain>
aps squad check
```

See [Squads](dev/squads.md) for full reference.

### Understand configuration paths
See [Configuration](dev/configuration.md) for XDG directories and migration from legacy paths.

## File Structure

```
docs/
├── INDEX.md                        ← this file
├── MESSENGERS_OVERVIEW.md
├── TELEGRAM_SETUP.md
├── DISCORD_SETUP.md
├── MESSENGER_SETUP_QUICK_REF.md
├── dev/
│   ├── squad-topologies-spec.md   ← theory
│   ├── squads.md                  ← implementation
│   ├── adapters.md                ← implementation
│   ├── scope.md                   ← implementation
│   └── configuration.md           ← XDG, config, migration
├── plans/
│   └── 2026-03-01-agntcy-integration-gap-analysis.md
└── stories/
    └── README.md
```

## Additional Resources

- [Story Index](stories/README.md)
- [End-to-end Tests](../tests/e2e/)
- [Type Definitions](../internal/core/messenger/types.go)

---

**Last Updated**: 2026-03-08
