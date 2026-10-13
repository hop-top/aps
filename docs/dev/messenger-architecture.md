# Messenger Architecture

Implementation details for APS message services and the legacy messenger
adapter-device pipeline.

## Current User-Facing Surface

New message integrations are profile-facing services:

```bash
aps service add <service-id> --type <adapter-alias> --profile <profile-id> [options]
```

Message adapter aliases are expanded through kit aliasing and persisted as
canonical service config.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/servicemd", "persisted-adapters"],
    text=True))
]]] -->
| Input | Persisted type | Persisted adapter |
| --- | --- | --- |
| `--type message --adapter telegram` | `message` | `telegram` |
| `--type discord` | `message` | `discord` |
| `--type slack` | `message` | `slack` |
| `--type sms` | `message` | `sms` |
| `--type teams` | `message` | `teams` |
| `--type telegram` | `message` | `telegram` |
| `--type uma` | `message` | `uma` |
| `--type whatsapp` | `message` | `whatsapp` |
<!-- [[[end]]] -->

Ticket aliases share the same command grammar but resolve to `ticket`, not
`message`: `email`, `github`, `gitlab`, `jira`, and `linear`. Ticket services
are mounted by the ticket HTTP adapter (`internal/adapters/ticket`) at
`POST /services/<id>/ticket/<adapter>`; see [Ticket Service Flow](#ticket-service-flow).

## Message Service Flow

```text
[Platform webhook or relay]
        |
        v
POST /services/<service-id>/webhook
        |
        v
[Normalize provider payload]
        |
        v
[Resolve messenger link mapping, sender route table, or default action]
        |
        v
[Execute profile action]
        |
        v
[Return platform-shaped JSON]
```

`aps serve` mounts the service route through the messenger HTTP adapter. The
route loads the persisted service, confirms `type: message`, and uses the
persisted adapter to pick the normalizer/denormalizer.

## Ticket Service Flow

```text
[Mail relay / IMAP poller / Jira / Linear / GitLab webhook]
        |
        v
POST /services/<service-id>/ticket/<adapter>
        |
        v
[Load service; require type: ticket and matching adapter]
        |
        v
[Generic service request auth: bearer/token/HMAC, timestamp, replay]
        |
        v
[Normalize adapter payload -> NormalizedTicket; allowed_senders gate]
        |
        v
[Resolve sender route table or default_action]
        |
        v
[Execute profile action: NormalizedTicket + service_id + route_key on stdin]
        |
        v
[Return status JSON: status, body (action stdout), ticket_id, thread_id]
```

The ticket adapter (`internal/adapters/ticket`) mirrors the messenger
adapter: `Adapter.RegisterRoutes` mounts the per-service pattern, `Handler`
validates with the shared `core/messenger.ServiceValidator` (same options as
message services, no provider hooks), `Normalizer` produces the
`NormalizedTicket`, and `Router` resolves through the service
(`serviceRouteResolver`: route table by author email, else `default_action`)
and executes through `protocol.APSCore.ExecuteRun`. There is no unauthenticated
catch-all route; a ticket service without `default_action` or a routing block
answers `422`. User guide: [Ticket services](../user/tickets.md).

## uma Bridge

`--type uma` (`type: message`, `adapter: uma`) hosts a uma engine in-process
for one service instead of an aps-native provider. It is opt-in per service;
the native providers stay the default. User guide:
[uma-hosted providers](../user/messengers.md#uma-hosted-providers-opt-in).

```text
[Twilio / Meta webhook]
        |
        v
aps serve                   /services/<id>/webhook, request passed untouched
        |
        v
Bridge.Webhook()            uma verifies the signature, records the event
        |                   (message.received)
        v
bot companion (uma)         subscribed to message.received, one turn per message
        |
        v
messenger.Runtime           provider = bridge: NormalizeIngress, then the
        |                   router and executor aps already uses
        v
action reply  ---> DeliverMessage ---> uma Messages.Reply ---> delivery worker
                                                                  |
                                                                  v
                                                      provider REST (Messages API / Graph)
```

Code: `internal/adapters/umabridge`, the only package that imports
`hop.top/uma`. The static contract of each uma adapter as a service (options,
env bindings) lives in `internal/core/service_uma.go`, so `service add`
validates without linking uma.

- **Engine.** One per service: memory store, an authorizer that admits only
  the bridge's two actors (`aps-bridge` provisions the channel and inbox and
  runs the receiver and delivery worker; `aps-bridge-bot` reads the
  subscription and replies) for a fixed action list and denies everything
  else, `ports.Credentials` over the service's `--env` bindings, an HTTP
  client, and a file-backed media sink when `Config.MediaDir` is set.
- **Credentials.** A uma credential purpose maps to one env binding
  (`twilio.rest` and `twilio.webhook` to `TWILIO_AUTH_TOKEN`; `meta.graph`,
  `meta.app_secret`, `meta.webhook.verify_token` to `WHATSAPP_ACCESS_TOKEN`,
  `WHATSAPP_APP_SECRET`, `WHATSAPP_VERIFY_TOKEN`). The binding value is a
  literal or `secret:NAME`, resolved with `core.ResolveSecretValue` on every
  request; nothing is cached or logged by the bridge.
- **Normalization.** The received uma message becomes a `NormalizedMessage`
  with `platform` `sms` or `whatsapp`, the provider message id as `id`,
  `messenger_name` set to the service id (routing keys on it), the sender
  address as `sender.id` and the receiving number or phone-number id as
  `channel.id`. Twilio WhatsApp ids keep their `whatsapp:` prefix, as the
  native provider has them. Inbound media is copied to
  `<media dir>/inbound/<message>/<n>` and listed as `file://` attachments.
- **Reply.** The runtime's `DeliverMessage` calls `Messages.Reply` on the
  turn's conversation with the turn's own operation id, so a repeated
  delivery replays instead of sending twice. The receipt is `accepted` (uma
  queued it); the engine's worker performs the send, and its
  `outcome_unknown` handling (reconcile before any resend) is uma's.
- **Errors.** The runtime's errors are final (it already retried delivery),
  so the bot turn drops them after `Config.OnResult` has seen them. Only an
  engine failure before the action ran is left to the bot's own retry.
- **Seams the host wires.** `internal/adapters/messenger/uma_mount.go` builds
  one Bridge per uma service and fills `Config`: `Router`/`Executor` are the
  messenger adapter's own `MessageRouter` and `serviceRuntimeExecutor` (route
  table or `default_action`, reply policy, chat mode, prior turns), `History`
  the adapter's lazily opened conversation store (the same one the native
  providers record to), `Hooks.OnDeliveryAttempt` and `OnResult` the service
  bookkeeping (`last_inbound`/`last_outbound`, delivery attempts, failures to
  the log), `MediaDir` `<data dir>/messenger/uma/<service-id>`. `Options` is
  the test seam for provider origins (`messenger.WithUmaOptions`).
- **Allowlists.** `allowed_numbers`, `allowed_senders`, `allowed_channels` are
  aps's: the bridge runs the native `ServiceValidator.ValidateMessage` on the
  normalized message, before media is copied out of the engine. A refusal is
  dropped, not retried; uma has already answered the provider.

### Webhook mount

`Adapter.RegisterRoutes` registers `/services/{service}/webhook` for every
method and dispatches in the handler:

- POST, any service: unchanged checks (404 unknown, 400 wrong type or no
  adapter), then the native handler. A uma service goes to
  `Handler.umaWebhook` instead (`WithUmaWebhook`), which is
  `umaMounts.serve`. A handler built without it answers `501` for a uma
  service.
- GET and every other method: a uma service is dispatched like a POST, so
  the provider's own methods reach its receiver (WhatsApp Cloud's GET
  subscription verification; the receiver answers `405` for what it does not
  serve). Any other service gets the mux's own refusal, replicated exactly
  (`405`, `Allow: POST`).

`umaMounts.serve` passes the request to `Bridge.Webhook()` untouched, so the
raw body, query and headers are what uma verifies; aps middleware that wraps
the adapter mux (request id, recovery, optional bearer auth) sees it first and
does not read the body. No aps signature check runs for a uma service: the
native `ValidateRequest` path is not entered.

- **Public URL.** uma verifies Twilio over `webhook_url` (mandatory, no query)
  plus the request's own query; the host, scheme and `X-Forwarded-*` headers
  the request carried are not consulted. The route's path is not part of the
  signature either, so a proxy path rewrite is harmless as long as
  `webhook_url` is the URL Twilio is configured with.
- **Body limit.** uma bounds the body (1 MiB default, `413`) before the adapter
  reads it; the native routes have no limit. The server's 15 s read and write
  timeouts bound the rest.
- **Lifecycle.** Services are loaded per request, like the native routes, so a
  bridge is built on a service's first request and kept in `umaMounts`,
  keyed by service id with a fingerprint of the provider-facing configuration
  (type, adapter, profile, options, env bindings, delivery). A changed
  fingerprint retires the bridge in the background and builds a new one;
  `last_inbound`/`last_outbound` writes and routing, which is read per message,
  are not part of it. A bridge that fails to build answers `503` (nothing about
  the cause in the body) and is retried on the next request. `Adapter.Stop`
  closes every bridge before the history store, waiting up to 10 s.
- **Not recorded by the bridge.** An event refused by the allowlist or failed
  before delivery leaves no service event, as on the native providers; the
  log carries it.

Not done here: mapping uma conversations onto aps session history (history is
the minimal existing wiring: the router records the inbound turn, the runtime
the delivered reply, keyed as the native SMS/WhatsApp providers key them),
durable uma state across restarts (the store is in memory, so provider-event
de-duplication also resets), retention for staged media, and retiring the
native SMS and WhatsApp transports.

### The uma dependency

hop.top/uma is unreleased and its repository is private, so `go.mod` points
at a sibling checkout with `replace` directives (one commented block at the
end of `go.mod`) and pins the matching pseudo-versions. The relative paths
resolve from `hops/<group>/<name>` worktrees. Until uma is resolvable, a
build from any other layout, CI, and any consumer fails. To drop the
dependency, delete `internal/adapters/umabridge`, the `uma` alias and
`internal/core/service_uma.go`, and the block in `go.mod`.

## Routing

For service routes, the service ID becomes the route key and the service profile
is the default profile context:

```bash
aps service add support-bot \
  --type slack \
  --profile assistant \
  --allowed-channel C01ABC2DEF \
  --default-action triage \
  --reply text
```

If `default_action` has no profile separator, APS expands it to
`<service-profile>=<action>`. Explicit cross-profile routing can still use
`profile=action`.

When one number or bot serves many organizations, replace `default_action`
with a sender route table (`--route-table`, optional `--contacts`): routes are
matched against the normalized sender key or the resolved contact, in declared
order, and must end with a terminal `match: unknown` fail-safe. See
[Message routing: sender route tables](message-routing.md).

The older adapter-device pipeline stores channel mappings in
`messenger-links.json`:

```bash
aps adapter messenger link add my-slack \
  --profile assistant \
  --mapping "C01ABC2DEF=assistant=triage"
```

That pipeline is still useful for subprocess devices, but service docs should
not use the removed `aps profile link-messenger` or `aps messengers create
--template` forms.

## Normalized Message Format

All message adapters normalize to:

```json
{
  "id": "msg_unique_id",
  "platform": "telegram|discord|slack|teams|sms|whatsapp",
  "profile_id": "assistant",
  "timestamp": "2026-05-11T10:30:00Z",
  "sender": {
    "id": "user_id",
    "name": "display_name",
    "platform_handle": "handle",
    "platform_id": "platform_user_id"
  },
  "channel": {
    "id": "channel_id",
    "name": "channel_name",
    "type": "direct|group|broadcast|topic",
    "platform_id": "platform_channel_id"
  },
  "text": "message content",
  "thread": { "id": "thread_id", "type": "reply" },
  "attachments": [],
  "platform_metadata": {}
}
```

`github` and `email` still exist in the lower-level messenger normalizer for
older message-adapter code, but user-facing service aliases route them to
`ticket` services, whose email normalizer accepts the same flat email JSON.

## Conversation And Thread Policy

Message services derive APS conversation/session state from the normalized
service/platform/channel/user/thread fields. The stable `ConversationID` is the
outer channel or direct-message pair; the stable `SessionID` narrows that
conversation to a platform thread when one exists.

Direct-message and phone sessions include both the receiving channel/number and
the sender ID, so two SMS or WhatsApp senders using the same receiving number
do not share multi-turn state. Group/channel messages without a platform thread
continue in the channel-root session; replies with `thread.id` continue in the
platform thread session.

Attachments, mentions, commands, and unsupported event behavior are defined in
[Message conversation and thread policy](message-conversation-policy.md).

Turns are persisted per conversation in `<data-dir>/messages/conversations.db`;
routed actions receive `conversation` and session-scoped `prior_turns` on stdin
and `RunInput.ThreadID` set to the session key. `aps service conversation
list|show` queries the store. See the Thread History section of the policy.

## Adapter Support

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/messengermd", "arch-support"],
    text=True))
]]] -->
| Adapter | Normalize support | Denormalize support | Service maturity |
| --- | --- | --- | --- |
| Discord | message-create style JSON | content response JSON | Ready when mounted with `aps serve`; Gateway client is external |
| Slack | Events API event envelope JSON | text response JSON | Ready when mounted with `aps serve`; app verification is external |
| SMS | Twilio-style or generic phone fields in JSON/form | text response metadata | Ready for Twilio or JSON relays |
| Teams | Bot Framework `message` activity JSON | message activity JSON | Ready when mounted with `aps serve`; Azure Bot registration is external |
| Telegram | Bot API `message` and `edited_message` JSON | `sendMessage` JSON | Ready when mounted with `aps serve` |
| WhatsApp | Cloud API JSON or Twilio-style WhatsApp JSON/form | text/template response metadata | Ready for Cloud API and Twilio-compatible relays |
<!-- [[[end]]] -->

## Capability Matrix

Ingress, delivery, and feature capabilities as reported by each first-class
provider's live `Metadata()`; platforms without a first-class provider show
only their documented support status.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/messengermd", "capability-matrix"],
    text=True))
]]] -->
| Platform | Ingress modes | Delivery modes | Threads | Attachments | Reactions |
| --- | --- | --- | --- | --- | --- |
| Discord | `webhook`, `stream` | `text`, `file` | Yes | Yes | No |
| Email | JSON relay route through `aps serve` | — | — | — | — |
| Slack | `webhook` | `text`, `file` | Yes | Yes | No |
| SMS | `webhook` | `text` | No | No | No |
| Teams | `webhook` | `text` | Yes | Yes | No |
| Telegram | `webhook` | `text` | Yes | Yes | No |
| WhatsApp | `webhook` | `text`, `file` | Yes | Yes | No |
<!-- [[[end]]] -->

## Runtime And Testing

Implemented service commands:

```bash
aps service add <service-id> --type <type-or-alias> --profile <profile-id>
aps service show <service-id>
aps service routes <service-id>
aps serve --addr 127.0.0.1:8080
```

Message service routes are mounted at:

```text
POST /services/<service-id>/webhook
```

This is the only messenger HTTP route `aps serve` mounts. The platform-keyed
`/messengers/{platform}/webhook` entrypoint on `messenger.Handler` is a
service-less component path: it skips request validation (provider auth
hooks, generic webhook auth, allowlists), so it is deliberately not exposed
over HTTP. Reach a platform through a message service.

Use `aps adapter messenger test <device>` only for adapter-device route
simulation. It is not a service-route test and it does not verify live platform
delivery.

## Storage

Service configuration is stored under the APS data directory:

```text
${XDG_DATA_HOME:-~/.local/share}/aps/services/<service-id>.yaml
```

Legacy adapter-device links are stored with profile data as
`messenger-links.json`. Older `~/.aps/messengers/<name>` examples are
pre-service documentation and should not be used for new service examples.

## Security Considerations

- Bind `aps serve` to a private interface unless the service is intentionally
  exposed.
- Use `--auth-token` on `aps serve` when routes are reachable by untrusted
  clients.
- Treat message text and attachments as untrusted action input.
- Keep platform tokens in secret-backed environment bindings; `aps service add`
  stores the binding metadata, not a full platform app installation.
- Platform-specific signature verification and URL verification are adapter or
  relay responsibilities unless implemented on the service route.

## Related

- [Message routing: sender route tables](message-routing.md)
- [Agent messenger patterns](../agent/messenger-patterns.md)
- [User messenger guide](../user/messengers.md)
- [Service UX draft](service-ux-draft.md)
