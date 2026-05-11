# Discord Message Service Guide

This guide shows how to receive Discord message payloads through an APS message
service.

## What You Need

1. A Discord bot token from the [Discord Developer Portal](https://discord.com/developers/applications).
2. An APS profile with the action you want to run.
3. Discord guild and channel IDs.
4. A Discord Gateway client, bot process, or relay that can POST message-create
   JSON to `aps serve`.
5. For Discord Interactions, the application public key from the Developer
   Portal.

`aps service add` records APS routing configuration. It does not create or run a
Discord Gateway client.

## Create The Service

```bash
aps profile create my-agent

aps service add discord-support \
  --type discord \
  --profile my-agent \
  --allowed-guild 9876543210 \
  --allowed-channel 1234567890123456789 \
  --default-action handle-discord \
  --reply text \
  --env DISCORD_BOT_TOKEN=secret:DISCORD_BOT_TOKEN

aps service show discord-support
aps service routes discord-support
```

`--type discord` resolves through kit aliasing to:

```text
type: message
adapter: discord
```

## Run The Endpoint

```bash
aps serve --addr 127.0.0.1:8080 --auth-token "$APS_SERVICE_TOKEN"
```

The service route is:

```text
POST /services/discord-support/webhook
```

Configure your Discord bot process or relay to POST Discord message-create
payloads to that route.

## Get Channel IDs

1. Enable Developer Mode in Discord.
2. Right-click a channel.
3. Select "Copy Channel ID".

Discord channel IDs are numeric strings, for example
`1234567890123456789`.

## Smoke Test

Use `aps service test` for static validation:

```bash
aps service test discord-support
```

To exercise the route manually, run `aps serve` and POST a Discord-shaped JSON
payload:

```bash
curl -X POST http://127.0.0.1:8080/services/discord-support/webhook \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $APS_SERVICE_TOKEN" \
  -d '{"id":"111","channel_id":"1234567890123456789","guild_id":"9876543210","content":"hello","author":{"id":"456","username":"alice"}}'
```

The service normalizes the message, routes it to `my-agent=handle-discord`, and
returns Discord response JSON when the action succeeds.

Gateway relays can also post Discord gateway envelopes. APS unwraps
`t: "MESSAGE_CREATE"` and normalizes the nested `d` payload.

## Interaction Signatures

Discord Interactions use Ed25519 signatures. Configure the service for signed
interaction requests with a public key:

```bash
aps service add discord-interactions \
  --type discord \
  --profile my-agent \
  --receive interaction \
  --default-action handle-discord \
  --env DISCORD_BOT_TOKEN=secret:DISCORD_BOT_TOKEN \
  --env DISCORD_PUBLIC_KEY=env:DISCORD_PUBLIC_KEY
```

The validator expects `X-Signature-Ed25519` and `X-Signature-Timestamp` headers.
If you set `discord_public_key` or `discord_public_key_env` in service options,
or bind `DISCORD_PUBLIC_KEY=env:<name>`, APS verifies `timestamp + body` using
that public key before normalizing.

## Normalized Message

Your action receives a JSON payload like:

```json
{
  "id": "111",
  "platform": "discord",
  "workspace_id": "9876543210",
  "profile_id": "my-agent",
  "sender": {
    "id": "456",
    "name": "alice",
    "platform_handle": "alice",
    "platform_id": "456"
  },
  "channel": {
    "id": "1234567890123456789",
    "type": "group",
    "platform_id": "1234567890123456789"
  },
  "text": "hello",
  "platform_metadata": {}
}
```

## Threads And Replies

The normalizer preserves Discord `thread_id`, `message_reference.message_id`,
or `referenced_message.id` as normalized thread context. Your profile action can
use that context when deciding how to reply.

Outbound delivery uses the Discord channel messages API. Reply threads are sent
with `message_reference`; topic-style threads are sent to the thread channel.
Attachment URLs are carried as embeds, with image attachments rendered as image
embeds. Direct file upload is not attempted from normalized URL-only
attachments.

## Multiple Servers Or Channels

Use separate services for simple service-level routing:

```bash
aps service add discord-alerts \
  --type discord \
  --profile alerts \
  --allowed-channel 1111111111111111111 \
  --default-action process-alert

aps service add discord-commands \
  --type discord \
  --profile commands \
  --allowed-channel 2222222222222222222 \
  --default-action process-command
```

Use legacy adapter-device links only when you need one subprocess device with a
channel mapping table:

```bash
aps adapter messenger create discord-bot --type messenger --strategy subprocess
aps adapter messenger link add discord-bot \
  --profile alerts \
  --mapping "1111111111111111111=alerts=process-alert" \
  --mapping "2222222222222222222=alerts=process-command" \
  --default-action "alerts=process-unknown"
aps adapter messenger start discord-bot
```

## Comparing Discord And Slack

| Feature | Discord | Slack |
| --- | --- | --- |
| Message service alias | `discord` | `slack` |
| Channel ID format | numeric string | `C...`/Slack channel ID |
| Native thread context | yes | yes, via `thread_ts` |
| External runtime | Gateway bot or relay | Slack Events app or relay |
| APS route | `/services/<id>/webhook` | `/services/<id>/webhook` |

## Troubleshooting

| Problem | Check |
| --- | --- |
| Route missing | `aps service routes discord-support` and `aps serve` |
| Bot cannot read channel | Discord bot permissions and channel ID |
| Guild rejected | Confirm the incoming `guild_id` matches `--allowed-guild` |
| Message not routed | Confirm `--default-action` exists on the service profile |
| Gateway events not arriving | Check your bot/relay process; APS does not run one for service config |
| Need route simulation | Use `aps adapter messenger test <device>` for legacy device links only |

## Security

- Grant the bot minimum channel permissions.
- Use `aps serve --auth-token` for exposed routes.
- Keep the bot token in a secret-backed environment variable.
- Verify Discord Interaction signatures before exposing interaction routes.
- Validate all message content in profile actions.
