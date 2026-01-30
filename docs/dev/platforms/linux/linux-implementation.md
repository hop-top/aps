# Linux Platform Adapter Implementation Requirements

## Overview

The Linux platform adapter provides advanced process isolation using Linux kernel features including namespaces, cgroups, seccomp, and AppArmor profiles.

## System Requirements

### Minimum Version
- Linux kernel 3.10 or later (for full namespace support)
- Linux kernel 4.14+ recommended (for cgroup v2)
- systemd or cgroup v2 support for resource limits

### Required Kernel Features
- User namespaces (`CONFIG_USER_NS`)
- PID namespaces (`CONFIG_PID_NS`)
- Mount namespaces (`CONFIG_MOUNT_NS`)
- Network namespaces (`CONFIG_NET_NS`)
- UTS namespaces (`CONFIG_UTS_NS`)
- IPC namespaces (`CONFIG_IPC_NS`)
- Control groups (`CONFIG_CGROUPS`)
- seccomp filter support (`CONFIG_SECCOMP`)

### Optional Kernel Features
- AppArmor (`CONFIG_SECURITY_APPARMOR`)
- SELinux (`CONFIG_SECURITY_SELINUX`)
- Landlock (kernel 5.13+)

### Required Tools
- `unshare` (from util-linux)
- `nsenter` (from util-linux)
- `mount` (from util-linux)
- `cgexec` (from libcgroup-tools, for cgroup v1)

## Architecture

### Components

```
linux_platform_adapter.go      # Main adapter implementation
namespaces.go                  # Namespace management
cgroups.go                     # Control group management
seccomp.go                     # seccomp filter management
apparmor.go                    # AppArmor profile management
session_tracking.go           # Session integration
```

### Integration Points

- Registers with `isolation.Manager` as `IsolationPlatform`
- Uses `session.Registry` for process tracking
- Integrates with `core.Config` for isolation configuration

## Namespace Isolation

### Namespace Types

```go
type NamespaceType string

const (
    NamespaceUser    NamespaceType = "user"
    NamespacePID     NamespaceType = "pid"
    NamespaceMount   NamespaceType = "mount"
    NamespaceNetwork NamespaceType = "network"
    NamespaceUTS     NamespaceType = "uts"
    NamespaceIPC     NamespaceType = "ipc"
)

type NamespaceConfig struct {
    Type    NamespaceType
    Enabled bool
    Flags   int
}
```

### Namespace Configuration

```go
type NamespaceSetup struct {
    User    *UserNamespaceConfig
    PID     *PIDNamespaceConfig
    Mount   *MountNamespaceConfig
    Network *NetworkNamespaceConfig
    UTS     *UTSNamespaceConfig
    IPC     *IPCNamespaceConfig
}

type UserNamespaceConfig struct {
    UIDMappings []UIDMapping
    GIDMappings []GIDMapping
    Rootless    bool
}

type UIDMapping struct {
    ContainerID int
    HostID      int
    Size        int
}

type PIDNamespaceConfig struct {
    MaxPIDs     int
    HidePID     bool
}

type MountNamespaceConfig struct {
    Mounts      []MountRule
    ReadOnly    bool
    ProcMount   bool
    SysMount    bool
}

type NetworkNamespaceConfig struct {
    EnableNetwork    bool
    EnableLoopback  bool
    VethPairs       []VethPair
}

type MountRule struct {
    Source      string
    Destination string
    Type        string
    Flags       int
    Options     string
}

type VethPair struct {
    Name        string
    Bridge      string
    IPAddress   string
}
```

### Namespace Creation

```go
// Create new namespaces
func (a *LinuxPlatformAdapter) CreateNamespaces(config *NamespaceSetup) (*NamespaceContext, error)

// Clone process into new namespaces
func (a *LinuxPlatformAdapter) CloneWithNamespaces(cmd *exec.Cmd, config *NamespaceSetup) error

// Enter existing namespaces
func (a *LinuxPlatformAdapter) EnterNamespaces(pid int, types []NamespaceType) error
```

### User Namespace Implementation

```go
// Setup UID/GID mappings
func (a *LinuxPlatformAdapter) setupUserNamespace(pid int, config *UserNamespaceConfig) error {
    // Write UID mappings
    uidMapPath := fmt.Sprintf("/proc/%d/uid_map", pid)
    if err := a.writeMappings(uidMapPath, config.UIDMappings); err != nil {
        return err
    }
    
    // Write GID mappings
    gidMapPath := fmt.Sprintf("/proc/%d/gid_map", pid)
    if err := a.writeMappings(gidMapPath, config.GIDMappings); err != nil {
        return err
    }
    
    // Disable setgroups for rootless
    if config.Rootless {
        if err := os.WriteFile(fmt.Sprintf("/proc/%d/setgroups", pid), []byte("deny"), 0644); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Mount Namespace Implementation

```go
// Setup mount namespace
func (a *LinuxPlatformAdapter) setupMountNamespace(config *MountNamespaceConfig) error {
    // Create new mount namespace
    if err := unix.Unshare(unix.CLONE_NEWNS); err != nil {
        return err
    }
    
    // Make all mounts private
    if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
        return err
    }
    
    // Apply mount rules
    for _, rule := range config.Mounts {
        if err := a.applyMountRule(rule); err != nil {
            return err
        }
    }
    
    // Mount proc
    if config.ProcMount {
        if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Control Groups (Cgroups)

### Cgroup Version Detection

```go
func DetectCgroupVersion() (int, error) {
    // Check for cgroup v2
    if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
        return 2, nil
    }
    
    // Check for cgroup v1
    if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
        return 1, nil
    }
    
    return 0, fmt.Errorf("no cgroup support detected")
}
```

### Cgroup Configuration

```go
type CgroupConfig struct {
    Name      string
    Version   int
    CPU       *CPUConfig
    Memory    *MemoryConfig
    IO        *IOConfig
    PIDs      *PIDsConfig
    Devices   *DevicesConfig
}

type CPUConfig struct {
    Shares      int
    QuotaUS     int64
    PeriodUS    int64
    Cpus        string // CPU affinity (e.g., "0-3")
    Mems        string // NUMA node affinity
}

type MemoryConfig struct {
    LimitMB     int64
    SwapLimitMB int64
    OOMKillDisable bool
    ReservationMB int64
}

type IOConfig struct {
    ReadBPS      int64
    WriteBPS     int64
    ReadIOPS     int64
    WriteIOPS    int64
}

type PIDsConfig struct {
    MaxPIDs int
}

type DevicesConfig struct {
    Allow   bool
    Rules   []DeviceRule
}

type DeviceRule struct {
    Type    string // "a" (all), "b" (block), "c" (char)
    Major   int
    Minor   int
    Access  string // "r", "w", "m"
    Allow   bool
}
```

### Cgroup v1 Implementation

```go
func (a *LinuxPlatformAdapter) setupCgroupV1(pid int, config *CgroupConfig) error {
    // Create cgroup directory
    cgroupPath := fmt.Sprintf("/sys/fs/cgroup/memory/%s", config.Name)
    if err := os.MkdirAll(cgroupPath, 0755); err != nil {
        return err
    }
    
    // Setup memory limits
    if config.Memory != nil {
        if config.Memory.LimitMB > 0 {
            limit := fmt.Sprintf("%dM", config.Memory.LimitMB)
            os.WriteFile(filepath.Join(cgroupPath, "memory.limit_in_bytes"), []byte(limit), 0644)
        }
    }
    
    // Setup CPU limits
    if config.CPU != nil {
        cpuPath := fmt.Sprintf("/sys/fs/cgroup/cpu/%s", config.Name)
        os.MkdirAll(cpuPath, 0755)
        
        if config.CPU.Shares > 0 {
            shares := fmt.Sprintf("%d", config.CPU.Shares)
            os.WriteFile(filepath.Join(cpuPath, "cpu.shares"), []byte(shares), 0644)
        }
    }
    
    // Add process to cgroup
    tasksPath := filepath.Join(cgroupPath, "tasks")
    return os.WriteFile(tasksPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}
```

### Cgroup v2 Implementation

```go
func (a *LinuxPlatformAdapter) setupCgroupV2(pid int, config *CgroupConfig) error {
    // Create cgroup directory
    cgroupPath := fmt.Sprintf("/sys/fs/cgroup/%s", config.Name)
    if err := os.MkdirAll(cgroupPath, 0755); err != nil {
        return err
    }
    
    // Enable controllers
    controllers := "cpu memory io pids"
    os.WriteFile(filepath.Join(cgroupPath, "cgroup.subtree_control"), []byte("+"+controllers), 0644)
    
    // Setup memory limits
    if config.Memory != nil {
        if config.Memory.LimitMB > 0 {
            limit := fmt.Sprintf("%dM", config.Memory.LimitMB)
            os.WriteFile(filepath.Join(cgroupPath, "memory.max"), []byte(limit), 0644)
        }
    }
    
    // Setup CPU limits
    if config.CPU != nil {
        if config.CPU.QuotaUS > 0 && config.CPU.PeriodUS > 0 {
            quota := fmt.Sprintf("%d %d", config.CPU.QuotaUS, config.CPU.PeriodUS)
            os.WriteFile(filepath.Join(cgroupPath, "cpu.max"), []byte(quota), 0644)
        }
    }
    
    // Setup PID limits
    if config.PIDs != nil && config.PIDs.MaxPIDs > 0 {
        maxPIDs := fmt.Sprintf("%d", config.PIDs.MaxPIDs)
        os.WriteFile(filepath.Join(cgroupPath, "pids.max"), []byte(maxPIDs), 0644)
    }
    
    // Add process to cgroup
    procsPath := filepath.Join(cgroupPath, "cgroup.procs")
    return os.WriteFile(procsPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}
```

## seccomp Filters

### Filter Configuration

```go
type SeccompConfig struct {
    DefaultAction  string // "allow", "errno", "kill", "trap", "trace"
    Architectures  []string
    Syscalls       []SyscallRule
}

type SyscallRule struct {
    Names   []string
    Action  string
    Args    []SeccompArg
}

type SeccompArg struct {
    Index    uint
    Value    uint64
    ValueTwo uint64
    Op       string // "EQ", "NE", "LT", "LE", "GT", "GE", "MASKED_EQUAL"
}
```

### Default Profile

```json
{
  "defaultAction": "SCMP_ACT_ALLOW",
  "architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_X32"],
  "syscalls": [
    {
      "names": ["kexec_load", "kexec_file_load", "init_module", "finit_module"],
      "action": "SCMP_ACT_ERRNO"
    },
    {
      "names": ["ptrace"],
      "action": "SCMP_ACT_ERRNO",
      "args": [{"index": 0, "value": 0}]
    }
  ]
}
```

### Implementation

```go
func (a *LinuxPlatformAdapter) applySeccomp(pid int, config *SeccompConfig) error {
    // Load seccomp library
    libseccomp, err := seccomp.NewSeccomp()
    if err != nil {
        return err
    }
    defer libseccomp.Close()
    
    // Set default action
    if err := libseccomp.SetDefaultAction(config.DefaultAction); err != nil {
        return err
    }
    
    // Add syscall rules
    for _, rule := range config.Syscalls {
        for _, name := range rule.Names {
            if err := libseccomp.AddRule(name, rule.Action); err != nil {
                return err
            }
        }
    }
    
    // Apply to process
    return libseccomp.Load()
}
```

## AppArmor Profiles

### Profile Configuration

```go
type AppArmorConfig struct {
    ProfileName string
    Path       string // Path to profile file
    ExecMode   bool   // Use exec mode instead of attach
}
```

### Default Profile

```
#include <tunables/global>

profile aps-platform {
  #include <abstractions/base>
  
  # Allow basic file access
  /var/folders/** r,
  /tmp/** rw,
  
  # Allow network
  network inet stream,
  network inet dgram,
  
  # Deny module loading
  deny /sys/module/** w,
  
  # Allow execution
  /bin/** ix,
  /usr/bin/** ix,
}
```

### Implementation

```go
func (a *LinuxPlatformAdapter) applyAppArmor(cmd *exec.Cmd, config *AppArmorConfig) error {
    if config.ExecMode {
        // Use apparmor_parser to load profile
        if err := a.loadAppArmorProfile(config.Path); err != nil {
            return err
        }
        // Set exec profile
        cmd.Env = append(cmd.Env, fmt.Sprintf("AA_EXEC_PROFILE=%s", config.ProfileName))
    } else {
        // Attach profile to running process
        // Requires AppArmor >= 3.0
    }
    
    return nil
}

func (a *LinuxPlatformAdapter) loadAppArmorProfile(path string) error {
    cmd := exec.Command("apparmor_parser", "-r", path)
    return cmd.Run()
}
```

## Profile Configuration

```yaml
isolation:
  level: platform
  strict: false
  fallback: true
  platform:
    name: linux-namespaces
    cgroup_version: auto  # 1, 2, or auto
    namespaces:
      user:
        enabled: true
        rootless: false
      pid:
        enabled: true
        max_pids: 100
      mount:
        enabled: true
        read_only: false
        proc_mount: true
        sys_mount: false
      network:
        enabled: true
        enable_loopback: true
      uts:
        enabled: false
      ipc:
        enabled: true
    cgroups:
      name: "aps-{{profile_id}}"
      cpu:
        shares: 1024
        quota_us: 100000
        period_us: 100000
        cpus: "0-3"
      memory:
        limit_mb: 1024
        swap_limit_mb: 1024
      pids:
        max_pids: 100
    seccomp:
      enabled: true
      profile: "default"
    apparmor:
      enabled: false
      profile: ""
```

## Session Integration

```go
func (a *LinuxPlatformAdapter) Execute(command string, args []string) error {
    sessionID := uuid.New().String()
    
    session := &session.SessionInfo{
        ID:         sessionID,
        ProfileID:  a.context.ProfileID,
        ProfileDir: a.context.ProfileDir,
        Command:    command,
        PID:        0,
        Status:     session.SessionActive,
        Tier:       session.TierStandard,
        CreatedAt:  time.Now(),
        LastSeenAt: time.Now(),
        Environment: map[string]string{
            "linux_cgroup_path":    a.cgroupPath,
            "linux_namespace_user": getUserNamespaceID(),
            "linux_namespace_pid":  getPIDNamespaceID(),
            "seccomp_profile":     a.config.SeccompProfile,
            "apparmor_profile":     a.config.AppArmorProfile,
        },
    }
    
    registry := session.GetRegistry()
    if err := registry.Register(session); err != nil {
        return err
    }
    
    defer registry.Unregister(sessionID)
    
    // Create namespaces
    if err := a.createNamespaces(); err != nil {
        return err
    }
    
    // Execute command
    cmd := exec.Command(command, args...)
    if err := a.SetupEnvironment(cmd); err != nil {
        return err
    }
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    // Update session PID and setup cgroups
    session.PID = cmd.Process.Pid
    if err := a.setupCgroups(cmd.Process.Pid); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Apply seccomp
    if a.config.SeccompEnabled {
        if err := a.applySeccomp(cmd.Process.Pid, a.config.SeccompConfig); err != nil {
            cmd.Process.Kill()
            return err
        }
    }
    
    // Start heartbeat
    a.startHeartbeat(sessionID, cmd.Process.Pid, 30*time.Second)
    
    // Wait for completion
    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    return nil
}
```

## Testing

### Unit Tests

1. **Namespace Management**
   - Test namespace creation
   - Test namespace cloning
   - Test namespace configuration validation
   - Test rootless user namespaces

2. **Cgroup Management**
   - Test cgroup v1 setup
   - Test cgroup v2 setup
   - Test resource limit enforcement
   - Test process addition to cgroups

3. **seccomp Filters**
   - Test filter loading
   - Test syscall blocking
   - Test filter validation
   - Test default profile

4. **Integration Tests**
   - Test full isolation stack
   - Test resource limit enforcement
   - Test security boundaries
   - Test cleanup

## Security Considerations

### Namespace Security

1. **User Namespaces**
   - Validate UID/GID mappings
   - Prevent privilege escalation
   - Check for rootless constraints

2. **Mount Namespaces**
   - Verify mount sources
   - Prevent escape via bind mounts
   - Validate file permissions

3. **Network Namespaces**
   - Isolate network interfaces
   - Restrict network access
   - Prevent bridge attacks

### Cgroup Security

1. **Resource Limits**
   - Enforce maximum limits
   - Prevent resource exhaustion
   - Validate limit values

2. **Device Access**
   - Deny dangerous device access
   - Validate device permissions
   - Restrict device node creation

### seccomp Security

1. **Filter Validation**
   - Validate syscall names
   - Check filter complexity
   - Test filter effectiveness

2. **Escape Prevention**
   - Block dangerous syscalls
   - Validate argument filters
   - Test filter bypass attempts

## Performance Considerations

1. **Namespace Overhead**
   - Minimize namespace creation time
   - Reuse namespaces where possible
   - Optimize namespace cloning

2. **Cgroup Overhead**
   - Batch cgroup operations
   - Use efficient cgroup version
   - Minimize cgroup path resolution

3. **seccomp Overhead**
   - Optimize filter complexity
   - Cache filter profiles
   - Minimize syscall checks

## Known Limitations

1. **Kernel Version Requirements**
   - Some features require newer kernels
   - Rootless limitations on older kernels
   - Cgroup v2 not available on all systems

2. **Permission Requirements**
   - Some features require CAP_SYS_ADMIN
   - User namespaces require unprivileged user namespace support
   - seccomp requires seccomp syscall support

3. **Distribution Differences**
   - Different cgroup mount points
   - Different AppArmor profiles
   - Different systemd integration

## Troubleshooting

### Common Issues

**Namespaces fail to create**
- Check kernel configuration
- Verify permissions
- Check for namespace count limits

**Cgroups fail to setup**
- Verify cgroup version
- Check cgroup mount points
- Verify systemd integration

**seccomp filters fail to load**
- Check seccomp support in kernel
- Validate filter syntax
- Check for seccomp mode 2 support

### Debug Mode

Enable debug logging:

```yaml
isolation:
  level: platform
  platform:
    name: linux-namespaces
    debug: true
    log_level: debug
    log_file: /tmp/aps-linux-debug.log
```

## References

- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [Control Groups](https://www.kernel.org/doc/Documentation/cgroup-v2.txt)
- [seccomp](https://www.kernel.org/doc/Documentation/prctl/seccomp_filter.txt)
- [AppArmor](https://gitlab.com/apparmor/apparmor/-/wikis/home)
