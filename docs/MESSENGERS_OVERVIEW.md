# APS Message Services Overview

APS message services let profiles receive chat-like messages from Telegram,
Slack, Discord, SMS, and WhatsApp.

## Quick Navigation

| Platform | Alias | Channel format | Current mode |
| --- | --- | --- | --- |
| Telegram | `telegram` | numeric chat ID, for example `-1001234567890` | JSON webhook route through `aps serve` |
| Slack | `slack` | channel ID, for example `C01ABC2DEF` | JSON webhook route through `aps serve` |
| Discord | `discord` | numeric channel ID | JSON webhook route through `aps serve` |
| SMS | `sms` | receiving phone number | JSON relay route through `aps serve` |
| WhatsApp | `whatsapp` | phone number ID or receiving number | JSON webhook/relay route through `aps serve` |

Ticket and work-item platforms use ticket services instead:

| Alias | Canonical service |
| --- | --- |
| `email` | `type: ticket`, `adapter: email` |
| `github` | `type: ticket`, `adapter: github` |
| `gitlab` | `type: ticket`, `adapter: gitlab` |
| `jira` | `type: ticket`, `adapter: jira` |
| `linear` | `type: ticket`, `adapter: linear` |

## Setup Command

```bash
aps service add <service-id> --type <message-adapter-alias> --profile <profile-id> [options]
```

Example:

```bash
aps service add support-bot \
  --type telegram \
  --profile assistant \
  --allowed-chat "-1001234567890" \
  --default-action handle-telegram \
  --reply text \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN

aps service show support-bot
aps service routes support-bot
aps serve --addr 127.0.0.1:8080
```

The platform or a relay POSTs JSON to:

```text
POST /services/support-bot/webhook
```

## Architecture

```text
[Messenger Platform or Relay]
        |
        v
POST /services/<service-id>/webhook
        |
        v
[Normalize Message]
        |
        v
[Resolve Route]
        |
        v
[Execute Profile Action]
        |
        v
[Platform-Shaped JSON Response]
```

`aps service add` records APS service configuration. It does not create platform
apps, public tunnels, polling bots, Discord Gateway clients, or Twilio webhook
registrations.

## Normalized Message Format

```json
{
  "id": "msg_unique_id",
  "platform": "telegram|discord|slack|sms|whatsapp",
  "profile_id": "assistant",
  "timestamp": "2026-05-11T10:30:00Z",
  "sender": {
    "id": "user_id",
    "name": "display_name",
    "platform_id": "platform_user_id"
  },
  "channel": {
    "id": "channel_id",
    "name": "channel_name",
    "type": "direct|group|broadcast|topic"
  },
  "text": "message content",
  "thread": { "id": "thread_id", "type": "reply" },
  "attachments": [],
  "platform_metadata": {}
}
```

## Platform Support

| Feature | Telegram | Discord | Slack | SMS | WhatsApp |
| --- | --- | --- | --- | --- | --- |
| Service alias | `telegram` | `discord` | `slack` | `sms` | `whatsapp` |
| Normalization | Bot API message JSON | Message-create JSON | Events API JSON | Phone webhook JSON | Cloud API or Twilio-style JSON |
| Reply | `sendMessage` JSON | content JSON | text JSON | text metadata | text metadata |
| Thread context | replies | replies/threads | `thread_ts` | no | message context |
| External setup | Bot/webhook | Bot/Gateway or relay | App events | Provider relay if form encoded | Cloud webhook or relay |

## Common Patterns

### One Profile, Multiple Platforms

```bash
aps service add telegram-support --type telegram --profile assistant --default-action triage
aps service add slack-support --type slack --profile assistant --default-action triage
aps service add discord-support --type discord --profile assistant --default-action triage
```

Each service has its own route:

```bash
aps service routes telegram-support
aps service routes slack-support
aps service routes discord-support
```

### Different Channels, Different Actions

Service routing currently uses a default action on the service. Use legacy
adapter-device links if you need a route table with many channel-specific
mappings:

```bash
aps adapter messenger create alerts-bot --type messenger --strategy subprocess
aps adapter messenger link add alerts-bot \
  --profile alerts \
  --mapping "-1001111111111=alerts=critical" \
  --mapping "-1002222222222=alerts=warning" \
  --default-action "alerts=general"
```

### Ticket Alias Contrast

```bash
aps service add repo-inbox --type github --profile maintainer --dry-run
# type: ticket
# adapter: github

aps service add team-chat --type slack --profile assistant --dry-run
# type: message
# adapter: slack
```

## Operations

Implemented service commands:

```bash
aps service add <id> --type <type-or-alias> --profile <profile>
aps service show <id>
aps service routes <id>
aps serve --addr 127.0.0.1:8080
```

Legacy messenger-device commands:

```bash
aps adapter messenger list
aps adapter messenger create <name> --type messenger --strategy subprocess
aps adapter messenger start <name>
aps adapter messenger stop <name>
aps adapter messenger status <name>
aps adapter messenger logs <name> -f
aps adapter messenger link list
aps adapter messenger test <name>
```

## Troubleshooting Quick Reference

| Problem | Solution |
| --- | --- |
| Service route unknown | Run `aps service routes <id>` |
| Alias resolved to ticket | Use `telegram`, `slack`, `discord`, `sms`, or `whatsapp` for message services |
| Platform posts form data | Convert to JSON before POSTing to APS |
| Action did not run | Check service `--default-action` and profile action name |
| Platform cannot reach route | Check `aps serve --addr`, auth token, tunnel, and firewall |

## Next Steps

1. Create a service with `aps service add`.
2. Check alias resolution with `--dry-run`.
3. Start `aps serve`.
4. Configure the platform or relay to POST JSON to the printed route.
