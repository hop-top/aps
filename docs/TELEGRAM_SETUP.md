# Telegram Messenger Integration Guide

This guide walks you through setting up Telegram communication with APS agent profiles.

## Quick Start

The easiest way to set up Telegram integration is with the interactive setup script:

```bash
./scripts/setup-telegram.sh
```

This script will guide you through all necessary configuration steps interactively.

## What You'll Need

1. **A Telegram Bot Token** — obtained from [BotFather](https://t.me/botfather)
2. **An APS Profile** — where your agent actions run
3. **Telegram Channel/Group IDs** — where you want to send commands

## Setup Script Workflow

### Step 1: Profile Selection
Choose an existing profile or create a new one. This is the profile where your agent actions will execute.

```
Existing profiles:
  • my-agent
  • analysis-bot

Use an existing profile? [y/n]: y
Profile ID: my-agent
```

### Step 2: Telegram Messenger Setup
Create or reuse a Telegram messenger. You'll choose:

- **Messenger Name**: How to reference it (e.g., `my-telegram`)
- **Mode**:
  - **Subprocess**: Continuously running, listens for all messages
  - **Webhook**: Triggered by external events
- **Language**: Implementation language (Python, Go, etc.)

The script will ask for your Telegram bot token (create one via [@BotFather](https://t.me/botfather)).

```
Creating new Telegram messenger...
Telegram messenger name [my-telegram]:
Use subprocess mode (always listening for messages)? [y/n]: y
Preferred language for messenger implementation [python]: python
Telegram bot token: 123456789:ABCDefGHijKLmnoPQRstUVwxYZ
```

### Step 3: Channel Mappings
Map Telegram channels/groups to specific profile actions:

```
Telegram channel ID (or 'done' to finish): -1001234567890
Action to execute (e.g., handle-telegram, process-command): handle-telegram
✓ Mapped channel -1001234567890 → handle-telegram

Telegram channel ID (or 'done' to finish): done
```

**Channel ID Formats:**
- **Direct Messages**: Positive number (e.g., `123456789`)
- **Groups/Supergroups**: Negative number (e.g., `-1001234567890`)

To find a group's channel ID:
1. Send a test message to the group
2. Visit `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
3. Look for the `chat.id` field in the JSON response

### Step 4: Configuration Applied
The script links the messenger to your profile and stores the configuration.

### Step 5: Deploy
Review the summary and choose whether to start the messenger immediately:

```
Configuration Summary:
  Profile:     my-agent
  Messenger:   my-telegram
  Mappings:    1 channel(s)
  Default:     handle-telegram

Start the messenger now? [y/n]: y
✓ Messenger started
```

## Manual Configuration

If you prefer manual setup or need advanced options:

### 1. Create a Profile

```bash
aps profile create my-agent
```

### 2. Create a Telegram Messenger

```bash
aps messengers create my-telegram --template=subprocess --language=python
```

### 3. Configure the Messenger

Edit the messenger configuration:

```bash
nano ~/.aps/messengers/my-telegram/config.yaml
```

Add your Telegram token and settings. Store sensitive tokens in `.env`:

```bash
echo "TELEGRAM_TOKEN=your_token_here" > ~/.aps/messengers/my-telegram/.env
```

### 4. Link to Profile

```bash
aps profile link-messenger my-agent my-telegram \
  --channel "-1001234567890=my-agent:handle-telegram" \
  --default-action "my-agent:default-handler"
```

### 5. Start the Messenger

```bash
aps messengers start my-telegram
```

## Message Flow

When a user sends a message to your Telegram channel:

```
User → Telegram → my-telegram (messenger)
                       ↓
                  [Normalize]
                       ↓
                  [Route Lookup]
                       ↓
            aps my-agent:handle-telegram
                       ↓
                  [Execute]
                       ↓
                  Response (optional)
```

## Normalized Message Format

Your action receives messages normalized to JSON:

```json
{
  "id": "tg_msg_12345",
  "platform": "telegram",
  "profile_id": "my-agent",
  "timestamp": "2026-02-17T10:30:00Z",
  "sender": {
    "id": "user_123",
    "name": "John Doe",
    "platform_id": "123456789"
  },
  "channel": {
    "id": "-1001234567890",
    "name": "my-group",
    "type": "group"
  },
  "text": "execute my command",
  "platform_metadata": {
    "message_id": 456,
    "chat_id": "-1001234567890"
  }
}
```

## Operations

### Start Messenger
```bash
aps messengers start my-telegram
```

### Check Status
```bash
aps messengers status
```

### View Logs
```bash
aps messengers logs my-telegram
aps messengers logs my-telegram -f  # Follow logs
```

### Stop Messenger
```bash
aps messengers stop my-telegram
```

### List Mappings
```bash
aps profile link-info my-agent
```

## Troubleshooting

### Token Not Working
- Verify token format with [@BotFather](https://t.me/botfather)
- Ensure token is stored correctly in `.env`
- Check file permissions: `chmod 600 ~/.aps/messengers/my-telegram/.env`

### Messages Not Being Routed
- Verify channel ID with `getUpdates` API call
- Check link is enabled: `aps profile link-info my-agent`
- Review logs: `aps messengers logs my-telegram -f`

### Messenger Won't Start
- Check configuration: `cat ~/.aps/messengers/my-telegram/config.yaml`
- Verify token exists: `cat ~/.aps/messengers/my-telegram/.env`
- Look for errors: `aps messengers start my-telegram --verbose`

### Action Not Executing
- Verify action exists: `aps profile actions my-agent`
- Check mappings: `aps profile link-info my-agent`
- Review logs for execution errors

## Advanced Configuration

### Multiple Channels
Map multiple Telegram channels to different actions:

```bash
aps profile link-messenger my-agent my-telegram \
  --channel "-1001111111111=my-agent:handle-alerts" \
  --channel "-1002222222222=my-agent:handle-commands" \
  --channel "-1003333333333=my-agent:process-reports"
```

### Default Action
If a channel isn't mapped, use a default action:

```bash
aps profile link-messenger my-agent my-telegram \
  --channel "-1001234567890=my-agent:handle-telegram" \
  --default-action "my-agent:default-handler"
```

### Update Mappings
Add a new channel mapping to an existing link:

```bash
aps profile update-mapping my-agent my-telegram \
  -1009999999999 my-agent:new-action
```

### Webhook Mode
For serverless/webhook-based operation:

```bash
aps messengers create my-telegram-webhook --template=webhook --language=python
```

This creates an HTTP endpoint that processes Telegram updates without running a persistent process.

## Examples

### Example 1: Alert System
Set up a channel where Telegram alerts trigger actions:

```bash
./scripts/setup-telegram.sh
# Create profile: alert-bot
# Create messenger: telegram-alerts
# Map channel: -1002468135799 → alert-bot:handle-alert
# Start it up
```

Users can now send alerts in Telegram, and they execute in your alert-bot profile.

### Example 2: Command Bot
Create a bot that responds to commands:

```bash
aps profile create command-bot
aps profile add-action command-bot echo "Received: \$1"
aps messengers create cmd-telegram --template=subprocess --language=python
aps profile link-messenger command-bot cmd-telegram \
  --channel "your_chat_id=command-bot:echo"
aps messengers start cmd-telegram
```

Now users can message your bot and it echoes their command.

### Example 3: Multi-Channel Router
Route different channels to different profiles:

```bash
# Create two profiles
aps profile create notifications
aps profile create analytics

# Single messenger serves both
aps messengers create main-telegram --template=subprocess --language=python

# Link to both profiles
aps profile link-messenger notifications main-telegram \
  --channel "-1001111111111=notifications:process"
aps profile link-messenger analytics main-telegram \
  --channel "-1002222222222=analytics:process"

# Start once
aps messengers start main-telegram
```

## Security Notes

- **Token Storage**: Never commit `.env` files with tokens to version control
- **Channel Access**: Use private Telegram channels for sensitive operations
- **Action Validation**: Ensure your profile actions validate message content
- **Logging**: Review logs regularly; messenger logs may contain message content
- **Rate Limiting**: Be aware of Telegram's rate limits (~30 messages/second per bot)

## Integration with Other Messengers

The same setup script works with other messengers:

```bash
# Slack integration
./scripts/setup-messenger.sh --type=slack

# Discord integration
./scripts/setup-messenger.sh --type=discord

# GitHub webhook integration
./scripts/setup-messenger.sh --type=github

# Email integration
./scripts/setup-messenger.sh --type=email
```

Each messenger follows the same mapping and routing principles.
