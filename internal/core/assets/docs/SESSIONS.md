# Session Management

Complete guide to tracking and managing long-running APS sessions.

## What is a Session?

A session represents a long-running command or action executed under a profile. Sessions provide:

- **Tracking**: Monitor running processes and their status
- **Metadata**: Store execution context, environment, and timing
- **Control**: Attach to, detach from, and manage sessions
- **Security**: Admin SSH keys for session management
- **Persistence**: Session registry survives restarts

## Session Lifecycle

```
┌──────────┐    Register     ┌──────────┐    Update     ┌──────────┐
│  Start   │ ──────────────► │  Active  │ ────────────► │ Complete │
│ Command  │                │ Session  │              │ Session  │
└──────────┘                └──────────┘              └──────────┘
       │                          │                        │
       │                          ▼                        │
       │                   ┌──────────┐                    │
       └────────────────── │  Heart   │ ──────────────────┘
                          │  Beats   │
                          └──────────┘
```

## Session Types

### Interactive Sessions

Started by running a profile directly:

```bash
# Start an interactive shell
aps myagent

# Equivalent to:
aps run myagent -- /bin/zsh
```

### Command Sessions

Started with explicit commands:

```bash
# Run a command
aps myagent -- python3 -m http.server

# Run with arguments
aps myagent -- npm start
```

### Action Sessions

Started by running actions:

```bash
# Run an action
aps action run myagent deploy.sh

# Run with payload
echo '{"env": "prod"}' | aps action run myagent deploy.sh --payload-stdin
```

## Session Registry

The session registry is stored at `~/.aps/sessions/registry.json`:

```json
{
  "session-123": {
    "id": "session-123",
    "profile_id": "myagent",
    "profile_dir": "/home/user/.agents/profiles/myagent",
    "command": "/bin/zsh",
    "pid": 12345,
    "status": "active",
    "tier": "basic",
    "tmux_socket": "/tmp/tmux-1000/default",
    "created_at": "2026-01-21T10:30:00Z",
    "last_seen_at": "2026-01-21T10:35:00Z",
    "environment": {
      "APS_PROFILE_ID": "myagent",
      "APS_PROFILE_DIR": "/home/user/.agents/profiles/myagent"
    }
  }
}
```

## Session Status

| Status | Description |
|--------|-------------|
| `active` | Session is running and responding |
| `inactive` | Session is registered but not responding |
| `errored` | Session encountered an error |

## Session Tiers

| Tier | Description |
|------|-------------|
| `basic` | Standard session |
| `standard` | Enhanced session with monitoring |
| `premium` | Full-featured session with advanced features |

## Commands

### List Sessions

```bash
# List all active sessions
aps session list

# List sessions for a specific profile
aps session list --profile myagent

# Filter by status
aps session list --status active
aps session list --status inactive
aps session list --status errored

# Filter by tier
aps session list --tier basic
aps session list --tier standard
aps session list --tier premium

# Combine filters
aps session list --profile myagent --status active --tier premium
```

**Output:**
```
ID          PROFILE   PID   STATUS   TIER      CREATED              LAST SEEN
session-123 myagent   12345 active   basic     2026-01-21 10:30:00  10:35:00
session-124 prod      12346 active   standard  2026-01-21 10:32:00  10:34:00
session-125 dev       0     inactive basic     2026-01-21 09:00:00  09:15:00
```

### Attach to Session

```bash
# Attach to a running session
aps session attach session-123
```

**Note**: Full attach functionality is coming soon. Currently displays session information.

### Detach from Session

```bash
# Detach from a session
aps session detach session-123
```

**Note**: Full detach functionality is coming soon. Currently displays session information.

## Session Files

### Session Directory Structure

```
~/.aps/
  sessions/
    registry.json           # Session registry
  keys/
    <session-id>/
      admin_key             # Private SSH key (chmod 0600)
      admin_key.pub         # Public SSH key
  tmux.conf                 # Tmux configuration
```

### Session Registry

The session registry (`~/.aps/sessions/registry.json`) stores all active and recent sessions.

- **Format**: JSON
- **Permissions**: 0600 (owner read/write only)
- **Persistence**: Survives restarts
- **Cleanup**: Automatically removes inactive sessions after timeout

### Admin SSH Keys

Each session has its own SSH key pair for secure management:

```bash
~/.aps/keys/
  session-123/
    admin_key         # Private key (chmod 0600)
    admin_key.pub     # Public key
```

**Key Types:**
- RSA (4096-bit)
- Ed25519 (default)

**Purpose:**
- Secure session attachment
- Remote session management
- Authentication for privileged operations

### Tmux Configuration

The tmux configuration file (`~/.aps/tmux.conf`) controls session behavior:

```bash
# Status bar position (top | bottom)
status-bar top

# Disable idle timeout (on | off)
no-idle-timeout on

# Session directory
session-dir ~/.aps/sessions

# Keys directory
keys-dir ~/.aps/keys
```

## Session Timeout

Inactive sessions are automatically cleaned up after a timeout:

- **Default Timeout**: 30 minutes
- **Heartbeat**: Sessions update `last_seen_at` timestamp
- **Cleanup**: Registry removes sessions older than timeout

**Configuration** (coming soon):
```yaml
# In ~/.aps/tmux.conf
session-timeout-minutes: 30
```

## Integration with TUI

The APS TUI can display and manage sessions:

```bash
# Launch TUI
aps

# Navigate to sessions section
# View session status, PID, and metadata
# Select session for attach/detach
```

## Examples

### Monitor Running Services

```bash
# Start a web server session
aps web-server -- python3 -m http.server 8000

# List active sessions
aps session list

# Expected output:
# ID          PROFILE      PID   STATUS   TIER   CREATED              LAST SEEN
# session-001 web-server   12345 active   basic  2026-01-21 10:30:00  10:35:00
```

### Development Workflow

```bash
# Start development server
aps dev-agent -- npm run dev

# Start database
aps db-agent -- postgres -D /var/lib/postgresql/data

# List sessions
aps session list

# Attach to dev server
aps session attach session-001
```

### Batch Processing

```bash
# Start data processing action
aps action run data-agent process.py --payload-file data.json

# Monitor session
aps session list --profile data-agent --status active
```

### Long-Running Jobs

```bash
# Start background job
aps job-agent -- python3 job_worker.py

# Session automatically tracks PID
# Heartbeats keep session alive
# Session persists in registry
```

## Troubleshooting

### Session shows "inactive" status

**Cause**: Process stopped responding or died

**Solutions**:
1. Check if process is still running: `ps aux | grep <pid>`
2. Check system logs for errors
3. Restart the session
4. Verify environment variables and paths

### Session not appearing in list

**Cause**: Session not registered or registry issue

**Solutions**:
1. Check registry file: `cat ~/.aps/sessions/registry.json`
2. Verify file permissions: `ls -la ~/.aps/sessions/`
3. Ensure registry is valid JSON
4. Check for file system errors

### Cannot attach to session

**Cause**: Attach functionality not yet implemented or permission issue

**Solutions**:
1. Verify session is active: `aps session list --status active`
2. Check admin key permissions: `ls -la ~/.aps/keys/<session-id>/`
3. Ensure tmux socket exists (if applicable)
4. Full attach functionality coming soon

### Session registry corrupted

**Cause**: Unexpected shutdown or file system error

**Solutions**:
1. Backup current registry: `cp ~/.aps/sessions/registry.json ~/.aps/sessions/registry.json.bak`
2. Manually edit registry if needed
3. Delete corrupted registry (sessions will be lost but system will work)
4. Restart APS to recreate registry

## Best Practices

1. **Regular Cleanup**: Monitor inactive sessions and clean up periodically
2. **Descriptive Sessions**: Use meaningful profile names for easy identification
3. **Timeout Configuration**: Adjust timeout based on your workflow needs
4. **Heartbeats**: Ensure long-running processes update heartbeat if custom
5. **Backup Registry**: Periodically backup session registry for disaster recovery
6. **Key Security**: Ensure admin keys have 0600 permissions
7. **Monitor Resources**: Use session list to track resource usage via PID
8. **Documentation**: Document session purpose in profile notes.md

## Security Considerations

### Registry Permissions

- Session registry must have 0600 permissions
- Only owner can read/write registry
- Contains PIDs and environment variables (no secrets)

### Admin SSH Keys

- Private keys stored with 0600 permissions
- Keys unique per session
- Keys generated using cryptographically secure random
- RSA (4096-bit) or Ed25519 (default)

### Environment Variables

- Registry stores environment variable names
- Secret values NOT stored in registry
- Secrets remain in profile's secrets.env

## API Reference

### Session Commands

```bash
aps session list [--profile <id>] [--status <status>] [--tier <tier>]
aps session attach <session-id>
aps session detach <session-id>
```

### Session Registry

```go
// Get the session registry
registry := session.GetRegistry()

// Register a new session
registry.Register(&session.SessionInfo{
    ID:        "session-123",
    ProfileID: "myagent",
    Command:   "/bin/zsh",
    PID:       12345,
    Status:    session.SessionActive,
    Tier:      session.TierBasic,
})

// Get a session
sess, err := registry.Get("session-123")

// List all sessions
sessions := registry.List()

// List by profile
sessions := registry.ListByProfile("myagent")

// List by status
sessions := registry.ListByStatus(session.SessionActive)

// List by tier
sessions := registry.ListByTier(session.TierBasic)

// Update session status
registry.UpdateStatus("session-123", session.SessionInactive)

// Update heartbeat
registry.UpdateHeartbeat("session-123")

// Cleanup inactive sessions
expired := registry.CleanupInactive(30 * time.Minute)

// Save to disk
registry.SaveToDisk()

// Load from disk
registry.LoadFromDisk()
```

### SSH Key Management

```go
// Create SSH key manager
manager := session.NewSSHKeyManager()

// Generate RSA key pair
key, err := manager.GenerateKeyPair(session.SSHKeyRSA)

// Generate Ed25519 key pair
key, err := manager.GenerateKeyPair(session.SSHKeyEd25519)

// Install admin key for session
err := manager.InstallAdminKey("session-123", key)

// Get admin private key
key, err := manager.GetAdminKey("session-123")

// Get admin public key
pubKey, err := manager.GetAdminPublicKey("session-123")

// List all sessions with admin keys
sessions, err := manager.ListAdminSessions()

// Remove admin key
err := manager.RemoveAdminKey("session-123")
```

## Coming Soon

- **Full Attach/Detach**: Complete attach/detach functionality
- **Session Logs**: Capture and view session output
- **Session Metrics**: CPU, memory, network usage
- **Session Alerts**: Notifications for session events
- **Session Sharing**: Share sessions between users
- **Session Templates**: Pre-configured session types
- **Auto-Restart**: Automatically restart failed sessions

## Related Documentation

- [Profiles](PROFILES.md) - Profile management
- [Isolation](ISOLATION.md) - Isolation levels and security
- [CLI](CLI.md) - Complete command reference
- [Examples](EXAMPLES.md) - Practical session examples
