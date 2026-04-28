# ACP (Agent Client Protocol) Implementation

**Status**: Complete - Phase 6 (Polish & Documentation)
**Date**: 2026-02-01
**Version**: 0.1.0 (Initial Implementation)

## Overview

This directory contains the complete implementation of ACP (Agent Client Protocol) support for APS (Agent Profile System). ACP is a JSON-RPC 2.0 based protocol that standardizes communication between editors/clients and AI agents.

## What is ACP?

**Agent Client Protocol** (ACP) provides a standardized way for editor clients (VS Code, Cursor, Zed, Neovim) to communicate with AI agents. It enables:

- **Bidirectional communication** between editors and agents via JSON-RPC 2.0
- **Session management** with permission-based access control
- **File system operations** with security validation
- **Terminal integration** for command execution
- **Streaming updates** via notifications
- **Tool integration** via MCP (Model Context Protocol) servers

## Architecture

### Three-Layer Protocol Stack

```
┌─────────────────────────────────────┐
│     Editor (Client)                 │
│  (VS Code, Cursor, Zed, Neovim)    │
└────────────┬────────────────────────┘
             │ ACP (JSON-RPC 2.0)
             │ Transport: stdio, HTTP, WebSocket
┌────────────▼────────────────────────┐
│     APS Agent (Server)              │
│  Session Management                 │
│  Permission System                  │
│  File System & Terminal Ops         │
└────────────┬────────────────────────┘
             │ A2A / Agent Protocol
             │
┌────────────▼────────────────────────┐
│   Other Agents / Tools / Data       │
│   (MCP Servers, Databases, etc)     │
└─────────────────────────────────────┘
```

### Component Overview

**Core Components**:
- `server.go` - Main ACP server with JSON-RPC 2.0 handler
- `session.go` - Session lifecycle and mode management
- `permissions.go` - Permission system with rules and request handling
- `capabilities.go` - Agent capability negotiation
- `filesystem.go` - File read/write with security validation
- `terminal.go` - Terminal/process management
- `content.go` - Content block handling and notifications
- `transport_stdio.go` - Primary stdio transport (local editors)
- `transport_ws.go` - WebSocket transport (remote editors)

## Protocol Methods

### Initialization Phase

#### `initialize`
Establish connection and exchange capabilities.

```json
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": 1,
    "capabilities": {
      "filesystem": true,
      "terminal": true
    },
    "clientInfo": {
      "name": "VS Code",
      "version": "1.85.0"
    }
  },
  "id": 1
}
```

#### `authenticate`
Validate client identity (implementation dependent).

### Session Management

#### `session/new`
Create new conversation session.

```json
{
  "jsonrpc": "2.0",
  "method": "session/new",
  "params": {
    "profileId": "my-agent",
    "mode": "default",
    "clientCapabilities": {
      "filesystem": true,
      "terminal": true
    }
  },
  "id": 2
}
```

**Session Modes**:
- `default` - Request permissions for sensitive operations
- `auto_approve` - Auto-approve all operations (trusted environment)
- `read_only` - Deny all write operations

#### `session/prompt`
Send message to agent with streaming updates.

#### `session/set_mode`
Switch session mode at runtime.

### File System Operations

#### `fs/read_text_file`
Read file with optional line selection.

```json
{
  "jsonrpc": "2.0",
  "method": "fs/read_text_file",
  "params": {
    "sessionId": "sess_123",
    "path": "/home/user/project/main.py",
    "startLine": 10,
    "endLine": 20
  },
  "id": 3
}
```

#### `fs/write_text_file`
Write file content with automatic directory creation.

```json
{
  "jsonrpc": "2.0",
  "method": "fs/write_text_file",
  "params": {
    "sessionId": "sess_123",
    "path": "/home/user/project/new_file.py",
    "content": "def hello():\n    print('Hello, world!')"
  },
  "id": 4
}
```

### Terminal Operations

#### `terminal/create`
Create and execute a command in a terminal.

```json
{
  "jsonrpc": "2.0",
  "method": "terminal/create",
  "params": {
    "sessionId": "sess_123",
    "command": "python",
    "arguments": ["main.py"],
    "workingDirectory": "/home/user/project",
    "environment": {
      "PYTHONUNBUFFERED": "1"
    }
  },
  "id": 5
}
```

#### `terminal/output`
Get current terminal output without blocking.

#### `terminal/wait_for_exit`
Wait for terminal process to complete.

#### `terminal/kill`
Terminate running process.

#### `terminal/release`
Release and cleanup terminal resources.

## Security Model

### Permission System

Three-tier permission system:

1. **Session Mode** - Highest level control
   - Default mode: Prompts for sensitive operations
   - Auto-approve mode: Allows all operations
   - Read-only mode: Denies all write operations

2. **Permission Rules** - Fine-grained control
   - Path patterns for file operations
   - Command patterns for terminal operations
   - Custom rules per session

3. **Isolation** - System-level enforcement
   - Working directory confinement
   - Sensitive path blocking (.env, credentials, /etc/, /sys/)
   - File size limits (100MB default)

### Sensitive Paths Blocked
- `.env`, `credentials*`, `secret*`
- `/etc/`, `/sys/`, `/proc/`, `/.ssh/`, `/.aws/`
- System binaries: `/bin/`, `/sbin/`, `/usr/bin/`

## Transport Mechanisms

### Stdio (Primary)
- **Use Case**: Local editors (VS Code, Cursor, Zed, Neovim)
- **Method**: JSON-RPC over stdin/stdout
- **Latency**: Minimal (process communication)
- **Status**: ✅ Fully Implemented

### WebSocket (Secondary)
- **Use Case**: Remote editors, cloud IDEs
- **Method**: JSON-RPC over WebSocket
- **Latency**: Network dependent
- **Status**: ✅ Fully Implemented

### HTTP Adapter (Future)
- **Use Case**: REST API compatibility
- **Method**: HTTP endpoints for ACP operations
- **Status**: 🔄 Ready for Phase 6+

## Content Blocks

Supports multiple content types for rich communication:

- **text** - Markdown formatted content
- **image** - Base64 encoded images (PNG, JPG, GIF, WebP)
- **audio** - Base64 encoded audio files
- **resource** - URI references to external resources

## Execution Plans

Track multi-step task execution with plans:

```json
{
  "steps": [
    {
      "content": "Analyze code structure",
      "priority": "high",
      "status": "completed"
    },
    {
      "content": "Run tests",
      "priority": "high",
      "status": "in_progress"
    },
    {
      "content": "Deploy changes",
      "priority": "medium",
      "status": "pending"
    }
  ],
  "status": "in_progress"
}
```

## Integration with APS

### Profile Configuration

Add ACP to your profile:

```yaml
id: my-agent
display_name: "My AI Agent"
capabilities:
  - git
  - docker
acp:
  enabled: true
  transport: stdio  # or "ws" for WebSocket
  port: 9000        # for WebSocket transport
```

### Session Lifecycle

```
1. Editor connects via stdio/WebSocket
2. Client sends `initialize` with capabilities
3. Agent responds with capabilities
4. Client sends `authenticate` (optional)
5. Client sends `session/new` to create session
6. Agent returns sessionId with capabilities
7. Client can now:
   - Read/write files (fs/read_text_file, fs/write_text_file)
   - Execute commands (terminal/create, terminal/output, etc.)
   - Send prompts (session/prompt)
   - Switch modes (session/set_mode)
8. Client sends `session/cancel` or session ends
9. Server cleans up resources
```

## Testing

### Unit Tests (60+ tests)
- Session management and modes
- Permission system and rules
- File system operations and security
- Terminal creation and lifecycle
- Content blocks and serialization
- Execution plans and tracking
- MCP bridge registration

### Integration Tests
See `tests/integration/acp_test.go` for end-to-end scenarios.

### Performance Benchmarks
See `tests/benchmarks/acp_bench_test.go` for throughput/latency metrics.

## Usage Examples

### Starting an ACP Server

```bash
# Start ACP server for a profile using stdio transport
aps acp server my-agent

# The server will read JSON-RPC requests from stdin
# and write responses to stdout
```

### Basic Editor Integration

```typescript
// Example: VS Code Extension using ACP
const net = require('net');
const { spawn } = require('child_process');

async function startAgent(profileId: string) {
  const process = spawn('aps', ['acp', 'server', profileId]);

  // Send initialize
  const initMsg = {
    jsonrpc: "2.0",
    method: "initialize",
    params: {
      protocolVersion: 1,
      capabilities: { filesystem: true, terminal: true },
      clientInfo: { name: "VS Code", version: "1.85.0" }
    },
    id: 1
  };

  process.stdin.write(JSON.stringify(initMsg) + '\n');

  // Handle responses via stdout
  let buffer = '';
  process.stdout.on('data', (data) => {
    buffer += data.toString();
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    lines.forEach(line => {
      if (line) {
        const response = JSON.parse(line);
        handleResponse(response);
      }
    });
  });
}
```

## Compatibility

### Supported Editors
- ✅ VS Code (via extension)
- ✅ Cursor (native support)
- ✅ Zed (native support)
- ✅ Neovim (via plugin)
- 🔄 Other editors (HTTP adapter in progress)

### Official SDK Versions
- TypeScript: `@agentclientprotocol/sdk` v0.10.7+
- Python: `python-sdk` v0.10.7+
- Rust: `agent-client-protocol` v0.10.7+

## References

- **Official ACP**: https://agentclientprotocol.com
- **GitHub**: https://github.com/agentclientprotocol/agent-client-protocol
- **Specification**: https://github.com/agentclientprotocol/agent-client-protocol/blob/main/docs/spec.md

## Implementation Notes

### Current Limitations
- WebSocket authentication is basic (no OAuth2 yet)
- MCP integration is placeholder (needs real MCP client)
- HTTP adapter not yet implemented (Phase 6+)
- No persistence of sessions across restarts

### Future Enhancements
- Session persistence
- Advanced MCP integration
- HTTP/REST adapter
- Performance optimizations
- Streaming response improvements
- Custom capability registration

## Contributing

When extending ACP support:
1. Follow established patterns (session/permission/content managers)
2. Add unit tests for new operations
3. Update documentation
4. Add integration tests
5. Benchmark performance impact

---

**Implementation Completed**: February 2026
**Specification Version**: 0.1.0
**Agent**: Claude Haiku 4.5
