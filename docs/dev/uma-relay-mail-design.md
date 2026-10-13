# uma Bridge: Hosting `relay.messages` and `http.mail`

Status: decided. Items 1 to 4 and 6 are settled by the two codebases; item 5 by the owner on
2026-10-13: a bridged `http.mail` service is receive only, replies disabled. Background:
[uma bridge](messenger-architecture.md#uma-bridge),
[parity checklist](requirements/uma-bridge-parity-checklist.md) rows G1, G2, E1a to E5.

Code references are `file:line`. aps paths are relative to `internal/`: `cm/` is
`core/messenger/`, `am/` is `adapters/messenger/`, `ub/` is `adapters/umabridge/`. uma paths
(`uma:`) are the sibling checkout at `main` `02ae8cb`: `R/` is `adapter/messaging/relay/`, `W/`
is `adapter/web/webhookauth/`, `H/` is `adapter/mail/httpmail/`, `D/` is `docs/adapters/`.

## Summary

- `--uma-adapter relay.messages` hosts generic SMS and WhatsApp relays; `--uma-adapter
  http.mail` hosts the email relay, receive only.
- The generic webhook auth flags are read on both, not warned: aps maps them onto
  `webhookauth.Config` with aps's own header names, encodings and signed layout, so a sender
  keeps working. Two uma defaults stay in force, each with an explicit per-service opt-in:
  secrets of at least 16 bytes, and refusal of timestamps and replay ids the signature does not
  cover.
- Replay rejection lives in memory, as it does natively. No durable store is added.
- Relay replies go to a callback. Its host allowlist is global aps configuration, empty by
  default, so no reply leaves until the operator lists the host.
- `http.mail` is receive only, replies disabled (owner, 2026-10-13). An action's reply is
  refused, logged and shown as `reply_disabled` in the service status; `service add` warns when
  the reply mode would produce one. No pairing with a sending adapter is built.
- Form bodies: the generic relay loses them (native read them); the email relay never had them.

## 1. Service shape

The service type is `uma`, so the medium a native `sms` or `whatsapp` type carried becomes an
option. Inbound field names are fixed by aps to the names the native normalizer reads
(`am/normalizer.go:675-705`), so they need no option.

### `relay.messages`

| aps input | uma or bridge | Notes |
| --- | --- | --- |
| `--option medium=sms` or `whatsapp`, required | `medium`; aps platform `sms`, `whatsapp` | Picks the uma identity namespace and the aps history keys |
| `--from`, required | inbox destination | The number exactly as the gateway reports it in `to` |
| `--option callback_url`, required unless `--reply none` | `callback.url` | `https`, host on the [allowlist](#4-replies-and-the-callback-allowlist) |
| `--option callback_header` | `callback.auth: header`, `callback.header` | Default: `Authorization: Bearer` |
| `--env RELAY_CALLBACK_TOKEN=secret:NAME`, required with a callback | purpose `relay.callback` | Printable ASCII, no spaces (`R/send.go:131-163`) |
| Generic auth flags and options | `auth`, purpose `relay.webhook` | [Item 2](#2-webhook-auth-mapping) |
| fixed | `inbound.fields`: `from` `From`,`from`; `to` `To`,`to`; `body` `Body`,`body`,`text`; `message_id` `MessageSid`,`SmsSid`,`message_id`; `media` | Native names; `WaId` left out (below) |
| fixed | `inbound.generate_missing_id: true` | Native generates an id when none is sent (`am/normalizer.go:690-696`) |
| `--allowed-number`, `--default-action`, `--route-table`, `--reply`, `--history-turns` | aps | As for the other uma adapters |

`WaId` is the sender's WhatsApp number. Natively it is the last id fallback; under uma, whose
event id is the sender plus the message id (`R/webhook.go:144-147`), it would make every later
message of that sender a duplicate. A message carrying only `WaId` gets a generated id instead.

### `http.mail`

| aps input | uma or bridge | Notes |
| --- | --- | --- |
| `--option recipients=a@example.com,b@example.com`, required | `recipients`, every address mapped to one destination | A message to two listed addresses reaches the action once |
| `--option formats=json,rfc822` | `formats` | Default `json` |
| `--option recipient_header` | `recipient_header` | `rfc822` only |
| `--option trusted_authserv_id`, `--option authentication_results_field` | `evidence.*` | Without them every sender is `unverified` |
| `--reply` | no send path | Any mode but `none` warns; a reply is refused. [Item 5](#5-email-relay-replies) |
| Generic auth flags and options | `auth`, purpose `http.mail.webhook` | [Item 2](#2-webhook-auth-mapping) |
| fixed | `json_fields` defaults | uma's defaults are the native names: `from`, `to`, `subject`, `body`, `attachments` with `type`, `url`, `mime_type`, `size_bytes` (`H/config.go:220-244`) |
| `--allowed-sender` | aps | The native email allowlist, on the sender address |

### Secret bindings

| Purpose | Binding | Required |
| --- | --- | --- |
| `relay.webhook`, `http.mail.webhook` | The name `--auth-token-env` (bearer, token) or `--signature-secret-env` (hmac-sha256, ed25519) gives, bound with `--env NAME=secret:NAME` | Yes; `service add` refuses a service without the binding |
| `relay.callback` | `RELAY_CALLBACK_TOKEN` | With `callback_url` |

The binding is resolved per request with `core.ResolveSecretValue`, like every bridge
credential. Natively `auth_token_env` names a process variable (`cm/validation.go:555-560`);
the bridge reads the service binding of that name instead, and refuses a service without one, so
a migration adds one `--env` line. Literal `auth_token` and `signature_secret` options are never
read by the bridge: `service add` warns.

## 2. Webhook auth mapping

Native generic auth: `cm/validation.go:244-414`. uma: `W/config.go:50-190`,
`uma:docs/webhook-auth.md` "Senders already in the field".

| `auth_scheme` | Header default | `webhookauth.Config` | Signature covers |
| --- | --- | --- | --- |
| `bearer` | `Authorization` | `bearer` | nothing |
| `token` | `X-APS-Token` | `token` | nothing |
| `hmac-sha256` | `X-APS-Signature` | `hmac`, `sha256`, `hex`, prefix `sha256=` optional, payload `{body}` | body |
| `ed25519` | `X-Signature-Ed25519` | `ed25519`, `hex`, key `hex`, payload `{timestamp}{body}`, timestamp header `X-Signature-Timestamp` | timestamp, body |
| `slack-signing-secret` | | refused | |

- Scheme inference as natively (`cm/validation.go:256-262`): no scheme with a token option is
  `bearer`, with a signature option `hmac-sha256`. `auth_header` overrides the header.
- Timestamp: `timestamp_header`, or `require_timestamp` for `X-APS-Timestamp`; format
  `unix_or_rfc3339` (native: RFC 3339 with or without a fraction, or Unix seconds,
  `cm/validation.go:416-427`); tolerance `timestamp_tolerance`, default 5m, at most 24h (uma's
  limit; a larger or unparsable value is refused, where natively it fell back to 5m).
- Replay: `replay_id_header`, or `require_replay_check` for `X-APS-Delivery-ID`; by header.
  A missing id is `401` in both. The window is two tolerances with a timestamp (native: one),
  24h without (native: 5m): longer, never shorter.
- No unauthenticated mode. A native generic or email service without auth options ran with none
  (email only warned, `core/service.go:711-728`); `service add` refuses such a uma service.

Two uma defaults the native layout does not meet. aps keeps both and asks for consent:

- **Signed timestamp and replay id** (`W/config.go:163,272,309-314`). Natively they are checked
  after and apart from the signature for bearer, token and hmac, and ed25519 leaves the replay id
  unsigned (`cm/validation.go:288-414`). Opt-in: `--option auth_allow_unsigned=true`, or sign
  them once the sender does: `--option auth_payload={timestamp}.{body}` (hmac, ed25519).
  Trade-off: unsigned, a captured request can be sent again with a fresh timestamp or id, so only
  honest stale deliveries are stopped. That is the exposure the service has today.
- **Secrets of at least 16 bytes** (`W/config.go:100-105,180-182`). Natively any non-empty secret
  passes. Opt-in: `--option auth_min_secret_bytes=N`. Trade-off: under 16 bytes an HMAC key is
  below NIST SP 800-107's 112 bits. Without the option a shorter secret makes every request
  `503`; the bridge refuses to start instead and logs which option to set.

Without the consent `service add` refuses the service and names both ways out. Nothing is
relaxed silently. `auth_payload` must contain `{body}` once and may name `{timestamp}` and `{id}`
only when a timestamp or replay header is configured.

Smaller differences: the bearer scheme word is matched without regard to case, a repeated
credential header is refused, and an HMAC prefix other than `sha256=` does not decode.

`UmaIgnoredOptions` per adapter: on `relay.messages` and `http.mail` the generic auth flags are
read and no longer warn. Literal `auth_token` and `signature_secret` still warn. On the Twilio
and Meta adapters the generic auth options keep warning, now including `timestamp_header`,
`replay_id_header`, `require_timestamp`, `require_replay_check` and `timestamp_tolerance`, which
were ignored there without a word. The new options warn on adapters that do not read them.

## 3. Replay store

- The bridge's uma store is in memory (`ub/bridge.go:128`). `storereplay` keeps its state in the
  root store under a channel (`uma:docs/webhook-auth.md` "Durable store"); over the memory store
  it forgets on restart exactly as `webhookauth.MemoryReplayStore` does, so it adds nothing.
  The bridge passes a `MemoryReplayStore` to both factories (`http.mail` refuses replay rejection
  without one, `H/factory.go:57-62`).
- What a bridged service gets: replay rejection for the life of its bridge, which ends with the
  process or when a configuration change rebuilds it. After that, a captured request still inside
  its timestamp window is accepted again. Native: the same, an in-memory map per handler
  (`cm/validation.go:77-118`).
- Relay backstop: `dedupExecutor` skips a message whose reply is in history (newest 500 turns of
  its session). Sessions key on the sender, and uma's event id binds the sender to the gateway's
  message id, so an answered relay message replayed after a restart is not run again.
  `http.mail` writes no replies, so it has no backstop.
- A durable replay store waits for a durable uma store (history decision, open question 1).

## 4. Replies and the callback allowlist

- The action's reply goes through `DeliverMessage` to `Messages.Reply`, as for the other
  adapters. The worker posts it to `callback.url` with `Idempotency-Key: <operation id>`, the same
  on every attempt (`R/send.go:166-184`). A lost answer is `outcome_unknown` and never resent
  (accepted, G1).
- Allowlist: global aps configuration, not a service option. uma keeps `CallbackHosts` out of the
  channel configuration so that a channel cannot send the host's credentials elsewhere
  (`R/relay.go:42-59`). In aps the service holds both the URL and the credential binding, so a
  service file alone could otherwise send any profile secret to any host.

  ```yaml
  uma:
    relay:
      callback_hosts: [gateway.example.com, "gateway.example.com:8443"]
      allow_http: false  # local development only: the credential travels in clear text
  ```

  Each entry is `host` or `host:port` exactly as in the URL. Empty, the default, sends nothing.
- `service add` refuses a callback URL that is not `https` (`http` only with `allow_http`), that
  carries credentials, a query or a fragment, or whose host is not listed. uma checks again at
  `Validate` and `Open`. The bridge fingerprint includes the allowlist, so an edit applies on the
  next webhook.
- The bridge's default HTTP client follows redirects, and relay's own wrapper cannot stop a
  client that does (`R/redirect.go`; `D/relay.messages.md:63`). The relay gets its own client
  that never follows one.
- With `--reply none` a callback is optional: the channel only receives.

## 5. Email relay replies

Native: the webhook answer carries `{body, subject, data}` and the operator's bridge sends the
mail (`am/normalizer.go:901-910`, `am/handler.go:375`). `http.mail` answers `{"accepted": n}` and
declares no `send` (`D/http.mail.md:38`).

| Option | What it takes | Verdict |
| --- | --- | --- |
| A. Receive-only bridged email, replies disabled | Nothing more | Chosen. Useful when the action needs no reply on this path (triage, filing, alerting, an action that sends by its own means). Safe: auth mandatory, nothing answered |
| B. Pair with a sending adapter (`smtp.mail`, `imap.mail` submission, a provider) | A second channel and its credentials, a From identity, reply threading; the bridge maps one service to one channel | Not built |
| C. Keep the native email relay | Nothing | For every email service whose bridge reads the reply |

Decision, owner, 2026-10-13: A. A bridged `http.mail` service is receive only with replies
disabled; no pairing is built. An email service that relies on the reply JSON stays native.

What happens to a reply. It is refused, never sent and never dropped without a trace:

- After the action, the bridge takes the reply off the result before delivery. Nothing reaches
  uma; `DeliverMessage` also refuses `http.mail` should anything get past that.
- One `warn` log line names the service, the message and the reason.
- The service's last outbound event has status `reply_disabled`, so `aps service status` shows
  it. The action's own status is in the log line; a failed action is recorded as failed, as on
  every service.
- No outbound turn is written. The history decision writes one when uma accepts a reply
  ([messenger-architecture.md](messenger-architecture.md#conversation-history), question 2), and
  that turn is also what keeps an answered event from running again. A turn for a reply nobody
  received would put text into the next action's `prior_turns` that the sender never saw, and
  turns have no column to mark it unsent (question 6). So a redelivered mail runs again, like
  any event that sent no reply.

`service add` warns on stderr, once, when the reply mode would produce a reply (`--reply` other
than `none`, or unset, which means `text`). `service show` and `routes` say replies are
disabled.

## 6. Form bodies

- `http.mail` refuses form bodies (SendGrid Inbound Parse, the default Mailgun route;
  `D/http.mail.md:99-100,307-309`). The native email relay reads JSON only: forms are parsed for
  SMS and WhatsApp alone (`am/handler.go:1346-1360`). No regression: a hosted form-only service
  needed a converter before and still does.
- `relay.messages` reads JSON only. Native generic SMS and WhatsApp read forms too, and flat
  `NumMedia`, `MediaUrlN` keys. A gateway that posts forms switches to JSON before it moves; media
  goes in `media`.
- aps parses no forms for uma: that belongs in uma.

## History and identity

Keys follow the history decision: derived from the `NormalizedMessage`, equal to native's.

| | `relay.messages` | `http.mail` |
| --- | --- | --- |
| platform | the medium | `email` |
| sender | `from` as sent | the From address |
| channel | `--from` (equals `to`) | the listed recipient that selected the message |
| thread | none | the subject, type `topic`, as native (`am/normalizer.go:630-636`) |
| id | the gateway's message id, or generated | uma's message ref; the `Message-ID` is the sender's and not an identity (`D/http.mail.md:197-201`) |
| media | `media` URLs passed through, never fetched (native passed `MediaUrlN` through) | staged bytes as `file://` under the media dir; URL-only entries passed through, never fetched |
| metadata | `uma_adapter`, `uma_message` | plus `subject`, `sender_assurance` (`unverified` unless trusted) |

## Dependency

`relay` and `webhookauth` are in the root module, already replaced. `httpmail` is in module
`hop.top/uma/adapter/mail`, which joins the replace block. Every uma pin moves to the checkout's
commit, `02ae8cb`.
