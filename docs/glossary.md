# Glossary

One definition per APS messaging, session, service, and adapter term; the
overloaded words (session, messenger, thread, service) get a disambiguation
table below.

## Terms

| Term | Meaning | Where used | See |
| --- | --- | --- | --- |
| profile | Identity + config unit: YAML at `<data-dir>/profiles/<id>/profile.yaml` (persona, capabilities, scope, isolation, workspace link). | `aps profile ...`, `--profile` | [Profile schema](architecture.md#profile-schema) |
| profile action | Script under a profile; `aps action run <profile> <action>` feeds payload on stdin. Each routed message or ticket runs exactly one. | `aps action run`, `default_action` | [Routing exec](dev/message-routing.md#execution-and-isolation) |
| adapter action | Action declared in adapter manifest `config.actions`; `aps adapter exec <adapter> <action> --input k=v` passes inputs as `<PREFIX>_<INPUT>` env vars. | `aps adapter exec` | [Input contract](dev/adapters.md#input-contract) |
| adapter | Integration point between profiles and external systems; declared by `manifest.yaml` (`type`, `strategy`, `config.backend`, `config.actions`); enabled per profile via `aps adapter link`. | `aps adapter ...` | [Adapters](dev/adapters.md) |
| adapter type (kind) | Functional category: `actuator`, `desktop`, `messenger`, `mobile`, `protocol`, `scheduler`, `sense`. | manifest `type:`, `--type` on `aps adapter create` | [Adapter types](dev/adapters.md#adapter-types) |
| strategy | Adapter run mode: `builtin` (native Go, persistent), `script` (spawned per action, ephemeral), `subprocess` (managed child, persistent). | `strategy:`, `--strategy` | [Strategies](dev/adapters.md#loading-strategies) |
| backend | Tool an adapter delegates to; `config.backend` selects `backends/{{backend}}/` scripts (email default `himalaya`). Also `secrets.backend`, `voice.backends`: qualify. | `config.backend` | [Manifest](dev/adapters.md#the-manifest-manifestyaml) |
| adapter device | Legacy adapter instance (subprocess strategy); channel mappings in `messenger-links.json`. | `aps adapter messenger create\|start\|stop\|link\|test` | [Legacy devices](user/messengers.md#legacy-adapter-devices) |
| service | Profile-facing inbound route at `<data-dir>/services/<id>.yaml`: `type`, `adapter`, `profile`, `options`, optional `routing`. Types: message, ticket. | `aps service add\|show\|routes\|status\|test\|start\|stop` | [Service](#service) |
| message service | Service `type: message`: chat-like payloads at `POST /services/<id>/webhook`; normalize, dispatch one profile action, answer platform-shaped JSON. | `aps service add <id> --type <alias>` | [Messenger integration](user/messengers.md) |
| ticket service | Service `type: ticket` (`email`, `github`, `gitlab`, `jira`, `linear`): work-item events at `POST /services/<id>/ticket/<adapter>`, normalized to `NormalizedTicket`. | `aps service add --type jira` | [Tickets](user/tickets.md) |
| service alias | `--type telegram` persists `type: message`, `adapter: telegram` (kit aliasing); ticket aliases give `type: ticket`. | `--type <alias>`, `--type message --adapter <name>` | [Create a service](user/messengers.md#create-a-message-service) |
| service profile | Profile from `--profile` on `aps service add`: default routing context; where a bare `default_action` or route `action:` resolves. | service `profile:` | [Routing](dev/messenger-architecture.md#routing) |
| `aps serve` | APS HTTP server: mounts all message and ticket service routes plus REST API (`/profiles`, `/sessions`, `/a2a/...`); `--auth-token` = bearer auth. | `aps serve --addr <host:port>` | [REST API](architecture.md#rest-api-aps-serve) |
| messenger | Overloaded: adapter type, legacy adapter-device pipeline, or HTTP adapter behind message services. Qualify. | — | [Messenger](#messenger) |
| platform | Provider family: `telegram`, `discord`, `slack`, `teams`, `sms`, `whatsapp`; route key when no `service_id`. Not the `platform` isolation level. | normalized `platform` | [Identity](dev/message-conversation-policy.md#identity-inputs) |
| normalized message | `NormalizedMessage`: platform-independent JSON every message adapter emits (`id`, `platform`, `sender`, `channel`, `text`, ...); action stdin. | action stdin | [Shape](dev/messenger-architecture.md#normalized-message-format) |
| normalizer / denormalizer | Per-adapter pair picked from persisted `adapter`: provider payload to normalized message; reply to platform-shaped JSON. | message service flow | [Adapter support](dev/messenger-architecture.md#adapter-support) |
| platform_metadata | Provider extras on the normalized message: `service_id`, `messenger_name`, command/callback/mention IDs, stamped `routing` decision. | normalized `platform_metadata` | [Evaluation order](dev/message-routing.md#evaluation-order) |
| sender | Normalized `sender`: `id` (platform user ID, handle, or phone number), `name`, `platform_handle`, `platform_id`. | `sender.id`, `--allowed-number`, `--allowed-sender` | [Identity inputs](dev/message-conversation-policy.md#identity-inputs) |
| channel | Normalized `channel`: `id` (platform channel/chat ID; receiving number for SMS/WhatsApp), `name`, `type`, `platform_id`. | `channel.id`, `--allowed-chat`, `--allowed-channel` | [Identity](dev/message-conversation-policy.md#identity-inputs) |
| channel type | `channel.type`: `direct` (DM, phone), `group` (group, Slack channel, guild channel), `broadcast` (Telegram channel), `topic` (enumerated only). | `channel.type` | [Platform mapping](dev/message-conversation-policy.md#platform-mapping) |
| workspace_id | Platform workspace/server/team/guild; optional `:workspace:<id>` part of `ConversationID`; `--allowed-guild` target. Not the APS `--workspace`. | `workspace_id` | [Validation](MESSENGERS_OVERVIEW.md#validation-and-routing) |
| thread | Overloaded: platform thread reference, or APS session scope derived from it. Qualify. | — | [Thread](#thread) |
| conversation | Outer place messages arrive: a channel, or channel + sender pair for direct and phone messages. Groups persisted turns. | `conversation` on stdin | [Conversation ID](dev/message-conversation-policy.md#conversation-id) |
| ConversationID | `msgconv:v1:service:<service-or-platform>:platform:<platform>[:workspace:<id>]:channel:<id>[:sender:<sender-id>]`; `:sender:` for DMs/phone. | `conversation_id` | [Conversation ID](dev/message-conversation-policy.md#conversation-id) |
| session | Overloaded: process session (`aps session`) or message session (multi-turn thread key). Qualify. | — | [Session](#session) |
| SessionID | `msgsess:v1:<conversation-parts>[:thread:<thread-id>]`: deterministic, URL-escaped per component, stable across restarts; also `RunInput.ThreadID`. | `session_id`, `--session` | [Session ID](dev/message-conversation-policy.md#session-id) |
| session registry | Local store under `<data-dir>/sessions/`: `{session_id: {profile_id, status, ...}}` per process session; source of `aps session list`. | `aps session ...` | [Session engine](architecture.md#session-engine-internalcoresession) |
| conversation scope | `conversation.scope`: `channel` (group, no `thread.id`), `thread` (has `thread.id`), `direct` (DM/phone); not the profile `scope:`. | `conversation.scope` | [Session ID](dev/message-conversation-policy.md#session-id) |
| turn | One recorded exchange, inbound message or delivered reply, in `message_turns` (`direction`, `sender_id`, `text`, ...); newest 500 per conversation kept. | conversation store | [Store](dev/message-conversation-policy.md#conversation-store) |
| inbound | Turn `direction` for a message routed to a profile action (action mode) or handed to the chat runtime (chat mode); unsupported events never become turns. | `direction: inbound` | [Store](dev/message-conversation-policy.md#conversation-store) |
| outbound | Turn `direction` for a reply actually sent; `sender_id` = profile ID, `message_id` = inbound message answered. Failed/empty/`--reply none`: no turn. | `direction: outbound` | [Store](dev/message-conversation-policy.md#conversation-store) |
| prior_turns | Stdin array of turns before the current message in the same `session_id`, oldest first; always present, empty when none; current message never included. | action stdin | [Contract](dev/message-conversation-policy.md#attachment-contract) |
| history_turns | Service option bounding `prior_turns`; `aps service add --history-turns N`; default 20. | `options.history_turns` | [Contract](dev/message-conversation-policy.md#attachment-contract) |
| conversation store | SQLite `<data-dir>/messages/conversations.db` (`message_turns`); opened lazily on first turn; errors never block routing or delivery. | `aps service conversation ...` | [Store](dev/message-conversation-policy.md#conversation-store) |
| execution mode | How a service runs the routed profile: action execution (script, stdin payload) or chat execution (`execution: chat`). | service `execution:` | [Execution](dev/message-routing.md#execution-and-isolation) |
| default_action | Option naming the action for every inbound message; bare name = `<service-profile>=<action>`; `profile=action` crosses profiles; `routing` wins. | `--default-action` | [vs routing](dev/message-routing.md#relationship-to-default_action) |
| route table | `routing` block: sender-keyed routes (`match`, `profile`, `action`); first match wins; last must be `match: unknown`; load failure = routing failure, no fallback. | `--route-table` | [Schema](dev/message-routing.md#route-table-schema) |
| route key | What a message routes under: `service_id`, else `messenger_name`, else `platform`. Ticket actions get it as `route_key` on stdin. | `platform_metadata.service_id` | [Identity inputs](dev/message-conversation-policy.md#identity-inputs) |
| sender key | `sender.id` normalized for matching: phone/mail prefix stripped, emails lowercased, phone separators dropped (`+` added on `sms`/`whatsapp`), else verbatim. | `match:`, `keys:` | [Keys](dev/message-routing.md#sender-key-normalization) |
| contact snapshot | YAML contacts (`--contacts` or `contacts:` in the table): `id`, `name`, `org`, `keys`; keys normalized like senders; enables `org:`/`contact:` matches. | `--contacts` | [Contacts](dev/message-routing.md#contacts-snapshot) |

## Session

| Phrase | Means | Key / store | Surface |
| --- | --- | --- | --- |
| process session | Managed process of a profile; tmux-backed, optionally in a platform sandbox; registry status `active`, `inactive`, `errored`. | session ID in the session registry | `aps session ...`, `GET /sessions` |
| voice session | Process-registry entry registered by `aps voice start` for metadata and routing; `type: voice`. | session registry | `aps session list --type voice`, [Voice](dev/voice.md) |
| message session | Multi-turn scope inside one conversation; what an action sees as `prior_turns`. | `SessionID` (`msgsess:v1:...`), turn `session_id` | `aps service conversation show <id> --session <session-id>` |
| `RunInput.ThreadID`, `RunState.thread_id` | Run-state correlation key of a routed action; equals the message `session_id`. | `SessionID` | [Attachment contract](dev/message-conversation-policy.md#attachment-contract) |

## Messenger

| Phrase | Means | Surface |
| --- | --- | --- |
| messenger (adapter type) | Adapter kind `messenger` ("Telegram, Slack, etc."), one of seven kinds. | manifest `type: messenger`, `aps adapter messenger create <name> --type messenger` |
| messenger device (legacy) | Adapter-device pipeline: subprocess device plus channel-to-`profile=action` mappings in `messenger-links.json`; a matching legacy mapping still wins over a route table. | `aps adapter messenger ...`, `messenger-links.json` |
| messenger HTTP adapter | Component `aps serve` mounts message services through: loads the service, requires `type: message`, picks normalizer/denormalizer. Its `/messengers/{platform}/webhook` path is not exposed. | `POST /services/<id>/webhook` |
| `messenger_name` | Legacy or service route key carried in `platform_metadata`; loses to `service_id`. | normalized `platform_metadata.messenger_name` |
| message adapter | The `adapter:` a message service persists (`telegram`, `slack`, ...) and the `--type` alias selecting it. Prefer this over "messenger" for services. | `aps service add --type <alias>` |

## Thread

| Phrase | Means | Field / key |
| --- | --- | --- |
| platform thread | Provider reply/thread/topic reference: Slack `thread_ts`, Telegram `reply_to_message.message_id`, Discord `thread_id` (`topic`) or message reference (`reply`), WhatsApp Cloud `context.id`; none for SMS. | `thread.id`, `thread.type` |
| thread scope | Session scope chosen when `thread.id` exists; the session continues in that platform thread. | `conversation.scope: thread` |
| APS multi-turn thread | The message session itself ("multi-turn APS thread key"). | `SessionID` |
| ticket `thread_id` | Field of a ticket service's status JSON; the ticket adapter's thread, not a message thread. | ticket response `thread_id` |
| Teams conversation ID | `19:abc123@thread.tacv2` is a Bot Framework conversation ID: a `channel.id` for `--allowed-channel`, despite the `@thread` suffix. | `channel.id` |

## Service

| Phrase | Means | Route / command |
| --- | --- | --- |
| message service | `type: message`; chat-like payloads; one normalizer/denormalizer per adapter. | `POST /services/<id>/webhook` |
| ticket service | `type: ticket`; work-item events; no unauthenticated catch-all; `default_action` or `routing` required, else `422`. | `POST /services/<id>/ticket/<adapter>` |
| `aps serve` | Shared HTTP server mounting all service routes and the REST API. | `aps serve --addr 127.0.0.1:8080 [--auth-token ...]` |
| `aps service start` | Validates one service, prints its webhook URL, mounts the same routes as `aps serve`, runs in the foreground. | `aps service start <id>` |
| voice service | Voice backend process lifecycle; neither a message nor a ticket service. | `aps voice service start\|stop\|status` |
| `service_id` | Route key and the `service:` part of `ConversationID`. | `platform_metadata.service_id`, `aps service conversation list --service <id>` |

## Anti-patterns

| Wrong term | Right term |
| --- | --- |
| "session" for a chat thread, unqualified | message session / `SessionID`; "process session" for `aps session` |
| "thread" for the whole exchange | conversation / `ConversationID`; a thread is the narrowed scope |
| "messenger" for an `aps service add` route | message service |
| `aps adapter messenger ...` for a new integration | `aps service add`; adapter devices are legacy |
| `aps profile link-messenger`, `aps messengers create --template` | removed forms; `aps service add` |
| "adapter" for `--type telegram` | service alias; `adapter:` is the persisted field |
| "history" or "context" for the stdin array | `prior_turns`, bounded by `history_turns` |
| "reply turn" | outbound turn |
| `action: profile=name` inside a route table | `profile:` and `action:` set separately |
| "channel" for a Discord guild or Slack workspace | `workspace_id` |
| "backend" for an adapter | adapter; the backend is what it delegates to |
| "device" for a message service | service; a device is a legacy adapter instance |
| `/messengers/{platform}/webhook` as an endpoint | `/services/<id>/webhook` |
| `aps adapter messenger test` as a service test | `aps service test` |
