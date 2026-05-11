# Message Conversation And Thread Policy

Message services normalize platform events into `NormalizedMessage`, then derive
one APS conversation identity and one APS session identity from that normalized
shape.

This policy applies to message services (`telegram`, `slack`, `discord`, `sms`,
and `whatsapp`) and to the legacy messenger-device pipeline. It does not define
provider delivery APIs.

## Identity Inputs

APS uses these normalized fields:

| Field | Policy |
| --- | --- |
| `platform` | Provider family, for example `slack` or `sms`. |
| `platform_metadata.service_id` | Service route ID when present. |
| `platform_metadata.messenger_name` | Legacy or service route key when present. |
| `workspace_id` | Platform workspace, server, team, or guild when available. |
| `channel.id` | Platform channel/chat/conversation ID. For SMS and WhatsApp relays this is the receiving phone number or phone-number ID. |
| `channel.type` | `direct`, `group`, `broadcast`, or `topic`. |
| `sender.id` | Platform user ID, handle, or sender phone number. |
| `thread.id` | Platform thread/reply/topic identifier when available. |

`service_id` wins over `messenger_name`; otherwise APS uses `platform` as the
route key.

## Conversation ID

`ConversationID` is the outer place where messages arrive:

```text
msgconv:v1:service:<service-or-platform>:platform:<platform>[:workspace:<id>]:channel:<id>
```

For direct-message and phone conversations, APS also includes the sender:

```text
msgconv:v1:...:channel:<receiving-or-dm-id>:sender:<sender-id>
```

This means two SMS callers using the same receiving number do not share one
conversation. It is also safe for Slack, Discord, Telegram, and WhatsApp direct
messages because the sender component is either required for separation or
redundant with an already unique DM channel ID.

## Session ID

`SessionID` is the multi-turn APS thread key. It starts from the conversation
identity and narrows to a platform thread when `thread.id` exists:

```text
msgsess:v1:<conversation-parts>[:thread:<thread-id>]
```

Policy:

| Message shape | Scope | Multi-turn behavior |
| --- | --- | --- |
| Group/channel message without `thread.id` | `channel` | Continues in the channel root session. |
| Group/channel reply with `thread.id` | `thread` | Continues in that platform thread session. |
| Direct message without `thread.id` | `direct` | Continues in the sender/channel pair session. |
| Phone message without `thread.id` | `direct` | Continues in the sender/receiving-number pair session. |
| Direct or phone reply with `thread.id` | `thread` | Continues in the sender/channel pair plus platform thread. |

The key is deterministic and URL-escaped by component. It is stable across
process restarts for the same normalized identifiers.

## Platform Mapping

| Platform | Channel mapping | User mapping | Thread mapping |
| --- | --- | --- | --- |
| Telegram | `chat.id`; private chats are `direct`, groups/supergroups are `group`, channels are `broadcast`. | `from.id`, with username/name as display metadata. | `reply_to_message.message_id` as `thread.id`, type `reply`. |
| Slack | `event.channel`, with `channel_type` mapped to `direct` or `group`. | `event.user`. | `event.thread_ts` as `thread.id`, type `reply`. |
| Discord | `channel_id`; missing `guild_id` is `direct`, otherwise `group`. | `author.id`. | `thread_id` as type `topic`, or message reference ID as type `reply`. |
| SMS | Receiving number (`To`/`to`) is `channel.id`, sender number (`From`/`from`) is `sender.id`. | Sender phone number. | Not provided by SMS; use direct sender/receiver session. |
| WhatsApp | Cloud phone-number ID or receiving number is `channel.id`; sender phone is `sender.id`. | Sender phone/profile. | Cloud `context.id` as `thread.id`, type `reply`; Twilio-style relays use direct sender/receiver session. |

## Attachments

Attachments remain part of the normalized message payload and are passed to the
profile action with text and platform metadata. Message services do not download
or re-upload files as part of conversation identity.

Policy:

- Supported attachment classes are image, file/document, video, and audio.
- Providers may preserve a concrete file type in `attachment.type` when that is
  more specific than the class, for example `pdf` or `png`.
- Audio attachments may be claimed by the voice handler before normal action
  routing when that handler is configured.
- Attachments without a fetchable URL or provider media identifier are metadata
  only; actions should treat them as unavailable content.
- Attachment URLs and MIME types are untrusted input.

## Mentions And Commands

APS does not strip mentions, slash-command prefixes, bot usernames, or command
verbs from `text` during normalization. The routed profile action receives the
original normalized text and can decide whether a message is addressed to it.

Routing remains channel/service based:

- The service route or messenger link decides which profile action receives the
  message.
- Mentions and command prefixes are action input, not a second routing table.
- Platform command IDs, callback IDs, and mention entities should be preserved
  in `platform_metadata` when a normalizer supports them.

## Unsupported Message Types

Unsupported platform events should fail before routing with a normalization
error, or produce a normalized message with text/attachments/metadata when a
reasonable message-shaped payload exists.

Policy:

- Presence, typing, delivery receipts, read receipts, joins/leaves, reactions
  without message text, and app lifecycle events are not conversation turns.
- Unknown message subtypes are unsupported unless they can be represented as
  text, attachments, or platform metadata without losing the sender/channel
  identity.
- Unsupported events must not create sessions or execute profile actions.
- Webhook callers receive a status/error response; provider-specific retry and
  delivery behavior remains outside this policy.
