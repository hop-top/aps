# Messenger Patterns for Agents

Patterns for agents interacting with messenger platforms via the APS adapter CLI.

## Setup

```bash
# Create and link a messenger to a profile
aps messengers create <name> --template=subprocess --language=python
aps profile link-messenger <profile> <messenger> \
  --channel "<channel_id>=<profile>:<action>" \
  --default-action "<profile>:<fallback>"
```

## Operations

```bash
aps messengers start <name>       # start messenger
aps messengers status             # check all
aps messengers logs <name> -f     # follow logs
aps adapter test <id>             # test pipeline
aps adapter channels <id>         # list known channels
```

## Normalized Message Contract

All platforms deliver messages in a unified format:

```json
{
  "id": "msg_id",
  "platform": "telegram|discord|slack|github|email",
  "profile_id": "my-profile",
  "timestamp": "2026-02-17T10:30:00Z",
  "sender": {
    "id": "user_id",
    "name": "display_name",
    "platform_id": "platform_user_id"
  },
  "channel": {
    "id": "channel_id",
    "name": "channel_name",
    "type": "direct|group|topic|issue|thread"
  },
  "text": "message content",
  "thread": { "id": "thread_id", "type": "reply" },
  "platform_metadata": {}
}
```

## Multi-Messenger Coordination

A single profile can receive from multiple platforms simultaneously. Route by channel ID; use `--default-action` for unmapped channels.

```bash
# Same profile, two platforms
aps profile link-messenger my-agent my-telegram --channel "..."
aps profile link-messenger my-agent my-discord --channel "..."
aps messengers start my-telegram
aps messengers start my-discord
```

## Error Handling

| Condition | Handling |
|-----------|----------|
| Messenger not receiving | `aps adapter test <id>`; check `aps adapter logs <id>` |
| Channel not found | Verify channel ID format matches platform |
| Action not executing | `aps profile actions <profile>`; check action exists |
| Messenger won't start | Check config and `.env`; run with `--verbose` |

## Related

- [Messenger architecture](../dev/messenger-architecture.md) — normalized format, platform comparison, file layout
- [User messenger guide](../user/messengers.md) — platform setup instructions
