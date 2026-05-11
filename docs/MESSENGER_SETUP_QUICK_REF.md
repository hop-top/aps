# Message Service Quick Reference

Use `aps service` for new Telegram, Slack, Discord, SMS, and WhatsApp message
integrations.

## Fastest Path

```bash
aps service add support-bot \
  --type telegram \
  --profile my-agent \
  --allowed-chat "-1001234567890" \
  --default-action handle-message \
  --reply text \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN

aps service show support-bot
aps service routes support-bot
aps serve --addr 127.0.0.1:8080
```

The route is:

```text
POST /services/support-bot/webhook
```

Configure the platform or a relay to POST provider-shaped JSON to that route.

## Alias Reference

Message adapter aliases:

| Alias | Canonical config |
| --- | --- |
| `telegram` | `type: message`, `adapter: telegram` |
| `slack` | `type: message`, `adapter: slack` |
| `discord` | `type: message`, `adapter: discord` |
| `sms` | `type: message`, `adapter: sms` |
| `whatsapp` | `type: message`, `adapter: whatsapp` |

Ticket aliases, not message aliases:

| Alias | Canonical config |
| --- | --- |
| `email` | `type: ticket`, `adapter: email` |
| `github` | `type: ticket`, `adapter: github` |
| `gitlab` | `type: ticket`, `adapter: gitlab` |
| `jira` | `type: ticket`, `adapter: jira` |
| `linear` | `type: ticket`, `adapter: linear` |

Check alias resolution without writing:

```bash
aps service add demo --type slack --profile my-agent --dry-run
aps service add issue-demo --type jira --profile triage --dry-run
```

## Platform Examples

### Telegram

```bash
aps service add telegram-support \
  --type telegram \
  --profile assistant \
  --allowed-chat "-1001234567890" \
  --default-action handle-telegram \
  --reply text \
  --webhook-secret-token-env TELEGRAM_WEBHOOK_SECRET \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN
```

### Slack

```bash
aps service add slack-support \
  --type slack \
  --profile assistant \
  --allowed-channel C01ABC2DEF \
  --default-action triage \
  --reply text \
  --env SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN
```

### Discord

```bash
aps service add discord-support \
  --type discord \
  --profile assistant \
  --allowed-channel 1234567890123456789 \
  --default-action handle-discord \
  --reply text \
  --env DISCORD_TOKEN=secret:DISCORD_TOKEN
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
  --default-action handle-whatsapp \
  --reply text \
  --env WHATSAPP_TOKEN=secret:WHATSAPP_TOKEN
```

## Test And Verify

There is no `aps service test` command yet. Use these checks:

```bash
aps service show support-bot
aps service routes support-bot
aps serve --addr 127.0.0.1:8080
curl -X POST http://127.0.0.1:8080/services/support-bot/webhook \
  -H 'content-type: application/json' \
  -H "x-telegram-bot-api-secret-token: $TELEGRAM_WEBHOOK_SECRET" \
  -d '{"update_id":1000001,"message":{"message_id":1,"from":{"id":456},"chat":{"id":-1001234567890},"text":"hello"}}'
```

For legacy adapter devices:

```bash
aps adapter messenger test my-telegram --profile my-agent --channel "-1001234567890"
```

That command simulates the adapter-device route pipeline. It does not verify
that `aps serve` is reachable from the platform.

## Legacy Adapter Device Commands

Use this path only for subprocess devices or existing messenger link stores:

```bash
aps adapter messenger create my-telegram --type messenger --strategy subprocess
aps adapter messenger link add my-telegram \
  --profile my-agent \
  --mapping "-1001234567890=my-agent=handle-message" \
  --default-action "my-agent=default-handler"
aps adapter messenger start my-telegram
aps adapter messenger logs my-telegram -f
aps adapter messenger stop my-telegram
```

## Troubleshooting

| Problem | Check |
| --- | --- |
| Alias surprise | `aps service add <id> --type <alias> --profile <profile> --dry-run` |
| Route missing | `aps service routes <id>` and `aps serve` |
| No action execution | Confirm `--default-action` exists on the service profile |
| Platform cannot connect | Check tunnel, DNS, `--auth-token`, and bind address |
| SMS form posts fail | Convert provider form fields to JSON before POSTing to APS |

## Security Checklist

- Store real tokens in secret-backed environment variables.
- Use `aps serve --auth-token` for exposed HTTP routes.
- Validate all incoming message content in profile actions.
- Avoid logging sensitive message payloads.
- Grant platform apps minimum required permissions.
