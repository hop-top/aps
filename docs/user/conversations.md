# Conversations

> Availability: unreleased. Requires a build newer than aps/v0.6.0-alpha.0.

How a message service remembers what it has seen, and what a routed action gets
from that memory. Reference (key grammar, platform mapping, full stdin example):
[Message conversation and thread policy](../dev/message-conversation-policy.md).

Scope: message services under `aps serve`, added with
`aps service add --type telegram|slack|discord|sms|whatsapp|teams` or, for
email, `--type message --adapter email`. Adapter devices (`aps adapter exec`)
are a separate stateless pipeline and record nothing.

## What a turn is

One message, one direction. Inbound: a platform message routed to a profile.
Outbound: a reply APS delivered. Every turn carries the identity keys below
plus what was said.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/messengermd", "conversation-turn"],
    text=True))
]]] -->
| Field | Type | Optional | Meaning |
| --- | --- | --- | --- |
| `seq` | integer | No | Append order in the store; ascending, unique per store |
| `conversation_id` | string | No | Outer place: service, platform, workspace, channel; plus sender for DMs and phone |
| `session_id` | string | No | Conversation narrowed to a platform thread when the message carries one |
| `service_id` | string | Yes | Service route that recorded the turn |
| `platform` | string | No | Provider family, for example `sms` or `slack` |
| `profile_id` | string | Yes | Profile whose action handled the message |
| `action_name` | string | Yes | Action that ran |
| `direction` | string | No | `inbound` (message received) or `outbound` (reply delivered) |
| `message_id` | string | Yes | Platform message ID; outbound turns carry the inbound ID they answer |
| `channel_id` | string | Yes | Platform channel, chat, or receiving number |
| `sender_id` | string | Yes | Platform sender; the profile ID on outbound turns |
| `sender_name` | string | Yes | Sender display name when the platform provides one |
| `text` | string | No | Message or reply text as routed; mentions and command prefixes kept |
| `attachments` | array | Yes | Normalized attachment metadata |
| `timestamp` | timestamp (RFC 3339, UTC) | No | Platform time for inbound turns, delivery time for outbound; UTC |
<!-- [[[end]]] -->

## Where conversations live

One SQLite file under the APS data directory (`$APS_DATA_PATH`, else the XDG
data dir). Created on the first recorded turn; `aps serve` startup, route
registration, and read-only commands never create it. No file means nothing
recorded yet.

<!-- [[[cog
import subprocess
cog.out(subprocess.check_output(
    ["go", "run", "./internal/tools/messengermd", "conversation-store"],
    text=True))
]]] -->
| Fact | Value |
| --- | --- |
| Location | `<data-dir>/messages/conversations.db` |
| Table | `message_turns` |
| Prior turns per action run | `20` by default; `--history-turns N` per service |
| Turns kept per conversation | newest `500`; older pruned on append |
<!-- [[[end]]] -->

## How APS names a conversation and a session

Two keys, derived from the normalized message. Deterministic; stable across
restarts for the same platform identifiers.

- Conversation: the outer place messages arrive in — service, platform,
  workspace when the platform has one, channel. Direct messages and phone
  conversations add the sender, so two SMS callers on one receiving number
  never share a conversation.
- Session: the conversation, narrowed to a platform thread when the message
  carries one. Group messages without a thread continue in the channel root
  session; replies inside a thread continue in that thread's session.

SMS example — one caller, one receiving number, no platform threads, so the
session repeats the conversation parts:

```text
msgconv:v1:service:support-sms:platform:sms:channel:%2B15550001111:sender:%2B15559990000
msgsess:v1:service:support-sms:platform:sms:channel:%2B15550001111:sender:%2B15559990000
```

Grammar and per-platform channel, user, and thread mapping: policy doc,
sections Conversation ID, Session ID, Platform Mapping.

## When a turn is recorded

| Event | Turn |
| --- | --- |
| Message routed to a profile action | Inbound, recorded before the action runs |
| Message handed to chat mode | Inbound |
| Reply delivered by the provider | Outbound, after delivery succeeds |
| Reply carried in the webhook response (TwiML, provider reply JSON) | Outbound |
| Delivery fails | None |
| Empty reply, or service created with `--reply none` | None |
| Presence, typing, receipts, joins, reactions without text, unsupported events | None |

Outbound turns belong to the profile: `sender_id` is the profile ID,
`message_id` is the inbound message being answered.

## What the action receives

Stdin is still the normalized message JSON; it gains two keys:

- `conversation` — the identity above: `conversation_id`, `session_id`,
  `scope`, and the parts they were built from.
- `prior_turns` — turns recorded before this message in the same session,
  oldest first, newest last. Always an array; empty when nothing was recorded
  or history is unavailable. The current message is never in it.

`RunInput.ThreadID` (and `RunState.thread_id`) is the session ID, so run state
correlates with the thread. Full stdin example: policy doc, Attachment
Contract.

## Tuning history

`prior_turns` is bounded per action run by the service option `history_turns`.
Default: store facts table above. Set per service at creation; `0` keeps the
default.

```bash
aps service add support-sms --type sms --profile support --history-turns 50
```

## Inspecting

```bash
aps service conversation list
aps service conversation list --service support-sms --platform sms --limit 10
aps service conversation show <conversation-id>
aps service conversation show <conversation-id> --session <session-id> --limit 5
aps service conversation show <conversation-id> --format json
```

`list`: conversations most recently active first — turn count, last activity,
last direction, text preview. `show`: turns oldest first, newest last, the
same order and shape actions get in `prior_turns`. Both read-only; both honour
the global `--format table|json|yaml`.

## Failure behaviour

The store never blocks a message. Open, append, and query errors are logged;
routing, action execution, and delivery continue. When a history read fails
the action still runs, with an empty `prior_turns`.

## Retention

Per conversation, only the newest turns survive; older ones are pruned on
every append. Cap: store facts table above. The cap is per conversation, not
per session, so a busy channel prunes its quiet threads too.
