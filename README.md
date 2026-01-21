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

### Choose Your Isolation Level

```bash
# Process isolation (default, fastest)
./aps profile new myagent

# Platform isolation (macOS/Linux only, user-level sandbox)
./aps profile new myagent --isolation-level platform

# Container isolation (strongest isolation, requires Docker)
./aps profile new myagent --isolation-level container
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

**Additional documentation available in the repository:**
- `docs/MIGRATION.md` - Migration guide from process to platform/container isolation
- `docs/PERFORMANCE.md` - Performance benchmarks and optimization
- `docs/SECURITY_AUDIT.md` - Comprehensive security audit report
- `docs/RELEASE_NOTES.md` - Release notes and version history
- `docs/platforms/macos.md` - macOS platform isolation setup
- `docs/platforms/linux.md` - Linux platform isolation setup
- `docs/isolation/container.md` - Container isolation setup and configuration

## Features

- **Profile Isolation**: Separate environments for different agents, environments, or contexts with configurable isolation levels (process, platform, container)
- **Platform Isolation**: User-level sandboxing on macOS (via `dscl`) and Linux (via `useradd`) with ACL-based access control
- **Container Isolation**: Docker-based isolation with automatic Dockerfile generation, volume mounting, and resource limits
- **Session Management**: Track and manage long-running sessions with inspect, logs, and terminate commands
- **Session Attachment**: SSH-based session attachment for platform and container isolation
- **Secrets Management**: Secure credential storage with automatic environment injection
- **Action Automation**: Custom scripts triggered by CLI or webhooks
- **Git Integration**: Automatic gitconfig and SSH key management
- **Webhook Support**: Event-driven automation from GitHub, GitLab, and more
- **TUI Interface**: Interactive terminal user interface for easy profile management
- **Graceful Degradation**: Automatic fallback to available isolation levels when requested level is unavailable
- **Cross-Platform Support**: macOS, Linux, and Windows (platform isolation varies by OS)

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
  platform:
    # macOS/Linux specific settings
    # See docs/platforms/macos.md or docs/platforms/linux.md
  container:
    # Container specific settings
    image: "ubuntu:22.04"    # Base Docker image
    volumes:                # Volume mounts
      - source: /tmp
        target: /workspace
    resources:              # Resource limits
      memory: "1g"
      cpu: "0.5"
```

## Build Instructions

### Prerequisites
- Go 1.22+

### Platform Requirements

**macOS Platform Isolation:**
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools
- OpenSSH Server (`sudo systemsetup -setremotelogin on`)

**Linux Platform Isolation:**
- Linux kernel 3.10+
- `useradd`, `setfacl` commands
- OpenSSH Server

**Container Isolation:**
- Docker installed and running
- Admin privileges for Docker operations

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
# Unit tests
go test -v ./tests/unit

# E2E tests
go test -v ./tests/e2e

# Platform-specific tests (macOS only)
go test -v -tags darwin ./tests/unit/core/isolation

# Platform-specific tests (Linux only)
go test -v -tags linux ./tests/unit/core/isolation
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
aps session inspect <id> # Inspect session details
aps session logs <id>    # Show session logs
aps session terminate <id> # Terminate a session
aps docs               # Generate documentation
```

## Examples

See `~/.agents/docs/EXAMPLES.md` for detailed examples including:
- GitHub issue triage
- OpenAI chat assistant
- Data processing pipelines
- Multi-environment deployment
- Webhook integrations
- Containerized workflows
- Platform isolation use cases
- And more

**Platform-specific examples:**
- `docs/platforms/macos.md` - macOS sandbox examples
- `docs/platforms/linux.md` - Linux sandbox examples
- `docs/isolation/container.md` - Container isolation examples

## Isolation Levels

APS supports multiple isolation levels for running commands and actions:

- **process** (default): Runs commands in isolated processes with injected environment
- **platform**: Uses platform-specific sandboxing (macOS, Linux)
- **container**: Runs commands in isolated containers (Docker)

If the requested isolation level is unavailable, APS can gracefully degrade to a lower level based on configuration.

### Performance Comparison

| Isolation Level | Setup Time | Execution Overhead | Memory Overhead |
|---------------|------------|-------------------|----------------|
| Process | 0ms | < 5ms | ~10MB |
| Platform | 150-400ms* / < 50ms | 10-40ms | ~50-60MB |
| Container | 2-5s / 100-500ms | 20-50ms | 100-200MB |

\*First run includes user account creation. Subsequent runs are cached.

See `docs/PERFORMANCE.md` for detailed benchmarks.

### Platform Isolation

- **macOS**: User account sandboxing via `dscl`, ACL-based access control
- **Linux**: User account isolation via `useradd`, namespace/chroot/cgroups support
- Requires: Admin/sudo access, SSH server, admin public key

### Container Isolation

- Docker-based isolation with automatic Dockerfile generation
- Volume mounting, network configuration, resource limits
- SSH server and tmux integration for session management
- Requires: Docker installed

See `docs/isolation/container.md`, `docs/platforms/macos.md`, and `docs/platforms/linux.md` for platform-specific setup guides.

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

# Inspect session details
aps session inspect <session-id>

# View session logs
aps session logs <session-id>

# Terminate a session gracefully
aps session terminate <session-id>
```

### Session Attachment

- **Process Isolation**: Direct shell access
- **Platform Isolation**: SSH to sandbox user (macOS/Linux)
- **Container Isolation**: SSH into container

Sessions are tracked in `~/.aps/sessions/registry.json` and include metadata like PID, status, and heartbeat tracking.

## Testing

To run the E2E test suite:

```bash
go test -v ./tests/e2e
```

## Contributing

Contributions are welcome! Please read the [specification](spec.md) for detailed implementation requirements.

See `docs/RELEASE_NOTES.md` for recent changes and version history.

For developers working on the codebase, see `AGENTS.md` for implementation guidance and architecture details.

## License

MIT

## Support

For issues, questions, or contributions, please visit:
- GitHub: https://github.com/IdeaCraftersLabs/oss-aps-cli
- Issues: https://github.com/IdeaCraftersLabs/oss-aps-cli/issues
