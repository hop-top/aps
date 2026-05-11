# A2A Protocol Implementation Guide

**For Developers**: Technical implementation details for the A2A protocol integration in APS.

---

## Overview

This document provides technical details for developers working on or extending the A2A protocol implementation in APS.

## Current Maturity

The current user-facing A2A server is a reachable HTTP JSON-RPC listener with
filesystem-backed task storage and agent-card discovery. Task execution is
placeholder-level: `internal/a2a/executor.go` emits status transitions and an
agent text message shaped as `Processed: <input>`. It does not invoke profile
actions, chat, or LLM-backed work yet.

Supported through the normal CLI path:

| Capability | Current status |
| --- | --- |
| Server listener | Ready for HTTP JSON-RPC via `aps a2a server --profile <id>` |
| Agent card | Ready at `/.well-known/agent-card` |
| Task get/list/cancel | Implemented with filesystem storage |
| Streaming status events | Implemented by the A2A server/executor path |
| Push configuration methods | Stored in memory on the running server; webhook delivery is not implemented |
| Profile-backed execution | Placeholder; no profile action/chat dispatch |
| IPC/gRPC transport adapters | Component-level transport package code and tests; not the normal server path |
| API key auth helper | Component-level outbound request helper; not enforced by `aps a2a server` |
| mTLS/OpenID/OAuth | Planned/component-level only; not enforced by `aps a2a server` |

## Architecture

### Component Structure

```
internal/a2a/
├── config.go              # Configuration types and defaults
├── agentcard.go          # Agent Card generation
├── server.go             # A2A server implementation
├── client.go             # A2A client implementation
├── executor.go           # Task executor
├── storage.go            # Filesystem-based task storage
├── resolver.go           # Agent Card resolution
├── cache.go              # Agent Card caching
├── errors.go             # Error types
├── isolation.go          # Isolation tier mapping
└── transport/
    ├── interface.go      # Transport abstraction
    ├── ipc.go           # IPC transport (filesystem queues)
    ├── http.go          # HTTP/JSON-RPC transport
    ├── grpc.go          # gRPC transport
    ├── selector.go      # Transport selection logic
    └── auth.go          # Authentication helpers
```

### Data Flow

#### Server Request Processing

```
HTTP Request
    ↓
a2asrv.Handler
    ↓
Server.OnSendMessage()
    ↓
Executor.Execute()
    ↓
Storage.Save()
    ↓
Task Response
```

#### Client Task Creation

```
Client.SendMessage()
    ↓
a2aclient.Client
    ↓
Agent Card preferred transport
    ↓
Target Server
    ↓
Task Response
```

---

## Core Components

### Server Implementation

**File**: `internal/a2a/server.go`

The server uses the official `a2asrv` package and implements `a2asrv.RequestHandler`:

```go
type Server struct {
    profileID    string
    profile      *core.Profile
    storage      *Storage
    executor     *Executor
    queueManager eventqueue.Manager
    httpServer   *http.Server
    pushConfigs  map[string]*a2a.TaskPushConfig
}
```

**Key Methods**:
- `OnSendMessage`: Handle synchronous task creation
- `OnSendMessageStream`: Handle streaming task creation
- `OnGetTask`: Retrieve task details
- `OnCancelTask`: Cancel running task
- `OnSetTaskPushConfig`: Configure push notifications
- `OnGetExtendedAgentCard`: Return Agent Card

**Lifecycle**:
1. Create server with `NewServer(profile, config)`
2. Start HTTP server with `Start(ctx)`
3. Server exposes:
   - JSON-RPC endpoint at `/`
   - Agent Card at `/.well-known/agent-card`
4. Stop with `Stop()`

### Client Implementation

**File**: `internal/a2a/client.go`

The client wraps the official `a2aclient.Client`:

```go
type Client struct {
    profileID string
    profile   *core.Profile
    card      *a2a.AgentCard
    client    *a2aclient.Client
}
```

**Creation Flow**:
1. Load target profile
2. Generate/fetch Agent Card
3. Validate transport configuration
4. Create `a2aclient.Client` from Agent Card

**SDK Compatibility Note**:
SDK v0.3.4 returns `Message` from `SendMessage` instead of `Task`. The client handles both types for compatibility (see `client.go:109-129`).

### Storage Implementation

**File**: `internal/a2a/storage.go`

Custom filesystem-based `TaskStore` implementation:

```go
type Storage struct {
    config *StorageConfig
}

// Implements a2asrv.TaskStore
func (s *Storage) Save(ctx context.Context, task *a2a.Task, event a2a.Event, prev a2a.TaskVersion) (a2a.TaskVersion, error)
func (s *Storage) Get(ctx context.Context, taskID a2a.TaskID) (*a2a.Task, a2a.TaskVersion, error)
func (s *Storage) List(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error)
```

**Directory Structure**:
```
~/.local/share/aps/a2a/<profile-id>/
  tasks/<task-id>/
    meta.json           # Task metadata (JSON serialized)
    event_*.json        # Event history
  agent-cards/
    <agent-id>.json    # Cached Agent Cards
```

**Implementation Details**:
- JSON serialization for human readability
- File permissions: 0700 (dirs), 0600 (files)
- Task versioning via increment (simple monotonic counter)
- No database required

### Executor Implementation

**File**: `internal/a2a/executor.go`

Implements the task execution logic:

```go
type Executor struct {
    profile *core.Profile
    storage *Storage
}

// Implements a2asrv.AgentExecutor
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error
```

**Execution Flow**:
1. Receive task via `RequestContext`
2. Emit `TaskStatusUpdateEvent` (submitted → working)
3. Process the first text part with placeholder text handling
4. Emit `TaskStatusUpdateEvent` (working → completed)
5. Save to storage

Current response behavior:

```text
Processed: <first text part>
```

Do not document this executor as profile-backed task execution until a routing
contract exists and the executor calls the profile action/chat path.

**Extension Points**:
- Override `Execute` to implement custom task processing
- Use `queue.Write()` to emit progress events
- Access `reqCtx.Message` for task input
- Access `reqCtx.StoredTask` for task history

---

## Transport Layer

### Interface

**File**: `internal/a2a/transport/interface.go`

```go
type Transport interface {
    Send(ctx context.Context, endpoint string, task *a2a.Task) error
    Receive(ctx context.Context) (*a2a.Task, error)
}
```

### IPC Transport

**File**: `internal/a2a/transport/ipc.go`

Filesystem-based message queue for process isolation:

```go
type IPCTransport struct {
    queueDir string
}
```

**Implementation**:
- Queue directory: `<data>/ipc/queues/<profile-id>/incoming/`
- Message format: JSON files named `<timestamp>_<uuid>.json`
- Polling interval: 100ms
- File permissions: 0700

**Current status**: Component-level. The package has transport code and unit
coverage, but `aps a2a server` starts the HTTP JSON-RPC server path rather than
an IPC listener.

### HTTP Transport

**File**: `internal/a2a/transport/http.go`

JSON-RPC 2.0 over HTTP:

```go
type HTTPTransport struct {
    client *http.Client
    url    string
}
```

**Implementation**:
- Protocol: JSON-RPC 2.0
- Default endpoint: `http://127.0.0.1:8081/`
- Content-Type: `application/json`

**Current status**: Component-level helper for direct transport use. The
normal CLI server path is the A2A SDK JSON-RPC handler mounted by
`internal/a2a/server.go`.

### gRPC Transport

**File**: `internal/a2a/transport/grpc.go`

Protocol Buffers over HTTP/2:

```go
type GRPCTransport struct {
    conn *grpc.ClientConn
}
```

**Implementation**:
- Protocol: gRPC (HTTP/2 + Protobuf)
- Connection pooling via `grpc.Dial`
- Default address: `127.0.0.1:8081`

**Current status**: Component-level. Do not describe gRPC as a ready
user-facing A2A server transport until a CLI/server path mounts it.

### Transport Selection

**File**: `internal/a2a/transport/selector.go`

Transport selection helpers exist for:
1. Profile's `protocol_binding` setting
2. Isolation tier mapping
3. Agent Card `PreferredTransport`

```go
func SelectTransport(profile *core.Profile, card *a2a.AgentCard) (Transport, error)
```

**Component logic**:
- Process (Tier 1) → IPC
- Platform (Tier 2) → HTTP or gRPC (based on config)
- Container (Tier 3) → HTTP or gRPC with mTLS

This selector is not the same as the current `aps a2a server` startup path.

---

## Agent Card Generation

**File**: `internal/a2a/agentcard.go`

Generates valid A2A Agent Card from APS profile:

```go
func GenerateAgentCardFromProfile(profile *core.Profile) (*a2a.AgentCard, error)
```

**Mapping**:

| Profile Field | Agent Card Field |
|---------------|------------------|
| `profile.ID` | Used for agent identity |
| `profile.DisplayName` | `Description` |
| `profile.A2A.ListenAddr` | `URL` |
| `profile.A2A.ProtocolBinding` | `PreferredTransport` |
| `profile.Capabilities` | `Capabilities.Extensions` |

**Capabilities**:
- Streaming: `true` (via Go iterators)
- Push Notifications: `true` in the generated card, but current support is
  limited to storing push configuration on the running server. Webhook delivery
  for task updates is not implemented.
- State Transition History: `false` (not implemented)

**Validation**:
- Ensures required fields present
- Validates URL format
- Checks transport protocol supported
- See `agentcard_validation_test.go` for validation rules

---

## Configuration

### Profile A2A Config

**Type**: `core.A2AConfig` (in `internal/core/profile.go`)

```go
type A2AConfig struct {
    ProtocolBinding string `yaml:"protocol_binding,omitempty"`
    ListenAddr      string `yaml:"listen_addr,omitempty"`
    PublicEndpoint  string `yaml:"public_endpoint,omitempty"`
    SecurityScheme  string `yaml:"security_scheme,omitempty"`
    IsolationTier   string `yaml:"isolation_tier,omitempty"`
}
```

**Example**:
```yaml
a2a:
  protocol_binding: "jsonrpc"  # or "grpc", "http"
  listen_addr: "127.0.0.1:8081"
  public_endpoint: "http://localhost:8081"
```

### Storage Config

**Type**: `a2a.StorageConfig`

```go
type StorageConfig struct {
    BasePath       string
    TasksPath      string
    AgentCardsPath string
    IPCPath        string
}
```

**Default Paths**:
```go
basePath := filepath.Join(core.GetAgentsDir(), "a2a", profile.ID)
config := &a2a.StorageConfig{
    BasePath:       basePath,
    TasksPath:      filepath.Join(basePath, "tasks"),
    AgentCardsPath: filepath.Join(basePath, "agent-cards"),
    IPCPath:        filepath.Join(core.GetAgentsDir(), "ipc", "queues"),
}
```

---

## Error Handling

**File**: `internal/a2a/errors.go`

Custom error types for better error handling:

```go
var (
    ErrA2ANotEnabled        = errors.New("A2A not enabled for profile")
    ErrInvalidConfig        = errors.New("invalid A2A configuration")
    ErrAgentCardNotFound    = errors.New("agent card not found")
    ErrTransportNotSupported = errors.New("transport not supported")
)

func ErrInvalidAgentCard(msg string) error
func ErrStorageFailed(operation string, err error) error
func ErrClientFailed(operation string, err error) error
func ErrInvalidMessage(msg string) error
```

**Usage**:
```go
if profile.A2A == nil {
    return nil, a2a.ErrA2ANotEnabled
}
```

---

## Testing

### Unit Tests

**Location**: `tests/unit/a2a/`

**Coverage**:
- `client_test.go`: Client initialization
- `client_message_test.go`: Message sending
- `transport/ipc_test.go`: IPC transport
- `transport/http_test.go`: HTTP transport
- `transport/grpc_test.go`: gRPC transport
- `transport/selector_test.go`: Transport selection

**Package Tests**:
- `internal/a2a/config_test.go`: Configuration
- `internal/a2a/agentcard_generation_test.go`: Agent Card generation
- `internal/a2a/agentcard_validation_test.go`: Agent Card validation
- `internal/a2a/server_test.go`: Server initialization
- `internal/a2a/cache_test.go`: Caching logic

### E2E Tests

**Location**: `tests/e2e/`

**Coverage**:
- `a2a_server_test.go`: Server integration
- `a2a_client_test.go`: Client integration
- `a2a_transport_test.go`: Transport integration
- `server_helper.go`: Test helpers

**Running Tests**:
```bash
# Unit tests
go test ./internal/a2a/...
go test ./tests/unit/a2a/...

# E2E tests
go test ./tests/e2e/...

# All tests
go test ./...
```

---

## Extending the Implementation

### Adding Custom Executor Logic

```go
type MyExecutor struct {
    *a2a.Executor
}

func (e *MyExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
    // Custom task processing
    message := reqCtx.Message

    // Emit status update
    queue.Write(ctx, &a2a.TaskStatusUpdateEvent{
        TaskID: reqCtx.TaskID,
        Status: a2a.TaskStatus{
            State: a2a.TaskStateWorking,
        },
    })

    // Do work...
    result := processTask(message)

    // Emit completion
    queue.Write(ctx, &a2a.TaskStatusUpdateEvent{
        TaskID: reqCtx.TaskID,
        Status: a2a.TaskStatus{
            State: a2a.TaskStateCompleted,
        },
    })

    return nil
}
```

### Adding Custom Transport

1. Implement `transport.Transport` interface
2. Register in `transport/selector.go`
3. Add configuration option
4. Write tests

### Adding Custom Storage Backend

1. Implement `a2asrv.TaskStore` interface
2. Handle `Save`, `Get`, `List` operations
3. Ensure thread-safety
4. Write tests

---

## Performance Considerations

### Storage

- **Filesystem I/O**: Each task operation involves file reads/writes
- **Optimization**: Use caching for frequently accessed tasks
- **Scalability**: Consider database backend for high-volume scenarios

### Transports

- **IPC**: Polling-based (100ms), low latency for local communication
- **HTTP**: Connection overhead, suitable for cross-machine
- **gRPC**: Best performance, connection pooling, HTTP/2 multiplexing

### Memory

- **Event Queues**: In-memory queues per task (bounded size recommended)
- **Agent Cards**: Cached in memory and filesystem
- **Tasks**: Loaded on-demand, not kept in memory

---

## Security

### File Permissions

```go
os.MkdirAll(dir, 0700)  // Directories: owner only
os.WriteFile(file, data, 0600)  // Files: owner read/write only
```

### Authentication

**Component support** (via `transport/auth.go`):
- API Key can be applied to outbound HTTP requests with `X-API-Key`.
- mTLS configuration can be validated from profile secrets, but it is not wired
  into the current A2A server listener.
- OpenID is represented as a config enum, but `ApplyAuth` does not implement
  it.

**Configuration**:
```yaml
a2a:
  security_scheme: "apikey"
```

### Network Security

- Use HTTPS for production deployments
- Treat mTLS/OpenID/OAuth as planned until the server and client paths enforce
  them end to end
- Validate Agent Cards before connecting
- Sanitize task inputs

---

## Debugging

### Enable Debug Logging

```go
import "github.com/a2aproject/a2a-go/log"

log.SetLevel(log.LevelDebug)
```

### Inspect Task Storage

```bash
# View task metadata
cat ~/.local/share/aps/a2a/<profile-id>/tasks/<task-id>/meta.json | jq

# View events
ls -la ~/.local/share/aps/a2a/<profile-id>/tasks/<task-id>/event_*.json
```

### Network Debugging

```bash
# Test Agent Card endpoint
curl http://localhost:8081/.well-known/agent-card | jq

# Monitor HTTP requests (if using HTTP transport)
tcpdump -i lo0 -A 'port 8081'
```

---

## References

- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [a2a-go SDK Documentation](https://pkg.go.dev/github.com/a2aproject/a2a-go)
- [APS A2A Specification](../../specs/005-a2a-protocol/spec.md)
- [Test Examples](../../tests/unit/a2a/)
