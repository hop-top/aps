# Session Registry Design for Container Isolation

**Date**: 2026-01-21
**Status**: Draft
**Related**: Task T-0001

## Current SessionInfo Structure

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

## Proposed Extensions for Container Metadata

### 1. Enhanced SessionInfo with Container Fields

```go
type SessionInfo struct {
    // Existing fields (unchanged)
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

    // New container-specific fields
    IsolationType    IsolationLevel `json:"isolation_type,omitempty"`
    ContainerID      string         `json:"container_id,omitempty"`
    ContainerName    string         `json:"container_name,omitempty"`
    ContainerImage   string         `json:"container_image,omitempty"`
    ContainerStatus  ContainerStatus `json:"container_status,omitempty"`
    Volumes          []VolumeMount  `json:"volumes,omitempty"`
    Network          NetworkConfig  `json:"network,omitempty"`
    ResourceLimits   ResourceLimits `json:"resource_limits,omitempty"`
    EngineName       string         `json:"engine_name,omitempty"` // "docker", "podman"
    EngineVersion    string         `json:"engine_version,omitempty"`
    ExitCode         int            `json:"exit_code,omitempty"`
    ExitMessage      string         `json:"exit_message,omitempty"`
    HealthStatus     string         `json:"health_status,omitempty"`
    Hostname         string         `json:"hostname,omitempty"`
}
```

### 2. New Types for Container Sessions

```go
// ContainerStatus represents the state of a container
type ContainerStatus string

const (
    ContainerCreated  ContainerStatus = "created"
    ContainerRunning  ContainerStatus = "running"
    ContainerPaused   ContainerStatus = "paused"
    ContainerStopped  ContainerStatus = "stopped"
    ContainerRemoving ContainerStatus = "removing"
    ContainerExited   ContainerStatus = "exited"
    ContainerDead     ContainerStatus = "dead"
)

// VolumeMount represents a host-to-container volume mapping
type VolumeMount struct {
    Source   string `json:"source"`   // Host path or named volume
    Target   string `json:"target"`   // Container path
    Readonly bool   `json:"readonly"`
    Options  string `json:"options"`  // JSON-serialized additional options
}

// NetworkConfig specifies container network settings
type NetworkConfig struct {
    Mode     string   `json:"mode"`     // "bridge", "host", "none"
    Network  string   `json:"network"`  // Network name (for custom networks)
    Ports    []string `json:"ports"`    // Port mappings (simplified for JSON)
    DNS      []string `json:"dns"`
    Hostname string   `json:"hostname"`
}

// ResourceLimits defines CPU and memory constraints
type ResourceLimits struct {
    CPUQuota        int64  `json:"cpu_quota,omitempty"`
    CPUPeriod       int64  `json:"cpu_period,omitempty"`
    CPUShares       int64  `json:"cpu_shares,omitempty"`
    CPUSetCPUs      string `json:"cpu_set_cpus,omitempty"`
    CPUSetMems      string `json:"cpu_set_mems,omitempty"`
    MemoryLimit     int64  `json:"memory_limit,omitempty"`     // in bytes
    MemorySwap      int64  `json:"memory_swap,omitempty"`      // in bytes
    MemorySwappiness int64 `json:"memory_swappiness,omitempty"`
    DiskQuota       int64  `json:"disk_quota,omitempty"`       // optional, future
    DiskIOPS        int64  `json:"disk_iops,omitempty"`        // optional, future
}

// ContainerHealthStatus represents health check results
type ContainerHealthStatus string

const (
    HealthStarting ContainerHealthStatus = "starting"
    HealthHealthy  ContainerHealthStatus = "healthy"
    HealthUnhealthy ContainerHealthStatus = "unhealthy"
    HealthUnknown  ContainerHealthStatus = "unknown"
)
```

## Container Session Lifecycle

### Registration Flow

```go
func (c *ContainerIsolation) registerSession(containerID string, command string, args []string) error {
    fullCommand := strings.Join(append([]string{command}, args...), " ")

    // Get container details from engine
    containerJSON, err := c.engine.InspectContainer(containerID)
    if err != nil {
        return fmt.Errorf("failed to inspect container: %w", err)
    }

    // Get engine version
    engineVersion, _ := c.engine.Version()

    registry := session.GetRegistry()
    sess := &session.SessionInfo{
        ID:               c.tmuxSession,
        ProfileID:        c.context.ProfileID,
        ProfileDir:       c.context.ProfileDir,
        Command:          fmt.Sprintf("docker exec %s %s", containerID, fullCommand),
        PID:              0, // No host PID for containers (container PID space is different)
        Status:           session.SessionActive,
        Tier:             session.TierPremium, // Container isolation = premium tier
        TmuxSocket:       c.tmuxSocket,
        CreatedAt:        time.Now(),
        LastSeenAt:       time.Now(),

        // Container-specific fields
        IsolationType:    core.IsolationContainer,
        ContainerID:      containerID,
        ContainerName:    containerJSON.Name,
        ContainerImage:   containerJSON.Image,
        ContainerStatus:  ContainerRunning,
        Volumes:          c.parseVolumes(containerJSON.Mounts),
        Network:          c.parseNetwork(containerJSON.NetworkSettings),
        ResourceLimits:   c.config.Resources,
        EngineName:       c.engine.Name(),
        EngineVersion:    engineVersion,
        Hostname:         containerJSON.Config.Hostname,
        HealthStatus:     string(HealthStarting),

        Environment: map[string]string{
            "container_id":   containerID,
            "container_name": containerJSON.Name,
            "engine_name":    c.engine.Name(),
            "isolation_type": string(core.IsolationContainer),
        },
    }

    return registry.Register(sess)
}

func (c *ContainerIsolation) parseVolumes(mounts []docker.MountPoint) []session.VolumeMount {
    volumes := make([]session.VolumeMount, len(mounts))
    for i, m := range mounts {
        volumes[i] = session.VolumeMount{
            Source:   m.Source,
            Target:   m.Destination,
            Readonly: m.RW == false,
            Options:  "",
        }
    }
    return volumes
}

func (c *ContainerIsolation) parseNetwork(network *docker.NetworkSettings) session.NetworkConfig {
    return session.NetworkConfig{
        Mode:     "bridge",
        Network:  "",
        Ports:    c.parsePortMappings(network.Ports),
        DNS:      []string{},
        Hostname: "",
    }
}

func (c *ContainerIsolation) parsePortMappings(ports map[docker.Port][]docker.PortBinding) []string {
    var mappings []string
    for port, bindings := range ports {
        if len(bindings) > 0 {
            mapping := fmt.Sprintf("%s:%d->%s", bindings[0].HostIP, bindings[0].HostPort, port.Port())
            mappings = append(mappings, mapping)
        }
    }
    return mappings
}
```

### Session Updates

```go
// UpdateContainerStatus updates the container status in the session
func (r *SessionRegistry) UpdateContainerStatus(sessionID string, status ContainerStatus) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    session, exists := r.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session %s not found", sessionID)
    }

    session.ContainerStatus = status
    session.LastSeenAt = time.Now()

    // Update overall status based on container status
    switch status {
    case ContainerRunning:
        session.Status = SessionActive
        session.HealthStatus = string(HealthHealthy)
    case ContainerExited:
        session.Status = SessionInactive
        if session.ExitCode == 0 {
            session.HealthStatus = string(HealthHealthy)
        } else {
            session.HealthStatus = string(HealthUnhealthy)
        }
    case ContainerDead:
        session.Status = SessionErrored
        session.HealthStatus = string(HealthUnhealthy)
    }

    return nil
}

// UpdateContainerExitCode records the exit code when a container stops
func (r *SessionRegistry) UpdateContainerExitCode(sessionID string, exitCode int, message string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    session, exists := r.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session %s not found", sessionID)
    }

    session.ExitCode = exitCode
    session.ExitMessage = message
    session.LastSeenAt = time.Now()

    return nil
}

// UpdateContainerHealth updates the health check status
func (r *SessionRegistry) UpdateContainerHealth(sessionID string, health ContainerHealthStatus) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    session, exists := r.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session %s not found", sessionID
        )

    session.HealthStatus = string(health)
    session.LastSeenAt = time.Now()

    return nil
}
```

### Session Queries for Containers

```go
// GetContainerSessions returns all sessions using container isolation
func (r *SessionRegistry) GetContainerSessions() []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.IsolationType == core.IsolationContainer {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

// GetSessionsByContainerID returns session(s) for a specific container ID
func (r *SessionRegistry) GetSessionsByContainerID(containerID string) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.ContainerID == containerID {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

// GetSessionsByContainerImage returns all sessions using a specific image
func (r *SessionRegistry) GetSessionsByContainerImage(image string) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.ContainerImage == image {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

// GetSessionsByContainerStatus returns sessions with a specific container status
func (r *SessionRegistry) GetSessionsByContainerStatus(status ContainerStatus) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.ContainerStatus == status {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

// GetSessionsByEngine returns sessions using a specific container engine
func (r *SessionRegistry) GetSessionsByEngine(engineName string) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.EngineName == engineName {
            sessions = append(sessions, session)
        }
    }

    return sessions
}
```

## SSH Key Distribution for Containers

### Problem Statement

Unlike platform isolation (macOS sandbox users), containers don't have system users that can be configured with SSH keys. SSH key distribution must be handled differently:

1. **Volume-mounted SSH directory**: Mount host's SSH config into container
2. **In-container SSH key generation**: Generate keys during container build
3. **SSH agent forwarding**: Forward SSH agent from host to container

### Solution: Volume-Mounted SSH Directory

#### Configuration (profile.yaml)

```yaml
isolation:
  level: "container"
  container:
    image: "ubuntu:22.04"
    volumes:
      # Mount admin's SSH public key for git operations
      - source: "~/.ssh/id_ed25519.pub"
        target: "/root/.ssh/id_ed25519.pub"
        readonly: true
      - source: "~/.ssh/id_ed25519"
        target: "/root/.ssh/id_ed25519"
        readonly: true
      - source: "~/.ssh/config"
        target: "/root/.ssh/config"
        readonly: true
      - source: "~/.ssh/known_hosts"
        target: "/root/.ssh/known_hosts"
        readonly: true
```

#### Implementation

```go
func (c *ContainerIsolation) prepareSSHMounts() error {
    profile, err := core.LoadProfile(c.context.ProfileID)
    if err != nil {
        return err
    }

    // Check if SSH is enabled in profile
    if !profile.SSH.Enabled {
        return nil
    }

    // Get home directory
    home, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get home directory: %w", err)
    }

    sshDir := filepath.Join(home, ".ssh")

    // Define SSH files to mount
    sshFiles := []struct {
        hostPath   string
        container string
        readonly  bool
    }{
        {filepath.Join(sshDir, "id_ed25519.pub"), "/root/.ssh/id_ed25519.pub", true},
        {filepath.Join(sshDir, "id_ed25519"), "/root/.ssh/id_ed25519", true},
        {filepath.Join(sshDir, "config"), "/root/.ssh/config", true},
        {filepath.Join(sshDir, "known_hosts"), "/root/.ssh/known_hosts", true},
    }

    // Add to container volumes configuration
    for _, sshFile := range sshFiles {
        // Verify host file exists
        if _, err := os.Stat(sshFile.hostPath); os.IsNotExist(err) {
            log.Warnf("SSH file not found, skipping mount: %s", sshFile.hostPath)
            continue
        }

        mount := session.VolumeMount{
            Source:   sshFile.hostPath,
            Target:   sshFile.container,
            Readonly: sshFile.readonly,
        }

        c.config.Volumes = append(c.config.Volumes, mount)
    }

    return nil
}

func (c *ContainerIsolation) buildRunOptions() ContainerRunOptions {
    volumes := make([]string, len(c.config.Volumes))
    for i, v := range c.config.Volumes {
        readonly := ":ro"
        if !v.Readonly {
            readonly = ""
        }
        volumes[i] = fmt.Sprintf("%s:%s%s", v.Source, v.Target, readonly)
    }

    return ContainerRunOptions{
        Image:       c.config.Image,
        Environment: c.buildEnvironment(),
        WorkingDir:  c.context.WorkingDir,
        Volumes:     volumes,
        Network:     c.config.Network,
    }
}
```

### Alternative: SSH Agent Forwarding

For better security and without exposing private keys to containers:

```yaml
isolation:
  level: "container"
  container:
    image: "ubuntu:22.04"
    # SSH agent forwarding (via Docker)
    volumes:
      - source: "/run/host-services/ssh-auth.sock"
        target: "/run/host-services/ssh-auth.sock"
        readonly: true
```

```go
func (c *ContainerIsolation) enableSSHAgentForwarding() error {
    // Check if SSH agent socket exists
    sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
    if sshAuthSock == "" {
        // Try macOS Docker Desktop default location
        sshAuthSock = "/run/host-services/ssh-auth.sock"
    }

    if _, err := os.Stat(sshAuthSock); err == nil {
        mount := session.VolumeMount{
            Source:   sshAuthSock,
            Target:   "/run/host-services/ssh-auth.sock",
            Readonly: true,
        }
        c.config.Volumes = append(c.config.Volumes, mount)

        // Set SSH_AUTH_SOCK in container environment
        c.context.Environment["SSH_AUTH_SOCK"] = "/run/host-services/ssh-auth.sock"
    }

    return nil
}
```

## Tmux in Containers

### Problem

Tmux in containers presents challenges:

1. **Terminal handling**: Container has no direct terminal access
2. **Socket location**: Tmux socket in container filesystem, not host
3. **Session persistence**: Container restart loses tmux sessions
4. **Port forwarding**: Need to forward terminal I/O via Docker exec

### Solution: Host-Side Tmux, Container-Side Execution

#### Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Host System                       │
│                                                     │
│  ┌────────────────────────────────────────────┐    │
│  │   Tmux Session (host-side)                 │    │
│  │   Socket: /tmp/aps-tmux-profile-socket     │    │
│  │                                             │    │
│  │   Window 1: docker exec -it container...  │    │
│  └────────────────────────────────────────────┘    │
│                        │                            │
│                        ▼                            │
│  ┌────────────────────────────────────────────┐    │
│  │   Container Isolation Manager              │    │
│  └────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              Docker Container                        │
│                                                     │
│  Command runs in container, but I/O is forwarded    │
│  via docker exec to host-side tmux session           │
│                                                     │
└─────────────────────────────────────────────────────┘
```

#### Implementation

```go
func (c *ContainerIsolation) Execute(command string, args []string) error {
    // Create container if not exists
    if c.containerID == "" {
        if err := c.createContainer(); err != nil {
            return err
        }
    }

    // Start container if not running
    status, _ := c.engine.GetContainerStatus(c.containerID)
    if status != ContainerRunning {
        if err := c.engine.StartContainer(c.containerID); err != nil {
            return err
        }
    }

    // Create host-side tmux session
    c.tmuxSocket = filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-socket", c.context.ProfileID))
    c.tmuxSession = fmt.Sprintf("aps-%s-%d", c.context.ProfileID, time.Now().Unix())

    // Build docker exec command
    fullCommand := strings.Join(append([]string{command}, args...), " ")
    dockerCmd := []string{
        "exec",
        "-i", // Interactive
        "-t", // Allocate pseudo-TTY
        c.containerID,
        "/bin/sh", "-c", fullCommand,
    }

    // Wrap with docker CLI
    wrappedCmd := "docker " + strings.Join(dockerCmd, " ")

    // Create tmux session with docker exec command
    tmuxNewCmd := exec.Command("tmux", "-S", c.tmuxSocket, "new-session", "-d", "-s", c.tmuxSession, "-n", "aps", wrappedCmd)
    tmuxNewCmd.Stdout = os.Stdout
    tmuxNewCmd.Stderr = os.Stderr

    if err := tmuxNewCmd.Run(); err != nil {
        return fmt.Errorf("failed to create tmux session: %w", err)
    }

    // Register session
    if err := c.registerSession(c.containerID, command, args); err != nil {
        c.cleanupTmux()
        return fmt.Errorf("failed to register session: %w", err)
    }

    fmt.Printf("Session started: %s\n", c.tmuxSession)
    fmt.Printf("Container ID: %s\n", c.containerID)
    fmt.Printf("Tmux socket: %s\n", c.tmuxSocket)
    fmt.Printf("Attach with: aps session attach %s\n", c.tmuxSession)

    return nil
}
```

#### Session Registry for Container Tmux

```go
func (c *ContainerIsolation) registerSession(containerID string, command string, args []string) error {
    registry := session.GetRegistry()
    sess := &session.SessionInfo{
        ID:               c.tmuxSession,
        ProfileID:        c.context.ProfileID,
        ProfileDir:       c.context.ProfileDir,
        Command:          fmt.Sprintf("docker exec %s %s", containerID, strings.Join(args, " ")),
        PID:              0, // No host PID
        Status:           session.SessionActive,
        Tier:             session.TierPremium,
        TmuxSocket:       c.tmuxSocket,
        CreatedAt:        time.Now(),
        LastSeenAt:       time.Now(),

        // Container-specific
        IsolationType:    core.IsolationContainer,
        ContainerID:      containerID,
        ContainerImage:   c.config.Image,
        ContainerStatus:  ContainerRunning,
        Volumes:          c.config.Volumes,
        Network:          c.config.Network,
        ResourceLimits:   c.config.Resources,
        EngineName:       c.engine.Name(),
        EngineVersion:    "", // TODO: fetch from engine

        Environment: map[string]string{
            "container_id":    containerID,
            "engine_name":     c.engine.Name(),
            "isolation_type":  string(core.IsolationContainer),
            "tmux_socket":     c.tmuxSocket,
            "tmux_session":    c.tmuxSession,
        },
    }

    return registry.Register(sess)
}
```

### Container Restart Handling

When container restarts, tmux session needs to be recreated:

```go
func (c *ContainerIsolation) handleContainerRestart(containerID string) error {
    // Get existing session from registry
    sessions := session.GetRegistry().GetSessionsByContainerID(containerID)

    for _, sess := range sessions {
        // Clean up old tmux session
        if sess.TmuxSocket != "" {
            cleanupCmd := exec.Command("tmux", "-S", sess.TmuxSocket, "kill-session", "-t", sess.ID)
            _ = cleanupCmd.Run()
        }

        // Create new tmux session with reattached container
        if sess.Command != "" {
            newSessionID := fmt.Sprintf("aps-%s-%d", sess.ProfileID, time.Now().Unix())

            tmuxNewCmd := exec.Command("tmux", "-S", sess.TmuxSocket, "new-session", "-d", "-s", newSessionID, "-n", "aps", sess.Command)
            _ = tmuxNewCmd.Run()

            // Update session registry
            sess.ID = newSessionID
            sess.LastSeenAt = time.Now()
            sess.ContainerStatus = ContainerRunning
            _ = session.GetRegistry().UpdateSession(sess)
        }
    }

    return nil
}
```

## Summary

### Session Registry Changes
1. ✅ Extended `SessionInfo` with container-specific fields
2. ✅ Added new query methods for container sessions
3. ✅ Implemented status updates for container lifecycle
4. ✅ Added health status tracking

### SSH Key Distribution
1. ✅ Volume-mounted SSH directory approach
2. ✅ SSH agent forwarding alternative
3. ✅ Profile configuration support

### Tmux in Containers
1. ✅ Host-side tmux with container execution via docker exec
2. ✅ Session registry integration
3. ✅ Container restart handling

### Testing Requirements
- Unit tests for session registry queries
- Integration tests for container registration
- E2E tests for SSH key mounting
- Tests for tmux session creation and attachment
- Tests for container restart scenarios
