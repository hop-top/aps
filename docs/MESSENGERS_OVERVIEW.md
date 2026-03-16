# APS Messengers Overview

Complete reference for all supported messenger platforms and how to integrate them with APS profiles.

## Quick Navigation

| Platform | Guide | Token Source | Channel Format | Mode |
|----------|-------|--------------|----------------|------|
| **Telegram** | [Full Guide](TELEGRAM_SETUP.md) | [@BotFather](https://t.me/botfather) | Numeric (e.g., `-1001234567890`) | Subprocess |
| **Discord** | [Full Guide](DISCORD_SETUP.md) | [Developer Portal](https://discord.com/developers) | Numeric (e.g., `1234567890123456789`) | Subprocess |
| **Slack** | [Quick Ref](MESSENGER_SETUP_QUICK_REF.md) | [API Dashboard](https://api.slack.com) | Alphanumeric (e.g., `C01ABC2DEF`) | Subprocess |
| **GitHub** | [Quick Ref](MESSENGER_SETUP_QUICK_REF.md) | OAuth App | `org/repo` | Webhook |
| **Email** | [Quick Ref](MESSENGER_SETUP_QUICK_REF.md) | SMTP Config | Mailbox/Address | Webhook |

## Setup Command

For any platform, use:

```bash
./scripts/setup-messenger.sh --type=<platform>
```

### By Platform

**Telegram:**
```bash
./scripts/setup-messenger.sh --type=telegram
# or: ./scripts/setup-telegram.sh (specialized)
```

**Discord:**
```bash
./scripts/setup-messenger.sh --type=discord
```

**Slack:**
```bash
./scripts/setup-messenger.sh --type=slack
```

**GitHub:**
```bash
./scripts/setup-messenger.sh --type=github
```

**Email:**
```bash
./scripts/setup-messenger.sh --type=email
```

## Comparison Matrix

### Features

| Feature | Telegram | Discord | Slack | GitHub | Email |
|---------|----------|---------|-------|--------|-------|
| **Real-time Messages** | ✓ | ✓ | ✓ | ✗ | ✗ |
| **Free Tier** | ✓ Full | ✓ Full | Limited | ✓ | Varies |
| **Threads/Replies** | Limited | ✓ Native | ✓ Native | ✗ | ✗ |
| **Reactions** | Limited | ✓ Emoji | ✓ Emoji | ✗ | ✗ |
| **Slash Commands** | Limited | ✓ Native | ✓ Native | ✗ | ✗ |
| **Direct Messages** | ✓ | ✓ | ✓ | ✗ | ✓ |
| **Scheduled Messages** | ✗ | ✗ | ✓ | ✗ | ✓ |
| **Message History** | Unlimited | Unlimited | 90 days (free) | ✗ | ✓ |
| **Multi-user** | ✓ | ✓ | ✓ | ✗ | ✓ |
| **Rate Limits** | ~30/sec | ~10/sec | ~1/sec | ✗ | Varies |

### Setup Complexity

1. **Easiest**: Telegram
   - Visit @BotFather
   - Get token
   - Done

2. **Easy**: Discord
   - Developer Portal
   - Create bot
   - 5 minutes

3. **Medium**: Slack
   - API Dashboard
   - Create app
   - Set permissions

4. **Hard**: GitHub
   - OAuth app setup
   - Webhook configuration
   - Event routing

5. **Hard**: Email
   - SMTP configuration
   - MX records
   - Auth setup

## When to Use Each

### Telegram
**Best for**: Commands, quick notifications, broad audience
- ✓ Easy setup (@BotFather)
- ✓ Large community
- ✓ No cost
- ✗ Limited formatting

**Use cases**:
- Alert systems
- Command bots
- Notifications
- Quick polls

### Discord
**Best for**: Community servers, multi-team collaboration, gaming communities
- ✓ Rich features (threads, reactions, slash commands)
- ✓ Free and unlimited
- ✓ Large community
- ✓ Permission management

**Use cases**:
- Multi-project coordination
- Team notifications
- Community management
- Event automation

### Slack
**Best for**: Enterprise teams, business-critical workflows
- ✓ Enterprise features
- ✓ Workflow builder
- ✓ Permission management
- ✗ Paid pricing

**Use cases**:
- Team coordination
- Business notifications
- Workflow integration
- Enterprise automation

### GitHub
**Best for**: Developer workflows, CI/CD integration
- ✓ Tight SCM integration
- ✓ Webhook-based (serverless)
- ✓ Event filtering

**Use cases**:
- Build notifications
- PR automation
- Issue routing
- Code review alerts

### Email
**Best for**: Transactional notifications, integration with legacy systems
- ✓ Universal access
- ✓ No dependencies
- ✓ Logging/audit trail
- ✗ Slow (not real-time)

**Use cases**:
- Confirmations
- Receipts
- Audit logs
- Non-urgent notifications

## Architecture Comparison

All messengers use the same core architecture:

```
[Messenger Platform]
        ↓
   [Normalize Message]
   (unified format)
        ↓
   [Route Lookup]
   (find profile & action)
        ↓
   [Execute Action]
   (in profile context)
        ↓
   [Optional Response]
```

### Normalized Message Format

Every platform routes to:

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
  "reactions": [{"emoji": "👍", "count": 1}],
  "attachments": [...],
  "platform_metadata": { ... }
}
```

## Routing Configuration

Each messenger-profile link defines:

```bash
# Single channel to action
--channel "channel_id=profile:action"

# Multiple channels
--channel "id1=profile:action1" \
--channel "id2=profile:action2"

# Default action for unmapped channels
--default-action "profile:fallback"
```

## Multi-Messenger Architecture

Run multiple messengers simultaneously:

```bash
# Telegram
aps messengers create my-telegram --template=subprocess
aps profile link-messenger my-agent my-telegram --channel "..."
aps messengers start my-telegram

# Discord
aps messengers create my-discord --template=subprocess
aps profile link-messenger my-agent my-discord --channel "..."
aps messengers start my-discord

# Both running, both feeding the same profile
```

## Common Patterns

### Pattern 1: Alert Router
Different channels → Different alert actions

```bash
# Create specialized profiles
aps profile create alerts-critical
aps profile create alerts-warning
aps profile create alerts-info

# Single Telegram messenger
aps messengers create alerts-bot

# Route different channels to different severities
aps profile link-messenger alerts-critical alerts-bot \
  --channel "-1001111111111=alerts-critical:process"
aps profile link-messenger alerts-warning alerts-bot \
  --channel "-1002222222222=alerts-warning:process"
aps profile link-messenger alerts-info alerts-bot \
  --channel "-1003333333333=alerts-info:process"
```

### Pattern 2: Multi-Channel Single Action
Multiple channels → Same action

```bash
# All channels feed one handler
aps profile link-messenger my-profile my-messenger \
  --channel "channel1=my-profile:unified-handler" \
  --channel "channel2=my-profile:unified-handler" \
  --channel "channel3=my-profile:unified-handler"

# Handler identifies channel from normalized message
```

### Pattern 3: Multi-Platform Same Profile
Multiple messengers → Same profile

```bash
# Telegram integration
aps messengers create my-telegram
aps profile link-messenger my-agent my-telegram --channel "..."

# Discord integration
aps messengers create my-discord
aps profile link-messenger my-agent my-discord --channel "..."

# Same profile receives messages from both platforms
```

### Pattern 4: Fallback Chain
Unmapped channels use default action

```bash
aps profile link-messenger my-agent my-messenger \
  --channel "high-priority=my-agent:handle-urgent" \
  --default-action "my-agent:handle-generic"

# Messages to high-priority channel → handle-urgent
# Messages to any other channel → handle-generic
```

## Security Considerations

### Token Storage
```bash
# Store tokens in .env (not version control)
echo "TELEGRAM_TOKEN=xxx" > ~/.aps/messengers/my-telegram/.env
chmod 600 ~/.aps/messengers/my-telegram/.env
```

### Permission Levels

| Platform | Permission Model |
|----------|------------------|
| Telegram | No role-based (bot-level only) |
| Discord | Role-based (channel + guild) |
| Slack | Workspace + app-level |
| GitHub | OAuth scope-based |
| Email | None (basic auth) |

### Sensitive Data
- Store tokens in `.env` only
- Don't log message content to persistent logs
- Use private channels for sensitive messages
- Validate all input in profile actions
- Implement rate limiting

## Deployment Modes

### Subprocess (Always Listening)
```bash
# Poll for messages continuously
aps messengers create my-bot --template=subprocess
aps messengers start my-bot  # Runs until stopped
```

**Best for**: Telegram, Discord, Slack (continuous presence)

### Webhook (Event-Triggered)
```bash
# Listen for external triggers
aps messengers create my-webhook --template=webhook
# Platform (GitHub, Email) POSTs to webhook endpoint
```

**Best for**: GitHub, Email (external events)

## Troubleshooting Quick Reference

| Problem | Messenger | Solution |
|---------|-----------|----------|
| Token invalid | All | Verify token in `.env` |
| Bot not receiving | Telegram | Check Telegram group privacy |
| Bot not receiving | Discord | Verify bot permissions |
| Bot not receiving | Slack | Check app permissions & subscriptions |
| Slow routing | All | Check profile action performance |
| Messages lost | Webhook | Verify webhook URL & TLS |
| Channel not found | All | Verify channel ID format |

## Operations Commands

All messengers support:

```bash
aps messengers list              # List all messengers
aps messengers status            # Check status of all
aps messengers logs <name>       # View logs
aps messengers logs <name> -f    # Follow logs (streaming)
aps messengers start <name>      # Start messenger
aps messengers stop <name>       # Stop messenger
aps messengers delete <name>     # Delete messenger
```

Profile-specific:

```bash
aps profile link-info <profile>           # Show all linked messengers
aps profile link-messenger <profile> <messenger> --channel "..."
aps profile update-mapping <profile> <messenger> <channel> <action>
aps profile remove-mapping <profile> <messenger> <channel>
```

## Integration Patterns

### With CI/CD (GitHub)
```bash
# GitHub webhook → APS action → Deploy
- Platform: GitHub
- Event: push, pull_request
- Route: github-profile:handle-event
```

### With Notifications (Telegram/Discord)
```bash
# External system → Messenger webhook → APS action
- Prometheus alert → Telegram → profile:handle-alert
- App error → Discord → profile:handle-error
```

### With Team Coordination (Slack/Discord)
```bash
# User command → Messenger → APS action → Response
- /deploy-staging → Slack → profile:deploy
- /rollback → Discord → profile:rollback
```

## Best Practices

1. **Use specialized profiles** for each messenger type
2. **Map channels explicitly** rather than using defaults
3. **Store tokens securely** (`.env` with 600 permissions)
4. **Monitor logs** regularly for errors
5. **Test routing** before production deployment
6. **Validate input** in profile actions
7. **Use rate limiting** where applicable
8. **Keep tokens rotated** periodically
9. **Document channel mappings** for team
10. **Test failover** scenarios

## Next Steps

1. Choose your messenger: [Telegram](TELEGRAM_SETUP.md) | [Discord](DISCORD_SETUP.md) | [Quick Ref](MESSENGER_SETUP_QUICK_REF.md)
2. Run setup script: `./scripts/setup-messenger.sh --type=<type>`
3. Test integration: Send a message and watch logs
4. Implement profile actions
5. Scale to multiple channels/messengers
