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
- `~/.agents/docs/EXAMPLES.md` - Practical examples
- `~/.agents/docs/WEBHOOKS.md` - Webhook setup guide
- `~/.agents/docs/SECURITY.md` - Security best practices

## Features

- **Profile Isolation**: Separate environments for different agents, environments, or contexts
- **Secrets Management**: Secure credential storage with automatic environment injection
- **Action Automation**: Custom scripts triggered by CLI or webhooks
- **Git Integration**: Automatic gitconfig and SSH key management
- **Webhook Support**: Event-driven automation from GitHub, GitLab, and more
- **TUI Interface**: Interactive terminal user interface for easy profile management

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
```

This changes environment variables from `APS_*` to `MYTOOL_*`.

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
