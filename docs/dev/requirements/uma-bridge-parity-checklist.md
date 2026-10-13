# uma Parity Checklist: SMS, WhatsApp, Email Relay

What aps SMS, WhatsApp and email-relay services do today, row by row, mapped
to the uma adapters that would replace them. A red row is a retirement
blocker for that aps transport; a red row marked "bridge start: no" does not
stop a bridge adapter from carrying the green rows first.

## Scope and pins

| Side | Source | Pin |
| --- | --- | --- |
| aps | this repository, `internal/` | branch base `599cd6f` |
| uma | `hop-top/uma`, `main` | `650e527` (docs and `adapter/` source both read) |

Read dates: 2026-10-13. Every cite below was read, not recalled. aps paths are
relative to `internal/`. uma paths are prefixed `uma:`.

## How to read

| Column | Meaning |
| --- | --- |
| Behaviour | What an operator relies on today |
| aps | Where it lives, `file:line` |
| uma | Adapter doc page or source that covers it, `file:line` |
| Rec | Recorded cassette for the uma side: `yes` (dir), `no` (skips or none), `n/a` (inbound, nothing for a cassette to record) |
| Status | GREEN: uma declares and implements it, or a superset. RED: no uma equivalent, or uma refuses it |
| Bridge start | RED rows only: `no` = does not block a bridge adapter for SMS plus Twilio signature plus Messages REST plus TwiML reply; `n/a` = not an SMS row |

aps side has no recorded provider cassettes. Its tests drive `httptest` and
in-process fixtures; `e2e/conformance/cassettes` holds CLI story captures, not
provider traffic. Twilio signature tests sign with aps's own `TwilioSignature`
(self-referential); uma verifies against Twilio's documented worked example
(`uma:adapter/messaging/twilio/signature_test.go:35`).

## Totals

39 rows: 28 GREEN, 11 RED.

- RED behaviour (6): S6, G1, G2, T3, E1b, E4. None blocks the twilio.sms
  bridge start.
- RED evidence (5): V2 to V6. Recording gaps, tracked below. They gate
  retirement of the WhatsApp and mail paths, not the bridge start.

## 1. SMS over Twilio

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| S1 | `X-Twilio-Signature`: HMAC-SHA1 over URL plus sorted params, constant-time compare | `core/messenger/sms.go:36-80` (`hmac.Equal` :57) | `uma:adapter/messaging/twilio/signature.go:35-58`; `uma:docs/adapters/twilio.sms.md:87-90` | n/a | GREEN |
| S2 | Missing or wrong signature refused before any action runs | `adapters/messenger/handler.go:188-203`, `:1239-1243` (401) | `uma:adapter/messaging/twilio/sms/webhook.go:77-85` (403, sink never called); `uma:docs/adapters/twilio.sms.md:92-99` | n/a | GREEN |
| S3 | Signed URL: configured `webhook_url`, else rebuilt from `X-Forwarded-Proto`/`Host` and the request URI | `core/messenger/sms.go:45`; `adapters/messenger/handler.go:1368-1389` | `uma:adapter/messaging/twilio/sms/webhook.go:54,73-76`; `uma:adapter/messaging/twilio/sms/config.go:68-73`. `webhook_url` required (503 without), no query in config, request query appended when signing | n/a | GREEN, see M1 |
| S4 | Auth token from option, env or `secret:` ref | `core/messenger/sms.go:101-119` | `uma:adapter/messaging/twilio/sms/webhook.go:112-128` (credential purpose `twilio.webhook`, resolved by host, never from env) | n/a | GREEN, see M4 |
| S5 | Form-encoded inbound: `From`, `To`, `Body`, `MessageSid` to sender, channel, text, id | `adapters/messenger/normalizer.go:675-733`; `core/messenger/sms.go:256-311` | `uma:adapter/messaging/twilio/sms/webhook.go:132-198` | n/a | GREEN |
| S6 | JSON-bodied Twilio webhook, with `bodySHA256` in the URL body-hash checked. Documented: "Twilio form posts and JSON-style relays are both accepted" | `core/messenger/sms.go:53,82-99`; `docs/user/messengers.md:286`, `docs/MESSENGERS_OVERVIEW.md:405`. No aps test covers it | `uma:adapter/messaging/twilio/signature.go:36-38` refuses non-form bodies; test case `"json body"` at `uma:adapter/messaging/twilio/signature_test.go:75` pins `ErrMalformed` (400) | n/a | RED. Bridge start: no. Gap: `twilio-json-webhook` |
| S7 | Inbound MMS: media reaches the action as attachments (type, URL, mime) | `adapters/messenger/normalizer.go:718-730`; `core/messenger/sms.go:446-459` (Twilio `MediaUrl{n}` passed through) | `uma:adapter/messaging/twilio/sms/media.go:24-67` downloads from Media resources and stages through the engine's attachment sink, at most 10; `uma:docs/adapters/twilio.sms.md:101-107` | no (fake only; no media cassette) | GREEN, see M8 |
| S8 | Reply to the sender, inline in the webhook answer as TwiML (`<Response><Message>`); empty `<Response/>` when no reply | `adapters/messenger/handler.go:350-353`, `:361-366`, `:1208-1218`; test `adapters/messenger/sms_twiml_test.go:59,83` | uma answers empty TwiML (`uma:adapter/messaging/twilio/sms/webhook.go:107-108`) and replies through the Messages API (`send.go:26-53`); `uma:docs/adapters/twilio.sms.md:28-29` | yes (`send-and-fetch`, `rejected`, `refused-credentials`) | GREEN, see M5 |
| S9 | Messages REST outbound: `POST /2010-04-01/Accounts/{sid}/Messages.json`, basic auth, From/To/Body. Not on the production SMS path (see F2) | `core/messenger/sms.go:332-384` (endpoint :351) | `uma:adapter/messaging/twilio/sms/send.go:26-53`; `uma:docs/adapters/twilio.sms.md:109-114` | yes (`send-and-fetch`) | GREEN |
| S10 | REST failure classes: 429 and 5xx marked retriable, 4xx not | `core/messenger/sms.go:374-377` | `uma:docs/adapters/twilio.sms.md:111-114`, `:154-157`: 429 `rate_limited` with `Retry-After`, refused creds `unauthorized`, 5xx after send `outcome_unknown` (never blind-resent), `Reconcile` for lost answers | yes (`rejected`, `refused-credentials`, `reconcile-sent`, `reconcile-none`) | GREEN |
| S11 | Sender: one `from` number per service | `core/service.go:647-663` (`from` required); `core/messenger/sms.go:219` | `uma:adapter/messaging/twilio/sms/config.go:25-26,62-66` (`from` or `messaging_service_sid`) | yes | GREEN |

## 2. Generic relay (any provider, JSON in and out)

aps: `provider: generic` on `sms` and `whatsapp` services.

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| G1 | JSON in (`from`, `to`, `body`, `message_id`, optional media), JSON reply out in the webhook answer, for a provider with no uma adapter | `adapters/messenger/handler.go:355`, `:1326-1343`; `core/service.go:651,667` (`generic` accepted); test `adapters/messenger/sms_twiml_test.go:129` | none. No `generic`, JSON or HTTP-relay adapter in `uma:docs/adapters/README.md` capability matrix | n/a | RED. Bridge start: no. Gap: `generic-relay` |
| G2 | Generic webhook auth: `bearer`, `token`, `hmac-sha256` (`X-APS-Signature`), `ed25519`, timestamp tolerance, replay-id rejection. Also guards the email bridge | `core/messenger/validation.go:244-286,288-373,375-404`; `docs/MESSENGERS_OVERVIEW.md` "Generic Webhook Auth" | none. Each uma `receive_http` adapter verifies only its provider's signature | n/a | RED. Bridge start: no. Gap: `generic-webhook-auth` |

## 3. WhatsApp Cloud (Meta)

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| C1 | Subscription verification: GET, `hub.mode=subscribe`, verify token compared, `hub.challenge` echoed | `core/messenger/whatsapp.go:50-79`; `adapters/messenger/handler.go:148-150,653-669`; test `adapters/messenger/handler_test.go:171-204`. Unreachable in `aps serve`, see F1 | `uma:adapter/messaging/meta/whatsapp/webhook.go:187-202`; `uma:adapter/messaging/meta/webhook.go:63-72` (constant-time, digits-only challenge) | n/a | GREEN |
| C2 | `X-Hub-Signature-256` HMAC-SHA256 over raw body | `core/messenger/whatsapp.go:25-46`. Optional: no secret configured means no check (`core/service.go:663-696` only warns) | `uma:adapter/messaging/meta/whatsapp/webhook.go:215-223` (401; secret mandatory); `uma:adapter/messaging/meta/webhook.go:32-47` | n/a | GREEN, see M2 |
| C3 | Only the configured `phone_number_id` accepted | `core/messenger/validation.go:231-235` (403) | `uma:adapter/messaging/meta/whatsapp/webhook.go:248` (other numbers' changes skipped, 200) | n/a | GREEN |
| C4 | Inbound text: sender, profile name, phone-number channel, `wamid`, Meta timestamp | `core/messenger/whatsapp.go:297-365`; `adapters/messenger/normalizer.go:735-808` | `uma:adapter/messaging/meta/whatsapp/webhook.go:302-377` | no | GREEN |
| C5 | Reply context: `context.id` becomes `thread.id` (type `reply`), which feeds the conversation session key | `core/messenger/whatsapp.go:359-363`; `docs/dev/message-conversation-policy.md:78` | `uma:adapter/messaging/meta/whatsapp/webhook.go:312-314` keeps it in the inbound payload (`context.from`, `context.id`), not as an envelope thread. Bridge maps it back | n/a | GREEN, see M3 |
| C6 | Inbound media (image, audio, video, document, sticker), caption as text | `core/messenger/whatsapp.go:568-588` (Graph media id only, no download) | `uma:adapter/messaging/meta/whatsapp/webhook.go:354-362`, `media.go:40` (downloads with the token from the Graph and media hosts only, stages as attachment); `uma:docs/adapters/meta.whatsapp.md:85-90` | no | GREEN, see M8 |
| C7 | Outbound text: `POST {base}/{phone_number_id}/messages`, bearer token, `preview_url` false | `core/messenger/whatsapp.go:380-448` | `uma:adapter/messaging/meta/whatsapp/send.go:98-159`, `:163-192` | no (harness `cassette_test.go`, nothing recorded) | GREEN |
| C8 | Outbound template: `template_name` plus `language_code` (default `en_US`); every reply goes as that template, reply text dropped, no components; `template_required` makes it mandatory | `core/messenger/whatsapp.go:164-176,395-402`; `adapters/messenger/handler.go:966-975`; `cli/service/add.go:64-66,275-278` | `uma:adapter/messaging/meta/whatsapp/send.go:176-187`: part `meta.whatsapp.template` with `name` and `language.code`, `components` passed through; `uma:docs/adapters/meta.whatsapp.md:98-107` | no (`TestRecordedTemplateSend` skips) | GREEN, see M6 |
| C9 | Graph failures: 429 and 5xx retriable, error message surfaced | `core/messenger/whatsapp.go:438-441,613-619` | `uma:adapter/messaging/meta/whatsapp/send.go:84-97` (outcome mapping: `rejected`, `rate_limited`, `unauthorized`, `outcome_unknown` after a possible send, never resent) | no | GREEN |

## 4. WhatsApp over Twilio

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| T1 | Signature: same `X-Twilio-Signature` check as SMS when `provider: twilio` | `core/messenger/validation.go:122-130,151-160` (hook key `twilio`); `core/messenger/sms.go:36-80` | `uma:adapter/messaging/twilio/whatsapp/webhook.go:67-88`; JSON-bodied variant is S6 | n/a | GREEN |
| T2 | Inbound text; `whatsapp:+1555` addresses | `core/messenger/whatsapp.go:259-295` (prefix kept in ids) | `uma:adapter/messaging/twilio/whatsapp/webhook.go:131-193` (prefix stripped; non-WhatsApp address is 400). Also reports buttons and locations (`:228-256`) | n/a | GREEN, see M3, M7 |
| T3 | Inbound media reaches the action as attachments (type, URL, mime) | `core/messenger/whatsapp.go:269` via `smsAttachments` (`core/messenger/sms.go:446-459`) | `uma:adapter/messaging/twilio/whatsapp/webhook.go:240-247` keeps count and content types only; `uma:docs/adapters/twilio.whatsapp.md:96-98` "media URLs are not kept". No staging, unlike `twilio.sms` | no | RED. Bridge start: n/a. Gap: `twilio-whatsapp-media` |
| T4 | Outbound text through Twilio Messages API, `whatsapp:` prefixes added, not TwiML | `core/messenger/whatsapp.go:464-523`; `adapters/messenger/handler.go:671-759` | `uma:adapter/messaging/twilio/whatsapp/send.go:48-90` | no (`TestRecordedSendAndFetch` skips) | GREEN |
| T5 | Outbound template: `content_sid` plus variables. Not reachable from a service config today (see F3) | `core/messenger/whatsapp.go:479-491` (only `Metadata["content_sid"]`; `replyMetadata` never sets it) | `uma:adapter/messaging/twilio/whatsapp/send.go:24-38,154-174`; 24-hour window enforced when protocol state is present (`window.go`) | no | GREEN (superset) |

## 5. Email relay

aps does not poll a mailbox: an external bridge (IMAP poller, MTA hook,
inbound-parse webhook) POSTs one JSON object per message and sends the reply
JSON itself (`docs/MESSENGERS_OVERVIEW.md` "Email").

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| E1a | Mail in from a mailbox or MX (the IMAP-poller and MTA-hook cases). uma replaces the bridge | `adapters/messenger/normalizer.go:597-658` | `imap.mail` `receive` (`uma:docs/adapters/imap.mail.md:24-33,93-109`), `smtp.mail` `receive` (`uma:docs/adapters/smtp.mail.md:22-28,89-104`), `google.mail` and `microsoft.mail` (`receive`, `receive_http`) | no (see V4 to V6) | GREEN, see M9 |
| E1b | Mail in by HTTP POST of `{from,to,subject,body}`: inbound-parse services and custom MTA hooks | `adapters/messenger/normalizer.go:597-608`; `docs/MESSENGERS_OVERVIEW.md:493-495` names "inbound-parse webhook" | none. `smtp.mail` and `imap.mail` do not declare `receive_http` (`uma:docs/adapters/smtp.mail.md:36-38`, `imap.mail.md:41`); `google.mail` and `microsoft.mail` take only their own provider's push | n/a | RED. Bridge start: n/a. Gap: `mail-http-ingress` |
| E2 | `from` to sender, `to` to channel, `body` to text | `adapters/messenger/normalizer.go:598-628` | `uma:docs/adapters/smtp.mail.md:67-104` (recipients map to destinations, From is sender); `uma:docs/adapters/imap.mail.md:70-109` | no | GREEN |
| E3 | `subject` is the thread id | `adapters/messenger/normalizer.go:630-636` | threading by `In-Reply-To` then `References`, not subject (`uma:docs/adapters/smtp.mail.md:99-100`) | no | GREEN, see M3 |
| E4 | Attachments `[{type,url,mime_type,size_bytes}]` reach the action. Test: `adapters/messenger/normalizer_test.go:982` | `adapters/messenger/normalizer.go:638-655` (bridge-supplied URL passed through) | metadata only, no bytes, no URL: `uma:docs/adapters/smtp.mail.md:99` "Attachment bytes are not kept"; `uma:adapter/mail/google/inbound.go:152-162`; `uma:adapter/mail/microsoft/doc.go:41`; `imap.mail.md:107-109` | no | RED. Bridge start: n/a. Gap: `mail-attachments` |
| E5 | Reply reaches the sender. aps: reply JSON (`body`, `subject` fixed `Re: Action Result`, optional `data`) returned to the bridge, which sends it | `adapters/messenger/normalizer.go:901-910` | uma sends itself: `send` on `smtp.mail`, `imap.mail`, `google.mail`, `microsoft.mail`; `text/plain`, `Re:` subject, `In-Reply-To`/`References`; outcomes `provider_accepted`, `rejected`, `outcome_unknown`; `reconcile` on all but `smtp.mail` (`uma:docs/adapters/smtp.mail.md:106-112`, `imap.mail.md:110-118`). `data` payload has no mail form | no | GREEN |

## 6. Recorded evidence

Retirement needs recordings against real accounts, not only fakes. State of
each uma adapter, from `uma:docs/recording/README.md:23` ("Cassettes are
committed for GitHub issues and Twilio SMS only") and the per-adapter test
sections.

| # | Adapter | State | Status |
| --- | --- | --- | --- |
| V1 | `twilio.sms` | 7 cassette sets under `uma:adapter/messaging/twilio/sms/testdata/cassettes/twilio/` (`send-and-fetch`, `reconcile-sent`, `reconcile-none`, `rejected`, `refused-credentials`, `fetch-unknown`, `history-reply`). Inbound webhook and MMS media download are not HTTP-client traffic or not recorded: signature checked against Twilio's documented vector, media against a fake | GREEN |
| V2 | `twilio.whatsapp` | harness only, "none are recorded yet" (`uma:docs/adapters/twilio.whatsapp.md:111-118`); `testdata/` holds hand-written signed webhook forms only | RED. Existing uma task: record Twilio WhatsApp sandbox cassettes |
| V3 | `meta.whatsapp` | harness `cassette_test.go` skips; "none are recorded yet" (`uma:docs/adapters/meta.whatsapp.md:123-130`) | RED. Existing uma task: record Meta WhatsApp cassettes |
| V4 | `google.mail` | `TestRecordedGmail` skips (`uma:docs/adapters/google.mail.md:172-176`) | RED. Existing uma task: record Gmail cassettes |
| V5 | `microsoft.mail` | `TestRecordedGraph` skips (`uma:docs/adapters/microsoft.mail.md:161-165`) | RED. Existing uma task: record Microsoft Graph cassettes |
| V6 | `imap.mail`, `smtp.mail` | no cassettes and no record path: xrr cannot record IMAP or SMTP line protocols (`uma:docs/recording/imap-smtp.md`). The umbrella task for WhatsApp and mail cassettes cannot close this one | RED. Gap: `imap-smtp-evidence` |

Premise check: `twilio.sms` is the only adapter with real cassettes among the
SMS, WhatsApp and mail adapters. `github.issues` also has them
(`uma:adapter/ticket/github/testdata/cassettes`, 88 files), so "only adapter
in uma" is wrong; "only one on this list" is right.

## Gap names

Each RED row maps to one uma task whose title carries the name below and whose
description cites this file and the row number.

| Gap | Rows | uma task title |
| --- | --- | --- |
| `twilio-json-webhook` | S6, T1 | twilio.sms and twilio.whatsapp: accept JSON-bodied webhooks with bodySHA256 |
| `generic-relay` | G1 | Generic JSON message relay adapter for SMS and WhatsApp providers |
| `generic-webhook-auth` | G2 | Generic webhook auth for receive_http adapters: bearer, token, HMAC, Ed25519, timestamp, replay |
| `twilio-whatsapp-media` | T3 | twilio.whatsapp: stage inbound media through the attachment sink |
| `mail-http-ingress` | E1b | Mail HTTP ingress adapter for inbound-parse and MTA-hook relays |
| `mail-attachments` | E4 | Mail adapters: make inbound attachments reachable by stage-through-sink |
| `imap-smtp-evidence` | V6 | imap.mail and smtp.mail: live-server parity evidence without cassettes |

V2 to V5 ride on the existing recording tasks; no new task.

## Migration and bridge notes

Config translation the bridge (or the operator) must do. None is a uma gap.

| # | Note |
| --- | --- |
| M1 | `webhook_url` becomes mandatory and must have no query or fragment. aps derives it from forwarded headers when unset. If the provider console URL carries a query, configure the base URL; uma appends the request's query when verifying |
| M2 | Meta app secret becomes mandatory. aps accepts unsigned Cloud webhooks when no secret is set |
| M3 | Session identity inputs shift: Twilio addresses lose the `whatsapp:` prefix, email thread moves from subject to headers, WhatsApp reply context moves from envelope to payload. The bridge must rebuild the old sender, channel and thread fields, or conversation history restarts at cutover (`core/messenger/types.go:116-163`) |
| M4 | aps option names (`twilio_auth_token`, `auth_token`, `signature_secret`, `app_secret`, `verify_token`, `WHATSAPP_ACCESS_TOKEN`) map to uma credential purposes `twilio.webhook`, `twilio.rest`, `meta.app_secret`, `meta.webhook.verify_token`, `meta.graph`; the host resolves them |
| M5 | SMS and WhatsApp replies change from inline TwiML (SMS) to asynchronous Messages API calls: the REST credential, and `from` or a Messaging Service, are required, and status callbacks arrive at `webhook_url`. The action no longer has to finish inside Twilio's webhook timeout |
| M6 | aps `template_name`/`language_code` become a per-reply `meta.whatsapp.template` part built by the bridge. aps sends the template with no components and drops the reply text; keep that or add components deliberately |
| M7 | Twilio `from` is configured without `whatsapp:` in uma |
| M8 | Media staging, and the 24-hour window state for WhatsApp, need the host to supply an attachment sink and protocol state; without them media is summary-only and Twilio or Meta enforce the window |
| M9 | The email bridge process retires once uma holds the mailbox or MX; its generic webhook auth (G2) is no longer used for email |

## Retained by aps

Not uma concerns. The bridge must keep calling them on bridged traffic.

| Behaviour | aps |
| --- | --- |
| Sender and channel allowlists `allowed_numbers`, `allowed_senders` (exact or `*@domain`), `allowed_channels`, `allowed_chats` | `core/messenger/validation.go:208-242`; `core/allowed_senders.go:9-60`. uma has no allowlist; for `smtp.mail` it adds SPF/DKIM/DMARC sender evidence |
| Reply policy `reply: none/text/auto`, empty or failed output sends nothing | `adapters/messenger/handler.go:361-366,1018-1027`; `core/service.go:809-823` |
| Routing: `default_action`, sender route table, execution mode `chat` | `adapters/messenger/adapter.go:125-160`; `adapters/messenger/handler.go:798-822` |
| Conversation id, session id, `prior_turns`, outbound turn recording | `core/messenger/types.go:116-163`; `adapters/messenger/handler.go:997-1006` |
| Action input shape including `platform_metadata` (raw provider payload for aps today; typed envelope payload in uma) | `adapters/messenger/normalizer.go:715,798`; `docs/MESSENGERS_OVERVIEW.md` "Normalized Message Format" |

## aps findings

Surprises found while verifying. Not uma gaps; fixes belong in aps.

| # | Finding | Evidence |
| --- | --- | --- |
| F1 | WhatsApp Cloud GET verification is unreachable through `aps serve`. Only `POST /services/{service}/webhook` is mounted; GET answers 405 (probed against `Adapter.RegisterRoutes`, and stdlib mux semantics). Docs say verification uses the same service URL | `adapters/messenger/adapter.go:90`; `adapters/messenger/handler.go:148-150`; `docs/MESSENGERS_OVERVIEW.md:471-473`, `docs/user/messengers.md:278-279` |
| F2 | Messages REST for SMS (`SMSProvider`, `TwilioSMSTransport`) is not on any production path. Production SMS replies are TwiML only. Only tests and `tools/messengermd` construct it. `core/service.go:523-548` still describes SMS as "provider delivery" | `core/messenger/sms.go:176`; `tools/messengermd/main.go:112`; `adapters/messenger/handler.go:350-353` |
| F3 | Twilio WhatsApp `content_sid` is unreachable from a service config. With `provider: twilio` plus `template_name`, the `template_required` check passes and the message still goes out free-form | `core/messenger/whatsapp.go:171,479-491`; `adapters/messenger/handler.go:966-975` |
| F4 | Cloud status-only notifications (`statuses`, no `messages`) fail normalisation and answer 400, so Meta retries them | `core/messenger/whatsapp.go:306-308`; `adapters/messenger/handler.go:1095-1100` |
| F5 | The `bodySHA256` JSON path has no test | no `bodySHA256` hit in any `_test.go` |

## Line-ref notes

Earlier line hints for these paths had drifted or named the wrong file.
`handler.go:361` (`twilioSMSReply`) and `normalizer.go:597` (`normalizeEmail`)
are right. The handler is `adapters/messenger/handler.go`, not under
`core/`. `X-Hub-Signature-256`, the verify token, WhatsApp media and reply
context live in `core/messenger/whatsapp.go` (`:16`, `:25-79`, `:359-363`,
`:568-588`); `normalizer.go:668-1106` holds only normalisation and
denormalisation (`:668-808`, `:901-929`, `:1067-1090`). Cloud transport is
`whatsapp.go:380-448` (template `:395-402`), Twilio transport `:464-523`.
