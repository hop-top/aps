# Messenger Integration

APS message services receive chat-like platform payloads and route them to
profile actions.

## Supported Message Adapters

| Adapter alias | Channel ID format | Typical token source | Current support |
| --- | --- | --- | --- |
| `telegram` | numeric chat ID, for example `-1001234567890` | BotFather | JSON webhook route through `aps serve` |
| `slack` | channel ID, for example `C01ABC2DEF` | Slack API dashboard | JSON webhook route through `aps serve` |
| `discord` | numeric channel ID | Discord Developer Portal | JSON webhook route through `aps serve` |
| `sms` | receiving phone number, for example `+15551234567` | SMS provider such as Twilio | JSON relay route through `aps serve` |
| `whatsapp` | phone number ID or receiving number | WhatsApp Cloud API or Twilio | JSON webhook/relay route through `aps serve` |

`github`, `gitlab`, `jira`, `linear`, and `email` are ticket service aliases,
not message aliases.

## Create A Message Service

```bash
aps profile create my-agent

aps service add support-bot \
  --type telegram \
  --profile my-agent \
  --allowed-chat "-1001234567890" \
  --default-action handle-telegram \
  --reply text \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN

aps service show support-bot
aps service routes support-bot
aps service status support-bot --base-url https://hooks.example.com
aps service test support-bot
```

`--type telegram` resolves through kit aliasing to:

```text
type: message
adapter: telegram
```

Use the canonical form when you want to be explicit:

```bash
aps service add support-bot \
  --type message \
  --adapter telegram \
  --profile my-agent
```

## Run The Route

Start the APS HTTP server:

```bash
aps serve --addr 127.0.0.1:8080 --auth-token "$APS_SERVICE_TOKEN"
```

The route printed by `aps service routes support-bot` is mounted at:

```text
POST /services/support-bot/webhook
```

Configure the platform, an ingress gateway, or a small relay to POST
provider-shaped JSON to that URL. `aps service add` records APS service
configuration; it does not create a Slack app, Discord Gateway client, Telegram
poller, Twilio webhook, or public tunnel for you.

For provider setup, run status with the public origin:

```bash
aps service status support-bot --base-url https://hooks.example.com
```

The `webhook_url` line is the effective public endpoint:

```text
webhook_url: https://hooks.example.com/services/support-bot/webhook
```

Use a temporary HTTPS tunnel for local development and register the printed URL
with the provider. For production, use a stable DNS name with a reverse proxy or
load balancer that terminates TLS and forwards to APS on a private address. The
proxy must preserve method, path, query string, headers, and raw body so Slack,
Twilio, and WhatsApp signature checks continue to match provider input.

## Examples

### Slack

```bash
aps service add team-chat \
  --type slack \
  --profile assistant \
  --allowed-channel C01ABC2DEF \
  --default-action triage \
  --reply text \
  --env SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN \
  --env SLACK_SIGNING_SECRET=secret:SLACK_SIGNING_SECRET
```

### Discord

```bash
aps service add community-bot \
  --type discord \
  --profile assistant \
  --allowed-channel 1234567890123456789 \
  --default-action handle-discord \
  --reply text \
  --env DISCORD_BOT_TOKEN=secret:DISCORD_BOT_TOKEN
```

### SMS

```bash
aps service add sms-alerts \
  --type sms \
  --profile assistant \
  --provider twilio \
  --from +15559870002 \
  --allowed-number +15551230001 \
  --default-action handle-sms \
  --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

### WhatsApp

```bash
aps service add wa-support \
  --type whatsapp \
  --profile assistant \
  --provider whatsapp-cloud \
  --phone-number-id 123456789012345 \
  --allowed-number +15551230001 \
  --verify-token-env WHATSAPP_VERIFY_TOKEN \
  --signing-secret-env WHATSAPP_APP_SECRET \
  --default-action handle-whatsapp \
  --reply text \
  --env WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN \
  --env WHATSAPP_VERIFY_TOKEN=secret:WHATSAPP_VERIFY_TOKEN \
  --env WHATSAPP_APP_SECRET=secret:WHATSAPP_APP_SECRET
```

WhatsApp Cloud uses the same service URL for webhook verification and message
POSTs. APS validates `hub.verify_token`, echoes `hub.challenge`, validates
`X-Hub-Signature-256` when an app secret is configured, enforces
`--phone-number-id`, and routes only configured `--allowed-number` senders.

For Twilio WhatsApp, use `--provider twilio`, `--from whatsapp:+1555...`,
`--webhook-url` matching the Twilio console URL, and the Twilio account SID/auth
token env bindings. Twilio form posts and JSON-style relays are both accepted.

### Ticket Alias Contrast

```bash
aps service add jira-intake \
  --type jira \
  --profile triage \
  --site https://example.atlassian.net \
  --project OPS \
  --default-action triage \
  --reply comment
```

This persists `type: ticket`, `adapter: jira`. It is not a chat message
service and is not mounted at `/services/<id>/webhook`.

## Testing

Use:

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

For legacy messenger devices, use:

```bash
aps adapter messenger test my-telegram --profile my-agent --channel "-1001234567890"
```

That command exercises the adapter-device mapping pipeline; it does not prove
that `aps serve` is reachable from Telegram, Slack, Discord, SMS, or WhatsApp.

## Legacy Adapter Devices

Use adapter devices only when you need an external subprocess or existing
device link management:

```bash
aps adapter messenger create my-telegram --type messenger --strategy subprocess
aps adapter messenger link add my-telegram \
  --profile my-agent \
  --mapping "-1001234567890=my-agent=handle-telegram" \
  --default-action "my-agent=default-handler"
aps adapter messenger start my-telegram
aps adapter messenger logs my-telegram -f
```

## Troubleshooting

| Problem | Check |
| --- | --- |
| Service route missing | `aps service routes <service-id>` and `aps serve` |
| Alias resolved unexpectedly | `aps service add <id> --type <alias> --profile <profile> --dry-run` |
| Message not routed | Confirm `--default-action` or legacy channel mapping matches the incoming channel ID |
| Platform cannot reach APS | Check tunnel, DNS, auth token, and `aps serve --addr` binding |
| SMS provider posts forms | Add a relay that converts form fields to JSON before POSTing to APS |

## Security

- Use `aps serve --auth-token` for exposed routes.
- Keep tokens in secret-backed environment variables.
- Treat incoming message text, metadata, and attachments as untrusted input.
- Avoid logging sensitive message content.
- Grant platform apps the minimum permissions needed.
