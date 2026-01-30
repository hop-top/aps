# Container Isolation Adapter Implementation Requirements

## Overview

The container isolation adapter provides process-level isolation using Docker containers. This adapter leverages Docker's containerization features to provide strong isolation, resource limits, and portability across platforms.

## System Requirements

### Docker Requirements
- Docker Engine 20.10 or later
- Docker CLI (for user convenience)
- User must have permissions to run Docker

### Docker Version Features
- **20.10+**: Full feature support
- **19.03+**: Basic support with limitations
- **18.09-**: Not supported

### Platform Requirements
- **Linux**: Native Docker support
- **macOS**: Docker Desktop 4.0+
- **Windows**: Docker Desktop 4.0+ or Windows Containers

## Architecture

### Components

```
container_adapter.go       # Main adapter implementation
docker_client.go         # Docker API client
container_config.go      # Container configuration
volume_manager.go        # Volume mounting management
network_manager.go       # Network configuration
session_tracking.go      # Session integration
```

### Integration Points

- Registers with `isolation.Manager` as `IsolationContainer`
- Uses `session.Registry` for container tracking
- Integrates with `core.Config` for container configuration

## Docker Client

### Client Configuration

```go
type DockerClient struct {
    client    *client.Client
    host      string
    version   string
    tlsConfig *tls.Config
}

type ClientConfig struct {
    Host         string
    APIVersion   string
    TLSVerify    bool
    CertPath     string
    KeyPath      string
    CaPath       string
    Timeout      time.Duration
}
```

### Client Implementation

```go
// Create Docker client
func NewDockerClient(config *ClientConfig) (*DockerClient, error)

// Check Docker daemon availability
func (d *DockerClient) IsAvailable() bool

// Get Docker version
func (d *DockerClient) GetVersion() (*types.Version, error)

// List containers
func (d *DockerClient) ListContainers(all bool) ([]types.Container, error)
```

### Client Initialization

```go
func NewDockerClient(config *ClientConfig) (*DockerClient, error) {
    var opts []client.Opt
    
    // Set host
    if config.Host != "" {
        opts = append(opts, client.WithHost(config.Host))
    } else if host := os.Getenv("DOCKER_HOST"); host != "" {
        opts = append(opts, client.WithHost(host))
    }
    
    // Set API version
    if config.APIVersion != "" {
        opts = append(opts, client.WithAPIVersionNegotiation())
    }
    
    // Set TLS
    if config.TLSVerify {
        opts = append(opts, client.WithTLSClientConfig(
            config.CaPath,
            config.CertPath,
            config.KeyPath,
        ))
    }
    
    // Set timeout
    opts = append(opts, client.WithTimeout(config.Timeout))
    
    // Create client
    cli, err := client.NewClientWithOpts(opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %w", err)
    }
    
    return &DockerClient{
        client:  cli,
        host:    config.Host,
        version: config.APIVersion,
    }, nil
}
```

## Container Configuration

### Configuration Structure

```go
type ContainerConfig struct {
    Name          string
    Image         string
    Command       []string
    Env           []string
    WorkingDir    string
    User          string
    Volumes       []VolumeMount
    PortBindings  []PortBinding
    Network       NetworkConfig
    Resources     ResourceConfig
    Security      SecurityConfig
    Labels        map[string]string
    RestartPolicy RestartPolicy
}

type VolumeMount struct {
    Source      string
    Destination string
    ReadOnly    bool
    Type        string // "bind", "volume"
}

type PortBinding struct {
    ContainerPort int
    HostPort     int
    Protocol     string // "tcp", "udp"
}

type NetworkConfig struct {
    Mode      string // "bridge", "host", "none", "container:<id>"
    NetworkID string
    Disable   bool
}

type ResourceConfig struct {
    MemoryMB       int64
    MemorySwapMB   int64
    CPUPercent     float64
    CPUQuota       int64
    CPUPeriod      int64
    CPUShares      int64
    PidsLimit      int64
    DiskSpaceGB    int64
}

type SecurityConfig struct {
    ReadOnlyRoot   bool
    NoNewPrivs     bool
    User           string
    CapAdd         []string
    CapDrop        []string
    SeccompProfile string
    AppArmorProfile string
}

type RestartPolicy struct {
    Name              string // "no", "always", "on-failure", "unless-stopped"
    MaximumRetryCount int
}
```

### Profile-Based Configuration

```yaml
isolation:
  level: container
  strict: false
  fallback: true
  container:
    image: "aps/base:latest"
    command: ["/bin/bash"]
    working_dir: "/workspace"
    user: "aps"
    volumes:
      - source: "{{profile_dir}}"
        destination: "/workspace"
        read_only: false
        type: "bind"
      - source: "{{profile_dir}}/actions"
        destination: "/actions"
        read_only: true
        type: "bind"
    network:
      mode: "bridge"
      disable: false
    resources:
      memory_mb: 1024
      memory_swap_mb: 1024
      cpu_percent: 50.0
      cpu_quota: 50000
      cpu_period: 100000
      cpu_shares: 1024
      pids_limit: 100
    security:
      read_only_root: true
      no_new_privileges: true
      user: "aps:aps"
      cap_add: []
      cap_drop:
        - ALL
    restart_policy:
      name: "no"
```

## Volume Management

### Volume Mounting

```go
// Create volume mounts for profile
func (a *ContainerAdapter) CreateProfileVolumes(profileDir string) ([]VolumeMount, error)

// Bind mount profile directory
func (a *ContainerAdapter) BindMountProfile(profileDir string, dest string, readOnly bool) VolumeMount

// Create named volume
func (a *ContainerAdapter) CreateNamedVolume(name string) (string, error)

// Remove named volume
func (a *ContainerAdapter) RemoveNamedVolume(volumeID string) error
```

### Volume Mount Implementation

```go
func (a *ContainerAdapter) BindMountProfile(profileDir string, dest string, readOnly bool) VolumeMount {
    return VolumeMount{
        Source:      profileDir,
        Destination: dest,
        ReadOnly:    readOnly,
        Type:        "bind",
    }
}

func (a *ContainerAdapter) CreateProfileVolumes(profileDir string) ([]VolumeMount, error) {
    volumes := []VolumeMount{}
    
    // Mount profile directory as workspace
    volumes = append(volumes, VolumeMount{
        Source:      profileDir,
        Destination: "/workspace",
        ReadOnly:    false,
        Type:        "bind",
    })
    
    // Mount actions directory read-only
    volumes = append(volumes, VolumeMount{
        Source:      filepath.Join(profileDir, "actions"),
        Destination: "/actions",
        ReadOnly:    true,
        Type:        "bind",
    })
    
    return volumes, nil
}
```

## Network Management

### Network Configuration

```go
// Configure container network
func (a *ContainerAdapter) ConfigureNetwork(config *NetworkConfig) (*container.HostConfig, error)

// Create isolated network
func (a *ContainerAdapter) CreateIsolatedNetwork(name string) (string, error)

// Remove network
func (a *ContainerAdapter) RemoveNetwork(networkID string) error

// Connect container to network
func (a *ContainerAdapter) ConnectToNetwork(containerID, networkID string) error
```

### Network Implementation

```go
func (a *ContainerAdapter) ConfigureNetwork(config *NetworkConfig) (*container.HostConfig, error) {
    hostConfig := &container.HostConfig{}
    
    switch config.Mode {
    case "none":
        hostConfig.NetworkMode = container.NetworkMode("none")
    case "host":
        hostConfig.NetworkMode = container.NetworkMode("host")
    case "bridge":
        hostConfig.NetworkMode = container.NetworkMode("bridge")
    case "container":
        if config.NetworkID != "" {
            hostConfig.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%s", config.NetworkID))
        } else {
            return nil, fmt.Errorf("container ID required for container network mode")
        }
    default:
        hostConfig.NetworkMode = container.NetworkMode("bridge")
    }
    
    return hostConfig, nil
}
```

## Container Lifecycle

### Create Container

```go
func (a *ContainerAdapter) CreateContainer(config *ContainerConfig) (string, error) {
    ctx := context.Background()
    
    // Parse volumes
    binds, err := a.parseVolumes(config.Volumes)
    if err != nil {
        return "", err
    }
    
    // Parse port bindings
    portBindings, portSet, err := a.parsePorts(config.PortBindings)
    if err != nil {
        return "", err
    }
    
    // Create container
    resp, err := a.dockerClient.client.ContainerCreate(
        ctx,
        &container.Config{
            Image:        config.Image,
            Cmd:          config.Command,
            Env:          config.Env,
            WorkingDir:   config.WorkingDir,
            User:         config.User,
            Labels:       config.Labels,
            StopSignal:   "SIGTERM",
            AttachStdin:  true,
            AttachStdout: true,
            AttachStderr: true,
            Tty:         true,
            OpenStdin:    true,
        },
        &container.HostConfig{
            Binds:        binds,
            PortBindings: portBindings,
            NetworkMode:   a.getNetworkMode(config.Network),
            RestartPolicy: container.RestartPolicy{
                Name:              config.RestartPolicy.Name,
                MaximumRetryCount: config.RestartPolicy.MaximumRetryCount,
            },
            Resources: container.Resources{
                Memory:         config.Resources.MemoryMB * 1024 * 1024,
                MemorySwap:     config.Resources.MemorySwapMB * 1024 * 1024,
                CPUPeriod:     config.Resources.CPUPeriod,
                CPUQuota:      config.Resources.CPUQuota,
                CPUShares:      config.Resources.CPUShares,
                PidsLimit:      &config.Resources.PidsLimit,
                DiskQuota:     &config.Resources.DiskSpaceGB,
            },
            SecurityOpt: []string{},
            ReadonlyRootfs: config.Security.ReadOnlyRoot,
            CapAdd:        config.Security.CapAdd,
            CapDrop:       config.Security.CapDrop,
        },
        nil, // networking config
        nil, // platform specific
        config.Name,
    )
    
    if err != nil {
        return "", fmt.Errorf("failed to create container: %w", err)
    }
    
    return resp.ID, nil
}
```

### Start Container

```go
func (a *ContainerAdapter) StartContainer(containerID string) error {
    ctx := context.Background()
    
    if err := a.dockerClient.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
        return fmt.Errorf("failed to start container: %w", err)
    }
    
    return nil
}
```

### Execute Command in Container

```go
func (a *ContainerAdapter) ExecInContainer(containerID string, cmd []string) (string, error) {
    ctx := context.Background()
    
    // Create exec instance
    execConfig := types.ExecConfig{
        Cmd:          cmd,
        AttachStdin:  true,
        AttachStdout: true,
        AttachStderr: true,
        Tty:          true,
    }
    
    execResp, err := a.dockerClient.client.ContainerExecCreate(ctx, containerID, execConfig)
    if err != nil {
        return "", fmt.Errorf("failed to create exec: %w", err)
    }
    
    // Attach to exec
    hijackResp, err := a.dockerClient.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
        Detach: false,
        Tty:    true,
    })
    if err != nil {
        return "", fmt.Errorf("failed to attach to exec: %w", err)
    }
    defer hijackResp.Close()
    
    // Start exec
    if err := a.dockerClient.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{}); err != nil {
        return "", fmt.Errorf("failed to start exec: %w", err)
    }
    
    // Wait for completion
    statusCh, errCh := a.dockerClient.client.ContainerExecWait(ctx, execResp.ID)
    select {
    case err := <-errCh:
        return "", err
    case status := <-statusCh:
        if status.ExitCode != 0 {
            return "", fmt.Errorf("exec failed with exit code %d", status.ExitCode)
        }
    }
    
    return execResp.ID, nil
}
```

### Stop Container

```go
func (a *ContainerAdapter) StopContainer(containerID string, timeout time.Duration) error {
    ctx := context.Background()
    
    if err := a.dockerClient.client.ContainerStop(ctx, containerID, &timeout); err != nil {
        return fmt.Errorf("failed to stop container: %w", err)
    }
    
    return nil
}
```

### Remove Container

```go
func (a *ContainerAdapter) RemoveContainer(containerID string, force bool) error {
    ctx := context.Background()
    
    if err := a.dockerClient.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
        Force: force,
    }); err != nil {
        return fmt.Errorf("failed to remove container: %w", err)
    }
    
    return nil
}
```

## Session Integration

```go
func (a *ContainerAdapter) Execute(command string, args []string) error {
    sessionID := uuid.New().String()
    
    // Create container config
    config := a.buildContainerConfig()
    config.Command = append([]string{command}, args...)
    
    // Register session
    session := &session.SessionInfo{
        ID:         sessionID,
        ProfileID:  a.context.ProfileID,
        ProfileDir: a.context.ProfileDir,
        Command:    fmt.Sprintf("%s %s", command, strings.Join(args, " ")),
        PID:        0,
        Status:     session.SessionActive,
        Tier:       session.TierPremium,
        CreatedAt:  time.Now(),
        LastSeenAt: time.Now(),
        Environment: map[string]string{
            "container_id":    "", // Set after creation
            "container_image": config.Image,
            "container_name":  config.Name,
        },
    }
    
    registry := session.GetRegistry()
    if err := registry.Register(session); err != nil {
        return err
    }
    
    defer registry.Unregister(sessionID)
    
    // Create container
    containerID, err := a.CreateContainer(config)
    if err != nil {
        return err
    }
    
    // Update session with container ID
    session.Environment["container_id"] = containerID
    
    // Start container
    if err := a.StartContainer(containerID); err != nil {
        a.RemoveContainer(containerID, true)
        return err
    }
    
    // Watch container
    ctx := context.Background()
    statusCh, errCh := a.dockerClient.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
    
    // Start heartbeat
    a.startContainerHeartbeat(sessionID, containerID, 30*time.Second)
    
    // Wait for container
    select {
    case err := <-errCh:
        return fmt.Errorf("container wait error: %w", err)
    case status := <-statusCh:
        if status.StatusCode != 0 {
            return fmt.Errorf("%w: container exited with code %d", isolation.ErrExecutionFailed, status.StatusCode)
        }
    }
    
    return nil
}
```

## Implementation Details

### PrepareContext

```go
func (a *ContainerAdapter) PrepareContext(profileID string) (*isolation.ExecutionContext, error) {
    profile, err := core.LoadProfile(profileID)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", isolation.ErrInvalidProfile, err)
    }
    
    profileDir, err := core.GetProfileDir(profileID)
    if err != nil {
        return nil, err
    }
    
    context := &isolation.ExecutionContext{
        ProfileID:   profileID,
        ProfileDir:  profileDir,
        ProfileYaml: filepath.Join(profileDir, "profile.yaml"),
        SecretsPath: filepath.Join(profileDir, "secrets.env"),
        DocsDir:     filepath.Join(profileDir, "docs"),
        Environment: make(map[string]string),
        WorkingDir:  "/workspace",
    }
    
    // Load container configuration
    a.config = loadContainerConfig(profile.Isolation.Container)
    
    a.context = context
    return context, nil
}
```

### IsAvailable

```go
func (a *ContainerAdapter) IsAvailable() bool {
    // Check Docker daemon
    if !a.isDockerAvailable() {
        return false
    }
    
    // Check Docker version
    version, err := a.dockerClient.GetVersion()
    if err != nil {
        return false
    }
    
    // Check version >= 20.10
    if version.APIVersion < "1.41" {
        return false
    }
    
    return true
}

func (a *ContainerAdapter) isDockerAvailable() bool {
    // Try to ping Docker daemon
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    _, err := a.dockerClient.client.Ping(ctx)
    return err == nil
}
```

## Testing

### Unit Tests

1. **Docker Client**
   - Test client creation
   - Test connection to daemon
   - Test version checking
   - Test error handling

2. **Container Configuration**
   - Test config parsing
   - Test volume parsing
   - Test network configuration
   - Test resource limits

3. **Volume Management**
   - Test volume creation
   - Test volume mounting
   - Test volume cleanup

4. **Container Lifecycle**
   - Test container creation
   - Test container start
   - Test container execution
   - Test container stop
   - Test container removal

### Integration Tests

1. **Full Workflow**
   - Test create/start/stop/remove
   - Test command execution
   - Test environment injection
   - Test volume mounting

2. **Resource Limits**
   - Test memory limits
   - Test CPU limits
   - Test PIDs limits

3. **Network Isolation**
   - Test bridge mode
   - Test host mode
   - Test none mode

## Security Considerations

### Container Security

1. **Image Security**
   - Use signed images when possible
   - Scan images for vulnerabilities
   - Pin to specific image tags

2. **Runtime Security**
   - Enable read-only root filesystem
   - Disable privilege escalation
   - Drop all capabilities by default
   - Use non-root user

3. **Network Security**
   - Use bridge network by default
   - Restrict port bindings
   - Isolate container networks

4. **Resource Security**
   - Enforce memory limits
   - Enforce CPU limits
   - Enforce PIDs limits
   - Prevent resource exhaustion

## Performance Considerations

1. **Container Creation**
   - Reuse base images
   - Pre-pull images
   - Minimize layer count

2. **Volume Management**
   - Use bind mounts for profiles
   - Avoid unnecessary volumes
   - Clean up unused volumes

3. **Network Configuration**
   - Use efficient network modes
   - Minimize network overhead
   - Use DNS caching

## Known Limitations

1. **Docker Dependencies**
   - Requires Docker daemon
   - Requires Docker CLI for some operations
   - Platform-specific limitations

2. **Resource Limits**
   - Memory limits are soft limits
   - CPU limits are best-effort
   - Disk limits are approximate

3. **Networking**
   - Host networking bypasses isolation
   - Port binding may conflict
   - Network configuration varies by platform

## Future Enhancements

1. **Advanced Features**
   - Container health checks
   - Container metrics collection
   - Container auto-restart policies

2. **Multi-Container Support**
   - Docker Compose integration
   - Container orchestration
   - Service discovery

3. **Security Enhancements**
   - gVisor integration
   - Kata Containers support
   - Firecracker integration

## Troubleshooting

### Common Issues

**Container creation fails**
- Check Docker daemon status
- Verify image exists locally
- Check disk space
- Review error logs

**Volume mounting fails**
- Check volume path permissions
- Verify path exists on host
- Check SELinux/AppArmor restrictions

**Network configuration fails**
- Check port availability
- Verify network mode support
- Check firewall rules

### Debug Mode

Enable debug logging:

```yaml
isolation:
  level: container
  container:
    image: "aps/base:latest"
    debug: true
    log_level: debug
    log_file: /tmp/aps-container-debug.log
```

## Related Documentation

### APS Documentation
- [overview.md](overview.md) - Container platform user guide and quick start
- [../../implementation/summaries/container-isolation-summary.md](../../implementation/summaries/container-isolation-summary.md) - Implementation summary with file details
- [../../architecture/design/container-design-summary.md](../../architecture/design/container-design-summary.md) - High-level design decisions
- [../../architecture/design/container-isolation-interface.md](../../architecture/design/container-isolation-interface.md) - ContainerEngine and ImageBuilder interfaces
- [../../architecture/design/container-session-registry.md](../../architecture/design/container-session-registry.md) - Session metadata and registry extensions
- [../../architecture/interfaces/adapter-interface-compliance.md](../../architecture/interfaces/adapter-interface-compliance.md) - Required interface compliance
- [../../testing/container-test-strategy.md](../../testing/container-test-strategy.md) - Test strategy and test matrix
- [../../security/security-audit.md](../../security/security-audit.md) - Security audit and container considerations

### External References
- [Docker API Reference](https://docs.docker.com/engine/api/sdk/)
- [Docker Go SDK](https://github.com/docker/docker/client)
- [Docker Security](https://docs.docker.com/engine/security/)
- [Container Resource Constraints](https://docs.docker.com/config/containers/resource_constraints/)
