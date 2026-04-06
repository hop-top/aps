# Container Isolation Interface Design

**Date**: 2026-01-21
**Status**: Draft
**Related**: Task T-0002

## Current IsolationManager Interface

```go
type IsolationManager interface {
    PrepareContext(profileID string) (*ExecutionContext, error)
    SetupEnvironment(cmd interface{}) error
    Execute(command string, args []string) error
    ExecuteAction(actionID string, payload []byte) error
    Cleanup() error
    Validate() error
    IsAvailable() bool
}
```

## Proposed Additions for Container Isolation

### 1. New Interface Methods

```go
type IsolationManager interface {
    // Existing methods
    PrepareContext(profileID string) (*ExecutionContext, error)
    SetupEnvironment(cmd interface{}) error
    Execute(command string, args []string) error
    ExecuteAction(actionID string, payload []byte) error
    Cleanup() error
    Validate() error
    IsAvailable() bool

    // New methods for container isolation
    GetContainerID() string
    GetContainerStatus() ContainerStatus
    ApplyResourceLimits(limits ResourceLimits) error
    StopContainer(timeout time.Duration) error
    GetContainerLogs(ctx context.Context, opts LogOptions) (<-chan LogMessage, error)
}
```

### 2. New Types

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

// ResourceLimits defines CPU and memory constraints
type ResourceLimits struct {
    // CPU
    CPUQuota     int64   // in microseconds (e.g., 100000 = 100ms per 100ms period)
    CPUPeriod    int64   // CPU quota period in microseconds (default 100000)
    CPUShares    int64   // Relative CPU weight (default 1024)
    CPUSetCPUs   string  // CPUs to use (e.g., "0-3" or "0,2")
    CPUSetMems   string  // Memory nodes to use (NUMA)

    // Memory
    MemoryLimit  int64   // Memory limit in bytes
    MemorySwap   int64   // Total memory + swap limit in bytes (-1 = unlimited)
    MemorySwappiness int64 // Swappiness (0-100)

    // Disk (optional, future)
    DiskQuota    int64   // Disk quota in bytes
    DiskIOPS    int64   // Disk I/O operations per second
}

// LogOptions defines options for retrieving container logs
type LogOptions struct {
    Since      time.Time
    Until      time.Time
    Follow     bool
    Tail       string  // "all" or number of lines
    ShowStdout bool
    ShowStderr bool
    Timestamps bool
}

// LogMessage represents a single log line from a container
type LogMessage struct {
    Timestamp time.Time
    Stream    string  // "stdout" or "stderr"
    Line      string
}

// VolumeMount represents a host-to-container volume mapping
type VolumeMount struct {
    Source   string  // Host path or named volume
    Target   string  // Container path
    Readonly bool
    Options  []string // Additional mount options
}

// NetworkConfig specifies container network settings
type NetworkConfig struct {
    Mode     string  // "bridge", "host", "none", "custom"
    Network  string  // Network name (for custom networks)
    Ports    []PortMapping
    DNS      []string
    Hostname string
}

// PortMapping defines port forwarding
type PortMapping struct {
    HostIP        string
    HostPort      uint16
    ContainerPort uint16
    Protocol      string // "tcp" or "udp"
}
```

### 3. Container-Specific IsolationManager Implementation

```go
// ContainerIsolation provides container-based isolation
type ContainerIsolation struct {
    context     *ExecutionContext
    engine      ContainerEngine
    containerID string
    status      ContainerStatus
    config      *ContainerConfig
    imageTag    string
    limits      ResourceLimits
}

// ContainerEngine defines the interface for container runtimes
type ContainerEngine interface {
    // Engine operations
    Name() string
    Version() (string, error)
    Ping() error

    // Image operations
    BuildImage(ctx ImageBuildContext) (string, error)
    PullImage(image string) error
    RemoveImage(image string, force bool) error

    // Container lifecycle
    CreateContainer(opts ContainerRunOptions) (string, error)
    StartContainer(id string) error
    StopContainer(id string, timeout time.Duration) error
    RemoveContainer(id string, force bool) error

    // Container operations
    ExecContainer(id string, cmd []string) (int, error)
    GetContainerStatus(id string) (ContainerStatus, error)
    GetContainerLogs(id string, opts LogOptions) (<-chan LogMessage, error)
    UpdateContainerResources(id string, limits ResourceLimits) error

    // Health checks
    Available() bool
}
```

### 4. Enhanced ExecutionContext

```go
type ExecutionContext struct {
    // Existing fields
    ProfileID   string
    ProfileDir  string
    ProfileYaml string
    SecretsPath string
    DocsDir     string
    Environment map[string]string
    WorkingDir  string

    // New container-specific fields
    ContainerID     string
    ContainerName   string
    ContainerImage  string
    Volumes         []VolumeMount
    Network         NetworkConfig
    ResourceLimits  ResourceLimits
    IsolationLevel  IsolationLevel
}
```

## Interface Improvements Analysis

### Current Strengths
- Simple, focused interface
- Works well for process and platform isolation
- Clear lifecycle: prepare -> setup -> execute -> cleanup

### Gaps for Container Isolation
1. **No container lifecycle visibility**: Cannot check if container is running/exited
2. **No resource limit control**: Cannot adjust CPU/memory after creation
3. **No log streaming**: Cannot retrieve container logs for debugging
4. **No exec access**: Cannot run arbitrary commands in existing container
5. **No volume management**: No explicit volume mount interface

### Recommended Changes

#### Option A: Extend Existing Interface (Recommended)
Add optional methods to `IsolationManager`:
- Implementations can return `ErrNotSupported` for unsupported features
- Maintains backward compatibility
- Allows gradual adoption

```go
var (
    ErrNotSupported = errors.New("operation not supported by this isolation level")
)

func (p *ProcessIsolation) GetContainerID() (string, error) {
    return "", ErrNotSupported
}

func (c *ContainerIsolation) GetContainerID() (string, error) {
    return c.containerID, nil
}
```

#### Option B: Create ContainerIsolationManager Sub-interface
```go
type ContainerIsolationManager interface {
    IsolationManager

    GetContainerID() string
    GetContainerStatus() ContainerStatus
    ApplyResourceLimits(limits ResourceLimits) error
    StopContainer(timeout time.Duration) error
    GetContainerLogs(ctx context.Context, opts LogOptions) (<-chan LogMessage, error)
}
```

**Pros**: Type-safe, clear contract for containers
**Cons**: Requires type assertions, more complex API

## Resource Limits Implementation Strategy

### Configuration Schema (profile.yaml)
```yaml
isolation:
  level: "container"
  container:
    image: "ubuntu:22.04"
    limits:
      # CPU limits
      cpu_quota: 100000      # 100ms per 100ms period (1 CPU)
      cpu_period: 100000     # Default period
      cpu_shares: 1024       # Relative weight
      cpu_set_cpus: "0-3"    # Use CPUs 0-3

      # Memory limits
      memory_mb: 1024        # 1GB
      memory_swap_mb: 2048   # 1GB memory + 1GB swap
      memory_swappiness: 60  # Swappiness 0-100
```

### Validation Rules
1. **CPU Quota**: Must be positive if specified
2. **CPU Period**: Must be > 0 (default 100000)
3. **Memory**: Must be positive if specified
4. **Memory Swap**: Must be >= memory limit or -1 for unlimited

### Platform-Specific Implementation

#### Linux (Docker/Podman)
```go
func (d *DockerEngine) UpdateContainerResources(id string, limits ResourceLimits) error {
    update := container.UpdateOptions{
        Resources: container.Resources{
            CPUQuota:  limits.CPUQuota,
            CPUPeriod: limits.CPUPeriod,
            CPUShares: limits.CPUShares,
            CpusetCpus: limits.CPUSetCPus,
            CpusetMems: limits.CPUSetMems,
            Memory:     limits.MemoryLimit,
            MemorySwap: limits.MemorySwap,
            MemoryReservation: 0, // soft limit
        },
    }

    _, err := d.client.ContainerUpdate(context.Background(), id, update)
    return err
}
```

#### macOS (Docker Desktop/Colima)
```go
// Same as Linux, via VM
func (d *DockerEngine) UpdateContainerResources(id string, limits ResourceLimits) error {
    update := container.UpdateOptions{
        Resources: container.Resources{
            CPUQuota:  limits.CPUQuota,
            CPUPeriod: limits.CPUPeriod,
            Memory:     limits.MemoryLimit,
            MemorySwap: limits.MemorySwap,
        },
    }

    _, err := d.client.ContainerUpdate(context.Background(), id, update)
    return err
}
```

## Edge Cases to Handle

### 1. Container Not Found
```go
func (c *ContainerIsolation) GetContainerStatus() (ContainerStatus, error) {
    if c.containerID == "" {
        return ContainerDead, fmt.Errorf("no container ID available")
    }

    return c.engine.GetContainerStatus(c.containerID)
}
```

### 2. Container Already Exists
```go
func (c *ContainerIsolation) Execute(command string, args []string) error {
    // Check if container exists and is running
    if c.containerID != "" {
        status, err := c.engine.GetContainerStatus(c.containerID)
        if err == nil && status == ContainerRunning {
            // Container already running, exec new command
            return c.execInExistingContainer(command, args)
        }
    }

    // Create new container
    return c.executeInNewContainer(command, args)
}
```

### 3. Resource Limits Too High
```go
func (p *Profile) ValidateContainerResources() error {
    if p.Isolation.Level != IsolationContainer {
        return nil
    }

    limits := p.Isolation.Container.Resources

    if limits.CPUQuota > 0 && limits.CPUPeriod <= 0 {
        return fmt.Errorf("cpu_period must be specified when cpu_quota is set")
    }

    if limits.MemoryMB <= 0 {
        return fmt.Errorf("memory_mb must be positive")
    }

    if limits.MemorySwapMB > 0 && limits.MemorySwapMB < limits.MemoryMB {
        return fmt.Errorf("memory_swap_mb must be >= memory_mb")
    }

    return nil
}
```

### 4. Volume Mount Failures
```go
func (c *ContainerIsolation) prepareVolumes() error {
    for _, mount := range c.config.Volumes {
        // Validate source exists on host
        if _, err := os.Stat(mount.Source); err != nil {
            return fmt.Errorf("volume source does not exist: %s: %w", mount.Source, err)
        }

        // Validate target path format
        if !filepath.IsAbs(mount.Target) {
            return fmt.Errorf("volume target must be absolute path: %s", mount.Target)
        }
    }

    return nil
}
```

## Fallback Behavior for Containers

### When Container Engine Not Available
```go
func (m *Manager) GetIsolationManager(profile *Profile) (IsolationManager, error) {
    if profile.Isolation.Level == IsolationContainer {
        engine, err := NewDockerEngine()
        if err != nil || !engine.Available() {
            if profile.Isolation.Fallback {
                log.Warn("Docker not available, falling back to platform isolation")
                return NewPlatformSandbox(), nil
            }
            if profile.Isolation.Strict {
                return nil, fmt.Errorf("container isolation requested but Docker not available (strict mode)")
            }
            return nil, fmt.Errorf("container isolation requested but Docker not available (fallback disabled)")
        }
        return NewContainerIsolation(engine, profile), nil
    }
    // ... handle other levels
}
```

### When Container Creation Fails
```go
func (c *ContainerIsolation) Execute(command string, args []string) error {
    containerID, err := c.engine.CreateContainer(c.buildRunOptions())
    if err != nil {
        if c.config.Fallback {
            log.Warnf("Container creation failed, falling back to platform: %v", err)
            // Reuse platform isolation as fallback
            return NewPlatformSandbox().Execute(command, args)
        }
        return fmt.Errorf("container creation failed: %w", err)
    }

    c.containerID = containerID
    // ... continue with execution
}
```

## Summary

### Recommended Changes
1. ✅ Add optional methods to `IsolationManager` interface
2. ✅ Define new types: `ContainerStatus`, `ResourceLimits`, `LogOptions`, etc.
3. ✅ Create `ContainerEngine` interface for runtime abstraction
4. ✅ Extend `ExecutionContext` with container-specific fields
5. ✅ Implement validation for resource limits
6. ✅ Handle edge cases with clear error messages
7. ✅ Support graceful fallback when containers unavailable

### Backward Compatibility
- Existing implementations (`ProcessIsolation`, `DarwinSandbox`) unchanged
- New methods return `ErrNotSupported` for non-container isolation
- Default behavior preserved for existing profiles

### Testing Requirements
- Unit tests for new interface methods
- Integration tests for `ContainerEngine` implementations
- E2E tests for container lifecycle
- Tests for fallback behavior
- Tests for resource limit validation
