<p>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="#" width="340">
    <source media="(prefers-color-scheme: light)" srcset="#" width="340">
    <img src="#" width="340" alt="APS Logo">
  </picture>
  <br>
  <a href="https://github.com/hop-top/aps/releases">
    <img src="https://img.shields.io/github/release/hop-top/aps.svg" alt="Latest Release">
  </a>
  <a href="https://pkg.go.dev/hop.top/aps">
    <img src="https://godoc.org/hop.top/aps?status.svg" alt="GoDoc">
  </a>
  <a href="https://github.com/hop-top/aps/actions">
    <img src="https://github.com/hop-top/aps/workflows/CI/badge.svg" alt="Build Status">
  </a>
  <a href="https://codecov.io/gh/hop-top/aps">
    <img src="https://codecov.io/gh/hop-top/aps/branch/main/graph/badge.svg" alt="Coverage">
  </a>
</p>

# APS (Agent Profile System)

> [!WARNING]
> **🚧 Do Not Use — History Will Be Rewritten 🚧**
>
> This repo is undergoing major restructuring as we selectively
> open-source internal tools built at
> [Idea Crafters LLC](https://ideacrafters.com). Git history **will be
> force-pushed and rewritten** multiple times. Do not fork, clone, or
> depend on this repo in any capacity until we tag a stable release.

APS is a local-first Agent Profile System that enables running commands and agent workflows under isolated profiles.

## Quick Start

### Install

```bash
# Build from source
git clone https://github.com/hop-top/aps.git
cd aps
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
- `docs/dev/implementation/guides/migration-guide.md` - Migration guide from process to platform/container isolation
- `docs/dev/testing/performance-benchmarks.md` - Performance benchmarks and optimization
- `docs/dev/security/security-audit.md` - Comprehensive security audit report
- `docs/dev/operations/releases/release-notes.md` - Release notes and version history
- `docs/dev/platforms/macos/overview.md` - macOS platform isolation setup
- `docs/dev/platforms/linux/overview.md` - Linux platform isolation setup
- `docs/dev/platforms/container/overview.md` - Container isolation setup and configuration

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
- **Voice Sessions**: Speech-to-speech backend integration (PersonaPlex, Moshi) with web, terminal, messenger, and telephony channels
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
    # See docs/dev/platforms/macos/overview.md or docs/dev/platforms/linux/overview.md
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
- Go 1.25.5+
- [mise](https://mise.jdx.dev) (recommended for dev tools)

```bash
# Install all required tools (go, goreleaser, act, etc.)
mise install
```

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

# Docker user journey tests
make docker-test-e2e-user
```

### Docker Testing

APS includes a Docker-based testing environment for testing in an isolated Linux environment that simulates a user's machine. This is particularly useful for testing installation, setup, and workflows without affecting your local development environment.

```bash
# Build the test environment
make docker-build-test

# Run all user journey tests
make docker-test-e2e-user

# Start an interactive test environment
make docker-test-up
make docker-test-shell  # For manual testing

# Cleanup
make docker-test-cleanup
```

For detailed documentation, see [Docker testing user guide](docs/agent/docker-testing.md) or [Docker testing strategy for developers](docs/dev/testing/docker-testing-strategy.md).

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
aps voice service start|stop|status  # Manage voice backend
aps voice start [--profile <id>] [--channel web|tui|telegram|twilio]
aps voice session list   # List active voice sessions
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
- `docs/dev/platforms/macos/overview.md` - macOS sandbox examples
- `docs/dev/platforms/linux/overview.md` - Linux sandbox examples
- `docs/dev/platforms/container/overview.md` - Container isolation examples

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

See `docs/dev/testing/performance-benchmarks.md` for detailed benchmarks.

### Platform Isolation

- **macOS**: User account sandboxing via `dscl`, ACL-based access control
- **Linux**: User account isolation via `useradd`, namespace/chroot/cgroups support
- Requires: Admin/sudo access, SSH server, admin public key

### Container Isolation

- Docker-based isolation with automatic Dockerfile generation
- Volume mounting, network configuration, resource limits
- SSH server and tmux integration for session management
- Requires: Docker installed

See `docs/dev/platforms/container/overview.md`, `docs/dev/platforms/macos/overview.md`, and `docs/dev/platforms/linux/overview.md` for platform-specific setup guides.

## Capability Management

APS can manage external tools, configurations, and dotfiles as "Capabilities". These are stored in `~/.aps/capabilities/` and can be linked into your workspace.

### Commands

```bash
# Install a capability from a directory
aps capability install ./my-tool --name my-tool

# List installed capabilities
aps capability list

# Link a capability to your current workspace
aps capability link my-tool --target ./local/path

# "Smart Link" a known tool (e.g., windsurf, copilot)
# This automatically resolves the target path based on standard conventions
aps capability link copilot

# "Adopt" an existing file into APS management
aps capability adopt ./my-config.yaml --name my-config

# "Watch" an external file (symlink into APS)
aps capability watch ./external/file --name my-ref

# Delete a capability
aps capability delete my-tool
```

### Environment Integration

You can automatically export environment variables for all your capabilities (e.g., `APS_MY_TOOL_PATH`).

Add this to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
eval "$(aps env)"
```

This ensures that whenever you install or remove a capability, your environment variables are updated (on next shell load or re-eval).

### Configuration

You can configure additional source directories for capabilities in `~/.config/aps/config.yaml`:

```yaml
capability_sources:
  - /shared/team/capabilities
  - ~/personal/capabilities
```

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

Contributions are welcome!

### Documentation Structure

*   **[User Documentation](docs/user/README.md)**: Guides for installing and using APS.
*   **[Developer Documentation](docs/dev/readme.md)**: Architecture, design specs, and implementation details.
*   **[Agent Documentation](docs/agent/README.md)**: Context and patterns for AI agents working on the codebase.

See `docs/dev/operations/releases/release-notes.md` for recent changes and version history.

For developers working on the codebase, see `AGENTS.md` for implementation guidance and architecture details.

## License

[MIT](https://github.com/hop-top/aps/raw/main/LICENSE)

## Support

For issues, questions, or contributions, please visit:
- GitHub: https://github.com/hop-top/aps
- Issues: https://github.com/hop-top/aps/issues
