# APS Message Services Overview

Last updated: 2026-05-11

APS message services let profiles receive chat-like messages from Telegram,
Slack, Discord, SMS, and WhatsApp through one service webhook shape.

Use exact commands below. Replace IDs, profiles, numbers, and secret names with
operator-owned values.

## Quick Navigation

| Platform | Alias | Channel control | Current ingress | Signature validation |
| --- | --- | --- | --- | --- |
| Telegram | `telegram` | `--allowed-chat` | Bot API update JSON | Telegram secret-token header |
| Slack | `slack` | `--allowed-channel` | Events API JSON | Slack signing secret |
| Discord | `discord` | `--allowed-channel`, `--allowed-guild` | Message JSON or relay | Interactions Ed25519 only |
| SMS | `sms` | `--allowed-number` | Twilio/generic phone JSON or form | Twilio signature when provider is `twilio` |
| WhatsApp | `whatsapp` | `--allowed-number`, `--phone-number-id` | Cloud API JSON or Twilio-style form/JSON | Cloud `X-Hub-Signature-256`; Twilio signature when provider is `twilio` |

Ticket and work-item platforms use ticket services instead:

| Alias | Canonical service |
| --- | --- |
| `email` | `type: ticket`, `adapter: email` |
| `github` | `type: ticket`, `adapter: github` |
| `gitlab` | `type: ticket`, `adapter: gitlab` |
| `jira` | `type: ticket`, `adapter: jira` |
| `linear` | `type: ticket`, `adapter: linear` |

## Operator Model

`aps service add` stores APS routing and provider configuration. It does not
create platform apps, bot users, public tunnels, Discord Gateway clients,
Twilio numbers, Meta webhooks, or provider-side webhook registrations.

The platform or relay POSTs provider-shaped payloads to:

```text
POST /services/<service-id>/webhook
```

APS then validates provider auth when configured, normalizes the message,
checks allowed channel/chat/number controls, dispatches the profile action, and
returns provider-shaped status JSON.

## Common Setup

1. Create or choose an APS profile and action.
2. Create the provider app, bot, phone number, or relay outside APS.
3. Store provider secrets in the runtime environment or secret manager.
4. Add the APS service with `aps service add`.
5. Start the service endpoint with `aps service start` or shared `aps serve`.
6. Register the printed webhook URL in the provider console or relay.
7. Validate with `aps service status` and `aps service test`.

### Public Ingress

Provider webhooks must reach the APS process over public HTTPS. Common patterns:

- Reverse proxy or load balancer terminating TLS, forwarding to APS.
- Tunnel for local development, forwarding a temporary HTTPS URL to
  `127.0.0.1:8080`.
- Provider-specific relay that validates native signatures, then POSTs JSON.

Bind APS to a private interface behind the proxy when possible:

```bash
aps service start support-bot \
  --addr 127.0.0.1:8080 \
  --base-url https://hooks.example.com
```

`--base-url` reports the externally reachable URL. It does not configure DNS,
TLS, provider webhooks, or proxy routing.

Effective provider URL shape:

```text
https://hooks.example.com/services/<service-id>/webhook
```

Use the same value in the provider console, relay config, `--webhook-url` when
a provider signs the exact URL, and `aps service status --base-url` output.
`aps service status <id> --base-url https://hooks.example.com` should print
`webhook_url: https://hooks.example.com/services/<id>/webhook`; that is the
operator-facing public endpoint to register.

Local tunnel checklist:

1. Start APS on loopback, for example `aps serve --addr 127.0.0.1:8080`.
2. Start a tunnel that forwards public HTTPS traffic to
   `http://127.0.0.1:8080`.
3. Run `aps service status <id> --base-url <tunnel-https-origin>`.
4. Register the printed `webhook_url` with the provider.
5. Re-register when the tunnel origin changes.

Production reverse proxy checklist:

- Terminate TLS with a publicly trusted certificate.
- Forward to a private APS listener, usually `127.0.0.1:8080` or a private
  service address.
- Preserve method, path, query string, headers, and raw body. Slack, Twilio,
  and WhatsApp signatures can fail if a proxy rewrites signed inputs.
- Forward `Host` and `X-Forwarded-*` headers for logs and diagnostics.
- Keep request body limits and timeouts high enough for provider retry windows,
  but do not expose unrelated APS routes unless the deployment intends to.

### Verification Challenges

| Provider | Challenge flow | APS expectation |
| --- | --- | --- |
| Telegram | Register the HTTPS URL with `setWebhook`; Telegram sends updates with the optional secret-token header. | Configure `--webhook-secret-token-env` when using Telegram's secret token. |
| Slack | Events API sends a URL verification payload to the request URL. | APS acknowledges the challenge and validates signed event callbacks with the raw body. |
| Discord | Gateway message relays have no provider HTTP challenge. Interactions use Ed25519 verification separately. | Protect relays with proxy or relay auth unless using signed Interactions. |
| Twilio SMS/WhatsApp | Twilio signs each request against the exact configured URL and parameters. | Set `--webhook-url` to the same public URL registered in Twilio. |
| WhatsApp Cloud | Meta sends `GET /services/<id>/webhook?hub.mode=subscribe&hub.verify_token=...&hub.challenge=...`. | Configure `--verify-token-env`; APS echoes `hub.challenge` only when the token matches. |

### Service Lifecycle

```bash
aps service add <service-id> --type <message-alias> --profile <profile>
aps service show <service-id>
aps service routes <service-id>
aps service status <service-id> --base-url https://hooks.example.com
aps service test <service-id>
aps service test <service-id> --probe --base-url https://hooks.example.com
aps service start <service-id> --addr 127.0.0.1:8080 \
  --base-url https://hooks.example.com
aps service stop <service-id>
```

`aps service start` runs a foreground HTTP server. Stop it with interrupt or by
stopping the owning process manager. `aps service stop` prints that operational
contract; it does not kill a background daemon.

Shared server path:

```bash
aps serve --addr 127.0.0.1:8080
```

## Provider Setup

### Telegram

Provider app setup:

1. Create a bot with BotFather.
2. Save the bot token as `TELEGRAM_BOT_TOKEN`.
3. Generate a webhook secret token and save it as `TELEGRAM_WEBHOOK_SECRET`.
4. Add the bot to the chat, group, supergroup, or channel.
5. Find the numeric chat ID from `getUpdates` or a relay log.
6. Register the APS URL with Telegram's `setWebhook`.

APS setup:

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

Webhook URL:

```text
https://hooks.example.com/services/telegram-support/webhook
```

Signature validation:

- Header: `X-Telegram-Bot-Api-Secret-Token`.
- Configure with `--webhook-secret-token` or `--webhook-secret-token-env`.
- If neither option is set, APS warns and accepts unsigned Telegram requests.

Controls and limits:

- `--allowed-chat` matches normalized `chat.id`.
- Replies use Bot API `sendMessage`; bot token required for outbound delivery.
- APS supports webhook mode. It does not run Telegram polling for services.

### Slack

Provider app setup:

1. Create a Slack app.
2. Add a bot user and install the app to the workspace.
3. Save the bot token as `SLACK_BOT_TOKEN`.
4. Save the Events API signing secret as `SLACK_SIGNING_SECRET`.
5. Subscribe to message events or app mentions needed by the workflow.
6. Register the APS service URL as the Events API request URL.

APS setup:

```bash
aps service add slack-support \
  --type slack \
  --profile assistant \
  --allowed-channel C01ABC2DEF \
  --default-action handle-slack \
  --reply text \
  --require-bot-mention \
  --bot-user-id U012BOT \
  --env SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN \
  --env SLACK_SIGNING_SECRET=secret:SLACK_SIGNING_SECRET
```

Webhook URL:

```text
https://hooks.example.com/services/slack-support/webhook
```

Signature validation:

- Headers: `X-Slack-Signature`, `X-Slack-Request-Timestamp`.
- APS validates Slack `v0` HMAC over `v0:<timestamp>:<body>`.
- Default timestamp tolerance is five minutes.
- URL verification challenges are acknowledged.

Controls and limits:

- `--allowed-channel` matches Slack channel IDs.
- `--require-bot-mention` blocks non-DM messages unless the event is
  `app_mention` or text contains `<@BOT_USER_ID>`.
- Slack bot messages are ignored and duplicate `event_id` deliveries are
  deduplicated.
- Replies use `chat.postMessage`; file upload is not the primary delivery path.

### Discord

Provider app setup:

1. Create a Discord application and bot.
2. Add the bot to the guild with minimum channel permissions.
3. Save the bot token as `DISCORD_BOT_TOKEN`.
4. Run a Discord Gateway client or relay that POSTs message-create JSON to APS.
5. Copy guild and channel IDs with Discord Developer Mode enabled.

APS setup:

```bash
aps service add discord-support \
  --type discord \
  --profile assistant \
  --allowed-guild 987654321012345678 \
  --allowed-channel 123456789012345678 \
  --default-action handle-discord \
  --reply text \
  --env DISCORD_BOT_TOKEN=secret:DISCORD_BOT_TOKEN
```

Webhook URL:

```text
https://hooks.example.com/services/discord-support/webhook
```

Signature validation:

- Discord Gateway message relays do not carry a provider HTTP signature.
- Protect the APS route with private ingress, proxy auth, or relay-side auth.
- Discord Interactions use `X-Signature-Ed25519` and
  `X-Signature-Timestamp`; APS has a lower-level validator for signed
  interaction bodies, but service validation currently treats message-service
  receive modes as `webhook` or `polling`.

Controls and limits:

- `--allowed-channel` matches Discord `channel_id`.
- `--allowed-guild` matches Discord `guild_id`.
- APS does not run a Discord Gateway client from `aps service add`.
- Replies use Discord channel messages API with bot token.
- Attachments are carried as URLs/embeds; direct file upload is not attempted.

### SMS

Provider app setup:

1. Buy or assign a provider phone number.
2. For Twilio, save `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN`.
3. Configure the phone number messaging webhook to the APS service URL.
4. Set the exact public URL in APS with `--webhook-url` for Twilio validation.
5. Add allowed sender numbers before exposing the route.

APS setup:

```bash
aps service add sms-alerts \
  --type sms \
  --profile assistant \
  --provider twilio \
  --from +15559870002 \
  --webhook-url https://hooks.example.com/services/sms-alerts/webhook \
  --allowed-number +15551230001 \
  --default-action handle-sms \
  --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

Webhook URL:

```text
https://hooks.example.com/services/sms-alerts/webhook
```

Signature validation:

- Twilio provider uses `X-Twilio-Signature`.
- APS validates the signature from the exact public `--webhook-url` and form
  parameters. Keep scheme, host, path, and query identical to provider config.
- JSON requests with `bodySHA256` in the URL are body-hash checked.

Controls and limits:

- `--allowed-number` matches sender, sender platform ID, channel, or channel
  platform ID. Use E.164 format consistently.
- Omitted `--allowed-number` leaves inbound sender routing open and emits a
  warning.
- Twilio form posts are accepted; generic providers should POST JSON.
- SMS has no native thread; APS separates sessions by sender and receiving
  number.

### WhatsApp

Provider app setup:

1. Choose Meta WhatsApp Cloud API or a Twilio WhatsApp relay.
2. For Cloud API, save `WHATSAPP_ACCESS_TOKEN`, `WHATSAPP_VERIFY_TOKEN`,
   and `WHATSAPP_APP_SECRET`.
3. For Twilio WhatsApp, save `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN`.
4. Configure the provider webhook or relay to reach the APS service URL.
5. Set `--phone-number-id` for Cloud API or `--from` for Twilio-style numbers.
6. Add allowed sender numbers before exposing the route.

Cloud API APS setup:

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

Twilio-style APS setup:

```bash
aps service add wa-twilio \
  --type whatsapp \
  --profile assistant \
  --provider twilio \
  --from whatsapp:+15559870002 \
  --webhook-url https://hooks.example.com/services/wa-twilio/webhook \
  --allowed-number whatsapp:+15551230001 \
  --default-action handle-whatsapp \
  --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

Webhook URL:

```text
https://hooks.example.com/services/wa-support/webhook
```

Signature validation:

- Cloud API webhook verification uses `GET /services/<id>/webhook` with
  `hub.mode=subscribe`, `hub.verify_token`, and `hub.challenge`. APS validates
  the verify token and echoes the challenge.
- Cloud API POST signatures use `X-Hub-Signature-256: sha256=<hex>`.
  Configure the app secret with `--signing-secret-env`, `--env
  WHATSAPP_APP_SECRET=secret:...`, or the corresponding options.
- Twilio-style WhatsApp uses the same `X-Twilio-Signature` validation as SMS
  when `--provider twilio` is set.

Controls and limits:

- `--allowed-number` matches sender or receiving number values.
- Cloud API normalizes `phone_number_id` as channel identity and rejects
  messages for another configured phone number ID.
- Twilio-style relays normalize `From`, `To`, and `Body`.
- Outbound Cloud delivery sends text replies or configured templates through
  `/{phone_number_id}/messages`. Set `--template-required` and
  `--template-name` when replies must be template-only.
- Twilio compatibility sends WhatsApp messages through Twilio's Messages API
  with `whatsapp:` sender/recipient prefixes.

## Normalized Message Format

Profile actions receive normalized message JSON. Example shape:

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

Conversation and session keys are derived from service, platform, workspace,
channel, sender, and thread fields. Full policy:
[Message conversation and thread policy](dev/message-conversation-policy.md).

## Validation And Routing

Allowed-source checks run after normalization:

| Option | Applies to | Match fields |
| --- | --- | --- |
| `--allowed-chat` | Telegram | `channel.id`, `channel.platform_id` |
| `--allowed-channel` | Slack, Discord | `channel.id`, `channel.platform_id` |
| `--allowed-guild` | Discord | `workspace_id` |
| `--allowed-number` | SMS, WhatsApp | `sender.id`, `sender.platform_id`, `channel.id`, `channel.platform_id` |

Routing currently uses one service-level `--default-action`. Use separate
services for simple channel separation. Use legacy adapter-device links only
when one subprocess device needs many channel-to-action mappings.

## Troubleshooting

| Problem | Check |
| --- | --- |
| Alias resolved to ticket | Use `telegram`, `slack`, `discord`, `sms`, or `whatsapp` for message services. |
| Service invalid | Run `aps service test <id>` and fix `config_issue` output. |
| Route unknown | Run `aps service routes <id>` and `aps service status <id> --base-url <url>`. |
| Platform cannot connect | Check public HTTPS, DNS, tunnel, proxy, firewall, and APS bind address. |
| Slack request rejected | Check timestamp skew, signing secret, and raw-body preservation through proxy. |
| Telegram request rejected | Check `X-Telegram-Bot-Api-Secret-Token` and secret env value. |
| Twilio request rejected | Check exact `--webhook-url`, request params, auth token, and body hash. |
| Message blocked | Check allowed channel/chat/guild/number options and normalized IDs. |
| Action did not run | Check service `--default-action` and profile action name. |
| Reply failed | Check outbound provider token, bot permissions, channel membership, and rate limits. |
| Need polling or Gateway runtime | Run a provider relay or legacy adapter device; service config is not that runtime. |

## Legacy Adapter Devices

Legacy messenger-device commands remain for subprocess devices and existing link
stores:

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

Prefer first-class `aps service` commands for new Telegram, Slack, Discord, SMS,
and WhatsApp routes.
