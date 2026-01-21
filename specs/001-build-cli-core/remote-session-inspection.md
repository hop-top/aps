# Remote Session Inspection

**Status**: Design Phase | **Date**: 2026-01-20

## Overview

Remote Session Inspection enables the host/admin user to attach to any profile's live session across all isolation tiers (process, platform, container). This provides both view-only monitoring and interactive control over active profile sessions using tmux as the terminal multiplexer.

---

## Use Cases

1. **Debugging** - Inspect what an agent is doing when it appears stuck
2. **Audit/Compliance** - Monitor session activity in real-time without interruption
3. **Intervention** - Take control of a session to correct misbehavior
4. **Training/Observation** - Watch how agents operate for learning purposes

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CLI / Admin User                            │
│                    (Host machine, primary user)                       │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
                    ┌──────────────────────┐
                    │  Session Manager     │
                    │  - List active       │
                    │  - Attach to tmux    │
                    │  - Detach            │
                    └──────────┬───────────┘
                               │
                    ┌──────────┴───────────┐
                    │   SSH Connection     │
                    │  (auth via admin key)│
                    └──────────┬───────────┘
                               │
                ┌──────────────┼──────────────┐
                ▼              ▼              ▼
          ┌──────────┐  ┌──────────┐  ┌──────────┐
          │ Process  │  │Platform  │  │Container │
          │Sandbox   │  │Sandbox   │  │Sandbox   │
          │ (host)   │  │(ded.usr) │  │(container)│
          └──────────┘  └──────────┘  └──────────┘
                │              │              │
                └──────────────┼──────────────┘
                               ▼
                     ┌──────────────────┐
                     │  tmux server     │
                     │  - Per profile   │
                     │  - Per command   │
                     └──────────────────┘
```

---

## Core Components

### 1. Session Registry

**Purpose**: Track all active tmux sessions across profiles and isolation tiers

**Location**: `~/.aps/sessions/registry.json`

**Schema**:
```json
{
  "sessions": [
    {
      "id": "agent-a-20250120-223045",
      "profile_id": "agent-a",
      "isolation_tier": "platform",
      "tmux_socket": "/tmp/aps-tmux-agent-a-socket",
      "tmux_session_name": "aps-agent-a-1705785045",
      "command": "claude chat",
      "pid": 12345,
      "created_at": "2025-01-20T22:30:45Z",
      "status": "active",
      "host_user": "jadb",
      "sandbox_user": "aps-agent-a"
    }
  ]
}
```

**Operations**:
- Register session on command execution
- Update status (active, detached, terminated)
- List sessions with filters (by profile, status, tier)
- Query by session ID

---

### 2. SSH Key Management

**Purpose**: Provide admin authentication to sandbox users/containers

**Key Location**: `~/.aps/keys/admin_{pub,priv}`

**Distribution**:
- **Tier 1 (Process)**: No SSH needed - direct tmux attachment
- **Tier 2 (Platform)**: Admin public key added to sandbox user's `~/.ssh/authorized_keys`
- **Tier 3 (Container)**: Admin public key added to container user's `~/.ssh/authorized_keys`

**Key Properties**:
- RSA 4096-bit or Ed25519
- Single static key pair generated on APS install
- No passphrase (for seamless admin access)
- Owner: Admin user has full control, no prompts

---

### 3. tmux Integration

**tmux Server per Profile**: Each active profile command runs in its own tmux session

**tmux Socket**: Named socket per profile to avoid conflicts
- Location: `/tmp/aps-tmux-{profile-id}-socket`
- Permissions: 0777 (for host/sandbox access)

**Session Naming Convention**:
- Format: `aps-{profile-id}-{timestamp}`
- Example: `aps-agent-a-1705785045`

**tmux Configuration** (`~/.aps/tmux.conf`):
```
# Allow view-only and control modes
set -g mouse on
set -g status-left "APS Session: #S"
set -g status-right "%Y-%m-%d %H:%M"
set -g allow-passthrough on

# No idle timeout (admin controls session lifecycle)
set -sg escape-time 0
```

---

### 4. Command Execution Flow

#### Original Profile Command

1. User runs: `aps run agent-a -- claude chat`
2. APS starts tmux server for profile (if not running)
3. APS creates tmux session with command
4. APS attaches user's terminal to tmux session (read-write)
5. APS registers session in registry

#### Admin Attachment (View Mode)

1. Admin runs: `aps session attach agent-a --mode view`
2. APS queries session registry for agent-a
3. APS connects via SSH (if Tier 2/3) or direct (if Tier 1)
4. APS runs: `tmux -S {socket} attach -t {session-name} -r`
   - `-r` flag: Read-only mode
5. Admin terminal shows live output, input disabled
6. Admin can detach with `Ctrl+B, D`

#### Admin Attachment (Control Mode)

1. Admin runs: `aps session attach agent-a --mode control`
2. APS queries session registry for agent-a
3. APS connects via SSH (if Tier 2/3) or direct (if Tier 1)
4. APS runs: `tmux -S {socket} attach -t {session-name}`
   - No `-r` flag: Full control mode
5. Admin terminal shows live output, input enabled
6. Both admin and original user can type simultaneously
7. Admin can detach with `Ctrl+B, D`

---

### 5. CLI Commands

```bash
# List active sessions
aps session list [--profile <id>] [--status <active|detached|terminated>]

# Attach to session
aps session attach <profile-id> [--mode view|control] [--latest]

# Detach from current session
aps session detach

# View session details
aps session inspect <session-id>

# View session logs (tmux capture)
aps session logs <session-id> [--lines <n>]

# Terminate session
aps session terminate <session-id>
```

---

## Isolation Tier Integration

### Tier 1: Process Isolation

**Setup**:
- tmux runs on host machine (same user)
- No SSH required
- Direct tmux socket access

**Command Flow**:
```
aps session attach agent-a --mode view
  ↓
tmux -S /tmp/aps-tmux-agent-a-socket attach -t aps-agent-a-12345 -r
```

### Tier 2: Platform Sandbox (macOS/Linux/Windows)

**macOS Example**:
```bash
# Admin attaches via SSH to sandbox user
ssh -i ~/.aps/keys/admin_priv aps-agent-a@localhost \
  "tmux -S /tmp/aps-tmux-agent-a-socket attach -t aps-agent-a-12345 -r"
```

**Linux Example**:
```bash
# Admin attaches via SSH to namespace user
ssh -i ~/.aps/keys/admin_priv aps-agent-a@localhost \
  "tmux -S /tmp/aps-tmux-agent-a-socket attach -t aps-agent-a-12345 -r"
```

**Windows Example**:
```bash
# Admin attaches via WinRM or SSH (if SSH server configured)
ssh -i ~/.aps/keys/admin_priv aps-agent-a@localhost \
  "tmux -S /tmp/aps-tmux-agent-a-socket attach -t aps-agent-a-12345 -r"
```

**Sandbox User Setup** (during profile creation/execution):
- Ensure `~/.ssh` directory exists with 0700 permissions
- Add admin public key to `~/.ssh/authorized_keys`
- Ensure `~/.ssh/authorized_keys` has 0600 permissions
- Ensure `sshd` or compatible SSH server is running

### Tier 3: Container Isolation

**Setup**:
- Admin public key added during container image build
- SSH server installed and running in container
- tmux socket mounted from host or created in container

**Command Flow**:
```bash
# Admin attaches via SSH to container
ssh -i ~/.aps/keys/admin_priv aps-agent-a@<container-ip> \
  "tmux -S /tmp/aps-tmux-agent-a-socket attach -t aps-agent-a-12345 -r"
```

**Container Image Requirements**:
- SSH server (openssh-server or dropbear)
- tmux
- Admin public key in `~/.ssh/authorized_keys`
- Network accessible from host

---

## Filesystem Structure

```
~/.aps/
├── keys/
│   ├── admin_pub          # Admin public key
│   └── admin_priv         # Admin private key (0600)
├── sessions/
│   ├── registry.json      # Active session metadata
│   └── logs/              # Session logs (tmux captures)
│       └── agent-a-20250120-223045.log
└── tmux.conf              # Global tmux configuration

/tmp/
└── aps-tmux-*.socket       # tmux socket files (one per profile)
```

---

## Error Handling

### Session Not Found
```
Error: No active session for profile 'agent-a'
Use 'aps session list' to see available sessions
```

### SSH Connection Failed (Tier 2/3)
```
Error: Failed to connect to sandbox user 'aps-agent-a'
Possible causes:
  - SSH server not running in sandbox
  - Admin key not properly configured
  - Network connectivity issue
```

### tmux Not Installed
```
Error: tmux is required for session inspection
Install: apt-get install tmux (Linux)
         brew install tmux (macOS)
         choco install tmux (Windows)
```

### Permission Denied
```
Error: Permission denied accessing tmux socket
Ensure: /tmp/aps-tmux-{profile-id}-socket has 0777 permissions
```

---

## Security Considerations

### Host/Admin Ownership
- Admin has full control over all sessions
- No kill switch for original user
- Admin can always attach regardless of original user state
- Original user can continue working during admin attachment

### SSH Key Security
- Private key stored at `~/.aps/keys/admin_priv` with 0600 permissions
- Key is static (no rotation) - rely on filesystem permissions
- No passphrase (deliberate tradeoff for seamless admin access)

### Session Persistence
- tmux sessions persist after original user disconnects
- Admin can view/control detached sessions
- Session lifecycle controlled by admin via `aps session terminate`

### Network Exposure (Tier 3)
- Container SSH should bind to localhost only
- Use Docker networking (bridge mode) with port forwarding only if needed
- Document exposure risks if containers have public IPs

---

## Implementation Phases

### Phase 1: Foundation
- [ ] Generate admin SSH key pair on APS install
- [ ] Implement session registry (CRUD operations)
- [ ] Add tmux configuration file
- [ ] Implement `aps session list` command

### Phase 2: Process Isolation
- [ ] Modify `RunCommand()` to wrap commands in tmux sessions
- [ ] Implement session registration on command start
- [ ] Implement `aps session attach` for Tier 1 (view and control modes)
- [ ] Implement `aps session detach`
- [ ] Add unit tests for session lifecycle

### Phase 3: Platform Sandbox Integration
- [ ] Modify sandbox setup to add admin key to `authorized_keys`
- [ ] Ensure SSH server is running in sandbox (or document requirements)
- [ ] Implement SSH connection logic for `aps session attach`
- [ ] Add platform-specific tests (macOS, Linux)
- [ ] Document SSH server requirements per platform

### Phase 4: Container Integration
- [ ] Modify container image builder to install SSH server and tmux
- [ ] Add admin key to container user's `authorized_keys`
- [ ] Configure container networking for SSH access
- [ ] Implement container attachment logic
- [ ] Add E2E tests for container sessions

### Phase 5: CLI Polish
- [ ] Implement `aps session inspect`
- [ ] Implement `aps session logs` (tmux capture)
- [ ] Implement `aps session terminate`
- [ ] Add tab completion for session IDs/profiles
- [ ] Add status indicators in list command

### Phase 6: Documentation & Testing
- [ ] Write admin guide for session inspection
- [ ] Document SSH server setup per platform
- [ ] Document tmux key bindings for admin users
- [ ] Cross-platform E2E test suite
- [ ] Security audit of SSH key handling

---

## Testing Strategy

### Unit Tests
- Session registry operations (add, list, update, delete)
- SSH connection mocking
- tmux command generation validation

### Integration Tests
- Create and attach to process-scoped sessions
- Create and attach to sandbox user sessions (requires SSH server)
- Verify view vs control mode behavior
- Session lifecycle (create, attach, detach, terminate)

### E2E Tests
- Full workflow: Create profile → Run command → Admin attaches (view) → Admin attaches (control) → Admin detaches
- Multiple simultaneous sessions
- Cross-platform matrix (darwin, linux, windows)
- Container isolation sessions

---

## Dependencies

**Required**:
- **tmux** (terminal multiplexer)
  - Install: `brew install tmux` (macOS), `apt-get install tmux` (Linux), `choco install tmux` (Windows)
  - Version: 2.3+ (for `-r` read-only mode)

**Optional (Tier 2/3)**:
- **SSH Server** (OpenSSH, Dropbear, or compatible)
  - Required for platform sandbox and container isolation
  - Must support public key authentication
  - Must run as sandbox user

**Go Packages**:
- `golang.org/x/crypto/ssh` (for SSH connections)
- Existing APS core packages (profile, isolation)

---

## Future Enhancements

### Session Recording
- Record all session I/O for audit/compliance
- Store in `~/.aps/sessions/recordings/`
- Playback with `aps session replay <session-id>`

### Multi-Admin Support
- Allow multiple admins to attach to same session
- Use tmux "allow-passthrough" for coordinated control
- Admin-specific audit logs

### Session Timeouts
- Auto-terminate sessions after N hours of inactivity
- Configurable per profile
- Override by admin

### Real-time Session Streaming
- Web-based session viewer (WebSocket)
- Integration with monitoring dashboards
- Mobile admin access

---

## References

- **tmux Documentation**: https://github.com/tmux/tmux/wiki
- **SSH Authorized Keys**: https://man.openbsd.org/sshd#AUTHORIZED_KEYS_FILE_FORMAT
- **tmux Read-only Mode**: `tmux attach -r`
- **Go SSH Client**: https://pkg.go.dev/golang.org/x/crypto/ssh
- **Isolation Architecture**: `specs/001-build-cli-core/isolation-architecture.md`
