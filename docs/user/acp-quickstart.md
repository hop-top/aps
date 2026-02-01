# ACP (Agent Client Protocol) Quick Start

## What is ACP?

ACP is the protocol that allows your editor (VS Code, Cursor, Zed, Neovim) to communicate with an APS agent. It's designed specifically for real-time, interactive agent sessions with permission controls.

**Key features:**
- ✅ Editor-native - Built for IDE integrations
- ✅ Real-time streaming - Get updates as the agent works
- ✅ Permission control - Approve sensitive operations
- ✅ Session-based - Maintain conversation context
- ✅ Multi-modal - Text, images, audio support

## Installation

### Prerequisites
- Go 1.21+
- APS installed (`aps` command available)

### Enable ACP for a Profile

Add to your profile configuration:

```yaml
profiles:
  my-agent:
    acp:
      enabled: true
      protocol_version: 1
      default_mode: "default"  # or "auto_approve" or "read_only"
```

## Starting an ACP Session

### 1. Start the ACP Server

```bash
aps acp start --profile my-agent
```

Output:
```
ACP server started on stdio
Ready for editor connections
Press Ctrl+C to stop
```

### 2. Connect from Your Editor

#### VS Code / Cursor
Install the APS extension and select "ACP" protocol.

#### Zed
Configuration in `~/.config/zed/settings.json`:
```json
{
  "agents": {
    "my-aps-agent": {
      "protocol": "acp",
      "command": "aps acp start --profile my-agent"
    }
  }
}
```

#### Neovim
Plugin configuration example:
```lua
require('aps').setup({
  protocol = 'acp',
  profile = 'my-agent'
})
```

## Using ACP

### Basic Workflow

1. **Initialize Connection**
   ```json
   // From editor
   {
     "jsonrpc": "2.0",
     "id": 1,
     "method": "initialize",
     "params": {
       "protocolVersion": 1
     }
   }
   ```

2. **Create a Session**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 2,
     "method": "session/new",
     "params": {
       "profileId": "my-agent",
       "mode": "default"
     }
   }
   ```
   Response:
   ```json
   {
     "jsonrpc": "2.0",
     "id": 2,
     "result": {
       "sessionId": "sess_abc123",
       "agentCapabilities": {
         "filesystem": {"read": true, "write": true},
         "terminal": {"create": true}
       }
     }
   }
   ```

3. **Send a Prompt**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 3,
     "method": "session/prompt",
     "params": {
       "sessionId": "sess_abc123",
       "userMessage": "Add a test for the login function"
     }
   }
   ```

### Operating Modes

#### Default Mode (Recommended)
Agent requests permission for sensitive operations:

```json
// Agent requests permission
{
  "method": "session/request_permission",
  "params": {
    "sessionId": "sess_abc123",
    "operation": "fs/write_text_file",
    "resource": "/path/to/file.ts"
  }
}

// Editor shows dialog and responds
{
  "result": {
    "approved": true,
    "timestamp": "2026-02-01T10:30:00Z"
  }
}
```

#### Auto-Approve Mode
Agent operations execute immediately:

```json
{
  "method": "session/set_mode",
  "params": {
    "sessionId": "sess_abc123",
    "mode": "auto_approve"
  }
}
```

#### Read-Only Mode
Agent can only read files and run non-destructive commands:

```json
{
  "method": "session/set_mode",
  "params": {
    "sessionId": "sess_abc123",
    "mode": "read_only"
  }
}
```

## Common Operations

### Reading a File

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "fs/read_text_file",
  "params": {
    "sessionId": "sess_abc123",
    "path": "src/auth.ts"
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": "export async function login(...) { ... }",
    "mimeType": "text/typescript"
  }
}
```

### Writing a File

Requires permission approval in default mode:

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "fs/write_text_file",
  "params": {
    "sessionId": "sess_abc123",
    "path": "src/auth.test.ts",
    "content": "import { test } from 'vitest';\n..."
  }
}
```

### Running a Command

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "terminal/create",
  "params": {
    "sessionId": "sess_abc123",
    "command": "npm",
    "arguments": ["test"],
    "workingDirectory": "/project"
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "terminalId": "term_xyz789"
  }
}
```

Get output:
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "terminal/output",
  "params": {
    "terminalId": "term_xyz789"
  }
}
```

## Troubleshooting

### Connection Refused

```
Error: ECONNREFUSED - Editor can't connect to ACP server
```

**Fix:** Ensure server is running:
```bash
aps acp start --profile my-agent
```

### Permission Denied

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "permission denied"
  }
}
```

**Fix:** In default mode, approve the permission request. Or switch to auto_approve mode if you trust the agent.

### File Not Found

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32002,
    "message": "resource not found"
  }
}
```

**Fix:** Check that the file path is correct and accessible.

### Session Timeout

```
Session expired after 30 minutes of inactivity
```

**Fix:** Create a new session:
```json
{
  "method": "session/new",
  "params": {"profileId": "my-agent"}
}
```

## Performance Tips

### ✅ Do This
- Use `read_only` mode for code reviews
- Use `auto_approve` for trusted environments
- Batch file operations when possible
- Close sessions when done

### ❌ Don't Do This
- Don't keep sessions open indefinitely
- Don't request permissions for every operation (use modes)
- Don't write very large files in one operation
- Don't run unlimited concurrent sessions

## Configuration Examples

### Minimal Config
```yaml
profiles:
  basic:
    acp:
      enabled: true
```

### Secure Config
```yaml
profiles:
  secure:
    acp:
      enabled: true
      protocol_version: 1
      default_mode: "default"
      allow_terminal_creation: false  # No terminal access
      allow_file_write: false         # Read-only
```

### Development Config
```yaml
profiles:
  dev:
    acp:
      enabled: true
      protocol_version: 1
      default_mode: "auto_approve"    # Fast iteration
      mcp_servers:
        - name: "code-tools"
          type: "stdio"
          command: "npm run tools"
```

## Next Steps

- 📖 Read the [full specification](https://agentclientprotocol.com)
- 🛠️ Install your editor extension
- 💻 Try a basic workflow
- 🔧 Configure your profile
- 📝 Start using ACP sessions

---

**Last Updated:** 2026-02-01
**Protocol Version:** 1.0
**Status:** ✅ Stable
