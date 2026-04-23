# Feature Specification: Agent Protocol Adapter

**Feature Branch**: `006-agent-protocol-adapter`
**Created**: 2026-01-21
**Status**: Draft
**Input**: Adopt LangChain Agent Protocol as API layer with adapter pattern for future protocol support

## Overview

Add an HTTP API layer to APS that implements the [LangChain Agent Protocol](https://langchain-ai.github.io/agent-protocol/) specification. Use an adapter pattern to decouple the protocol-specific HTTP endpoints from the core APS execution engine, enabling future support for alternative protocols (OpenAI Assistants API, etc.) without modifying core logic.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Protocol Adapters                     │
├──────────────────┬───────────────────┬───────────────────┤
│  AgentProtocol   │  OpenAIAssistants │   Future...       │
│  Adapter         │  Adapter          │   Adapter         │
└────────┬─────────┴─────────┬─────────┴─────────┬─────────┘
         │                   │                   │
         ▼                   ▼                   ▼
┌──────────────────────────────────────────────────────────┐
│                   APS Core Interface                      │
│  ExecuteRun() | GetSession() | ListAgents() | Store*()   │
└──────────────────────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────────────────────┐
│                   APS Execution Layer                     │
│  Profiles, Isolation, Secrets, Git Identity, Actions     │
└──────────────────────────────────────────────────────────┘
```

## Requirements

### Functional Requirements

- **FR-001**: System MUST expose a protocol-agnostic core interface (`APSCore`) that adapters translate to/from.
- **FR-002**: System MUST implement an Agent Protocol adapter supporting the Runs, Threads, Store, and Agents endpoints.
- **FR-003**: `POST /runs/wait` MUST execute an action synchronously and return the final result.
- **FR-004**: `POST /runs/stream` MUST execute an action and stream output as Server-Sent Events (SSE).
- **FR-005**: `GET /runs/{run_id}` MUST return run status (pending, running, completed, failed, cancelled).
- **FR-006**: `POST /runs/{run_id}/cancel` MUST terminate a running action.
- **FR-007**: `POST /threads` MUST create a session mapped to a profile.
- **FR-008**: `GET /threads/{thread_id}` MUST return session state.
- **FR-009**: `POST /agents/search` MUST list available profiles with optional filtering.
- **FR-010**: `GET /agents/{agent_id}` MUST return profile metadata (name, description).
- **FR-011**: `GET /agents/{agent_id}/schemas` MUST return JSON Schema for action inputs/outputs.
- **FR-012**: Store endpoints MUST map to profile-scoped key-value storage.
- **FR-013**: System MUST allow multiple adapters to be registered and served simultaneously on different paths or ports.

### Non-Functional Requirements

- **NFR-001**: Adding a new protocol adapter MUST NOT require changes to `internal/core`.
- **NFR-002**: Adapter registration MUST be declarative (config or code, not hardcoded).
- **NFR-003**: Streaming responses MUST not buffer entire output before sending.

## User Scenarios & Testing

### User Story 1 - Stateless Run (Priority: P1)

As an external client, I want to trigger an action via HTTP and receive the result so that I can integrate APS with other systems.

**Acceptance Scenarios**:

1. **Given** a profile `myagent` with action `hello`, **When** I POST to `/runs/wait` with `{"agent_id": "myagent", "input": {"action": "hello"}}`, **Then** I receive the action output in the response body.
2. **Given** a non-existent profile, **When** I POST to `/runs/wait`, **Then** I receive HTTP 404.
3. **Given** an action that fails, **When** I POST to `/runs/wait`, **Then** I receive HTTP 200 with `status: failed` and error details.

---

### User Story 2 - Streaming Run (Priority: P1)

As an external client, I want to stream action output in real-time so that I can display progress to users.

**Acceptance Scenarios**:

1. **Given** a profile `myagent` with action `longrun`, **When** I POST to `/runs/stream`, **Then** I receive SSE events as the action produces output.
2. **Given** a streaming run, **When** the action completes, **Then** I receive a final event with `status: completed`.

---

### User Story 3 - Run Cancellation (Priority: P2)

As an external client, I want to cancel a running action so that I can stop runaway processes.

**Acceptance Scenarios**:

1. **Given** a running action, **When** I POST to `/runs/{run_id}/cancel`, **Then** the process is terminated and status becomes `cancelled`.
2. **Given** an already-completed run, **When** I POST to cancel, **Then** I receive HTTP 400.

---

### User Story 4 - Agent Discovery (Priority: P2)

As an external client, I want to list available agents and their capabilities so that I can build dynamic UIs.

**Acceptance Scenarios**:

1. **Given** profiles `agent-a` and `agent-b`, **When** I POST to `/agents/search`, **Then** I receive both agents with metadata.
2. **Given** profile `agent-a` with actions `foo` and `bar`, **When** I GET `/agents/agent-a/schemas`, **Then** I receive JSON Schema for each action.

---

### User Story 5 - Thread/Session Management (Priority: P3)

As an external client, I want to maintain session state across multiple runs so that I can build multi-turn interactions.

**Acceptance Scenarios**:

1. **Given** no existing threads, **When** I POST to `/threads` with `{"agent_id": "myagent"}`, **Then** a session is created and ID returned.
2. **Given** an existing thread, **When** I POST `/threads/{id}/runs`, **Then** the run executes within that session context.

## Key Entities

### APSCore Interface

```go
type APSCore interface {
    // Runs
    ExecuteRun(ctx context.Context, input RunInput) (*RunHandle, error)
    CancelRun(ctx context.Context, runID string) error
    GetRun(ctx context.Context, runID string) (*RunState, error)

    // Sessions (Threads)
    CreateSession(ctx context.Context, agentID string, metadata map[string]any) (string, error)
    GetSession(ctx context.Context, sessionID string) (*SessionState, error)
    ListSessions(ctx context.Context, filter SessionFilter) ([]SessionInfo, error)

    // Agents
    ListAgents(ctx context.Context, filter AgentFilter) ([]AgentInfo, error)
    GetAgent(ctx context.Context, agentID string) (*AgentInfo, error)
    GetAgentSchemas(ctx context.Context, agentID string) (*AgentSchemas, error)

    // Store
    StoreGet(ctx context.Context, namespace, key string) (any, error)
    StorePut(ctx context.Context, namespace, key string, value any) error
    StoreDelete(ctx context.Context, namespace, key string) error
    StoreSearch(ctx context.Context, namespace string, query string) ([]StoreItem, error)
}
```

### RunInput

```go
type RunInput struct {
    AgentID     string         `json:"agent_id"`
    ActionID    string         `json:"action,omitempty"`
    Input       map[string]any `json:"input"`
    SessionID   string         `json:"thread_id,omitempty"`
    StreamMode  StreamMode     `json:"stream_mode,omitempty"` // none, output, steps
}
```

### RunState

```go
type RunState struct {
    ID        string         `json:"run_id"`
    AgentID   string         `json:"agent_id"`
    SessionID string         `json:"thread_id,omitempty"`
    Status    RunStatus      `json:"status"` // pending, running, completed, failed, cancelled
    Output    any            `json:"output,omitempty"`
    Error     string         `json:"error,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type RunStatus string

const (
    RunStatusPending   RunStatus = "pending"
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCancelled RunStatus = "cancelled"
)
```

### ProtocolAdapter Interface

```go
type ProtocolAdapter interface {
    // Name returns the adapter identifier (e.g., "agent-protocol", "openai-assistants")
    Name() string

    // RegisterRoutes mounts the adapter's HTTP handlers
    RegisterRoutes(mux *http.ServeMux, core APSCore)
}
```

## Agent Protocol Endpoints

The Agent Protocol adapter implements these endpoints per the [specification](https://github.com/langchain-ai/agent-protocol):

### Runs

| Method | Path | Description | APS Mapping |
|--------|------|-------------|-------------|
| `POST` | `/runs/wait` | Create run, wait for completion | `ExecuteRun()` sync |
| `POST` | `/runs/stream` | Create run, stream output | `ExecuteRun()` + SSE |
| `POST` | `/runs` | Create background run | `ExecuteRun()` async |
| `GET` | `/runs/{run_id}` | Get run status | `GetRun()` |
| `POST` | `/runs/{run_id}/cancel` | Cancel run | `CancelRun()` |
| `DELETE` | `/runs/{run_id}` | Delete run record | (cleanup) |
| `GET` | `/runs/{run_id}/wait` | Wait for existing run | `GetRun()` + poll |
| `GET` | `/runs/{run_id}/stream` | Join run stream | (reconnect SSE) |

### Threads

| Method | Path | Description | APS Mapping |
|--------|------|-------------|-------------|
| `POST` | `/threads` | Create thread | `CreateSession()` |
| `POST` | `/threads/search` | Search threads | `ListSessions()` |
| `GET` | `/threads/{thread_id}` | Get thread | `GetSession()` |
| `GET` | `/threads/{thread_id}/history` | Get state history | (session history) |
| `POST` | `/threads/{thread_id}/runs` | Create run in thread | `ExecuteRun()` |
| `GET` | `/threads/{thread_id}/runs` | List thread runs | (run history) |
| `DELETE` | `/threads/{thread_id}` | Delete thread | (cleanup) |
| `PATCH` | `/threads/{thread_id}` | Update thread metadata | (session update) |

### Agents

| Method | Path | Description | APS Mapping |
|--------|------|-------------|-------------|
| `POST` | `/agents/search` | List agents | `ListAgents()` |
| `GET` | `/agents/{agent_id}` | Get agent info | `GetAgent()` |
| `GET` | `/agents/{agent_id}/schemas` | Get agent schemas | `GetAgentSchemas()` |

### Store

| Method | Path | Description | APS Mapping |
|--------|------|-------------|-------------|
| `PUT` | `/store/items` | Create/update item | `StorePut()` |
| `GET` | `/store/items` | Get item | `StoreGet()` |
| `DELETE` | `/store/items` | Delete item | `StoreDelete()` |
| `POST` | `/store/items/search` | Search items | `StoreSearch()` |
| `POST` | `/store/namespaces` | List namespaces | (list profiles) |

## Directory Structure

```
internal/
  core/
    protocol/
      interface.go      # APSCore interface definition
      types.go          # RunInput, RunState, etc.
      impl.go           # Default APSCore implementation wrapping existing core
  adapters/
    registry.go         # Adapter registration
    agentprotocol/
      adapter.go        # Agent Protocol adapter
      runs.go           # /runs handlers
      threads.go        # /threads handlers
      agents.go         # /agents handlers
      store.go          # /store handlers
      sse.go            # SSE streaming utilities
    openai/             # (future) OpenAI Assistants adapter
      adapter.go
```

## CLI Integration

New command to serve the API:

```bash
# Serve Agent Protocol API on default port
aps serve

# Serve on specific port
aps serve --port 8080

# Serve specific adapter only
aps serve --adapter agent-protocol

# Serve alongside webhook server
aps serve --with-webhooks
```

## Dependencies & Assumptions

- **Dependency**: Existing `internal/core` execution engine.
- **Dependency**: `internal/core/session` for session management.
- **Assumption**: SSE streaming is sufficient (no WebSocket requirement in AP spec).
- **Assumption**: Single adapter (Agent Protocol) is built first; others added on demand.

## Success Criteria

### Measurable Outcomes

- **SC-001**: `POST /runs/wait` returns action output within 100ms overhead of direct execution.
- **SC-002**: `POST /runs/stream` delivers first SSE event within 50ms of output availability.
- **SC-003**: `POST /runs/{id}/cancel` terminates process within 1 second.
- **SC-004**: All Agent Protocol endpoints pass compliance tests.
- **SC-005**: Adding a new adapter requires only new files in `internal/adapters/`, no changes to `internal/core/`.

### Edge Cases

- **EC-001**: Run cancelled before process starts (status: cancelled, no process to kill).
- **EC-002**: Client disconnects mid-stream (cleanup resources, mark run appropriately).
- **EC-003**: Concurrent runs on same thread (reject per AP spec, HTTP 409).
- **EC-004**: Invalid agent_id or action (HTTP 404 with clear error message).
- **EC-005**: Store namespace doesn't exist (auto-create vs error - TBD).
