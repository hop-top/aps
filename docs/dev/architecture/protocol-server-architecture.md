# Protocol Server Architecture - Unified Interface Pattern

## Overview

All agent communication protocols in APS implement a **common `ProtocolServer` interface**, with specialized adapters handling transport-specific concerns. This unified pattern allows protocols to be registered, started, stopped, and monitored through a consistent API.

**HTTP is treated as an optional transport adapter layer**, not as a requirement for all protocols.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│ Client Applications                                              │
│ (External, Editor, Other Agents)                                │
└─────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│ Transport Layer (HTTP, WebSocket, stdio, IPC)                  │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌──────────────┐  ┌────────────────────┐ │
│ │ HTTPBridge      │  │ HTTP Adapter │  │ Native Transports  │ │
│ │ (stdio→HTTP)    │  │ (registration)  │ (stdio, WebSocket) │ │
│ └─────────────────┘  └──────────────┘  └────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│ ProtocolServer Interface (Common)                              │
│ • Name() string                                                  │
│ • Start(ctx context.Context, config interface{}) error         │
│ • Stop() error                                                  │
│ • Status() string                                              │
└─────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│ Protocol Implementations                                         │
│ ┌────────────────────┐  ┌───────────┐  ┌──────────────────┐   │
│ │ Agent Protocol     │  │ A2A       │  │ ACP              │   │
│ │ (LangChain)        │  │ (Agent→A) │  │ (Editor→Agent)   │   │
│ └────────────────────┘  └───────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│ APSCore (Shared Business Logic)                                │
│ • Sessions, Agents, Profiles, Storage, Execution              │
└─────────────────────────────────────────────────────────────────┘
```

## Interface Hierarchy

### Base Interface: `ProtocolServer`

```go
type ProtocolServer interface {
    // Name returns the protocol name (e.g., "agent-protocol", "a2a", "acp")
    Name() string

    // Start initializes and starts the protocol server
    // config is protocol-specific configuration (can be nil)
    Start(ctx context.Context, config interface{}) error

    // Stop gracefully stops the protocol server
    Stop() error

    // Status returns the current status ("running", "stopped", "error")
    Status() string
}
```

**Implemented by:**
- ✅ Agent Protocol Adapter
- ✅ A2A Server
- ✅ ACP Server
- ✅ All HTTP Bridges

### Extended Interfaces

#### 1. `HTTPProtocolAdapter` - HTTP Routing Registration

```go
type HTTPProtocolAdapter interface {
    ProtocolServer

    // RegisterRoutes registers HTTP routes on the provided mux
    // Called during shared HTTP server initialization
    RegisterRoutes(mux *http.ServeMux, core APSCore) error
}
```

**Used by:**
- Agent Protocol Adapter - registers `/v1/runs`, `/v1/threads`, etc. routes

**Pattern:** Protocols implementing this interface share a single HTTP server on port 8080.

#### 2. `StandaloneProtocolServer` - Autonomous HTTP/Transport Server

```go
type StandaloneProtocolServer interface {
    ProtocolServer

    // GetAddress returns the address where the server is listening
    // Returns empty string for non-network servers (stdio)
    GetAddress() string
}
```

**Used by:**
- A2A Server - manages own HTTP server (port 8081)
- ACP Server - manages own stdio/WebSocket transports

**Pattern:** Protocols implementing this interface manage their own server lifecycle and can run on separate ports/transports.

#### 3. `HTTPBridge` - Transport Adapter

```go
type HTTPBridge interface {
    ProtocolServer

    // GetHTTPHandler returns an HTTP handler that exposes the protocol
    // Translates HTTP requests/responses to protocol's native format
    GetHTTPHandler() http.Handler
}
```

**Used by:**
- `DefaultHTTPBridge` - generic HTTP bridge for any protocol
- `JSONRPCHTTPBridge` - specialized JSON-RPC over HTTP bridge

**Pattern:** Allows stdio-based protocols (like ACP) to be exposed via HTTP for remote access.

## Implementation Details

### All Protocols Implement `ProtocolServer`

**Agent Protocol** (line 19 in adapter.go):
```go
var _ protocol.ProtocolServer = (*AgentProtocolAdapter)(nil)
var _ protocol.HTTPProtocolAdapter = (*AgentProtocolAdapter)(nil)
```

**A2A** (line 20-22 in server.go):
```go
var _ protocol.ProtocolServer = (*Server)(nil)
var _ protocol.StandaloneProtocolServer = (*Server)(nil)
```

**ACP** (line 41-43 in server.go):
```go
var _ protocol.ProtocolServer = (*Server)(nil)
var _ protocol.StandaloneProtocolServer = (*Server)(nil)
```

### HTTP Bridge Implementations

#### DefaultHTTPBridge
Simple generic bridge for any `ProtocolServer`:
```go
bridge := NewDefaultHTTPBridge(anyProtocol)
handler := bridge.GetHTTPHandler()
```

#### JSONRPCHTTPBridge
Specialized for JSON-RPC 2.0 protocols:
```go
bridge := NewJSONRPCHTTPBridge(acpServer)
handler := bridge.GetHTTPHandler()
// Now ACP (stdio) is accessible via HTTP endpoints
```

#### ProtocolServerAdapter
Adapts any `ProtocolServer` to be an `HTTPProtocolAdapter`:
```go
adapter := NewProtocolServerAdapter(standaloneServer)
adapter.RegisterRoutes(mux, core)  // Can register routes if needed
```

## Server Lifecycle Patterns

### Pattern 1: HTTP Adapter (Agent Protocol)

```
1. AgentProtocolAdapter created
2. During serve startup:
   a. RegisterRoutes() called
   b. Routes mounted in shared HTTP mux
   c. Single HTTP server started (port 8080)
3. Status() returns "running" while main HTTP server runs
4. Stop() called when main server shuts down
```

**Characteristics:**
- Shares HTTP port with other adapters
- Lightweight lifecycle (no independent server)
- Stateless request-response model

### Pattern 2: Standalone Server (A2A, ACP)

```
1. A2AServer/ACPServer created
2. Start(ctx, config) called:
   a. Creates its own HTTP/transport server
   b. Starts listening on configured port/transport
   c. Runs independently in background goroutine
   d. Monitors context cancellation
3. Status() returns "running" while server runs
4. Stop() gracefully shuts down server
```

**Characteristics:**
- Independent server per protocol
- Can use different transports (HTTP, stdio, WebSocket)
- Persistent bidirectional communication
- Can run on different ports/sockets

### Pattern 3: HTTP Bridge (Exposing stdio via HTTP)

```
1. ACPServer created (native stdio transport)
2. JSONRPCHTTPBridge created wrapping ACPServer
3. Bridge.GetHTTPHandler() returns HTTP handler
4. Handler mounted in main HTTP mux (port 8080)
5. HTTP clients → HTTP handler → JSONRPCHTTPBridge → ACPServer (stdio)
```

**Characteristics:**
- Translates between HTTP and native protocol
- Optional layer for remote access
- Can be applied to any protocol
- Reuses main HTTP server infrastructure

## Protocol Registration

### Registry Separation

```go
type ProtocolRegistry struct {
    httpAdapters      map[string]protocol.HTTPProtocolAdapter
    standaloneServers map[string]protocol.StandaloneProtocolServer
}

// Register HTTP adapters (integrated into main mux)
func (r *ProtocolRegistry) RegisterHTTPAdapter(name string, adapter HTTPProtocolAdapter)

// Register standalone servers (independent lifecycle)
func (r *ProtocolRegistry) RegisterStandaloneServer(name string, server StandaloneProtocolServer)
```

### Startup Flow (`internal/cli/serve.go`)

```go
func runServe() {
    // 1. Create HTTP mux
    mux := http.NewServeMux()

    // 2. Register all HTTP adapters' routes
    registry.RegisterHTTPRoutes(mux, core)
    // → Agent Protocol routes now in mux

    // 3. Start main HTTP server
    http.ListenAndServe(":8080", mux)

    // 4. (Optional) Start standalone servers
    registry.StartStandaloneServer("a2a", config)
    registry.StartStandaloneServer("acp", config)
}
```

## Benefits of Unified Interface

### ✅ Consistent API
All protocols use same `Name()`, `Start()`, `Stop()`, `Status()` methods.

### ✅ Pluggable Architecture
New protocols can be added without modifying core registration logic.

### ✅ Transport Agnostic
HTTP, stdio, WebSocket, IPC - all treated as transport layers.

### ✅ Shared Business Logic
All protocols use same `APSCore` for sessions, agents, profiles.

### ✅ Flexible Deployment
- Integrate via HTTP routes (lightweight)
- Run standalone (independent)
- Expose via HTTP bridge (remote access)

## Practical Examples

### Example 1: Using Any Protocol Uniformly

```go
var protocols []protocol.ProtocolServer
protocols = append(protocols, agentProtocolAdapter)
protocols = append(protocols, a2aServer)
protocols = append(protocols, acpServer)

// Start all protocols with same code
for _, p := range protocols {
    if err := p.Start(ctx, config); err != nil {
        log.Fatal(err)
    }
}

// Monitor all protocols
for _, p := range protocols {
    fmt.Printf("%s: %s\n", p.Name(), p.Status())
}
```

### Example 2: Expose Standalone Server via HTTP

```go
// ACP runs on stdio natively
acpServer := acp.NewServer("my-profile", core)

// Create HTTP bridge for remote access
bridge := protocol.NewJSONRPCHTTPBridge(acpServer)

// Mount bridge handler in main HTTP server
mux.Handle("/acp/", bridge.GetHTTPHandler())

// Now both work:
// 1. stdio: ACP protocol via stdio (local editors)
// 2. HTTP: Bridge endpoint (remote clients)
```

### Example 3: Register Protocol at Compile Time

```go
// In internal/adapters/init.go
func init() {
    // HTTP adapter - integrates into main mux
    registry.RegisterHTTPAdapter("agent-protocol", NewAgentProtocolAdapter())

    // Standalone servers - run independently
    registry.RegisterStandaloneServer("a2a", a2a.NewServer)
    registry.RegisterStandaloneServer("acp", acp.NewServer)
}
```

## Key Design Decisions

1. **All protocols implement `ProtocolServer`** - Common foundation
2. **HTTP is optional** - Not all protocols need HTTP, it's a transport
3. **Two registration patterns** - Adapters (shared HTTP) vs Standalone (independent)
4. **HTTPBridge for exposure** - Expose any protocol via HTTP if needed
5. **Separation of concerns** - Transport logic separate from protocol logic

## Files

- `internal/core/protocol/server.go` - Interface definitions
- `internal/core/protocol/http_bridge.go` - Bridge implementations
- `internal/adapters/agentprotocol/adapter.go` - HTTP adapter example
- `internal/a2a/server.go` - Standalone server example
- `internal/acp/server.go` - Standalone server with custom transport

---

**Refactored:** 2026-02-01
**Pattern:** Unified ProtocolServer with optional HTTP transport adapters
