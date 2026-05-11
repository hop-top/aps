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

| Input | Persisted type | Persisted adapter |
| --- | --- | --- |
| `--type message --adapter telegram` | `message` | `telegram` |
| `--type telegram` | `message` | `telegram` |
| `--type slack` | `message` | `slack` |
| `--type discord` | `message` | `discord` |
| `--type sms` | `message` | `sms` |
| `--type whatsapp` | `message` | `whatsapp` |

Ticket aliases share the same command grammar but resolve to `ticket`, not
`message`: `email`, `github`, `gitlab`, `jira`, and `linear`.

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
[Resolve service default action or messenger link mapping]
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
  "platform": "telegram|discord|slack|sms|whatsapp",
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

`github` and `email` still exist in the lower-level normalizer for older
message-adapter code, but user-facing service aliases route them to `ticket`
services.

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

## Adapter Support

| Adapter | Normalize support | Denormalize support | Service maturity |
| --- | --- | --- | --- |
| Telegram | Bot API `message` and `edited_message` JSON | `sendMessage` JSON | Ready when mounted with `aps serve` |
| Slack | Events API event envelope JSON | text response JSON | Ready when mounted with `aps serve`; app verification is external |
| Discord | message-create style JSON | content response JSON | Ready when mounted with `aps serve`; Gateway client is external |
| SMS | Twilio-style or generic phone fields in JSON | text response metadata | Ready for JSON relays; native form webhook handling is external |
| WhatsApp | Cloud API JSON or Twilio-style WhatsApp JSON | text response metadata | Ready for JSON relays |

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

Legacy component routes still exist at:

```text
POST /messengers/{platform}/webhook
```

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

- [Agent messenger patterns](../agent/messenger-patterns.md)
- [User messenger guide](../user/messengers.md)
- [Service UX draft](service-ux-draft.md)
