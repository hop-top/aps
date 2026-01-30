# macOS Platform Adapter Implementation Requirements

## Overview

The macOS platform adapter provides process-level isolation using macOS-specific security mechanisms including sandboxing, resource limits, and process attributes.

## System Requirements

### Minimum Version
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools

### Required Frameworks
- `System` framework for sandbox integration
- `SystemConfiguration` for network policies
- `Security` framework for code signing

### Optional Dependencies
- `codesign` for binary signing (recommended for sandbox profiles)
- `taskinfo` for process attribute inspection

## Architecture

### Components

```
macos_platform_adapter.go       # Main adapter implementation
sandbox_policy.go                # Sandbox profile management
resource_limits.go              # Resource limit management
process_attributes.go           # Process attribute control
session_tracking.go             # Session integration
```

### Integration Points

- Registers with `isolation.Manager` as `IsolationPlatform`
- Uses `session.Registry` for process tracking
- Integrates with `core.Config` for sandbox profile configuration

## Sandbox Integration

### Sandbox Profiles

macOS sandbox profiles define security policies for process execution.

#### Profile Structure

```go
type SandboxProfile struct {
    Name           string
    Version        string
    AllowedPaths   []PathRule
    DeniedPaths    []PathRule
    NetworkRules   []NetworkRule
    IPCRules       []IPCRule
    ProcessRules   []ProcessRule
}

type PathRule struct {
    Path        string
    Mode        string // "read", "write", "read-write"
    Recursive   bool
}

type NetworkRule struct {
    Type    string // "tcp", "udp"
    Action  string // "allow", "deny"
    Address string
    Port    string
}

type IPCRule struct {
    Type    string // "mach", "semaphore"
    Action  string // "allow", "deny"
}

type ProcessRule struct {
    Type    string // "fork", "exec"
    Action  string // "allow", "deny"
}
```

#### Default Profile

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>version</key>
    <integer>2</integer>
    <key>proc-info</key>
    <dict>
        <key>starts-internet</key>
        <true/>
    </dict>
    <key>network-client</key>
    <true/>
    <key>network-server</key>
    <false/>
    <key>file-read-data</key>
    <array>
        <string>/var/folders/</string>
        <string>/Users/</string>
        <string>/tmp/</string>
    </array>
    <key>file-write-data</key>
    <array>
        <string>/var/folders/</string>
        <string>/tmp/</string>
    </array>
    <key>exec</key>
    <true/>
</dict>
</plist>
```

#### Profile Loading

```go
// Load sandbox profile from file
func LoadSandboxProfile(path string) (*SandboxProfile, error)

// Load default profile
func LoadDefaultProfile() (*SandboxProfile, error)

// Validate profile syntax
func ValidateProfile(profile *SandboxProfile) error
```

### Sandbox Enforcement

```go
// Apply sandbox profile to process
func (a *MacOSPlatformAdapter) ApplySandbox(pid int, profile *SandboxProfile) error

// Create sandboxed process
func (a *MacOSPlatformAdapter) CreateSandboxedProcess(cmd *exec.Cmd, profile *SandboxProfile) (*os.Process, error)
```

## Resource Limits

### Using `task_set_policy`

macOS provides `task_set_policy` API for resource limit enforcement.

#### Supported Limits

```go
type ResourceLimits struct {
    CPULimit       int         // CPU time limit in seconds
    MemoryLimitMB  int         // Memory limit in MB
    DiskLimitGB    int         // Disk write limit in GB
    FileDescriptor int         // Max file descriptors
    ProcessCount   int         // Max child processes
    NetworkIO      NetworkLimit // Network I/O limits
}

type NetworkLimit struct {
    UploadMBPerSec   int
    DownloadMBPerSec int
}
```

#### Implementation

```go
// Set CPU limit
func SetCPULimit(pid int, limit int) error

// Set memory limit
func SetMemoryLimit(pid int, limitMB int) error

// Set file descriptor limit
func SetFileDescriptorLimit(pid int, limit int) error
```

### Configuration

Profile-based configuration:

```yaml
isolation:
  level: platform
  strict: false
  fallback: true
  platform:
    name: macos-sandbox
    sandbox_profile: custom-profile.sb
    resource_limits:
      cpu_limit_seconds: 300
      memory_limit_mb: 1024
      file_descriptor_limit: 1024
      process_count_limit: 50
```

## Process Attributes

### Control Flags

```go
type ProcessAttributes struct {
    NoNewPrivs      bool
    NoRemoteAccess  bool
    NoIPC           bool
    NoNetwork       bool
    NoFileSystemWrite bool
}
```

### Implementation

```go
// Apply process attributes
func (a *MacOSPlatformAdapter) ApplyProcessAttributes(pid int, attrs ProcessAttributes) error

// Set process flags
func SetProcessFlag(pid int, flag string, value bool) error
```

## Session Integration

### Session Registration

```go
func (a *MacOSPlatformAdapter) Execute(command string, args []string) error {
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
            "macos_sandbox_profile": a.config.SandboxProfile,
            "macos_resource_policy": a.config.ResourcePolicy,
        },
    }
    
    registry := session.GetRegistry()
    if err := registry.Register(session); err != nil {
        return err
    }
    
    defer registry.Unregister(sessionID)
    
    // Execute with sandbox
    return a.executeWithSandbox(command, args, sessionID)
}
```

### Heartbeat Updates

```go
func (a *MacOSPlatformAdapter) startHeartbeat(sessionID string, pid int, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            if !isProcessRunning(pid) {
                a.updateSessionStatus(sessionID, session.SessionInactive)
                return
            }
            registry := session.GetRegistry()
            _ = registry.UpdateHeartbeat(sessionID)
        }
    }()
}
```

## Implementation Details

### PrepareContext

```go
func (a *MacOSPlatformAdapter) PrepareContext(profileID string) (*isolation.ExecutionContext, error) {
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
        WorkingDir:  profileDir,
    }
    
    // Load macOS-specific configuration
    a.config = loadMacOSConfig(profile.Isolation.Platform)
    
    a.context = context
    return context, nil
}
```

### Execute

```go
func (a *MacOSPlatformAdapter) Execute(command string, args []string) error {
    // Load sandbox profile
    profile, err := a.loadSandboxProfile()
    if err != nil {
        return fmt.Errorf("failed to load sandbox profile: %w", err)
    }
    
    // Create command
    cmd := exec.Command(command, args...)
    
    // Setup environment
    if err := a.SetupEnvironment(cmd); err != nil {
        return err
    }
    
    // Apply sandbox
    if err := a.ApplySandbox(cmd, profile); err != nil {
        return fmt.Errorf("failed to apply sandbox: %w", err)
    }
    
    // Set resource limits
    if err := a.setResourceLimits(cmd.Process); err != nil {
        return err
    }
    
    // Start process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    // Update session PID
    a.sessionPID = cmd.Process.Pid
    
    // Start heartbeat
    a.startHeartbeat(a.sessionID, cmd.Process.Pid, 30*time.Second)
    
    // Wait for completion
    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    return nil
}
```

### IsAvailable

```go
func (a *MacOSPlatformAdapter) IsAvailable() bool {
    if runtime.GOOS != "darwin" {
        return false
    }
    
    // Check macOS version
    if !isMacOSSupported() {
        return false
    }
    
    // Check for required frameworks
    if !hasSandboxFramework() {
        return false
    }
    
    return true
}

func isMacOSSupported() bool {
    var out bytes.Buffer
    cmd := exec.Command("sw_vers", "-productVersion")
    cmd.Stdout = &out
    if err := cmd.Run(); err != nil {
        return false
    }
    
    version := strings.TrimSpace(out.String())
    return compareVersions(version, "10.15") >= 0
}
```

## Testing

### Unit Tests

1. **Sandbox Profile Loading**
   - Test loading default profile
   - Test loading custom profile
   - Test profile validation
   - Test invalid profile handling

2. **Resource Limits**
   - Test CPU limit setting
   - Test memory limit setting
   - Test file descriptor limits
   - Test limit validation

3. **Process Attributes**
   - Test attribute application
   - Test flag combinations
   - Test invalid attribute handling

4. **Execution**
   - Test simple command execution
   - Test command with arguments
   - Test environment injection
   - Test sandbox enforcement

### Integration Tests

1. **Sandbox Enforcement**
   - Test file access restrictions
   - Test network access restrictions
   - Test IPC restrictions

2. **Resource Limits**
   - Test CPU limit enforcement
   - Test memory limit enforcement
   - Test cleanup on limit violation

3. **Session Tracking**
   - Test session registration
   - Test status updates
   - Test heartbeat updates

## Security Considerations

### Sandbox Profile Security

1. **Path Rules**
   - Use absolute paths only
   - Validate path exists before adding
   - Avoid wildcards in path rules

2. **Network Rules**
   - Deny all by default
   - Explicitly allow required networks
   - Restrict port ranges

3. **Code Signing**
   - Sign sandbox profiles with valid certificate
   - Validate signature before loading
   - Revoke compromised certificates

### Resource Limit Security

1. **Limit Validation**
   - Enforce maximum limits
   - Prevent resource exhaustion
   - Validate limit values

2. **Escape Prevention**
   - Use `setuid`-free code
   - Avoid privilege escalation
   - Disable execve after setup

## Performance Considerations

1. **Sandbox Application**
   - Pre-load profiles for reuse
   - Cache compiled profiles
   - Avoid re-validation

2. **Resource Limits**
   - Use efficient syscalls
   - Batch limit applications
   - Minimize context switches

3. **Session Tracking**
   - Use async heartbeat updates
   - Batch registry updates
   - Optimize memory usage

## Known Limitations

1. **Sandbox Profile Complexity**
   - Complex profiles may have performance impact
   - Some operations may be incompatible with sandbox
   - Profile validation may miss edge cases

2. **Resource Limit Precision**
   - CPU limits are approximate
   - Memory limits may vary by activity
   - Network limits are best-effort

3. **Process Attribute Scope**
   - Some attributes require root privileges
   - Not all attributes are supported on all macOS versions
   - Process attributes may not persist after fork/exec

## Future Enhancements

1. **Advanced Sandbox Features**
   - Dynamic profile modification
   - Profile templates and inheritance
   - Sandbox-aware process spawning

2. **Resource Monitoring**
   - Real-time resource usage tracking
   - Alert-based limit enforcement
   - Historical usage analytics

3. **Integration with macOS APIs**
   - XPC service isolation
   - Launch daemon integration
   - App Store sandbox profiles

## Troubleshooting

### Common Issues

**Sandbox profile fails to load**
- Check profile syntax with `sandbox-exec -p`
- Verify code signing certificate
- Check macOS version compatibility

**Resource limits not enforced**
- Verify process permissions
- Check limit values are valid
- Monitor system resource availability

**Process exits unexpectedly**
- Check sandbox restrictions
- Review system logs for errors
- Verify resource limits

### Debug Mode

Enable debug logging:

```yaml
isolation:
  level: platform
  platform:
    name: macos-sandbox
    debug: true
    log_level: debug
    log_file: /tmp/aps-macos-debug.log
```

## References

- [Apple Sandbox Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/AppSandboxDesignGuide/)
- [macOS Sandbox Profile Syntax](https://developer.apple.com/library/archive/documentation/Security/Conceptual/AppSandboxDesignGuide/AboutAppSandbox/AboutAppSandbox.html)
- [Task Policy API](https://developer.apple.com/documentation/kernel/task_policy)
