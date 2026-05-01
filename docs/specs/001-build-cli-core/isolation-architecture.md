# Cross-Platform Isolation Architecture

**Status**: Design Phase | **Date**: 2026-01-20

## Overview

This document outlines the cross-platform isolation strategy for APS, inspired by SandVault's macOS-only approach but generalized for all platforms. The goal is to provide configurable isolation levels while maintaining platform portability.

---

## Isolation Tiers

APS will support three isolation tiers, selectable per profile or globally:

### Tier 1: Process-Scoped (Current Baseline)
- **Description**: Environment-only isolation with same user process
- **Platform Support**: ✅ All (darwin, linux, windows)
- **Security Level**: Low
- **Performance**: < 50ms overhead
- **Implementation**: Current `InjectEnvironment()` approach

### Tier 2: Platform Sandbox
- **Description**: OS-native sandboxing using platform-specific features
- **Platform Support**:
  - ✅ **macOS**: User account isolation (dscl), ACLs, launchctl management
  - ✅ **Linux**: User namespaces, chroot, setfacl, cgroups
  - ✅ **Windows**: Restricted tokens, job objects, Windows Sandbox APIs
- **Security Level**: Medium
- **Performance**: 100-500ms overhead (sandbox setup)
- **Implementation**: Platform adapters

### Tier 3: Container Isolation (Nuclear Option)
- **Description**: Full container per profile, custom-built based on capabilities
- **Platform Support**:
  - ✅ **Linux**: Docker/Podman (native)
  - ✅ **macOS**: Docker Desktop or Colima (via VM)
  - ✅ **Windows**: Docker Desktop via WSL2
- **Security Level**: High
- **Performance**: 1-5s overhead (container start)
- **Implementation**: Container engine adapters

---

## Architecture: Interface-Adapter Pattern

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Core Engine                              │
│  (profile.go, execution.go, action.go, webhook.go)               │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
           ┌────────────────────┐
           │ IsolationManager │ (interface)
           └────────┬───────┘
                    │
        ┌───────────┼───────────┬──────────────┐
        │           │           │              │
        ▼           ▼           ▼              ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐  ┌─────────┐
   │ Process │ │Platform │ │Container│  │  Mock   │
   │Adapter  │ │Sandbox │ │Adapter  │  │Adapter  │
   └─────────┘ └─────────┘ └─────────┘  └─────────┘
        │           │           │              │
   ┌────┴────┐ ┌──┴──┐   ┌───┴──┐       │
   │ All     │ │darwin│   │Docker │       │
   │ Platforms│ │linux │   │Podman │       │
   └─────────┘ │win  │   └───────┘       │
               └──────┘                  │
                Platform-specific            │
                implementations           │
                                          │
                                          ▼
                                   (fallback for
                                    unsupported)
```

---

## Interface Definitions

### IsolationManager Interface

```go
// internal/core/isolation.go

// IsolationLevel defines isolation tier
type IsolationLevel string

const (
    IsolationProcess   IsolationLevel = "process"
    IsolationPlatform IsolationLevel = "platform"
    IsolationContainer IsolationLevel = "container"
)

// IsolationManager defines the contract for isolation strategies
type IsolationManager interface {
    // Name returns the isolation strategy name
    Name() string

    // Supported returns true if this isolation is available on current platform
    Supported() bool

    // Setup prepares the isolation environment before execution
    Setup(profile *Profile, ctx *ExecutionContext) error

    // Teardown cleans up after execution
    Teardown(profile *Profile, ctx *ExecutionContext) error

    // PrepareCommand adapts the command for this isolation level
    PrepareCommand(cmd *exec.Cmd, profile *Profile, ctx *ExecutionContext) error

    // Validate checks if profile requirements can be met
    Validate(profile *Profile) error
}

// ExecutionContext holds runtime state for isolation
type ExecutionContext struct {
    ProfileID    string
    Command      string
    Args         []string
    WorkingDir   string
    Environment  []string
    Metadata     map[string]interface{}
}
```

---

## Platform Adapters

### Process Adapter (Tier 1)

**File**: `internal/core/isolation/process.go`

```go
type ProcessIsolation struct{}

func (p *ProcessIsolation) Name() string {
    return "process"
}

func (p *ProcessIsolation) Supported() bool {
    return true // Always available
}

func (p *ProcessIsolation) Setup(profile *Profile, ctx *ExecutionContext) error {
    // No setup needed - just environment injection
    return nil
}

func (p *ProcessIsolation) PrepareCommand(cmd *exec.Cmd, profile *Profile, ctx *ExecutionContext) error {
    // Current InjectEnvironment behavior
    return InjectEnvironment(cmd, profile)
}

func (p *ProcessIsolation) Teardown(profile *Profile, ctx *ExecutionContext) error {
    return nil // Nothing to clean up
}

func (p *ProcessIsolation) Validate(profile *Profile) error {
    return nil // Always valid
}
```

---

### Platform Sandbox Adapter (Tier 2)

**File**: `internal/core/isolation/platform_sandbox.go`

**Build Tags**:
```go
//go:build darwin || linux || windows
// +build darwin linux windows
```

#### macOS Implementation

```go
// internal/core/isolation/darwin.go
//go:build darwin
// +build darwin

import (
    "fmt"
    "os/exec"
    "runtime"
)

type DarwinSandbox struct {
    user      string
    groupName string
}

func (d *DarwinSandbox) Name() string {
    return "macOS-user-account"
}

func (d *DarwinSandbox) Supported() bool {
    return runtime.GOOS == "darwin"
}

func (d *DarwinSandbox) Setup(profile *Profile, ctx *ExecutionContext) error {
    // Create/ensure sandbox user (similar to SandVault)
    // Configure ACLs on shared workspace
    // Setup sudoers for passwordless switching
    return d.createSandboxUser(profile.ID)
}

func (d *DarwinSandbox) PrepareCommand(cmd *exec.Cmd, profile *Profile, ctx *ExecutionContext) error {
    // Inject environment and wrap with sudo to sandbox user
    // Inject environment
    if err := InjectEnvironment(cmd, profile); err != nil {
        return err
    }

    // Wrap command with sudo to switch to sandbox user
    cmd.Args = append([]string{"-u", d.user, cmd.Path}, cmd.Args...)
    cmd.Path = "/usr/bin/sudo"
    return nil
}

func (d *DarwinSandbox) Teardown(profile *Profile, ctx *ExecutionContext) error {
    // Cleanup processes via launchctl
    return d.cleanupProcesses(d.user)
}

func (d *DarwinSandbox) Validate(profile *Profile) error {
    // Check if dscl is available
    if _, err := exec.LookPath("dscl"); err != nil {
        return fmt.Errorf("dscl not available: %w", err)
    }
    return nil
}

func (d *DarwinSandbox) createSandboxUser(profileID string) error {
    // Implementation similar to SandVault:
    // - Check/create user via dscl
    // - Configure passwordless sudo
    // - Setup SSH key for login
    // - Configure ACLs
    return nil
}
```

#### Linux Implementation

```go
// internal/core/isolation/linux.go
//go:build linux
// +build linux

import (
    "fmt"
    "os/exec"
    "runtime"
)

type LinuxSandbox struct {
    namespaceID string
}

func (l *LinuxSandbox) Name() string {
    return "linux-user-namespace"
}

func (l *LinuxSandbox) Supported() bool {
    return runtime.GOOS == "linux"
}

func (l *LinuxSandbox) Setup(profile *Profile, ctx *ExecutionContext) error {
    // Create user namespace
    // Setup chroot environment
    // Configure filesystem ACLs with setfacl
    return l.setupUserNamespace(profile.ID)
}

func (l *LinuxSandbox) PrepareCommand(cmd *exec.Cmd, profile *Profile, ctx *ExecutionContext) error {
    // Inject environment
    if err := InjectEnvironment(cmd, profile); err != nil {
        return err
    }

    // Wrap with unshare for namespace isolation
    cmd.Args = append([]string{"-r", "-U", "-C", "--mount-proc", cmd.Path}, cmd.Args...)
    cmd.Path = "/usr/bin/unshare"
    return nil
}

func (l *LinuxSandbox) Teardown(profile *Profile, ctx *ExecutionContext) error {
    // Cleanup namespace resources
    return l.cleanupNamespace(l.namespaceID)
}

func (l *LinuxSandbox) Validate(profile *Profile) error {
    // Check if unshare is available
    if _, err := exec.LookPath("unshare"); err != nil {
        return fmt.Errorf("unshare not available: %w", err)
    }
    return nil
}
```

#### Windows Implementation

```go
// internal/core/isolation/windows.go
//go:build windows
// +build windows

import (
    "fmt"
    "runtime"
    "syscall"
    "unsafe"
)

type WindowsSandbox struct {
    tokenHandle syscall.Handle
}

func (w *WindowsSandbox) Name() string {
    return "windows-restricted-token"
}

func (w *WindowsSandbox) Supported() bool {
    return runtime.GOOS == "windows"
}

func (w *WindowsSandbox) Setup(profile *Profile, ctx *ExecutionContext) error {
    // Create restricted token
    // Setup job object for process tracking
    return w.createRestrictedToken(profile.ID)
}

func (w *WindowsSandbox) PrepareCommand(cmd *exec.Cmd, profile *Profile, ctx *ExecutionContext) error {
    // Inject environment
    if err := InjectEnvironment(cmd, profile); err != nil {
        return err
    }

    // Use restricted token via Windows API
    // This requires modifying process token before creation
    return w.applyRestrictedToken(cmd)
}

func (w *WindowsSandbox) Teardown(profile *Profile, ctx *ExecutionContext) error {
    // Close token handle and job object
    return w.cleanupToken(w.tokenHandle)
}

func (w *WindowsSandbox) Validate(profile *Profile) error {
    // Check Windows version supports restricted tokens
    return w.checkWindowsSupport()
}
```

---

### Container Adapter (Tier 3)

**File**: `internal/core/isolation/container.go`

```go
// ContainerIsolation provides container-based isolation
type ContainerIsolation struct {
    engine     ContainerEngine
    imageBuilder ImageBuilder
}

// ContainerEngine defines the interface for container runtimes
type ContainerEngine interface {
    // Available returns true if the container engine is installed and functional
    Available() bool

    // BuildImage builds a container image from the profile configuration
    BuildImage(ctx ImageBuildContext) (string, error)

    // RunContainer executes a command in a new container
    RunContainer(opts ContainerRunOptions) (int, error)

    // StopContainer stops a running container
    StopContainer(id string) error

    // RemoveContainer removes a container
    RemoveContainer(id string) error
}

// ImageBuilder generates container images from profiles
type ImageBuilder interface {
    // Generate creates a Dockerfile/Containerfile from profile capabilities
    Generate(profile *Profile) (string, error)

    // BuildOptions returns build configuration
    BuildOptions(profile *Profile) BuildOptions
}

// ImageBuildContext holds information for image building
type ImageBuildContext struct {
    Profile      *Profile
    ImageTag     string
    BuildDir     string
    Dependencies []string
}

// ContainerRunOptions specifies how to run a container
type ContainerRunOptions struct {
    Image       string
    Command     []string
    Environment []string
    WorkingDir  string
    Volumes     []VolumeMount
    Network     NetworkConfig
    User        string
    Limits      ResourceLimits
}

// VolumeMount represents a host-to-container volume mapping
type VolumeMount struct {
    Source      string
    Target      string
    Readonly    bool
}

// NetworkConfig specifies container network settings
type NetworkConfig struct {
    Enabled bool
    Mode    string // "bridge", "host", "none"
}

// ResourceLimits defines CPU and memory constraints
type ResourceLimits struct {
    CPU    string // e.g., "1.0", "0.5" for 1 or 0.5 CPU cores
    Memory string // e.g., "1g", "512m" for 1GB or 512MB
}
```

#### Docker Engine

```go
// internal/core/isolation/docker.go

import (
    "context"
    "strings"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/client"
)

type DockerEngine struct {
    client *client.Client
}

func NewDockerEngine() (*DockerEngine, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, err
    }
    return &DockerEngine{client: cli}, nil
}

func (d *DockerEngine) Name() string {
    return "docker"
}

func (d *DockerEngine) Supported() bool {
    if d.client == nil {
        if _, err := NewDockerEngine(); err != nil {
            return false
        }
    }
    return true
}

func (d *DockerEngine) BuildImage(ctx ImageBuildContext) (string, error) {
    // Generate Dockerfile from profile capabilities
    dockerfile, err := d.imageBuilder.Generate(ctx.Profile)
    if err != nil {
        return "", err
    }

    // Build container image
    resp, err := d.client.ImageBuild(
        context.Background(),
        strings.NewReader(dockerfile),
        types.ImageBuildOptions{
            Dockerfile: "Dockerfile",
            Tags:      []string{ctx.ImageTag},
            Remove:     true,
        },
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Read build output (could be streamed for progress)
    // ...

    return ctx.ImageTag, nil
}

func (d *DockerEngine) RunContainer(opts ContainerRunOptions) (string, error) {
    // Create volume mounts
    binds := make([]string, len(opts.Volumes))
    for i, v := range opts.Volumes {
        readonly := ":ro"
        if !v.Readonly {
            readonly = ""
        }
        binds[i] = fmt.Sprintf("%s:%s%s", v.Source, v.Target, readonly)
    }

    // Create container
    container, err := d.client.ContainerCreate(
        context.Background(),
        &types.ContainerCreateConfig{
            Image:        opts.Image,
            Cmd:          opts.Command,
            Env:          opts.Environment,
            WorkingDir:    opts.WorkingDir,
            HostConfig: &types.HostConfig{
                Binds:    binds,
                NetworkMode: types.NetworkMode(opts.Network.Mode),
            },
        },
        nil,
        nil,
        nil,
        "",
    )
    if err != nil {
        return "", err
    }

    // Start container
    if err := d.client.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{}); err != nil {
        return "", err
    }

    return container.ID, nil
}

func (d *DockerEngine) StopContainer(id string) error {
    timeout := 30 * time.Second
    return d.client.ContainerStop(context.Background(), id, &timeout)
}

func (d *DockerEngine) RemoveContainer(id string) error {
    return d.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{
        Force: true,
    })
}
```

---

## Profile Configuration

### Extended profile.yaml Schema

```yaml
id: agent-a
display_name: "Agent A"

# Isolation configuration (new section)
isolation:
  # Isolation tier: process | platform | container
  level: "platform"

  # Fallback behavior
  strict: false       # If true, fail fast on unavailable features
  fallback: true     # Allow graceful degradation to process isolation

  # Platform sandbox settings (used when level=platform)
  platform:
    # Shared workspace path (similar to SandVault's /Users/Shared/sv-$USER)
    # Both host user and sandbox user have read/write access here
    shared_workspace: "/Users/Shared/aps-$USER"

  # Container-specific settings (only used when level=container)
  container:
    base_image: "ubuntu:22.04"
    packages:
      - nodejs
      - python3
      - git
    build_steps:
      - type: "shell"
        run: "npm install -g @anthropic-ai/claude-code@latest"
    mounts:
      - source: "/Users/shared/workspace"
        target: "/workspace"
      - source: "/Users/jadb/.agents/profiles/agent-a/secrets.env"
        target: "/secrets.env"
        readonly: true
    # Resource limits (optional)
    limits:
      cpu: "1.0"      # Number of CPU cores (e.g., "1.0", "0.5")
      memory: "1g"     # Memory limit (e.g., "1g", "512m")
    # Network configuration
    network:
      mode: "bridge"    # Options: bridge | host | none

# Tool configuration (optional)
tools:
  claude:
    type: "script"
    path: "tools/claude.sh"
    version: "1.2.0"    # Optional specific version
    auto_install: true
  gemini:
    type: "script"
    path: "tools/gemini.sh"
    auto_install: true
```

---

## Fallback Strategy

### Configuration

`~/.config/aps/config.yaml`:

```yaml
# Global isolation settings
isolation:
  # Default isolation level for profiles without explicit setting
  default_level: "process"  # process | platform | container

  # Allow graceful degradation to process isolation if requested level unavailable
  fallback_enabled: true

  # If true, fail fast when requested isolation is unavailable (no fallback)
  strict_mode: false

  # Container engine settings
  container:
    default_engine: "docker"  # docker | podman
    image_cache_dir: "~/.aps/containers/images"
    container_cache_dir: "~/.aps/containers/cache"
    auto_cleanup: true
```

### Fallback Logic

```go
// internal/core/isolation/manager.go

import (
    "fmt"
    "log"
    "os/exec"
    "runtime"
)

// Manager provides isolation strategies based on configuration
type Manager struct {
    config *Config
}

// GetIsolationManager returns the appropriate isolation adapter for a profile
func (m *Manager) GetIsolationManager(profile *Profile) (IsolationManager, error) {
    config, _ := LoadConfig()

    // 1. Try profile-specified level
    requestedLevel := profile.Isolation.Level
    if requestedLevel == "" {
        requestedLevel = config.Isolation.DefaultLevel
    }

    // 2. Get adapter for requested level
    adapter := m.getAdapterForLevel(requestedLevel)

    // 3. Check support
    if !adapter.Supported() {
        if config.Isolation.StrictMode {
            return nil, fmt.Errorf(
                "isolation level '%s' requested but not supported on %s (strict mode enabled)",
                requestedLevel, runtime.GOOS,
            )
        }

        if config.Isolation.FallbackEnabled {
            log.Warnf(
                "isolation level '%s' unavailable on %s, falling back to 'process'",
                requestedLevel, runtime.GOOS,
            )
            return &ProcessIsolation{}, nil
        }

        return nil, fmt.Errorf(
            "isolation level '%s' not supported on %s and fallback disabled",
            requestedLevel, runtime.GOOS,
        )
    }

    // 4. Validate profile compatibility
    if err := adapter.Validate(profile); err != nil {
        return nil, fmt.Errorf("profile requirements not met: %w", err)
    }

    return adapter, nil
}

func (m *Manager) getAdapterForLevel(level IsolationLevel) IsolationManager {
    switch level {
    case IsolationProcess:
        return &ProcessIsolation{}
    case IsolationPlatform:
        return m.getPlatformAdapter()
    case IsolationContainer:
        return m.getContainerAdapter()
    default:
        return &ProcessIsolation{}
    }
}

func (m *Manager) getPlatformAdapter() IsolationManager {
    switch runtime.GOOS {
    case "darwin":
        return &DarwinSandbox{}
    case "linux":
        return &LinuxSandbox{}
    case "windows":
        return &WindowsSandbox{}
    default:
        return &ProcessIsolation{}
    }
}

func (m *Manager) getContainerAdapter() IsolationManager {
    config, _ := LoadConfig()

    switch config.Isolation.Container.DefaultEngine {
    case "docker":
        if engine, err := NewDockerEngine(); err == nil {
            return &ContainerIsolation{
                engine: engine,
                imageBuilder: &DockerfileBuilder{},
            }
        }
    case "podman":
        // Podman implementation
        // return &ContainerIsolation{...}
    }

    // Fallback to process isolation if no container engine available
    log.Warn("No container engine available, falling back to process isolation")
    return &ProcessIsolation{}
}
```

---

## AI Tool Integration (Agnostic + Customizable)

### Approach 1: Profile Scripts

Profiles can define wrapper scripts for AI tools:

```yaml
# profile.yaml
tools:
  claude:
    type: "script"
    path: "tools/claude.sh"
    auto_install: true
  gemini:
    type: "script"
    path: "tools/gemini.sh"
    auto_install: true
  codex:
    type: "script"
    path: "tools/codex.sh"
    auto_install: true
```

```bash
# ~/.agents/profiles/agent-a/tools/claude.sh
#!/usr/bin/env bash
set -euo pipefail

# Auto-install if needed
if ! command -v claude &> /dev/null; then
    echo "Installing Claude Code..."
    npm install -g @anthropic-ai/claude-code@latest
fi

# Execute with profile context
exec claude "$@"
```

### Approach 2: Container Build Steps

For container isolation, tools can be installed during image build:

```yaml
# profile.yaml
isolation:
  level: "container"
  container:
    base_image: "ubuntu:22.04"
    packages:
      - nodejs
      - python3
      - git
    build_steps:
      - type: "shell"
        run: "npm install -g @anthropic-ai/claude-code@latest"
      - type: "shell"
        run: "pip install -q google-generativeai"
```

### Approach 3: Generic Tool Registry

APS provides helper functions for common AI tool patterns:

```go
// internal/core/tools/manager.go

// Tool represents an AI tool or utility
type Tool struct {
    Name         string
    Description  string
    InstallCmd   []string
    VerifyCmd    []string
    PlatformTags []string  // e.g., ["all"], ["darwin"], ["linux"]
}

// Common tools registry (extendable by users)
var ToolRegistry = map[string]Tool{
    "claude": {
        Name:        "Claude Code",
        Description: "Anthropic's AI coding assistant",
        InstallCmd:  []string{"npm", "install", "-g", "@anthropic-ai/claude-code@latest"},
        VerifyCmd:   []string{"claude", "--version"},
        PlatformTags: []string{"all"},
    },
    "gemini": {
        Name:        "Google Gemini CLI",
        Description: "Google's AI coding assistant",
        InstallCmd:  []string{"npm", "install", "-g", "@google/gemini-cli"},
        VerifyCmd:   []string{"gemini", "--version"},
        PlatformTags: []string{"all"},
    },
    "codex": {
        Name:        "OpenAI Codex",
        Description: "OpenAI's AI coding assistant",
        InstallCmd:  []string{"npm", "install", "-g", "@openai/codex-cli"},
        VerifyCmd:   []string{"codex", "--version"},
        PlatformTags: []string{"all"},
    },
}

// EnsureTool checks if tool exists and installs if needed
func EnsureTool(name string) error {
    tool, ok := ToolRegistry[name]
    if !ok {
        return fmt.Errorf("tool '%s' not in registry", name)
    }

    // Check if already installed
    if _, err := exec.LookPath(tool.InstallCmd[0]); err == nil {
        // Verify it works
        if tool.VerifyCmd != nil {
            cmd := exec.Command(tool.VerifyCmd[0], tool.VerifyCmd[1:]...)
            if err := cmd.Run(); err != nil {
                log.Warnf("Tool '%s' verification failed, reinstalling...", name)
            } else {
                log.Debugf("Tool '%s' already installed and verified", name)
                return nil
            }
        } else {
            return nil
        }
    }

    // Install the tool
    log.Infof("Installing tool '%s' (%s)", name, tool.Name)
    cmd := exec.Command(tool.InstallCmd[0], tool.InstallCmd[1:]...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to install tool '%s': %w", name, err)
    }

    log.Infof("Tool '%s' installed successfully", name)
    return nil
}
```

---

## Integration with Existing Core

### Modified execution.go

```go
// internal/core/execution.go

// RunCommand executes a command within a profile's context
func RunCommand(profileID string, command string, args []string) error {
    profile, err := LoadProfile(profileID)
    if err != nil {
        return err
    }

    // Get isolation manager
    manager := &isolation.Manager{}
    isoManager, err := manager.GetIsolationManager(profile)
    if err != nil {
        return err
    }

    // Create execution context
    ctx := &isolation.ExecutionContext{
        ProfileID:  profileID,
        Command:    command,
        Args:       args,
        WorkingDir: "", // Use current directory or profile-specific
        Environment: os.Environ(),
    }

    // Setup isolation
    if err := isoManager.Setup(profile, ctx); err != nil {
        return fmt.Errorf("isolation setup failed: %w", err)
    }
    defer isoManager.Teardown(profile, ctx)

    // Prepare command
    cmd := exec.Command(command, args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := isoManager.PrepareCommand(cmd, profile, ctx); err != nil {
        return err
    }

    // Execute
    return cmd.Run()
}
```

---

## Implementation Phases

### Phase 1: Foundation (MVP)
- [ ] Define `IsolationManager` interface in `internal/core/isolation/manager.go`
- [ ] Implement `ProcessIsolation` adapter (refactor existing `InjectEnvironment` code)
- [ ] Add isolation configuration to `Profile` struct and schema
- [ ] Update `RunCommand()` to use isolation manager
- [ ] Implement fallback logic with config support
- [ ] Add tests for process isolation (existing tests should pass)

### Phase 2: macOS Sandbox Adapter
- [ ] Implement `DarwinSandbox` adapter in `internal/core/isolation/darwin.go`
- [ ] User account creation/management via `dscl`
- [ ] ACL configuration for shared workspace using `chmod +a`
- [ ] Passwordless sudo/SSH setup
- [ ] Process management via `launchctl`
- [ ] E2E tests for macOS isolation
- [ ] Documentation for macOS-specific setup

### Phase 3: Linux Sandbox Adapter
- [ ] Implement `LinuxSandbox` adapter in `internal/core/isolation/linux.go`
- [ ] User namespace setup using `unshare`
- [ ] Chroot environment creation
- [ ] ACL configuration using `setfacl`
- [ ] Cgroups for resource limiting (optional)
- [ ] E2E tests for Linux isolation
- [ ] Documentation for Linux-specific setup

### Phase 4: Windows Sandbox Adapter
- [ ] Implement `WindowsSandbox` adapter in `internal/core/isolation/windows.go`
- [ ] Restricted token creation via Windows API
- [ ] Job object for process tracking
- [ ] E2E tests for Windows isolation
- [ ] Documentation for Windows-specific setup

### Phase 5: Container Isolation
- [ ] Design `ContainerEngine` and `ImageBuilder` interfaces
- [ ] Implement `DockerfileBuilder` for profile-to-Dockerfile generation
- [ ] Implement `DockerEngine` adapter in `internal/core/isolation/docker.go`
- [ ] Container lifecycle management (create, start, stop, remove)
- [ ] Volume mounting for profile data
- [ ] Network configuration support
- [ ] E2E tests for Docker isolation
- [ ] [Future] `PodmanEngine` adapter

### Phase 6: Tool Management
- [ ] Implement generic tool registry in `internal/core/tools/manager.go`
- [ ] Add common tools (claude, gemini, codex)
- [ ] Support for profile scripts (`tools/` directory)
- [ ] Support for container build steps
- [ ] Auto-installation logic
- [ ] Tool verification checks
- [ ] Documentation for custom tool setup

### Phase 7: Validation & Polish
- [ ] Cross-platform E2E test suite (darwin, linux, windows)
- [ ] Performance benchmarks per isolation tier
- [ ] Security audit for each adapter
- [ ] Documentation updates
- [ ] Migration guide from process-scoped profiles
- [ ] Prepare for v0.5.x (container isolation) GA release
- [ ] Create v1.0.0 roadmap (production-ready milestone)

---

## Versioning Strategy

### v0.1.x (Foundation)
- ✅ Process isolation (Tier 1) - stable, tested
- ✅ IsolationManager interface - complete
- ✅ Fallback logic - implemented
- ✅ Configuration system - complete
- 📄 Documentation for Tier 1

### v0.2.x (macOS Platform Sandbox)
- 🔄 macOS platform sandbox adapter - stable
- 🔄 User account isolation (dscl, ACLs, launchctl)
- 🔄 Passwordless sudo/SSH setup
- 📄 Documentation for macOS setup

### v0.3.x (Linux Platform Sandbox)
- 🔄 Linux platform sandbox adapter - stable
- 🔄 User namespace isolation (unshare, chroot)
- 🔄 ACL configuration (setfacl)
- 📄 Documentation for Linux setup

### v0.4.x (Windows Platform Sandbox)
- 🔄 Windows platform sandbox adapter - stable
- 🔄 Restricted token isolation
- 🔄 Job object for process tracking
- 📄 Documentation for Windows setup

### v0.5.x (Container Isolation)
- 🔄 Docker container isolation - stable
- 🔄 Image builder from profile capabilities
- 🔄 Resource limits (CPU/memory)
- 📄 Documentation for container setup

**Constraint**: Each minor version (0.1.x, 0.2.x, 0.3.x, 0.4.x) must be fully cross-platform compatible. No platform-specific feature gates in minor releases.

**Major Release Consideration**: After v0.5.x is stable, consider v1.0.0 as the "production-ready" release with:
- All isolation tiers battle-tested
- Default isolation possibly upgraded from Tier 1
- Potential breaking changes to profile schema (with migration guide)

---

## Security Considerations

### Process Isolation (Tier 1)
- **Isolation**: Only environment variable separation
- **Filesystem Access**: Same user, same filesystem access
- **Process Boundaries**: Same user, no privilege separation
- **Use Case**: Development environments, trusted tools
- **Not Suitable For**: Untrusted code execution, sensitive credential isolation

### Platform Sandbox (Tier 2)
- **Isolation**: OS-level user/process boundaries
- **Filesystem Access**: Restricted via ACLs/user permissions
- **Process Boundaries**: Separate user/process groups
- **Use Case**: Multi-tenant agents, partial trust scenarios
- **Risk**: Platform-specific implementation complexity, need thorough testing

### Container Isolation (Tier 3)
- **Isolation**: Full kernel-level isolation
- **Filesystem Access**: Custom filesystem per container
- **Process Boundaries**: Separate network namespace, process tree
- **Use Case**: High-security requirements, untrusted code execution
- **Risk**: Container escape vulnerabilities, requires host dependencies

---

## Directory Structure Updates

```
internal/
  core/
    isolation/
      manager.go           # IsolationManager interface + registry
      process.go          # Tier 1 adapter (all platforms)
      platform_sandbox.go  # Tier 2 common logic (build-tagged)
      darwin.go           # macOS implementation
      linux.go            # Linux implementation
      windows.go          # Windows implementation
      container.go        # Tier 3 adapter
      docker.go           # Docker engine implementation
      # podman.go         # Podman engine (future)
    tools/
      manager.go          # Tool registry & installation
      common.go          # Common tools (claude, gemini, codex)
      # (future) custom.go  # User-defined tool support
  cli/
    isolation/           # CLI commands for isolation management
      setup.go           # aps isolation setup
      validate.go        # aps isolation validate
      test.go           # aps isolation test
      status.go          # aps isolation status
```

---

## Migration Path

### For Existing Users

1. **No Changes Required** - Process isolation remains default behavior
2. **Opt-In Upgrade** - Users can enable higher isolation per profile
3. **Gradual Migration** - Start with one profile, test, then expand to others

**Example Migration**:

```bash
# Create new profile with platform isolation
aps profile create secure-agent --isolation-level platform

# Or update existing profile
# Edit: ~/.agents/profiles/agent-a/profile.yaml
# Add:
#   isolation:
#     level: "platform"

# Test the profile
aps run agent-a -- whoami
aps run agent-a -- env | grep APS_
```

### For New Users

1. **Recommended**: Start with process isolation (default)
2. **Evaluate**: Assess security requirements for your use case
3. **Upgrade**: Enable platform/container isolation as needed
4. **Configure**: Customize fallback behavior based on strictness needs

---

## Testing Strategy

### Unit Tests
- Each adapter tested in isolation
- Mock filesystem/exec for portable tests
- Test interface compliance

### Integration Tests
- Test actual sandbox creation/teardown
- Verify filesystem permissions and ACLs
- Test process isolation boundaries
- Mock external dependencies (e.g., Docker daemon) for CI/CD

### E2E Tests
- Full profile lifecycle per platform
- Cross-platform matrix (darwin, linux, windows)
- Security boundary tests (file access, process visibility)
- Fallback logic validation

### Performance Tests
- Benchmark setup time per isolation tier
- Measure environment injection overhead
- Container startup time (cold/warm)

---

## Resolved Design Decisions

1. **Container Networking**
   - **Decision**: Containers support configurable network modes per profile
   - **Implementation**: `network.mode` with options: `bridge` | `host` | `none`
   - **Default**: `bridge` mode for reasonable security/compatibility balance

2. **Resource Limits**
   - **Decision**: CPU and memory limits supported for Tier 2 (platform sandbox) and Tier 3 (container)
   - **Implementation**: `limits.cpu` (e.g., `"1.0"`, `"0.5"`) and `limits.memory` (e.g., `"1g"`, `"512m"`)
   - **Platform Support**:
     - Linux: cgroups (Tier 2), Docker limits (Tier 3)
     - macOS: launchd CPU/memory throttling (Tier 2), Docker limits (Tier 3)
     - Windows: Job object limits (Tier 2), Docker limits (Tier 3)

3. **Image Caching**
   - **Decision**: Aggressive LRU cache with configurable size limit
   - **Implementation**: Cache directory `~/.aps/containers/images` with default 5GB limit
   - **Eviction Policy**: Least-recently-used images deleted when limit exceeded
   - **Configuration**: `config.yaml` option to override cache size

4. **Tool Versioning**
   - **Decision**: Profiles can specify exact tool versions
   - **Implementation**: `tools.<name>.version` field (e.g., `claude@1.2.0`)
   - **Auto-Install**: If version specified, installer will fetch exact version
   - **Fallback**: If no version specified, latest stable version is installed

5. **Shared Workspace for Platform Sandboxes**
   - **Decision**: Shared workspace path similar to SandVault approach
   - **Implementation**: Configurable `isolation.platform.shared_workspace` (default: `/Users/Shared/aps-$USER`)
   - **Permissions**: Both host user and sandbox user have read/write access via ACLs
   - **Cross-Platform**:
     - macOS: `chmod +a` ACLs
     - Linux: `setfacl` ACLs
     - Windows: Shared folder permissions via user group

---

## References

- APS macOS user account isolation implementation

- **Go Build Tags**: https://pkg.go.dev/cmd/go#hdr-Build_constraints
- **Docker SDK**: https://github.com/docker/docker-client
- **Linux Namespaces**: https://man7.org/linux/man-pages/man7/namespaces.7.html
- **Linux ACLs**: https://man7.org/linux/man-pages/man1/setfacl.1.html
- **Windows Restricted Tokens**: https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-restricted_token
- **Go os/exec Package**: https://pkg.go.dev/os/exec
- **Charmbracelet Bubble Tea**: https://github.com/charmbracelet/bubbletea
