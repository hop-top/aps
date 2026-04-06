# Messenger Integration

APS supports multiple messenger platforms for routing messages to profile actions.

## Supported Platforms

| Platform | Token Source | Channel Format | Mode |
|----------|-------------|----------------|------|
| Telegram | [@BotFather](https://t.me/botfather) | Numeric (e.g., `-1001234567890`) | Subprocess |
| Discord | [Developer Portal](https://discord.com/developers) | Numeric (e.g., `1234567890123456789`) | Subprocess |
| Slack | [API Dashboard](https://api.slack.com) | Alphanumeric (e.g., `C01ABC2DEF`) | Subprocess |
| GitHub | OAuth App | `org/repo` | Webhook |
| Email | SMTP Config | Mailbox/Address | Webhook |

## Setup

```bash
# Interactive setup (recommended)
./scripts/setup-messenger.sh --type=<platform>

# Platform-specific shortcuts
./scripts/setup-messenger.sh --type=telegram --profile=my-agent --messenger=my-telegram --yes
./scripts/setup-messenger.sh --type=discord --profile=my-bot --messenger=discord-bot
```

## Telegram

1. Create a bot via [@BotFather](https://t.me/botfather) and copy the token
2. Find your channel ID: send a message, then visit `https://api.telegram.org/bot<TOKEN>/getUpdates` and look for `chat.id`
3. Channel ID formats: positive for DMs (e.g., `123456789`), negative for groups (e.g., `-1001234567890`)

```bash
aps profile create my-agent
aps messengers create my-telegram --template=subprocess --language=python
aps profile link-messenger my-agent my-telegram \
  --channel "-1001234567890=my-agent:handle-telegram" \
  --default-action "my-agent:default-handler"
aps messengers start my-telegram
```

### Webhook Mode

For serverless operation without a persistent process:

```bash
aps messengers create my-telegram-webhook --template=webhook --language=python
```

## Discord

1. Create an app at [Discord Developer Portal](https://discord.com/developers/applications)
2. Go to "Bot" section, click "Add Bot", copy the token
3. Set permissions: `bot` + `applications.commands` scopes; `Send Messages`, `Read Message History`, `Read Messages/View Channels`
4. Go to "OAuth2" > "URL Generator", select scopes and permissions, copy the generated URL
5. Open the URL to invite the bot to your server
6. Enable Developer Mode in Discord (User Settings > Advanced), right-click channel > "Copy Channel ID"

```bash
aps messengers create my-discord --template=subprocess --language=python
echo "DISCORD_TOKEN=your_token" > ~/.aps/messengers/my-discord/.env
chmod 600 ~/.aps/messengers/my-discord/.env
aps profile link-messenger my-agent my-discord \
  --channel "1234567890123456789=my-agent:handle-discord"
aps messengers start my-discord
```

### Discord vs Slack

| Feature | Discord | Slack |
|---------|---------|-------|
| Channel ID Format | Numeric (19 digits) | Alphanumeric (C...) |
| Free Tier | Full featured | Limited |
| Message History | Unlimited | 90 days (free) |
| Rate Limits | ~10 msg/sec | ~1 msg/sec |

## Common Patterns

### Multiple Channels to Different Actions

```bash
aps profile link-messenger my-agent my-telegram \
  --channel "alerts_channel=my-agent:handle-alerts" \
  --channel "commands_channel=my-agent:handle-commands" \
  --channel "reports_channel=my-agent:handle-reports"
```

### Single Messenger to Multiple Profiles

```bash
aps profile link-messenger alerts main-bot --channel "ch1=alerts:process"
aps profile link-messenger analytics main-bot --channel "ch2=analytics:process"
aps messengers start main-bot  # serves both
```

### Run Multiple Platforms Simultaneously

```bash
aps messengers start my-telegram
aps messengers start my-discord
# Same profile receives from both
```

### Default Action for Unmapped Channels

```bash
aps profile link-messenger my-agent my-messenger \
  --channel "high-priority=my-agent:handle-urgent" \
  --default-action "my-agent:handle-generic"
```

### Update Channel Mappings

```bash
aps profile update-mapping my-agent my-telegram "-1005555555555" "my-agent:new-action"
aps profile remove-mapping my-agent my-telegram "-1002222222222"
aps profile set-default-action my-agent my-telegram "my-agent:fallback"
```

## Operations

```bash
aps messengers status               # check all messengers
aps messengers logs <name> -f       # follow logs
aps messengers stop <name>          # stop messenger
aps messengers start <name>         # start messenger
aps profile link-info <profile>     # show linked messengers
```

### Environment Variables

```bash
TELEGRAM_TOKEN=xxx aps messengers start my-telegram   # override token
DEBUG=1 aps messengers start my-telegram               # debug logging
APS_CONFIG_DIR=/etc/aps aps messengers start my-telegram
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Token invalid | Verify in `.env`; re-copy from platform |
| Bot not receiving (Telegram) | Check group privacy settings; verify with `getUpdates` API |
| Bot not receiving (Discord) | Verify bot permissions in server settings |
| Messages not routing | `aps profile link-info`; verify channel mappings |
| Messenger won't start | `aps messengers start <name> --verbose`; check config and `.env` |
| Action not executing | `aps profile actions <profile>`; check action exists |
| Port conflict | Check with `lsof -i :<port>` |

## Security

- Store tokens in `.env` only (`chmod 600`)
- Never commit `.env` files to version control
- Use private channels for sensitive operations
- Validate input in profile actions
- Grant bots minimum necessary permissions
- Rotate credentials periodically
- Review messenger logs for leaked content
- Be aware of platform rate limits (Telegram ~30/sec, Discord ~10/sec, Slack ~1/sec)
