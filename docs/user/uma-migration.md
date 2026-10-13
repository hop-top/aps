# Move SMS and WhatsApp Services to uma

> Availability: unreleased. Requires a build newer than aps/v0.6.0-alpha.0.

Use this when a message service runs on the aps-native SMS or WhatsApp
transport (`--type sms`, `--type whatsapp`) or is the email relay, and you want
it hosted by an in-process uma engine instead. Migration is opt-in per service. The native
transports stay the default and keep working; nothing here sets a removal date.

## Quick path

Find the row for your service.

| Native service | Becomes | Moves today |
| --- | --- | --- |
| `--type sms --provider twilio` | `--type uma --uma-adapter twilio.sms` | yes |
| `--type whatsapp --provider twilio` | `--type uma --uma-adapter twilio.whatsapp` | yes |
| `--type whatsapp --provider whatsapp-cloud` | `--type uma --uma-adapter meta.whatsapp` | yes, no templates |
| `--provider generic` (SMS, WhatsApp) | `--type uma --uma-adapter relay.messages` | yes; your gateway serves a [callback](#generic-relay-relaymessages) |
| Email relay (`--adapter email`) | `--type uma --uma-adapter http.mail` | yes, [receive only](#email-relay-httpmail): no reply |

1. Re-add the service under the **same id** with `--force`. The id is the URL:
   the route stays `/services/<id>/webhook`, so the provider console keeps its
   callback URL.
2. Apply the per-transport changes below (a flag or two, one secret binding).
3. Restart `aps serve`; a service added or changed while it runs is picked up
   on its next webhook.
4. [Verify](#verify), and keep the old command at hand for [rollback](#roll-back).

Every `uma` service shares these changes:

- The provider signature is checked by uma, once. aps runs no signature check
  of its own for the service.
- A reply is queued and sent by uma afterwards. `delivery_status: accepted`
  means queued; a later provider failure is not reported back to the action.
- A sender your allowlist refuses is answered `200` (uma already recorded the
  event); native answers `403`. The refusal is in the server log.
- `aps service show` prints `auth: none` for a Twilio or Meta `uma` service.
  That is aps's own check, not the provider's; uma still verifies the
  signature. For `relay.messages` and `http.mail` it prints the webhook auth uma
  enforces.
- `--provider`, `--verify-token-env` and `--signing-secret-env` are not read.
  Bindings come from `--env` only. `service add` warns for each such flag; see
  [Flags that do nothing](#flags-that-do-nothing-on-a-uma-service).
- `aps service test --probe` does not apply; see [Verify](#verify).

## SMS over Twilio

Before:

```bash
aps service add sms-main \
  --type sms --profile assistant --provider twilio \
  --from +15559870002 \
  --allowed-number +15551230001 \
  --default-action handle-sms --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

After:

```bash
aps service add sms-main --force \
  --type uma --uma-adapter twilio.sms --profile assistant \
  --from +15559870002 \
  --webhook-url https://hooks.example.com/services/sms-main/webhook \
  --allowed-number +15551230001 \
  --default-action handle-sms --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

Config: swap `--type sms` for `--type uma --uma-adapter twilio.sms`, drop
`--provider`, add `--webhook-url`. Same id, same bindings.

What you will see:

- **`--webhook-url` is mandatory and must have no query or fragment.** uma signs
  against exactly this URL, where native rebuilds it from forwarded headers. It
  must equal the URL in the Twilio console, character for character; a
  mismatch makes every request `403`. `service add` refuses a service without
  it, or with a query.
- **No inline TwiML reply.** Twilio gets an empty TwiML answer at once; the
  reply goes out afterwards as a Messages API call. It needs the REST
  credential (`TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`) and `--from`. The
  action no longer has to finish inside Twilio's webhook timeout.
- **Bad signature is `403`**, native answers `401`.
- **MMS media lands on disk.** An inbound MMS reaches the action as
  `attachments` with `file://` URLs under `<aps data dir>/messenger/uma/<id>/`,
  owner-only, never deleted by aps. Native passed Twilio's `MediaUrl` through.
- **Delivery status callbacks** go to the same `--webhook-url`.
- **JSON-bodied Twilio webhooks** (a `bodySHA256` in the URL) get `400` from
  this build's uma service. Native refuses them too (`401`), so nothing
  that worked stops working.
- `--allowed-number` keeps working, with `+` numbers as before.

## WhatsApp over Twilio

Before:

```bash
aps service add wa-tw \
  --type whatsapp --profile assistant --provider twilio \
  --from whatsapp:+15559870002 \
  --webhook-url https://hooks.example.com/services/wa-tw/webhook \
  --allowed-number whatsapp:+15551230001 \
  --default-action handle-whatsapp --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

After:

```bash
aps service add wa-tw --force \
  --type uma --uma-adapter twilio.whatsapp --profile assistant \
  --from +15559870002 \
  --webhook-url https://hooks.example.com/services/wa-tw/webhook \
  --allowed-number whatsapp:+15551230001 \
  --default-action handle-whatsapp --reply text \
  --env TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID \
  --env TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN
```

Config: swap the type, drop `--provider`, and write `--from` **without** the
`whatsapp:` prefix (`service add` refuses it). Keep the prefix on
`--allowed-number`, as natively.

What you will see: everything from [SMS over Twilio](#sms-over-twilio) that
applies (mandatory `--webhook-url`, asynchronous replies, `403` on a bad
signature), plus:

- Replies are free-form text. uma can send Twilio Content Templates
  (`content_sid`); the aps bridge only sends text, so `content_sid` is
  unreachable here, as it is natively. `--template-name`, `--language-code`
  and `--template-required` are accepted and ignored; `service add` warns.
- Inbound media is not staged by this build's uma for WhatsApp; the action sees
  the text only.

## WhatsApp Cloud (Meta)

Before:

```bash
aps service add wa-cloud \
  --type whatsapp --profile assistant --provider whatsapp-cloud \
  --phone-number-id 123456789012345 \
  --allowed-number 15551230001 \
  --verify-token-env WHATSAPP_VERIFY_TOKEN \
  --signing-secret-env WHATSAPP_APP_SECRET \
  --default-action handle-whatsapp --reply text \
  --env WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN \
  --env WHATSAPP_VERIFY_TOKEN=secret:WHATSAPP_VERIFY_TOKEN \
  --env WHATSAPP_APP_SECRET=secret:WHATSAPP_APP_SECRET
```

After:

```bash
aps service add wa-cloud --force \
  --type uma --uma-adapter meta.whatsapp --profile assistant \
  --phone-number-id 123456789012345 \
  --allowed-number 15551230001 \
  --default-action handle-whatsapp --reply text \
  --env WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN \
  --env WHATSAPP_VERIFY_TOKEN=secret:WHATSAPP_VERIFY_TOKEN \
  --env WHATSAPP_APP_SECRET=secret:WHATSAPP_APP_SECRET
```

Config: swap the type, drop `--provider`, `--verify-token-env` and
`--signing-secret-env`. The three bindings stay; they are the only source.
Cloud sender ids are digits, no `+`, so `--allowed-number` takes `15551230001`.

What you will see:

- **The app secret is mandatory.** Native accepts unsigned Cloud webhooks when
  none is set (a warning). uma answers `401` without a valid
  `X-Hub-Signature-256`, and `service add` refuses a service without
  `WHATSAPP_APP_SECRET`.
- **Meta's GET verification works at the service URL.** Native answers `405` to
  it through `aps serve`. Keep the callback URL and the verify token at Meta.
- **Status-only notifications** (delivered, read) are answered `200`. Native
  answers `400`, so Meta retried them.
- **Do not migrate a service that depends on templates.** `--template-name`,
  `--language-code` and `--template-required` are accepted and ignored: the
  bridge sends text only, and `service add` warns about each. Outside the
  24-hour window Meta refuses free-form text.
- **Replies are asynchronous**, as for Twilio; `delivery_status: accepted` is
  queued.

## Flags that do nothing on a uma service

`aps service add --type uma` takes every flag a message service takes, but the
bridge reads only some of them. A flag it ignores is not refused: `service add`
prints one `warning:` line per flag, saves the service and exits `0`. The line
names the flag and what to use instead; it never prints the value.

```text
warning: --provider has no effect on a uma service: pick the provider with --uma-adapter
```

| Flag | Why it does nothing | Use instead |
| --- | --- | --- |
| `--template-name`, `--language-code`, `--template-required` | The bridge sends text only; no template is ever sent | Nothing. Keep a service that needs templates native |
| `--verify-token`, `--verify-token-env` | The verify token is read from its binding only | `--env WHATSAPP_VERIFY_TOKEN=secret:NAME` (`meta.whatsapp`) |
| `--signing-secret-env` | Secrets are read from bindings only | `--env WHATSAPP_APP_SECRET=secret:NAME` (`meta.whatsapp`), `--env TWILIO_AUTH_TOKEN=secret:NAME` (Twilio) |
| Generic webhook auth flags and options (below) on Twilio and Meta adapters | Generic webhook auth does not run there; uma checks the provider signature | Nothing. `relay.messages` and `http.mail` read them |
| `--provider` | The uma adapter picks the provider | `--uma-adapter` |
| `--receive polling` | A uma service receives by webhook only | Omit it, or `--receive webhook` |
| `--phone-number-id` on `twilio.sms`, `twilio.whatsapp` | Only `meta.whatsapp` reads it | Omit it |
| `--from`, `--webhook-url` on `meta.whatsapp` | Not read; the callback URL is set at Meta | Omit them |
| `--webhook-url` on `relay.messages`, `http.mail`; `--from` on `http.mail` | They check webhook auth, not a signed URL; mail is accepted for `recipients` | Omit them |
| `--option` `medium`, `callback_url`, `callback_header` | Only `relay.messages` reads them | Omit them elsewhere |
| `--option` `recipients`, `formats`, `recipient_header`, `trusted_authserv_id`, `authentication_results_field` | Only `http.mail` reads them | Omit them elsewhere |
| `--reply` other than `none` on `http.mail`, or unset | `http.mail` receives only: a reply is refused and logged, never sent | `--reply none` |
| `--webhook-secret-token`, `--webhook-secret-token-env`, `--bot-user-id`, `--require-bot-mention`, `--dedup-ttl` | Telegram and Slack options | Omit them |
| `--site`, `--project`, `--jql`, `--linear-workspace`, `--team`, `--group`, `--events` | Ticket options | Omit them |

The generic webhook auth flags and options are `--auth-scheme`,
`--auth-token-env`, `--signature-secret-env`, and the `--option` keys
`auth_header`, `timestamp_header`, `require_timestamp`, `timestamp_tolerance`,
`replay_id_header`, `require_replay_check`, `auth_payload`,
`auth_allow_unsigned` and `auth_min_secret_bytes`.

A key set with `--option` that falls in the table (`template_name`,
`auth_token`, ...) warns as `--option <key>`; an unset `--reply` on `http.mail`
warns as `the default --reply text`. These flags are read: `--type`,
`--uma-adapter`, `--profile`, `--from` and `--webhook-url` (Twilio),
`--phone-number-id` (`meta.whatsapp`), `--env` bindings, `--allowed-number`,
`--allowed-sender`, `--default-action`, `--route-table`, `--contacts`, `--reply`,
`--history-turns`, `--description` and `--label`. A flag left unset, or a
boolean left `false`, never warns.

The warning goes to stderr. It is not printed with `--format json`,
`--format yaml` or `--quiet`, so a script that parses the command's output never
sees it. Unlike the notice below, it is printed when output is piped, and
`--no-hints` and `HOP_QUIET_HINTS=1` do not silence it: a flag with no effect is
a mistake in the command, not a hint.

## Generic relay (`relay.messages`)

The owner accepted on 2026-10-13 that replies leave by a callback your gateway
serves, never in the webhook answer. Inbound keeps the payload and the auth
headers your gateway sends today.

Before:

```bash
aps service add sms-relay \
  --type sms --profile assistant --provider generic \
  --from +15559870002 \
  --allowed-number +15551230001 \
  --default-action handle-sms --reply text \
  --auth-scheme hmac-sha256 --signature-secret-env RELAY_SIGNING_SECRET \
  --option require_timestamp=true --option require_replay_check=true
```

After, allow the callback host once in the global configuration
(`~/.config/aps/config.yaml`; none is allowed by default):

```yaml
uma:
  relay:
    callback_hosts: [gateway.example.com]
```

```bash
aps service add sms-relay --force \
  --type uma --uma-adapter relay.messages --profile assistant \
  --option medium=sms --from +15559870002 \
  --option callback_url=https://gateway.example.com/aps/reply \
  --allowed-number +15551230001 \
  --default-action handle-sms --reply text \
  --auth-scheme hmac-sha256 --signature-secret-env RELAY_SIGNING_SECRET \
  --option require_timestamp=true --option require_replay_check=true \
  --option auth_allow_unsigned=true \
  --env RELAY_SIGNING_SECRET=secret:RELAY_SIGNING_SECRET \
  --env RELAY_CALLBACK_TOKEN=secret:RELAY_CALLBACK_TOKEN
```

Config: swap the type, drop `--provider`, add `--option medium` (`sms` or
`whatsapp`: the type no longer says it), `--option callback_url` and two
bindings. Keep every auth flag and option as it was.

What you will see:

- **Replies are a `POST` to your callback.** JSON `medium`, `to`, `body`,
  `in_reply_to` (your message id), `operation_id`, `attempt_id`, with
  `Authorization: Bearer <RELAY_CALLBACK_TOKEN>` (or the header named by
  `--option callback_header`) and `Idempotency-Key: <operation_id>`, the same on
  every attempt. Answer `2xx`, optionally `{"message_id": "..."}`. The webhook
  answer is `{"status":"accepted"}` and carries no reply. Redirects are not
  followed. A reply whose answer was lost (timeout, `5xx`, `408`, `409`) is not
  sent again; look the key up in your gateway's log.
- **The callback host must be listed** in `uma.relay.callback_hosts` (`host` or
  `host:port`, exactly as in the URL) and the URL must be `https`, with no
  credentials, query or fragment. `service add` refuses it otherwise.
  `uma.relay.allow_http: true` permits `http`, for local development only.
- **The secret is a binding.** Natively `--signature-secret-env NAME` (or
  `--auth-token-env`) reads the process variable; under uma bind it with
  `--env NAME=secret:NAME`. `service add` refuses a service without auth.
- **Unsigned timestamp and replay id need consent.** aps's HMAC signs the body
  only, so `X-APS-Timestamp` and `X-APS-Delivery-ID` are outside the signature.
  `--option auth_allow_unsigned=true` accepts that, as aps did; once your
  gateway signs them, use `--option auth_payload={timestamp}.{body}` instead.
  `service add` refuses the service with neither.
- **Secrets under 16 bytes need `--option auth_min_secret_bytes=N`.** Without
  it the route answers `503` and the server log names the option.
- **Bad auth is `401`** (wrong or missing signature, stale timestamp), a
  repeated delivery id `409`, a body over 64 KiB `413`.
- **JSON only.** Form bodies are refused, and flat `NumMedia`/`MediaUrlN` keys
  are not read: send media as `"media": ["https://..."]`. Media URLs reach the
  action as attachments, never fetched. `From`/`from`, `To`/`to`,
  `Body`/`body`/`text` and `MessageSid`/`SmsSid`/`message_id` are read as
  before; a message without an id gets a generated one, and `WaId` is not used
  as an id.
- **`--from` must be the number your gateway sends as `to`.** uma routes
  inbound by it.
- **Replay protection is in memory**, as natively: a restart or a configuration
  change forgets it. An answered message replayed after a restart is still not
  run again: its reply is in history.

## Email relay (`http.mail`)

Receive only, by the owner's decision of 2026-10-13. The native email relay
returns the action's reply as JSON in the webhook answer, for your bridge to
send. The uma path does not: the answer is `{"accepted": 1}` and the reply is
refused. **If your bridge sends that reply, stay on the native email relay for
now.**

Before:

```bash
aps service add mail-inbox \
  --type message --adapter email --profile assistant \
  --allowed-sender '*@partner.org' \
  --default-action handle-email --reply text \
  --auth-scheme bearer --auth-token-env MAIL_BRIDGE_TOKEN
```

After:

```bash
aps service add mail-inbox --force \
  --type uma --uma-adapter http.mail --profile assistant \
  --option recipients=support@example.com \
  --allowed-sender '*@partner.org' \
  --default-action handle-email --reply none \
  --auth-scheme bearer --auth-token-env MAIL_BRIDGE_TOKEN \
  --env MAIL_BRIDGE_TOKEN=secret:MAIL_BRIDGE_TOKEN
```

Config: swap the type, add `--option recipients` (the addresses you accept mail
for) and the binding, set `--reply none`.

What you will see:

- **No reply, ever.** With a reply mode other than `none` the action still runs;
  its reply is refused, logged on the server (`uma reply not sent`) and shown as
  `reply_disabled` in `aps service status`. No outbound turn is written.
  `service add` warns once, `aps service show` and `routes` say replies are
  disabled.
- **Authentication is mandatory.** Natively a missing auth only warned.
- **Mail to an address not in `recipients` is refused (`422`).** A message to
  two listed addresses reaches the action once, on the first.
- **JSON as before, and raw messages.** The `from`, `to`, `subject`, `body`
  and `attachments` members are read as before. With
  `--option formats=json,rfc822` the relay may post the message itself as
  `message/rfc822`. Form bodies (SendGrid Inbound Parse, the default Mailgun
  route) are refused, as natively.
- **Attachment bytes land on disk.** An attachment with base64 `content` (or in
  a raw message) reaches the action as a `file://` URL under
  `<aps data dir>/messenger/uma/<service-id>/`. One with only a `url` is passed
  through, never fetched, as natively.
- **The sender is unverified** (`platform_metadata.sender_assurance`), unless
  `--option trusted_authserv_id` names the relay that stamps
  `Authentication-Results` (`--option authentication_results_field` for JSON).
- Sessions keep the native keys: the recipient is the channel and the subject
  the thread.

Details: [design](../dev/uma-relay-mail-design.md),
[parity checklist](../dev/requirements/uma-bridge-parity-checklist.md) notes M10
and M11.

## Verify

After the restart:

```bash
aps service show sms-main
aps service routes sms-main
aps service test sms-main
```

Expect `adapter: uma`, `receives: HTTP POST /services/sms-main/webhook (uma
verifies the provider signature)`, `maturity: ready`, the same route as before,
and `config_valid: true`.

Probe the live route (`aps serve --addr 127.0.0.1:8080`):

```bash
curl -i -X POST http://127.0.0.1:8080/services/sms-main/webhook -d Body=hi
curl -i "http://127.0.0.1:8080/services/wa-cloud/webhook\
?hub.mode=subscribe&hub.verify_token=$WHATSAPP_VERIFY_TOKEN&hub.challenge=1234"
```

- Unsigned Twilio POST: `403`, the action never ran. Unsigned Cloud POST: `401`.
- Cloud verification with the right token: `200` and the challenge as body;
  a wrong token: `403`.

Then send a real message. `aps service status sms-main` shows `last_inbound`,
`last_outbound` and `delivery_status: accepted` once the reply is queued.

## Roll back

Switch the service back by re-adding it under the same id with the original
native command (the "Before" blocks above) and `--force`, then restart
`aps serve`. The route, the provider console and the profile secrets do not
change. A restart or a rebuild drops uma's in-memory state: replies not yet
sent are lost, and the provider's own retries are not recognised as duplicates.

## Notice on native services

On a terminal, `aps service add`, `show` and `status` print one line after
their output for a native service of the three Twilio and Meta shapes, pointing
here. `--provider generic` and email services get none: their move needs a
change outside aps (a callback; the loss of the email reply). It is a next-step hint:
not printed when output is piped, with `--format json` or `--format yaml`, with
`--quiet`, with `--no-hints`, or with `HOP_QUIET_HINTS=1`. It never changes an
exit code.

## History at cutover

> TO CONFIRM. Not stated as fact: how a uma conversation maps onto aps session
> history at cutover (whether prior turns carry over or restart) is a
> decision in progress. Replace this section with the confirmed behaviour
> before relying on it, and do not plan a cutover around either outcome until
> then.

## See also

- [Message Services](messengers.md#uma-hosted-providers-opt-in): the
  `--type uma` reference, cutover URL rules and per-adapter bindings.
- [uma parity checklist](../dev/requirements/uma-bridge-parity-checklist.md):
  row-by-row evidence behind the differences above.
