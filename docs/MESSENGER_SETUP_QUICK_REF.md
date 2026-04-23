# Messenger Setup Quick Reference

## Fastest Path (Interactive)

```bash
# For Telegram (recommended - step-by-step guidance)
./scripts/setup-telegram.sh

# For any messenger (Telegram, Slack, GitHub, Email)
./scripts/setup-messenger.sh
```

These scripts walk you through everything interactively.

---

## Command-Line Shortcuts

### Telegram Setup (Predefined)
```bash
./scripts/setup-messenger.sh \
  --type=telegram \
  --profile=my-agent \
  --messenger=my-telegram \
  --yes
```

### Slack Setup
```bash
./scripts/setup-messenger.sh \
  --type=slack \
  --profile=my-bot \
  --messenger=slack-bot
```

### Discord Setup
```bash
./scripts/setup-messenger.sh \
  --type=discord \
  --profile=my-bot \
  --messenger=discord-bot
```

### GitHub Webhooks
```bash
./scripts/setup-messenger.sh \
  --type=github \
  --profile=github-handler
```

---

## Manual One-Liner Examples

### Create Telegram Integration (3 steps)
```bash
# 1. Create profile
aps profile create my-agent

# 2. Create messenger
aps messengers create my-telegram --template=subprocess --language=python

# 3. Link with mapping
aps profile link-messenger my-agent my-telegram \
  --channel "-1001234567890=my-agent:handle-message" \
  --default-action "my-agent:default-handler"

# 4. Start
aps messengers start my-telegram
```

### Verify Everything Works
```bash
# Check messenger is running
aps messengers status

# View logs
aps messengers logs my-telegram -f

# List profile links
aps profile link-info my-agent
```

---

## Common Patterns

### Multiple Channels → Different Actions
```bash
aps profile link-messenger my-agent my-telegram \
  --channel "alerts_channel=my-agent:handle-alerts" \
  --channel "commands_channel=my-agent:handle-commands" \
  --channel "reports_channel=my-agent:handle-reports"
```

### Single Messenger → Multiple Profiles
```bash
# Create two profiles
aps profile create alerts
aps profile create analytics

# Create one messenger
aps messengers create main-bot --template=subprocess --language=python

# Link both
aps profile link-messenger alerts main-bot \
  --channel "alert_channel=alerts:process"
aps profile link-messenger analytics main-bot \
  --channel "analytics_channel=analytics:process"

# One start command serves both
aps messengers start main-bot
```

### Disable a Messenger Temporarily
```bash
# Without deleting everything
aps messengers stop my-telegram
aps messengers start my-telegram
```

### Update Channels (Add/Remove)
```bash
# Add new channel mapping
aps profile update-mapping my-agent my-telegram \
  "-1005555555555" "my-agent:new-action"

# Remove a channel
aps profile remove-mapping my-agent my-telegram "-1002222222222"

# Change default action
aps profile set-default-action my-agent my-telegram "my-agent:fallback"
```

---

## Messenger Info Reference

| Messenger | Channel ID Format | Token Source | Mode |
|-----------|-------------------|--------------|------|
| Telegram | Numeric (e.g., `-1001234567890`) | [@BotFather](https://t.me/botfather) | Subprocess |
| Slack | Alphanumeric (e.g., `C01ABC2DEF`) | [API Dashboard](https://api.slack.com) | Subprocess |
| Discord | Numeric (e.g., `1234567890123456789`) | [Developer Portal](https://discord.com/developers) | Subprocess |
| GitHub | `org/repo` | OAuth App | Webhook |
| Email | Mailbox (e.g., `inbox@domain.com`) | SMTP Server | Webhook |

---

## Troubleshooting

### "aps command not found"
```bash
# Install APS CLI
go install ./cmd/aps
# Or add to PATH if already installed
export PATH=$PATH:$(go env GOPATH)/bin
```

### "Messenger won't start"
```bash
# Check config
cat ~/.aps/messengers/my-telegram/config.yaml

# Check token exists
cat ~/.aps/messengers/my-telegram/.env

# Run with verbose output
aps messengers start my-telegram --verbose
```

### "Messages not being routed"
```bash
# Verify link exists
aps profile link-info my-agent

# Check messenger is running
aps messengers status

# Watch logs in real-time
aps messengers logs my-telegram -f

# Verify channel ID is correct
# For Telegram: https://api.telegram.org/bot<TOKEN>/getUpdates
```

### "Action not executing"
```bash
# List available actions
aps profile actions my-agent

# Check action directly
aps my-agent:handle-message '{"text":"test"}'

# Review execution logs
aps messengers logs my-telegram -e 20
```

---

## File Locations

```
~/.aps/profiles/
  my-agent/
    config.yaml          # Profile config
    links/               # Messenger links
      my-telegram.yaml   # Link definition

~/.aps/messengers/
  my-telegram/
    config.yaml          # Messenger config
    .env                 # Tokens & secrets
    main.py              # Entry point
    requirements.txt     # Dependencies
```

---

## Environment Variables

```bash
# Override messenger token at runtime
TELEGRAM_TOKEN=xxx aps messengers start my-telegram

# Enable debug logging
DEBUG=1 aps messengers start my-telegram

# Custom config directory
APS_CONFIG_DIR=/etc/aps aps messengers start my-telegram
```

---

## Next Steps After Setup

1. **Test Message Routing**
   ```bash
   # Send a message to your Telegram group
   # Watch logs to verify it was received
   aps messengers logs my-telegram -f
   ```

2. **Implement Your Action**
   ```bash
   # Edit your action in the profile
   aps profile edit-action my-agent handle-message
   ```

3. **Monitor in Production**
   ```bash
   # Keep logs visible
   aps messengers logs my-telegram -f

   # Watch status
   watch -n 1 'aps messengers status'
   ```

4. **Scale to More Channels**
   ```bash
   # Add more channel mappings as needed
   aps profile update-mapping my-agent my-telegram \
     "-1009999999999" "my-agent:another-action"
   ```

---

## Security Checklist

- [ ] Token stored in `.env` file (not version control)
- [ ] `.env` file permissions: `chmod 600 ~/.aps/messengers/*/. env`
- [ ] Profile actions validate input
- [ ] Sensitive data not logged in messenger output
- [ ] Private channels used for sensitive operations
- [ ] Regular review of messenger logs
- [ ] Credentials rotated periodically

---

## Additional Resources

- **Full Telegram Guide**: `docs/TELEGRAM_SETUP.md`
- **Full Discord Guide**: `docs/DISCORD_SETUP.md`
- **Architecture**: `docs/stories/050-multi-device-workspace-access.md`
- **API Reference**: `docs/` (messenger types)
- **Examples**: `tests/e2e/` (real usage patterns)
