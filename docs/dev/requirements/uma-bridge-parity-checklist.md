# uma Parity Checklist: SMS, WhatsApp, Email Relay

What aps SMS, WhatsApp and email-relay services do today, row by row, mapped
to the uma adapters that would replace them. A RED row is a retirement
blocker; an AMBER row has a uma equivalent in the tree read but a named
behaviour residual stops it counting as parity; a WAIVED row is implemented
and cited but has no recorded cassette, which the owner waived on 2026-10-13.
GREEN and WAIVED rows count. A row marked `[i]` leans on uma work that is on
the integration branch only.

## State as of

| Side | Source | Pin |
| --- | --- | --- |
| aps | this repository, `internal/` | `ae9e407` (uma bridge tip `296ea84` plus this checklist) |
| uma | `hop-top/uma`, branch `integration/unmerged` | `1afff92` = `main` `650e527` plus 57 commits |
| aps build | `go.mod:178-196` replace block | sibling checkout of uma `main` at `650e527`: the integration work is NOT in aps's build |
| Previous grade | aps `599cd6f`, uma `main` `650e527` | 28 GREEN, 11 RED |

Read dates: 2026-10-13. Every cite below was read, not recalled. uma was read
from the integration tree, not `main`. Nothing in the uma repo was changed.

## How to read

| Column | Meaning |
| --- | --- |
| Behaviour | What an operator relies on today |
| aps | Where it lives, `file:line` |
| uma | Adapter doc page or source that covers it, `file:line`, integration tree |
| Rec | Recorded cassette for the uma side: `yes`, `no` (fake or doc-written fixtures only), `n/a` (nothing for a cassette to record) |
| Status | GREEN, WAIVED, AMBER, RED as below; `[i]` = depends on uma work not on uma `main` |

Grading is mechanical:

- GREEN: uma implements the behaviour (cited) and Rec is `yes` or `n/a`.
- WAIVED: uma implements it (cited), Rec is `no`. Cassette waived by owner,
  2026-10-13: the recording task was skipped ("no need to record; parity
  evidence waived"). Evidence held: scripted fakes and the adapter docs, not
  provider traffic. Behaviour against the real provider stays unverified.
- AMBER: uma implements it (cited) but the operator must change a system
  outside aps to keep the behaviour, or part of it is missing. Named.
- RED: no uma equivalent, or uma refuses it.

Cite prefixes. aps paths are relative to `internal/`: `cm/` is
`core/messenger/`, `am/` is `adapters/messenger/`. uma paths: `T/` is
`uma:adapter/messaging/twilio/`, `Ts/` is `T/sms/`, `Tw/` is `T/whatsapp/`,
`M/` is `uma:adapter/messaging/meta/`, `Mw/` is `M/whatsapp/`, `R/` is
`uma:adapter/messaging/relay/`, `W/` is `uma:adapter/web/webhookauth/`, `Ml/`
is `uma:adapter/mail/`, `D/` is `uma:docs/adapters/`, `Dx/` is `uma:docs/`.

aps side has no recorded provider cassettes. Its tests drive `httptest` and
in-process fixtures; `e2e/conformance/cassettes` holds CLI story captures, not
provider traffic. Twilio signature tests sign with aps's own `TwilioSignature`
(self-referential); uma verifies against Twilio's documented worked example
(`T/signature_test.go:35`) and Twilio's SDK vector (`T/signature_json_test.go:45`).

## Totals

39 rows: 18 GREEN, 18 WAIVED, 3 AMBER, 0 RED. Previous grade: 28 GREEN, 11 RED.
Before the waiver the same tree graded 18 GREEN, 16 AMBER, 5 RED.

| Scope | Rows | GREEN | WAIVED | AMBER | RED |
| --- | --- | --- | --- | --- | --- |
| SMS, generic relay, WhatsApp Cloud, WhatsApp over Twilio, V1 to V3 | 30 | 18 | 11 | 1 | 0 |
| Email relay, V4 to V6 | 9 | 0 | 7 | 2 | 0 |
| All | 39 | 18 | 18 | 3 | 0 |

- RED to GREEN `[i]` (2): S6, G2.
- RED to AMBER `[i]` (3): G1, E1b, E4. The six behaviour REDs of the previous
  grade (S6, G1, G2, T3, E1b, E4) are all answered on the integration branch;
  none stays RED. T3 moved to WAIVED `[i]`.
- GREEN to WAIVED (12): S7, C4, C6, C7, C8, C9, T4, T5, E1a, E2, E3, E5. No
  behaviour changed: Rec is `no`, so the row rests on fakes and docs.
- RED to WAIVED (5): V2 to V6, the recording rows.
- Remaining AMBER (3), each a behaviour residual, none about recordings: G1
  (reply by callback), E1b (`http.mail` receives only, no form bodies), E4
  (`http.mail` stages no attachment bytes).

## Retirement gate status

Retiring the aps-native SMS and WhatsApp transports needs all of (a), (b), (c).
None is met today; (a) is one decision away in the SMS and WhatsApp scope.

| Gate | Status | What is missing |
| --- | --- | --- |
| (a) Every behaviour row green or waived; cassettes for `twilio.whatsapp`, `meta.whatsapp` waived by owner 2026-10-13 | NOT MET | G1 AMBER (callback reply); mail: E1b, E4 AMBER |
| (b) A tagged uma release | NOT MET | No tags; private and unresolvable; vanity server skips subpaths |
| (c) Every operator yaml migrated to `uma`, with a migration note | NOT MET | Note exists; no inventory; generic relay cannot move yet |

### Gate (a) detail

- Waiver, dated 2026-10-13: the owner set the uma task that recorded
  `twilio.whatsapp`, `meta.whatsapp` and mail cassettes to skipped ("no need to
  record; parity evidence waived"). Rows that lacked only a cassette (S7, C4,
  C6 to C9, T3 to T5, E1a, E2, E3, E5, V2 to V6) are WAIVED, not blockers. The
  evidence behind them is scripted fakes and the uma adapter docs; nothing was
  tried against the real providers.
- Zero RED rows. In the SMS and WhatsApp scope one AMBER row remains: G1. The
  reply no longer rides the webhook answer, so an operator gateway must serve
  a callback. The owner decides: accept it as a documented change (M10), or
  keep a native generic relay. Until then (a) is not met in that scope.
- Mail rows E1b and E4 stay AMBER for missing behaviour (`http.mail` receives
  only; it stages no attachment bytes). Retirement is scoped to SMS and
  WhatsApp; whether the mail rows gate it is the owner's call.
- The mail rows' evidence: a uma task building live IMAP and SMTP evidence
  against a local mail container is separate and still in progress. If it
  lands, cite it on E1a to E5 and V6.
- Bridge gap, not a uma row: the aps bridge sends text replies only, so
  WhatsApp templates (C8, T5) are unreachable through `--type uma` (F6).

### Gate (b) facts

Verified 2026-10-13.

- uma has no tags: `git tag` is empty on `main` and on the integration branch.
- `hop.top/uma` is private and unresolvable to an anonymous consumer: GitHub
  answers 404 to `hop-top/uma`, and `proxy.golang.org` answers 404 for it.
- The vanity server answers `go-import` at `hop.top/uma` only. Requests for
  `hop.top/uma/adapter/messaging/twilio` and `.../meta` return the site page
  with no `go-import` tag. A tagged public release still would not resolve the
  adapter submodules until the server answers subpaths.
- aps imports uma through a local replace block (`go.mod:178-196`) pinned to a
  sibling checkout of `main`. A build outside the hub layout fails.
- aps needs three uma modules today (`hop.top/uma`,
  `hop.top/uma/adapter/messaging/twilio`, `.../meta`). The integration work
  adds `relay.messages` (root module) and `http.mail` (`hop.top/uma/adapter/mail`).

Owner prerequisites, none done here: merge the integration work into uma
`main`; tag each module aps imports; make the repository public or configure
private-module access; make the vanity server answer `go-import` for adapter
subpaths; replace the aps replace block with tagged versions. Tags, releases
and vanity changes are outside this preparation.

### Gate (c) detail

Can an operator move each native shape today, with the build aps ships?

| aps-native shape | uma adapter | In the aps bridge | Moves today |
| --- | --- | --- | --- |
| `sms`, `--provider twilio` | `twilio.sms` | yes | yes; JSON-bodied relays need uma integration |
| `whatsapp`, `--provider twilio` | `twilio.whatsapp` | yes | yes; media staging, JSON bodies need integration |
| `whatsapp`, `--provider whatsapp-cloud` | `meta.whatsapp` | yes | yes, without templates (F6) |
| `sms` or `whatsapp`, `--provider generic` | `relay.messages` | no | no |
| `email` relay (outside retirement scope) | `http.mail` plus a sender | no | no |

The [migration note](../../user/uma-migration.md) tells operators how to move.
Nothing here enumerates their services: the inventory of operator yamls
(`services/*.yaml` under the aps data dir) is the owner's to collect.

## 1. SMS over Twilio

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| S1 | HMAC-SHA1 over URL plus sorted params, constant time | `cm/sms.go:36-80` | `T/signature.go:37-60`; `D/twilio.sms.md:89-91` | n/a | GREEN |
| S2 | Bad or missing signature refused before any action | `am/handler.go:208-223`, `:1259-1263` | `Ts/webhook.go:80-88,102`; `D/twilio.sms.md:93-100` | n/a | GREEN |
| S3 | Signed URL: `webhook_url`, else rebuilt from forwarded headers | `cm/sms.go:45`; `am/handler.go:1388-1409` | `Ts/webhook.go:57,76-79`; `Ts/config.go:68-74` | n/a | GREEN, M1 |
| S4 | Auth token from option, env or `secret:` ref | `cm/sms.go:101-119` | `Ts/webhook.go:115-131` | n/a | GREEN, M4 |
| S5 | Form inbound: `From`, `To`, `Body`, `MessageSid` | `am/normalizer.go:675-733`; `cm/sms.go:256-311` | `Ts/webhook.go:133-201` | n/a | GREEN |
| S6 | JSON-bodied webhook, `bodySHA256` in the URL checked | `cm/sms.go:53,82-99`; native refuses it (F5) | `T/signature_json.go:49-77`; `Ts/webhook.go:80`; `D/twilio.sms.md:110-129` | n/a | GREEN `[i]` |
| S7 | Inbound MMS media reaches the action | `am/normalizer.go:718-730`; `cm/sms.go:446-459` | `Ts/media.go:19-35`; `T/stage.go:44-88` | no | WAIVED, M8 |
| S8 | Reply inline as TwiML; empty `<Response/>` when none | `am/handler.go:370-373,381-386,1228-1238` | `Ts/webhook.go:110-111`; `Ts/send.go:26-58` | yes | GREEN, M5 |
| S9 | Messages REST outbound, basic auth (not on the SMS path, F2) | `cm/sms.go:332-384` | `Ts/send.go:26-58`; `D/twilio.sms.md:131-136` | yes | GREEN |
| S10 | REST failure classes: 429, 5xx retriable; 4xx not | `cm/sms.go:374-377` | `D/twilio.sms.md:131-136,176-179`; `Ts/send.go:60-85` | yes | GREEN |
| S11 | Sender: one `from` number per service | `core/service.go:658-660`; `cm/sms.go:219` | `Ts/config.go:25-26,62-67` | yes | GREEN |

- S1: uma checks through twilio-go's validator, constant time
  (`T/signature.go:32-33`); a repeated form parameter is refused.
- S2: uma answers 403, and the sink is never called before verification.
  aps answers 401.
- S3: `webhook_url` is required (503 without), carries no query in config, and
  the request's query is appended when signing.
- S4: credential purpose `twilio.webhook`, resolved by the host, never from env.
- S6: URL signature and body hash both checked in constant time; a JSON body
  without exactly one valid `bodySHA256` is 400. Tests: Twilio SDK vector
  `T/signature_json_test.go:45`, webhook `Ts/webhook_json_test.go:47`.
- S7: media downloaded from the account's Media resources with the REST
  credential, staged by media SID, at most 10. Fake only (V1 records none).
- S8: the 3 cassette sets are `send-and-fetch`, `rejected`, `refused-credentials`.
- S10: 429 `rate_limited`, refused credentials `unauthorized`, 5xx after send
  `outcome_unknown` (never blind-resent); cassettes `rejected`,
  `refused-credentials`, `reconcile-sent`, `reconcile-none`.
- S11: `from` or `messaging_service_sid`.

## 2. Generic relay (any provider, JSON in and out)

aps: `provider: generic` on `sms` and `whatsapp` services.

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| G1 | JSON in, JSON reply out in the webhook answer; no uma provider adapter | `am/handler.go:375,1346-1363`; `core/service.go:655,673` | `R/webhook.go:54-93`; reply by callback `R/send.go:70-111`; `D/relay.messages.md:37-65` | n/a | AMBER `[i]`, M10 |
| G2 | Generic webhook auth: bearer, token, hmac, ed25519, timestamp, replay | `cm/validation.go:244-404` | `W/config.go:13-116,128-190`; `Dx/webhook-auth.md:51-94`; `R/config.go:213-240` | n/a | GREEN `[i]`, M10 |

- G1: inbound keeps the aps field names (`from`, `to`, `body`, `message_id`,
  `media`; `D/relay.messages.md:154-165`). The reply leaves by a callback the
  operator's gateway serves, never in the webhook answer. A gateway that
  reads the reply from the answer must change. No `reconcile`: a lost callback
  answer stays `outcome_unknown`. Test: `am/sms_twiml_test.go:129` (aps side).
- G2: uma's `webhookauth` expresses the aps conventions
  (`Dx/webhook-auth.md:84-94`). Differences: no unauthenticated mode, secrets
  under 16 bytes refused unless `MinSecretBytes` is set, replay store in
  memory unless the host supplies one (`Dx/webhook-auth.md:109-125`).

## 3. WhatsApp Cloud (Meta)

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| C1 | Verification: GET, `hub.mode`, verify token, `hub.challenge` echoed | `cm/whatsapp.go:50-79`; `am/handler.go:156-158,673-689` | `Mw/webhook.go:187-202`; `M/webhook.go:57-72` | n/a | GREEN |
| C2 | `X-Hub-Signature-256` HMAC-SHA256 over the raw body | `cm/whatsapp.go:25-46`; `core/service.go:687-689` | `Mw/webhook.go:215-223`; `M/webhook.go:27-47` | n/a | GREEN, M2 |
| C3 | Only the configured `phone_number_id` accepted | `cm/validation.go:231-235` | `Mw/webhook.go:248` | n/a | GREEN |
| C4 | Inbound text: sender, profile name, `wamid`, timestamp | `cm/whatsapp.go:297-365`; `am/normalizer.go:735-808` | `Mw/webhook.go:302-377` | no | WAIVED |
| C5 | Reply context: `context.id` becomes `thread.id` | `cm/whatsapp.go:359-363` | `Mw/webhook.go:312-314` | n/a | GREEN, M3 |
| C6 | Inbound media, caption as text | `cm/whatsapp.go:568-588` | `Mw/webhook.go:354-362`; `Mw/media.go:40-92` | no | WAIVED, M8 |
| C7 | Outbound text: `POST {base}/{phone_number_id}/messages` | `cm/whatsapp.go:380-448` | `Mw/send.go:98-116,128-159` | no | WAIVED |
| C8 | Outbound template; reply text dropped; `template_required` | `cm/whatsapp.go:164-176,395-402`; `am/handler.go:986-995` | `Mw/send.go:176-187`; `D/meta.whatsapp.md:98-107` | no | WAIVED, M6, F6 |
| C9 | Graph failures: 429, 5xx retriable, message surfaced | `cm/whatsapp.go:438-441,613-619` | `Mw/outcome.go:82`; `D/meta.whatsapp.md:38` | no | WAIVED |

- C1: unreachable for a native service through `aps serve` (F1). `GET` is
  answered by the uma receiver at the same URL.
- C2: the secret is mandatory in uma (401 without a valid signature); aps
  only warns when none is set (`core/service.go:687-689`).
- C3: uma skips other numbers' changes and answers 200.
- C4: fixtures are written to the documentation (`D/meta.whatsapp.md:123-130`).
- C5: uma keeps `context.from` and `context.id` in the inbound payload, not as
  an envelope thread; the bridge maps it back. Native test:
  `am/handler_test.go:171`.
- C6: the token goes to the Graph and media hosts only; staged as attachment
  when the sink stages (`D/meta.whatsapp.md:85-90`).
- C8: uma sends part `meta.whatsapp.template` (`name`, `language.code`,
  `components` passed through). `TestRecordedTemplateSend` skips. The aps flags
  are `cli/service/add.go:69-71,282-286`.
- C9: outcomes `rejected`, `rate_limited`, `unauthorized`, `outcome_unknown`
  after a possible send, never resent; no `reconcile` (`D/meta.whatsapp.md:38`).

## 4. WhatsApp over Twilio

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| T1 | Same `X-Twilio-Signature` check as SMS when `provider: twilio` | `cm/validation.go:122-130,151-160` | `Tw/webhook.go:61-92` | n/a | GREEN |
| T2 | Inbound text; `whatsapp:+1555` addresses | `cm/whatsapp.go:259-295` | `Tw/webhook.go:143-205,239-269` | n/a | GREEN, M3, M7 |
| T3 | Inbound media reaches the action as attachments | `cm/whatsapp.go:269`; `cm/sms.go:446-459` | `Tw/media.go:19-35`; `T/stage.go:44-88`; `D/twilio.whatsapp.md:101-115` | no | WAIVED `[i]`, M8 |
| T4 | Outbound text through the Messages API, not TwiML | `cm/whatsapp.go:464-523`; `am/handler.go:691-779` | `Tw/send.go:48-90` | no | WAIVED |
| T5 | Outbound template: `content_sid` plus variables (F3) | `cm/whatsapp.go:479-491` | `Tw/send.go:24-38,154-174`; `Tw/window.go` | no | WAIVED, F6 |

- T1: the JSON-bodied variant is S6 (`Tw/webhook.go:84`).
- T2: prefix stripped in uma, a non-WhatsApp address is 400; buttons and
  locations are reported as typed payloads.
- T3: Twilio media URLs are not kept; items are downloaded with the REST
  credential and staged by media SID, at most 10. A failure a redelivery may
  fix answers 503. Fake only; `TestRecordedSendAndFetch` skips.
- T5: 24-hour window enforced when protocol state is present
  (`D/twilio.whatsapp.md:81-91`); a superset of aps. Not reachable through the
  aps bridge (F6).

## 5. Email relay

aps does not poll a mailbox: an external bridge (IMAP poller, MTA hook,
inbound-parse webhook) POSTs one JSON object per message and sends the reply
JSON itself (`docs/MESSENGERS_OVERVIEW.md` "Email"). Outside the retirement
scope (SMS and WhatsApp); graded because this checklist covers it.

| # | Behaviour | aps | uma | Rec | Status |
| --- | --- | --- | --- | --- | --- |
| E1a | Mail in from a mailbox or MX; uma replaces the bridge | `am/normalizer.go:597-658` | `D/imap.mail.md:93-142`; `D/smtp.mail.md:89-122` | no | WAIVED, M9 |
| E1b | Mail in by HTTP POST of `{from,to,subject,body}` | `am/normalizer.go:597-608` | `D/http.mail.md:103-139`; `Ml/httpmail/handler.go:44-114` | no | AMBER `[i]`, M11 |
| E2 | `from` to sender, `to` to channel, `body` to text | `am/normalizer.go:598-628` | `D/smtp.mail.md:89-119`; `D/imap.mail.md:93-142` | no | WAIVED |
| E3 | `subject` is the thread id | `am/normalizer.go:630-636` | `D/smtp.mail.md:98-100` | no | WAIVED, M3 |
| E4 | Attachments `[{type,url,mime_type,size_bytes}]` reach the action | `am/normalizer.go:638-655` | `Ml/stage/stage.go:66-91`; `D/http.mail.md:221-236` | no | AMBER `[i]`, M8, M11 |
| E5 | Reply reaches the sender (aps: reply JSON to the bridge) | `am/normalizer.go:901-910` | `D/smtp.mail.md:124-130`; `D/imap.mail.md:144-152` | no | WAIVED |

- E1a: `imap.mail` and `smtp.mail` `receive`; `google.mail` and
  `microsoft.mail` `receive` and `receive_http`. No recordings (V4 to V6).
  Message identity is destination plus a digest of the bytes, not the
  `Message-ID` (`Ml/smtp/envelope.go:127-128`, `Ml/imap/ingress.go:160-164`),
  on the integration branch only.
- E1b: `http.mail` takes JSON (default members `from`, `to`, `subject`,
  `body`, so the aps bridge payload fits) and `message/rfc822`. Auth is
  mandatory (`D/http.mail.md:72-101`). Sender evidence is `unverified` unless
  `evidence.trusted_authserv_id` is declared (`D/http.mail.md:161-198`). Form
  bodies (SendGrid Inbound Parse, default Mailgun route) are refused. Receive
  only: no reply path (`D/http.mail.md:38`). Hosted-service recipes are from
  their documentation, untried (`D/http.mail.md:92-101`); no record path.
- E3: uma threads by `In-Reply-To` then `References`.
- E4: bytes are staged for `smtp.mail` (`D/smtp.mail.md:101-111`), `imap.mail`
  (`D/imap.mail.md:130-139`), `google.mail` (`D/google.mail.md:147-159`) and
  `microsoft.mail` (`D/microsoft.mail.md:141-156`), at most 10 of 25 MiB each,
  on the integration branch only. `http.mail` describes attachments and keeps
  a claimed `url`; it stages nothing (`D/http.mail.md:221-236`).
- E5: uma sends itself, from `smtp.mail`, `imap.mail`, `google.mail` and
  `microsoft.mail`. The `data` payload has no mail form. `http.mail` cannot send.

## 6. Recorded evidence

Retirement needs recordings against real accounts, not only fakes. From
`Dx/recording/README.md:24` ("Cassettes are committed for GitHub issues and
Twilio SMS only") and `D/README.md:147-169`.

| # | Adapter | State | Status |
| --- | --- | --- | --- |
| V1 | `twilio.sms` | 7 cassette sets under `T/sms/testdata/cassettes/twilio/`; inbound webhook and media download not recorded | GREEN |
| V2 | `twilio.whatsapp` | harness only, none recorded (`D/twilio.whatsapp.md:131-135`) | WAIVED. Recording task skipped by owner |
| V3 | `meta.whatsapp` | harness skips, none recorded (`D/meta.whatsapp.md:123-130`) | WAIVED. Recording task skipped by owner |
| V4 | `google.mail` | `TestRecordedGmail` skips (`D/google.mail.md:187-193`) | WAIVED. Recording task skipped by owner |
| V5 | `microsoft.mail` | `TestRecordedGraph` skips (`D/microsoft.mail.md:179-185`) | WAIVED. Recording task skipped by owner |
| V6 | `imap.mail`, `smtp.mail`, `http.mail` | no record path (`Dx/recording/imap-smtp.md`, `D/README.md:152-156`) | WAIVED. Live evidence task in progress |

- V1: sets `send-and-fetch`, `reconcile-sent`, `reconcile-none`, `rejected`,
  `refused-credentials`, `fetch-unknown`, `history-reply`. The inbound webhook
  is checked against Twilio's vector; media against a fake.
- V2: `testdata/` holds hand-written signed webhook forms only.
- V6: xrr cannot record IMAP or SMTP line protocols. A separate uma task is
  building live IMAP and SMTP evidence against a local mail container; cite
  it here if it lands.
- Waiver: V2 to V5 are WAIVED by the owner, 2026-10-13; see Gate (a) detail.
- Premise check: `twilio.sms` is the only adapter with real cassettes among
  the SMS, WhatsApp and mail adapters. `github.issues` also has them
  (`uma:adapter/ticket/github/testdata/cassettes`), so "only adapter in uma"
  is wrong; "only one on this list" is right. `relay.messages` needs none: the
  relay is the operator's own service (`D/README.md:164`).

## Gap names

Each row that was RED maps to one uma task whose title carries the name below and whose
description cites this file and the row number. The first six were answered
on the integration branch; State says what remains.

| Gap | Rows | uma task title | State |
| --- | --- | --- | --- |
| `twilio-json-webhook` | S6, T1 | twilio.sms and twilio.whatsapp: accept JSON-bodied webhooks with bodySHA256 | answered `[i]` |
| `generic-relay` | G1 | Generic JSON message relay adapter for SMS and WhatsApp providers | answered `[i]`; callback-only reply |
| `generic-webhook-auth` | G2 | Generic webhook auth for receive_http adapters | answered `[i]` |
| `twilio-whatsapp-media` | T3 | twilio.whatsapp: stage inbound media through the attachment sink | answered `[i]`; recording waived |
| `mail-http-ingress` | E1b | Mail HTTP ingress adapter for inbound-parse and MTA-hook relays | answered `[i]`; receive only |
| `mail-attachments` | E4 | Mail adapters: stage inbound attachments through the sink | answered `[i]`; `http.mail` stages nothing |
| `imap-smtp-evidence` | V6 | imap.mail and smtp.mail: live-server parity evidence without cassettes | in progress, separate from the waived recordings |

V2 to V5 rode on the recording task, now skipped; no new task. Two residuals are
named by uma as not built and have no task: `http.mail` `send`
(`D/http.mail.md:38`) and `http.mail` attachment staging.

## Migration and bridge notes

Config translation the bridge (or the operator) must do. None is a uma gap.

| # | Note |
| --- | --- |
| M1 | `webhook_url` mandatory, no query or fragment. aps derives it from forwarded headers when unset. A console URL with a query: configure the base URL; uma appends the request's query |
| M2 | Meta app secret mandatory. aps accepts unsigned Cloud webhooks when none is set |
| M3 | Session identity inputs shift: Twilio loses the `whatsapp:` prefix, email thread moves from subject to headers, WhatsApp reply context moves from envelope to payload. History at cutover is a separate decision, not stated (`cm/types.go:116-163`) |
| M4 | aps option names (`twilio_auth_token`, `signature_secret`, `app_secret`, `verify_token`, `WHATSAPP_ACCESS_TOKEN`) map to uma purposes `twilio.webhook`, `twilio.rest`, `meta.app_secret`, `meta.webhook.verify_token`, `meta.graph` |
| M5 | Replies change from inline TwiML (SMS) to asynchronous Messages API calls: REST credential and `from` or a Messaging Service required; status callbacks arrive at `webhook_url` |
| M6 | `template_name`/`language_code` become a per-reply `meta.whatsapp.template` part the bridge must build. aps sends the template with no components and drops the text. The bridge does not build it yet (F6) |
| M7 | Twilio `from` is configured without `whatsapp:` in uma |
| M8 | Media staging needs an attachment sink (the aps bridge supplies one under the data dir) and, for WhatsApp windows, protocol state; without them media is summary-only |
| M9 | The email bridge retires once uma holds the mailbox or MX; generic webhook auth (G2) is no longer used for email |
| M10 | Generic relay: the gateway serves a callback URL (host in `CallbackHosts`, `https`, no query) and authenticates webhooks via `auth`; form bodies, flat `NumMedia` keys unread. `relay.messages` is not in the aps bridge catalogue |
| M11 | Email via `http.mail`: receive only, so replies need a sending channel; form bodies refused; sender evidence `unverified` by default. `http.mail` is not in the aps bridge catalogue |

## Retained by aps

Not uma concerns. The bridge must keep calling them on bridged traffic.

| Behaviour | aps |
| --- | --- |
| Allowlists `allowed_numbers`, `allowed_senders` (exact or `*@domain`), `allowed_channels`, `allowed_chats`. uma has none; `smtp.mail` adds SPF/DKIM/DMARC evidence | `cm/validation.go:208-242`; `core/allowed_senders.go:9-60` |
| Reply policy `reply: none/text/auto`; empty or failed output sends nothing | `am/handler.go:381-386,1038-1047`; `core/service.go:816-830` |
| Routing: `default_action`, sender route table, execution mode `chat` | `am/adapter.go:187-222`; `am/handler.go:818-842` |
| Conversation id, session id, `prior_turns`, outbound turn recording | `cm/types.go:116-163`; `am/handler.go:1017-1026` |
| Action input shape incl. `platform_metadata` (raw payload in aps, typed envelope payload in uma) | `am/normalizer.go:715,798`; `docs/MESSENGERS_OVERVIEW.md` "Normalized Message Format" |

## aps findings

Surprises found while verifying. Not uma gaps; fixes belong in aps.

| # | Finding | Evidence |
| --- | --- | --- |
| F1 | WhatsApp Cloud GET verification is unreachable natively via `aps serve`: GET answers 405 (probed). A `uma` service answers it: 200, challenge echoed | `am/adapter.go:126-132`; `am/handler.go:156-158`; `docs/user/messengers.md:279-280` |
| F2 | Messages REST for SMS (`SMSProvider`, `TwilioSMSTransport`) is on no production path; SMS replies are TwiML only. `core/service.go:530` says "provider delivery" | `cm/sms.go:176`; `tools/messengermd/main.go:112`; `am/handler.go:370-373` |
| F3 | Twilio WhatsApp `content_sid` is unreachable from a service config: `template_required` passes and the message still goes out free-form | `cm/whatsapp.go:171,479-491`; `am/handler.go:986-995` |
| F4 | Cloud status-only notifications fail normalisation and answer 400 (probed), so Meta retries them. uma maps them to status envelopes and answers 200 | `cm/whatsapp.go:306-308`; `am/handler.go:1115-1120`; `Mw/webhook.go:268-276` |
| F5 | The `bodySHA256` JSON path has no aps test; a Twilio-signed JSON request gets 401 from a native service (body parsed as a query string; probed) | `cm/sms.go:49-52`; `am/handler.go:1365-1374`; no `bodySHA256` hit in any aps `_test.go` |
| F6 | A `uma` service ignores `template_name`, `language_code`, `template_required`: the bridge sends text only and `service add` accepts the flags silently | `internal/adapters/umabridge/provider.go` (`DeliverMessage`); `core/service_uma.go` |

## Line-ref notes

Earlier line hints for these paths had drifted or named the wrong file.
`am/handler.go:381` (`twilioSMSReply`) and `am/normalizer.go:597`
(`normalizeEmail`) are right. The handler is `adapters/messenger/handler.go`,
not under `core/`. `X-Hub-Signature-256`, the verify token, WhatsApp media and
reply context live in `cm/whatsapp.go` (`:16`, `:25-79`, `:359-363`,
`:568-588`); `am/normalizer.go:668-1106` holds only normalisation and
denormalisation (`:668-808`, `:901-929`, `:1067-1090`). Cloud transport is
`cm/whatsapp.go:380-448` (template `:395-402`), Twilio transport `:464-523`.
The `uma` bridge commits shifted `am/handler.go` by 8 to 20 lines,
`am/adapter.go` by 62 and `core/service.go` by 4 to 7; the cites above are the
shifted ones.
