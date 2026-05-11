# ACP (Agent Client Protocol) Implementation Guide

## Overview

ACP is a JSON-RPC 2.0 based protocol for editor-to-agent communication. In APS, the user-facing ACP server is a standalone protocol server over stdio JSON-RPC.

**Status:** stdio server ready. HTTP and WebSocket transport helpers exist as component/prototype code, but `aps acp server` does not expose them as supported listeners.

## Architecture

### Protocol Server Type
- **Implements:** `protocol.ProtocolServer` + `protocol.StandaloneProtocolServer`
- **Location:** `internal/acp/`
- **Transport:** stdio JSON-RPC through `aps acp server <profile>`
- **Model:** Bidirectional streaming with sessions

Remote ACP transport maturity:

- WebSocket transport helpers exist in `internal/acp/transport_ws.go`, but the CLI does not start them.
- HTTP bridge helpers exist in `internal/core/protocol/http_bridge.go`, but they return bridge/status-shaped responses and are not mounted by `aps serve`.
- Non-stdio ACP transport values are rejected by `aps acp toggle` and by `Server.Start` when passed directly.

### Key Components

```
internal/acp/
├── server.go              # Main ACP server
├── handler.go             # JSON-RPC 2.0 request handler
├── session.go             # ACP session management
├── session_manager.go     # Session lifecycle
├── terminal.go            # Terminal command execution
├── terminal_manager.go    # Terminal lifecycle
├── filesystem.go          # File system operations
├── permissions.go         # Permission request system
├── capabilities.go        # Capability negotiation
├── content_blocks.go      # Content block handling
├── mcp_bridge.go          # MCP server integration
├── types.go               # Protocol types
└── errors.go              # Error codes
```

## Protocol Methods Implemented

### Initialization
- `initialize` - Establish connection, negotiate version
- `authenticate` - Client authentication

### Session Management
- `session/new` - Create new session with mode
- `session/load` - Resume previous session
- `session/prompt` - Send user message
- `session/cancel` - Cancel operation
- `session/update` - Receive streaming updates (notification)
- `session/set_mode` - Switch operating mode

### File System (fs/*)
- `fs/read_text_file` - Read file contents
- `fs/write_text_file` - Write file contents

### Terminal (terminal/*)
- `terminal/create` - Spawn command with args/env
- `terminal/output` - Get current output
- `terminal/wait_for_exit` - Block for completion
- `terminal/kill` - Terminate command
- `terminal/release` - Free resources

## Session Modes

ACP supports three operating modes that control permission behavior:

### 1. `default` Mode
- **Behavior:** Request user permission for sensitive operations
- **Permission Flow:** Operation → Request → User approval → Execute
- **Use Case:** Standard interactive sessions with untrusted agents

### 2. `auto_approve` Mode
- **Behavior:** Automatically approve all operations
- **Permission Flow:** Operation → Execute (no approval)
- **Use Case:** Trusted environments, automated workflows

### 3. `read_only` Mode
- **Behavior:** Deny all write operations, allow reads
- **Permission Flow:** Write operations → Denied, Read operations → Execute
- **Use Case:** Safe inspection mode, read-only analysis

## Capability Negotiation

### Agent Capabilities (Advertised)
```json
{
  "filesystem": {
    "read": true,
    "write": true
  },
  "terminal": {
    "create": true,
    "interactive": true
  },
  "session_modes": ["default", "auto_approve", "read_only"],
  "content_types": ["text", "image", "audio"],
  "mcp_servers": ["tool1", "tool2"]
}
```

### Client Capabilities (Requested)
```json
{
  "filesystem": {
    "readTextFile": true,
    "writeTextFile": false
  },
  "terminal": true
}
```

## Security Model

### File System Access
- Validates paths are within working directory
- Checks session permission rules
- Integrates with isolation system
- Denies access to sensitive files (.env, credentials)

### Terminal Operations
- Enforces session mode restrictions
- Runs within isolation context
- Manages process lifetime
- Cleans up resources on session end

### Permission System
- Three-tier evaluation: SessionMode → PermissionRule → UserApproval
- Cacheable permissions per session
- Audit trail of decisions

## Integration Points

### APSCore Interface
```go
type ACPCore interface {
    protocol.APSCore
    
    // File system (isolated)
    ReadTextFile(sessionID, path string) (string, error)
    WriteTextFile(sessionID, path, content string) error
    
    // Terminal operations
    CreateTerminal(sessionID, cmd string, args []string) (terminalID string, err error)
    WaitForTerminalExit(terminalID string) (exitCode int, err error)
    KillTerminal(terminalID string) error
    
    // Session modes
    SetSessionMode(sessionID string, mode SessionMode) error
    RequestPermission(sessionID, operation, resource string) (bool, error)
}
```

### Isolation Integration
- Validates file paths within isolation context
- Executes terminals with isolation environment
- Respects isolation tier capabilities

### MCP Bridge
- Connects MCP servers as callable tools
- Exposes MCP resources to client
- Manages tool call execution

## Testing

### Unit Tests (`tests/unit/acp/`)
- JSON-RPC message parsing
- Session lifecycle
- Permission evaluation
- Terminal management
- Content block handling
- Capability negotiation

### Integration Tests (`tests/integration/acp/`)
- Full workflow with permission flow
- Mode enforcement
- File system isolation
- Terminal lifecycle
- MCP integration

### E2E Tests (`tests/e2e/acp_test.go`)
- Complete editor workflow
- Streaming updates
- Permission approvals
- Cleanup and shutdown

## CLI Commands

```bash
# Start ACP server for a profile
aps acp server my-agent

# Enable ACP configuration for a profile
aps acp toggle --profile my-agent --enabled=on --transport stdio

# HTTP and WebSocket transports are not wired to aps acp server yet
aps acp toggle --profile my-agent --transport ws
# error: ACP transport "ws" is not wired to aps acp server yet; use --transport=stdio
```

## Configuration

### Profile YAML
```yaml
profiles:
  my-agent:
    acp:
      enabled: true
      protocol_version: 1
      # Permission defaults
      default_mode: "default"
      allow_terminal_creation: true
      allow_file_write: true
      # MCP servers
      mcp_servers:
        - name: "my-tools"
          type: "stdio"
          command: "./tools/my-tools"
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| Message roundtrip | <5ms |
| Session creation | <10ms |
| Terminal spawn | <50ms |
| File read | <5ms (varies with file size) |
| Concurrent sessions | 1000+ |
| Memory per session | ~100KB |

## Limitations & Future Work

### Current Limitations
- User-facing ACP transport is stdio only
- HTTP bridge is component-only and not mounted by `aps serve`
- WebSocket helper is not wired to `aps acp server`
- Session persistence not implemented (Phase 7)
- No OAuth2 support (Phase 8)
- Content blocks limited to text/image/audio

### Phase 7: HTTP Exposure
```go
// Prototype-only: construct an HTTP bridge component.
// This is not mounted by aps serve and does not make ACP a supported HTTP service.
bridge := protocol.NewJSONRPCHTTPBridge(acpServer)
mux.Handle("/acp/", bridge.GetHTTPHandler())
```

### Phase 8: Advanced Features
- Session persistence across restarts
- OAuth2/JWT authentication
- Multi-language SDK support
- Real-time cursor/view synchronization

## References

- **Official Spec:** https://agentclientprotocol.com
- **GitHub:** https://github.com/agentclientprotocol/agent-client-protocol
- **Architecture:** See `docs/dev/architecture/protocol-server-architecture.md`
- **Unified Interface:** See `docs/dev/protocol-interface-unification.md`

---

**Implementation Status:** stdio ACP server ready; remote transports component/planned
**Last Updated:** 2026-05-11
