# IC Ops Team Communication Architecture

Routing architecture for 9 agent profiles + 3 humans
across A2A, Telegram, and email channels.

## Diagram

See [comms-routing-v1.mmd](IC_TEAM_COMMS/comms-routing-v1.mmd).

## Channels

### A2A Protocol (JSON-RPC)

```
Agent A --[send-task]--> Agent B (127.0.0.1:<port>)
         <--[response]--
```

- Transport: HTTP/JSON-RPC on localhost
- Discovery: `/.well-known/agent-card` per agent
- Port range: 8081 (noor) through 8089 (hana)
- Use case: agent-to-agent task delegation, status queries

### Email (himalaya + Gmail)

```
Human --[email]--> jad+<name>@ideacrafters.com
                        |
                   Gmail filter --> label agents/<name>
                        |
                   himalaya poll / mxhook webhook
                        |
                   aps adapter exec email read --profile <name>
```

- Backend: himalaya (IMAP/SMTP via Gmail)
- +subaddressing: all aliases route to same inbox
- Filters: per-agent Gmail labels for triage
- Outbound: `aps adapter exec email send --profile <name>`

### Telegram Bot

```
Human --[message]--> @IdeaCraftersBot
                          |
                     keyword match --> route to agent
                          |
                     aps handle-telegram --profile <name>
                          |
                     agent response --> bot reply
```

- Mode: subprocess (long-polling)
- Routing: regex keyword match per department
- Default: noor (ops lead)
- Config: `~/.ops/.data/aps/devices/ic-telegram/manifest.yaml`

## Agent Port Registry

| Agent | Department | A2A Port |
|-------|-----------|----------|
| noor  | Operations | 8081 |
| sami  | Engineering | 8082 |
| rami  | QA | 8083 |
| kai   | Product | 8084 |
| lina  | Marketing | 8085 |
| amir  | Community | 8086 |
| zara  | Sales | 8087 |
| farid | Finance | 8088 |
| hana  | Legal | 8089 |

## Human Access Matrix

| Human | Email | Telegram | A2A |
|-------|-------|----------|-----|
| Jad (founder) | all agents | bot (any dept) | CLI |
| Monaam (co-founder) | all agents | bot | -- |
| Amine (devops) | sami, noor | bot (eng/ops) | -- |

## Capabilities per Profile

All 9 profiles have:
- `a2a` capability (enabled, unique port)
- `webhooks` capability
- `email` adapter linked
- `contacts` adapter linked
- `ic-telegram` adapter (pending token + link)

## Pending Setup

1. Telegram bot token from @BotFather
2. Link ic-telegram adapter to all profiles
3. Gmail filter import (one-time manual)
4. Gmail Send-As aliases (per runbook)
5. Start A2A servers for active agents
