# APS Documentation Index

Quick navigation to all APS documentation.

## Getting Started

- [**Glossary**](glossary.md) — Terms and disambiguation: session, messenger, thread, service, adapter
- [**Messengers Overview**](MESSENGERS_OVERVIEW.md) — Platform comparison, routing architecture, best practices
- [**Conversations**](user/conversations.md) — Turns, conversation and session keys, what actions receive, history commands
- [**Voice**](dev/voice.md) — Speech-to-speech backend, channel adapters, CLI commands
- [**Adapters**](dev/adapters.md) — What adapters are, types, manifest input contract, CLI usage, export/import
- [**Squads**](dev/squads.md) — Squad topologies, contracts, checklist, CLI usage
- [**Configuration**](dev/configuration.md) — XDG directories, config file, profile storage, migration

## Messenger Setup

### Platform Guides
- [**Telegram Setup**](TELEGRAM_SETUP.md) — 2–3 min setup, best for commands and alerts
- [**Discord Setup**](DISCORD_SETUP.md) — 5 min setup, rich features, community use
- [**Messenger Quick Reference**](MESSENGER_SETUP_QUICK_REF.md) — One-liners and troubleshooting for all platforms
- [**Ticket Services**](user/tickets.md) — Email, Jira, Linear, GitLab inbound routes: auth, payloads, sender routing
- [**Message Services**](user/messengers.md) — Service setup, routes, testing, conversation history

### Scripts
- [**Scripts Documentation**](../scripts/README.md) — `setup-telegram.sh`, `setup-messenger.sh`

## Architecture & Design

- [**Squad Topologies Spec**](dev/squad-topologies-spec.md) — Theory: four squad types, three interaction modes, context load
- [**Squads Implementation**](dev/squads.md) — What's built: types, manager, contracts, router, evolution, CLI
- [**Scope System**](dev/scope.md) — Unified scope type, intersection logic, multi-layer resolution
- [**Capability Bundles**](dev/bundles.md) — Named presets grouping capabilities, scope rules, env vars, and services
- [**Voice**](dev/voice.md) — Backend lifecycle, channel adapters (web/TUI/messenger/telephony), session routing
- [**Message Routing**](dev/message-routing.md) — Sender route tables for message services: contacts snapshot, exact/glob match, terminal fail-safe
- [**Messenger Architecture**](dev/messenger-architecture.md) — Webhook flow, routing, turn store
- [**Message Conversation Policy**](dev/message-conversation-policy.md) — Identity keys, turn storage, `prior_turns`
- [**A2A Implementation**](dev/a2a-implementation.md) — Agent-to-agent protocol
- [**ACP Implementation**](dev/acp-implementation.md) — Agent control protocol
- [**Protocol Interface Unification**](dev/protocol-interface-unification.md) — Protocol abstraction layer

## Development Plans

- [**AGNTCY Integration Gap Analysis**](plans/2026-03-01-agntcy-integration-gap-analysis.md)

## Quick Navigation by Task

### Set up a messenger
1. Read [Messengers Overview](MESSENGERS_OVERVIEW.md)
2. Choose: [Telegram](TELEGRAM_SETUP.md) | [Discord](DISCORD_SETUP.md)
3. Run: `./scripts/setup-messenger.sh --type=<platform>`

### Inspect conversation history
```bash
aps service conversation list --service <service-id>
aps service conversation show <conversation-id>
```

See [Conversation History](user/messengers.md#conversation-history) for what routed
actions receive (`conversation`, `prior_turns`) and
[Message Conversation Policy](dev/message-conversation-policy.md) for identity derivation.

### Use voice
```bash
aps voice service start
aps voice start --profile <id> [--channel web|tui|telegram|twilio]
aps voice session list
```

See [Voice](dev/voice.md) for full reference.

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
│   ├── voice.md                   ← voice subsystem
│   ├── squad-topologies-spec.md   ← theory
│   ├── squads.md                  ← implementation
│   ├── adapters.md                ← implementation
│   ├── scope.md                   ← implementation
│   ├── bundles.md                 ← capability bundles
│   └── configuration.md           ← XDG, config, migration
├── plans/
│   ├── 2026-03-16-voice-integration-design.md
│   └── 2026-03-01-agntcy-integration-gap-analysis.md
└── stories/
    └── README.md
```

## Additional Resources

- [Story Index](stories/README.md)
- [Conventions: Version Markers](conventions/version-markers.md)
- [Conventions: Stories](conventions/stories.md)
- [End-to-end Tests](../tests/e2e/)
- [Type Definitions](../internal/core/messenger/types.go)

---

**Last Updated**: 2026-09-03
