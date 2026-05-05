# Getting Started with APS

APS (Agent Profile System) is a local-first tool that lets you run commands and agent workflows under isolated profiles.

## Quick Start

### 1. Install APS

```bash
# Build from source
git clone https://github.com/IdeaCraftersLabs/oss-aps-cli.git
cd oss-aps-cli
make build
```

### 2. Create Your First Profile

```bash
# Create a new profile
./aps profile create myagent --display-name "My AI Agent" --email "agent@example.com"
```

This creates a profile at `~/.agents/profiles/myagent/` with:
- `profile.yaml` - Profile configuration
- `secrets.env` - Environment variables (tokens, keys)
- `gitconfig` - Git configuration (if email provided)
- `notes.md` - Notes for this profile
- `actions/` - Directory for custom action scripts

### 3. Run Commands Under Your Profile

```bash
# Run a simple command
./aps myagent -- echo "Hello from agent!"

# Run an interactive shell
./aps myagent

# Run git commands with the profile's git config
./aps myagent -- git status

# Run external LLM CLIs with the profile's secrets and identity
./aps run myagent -- claude "summarize this branch"
./aps run myagent -- codex "write tests"
./aps run myagent -- gemini "summarize docs/user"
./aps run myagent -- opencode "inspect failures"
```

### 4. List Your Profiles

```bash
./aps profile list
```

### 5. View Profile Details

```bash
./aps profile show myagent
```

## What is a Profile?

A profile is an isolated environment that contains:

- **Identity**: Git config, GitHub username, etc.
- **Credentials**: API tokens, SSH keys, etc.
- **Preferences**: Language, timezone, shell
- **Capabilities**: What the agent can do
- **Actions**: Custom scripts the agent can run

## Environment Variables

APS automatically injects profile-specific environment variables when running commands:

- `APS_PROFILE_ID` - The profile ID
- `APS_PROFILE_DIR` - Path to the profile directory
- `APS_PROFILE_YAML` - Path to profile.yaml
- `APS_PROFILE_SECRETS` - Path to secrets.env
- `APS_PROFILE_DOCS_DIR` - Path to docs directory

Plus any secrets you define in `secrets.env`.

External LLM CLIs launched through `aps run` or shorthand execution inherit
the same environment. Use this for Claude Code, Codex, Gemini CLI, OpenCode,
or similar tools until native `aps chat <profile>` is available.

## Next Steps

- Read [CLI.md](CLI.md) for complete command reference
- Read [PROFILES.md](PROFILES.md) for detailed profile management
- Read [ISOLATION.md](ISOLATION.md) for isolation levels and security
- Read [SESSIONS.md](SESSIONS.md) for session management
- Read [EXAMPLES.md](EXAMPLES.md) for practical use cases
- Read [WEBHOOKS.md](WEBHOOKS.md) to set up webhooks
- Read [SECURITY.md](SECURITY.md) for security best practices

## Shell Integration

### Auto-completion

```bash
# For zsh
echo 'source <(./aps completion zsh)' >> ~/.zshrc

# For bash
echo 'source <(./aps completion bash)' >> ~/.bashrc
```

### Profile Aliases

Create aliases for quick access:

```bash
eval "$(./aps alias)"
```

This lets you run:
```bash
myagent echo "Hello!"
myagent git status
```

## Configuration

APS supports global configuration at `~/.config/aps/config.yaml`:

```yaml
prefix: MYTOOL
isolation:
  default_level: process  # process | platform | container
  fallback_enabled: true   # Allow fallback to lower isolation levels
```

This changes environment variables from `APS_*` to `MYTOOL_*` and configures default isolation behavior.

### Profile Isolation

Profiles can also specify isolation settings in `profile.yaml`:

```yaml
isolation:
  level: process  # process | platform | container
  strict: false   # Fail if requested level is unavailable
  fallback: true  # Allow fallback to lower isolation levels
```

See [ISOLATION.md](ISOLATION.md) for details.

## Directory Structure

All APS data lives under `~/.agents/` and `~/.aps/`:

```
~/.agents/
  profiles/
    myagent/
      profile.yaml
      secrets.env
      gitconfig
      actions/
      notes.md
  docs/
    README.md
    CLI.md
    PROFILES.md
    ISOLATION.md
    SESSIONS.md
    SECURITY.md
    EXAMPLES.md
    WEBHOOKS.md

~/.aps/
  sessions/
    registry.json
  keys/
    <session-id>/
      admin_key
      admin_key.pub
  tmux.conf
```

## TUI Mode

Run APS without arguments to launch the interactive Terminal UI:

```bash
./aps
```

The TUI provides:
- Profile selection
- Profile details
- Action list and execution
- Log output viewer
