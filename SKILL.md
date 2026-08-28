---
name: aps
description: "Local-first Agent Profile System: run commands and agent workflows under isolated profiles (process, platform, or container isolation) with per-profile secrets and git identity injection, session tracking, action automation, messenger webhooks (Telegram, Slack, Discord, Teams, SMS, WhatsApp, email), ticket routing, A2A/ACP protocols, skills, squads, and voice sessions. Use when running a command under an agent profile, creating or managing profiles and their secrets, wiring messenger or webhook integrations, attaching to sessions, or scripting the aps CLI."
---

# Using aps

`aps` gives every agent (or context) its own profile: an identity, a
secrets store, a git config, actions, adapters, and an isolation level.
`aps run <profile> -- <cmd>` executes anything inside that environment;
everything else in the CLI manages the profiles and the surfaces
(messengers, webhooks, protocols, voice) that feed work into them.

This page is a router. Find your intent, follow the link.

## I want to…

| I want to… | Read |
|------------|------|
| Install aps | [docs/user/getting-started/install.md](docs/user/getting-started/install.md) |
| Walk through first-run basics | [docs/user/getting-started/tutorial.md](docs/user/getting-started/tutorial.md) |
| Create a profile and run a command under it | [README.md — Quick Start](README.md#quick-start) |
| Look up global flags, exit codes, `--note` inventory | [docs/cli/reference.md](docs/cli/reference.md) |
| Skim command one-liners as a human | [docs/cheatsheet-human.md](docs/cheatsheet-human.md) |
| Drive aps from an agent or script | [docs/cheatsheet-agent.md](docs/cheatsheet-agent.md) |
| Hook a profile to Telegram, Slack, Teams, Discord, SMS, WhatsApp, or email | [docs/MESSENGERS_OVERVIEW.md](docs/MESSENGERS_OVERVIEW.md) |
| Copy-paste messenger setup one-liners | [docs/MESSENGER_SETUP_QUICK_REF.md](docs/MESSENGER_SETUP_QUICK_REF.md) |
| Understand messenger user flows end to end | [docs/user/messengers.md](docs/user/messengers.md) |
| Route inbound tickets (email, Jira, Linear, GitLab) | [docs/user/tickets.md](docs/user/tickets.md) |
| Talk to a profile over the A2A protocol | [docs/user/a2a-quickstart.md](docs/user/a2a-quickstart.md), examples in [docs/user/a2a-examples.md](docs/user/a2a-examples.md) |
| Talk to a profile over the ACP protocol | [docs/user/acp-quickstart.md](docs/user/acp-quickstart.md) |
| Reach my profiles from another machine | [docs/user/remote-access.md](docs/user/remote-access.md) |
| Install or author agent skills | [docs/user/skills/README.md](docs/user/skills/README.md) |
| Start voice (speech-to-speech) sessions | [docs/dev/voice.md](docs/dev/voice.md) |
| Group profiles into squads | [docs/dev/squads.md](docs/dev/squads.md) |
| Understand config files, XDG paths, and data locations | [docs/dev/configuration.md](docs/dev/configuration.md) |
| Connect adapters (messengers, protocols, schedulers, senses…) | [docs/dev/adapters.md](docs/dev/adapters.md) |
| Understand the system architecture | [docs/architecture.md](docs/architecture.md), deeper in [docs/dev/readme.md](docs/dev/readme.md) |
| Work on the codebase as an AI agent | [docs/agent/README.md](docs/agent/README.md) |
| Modify aps itself (build, test, lint, release) | [DEVELOPING.md](DEVELOPING.md) |
| Browse everything | [docs/INDEX.md](docs/INDEX.md) |

## Orientation in 30 seconds

```bash
aps profile create myagent        # create a profile
aps run myagent -- git status     # run anything under it
aps myagent claude "review this"  # shorthand; secrets.env injected
aps session list                  # what is running
aps docs                          # generate the full user docs locally
```

Profile data lives under `~/.agents/`, session state under `~/.aps/`,
config at `~/.config/aps/config.yaml`. `aps help` groups every command
by intent: interact, organize, pipelines, security, instance.
