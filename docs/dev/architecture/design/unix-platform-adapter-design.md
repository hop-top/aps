# Unix Platform Adapter Design

## Overview

This document proposes improvements to the `IsolationManager` interface design to better support Unix platforms (macOS and Linux), addresses platform-specific edge cases, and defines Unix-only features.

## Current Interface Analysis

### Existing `IsolationManager` Interface

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

### Current `ExecutionContext` Structure

```go
type ExecutionContext struct {
    ProfileID   string
    ProfileDir  string
    ProfileYaml string
    SecretsPath string
    DocsDir     string
    Environment map[string]string
    WorkingDir  string
}
```

## Proposed Interface Improvements for Unix

### 1. Extended `ExecutionContext`

Unix platforms require additional metadata for process isolation and user management:

```go
type ExecutionContext struct {
    ProfileID   string
    ProfileDir  string
    ProfileYaml string
    SecretsPath string
    DocsDir     string
    Environment map[string]string
    WorkingDir  string

    // Unix-specific fields
    UID         int
    GID         int
    HomeDir     string
    Shell       string
    User         string
    Group        string
    NamespaceID  string
    CgroupPath   string
    TmuxSocket  string
}
```

**Rationale:**
- `UID`/`GID`: Required for user namespace isolation on Linux and macOS
- `HomeDir`: Unix home directory differs by user and platform
- `Shell`: Shell varies between platforms (bash, zsh, sh)
- `User`/`Group`: User and group information for process execution
- `NamespaceID`: Namespace identifier for Linux namespace isolation
- `CgroupPath`: cgroup path for Linux resource limits
- `TmuxSocket`: tmux socket path for session management

### 2. Extended `IsolationManager` Interface

Add Unix-specific methods:

```go
type IsolationManager interface {
    PrepareContext(profileID string) (*ExecutionContext, error)
    SetupEnvironment(cmd interface{}) error
    Execute(command string, args []string) error
    ExecuteAction(actionID string, payload []byte) error
    Cleanup() error
    Validate() error
    IsAvailable() bool

    // Unix-specific methods
    GetUserContext(profileID string) (*UserContext, error)
    GetNamespaceContext() (*NamespaceContext, error)
    GetResourceLimits() (*ResourceLimits, error)
    GetTmuxContext() (*TmuxContext, error)
}
```

### 3. New Supporting Types

```go
type UserContext struct {
    UID       int
    GID       int
    HomeDir   string
    Shell     string
    User      string
    Group     string
    Groups    []GroupInfo
}

type GroupInfo struct {
    GID  int
    Name string
}

type NamespaceContext struct {
    UserNamespaceID    string
    PIDNamespaceID     string
    MountNamespaceID   string
    NetworkNamespaceID string
    UTSNamespaceID     string
    IPCNamespaceID     string
    CgroupVersion      int
}

type ResourceLimits struct {
    CPULimit      int
    MemoryLimitMB int
    PIDsLimit     int
    DiskLimitGB   int
    IOReadBPS     int64
    IOWriteBPS    int64
}

type TmuxContext struct {
    SocketPath string
    SessionName string
    ServerPID  int
}
```

## Platform-Specific Edge Cases

### macOS-Specific Edge Cases

#### 1. User Management via `dscl`

macOS uses `dscl` (Directory Service Command Line) for user management:

```go
type macOSUserManager struct{}

func (m *macOSUserManager) CreateUser(username string) (*UserContext, error) {
    cmd := exec.Command("dscl", ".", "-create", "/Users/"+username)
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    cmd = exec.Command("dscl", ".", "-create", "/Users/"+username, "UserShell", "/bin/zsh")
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("failed to set user shell: %w", err)
    }

    // Set other user attributes...
    return m.GetUserContext(username)
}

func (m *macOSUserManager) GetUserContext(username string) (*UserContext, error) {
    cmd := exec.Command("dscl", ".", "-read", "/Users/"+username)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("failed to read user: %w", err)
    }

    return m.parseUserContext(output)
}
```

**Edge Cases:**
- macOS requires admin privileges for user creation
- macOS requires SIP (System Integrity Protection) consideration
- macOS User ID allocation (IDs 500-1000 reserved for system users)
- macOS home directory location (`/Users/username` vs custom)

#### 2. Sandbox Limitations

macOS Sandbox has specific limitations:

```go
type MacOSSandboxConfig struct {
    ProfilePath    string
    AllowedPaths   []string
    DeniedPaths    []string
    NetworkRules   []NetworkRule
    ProfileMode    string // "strict", "permissive", "none"
}

func (m *macOSSandboxAdapter) ApplySandbox(pid int, config *MacOSSandboxConfig) error {
    // Edge case: Some operations require code signing
    if !m.isCodeSigned() {
        return fmt.Errorf("sandbox requires code signing")
    }

    // Edge case: Path rules must use absolute paths
    for _, path := range config.AllowedPaths {
        if !filepath.IsAbs(path) {
            return fmt.Errorf("path must be absolute: %s", path)
        }
    }

    // Edge case: Network restrictions may break TUI apps
    if m.isTUIApplication() && len(config.NetworkRules) > 0 {
        return fmt.Errorf("network restrictions incompatible with TUI")
    }
}
```

**Edge Cases:**
- Sandbox profiles require code signing
- Path rules must be absolute paths
- Network restrictions may break TUI applications
- Some operations (file I/O) may be restricted even with rules

#### 3. Resource Limit Precision

macOS `task_set_policy` has limited precision:

```go
func (m *macOSSandboxAdapter) SetCPULimit(pid int, percent float64) error {
    // Edge case: CPU limits are approximate, not exact
    // Actual behavior varies by macOS version
    taskPolicy, err := m.getTaskPolicy(pid)
    if err != nil {
        return err
    }

    // Approximate CPU limit (not guaranteed)
    taskPolicy.cpuPercentage = percent
    return m.setTaskPolicy(pid, taskPolicy)
}
```

**Edge Cases:**
- CPU limits are approximate, not guaranteed
- Memory limits may be rounded to page size
- I/O limits are not supported on all macOS versions
- Resource limits may be ignored by system processes

### Linux-Specific Edge Cases

#### 1. Namespace Isolation via `unshare`

Linux namespace isolation has specific edge cases:

```go
type LinuxNamespaceManager struct{}

func (l *LinuxNamespaceManager) CreateUserNamespace() (string, error) {
    // Edge case: User namespace requires kernel support
    if !l.hasKernelFeature("user_namespace") {
        return "", fmt.Errorf("kernel does not support user namespaces")
    }

    // Edge case: Rootless namespace creation requires unprivileged user namespace support
    if !l.hasKernelFeature("unprivileged_user_namespace") {
        return "", fmt.Errorf("kernel does not support unprivileged user namespaces")
    }

    nsID, err := l.createNamespace("user")
    if err != nil {
        return "", fmt.Errorf("failed to create namespace: %w", err)
    }

    return nsID, nil
}

func (l *LinuxNamespaceManager) createNamespace(nsType string) (string, error) {
    // Edge case: /proc must be mounted for namespace operations
    if _, err := os.Stat("/proc/self/ns"); os.IsNotExist(err) {
        return "", fmt.Errorf("/proc not mounted")
    }

    // Edge case: Namespace creation may fail due to resource limits
    // Check /proc/sys/user/max_user_namespaces
    maxNS, err := l.readMaxNamespaces()
    if err != nil {
        return "", err
    }

    currentNS, err := l.countCurrentNamespaces()
    if err != nil {
        return "", err
    }

    if currentNS >= maxNS {
        return "", fmt.Errorf("namespace limit reached: %d/%d", currentNS, maxNS)
    }

    return l.createNamespaceInternal(nsType)
}
```

**Edge Cases:**
- Kernel feature support varies by distribution
- Unprivileged namespaces require specific kernel configuration
- Namespace count limits may be reached
- /proc must be mounted for namespace operations
- Namespaces are not automatically cleaned up

#### 2. File ACLs via `setfacl`

Linux file ACLs have specific edge cases:

```go
type LinuxACLManager struct{}

func (l *LinuxACLManager) SetACL(path string, uid int, permissions string) error {
    // Edge case: Filesystem must support ACLs
    if !l.hasACLSupport(path) {
        return fmt.Errorf("filesystem does not support ACLs: %s", path)
    }

    // Edge case: Parent directories must have execute permission
    dir := filepath.Dir(path)
    if err := l.checkExecutePermission(dir); err != nil {
        return fmt.Errorf("parent directory lacks execute permission: %w", err)
    }

    cmd := exec.Command("setfacl", "-m", fmt.Sprintf("u:%d:%s", uid, permissions), path)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to set ACL: %w", err)
    }

    return nil
}

func (l *LinuxACLManager) hasACLSupport(path string) bool {
    // Check filesystem type
    fsType, err := l.getFilesystemType(path)
    if err != nil {
        return false
    }

    // Edge case: Some filesystems don't support ACLs
    switch fsType {
    case "ext2", "ext3", "ext4", "xfs", "btrfs":
        return true
    case "tmpfs", "proc", "sysfs":
        return false
    default:
        return false
    }
}
```

**Edge Cases:**
- Not all filesystems support ACLs (e.g., tmpfs, proc)
- Parent directories must have execute permission
- ACL entries may be ignored if default umask is restrictive
- Some distributions disable ACLs by default

#### 3. Cgroup Version Compatibility

Linux cgroup v1 and v2 have different APIs:

```go
type LinuxCgroupManager struct {
    version int
}

func (l *LinuxCgroupManager) DetectVersion() (int, error) {
    // Edge case: cgroup v2 not available on all systems
    if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
        return 2, nil
    }

    // Edge case: cgroup v1 may not be fully available
    if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
        return 1, nil
    }

    return 0, fmt.Errorf("no cgroup support detected")
}

func (l *LinuxCgroupManager) SetMemoryLimit(path string, limitMB int64) error {
    switch l.version {
    case 1:
        // Edge case: cgroup v1 uses different path structure
        return l.setMemoryLimitV1(path, limitMB)
    case 2:
        // Edge case: cgroup v2 has different memory limit format
        return l.setMemoryLimitV2(path, limitMB)
    default:
        return fmt.Errorf("unknown cgroup version: %d", l.version)
    }
}

func (l *LinuxCgroupManager) setMemoryLimitV1(path string, limitMB int64) error {
    // Edge case: Memory limit must be in bytes for v1
    limitBytes := limitMB * 1024 * 1024
    limitFile := filepath.Join(path, "memory.limit_in_bytes")
    return os.WriteFile(limitFile, []byte(fmt.Sprintf("%d", limitBytes)), 0644)
}

func (l *LinuxCgroupManager) setMemoryLimitV2(path string, limitMB int64) error {
    // Edge case: Memory limit can use MB suffix for v2
    limitFile := filepath.Join(path, "memory.max")
    return os.WriteFile(limitFile, []byte(fmt.Sprintf("%dM", limitMB)), 0644)
}
```

**Edge Cases:**
- cgroup v1 and v2 have different API structures
- Some resource limits are only available in v2
- cgroup v2 may not be fully supported by all distributions
- Path structure differs between versions

## Unix-Only Features

### 1. Shared Workspace Paths

Unix platforms support shared workspace paths via bind mounts:

```go
type UnixWorkspaceManager struct{}

func (u *UnixWorkspaceManager) MountSharedPath(source, target string) error {
    // Edge case: Target must exist before mounting
    if err := os.MkdirAll(target, 0755); err != nil {
        return fmt.Errorf("failed to create target directory: %w", err)
    }

    // Edge case: Mount must be done with root privileges
    if os.Geteuid() != 0 {
        return fmt.Errorf("shared path mounting requires root privileges")
    }

    cmd := exec.Command("mount", "--bind", source, target)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to mount shared path: %w", err)
    }

    return nil
}

func (u *UnixWorkspaceManager) UnmountSharedPath(path string) error {
    cmd := exec.Command("umount", path)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to unmount: %w", err)
    }
    return nil
}
```

**Use Cases:**
- Share configuration files between profiles
- Share cache directories
- Share build artifacts

**Limitations:**
- Requires root privileges
- Not available on all filesystems
- May conflict with existing mounts

### 2. User Namespace Isolation

Linux user namespace isolation allows rootless containers:

```go
type LinuxUserNamespaceManager struct{}

func (l *LinuxUserNamespaceManager) CreateMapFile(uidMapPath string, mappings []UIDMapping) error {
    // Edge case: UID mapping must be within allowed ranges
    for _, m := range mappings {
        if m.Size <= 0 {
            return fmt.Errorf("invalid UID mapping size: %d", m.Size)
        }

        // Edge case: Container ID must be >= 0
        if m.ContainerID < 0 {
            return fmt.Errorf("invalid container UID: %d", m.ContainerID)
        }
    }

    var content strings.Builder
    for _, m := range mappings {
        content.WriteString(fmt.Sprintf("%d %d %d\n", m.ContainerID, m.HostID, m.Size))
    }

    return os.WriteFile(uidMapPath, []byte(content.String()), 0644)
}

func (l *LinuxUserNamespaceManager) DisableSetgroups(pid int) error {
    // Edge case: setgroups file must exist
    setgroupsPath := fmt.Sprintf("/proc/%d/setgroups", pid)
    if _, err := os.Stat(setgroupsPath); os.IsNotExist(err) {
        return fmt.Errorf("setgroups not supported for pid %d", pid)
    }

    return os.WriteFile(setgroupsPath, []byte("deny"), 0644)
}
```

**Use Cases:**
- Rootless container execution
- Isolated user mappings
- Different UIDs/GIDs inside namespace

**Limitations:**
- Requires kernel 3.8+
- Not all distributions enable unprivileged namespaces
- UID/GID mapping can be complex

### 3. Process Signaling

Unix platforms support rich process signaling:

```go
type UnixProcessManager struct{}

func (u *UnixProcessManager) SignalProcess(pid int, sig syscall.Signal) error {
    process, err := os.FindProcess(pid)
    if err != nil {
        return fmt.Errorf("failed to find process: %w", err)
    }

    return process.Signal(sig)
}

func (u *UnixProcessManager) KillProcessTree(rootPID int) error {
    // Edge case: Must kill children before parent
    children, err := u.getChildPIDs(rootPID)
    if err != nil {
        return err
    }

    // Kill children first (reverse order)
    for i := len(children) - 1; i >= 0; i-- {
        if err := u.SignalProcess(children[i], syscall.SIGTERM); err != nil {
            // Continue killing other children even if one fails
            continue
        }
    }

    // Kill parent last
    return u.SignalProcess(rootPID, syscall.SIGTERM)
}

func (u *UnixProcessManager) getChildPIDs(pid int) ([]int, error) {
    procPath := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
    data, err := os.ReadFile(procPath)
    if err != nil {
        return nil, err
    }

    return u.parseChildPIDs(data)
}
```

**Use Cases:**
- Graceful process termination
- Process tree management
- Signal-based cleanup

**Limitations:**
- `/proc` must be mounted
- Child PID enumeration may race with process exit
- Some processes may ignore signals

## Cross-Platform Issues

### 1. Path Separator Differences

**macOS/Linux**: `/`
**Windows**: `\`

**Solution**: Always use `filepath.Join()` and `filepath.Separator`

### 2. Environment Variable Case Sensitivity

**macOS/Linux**: Case sensitive
**Windows**: Case insensitive

**Solution**: Normalize environment variable names to uppercase

### 3. Process Tree Handling

**macOS/Linux**: `/proc` filesystem available
**Windows**: Use Windows API

**Solution**: Use abstraction layer or conditionally compiled code

### 4. Permission Models

**macOS**: POSIX + Sandbox
**Linux**: POSIX + Namespaces + cgroups + SELinux/AppArmor
**Windows**: ACLs + Job Objects

**Solution**: Use platform-specific isolation adapters

### 5. Shell Differences

**macOS**: zsh (default), bash
**Linux**: bash (default), zsh, dash
**Windows**: PowerShell, cmd.exe

**Solution**: Detect shell and adapt command syntax

## Error Handling Patterns

### 1. Consistent Error Types

Define Unix-specific errors:

```go
var (
    ErrNamespaceNotSupported = errors.New("namespace not supported")
    ErrNamespaceLimitReached  = errors.New("namespace limit reached")
    ErrCgroupNotSupported    = errors.New("cgroup not supported")
    ErrACLNotSupported       = errors.New("ACL not supported")
    ErrPrivilegeRequired      = errors.New("privilege required")
    ErrUserNotFound          = errors.New("user not found")
    ErrGroupNotFound         = errors.New("group not found")
    ErrMountFailed            = errors.New("mount failed")
    ErrSignalFailed           = errors.New("signal failed")
)
```

### 2. Error Wrapping Pattern

Wrap errors with context:

```go
func (u *UnixAdapter) SetupNamespaces() error {
    ns, err := u.createNamespace("user")
    if err != nil {
        return fmt.Errorf("failed to create user namespace: %w", err)
    }

    if err := u.applyNamespace(ns); err != nil {
        return fmt.Errorf("failed to apply namespace: %w", err)
    }

    return nil
}
```

### 3. Error Recovery

Implement graceful degradation:

```go
func (u *UnixAdapter) SetupCgroups() error {
    version, err := u.detectCgroupVersion()
    if err != nil {
        return fmt.Errorf("cgroup not available, continuing without: %w", err)
    }

    switch version {
    case 1:
        return u.setupCgroupsV1()
    case 2:
        return u.setupCgroupsV2()
    default:
        return fmt.Errorf("unknown cgroup version: %d", version)
    }
}
```

## Recommendations

### 1. Interface Extensions

- Add Unix-specific methods to `IsolationManager`
- Extend `ExecutionContext` with Unix metadata
- Create Unix-specific supporting types

### 2. Platform Detection

Implement platform detection utility:

```go
func GetUnixPlatform() string {
    switch runtime.GOOS {
    case "darwin":
        return "macos"
    case "linux":
        return "linux"
    default:
        return "unknown"
    }
}

func GetUnixVersion() (string, error) {
    switch runtime.GOOS {
    case "darwin":
        return getmacOSVersion()
    case "linux":
        return getLinuxVersion()
    default:
        return "", fmt.Errorf("unknown platform")
    }
}
```

### 3. Feature Detection

Implement feature detection at runtime:

```go
func DetectUnixFeatures() (*UnixFeatures, error) {
    features := &UnixFeatures{
        UserNamespaces:  hasFeature("user_namespace"),
        Cgroups:         hasFeature("cgroups"),
        Namespaces:      hasFeature("namespaces"),
        ACLs:           hasFeature("acls"),
        Tmux:           hasCommand("tmux"),
        Unshare:        hasCommand("unshare"),
        Setfacl:        hasCommand("setfacl"),
        Dscl:           hasCommand("dscl"),
    }

    return features, nil
}
```

## Acceptance Criteria

✅ **Interface Design**
- [ ] `IsolationManager` interface includes Unix-specific methods
- [ ] `ExecutionContext` includes Unix metadata fields
- [ ] Supporting types for Unix features defined

✅ **Edge Cases**
- [ ] macOS `dscl` capabilities documented
- [ ] Linux `unshare`/`setfacl` capabilities documented
- [ ] Unix-only features (shared workspace paths) documented
- [ ] Cross-platform issues identified

✅ **Error Handling**
- [ ] Unix-specific error types defined
- [ ] Error wrapping pattern documented
- [ ] Error recovery patterns documented

✅ **Platform Detection**
- [ ] Unix platform detection utility proposed
- [ ] Feature detection mechanism proposed
- [ ] Version detection mechanism proposed
