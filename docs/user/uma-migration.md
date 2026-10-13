# Move SMS and WhatsApp Services to uma

> Availability: unreleased. Requires a build newer than aps/v0.6.0-alpha.0.

Use this when a message service runs on the aps-native SMS or WhatsApp
transport (`--type sms`, `--type whatsapp`) and you want it hosted by an
in-process uma engine instead. Migration is opt-in per service. The native
transports stay the default and keep working; nothing here sets a removal date.

## Quick path

Find the row for your service. Only the first three move today.

| Native service | Becomes | Moves today |
| --- | --- | --- |
| `--type sms --provider twilio` | `--type uma --uma-adapter twilio.sms` | yes |
| `--type whatsapp --provider twilio` | `--type uma --uma-adapter twilio.whatsapp` | yes |
| `--type whatsapp --provider whatsapp-cloud` | `--type uma --uma-adapter meta.whatsapp` | yes, no templates |
| `--provider generic` (SMS, WhatsApp) | uma `relay.messages` | [not yet](#not-yet-generic-relay-and-email-relay) |
| Email relay | uma `http.mail` plus a sending adapter | [not yet](#not-yet-generic-relay-and-email-relay) |

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
- `aps service show` prints `auth: none` for a `uma` service. That is aps's own
  check, not the provider's; uma still verifies the signature.
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
| `--auth-scheme`, `--auth-token-env`, `--signature-secret-env` | Generic webhook auth does not run on a uma route; uma checks the provider signature | Nothing. The provider signature check is always on |
| `--provider` | The uma adapter picks the provider | `--uma-adapter` |
| `--receive polling` | A uma service receives by webhook only | Omit it, or `--receive webhook` |
| `--phone-number-id` on `twilio.sms`, `twilio.whatsapp` | Only `meta.whatsapp` reads it | Omit it |
| `--from`, `--webhook-url` on `meta.whatsapp` | Not read; the callback URL is set at Meta | Omit them |
| `--webhook-secret-token`, `--webhook-secret-token-env`, `--bot-user-id`, `--require-bot-mention`, `--dedup-ttl` | Telegram and Slack options | Omit them |
| `--site`, `--project`, `--jql`, `--linear-workspace`, `--team`, `--group`, `--events` | Ticket options | Omit them |

A key set with `--option` that falls in the table (`template_name`,
`auth_token`, ...) warns as `--option <key>`. These flags are read: `--type`,
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

## Not yet: generic relay and email relay

This build cannot host them. `--uma-adapter relay.messages` and `http.mail` are
refused (`unsupported uma adapter ... one of meta.whatsapp, twilio.sms,
twilio.whatsapp`). Leave `--provider generic` services and the email relay on
the native transport. The uma side differs enough to plan for.

### Generic relay (`relay.messages`)

Replies leave by a callback your gateway serves, never in the webhook answer.
The owner accepted that on 2026-10-13: it is a planned change, not a blocker.
Migration still waits for the aps bridge to host `relay.messages`, which it does
not yet, so there is nothing to run today. When it does, your gateway needs:

- **A callback endpoint.** Each reply is one JSON `POST` to it (`medium`, `to`,
  `body`, `in_reply_to`, `operation_id`, `attempt_id`) that you answer `2xx`. A
  gateway that reads the reply out of the webhook answer must change.
- **Authentication on its webhooks.** uma refuses a relay channel without one:
  bearer, token, HMAC or Ed25519. There is no unauthenticated mode.
- **The callback host on the allowlist.** The callback URL is `https`, with no
  credentials, query or fragment, and its host (`host` or `host:port`) must be
  on the uma host's callback host allowlist. With an empty list no reply can
  leave. Redirects are not followed. The allowlist is a uma host setting, not a
  service flag; the bridge sets none today.
- **No resend of a lost answer.** uma cannot look a reply up, so one whose
  answer was lost is not sent again. Each call carries an `Idempotency-Key`;
  check your gateway's log for it.

Inbound keeps the field names (`from`, `to`, `body`, `message_id`, `media`).
Form bodies and flat `NumMedia` media keys are not read.

### Email relay (`http.mail`)

Receive only. Replies need a separate sending channel, and form-encoded routes
(SendGrid Inbound Parse, the default Mailgun route) are not accepted. Attachment
bytes the relay embeds (base64 `content`, or a raw `message/rfc822` body) are
staged; an entry with only a URL is described and never fetched.

Details: [parity checklist](../dev/requirements/uma-bridge-parity-checklist.md),
notes M10 and M11.

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
their output for a native service of the three shapes that move today, pointing
here. `--provider generic` and email services get none. It is a next-step hint:
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
