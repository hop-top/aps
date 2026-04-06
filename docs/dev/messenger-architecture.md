# Messenger Architecture

Design and implementation details for the APS messenger integration layer.

## Message Flow

```
[Messenger Platform] → [Normalize Message] → [Route Lookup] → [Execute Action] → [Optional Response]
```

All platforms normalize incoming messages to a unified JSON structure before routing to profile actions.

## Normalized Message Format

```json
{
  "id": "msg_unique_id",
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
  "reactions": [{"emoji": "...", "count": 1}],
  "attachments": [],
  "platform_metadata": {}
}
```

## Routing

Channel-to-action mappings are defined via the link CLI:

```bash
--channel "channel_id=profile:action"       # explicit mapping
--default-action "profile:fallback"         # unmapped channels
```

A single messenger can serve multiple profiles. A single profile can receive from multiple messengers.

## Deployment Modes

**Subprocess** (continuous polling) — for Telegram, Discord, Slack. The messenger process runs persistently and polls for messages.

**Webhook** (event-triggered) — for GitHub, Email. An HTTP endpoint receives events from the platform.

## Platform Comparison

| Feature | Telegram | Discord | Slack | GitHub | Email |
|---------|----------|---------|-------|--------|-------|
| Real-time | Yes | Yes | Yes | No | No |
| Threads | Limited | Native | Native | No | No |
| Reactions | Limited | Emoji | Emoji | No | No |
| Slash Commands | Limited | Native | Native | No | No |
| Voice Messages | Yes (via APS voice) | No | No | No | No |
| Rate Limits | ~30/sec | ~10/sec | ~1/sec | N/A | Varies |
| Free Tier | Full | Full | Limited | Yes | Varies |

## Permission Models

| Platform | Model |
|----------|-------|
| Telegram | Bot-level only (no role-based) |
| Discord | Role-based (channel + guild permissions) |
| Slack | Workspace + app-level scopes |
| GitHub | OAuth scope-based |
| Email | Basic auth |

## File Layout

```
~/.aps/profiles/<name>/links/<messenger>.yaml   # Link definitions
~/.aps/messengers/<name>/config.yaml            # Messenger config
~/.aps/messengers/<name>/.env                   # Tokens (chmod 600)
~/.aps/messengers/<name>/main.py                # Entry point
~/.aps/messengers/<name>/requirements.txt       # Dependencies
```

## Security Considerations

- Tokens stored in `.env` with `chmod 600` permissions
- Messenger logs may contain message content — review regularly
- Validate all input in profile actions
- Use private channels for sensitive operations
- Implement rate limiting where applicable

## Related

- [Voice integration](voice.md) — voice message support for Telegram/WhatsApp
- [User messenger guide](../user/messengers.md) — setup instructions for end users
- [Agent messenger patterns](../agent/messenger-patterns.md) — agent-specific usage
