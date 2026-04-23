# Discord Messenger Integration Guide

This guide walks you through setting up Discord communication with APS agent profiles.

## Quick Start

The easiest way to set up Discord integration is with the interactive setup script:

```bash
./scripts/setup-messenger.sh --type=discord
```

Or for a more hands-on experience:

```bash
./scripts/setup-telegram.sh  # Generic script that asks for messenger type
```

## What You'll Need

1. **A Discord Bot Token** — obtained from [Discord Developer Portal](https://discord.com/developers/applications)
2. **An APS Profile** — where your agent actions run
3. **Discord Channel IDs** — where you want to send commands

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

### Step 2: Discord Messenger Setup
Create a new Discord messenger. You'll choose:

- **Messenger Name**: How to reference it (e.g., `my-discord`, `discord-bot`)
- **Mode**:
  - **Subprocess**: Continuously running, listens for all messages
  - **Webhook**: Triggered by external events
- **Language**: Implementation language (Python, Go, etc.)

The script will ask for your Discord bot token (get it from [Discord Developer Portal](https://discord.com/developers/applications)).

```
Creating new Discord messenger...
Discord messenger name [my-discord]: my-bot
Use subprocess mode (always listening for messages)? [y/n]: y
Preferred language for messenger implementation [python]: python
Discord bot token: MTk4NjIyNDgzNDU2Mjk0...
```

### Step 3: Channel Mappings
Map Discord channels to specific profile actions:

```
Discord channel ID (or 'done' to finish): 1234567890123456789
Action to execute (e.g., handle-discord, process-command): handle-discord
✓ Mapped channel 1234567890123456789 → handle-discord

Discord channel ID (or 'done' to finish): 9876543210987654321
Action to execute (e.g., handle-discord, process-command): handle-alerts
✓ Mapped channel 9876543210987654321 → handle-alerts

Discord channel ID (or 'done' to finish): done
```

### Step 4: Configuration Applied
The script links the messenger to your profile and stores the configuration.

### Step 5: Deploy
Review the summary and choose whether to start the messenger immediately:

```
Configuration Summary:
  Profile:     my-agent
  Messenger:   my-discord
  Mappings:    2 channel(s)

Start the messenger now? [y/n]: y
✓ Messenger started
```

## Manual Configuration

If you prefer manual setup or need advanced options:

### 1. Create a Profile

```bash
aps profile create my-agent
```

### 2. Create a Discord Messenger

```bash
aps messengers create my-discord --template=subprocess --language=python
```

### 3. Configure the Messenger

Edit the messenger configuration:

```bash
nano ~/.aps/messengers/my-discord/config.yaml
```

Store your Discord token in `.env`:

```bash
echo "DISCORD_TOKEN=your_token_here" > ~/.aps/messengers/my-discord/.env
chmod 600 ~/.aps/messengers/my-discord/.env
```

### 4. Link to Profile

```bash
aps profile link-messenger my-agent my-discord \
  --channel "1234567890123456789=my-agent:handle-discord" \
  --default-action "my-agent:default-handler"
```

### 5. Start the Messenger

```bash
aps messengers start my-discord
```

## Getting Your Discord Bot Token

### Step 1: Create Application
1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application"
3. Give it a name (e.g., "APS Agent Bot")
4. Accept the terms and create

### Step 2: Create Bot
1. Go to "Bot" section in the left sidebar
2. Click "Add Bot"
3. Copy the token (this is what you need for `DISCORD_TOKEN`)

### Step 3: Set Permissions
1. In Bot section, scroll to "TOKEN PERMISSIONS"
2. Enable these scopes:
   - `bot`
   - `applications.commands`
3. Enable these permissions:
   - `Send Messages`
   - `Read Message History`
   - `Read Messages/View Channels`

### Step 4: Get OAuth URL
1. Go to "OAuth2" → "URL Generator"
2. Select scopes: `bot`, `applications.commands`
3. Select permissions: `Send Messages`, `Read Message History`, `Read Messages/View Channels`
4. Copy the generated URL

### Step 5: Invite Bot to Server
1. Open the URL in your browser
2. Select the Discord server to add the bot to
3. Click "Authorize"

### Step 6: Get Channel IDs
To find your Discord channel ID:

**Method 1: Discord UI**
1. Enable Developer Mode in Discord (User Settings → Advanced → Developer Mode)
2. Right-click on the channel
3. Select "Copy Channel ID"

**Method 2: API**
1. Send a test message in your channel
2. Run: `curl "https://discordapp.com/api/channels/{channel_id}/messages?access_token={bot_token}"`
3. Look for the channel ID in the response

## Message Flow

When a user sends a message to your Discord channel:

```
User → Discord Server → my-discord (messenger)
                            ↓
                       [Normalize]
                            ↓
                       [Route Lookup]
                            ↓
                 aps my-agent:handle-discord
                            ↓
                       [Execute]
                            ↓
                       Response (optional)
```

## Normalized Message Format

Your action receives messages normalized to JSON:

```json
{
  "id": "discord_msg_789123",
  "platform": "discord",
  "profile_id": "my-agent",
  "timestamp": "2026-02-17T10:30:00Z",
  "sender": {
    "id": "user_456",
    "name": "alice",
    "platform_id": "123456789012345678"
  },
  "channel": {
    "id": "1234567890123456789",
    "name": "commands",
    "type": "text"
  },
  "text": "execute my command",
  "thread": {
    "id": "9876543210987654321",
    "type": "reply"
  },
  "platform_metadata": {
    "message_id": "1234567890",
    "guild_id": "9876543210"
  }
}
```

## Operations

### Start Messenger
```bash
aps messengers start my-discord
```

### Check Status
```bash
aps messengers status
```

### View Logs
```bash
aps messengers logs my-discord
aps messengers logs my-discord -f  # Follow logs
```

### Stop Messenger
```bash
aps messengers stop my-discord
```

### List Mappings
```bash
aps profile link-info my-agent
```

## Troubleshooting

### Bot Token Not Working
- Verify you copied the correct token from Developer Portal
- Ensure bot is invited to your server
- Check that bot has "Send Messages" permission

### Bot Not Receiving Messages
- Verify bot has "Read Messages/View Channels" permission
- Check channel ID is correct (enable Developer Mode in Discord)
- Ensure messenger is running: `aps messengers status`
- Review logs: `aps messengers logs my-discord -f`

### Messenger Won't Start
- Check configuration: `cat ~/.aps/messengers/my-discord/config.yaml`
- Verify token exists: `cat ~/.aps/messengers/my-discord/.env`
- Look for errors: `aps messengers start my-discord --verbose`

### Action Not Executing
- Verify action exists: `aps profile actions my-agent`
- Check mappings: `aps profile link-info my-agent`
- Review logs for execution errors

### "Permissions Error" or "Cannot Access"
- Go to Discord server settings
- Under Roles, ensure bot role has required permissions
- Try re-inviting bot with correct permissions

## Advanced Configuration

### Multiple Channels
Map multiple Discord channels to different actions:

```bash
aps profile link-messenger my-agent my-discord \
  --channel "1111111111111111111=my-agent:handle-alerts" \
  --channel "2222222222222222222=my-agent:handle-commands" \
  --channel "3333333333333333333=my-agent:handle-reports"
```

### Default Action
If a channel isn't mapped, use a default action:

```bash
aps profile link-messenger my-agent my-discord \
  --channel "1234567890123456789=my-agent:handle-discord" \
  --default-action "my-agent:default-handler"
```

### Thread Handling
Discord supports thread replies. Your messenger can:
- Route thread messages to specific actions
- Send responses in threads or main channel
- Track thread context in `platform_metadata`

### Reactions
Discord message reactions are captured in the normalized message:

```json
"reactions": [
  {"emoji": "👍", "count": 5},
  {"emoji": "🚀", "count": 2}
]
```

### Slash Commands
For slash command support, configure in your messenger implementation:

```python
# Listen for /command interactions
# Route to appropriate profile action
# Send response via interaction webhook
```

## Examples

### Example 1: Command Bot
Set up a bot that responds to commands:

```bash
aps profile create command-bot
aps profile add-action command-bot echo "You said: \$1"
aps messengers create discord-cmds --template=subprocess --language=python
aps profile link-messenger command-bot discord-cmds \
  --channel "your_channel_id=command-bot:echo"
aps messengers start discord-cmds
```

Now users can message the bot and get responses.

### Example 2: Alert System
Route different channels to alert handlers:

```bash
aps profile create alerts
aps messengers create discord-alerts --template=subprocess --language=python
aps profile link-messenger alerts discord-alerts \
  --channel "alerts_channel_id=alerts:process-alert" \
  --channel "critical_channel_id=alerts:process-critical" \
  --default-action "alerts:process-unknown"
```

### Example 3: Multi-Server Setup
One messenger serves multiple servers:

```bash
# Create messenger once
aps messengers create my-discord --template=subprocess --language=python

# Link to multiple profiles
aps profile link-messenger alerts my-discord \
  --channel "server1_channel_id=alerts:process"
aps profile link-messenger analytics my-discord \
  --channel "server2_channel_id=analytics:process"

# Start once, serves all
aps messengers start my-discord
```

## Security Notes

- **Token Storage**: Never commit `.env` files with tokens to version control
- **Channel Access**: Use private Discord channels for sensitive operations
- **Action Validation**: Ensure your profile actions validate message content
- **Permissions**: Grant bot minimum necessary permissions
- **Logging**: Be aware that messenger logs may contain message content
- **Rate Limiting**: Discord has rate limits (~10 messages/second per bot)

## Integration with Other Messengers

You can run Discord alongside other messengers:

```bash
# Telegram bot
aps messengers create my-telegram --template=subprocess --language=python
aps profile link-messenger my-agent my-telegram --channel "..."

# Discord bot
aps messengers create my-discord --template=subprocess --language=python
aps profile link-messenger my-agent my-discord --channel "..."

# Both running simultaneously
aps messengers start my-telegram
aps messengers start my-discord
```

Different channels can route to the same or different profile actions.

## Comparing Discord and Slack

| Feature | Discord | Slack |
|---------|---------|-------|
| **Channel ID Format** | Numeric (19 digits) | Alphanumeric (C...) |
| **Free Tier** | ✓ Full featured | Limited |
| **Message History** | Unlimited | Limited (90 days free) |
| **Threads** | ✓ Native support | ✓ Native support |
| **Reactions** | ✓ Emoji reactions | ✓ Emoji reactions |
| **Slash Commands** | ✓ Native | ✓ Native |
| **Rate Limits** | ~10 msg/sec | ~1 msg/sec |
| **Setup Complexity** | Medium | Medium |

Both work equally well with APS messaging framework.
