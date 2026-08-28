# Messenger Integration

APS message services receive chat-like platform payloads and route them to
profile actions.

## Supported Message Adapters

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/messengermd", "user-support"],
    text=True))
]]] -->
| Adapter alias | Channel ID format | Typical token source | Current support |
| --- | --- | --- | --- |
| `discord` | Numeric channel ID (e.g., 1234567890123456789) | Discord Developer Portal | JSON webhook route through `aps serve` |
| `--type message --adapter email` | Mailbox name or email address (e.g., inbox, work@co.com) | your email bridge (IMAP poller, MTA hook) | JSON relay route through `aps serve` |
| `slack` | Alphanumeric channel ID (e.g., C01ABC2DEF) | Slack API dashboard | JSON webhook route through `aps serve` |
| `sms` | Phone number receiving SMS (e.g., +15551234567) | SMS provider such as Twilio | JSON relay route through `aps serve` |
| `teams` | Bot Framework conversation ID (e.g., 19:abc123@thread.tacv2) | Azure Bot registration (App ID + client secret) | JSON webhook route through `aps serve` |
| `telegram` | Numeric chat ID (e.g., -1001234567890) | BotFather | JSON webhook route through `aps serve` |
| `whatsapp` | WhatsApp phone number ID or receiving number (e.g., 123456789012345) | WhatsApp Cloud API or Twilio | JSON webhook/relay route through `aps serve` |
<!-- [[[end]]] -->

`github`, `gitlab`, `jira`, `linear`, and `email` are ticket service aliases,
not message aliases; they mount at `/services/<id>/ticket/<adapter>` — see
[Ticket services](tickets.md). The email message adapter is addressed in
canonical form.

## Credential Bindings

`--env KEY=VALUE` binds a credential. `VALUE` is either a literal or a
reference:

| Form | Meaning |
| --- | --- |
| `--env SLACK_BOT_TOKEN=xoxb-abc123` | Literal. Stored verbatim in the service config. |
| `--env SLACK_BOT_TOKEN=secret:slack_bot` | Reference to the secret named `slack_bot`. |

A `secret:NAME` reference is resolved when the credential is used, in order:

1. the profile secret store, whose backend is set by `secrets.backend` in
   config (see below);
2. the process environment variable `NAME`.

### Secret Store Backends

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/configmd", "secret-backends"],
    text=True))
]]] -->
| `secrets.backend` | Where secrets live | Required config |
| --- | --- | --- |
| `file` (default) | the profile's `secrets.env`, mode 0600 | — |
| `env` | `APS_SECRET_<NAME>` in the environment | `prefix` to override `APS_SECRET_` |
| `keyring` | OS keychain | `service` (defaults to `aps/<profile>`) |
| `onepassword` | 1Password, via the `op` CLI or Connect | `vault`; plus `connect_url` + `token` for Connect |
| `openbao` † | OpenBao / Vault KV v2 | `addr`, `token`; `mount` defaults to `secret` |
| `infisical` | Infisical | `addr`, `project`, `env`, `token` |
| `ghsecrets` | GitHub Actions secrets | `repo` (defaults to the current repo) |
<!-- [[[end]]] -->

† `openbao` is opt-in: it pulls the Vault API client and ~17 transitive
modules, so the stock `aps` binary omits it entirely. Selecting it in a build
that lacks it reports how to enable it rather than failing obscurely.

Releases ship a separate `aps-vault` archive with the backend compiled in —
same CLI, same version, plus `openbao`. Download that instead of `aps`, or
build from source:

```bash
go build -tags openbao ./cmd/aps
```

Backend credentials should not be written into the config file: set
`token_env` to the name of an environment variable holding the token instead,
and it is read at open time.

```yaml
secrets:
  backend: onepassword
  vault: Engineering
```

If neither resolves, the credential is empty and the request fails. It does
**not** fall back to an environment variable named after the binding key —
a reference resolves only the name it declares, so a missing secret can never
silently authenticate with an unrelated ambient variable.

Prefer `secret:NAME` over literals: literals are written into the service
config file in plaintext.

### When The Store Is Unavailable

The store is read once per profile and the result held for the lifetime of the
process, so an inbound message does not re-read the backend — a `keyring`
profile prompts the OS keychain once, not once per message. To pick up a
secret you have just changed, restart the service.

A **failed** read is not cached. If the backend is unreachable at the first
lookup — vault down, keyring locked, `secrets.env` not yet written — the next
lookup retries, so a service recovers on its own once the store comes back
without needing a restart. While the store is down, `secret:` references
resolve to nothing and requests fail rather than falling back.

Each failing profile logs at most one warning per minute:

```
WARN reading profile secrets failed; will retry profile=my-agent error=...
```

A single line does not mean a single failed request — it is throttled. Once
the store recovers, a later outage warns again immediately.

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
  --profile my-agent \
  --default-action handle-telegram \
  --env TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN
```

`aps service add` validates the assembled config before anything is written.
An invalid config (for example `--type sms` without `--provider`/`--from`)
prints `config_valid: false` plus each `config_issue:` line, exits 1, and
leaves nothing on disk -- no service file, no webhook route. Re-adding an ID
that already exists is refused; pass `--force` to overwrite the stored record
deliberately. `--dry-run` prints the same resolution and validation report
without writing.

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

### Teams

```bash
aps service add teams-chat \
  --type teams \
  --profile assistant \
  --allowed-channel "19:abc123@thread.tacv2" \
  --default-action triage \
  --reply text \
  --option tenant_id=f8cdef31-a31e-4b4a-93e4-5f571e91255a \
  --env TEAMS_APP_ID=secret:TEAMS_APP_ID \
  --env TEAMS_APP_PASSWORD=secret:TEAMS_APP_PASSWORD
```

Inbound requests are authenticated by the Microsoft-signed Bot Framework JWT
in the `Authorization` header; there is no signing secret to configure.
`tenant_id` selects the single-tenant token endpoint for outbound replies.
`aps service test --probe` cannot mint that JWT, so verify Teams services
with a real message from the installed app.

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

### Email

```bash
aps service add mail-inbox \
  --type message \
  --adapter email \
  --profile assistant \
  --allowed-sender alice@example.com \
  --allowed-sender '*@partner.org' \
  --default-action handle-email \
  --reply text \
  --auth-scheme bearer \
  --auth-token-env MAIL_BRIDGE_TOKEN
```

An email bridge POSTs `{"from","to","subject","body"}` JSON to the service
URL. `--allowed-sender` accepts exact addresses or `*@domain` globs,
case-insensitive; with none set any sender routes (validation warns). The
bridge authenticates with the generic webhook auth flags (`--auth-scheme`
plus `--auth-token-env` or `--signature-secret-env`); without them the config
is valid but validation warns that the route is open. Details:
[Email](../MESSENGERS_OVERVIEW.md#email) and
[Generic webhook auth](../MESSENGERS_OVERVIEW.md#generic-webhook-auth).

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
service and is not mounted at `/services/<id>/webhook`; `aps serve` mounts it
at `/services/jira-intake/ticket/jira` instead. Setup, auth, payloads, and the
email adapter: [Ticket services](tickets.md).

## Routing Many Organizations On One Number

One service has one `--default-action`. To dispatch by sender instead, give the
service a route table and (optionally) a contacts snapshot:

```bash
aps service add support-line \
  --type whatsapp \
  --profile triage \
  --provider twilio \
  --from whatsapp:+15550100002 \
  --route-table routes/support-line.yaml \
  --contacts contacts/support-line.yaml \
  --env TWILIO_ACCOUNT_SID=secret:twilio_sid \
  --env TWILIO_AUTH_TOKEN=secret:twilio_token
```

Routes match the normalized sender (Twilio `whatsapp:+1555...` and WhatsApp
Cloud `1555...` compare equal), a contact's `org:`, or a `contact:` id, first
match wins in file order, and the last route must be `match: unknown` so
unknown senders always land somewhere (typically triage). Schema and rules:
[Message routing](../dev/message-routing.md).

## Generic Webhook Auth

Providers with a native signature (Slack, Telegram, Twilio, WhatsApp Cloud,
Discord interactions) are validated by their provider hook. For everything
else -- the email bridge, an SMS/WhatsApp `--provider generic` relay, or any
provider you want to wrap behind your own HMAC -- set the generic scheme on
`service add`:

| Flag | Option written | Meaning |
| --- | --- | --- |
| `--auth-scheme` | `auth_scheme` | `bearer`, `token`, `hmac-sha256`, `ed25519`, or `slack-signing-secret` |
| `--auth-token-env` | `auth_token_env` | env var holding the bearer/token secret (`Authorization: Bearer ...` or `X-APS-Token`) |
| `--signature-secret-env` | `signature_secret_env` | env var holding the HMAC secret (`X-APS-Signature: sha256=<hex>`) or Ed25519 public key |
| `--option KEY=VALUE` | any | escape hatch for options without a flag, e.g. `timestamp_header`, `require_replay_check=true`, `auth_header`; repeatable; a named flag wins over `--option` on the same key |

Secrets never go on the command line: the flags name environment variables.
The literal `auth_token` / `signature_secret` options remain yaml-only.
`--signing-secret-env` is different: it feeds the Slack and WhatsApp
provider-native signature checks, not generic auth.

`aps service show <id>` prints the effective result under `auth:` (scheme,
header, env names, timestamp/replay headers) or `auth: none`.

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

## Conversation History

Every routed inbound message and every delivered reply is recorded as a turn,
keyed by the conversation identity (service, platform, channel, sender, and
platform thread). The routed action receives the newest turns of the same
session as `prior_turns` on stdin (default 20; set per service with
`--history-turns N`), plus a `conversation` object with the identity keys.

Inspect what a profile has seen:

```bash
aps service conversation list --service support-bot
aps service conversation show <conversation-id>
aps service conversation show <conversation-id> --limit 5 --format json
```

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
| `service config is invalid` on add | Nothing was saved; fix each `config_issue:` line and re-run |
| `already exists` on add | Re-run with `--force` to overwrite the stored service deliberately |
| Message not routed | Confirm `--default-action`, `--route-table`, or legacy channel mapping matches the incoming channel/sender |
| Route table rejected | `aps service show <id>` prints `routing_error:`; every table must end with `match: unknown` |
| Platform cannot reach APS | Check tunnel, DNS, auth token, and `aps serve --addr` binding |
| SMS provider posts forms | Add a relay that converts form fields to JSON before POSTing to APS |

## Security

- Use `aps serve --auth-token` for exposed routes.
- Keep tokens in secret-backed environment variables.
- Treat incoming message text, metadata, and attachments as untrusted input.
- Avoid logging sensitive message content.
- Grant platform apps the minimum permissions needed.
