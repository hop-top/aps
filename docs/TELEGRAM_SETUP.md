# Telegram Message Service Guide

This guide shows how to receive Telegram Bot API updates through an APS message
service.

## What You Need

1. A Telegram bot token from [BotFather](https://t.me/botfather).
2. An APS profile with the action you want to run.
3. A chat ID for the direct message, group, supergroup, or channel.
4. A way for Telegram to reach `aps serve`, such as a public HTTPS ingress or a
   relay.

## Create The Service

```bash
aps profile create my-agent

aps service add telegram-support \
  --type telegram \
  --profile my-agent \
  --allowed-chat "-1001234567890" \
  --default-action handle-telegram \
  --reply text \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN

aps service show telegram-support
aps service routes telegram-support
```

`--type telegram` resolves through kit aliasing to:

```text
type: message
adapter: telegram
```

## Run The Endpoint

```bash
aps serve --addr 127.0.0.1:8080 --auth-token "$APS_SERVICE_TOKEN"
```

The service route is:

```text
POST /services/telegram-support/webhook
```

Configure Telegram, your ingress, or your relay to POST Bot API update JSON to
that route. `aps service add` does not set the Telegram webhook or run a
polling bot.

## Find A Chat ID

1. Send a test message to the bot or group.
2. Visit `https://api.telegram.org/bot<TOKEN>/getUpdates`.
3. Look for `message.chat.id`.

Common formats:

| Chat type | Example |
| --- | --- |
| Direct message | `123456789` |
| Group or supergroup | `-1001234567890` |
| Channel | negative numeric ID |

## Smoke Test

There is no `aps service test` command yet. Use `aps serve` and POST a
Telegram-shaped JSON payload:

```bash
curl -X POST http://127.0.0.1:8080/services/telegram-support/webhook \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $APS_SERVICE_TOKEN" \
  -d '{"message":{"message_id":1,"from":{"id":456,"first_name":"Alice"},"chat":{"id":-1001234567890,"type":"supergroup"},"text":"hello"}}'
```

The service normalizes the update, routes it to `my-agent=handle-telegram`, and
returns Telegram `sendMessage` response JSON when the action succeeds.

## Normalized Message

Your action receives a JSON payload like:

```json
{
  "id": "msg_...",
  "platform": "telegram",
  "profile_id": "my-agent",
  "sender": {
    "id": "456",
    "name": "Alice",
    "platform_id": "456"
  },
  "channel": {
    "id": "-1001234567890",
    "type": "group",
    "platform_id": "-1001234567890"
  },
  "text": "hello",
  "platform_metadata": {}
}
```

## Multiple Channels

The service form currently has one `default_action`. Use separate services for
simple separation:

```bash
aps service add telegram-alerts \
  --type telegram \
  --profile alerts \
  --allowed-chat "-1001111111111" \
  --default-action process-alert

aps service add telegram-commands \
  --type telegram \
  --profile commands \
  --allowed-chat "-1002222222222" \
  --default-action process-command
```

Use legacy adapter-device links only when you need an explicit channel mapping
table:

```bash
aps adapter messenger create main-telegram --type messenger --strategy subprocess
aps adapter messenger link add main-telegram \
  --profile notifications \
  --mapping "-1001111111111=notifications=process" \
  --mapping "-1002222222222=notifications=process-command" \
  --default-action "notifications=default-handler"
aps adapter messenger start main-telegram
```

## Troubleshooting

| Problem | Check |
| --- | --- |
| Route missing | `aps service routes telegram-support` and `aps serve` |
| Bot token rejected | Verify the token with BotFather and your relay/platform config |
| Message not routed | Confirm `--default-action` exists on the service profile |
| Telegram cannot reach APS | Check public HTTPS ingress, tunnel, auth token, and bind address |
| Need polling | Use a legacy adapter subprocess or external relay; `aps service add` is webhook-route configuration |

## Security

- Use private groups/channels for sensitive commands.
- Use `aps serve --auth-token` if exposed beyond localhost.
- Keep the bot token in a secret-backed environment variable.
- Validate message text and attachments in profile actions.
