# oss-aps-cli Development Guidelines

## Active Technologies

- **Go 1.25.5**: Core application language.
- **Cobra**: CLI framework for subcommands and flags.
- **Bubble Tea / Lip Gloss**: TUI framework for interactive screens.
- **YAML v3**: Configuration parsing for `profile.yaml` and global `config.yaml`.
- **GoDotEnv**: Secrets parsing from `secrets.env`.
- **Testify**: Assertion library for unit and E2E tests.
- **XDG Base Directory**: Standardized configuration discovery via `os.UserConfigDir()`.

## Project Structure

```text
bin/                 # Compiled binaries
cmd/aps/             # CLI Entry point
internal/
  cli/               # Cobra command definitions
  core/              # Core business logic (Profile, Config, Execution, Webhooks)
  tui/               # Bubble Tea models and views
specs/               # Feature specifications and implementation plans
tests/
  e2e/               # End-to-end integration tests
  unit/              # Centralized unit tests
```

For the most accurate representation of the codebase, see @.xray map.

Use `xray --help` for more.

## Commands

- `make build`: Build binaries for all platforms.
- `make build-local`: Build binary for current platform.
- `go test ./tests/unit/...`: Run all unit tests.
- `go test -v ./tests/e2e`: Run full E2E test suite.
- `go test ./...`: Run all tests (unit and E2E).
- `go fmt ./...`: Format all source code.
- `go vet ./...`: Run static analysis.

## Code Style

- **Standard Go Conventions**: Follow `Effective Go`.
- **Internal Package**: Keep core logic in `internal/core` to prevent external imports.
- **TDD-First**: Write failing tests before implementing feature logic.
- **Environment Prefixes**: Use dynamic prefixes (default `APS_`) for injected variables.
- **Security**: Strictly enforce `0600` permissions for secret files.


<!-- MANUAL ADDITIONS START -->
- **Secrets Management**: Always redact secret values when printing to stdout/stderr.
- **TUI/CLI Parity**: Every feature exposed in TUI must also be accessible via a scriptable CLI command.
- **Git Commit Messages**: Stick to conventional commit messages and NEVER include co-author.

## A2A Protocol Integration

### Implementation Status (2026-01-30)

**Complete**:
- ✅ Phase 1-3: Setup, Foundation, Agent Cards
- ✅ Phase 4: A2A Server (full implementation)
- ✅ Phase 5: A2A Client (core operations)
- ✅ Phase 6: Transport Adapters (IPC, HTTP, gRPC)
- ✅ Phase 7: CLI Integration (all commands)

**Pending**:
- ⏳ Phase 8: Documentation and examples
- ⏳ E2E integration tests for CLI commands
- ⏳ Quickstart guide verification

### A2A Architecture Guidelines

**SDK Version**: `github.com/a2aproject/a2a-go@v0.3.4`

**Key Components**:
- `internal/a2a/server.go`: A2A server implementation using `a2asrv`
- `internal/a2a/client.go`: A2A client implementation using `a2aclient`
- `internal/a2a/agentcard.go`: Agent Card generation from profiles
- `internal/a2a/storage.go`: Custom filesystem-based task storage
- `internal/a2a/executor.go`: Profile executor for task execution
- `internal/cli/a2a/`: CLI command group for A2A operations

**Storage Structure**:
```
~/.agents/a2a/<profile-id>/
  tasks/<task-id>/
    meta.json              # Task metadata
    messages/*.json        # Message history
    event_*.json          # Event history
  agent-cards/            # Cached Agent Cards
```

**CLI Commands**:
- `aps a2a server -p <profile>`: Start A2A server
- `aps a2a send-task -t <target> -m <message>`: Send task message
- `aps a2a list-tasks -p <profile>`: List tasks
- `aps a2a get-task <task-id> -p <profile>`: Get task details
- `aps a2a cancel-task <task-id> -t <target>`: Cancel task
- `aps a2a show-card -p <profile>`: Show Agent Card
- `aps a2a fetch-card -u <url>`: Fetch remote Agent Card
- `aps a2a subscribe-task <task-id> -t <target> --webhook <url>`: Subscribe to task

**Transport Selection**:
- Process (Tier 1): Custom IPC via filesystem queues
- Platform (Tier 2): HTTP/JSON-RPC or gRPC
- Container (Tier 3): HTTP/gRPC with mTLS

**Profile Configuration**:
```yaml
a2a:
  enabled: true
  protocol_binding: "jsonrpc"  # or "grpc"
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://localhost:8081"
```

**Agent Card Discovery**:
- Well-known endpoint: `http://<server>/.well-known/agent-card`
- Cached in storage after first resolution
- Auto-generated from profile configuration

### Testing Guidelines

**Unit Tests**:
- `tests/unit/a2a/`: A2A package unit tests
- `tests/unit/a2a/transport/`: Transport adapter tests

**E2E Tests**:
- `tests/e2e/a2a_server_test.go`: Server integration tests
- `tests/e2e/a2a_client_test.go`: Client integration tests
- `tests/e2e/a2a_transport_test.go`: Transport integration tests

**Test Commands**:
```bash
go test ./internal/a2a/...          # A2A package tests
go test ./tests/unit/a2a/...        # Unit tests
go test ./tests/e2e/...             # E2E tests
go test ./...                        # All tests
```

### Implementation Notes

1. **SDK Compatibility**: Using a2a-go v0.3.4, which returns `Message` from `SendMessage` instead of `Task`
2. **Storage Backend**: Custom filesystem implementation (not in-memory) for persistence
3. **Task Lifecycle**: Fully implemented (submitted → working → completed/failed/cancelled)
4. **Streaming**: Implemented using Go 1.23+ iterators (`iter.Seq2`)
5. **Push Notifications**: Configured via `SetTaskPushConfig`
6. **Error Handling**: Custom error types in `internal/a2a/errors.go`

### Common Patterns

**Creating a Server**:
```go
config := &a2a.StorageConfig{BasePath: "~/.agents/a2a/profile-id"}
server, err := a2a.NewServer(profile, config)
server.Start(ctx)
```

**Creating a Client**:
```go
client, err := a2a.NewClient(targetProfileID, targetProfile)
task, err := client.SendMessage(ctx, message)
```

**Generating Agent Card**:
```go
card, err := a2a.GenerateAgentCardFromProfile(profile)
```

### References

- Spec: `specs/005-a2a-protocol/spec.md`
- Plan: `specs/005-a2a-protocol/plan.md`
- Tasks: `specs/005-a2a-protocol/tasks.md`
- Official A2A: https://a2a-protocol.org/latest/specification/
- Go SDK: https://github.com/a2aproject/a2a-go

<!-- MANUAL ADDITIONS END -->
