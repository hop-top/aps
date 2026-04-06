# Session Inspection Requirements for Platform Adapters

This document defines how platform adapters should integrate with the session management system and what inspection capabilities they must provide.

## Overview

The APS session management system tracks long-running processes across different isolation levels. Platform adapters must register sessions, update status, and support inspection operations.

## Session Registry Structure

The session registry (`internal/core/session/registry.go`) tracks sessions with the following structure:

```go
type SessionInfo struct {
    ID          string            `json:"id"`
    ProfileID   string            `json:"profile_id"`
    ProfileDir  string            `json:"profile_dir,omitempty"`
    Command     string            `json:"command"`
    PID         int               `json:"pid"`
    Status      SessionStatus     `json:"status"`
    Tier        SessionTier       `json:"tier,omitempty"`
    TmuxSocket  string            `json:"tmux_socket,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    LastSeenAt  time.Time         `json:"last_seen_at"`
    Environment map[string]string `json:"environment,omitempty"`
}
```

## Required Session Operations

### 1. Session Registration

When an adapter starts a long-running process, it MUST:

1. Generate a unique session ID (UUID v4)
2. Create a `SessionInfo` structure
3. Register with the session registry
4. Store the session ID for later updates

**Implementation Requirements**:

```go
// In your adapter's Execute method for long-running processes
func (a *MyAdapter) ExecuteLongRunning(command string, args []string) error {
    sessionID := uuid.New().String()
    
    session := &session.SessionInfo{
        ID:         sessionID,
        ProfileID:  a.context.ProfileID,
        ProfileDir: a.context.ProfileDir,
        Command:    command,
        PID:        0, // Set after process starts
        Status:     session.SessionActive,
        Tier:       determineTier(command),
        CreatedAt:  time.Now(),
        LastSeenAt: time.Now(),
    }
    
    registry := session.GetRegistry()
    if err := registry.Register(session); err != nil {
        return fmt.Errorf("failed to register session: %w", err)
    }
    
    // Start process and update PID
    // ...
}
```

**Required Fields**:
- `ID`: Must be a UUID v4
- `ProfileID`: From execution context
- `ProfileDir`: From execution context
- `Command`: The command being executed
- `PID`: Process ID (set after process starts)
- `Status`: Initial status should be `SessionActive`
- `Tier`: Based on command type (basic/standard/premium)
- `CreatedAt`: Time of session creation
- `LastSeenAt`: Time of last heartbeat

### 2. Session Status Updates

Adapters MUST update session status during execution:

```go
// Update status when process completes
func (a *MyAdapter) updateSessionStatus(sessionID string, status session.SessionStatus) error {
    registry := session.GetRegistry()
    return registry.UpdateStatus(sessionID, status)
}
```

**Status Transitions**:
- `SessionActive` → `SessionInactive` (normal completion)
- `SessionActive` → `SessionErrored` (execution failure)
- `SessionErrored` → `SessionActive` (restart/retry)

### 3. Heartbeat Updates

For long-running processes, adapters MUST send periodic heartbeats:

```go
// Send heartbeat at regular intervals (e.g., every 30s)
func (a *MyAdapter) sendHeartbeat(sessionID string) error {
    registry := session.GetRegistry()
    return registry.UpdateHeartbeat(sessionID)
}
```

**Heartbeat Frequency**:
- Minimum: Every 60 seconds
- Recommended: Every 30 seconds
- Maximum: Every 5 minutes

### 4. Session Cleanup

When execution completes (success or failure), adapters MUST:

1. Unregister the session
2. Clean up adapter-specific resources
3. Remove orphaned processes/containers

```go
func (a *MyAdapter) Execute(command string, args []string) error {
    sessionID := uuid.New().String()
    // ... register session
    
    defer func() {
        // Unregister session
        registry := session.GetRegistry()
        _ = registry.Unregister(sessionID)
        
        // Clean up adapter resources
        _ = a.Cleanup()
    }()
    
    // Execute command
    // ...
}
```

## Platform-Specific Inspection Requirements

### Process Isolation Adapter

**PID Tracking**:
- MUST set `PID` field to actual process ID
- MUST verify process is still running during heartbeat
- MUST handle process termination gracefully

**Environment Capture** (Optional):
- Can capture process environment in `Environment` field
- MUST redact secret values if captured

**Example**:
```go
cmd := exec.Command(command, args...)
if err := cmd.Start(); err != nil {
    return err
}

session.PID = cmd.Process.Pid
session.Environment = collectEnvironment(cmd) // optional
```

### Linux Platform Adapter

**Namespace Information**:
- SHOULD capture namespace IDs in session metadata
- SHOULD track cgroup limits applied

**Container/Pod Tracking**:
- If using container runtime, track container ID
- Store container ID in adapter-specific metadata

**Example**:
```go
session.PID = cmd.Process.Pid
session.Environment = map[string]string{
    "linux_namespace_user":   getUserNamespaceID(cmd.Process.Pid),
    "linux_namespace_pid":    getPIDNamespaceID(cmd.Process.Pid),
    "linux_namespace_mount":  getMountNamespaceID(cmd.Process.Pid),
    "linux_namespace_network": getNetworkNamespaceID(cmd.Process.Pid),
}
```

### macOS Platform Adapter

**Sandbox Profile Tracking**:
- SHOULD record which sandbox profile is active
- SHOULD track sandbox restrictions applied

**Process Attributes**:
- SHOULD capture process attributes for inspection

**Example**:
```go
session.PID = cmd.Process.Pid
session.Environment = map[string]string{
    "macos_sandbox_profile": sandboxProfileName,
    "macos_resource_policy": resourcePolicyName,
}
```

### Windows Platform Adapter

**Job Object Tracking**:
- MUST track Job Object handle
- SHOULD store Job Object ID in metadata

**Security Level Tracking**:
- SHOULD record Windows Security Level
- SHOULD track AppContainer ID if applicable

**Example**:
```go
session.PID = cmd.Process.Pid
session.Environment = map[string]string{
    "windows_job_object":      jobObjectName,
    "windows_security_level":  securityLevel,
    "windows_appcontainer_id": appContainerID,
}
```

### Container Isolation Adapter

**Container Tracking**:
- MUST track container ID
- MUST track image used
- SHOULD track resource limits applied

**Container Lifecycle**:
- MUST handle container start/stop events
- MUST clean up containers on session termination

**Example**:
```go
containerID, err := docker.CreateContainer(config)
if err != nil {
    return err
}

session.PID = 0 // containers may not have a PID
session.Environment = map[string]string{
    "container_id":    containerID,
    "container_image": config.Image,
    "container_name":  containerName,
}
```

## Session Inspection Interface

Platform adapters MUST provide inspection methods:

### Required Methods

```go
// GetSessionInfo returns information about a running session
func (a *MyAdapter) GetSessionInfo(sessionID string) (*SessionInfo, error)

// ListSessions returns all active sessions for this adapter
func (a *MyAdapter) ListSessions() []*SessionInfo

// InspectSession returns detailed session metadata
func (a *MyAdapter) InspectSession(sessionID string) (map[string]interface{}, error)
```

### Session Metadata

Each adapter SHOULD provide adapter-specific metadata:

```go
type AdapterSessionMetadata struct {
    SessionID      string
    AdapterType    string
    IsolationLevel string
    Platform       string
    ProcessInfo    ProcessMetadata
    Resources      ResourceMetadata
    Security       SecurityMetadata
}

type ProcessMetadata struct {
    PID              int
    PPID             int
    StartTime        time.Time
    CPUUsage         float64
    MemoryUsageMB    int64
    Threads          int
}

type ResourceMetadata struct {
    CPULimit      string
    MemoryLimitMB int64
    DiskLimitGB   int64
    NetworkPolicy string
}

type SecurityMetadata struct {
    NamespaceIDs    map[string]string
    SandboxProfile  string
    SecurityLevel   string
    AppContainerID  string
}
```

## Session Registry Integration

### Registration Flow

```
1. Adapter prepares execution context
2. Adapter creates SessionInfo structure
3. Adapter calls registry.Register(session)
4. Adapter executes command
5. Adapter updates PID in session
6. Adapter sends periodic heartbeats
7. Adapter updates status on completion
8. Adapter calls registry.Unregister(sessionID)
```

### Inspection Flow

```
1. User requests session inspection (CLI/TUI)
2. Session registry provides SessionInfo
3. Adapter provides detailed metadata (optional)
4. Results are displayed to user
```

## CLI Integration

Platform adapters must support CLI commands for session inspection:

### Session List

```bash
# List all sessions
aps session list

# List sessions for a specific profile
aps session list --profile agent-a

# Filter by status
aps session list --status active

# Filter by tier
aps session list --tier premium
```

### Session Show

```bash
# Show session details
aps session show <session-id>

# Show adapter-specific metadata
aps session show <session-id> --details
```

### Session Attach

```bash
# Attach to a running session
aps session attach <session-id>

# Platform-specific attachment methods supported
```

## Testing Requirements

### Unit Tests

Each adapter MUST have tests for:

1. **Session Registration**
   - Test successful registration
   - Test duplicate session ID handling
   - Test invalid session data

2. **Session Updates**
   - Test status updates
   - Test heartbeat updates
   - Test PID updates

3. **Session Cleanup**
   - Test session unregistration
   - Test resource cleanup
   - Test orphan session handling

4. **Inspection**
   - Test session info retrieval
   - Test session listing
   - Test metadata collection

### Integration Tests

1. **Session Lifecycle**
   - Create session
   - Update status
   - Send heartbeats
   - Cleanup session

2. **Multi-Session Management**
   - Multiple concurrent sessions
   - Session filtering
   - Session termination

3. **Platform-Specific Inspection**
   - Verify PID tracking
   - Verify namespace capture
   - Verify container tracking

## Performance Requirements

- Session registration: < 50ms
- Status update: < 10ms
- Heartbeat update: < 10ms
- Session inspection: < 100ms
- Session list retrieval: < 200ms

## Error Handling

### Required Error Cases

- Session registration failure
- Session not found
- Invalid session ID
- Update failure
- Cleanup failure

### Error Recovery

- If session registration fails, abort execution
- If heartbeat fails, log warning but continue
- If cleanup fails, log error and attempt manual cleanup
- If session not found during update, create new session

## Security Considerations

1. **Session IDs**
   - Use cryptographically secure random UUIDs
   - Don't expose internal PIDs in session IDs

2. **Environment Capture**
   - Redact secret values
   - Don't log sensitive environment variables

3. **Process Information**
   - Only expose necessary process details
   - Don't expose process memory or file descriptors

4. **Access Control**
   - Users can only inspect their own sessions
   - Cross-profile session inspection requires explicit permission

## Backward Compatibility

Existing adapters must continue to work without requiring session management:

- Session registration is OPTIONAL for short-lived processes
- Adapters MUST gracefully handle missing session information
- Inspection commands should return empty results for non-registered sessions

## Documentation Requirements

Each adapter MUST document:

1. **Session Support**
   - Whether adapter supports session tracking
   - What session features are supported
   - Known limitations

2. **Session Metadata**
   - What metadata is captured
   - How to interpret metadata
   - Security considerations

3. **Platform-Specific Behavior**
   - How PID tracking works
   - What platform-specific data is captured
   - Any special handling required
