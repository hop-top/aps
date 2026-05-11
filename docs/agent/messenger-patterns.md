# Messenger Patterns for Agents

Patterns for receiving chat-like platform messages through profile-facing APS
services.

## Setup

Use `aps service add` for new message services. Adapter names such as
`telegram`, `slack`, `discord`, `sms`, and `whatsapp` are service type aliases:
kit aliasing resolves them to canonical `type: message` plus the concrete
adapter.

```bash
aps service add support-bot \
  --type telegram \
  --profile my-agent \
  --allowed-chat "-1001234567890" \
  --default-action handle-telegram \
  --reply text

aps service show support-bot
aps service routes support-bot
aps service status support-bot --base-url https://hooks.example.com
aps service test support-bot
```

`aps service show` reports the resolved type, adapter, profile, route, reply
shape, and maturity. `aps service routes support-bot` prints:

```text
/services/support-bot/webhook
```

Run `aps serve` to mount the message service route. Platform webhooks or a
relay should POST provider-shaped JSON to that route.

## Route And Test Expectations

The message service route is:

```text
POST /services/<service-id>/webhook
```

The handler normalizes the provider payload, resolves a channel route, executes
the target profile action, and returns platform-shaped JSON. For service routes,
`--default-action handle-telegram` resolves inside the service profile; explicit
legacy mappings use `profile=action`.

Current lightweight checks are:

```bash
aps service show support-bot
aps service routes support-bot
aps service status support-bot --base-url https://hooks.example.com
aps service test support-bot
aps serve --addr 127.0.0.1:8080
curl -X POST http://127.0.0.1:8080/services/support-bot/webhook \
  -H 'content-type: application/json' \
  -d '{
    "message": {
      "message_id": 1,
      "from": {"id": 456},
      "chat": {"id": -1001234567890},
      "text": "hello"
    }
  }'
```

`aps adapter messenger test <device>` is for the older adapter-device link
pipeline. It checks normalization, route lookup, and simulated delivery for a
messenger device; it does not prove that an external platform webhook or bot
client is delivering messages to `aps serve`.

## Supported Message Adapters

| Adapter alias | Canonical config | Incoming payload support | Reply shape | Notes |
| --- | --- | --- | --- | --- |
| `telegram` | `type: message`, `adapter: telegram` | Telegram Bot API `message` and `edited_message` JSON | Telegram `sendMessage` JSON | Chat IDs are numeric; groups usually start with `-100`. |
| `slack` | `type: message`, `adapter: slack` | Slack Events API event envelope JSON | Slack text response JSON | Slack URL verification and app provisioning are outside `aps service add`. |
| `discord` | `type: message`, `adapter: discord` | Discord message-create style JSON | Discord content response JSON | Discord Gateway/bot runtime is not created by `aps service add`. |
| `sms` | `type: message`, `adapter: sms` | Twilio-style form or JSON phone fields | Text response metadata | Twilio signatures require exact public `--webhook-url`. |
| `whatsapp` | `type: message`, `adapter: whatsapp` | WhatsApp Cloud API JSON or Twilio-style WhatsApp JSON | Text response metadata | Use `--phone-number-id` for Cloud API channel IDs. |

## Message Vs Ticket Aliases

Chat platforms resolve to `message`. Work-item platforms resolve to `ticket`:

```bash
aps service add support-bot --type slack --profile assistant --dry-run
# type: message
# adapter: slack

aps service add jira-intake --type jira --profile triage --dry-run
# type: ticket
# adapter: jira
```

Do not use ticket aliases such as `github`, `gitlab`, `jira`, or `linear` for
chat-message routing. They are persisted as ticket services.

## Legacy Adapter Device Links

Use the adapter-device CLI only when you are managing a messenger subprocess or
an existing `messenger-links.json` route table.

```bash
aps adapter messenger create my-telegram --type messenger --strategy subprocess
aps adapter messenger link add my-telegram \
  --profile my-agent \
  --mapping "-1001234567890=my-agent=handle-telegram" \
  --default-action "my-agent=default-handler"
aps adapter messenger start my-telegram
aps adapter messenger test my-telegram --profile my-agent --channel "-1001234567890"
```

## Related

- [Messenger architecture](../dev/messenger-architecture.md)
- [User messenger guide](../user/messengers.md)
- [Service UX draft](../dev/service-ux-draft.md)
