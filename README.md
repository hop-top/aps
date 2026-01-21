# APS (Agent Profile System)

APS is a local-first Agent Profile System that enables running commands and agent workflows under isolated profiles.

## Quick Start

### Install

```bash
# Build from source
git clone https://github.com/IdeaCraftersLabs/oss-aps-cli.git
cd oss-aps-cli
make build
```

### Create Your First Profile

```bash
./aps profile new myagent --display-name "My AI Agent" --email "agent@example.com"
```

### Run Commands

```bash
# Run a command under your profile
./aps myagent -- echo "Hello from agent!"

# Run git with profile's git config
./aps myagent -- git status

# Start an interactive shell
./aps myagent
```

### Generate Documentation

```bash
./aps docs
```

Documentation will be generated at `~/.agents/docs/`.

**Then read:**
- `~/.agents/docs/README.md` - Getting started guide
- `~/.agents/docs/CLI.md` - Complete command reference
- `~/.agents/docs/PROFILES.md` - Profile management guide
- `~/.agents/docs/ISOLATION.md` - Isolation levels and configuration
- `~/.agents/docs/SESSIONS.md` - Session management guide
- `~/.agents/docs/EXAMPLES.md` - Practical examples
- `~/.agents/docs/WEBHOOKS.md` - Webhook setup guide
- `~/.agents/docs/SECURITY.md` - Security best practices

## Features

- **Profile Isolation**: Separate environments for different agents, environments, or contexts with configurable isolation levels (process, platform, container)
- **Session Management**: Track and manage long-running sessions with attach/detach support
- **Secrets Management**: Secure credential storage with automatic environment injection
- **Action Automation**: Custom scripts triggered by CLI or webhooks
- **Git Integration**: Automatic gitconfig and SSH key management
- **Webhook Support**: Event-driven automation from GitHub, GitLab, and more
- **TUI Interface**: Interactive terminal user interface for easy profile management
- **Graceful Degradation**: Automatic fallback to available isolation levels when requested level is unavailable

## Directory Structure

All APS data lives under `~/.agents/`:

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
    SECURITY.md
    EXAMPLES.md
    WEBHOOKS.md
    ISOLATION.md
    SESSIONS.md

~/.aps/
  sessions/
    registry.json
  keys/
    <session-id>/
      admin_key
      admin_key.pub
  tmux.conf
```

## Shell Integration

### Auto-completion

```bash
# For zsh
echo 'source <(./aps completion zsh)' >> ~/.zshrc

# For bash
echo 'source <(./aps completion bash)' >> ~/.bashrc
```

### Profile Aliases

```bash
eval "$(./aps alias)"
```

Then run:
```bash
myagent echo "Hello!"
```

## Configuration

Global configuration at `~/.config/aps/config.yaml`:

```yaml
prefix: MYTOOL
isolation:
  default_level: process  # process | platform | container
  fallback_enabled: true   # Allow fallback to lower isolation levels
```

This changes environment variables from `APS_*` to `MYTOOL_*` and configures the default isolation behavior.

### Profile Isolation

Profiles can also specify isolation settings in `profile.yaml`:

```yaml
isolation:
  level: process  # process | platform | container
  strict: false   # Fail if requested level is unavailable
  fallback: true  # Allow fallback to lower isolation levels
```

## Build Instructions

### Prerequisites
- Go 1.22+

### Build for all platforms

```bash
make build
```

### Build for current platform

```bash
make build-local
```

### Test

```bash
go test -v ./tests/e2e
```

## Commands

```bash
aps                    # Launch TUI
aps help               # Show help
aps profile list       # List all profiles
aps profile new <id>   # Create a new profile
aps profile show <id>  # Show profile details
aps run <id> -- <cmd>  # Run command under profile
aps action list <id>   # List profile actions
aps action run <id> <action>  # Run action
aps session list      # List active sessions
aps session attach <id>  # Attach to a session
aps session detach <id>  # Detach from a session
aps docs               # Generate documentation
```

## Examples

See `~/.agents/docs/EXAMPLES.md` for detailed examples including:
- GitHub issue triage
- OpenAI chat assistant
- Data processing pipelines
- Multi-environment deployment
- Webhook integrations
- And more

## Isolation Levels

APS supports multiple isolation levels for running commands and actions:

- **process** (default): Runs commands in isolated processes with injected environment
- **platform**: Uses platform-specific sandboxing (coming soon)
- **container**: Runs commands in isolated containers (coming soon)

If the requested isolation level is unavailable, APS can gracefully degrade to a lower level based on configuration.

## Session Management

APS provides session tracking for long-running operations:

```bash
# List all active sessions
aps session list

# List sessions for a specific profile
aps session list --profile myagent

# Filter by status or tier
aps session list --status active
aps session list --tier premium
```

Sessions are tracked in `~/.aps/sessions/registry.json` and include metadata like PID, status, and heartbeat tracking.

## Testing

To run the E2E test suite:

```bash
go test -v ./tests/e2e
```

## Contributing

Contributions are welcome! Please read the [specification](spec.md) for detailed implementation requirements.

## License

MIT

## Support

For issues, questions, or contributions, please visit:
- GitHub: https://github.com/IdeaCraftersLabs/oss-aps-cli
- Issues: https://github.com/IdeaCraftersLabs/oss-aps-cli/issues
