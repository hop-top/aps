# Windows Platform Adapter Implementation Requirements

## Overview

The Windows platform adapter provides process isolation using Windows-specific security mechanisms including Job Objects, Windows Security Levels, and AppContainer isolation.

## System Requirements

### Minimum Version
- Windows 8 or Windows Server 2012 (for basic Job Objects)
- Windows 10 or Windows Server 2016 (for AppContainer)

### Required Windows Features
- Job Objects API (always available)
- Windows Security Levels
- AppContainer isolation (Windows 8+)

### Optional Dependencies
- Hyper-V (for enhanced isolation)
- Windows Defender Application Control (WDAC)

## Architecture

### Components

```
windows_platform_adapter.go     # Main adapter implementation
job_objects.go                # Job Object management
security_levels.go            # Security level management
appcontainer.go              # AppContainer isolation
session_tracking.go           # Session integration
```

### Integration Points

- Registers with `isolation.Manager` as `IsolationPlatform`
- Uses `session.Registry` for process tracking
- Integrates with `core.Config` for isolation configuration

## Job Objects

### Job Object Overview

Job Objects provide process grouping and resource management on Windows.

### Job Object Configuration

```go
type JobObjectConfig struct {
    Name           string
    LimitFlags     uint32
    BasicUIRestrictions uint32
    SecurityFlags  uint32
    ActiveProcessLimit uint32
    PriorityClass  uint32
    SchedulingClass uint32
    IORestrictions *IORestrictions
    CPURate       *CPURateLimit
    MemoryLimit   *MemoryLimit
    NetworkLimits *NetworkLimits
}

type IORestrictions struct {
    ReadBytesPerSec   uint64
    WriteBytesPerSec  uint64
}

type CPURateLimit struct {
    CyclesPerPeriod   uint64
    PeriodLengthUS    uint32
}

type MemoryLimit struct {
    ProcessMemoryLimitMB uint64
    JobMemoryLimitMB    uint64
}

type NetworkLimits struct {
    MaxBandwidth   uint64
    Tag           uint32
}
```

### Job Object Implementation

```go
// Create job object
func (a *WindowsPlatformAdapter) CreateJobObject(config *JobObjectConfig) (windows.Handle, error)

// Add process to job
func (a *WindowsPlatformAdapter) AddProcessToJob(job windows.Handle, pid int) error

// Set job limits
func (a *WindowsPlatformAdapter) SetJobLimits(job windows.Handle, config *JobObjectConfig) error

// Query job information
func (a *WindowsPlatformAdapter) QueryJobInformation(job windows.Handle) (*JobObjectInformation, error)
```

### Job Object Limits

```go
const (
    JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x2000
    JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION = 0x400
    JOB_OBJECT_LIMIT_BREAKAWAY_OK = 0x800
    JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK = 0x1000
    JOB_OBJECT_LIMIT_PRIORITY_CLASS = 0x20
    JOB_OBJECT_LIMIT_PROCESS_MEMORY = 0x100
    JOB_OBJECT_LIMIT_JOB_MEMORY = 0x200
    JOB_OBJECT_LIMIT_PROCESS_TIME = 0x1
    JOB_OBJECT_LIMIT_JOB_TIME = 0x2
    JOB_OBJECT_LIMIT_ACTIVE_PROCESS = 0x8
    JOB_OBJECT_LIMIT_AFFINITY = 0x10
    JOB_OBJECT_LIMIT_SCHEDULING_CLASS = 0x4000
    JOB_OBJECT_LIMIT_IO_RATE_CONTROL = 0x8000
)
```

### Job Object Creation

```go
func (a *WindowsPlatformAdapter) CreateJobObject(config *JobObjectConfig) (windows.Handle, error) {
    jobName, err := windows.UTF16PtrFromString(config.Name)
    if err != nil {
        return 0, err
    }
    
    // Create job object
    job, err := windows.CreateJobObject(nil, jobName)
    if err != nil {
        return 0, fmt.Errorf("failed to create job object: %w", err)
    }
    
    // Set limit flags
    var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
    info.BasicLimitInformation.LimitFlags = config.LimitFlags
    
    // Set process memory limit
    if config.MemoryLimit != nil && config.MemoryLimit.ProcessMemoryLimitMB > 0 {
        info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY
        info.ProcessMemoryLimit = config.MemoryLimit.ProcessMemoryLimitMB * 1024 * 1024
    }
    
    // Set job memory limit
    if config.MemoryLimit != nil && config.MemoryLimit.JobMemoryLimitMB > 0 {
        info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
        info.JobMemoryLimit = config.MemoryLimit.JobMemoryLimitMB * 1024 * 1024
    }
    
    // Set active process limit
    if config.ActiveProcessLimit > 0 {
        info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
        info.ActiveProcessLimit = config.ActiveProcessLimit
    }
    
    // Set priority class
    if config.PriorityClass > 0 {
        info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_PRIORITY_CLASS
        info.BasicLimitInformation.PriorityClass = config.PriorityClass
    }
    
    // Apply limits
    _, err = windows.SetInformationJobObject(
        job,
        windows.JobObjectExtendedLimitInformation,
        uintptr(unsafe.Pointer(&info)),
        uint32(unsafe.Sizeof(info)),
    )
    
    if err != nil {
        windows.CloseHandle(job)
        return 0, fmt.Errorf("failed to set job limits: %w", err)
    }
    
    return job, nil
}
```

## Windows Security Levels

### Security Level Configuration

```go
type SecurityLevel string

const (
    SecurityLevelBasic     SecurityLevel = "basic"
    SecurityLevelStandard  SecurityLevel = "standard"
    SecurityLevelRestricted SecurityLevel = "restricted"
    SecurityLevelStrict    SecurityLevel = "strict"
)

type SecurityConfig struct {
    Level           SecurityLevel
    IntegrityLevel  IntegrityLevel
    Token           *TokenConfig
    Privileges      *PrivilegeConfig
}

type IntegrityLevel string

const (
    IntegrityLevelUntrusted   IntegrityLevel = "untrusted"
    IntegrityLevelLow        IntegrityLevel = "low"
    IntegrityLevelMedium     IntegrityLevel = "medium"
    IntegrityLevelMediumPlus  IntegrityLevel = "medium-plus"
    IntegrityLevelHigh       IntegrityLevel = "high"
    IntegrityLevelSystem     IntegrityLevel = "system"
)

type TokenConfig struct {
    MandatoryLabel IntegrityLevel
    Privileges    []string
    Groups        []string
    Restricted    bool
}

type PrivilegeConfig struct {
    Remove []string
    Add    []string
    Disable []string
}
```

### Security Level Definitions

```go
var SecurityLevels = map[SecurityLevel]SecurityConfig{
    SecurityLevelBasic: {
        Level: SecurityLevelBasic,
        IntegrityLevel: IntegrityLevelMedium,
        Token: &TokenConfig{
            MandatoryLabel: IntegrityLevelMedium,
            Restricted: false,
        },
    },
    SecurityLevelStandard: {
        Level: SecurityLevelStandard,
        IntegrityLevel: IntegrityLevelMedium,
        Token: &TokenConfig{
            MandatoryLabel: IntegrityLevelMedium,
            Privileges: []string{
                "SeChangeNotifyPrivilege",
                "SeAssignPrimaryTokenPrivilege",
            },
            Restricted: false,
        },
    },
    SecurityLevelRestricted: {
        Level: SecurityLevelRestricted,
        IntegrityLevel: IntegrityLevelLow,
        Token: &TokenConfig{
            MandatoryLabel: IntegrityLevelLow,
            Privileges: []string{
                "SeChangeNotifyPrivilege",
            },
            Privileges: &PrivilegeConfig{
                Remove: []string{
                    "SeDebugPrivilege",
                    "SeTakeOwnershipPrivilege",
                    "SeRestorePrivilege",
                },
            },
            Restricted: true,
        },
    },
    SecurityLevelStrict: {
        Level: SecurityLevelStrict,
        IntegrityLevel: IntegrityLevelLow,
        Token: &TokenConfig{
            MandatoryLabel: IntegrityLevelLow,
            Restricted: true,
            Privileges: []string{},
        },
        Privileges: &PrivilegeConfig{
            Disable: []string{
                "SeAssignPrimaryTokenPrivilege",
                "SeDebugPrivilege",
                "SeIncreaseQuotaPrivilege",
                "SeLoadDriverPrivilege",
                "SeTakeOwnershipPrivilege",
                "SeTcbPrivilege",
            },
        },
    },
}
```

### Security Level Implementation

```go
// Apply security level to process
func (a *WindowsPlatformAdapter) ApplySecurityLevel(process windows.Handle, level SecurityLevel) error {
    config, ok := SecurityLevels[level]
    if !ok {
        return fmt.Errorf("unknown security level: %s", level)
    }
    
    // Set integrity level
    if err := a.setIntegrityLevel(process, config.IntegrityLevel); err != nil {
        return fmt.Errorf("failed to set integrity level: %w", err)
    }
    
    // Configure token
    if err := a.configureToken(process, config.Token); err != nil {
        return fmt.Errorf("failed to configure token: %w", err)
    }
    
    // Configure privileges
    if config.Privileges != nil {
        if err := a.configurePrivileges(process, config.Privileges); err != nil {
            return fmt.Errorf("failed to configure privileges: %w", err)
        }
    }
    
    return nil
}
```

## AppContainer Isolation

### AppContainer Overview

AppContainers provide lightweight sandboxing for processes on Windows 8+.

### AppContainer Configuration

```go
type AppContainerConfig struct {
    Name        string
    DisplayName string
    Description string
    Capabilities []string
    EnableLowPrivilegeAppContainer bool
    CreateNoWriteUp bool
}

type AppContainerCapabilities struct {
    InternetClient      bool
    InternetClientServer bool
    PrivateNetworkClientServer bool
    PicturesLibrary     bool
    VideosLibrary       bool
    MusicLibrary        bool
    DocumentsLibrary    bool
}
```

### AppContainer Implementation

```go
// Create AppContainer
func (a *WindowsPlatformAdapter) CreateAppContainer(config *AppContainerConfig) (windows.Handle, error)

// Create process in AppContainer
func (a *WindowsPlatformAdapter) CreateProcessInAppContainer(
    appName string,
    cmd string,
    args []string,
) (*os.Process, error)

// Get AppContainer SID
func (a *WindowsPlatformAdapter) GetAppContainerSID(appName string) (string, error)
```

### AppContainer Creation

```go
func (a *WindowsPlatformAdapter) CreateAppContainer(config *AppContainerConfig) (windows.Handle, error) {
    // Convert config to Windows structs
    sid, err := a.createAppContainerSID(config.Name)
    if err != nil {
        return 0, err
    }
    
    // Create app container profile
    handle, err := windows.CreateAppContainerProfile(
        sid,
        config.DisplayName,
        config.Description,
        nil, // capabilities
        0,    // capability count
        config.EnableLowPrivilegeAppContainer,
    )
    
    if err != nil {
        return 0, fmt.Errorf("failed to create app container: %w", err)
    }
    
    return handle, nil
}
```

### Capabilities

```go
// Common capabilities
const (
    CapabilityInternetClient      = "CAP_INTERNET_CLIENT"
    CapabilityInternetClientServer = "CAP_INTERNET_CLIENT_SERVER"
    CapabilityPrivateNetwork      = "CAP_PRIVATE_NETWORK_CLIENT_SERVER"
    CapabilityPicturesLibrary     = "CAP_PICTURES_LIBRARY"
    CapabilityVideosLibrary       = "CAP_VIDEOS_LIBRARY"
    CapabilityMusicLibrary        = "CAP_MUSIC_LIBRARY"
    CapabilityDocumentsLibrary    = "CAP_DOCUMENTS_LIBRARY"
)

// Add capability to AppContainer
func (a *WindowsPlatformAdapter) AddCapability(appName string, capability string) error {
    sid, err := a.getAppContainerSID(appName)
    if err != nil {
        return err
    }
    
    capabilitySID, err := a.getCapabilitySID(capability)
    if err != nil {
        return err
    }
    
    return windows.AddAppContainerPackageCapability(sid, capabilitySID)
}
```

## Profile Configuration

```yaml
isolation:
  level: platform
  strict: false
  fallback: true
  platform:
    name: windows-job-object
    job_object:
      name: "aps-{{profile_id}}"
      kill_on_close: true
      die_on_exception: false
      breakaway_ok: false
    resource_limits:
      process_memory_mb: 1024
      job_memory_mb: 1024
      active_processes: 50
      io_read_bytes_per_sec: 10485760
      io_write_bytes_per_sec: 10485760
    security:
      level: standard
      token_restricted: false
    appcontainer:
      enabled: false
      name: "aps-app-{{profile_id}}"
      capabilities:
        - internet_client
        - private_network
      enable_low_privilege: true
```

## Session Integration

```go
func (a *WindowsPlatformAdapter) Execute(command string, args []string) error {
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
            "windows_job_object":      a.jobObjectName,
            "windows_security_level":  a.securityLevel,
            "windows_appcontainer_id": a.appContainerID,
        },
    }
    
    registry := session.GetRegistry()
    if err := registry.Register(session); err != nil {
        return err
    }
    
    defer registry.Unregister(sessionID)
    
    // Create job object
    job, err := a.createJobObject()
    if err != nil {
        return err
    }
    defer windows.CloseHandle(job)
    
    // Create process
    cmd := exec.Command(command, args...)
    if err := a.SetupEnvironment(cmd); err != nil {
        return err
    }
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    // Add process to job
    session.PID = cmd.Process.Pid
    if err := a.addProcessToJob(job, cmd.Process.Pid); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Apply security level
    if err := a.applySecurityLevel(cmd.Process.Pid, a.securityLevel); err != nil {
        cmd.Process.Kill()
        return err
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

## Implementation Details

### PrepareContext

```go
func (a *WindowsPlatformAdapter) PrepareContext(profileID string) (*isolation.ExecutionContext, error) {
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
    
    // Load Windows-specific configuration
    a.config = loadWindowsConfig(profile.Isolation.Platform)
    
    a.context = context
    return context, nil
}
```

### Execute

```go
func (a *WindowsPlatformAdapter) Execute(command string, args []string) error {
    // Create job object
    job, err := a.createJobObject()
    if err != nil {
        return err
    }
    defer windows.CloseHandle(job)
    
    // Create command
    cmd := exec.Command(command, args...)
    
    // Setup environment
    if err := a.SetupEnvironment(cmd); err != nil {
        return err
    }
    
    // Set job object information in process
    a.setJobObjectInProcess(cmd)
    
    // Start process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    // Add to job object
    if err := a.addProcessToJob(job, cmd.Process.Pid); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Apply security level
    if err := a.applySecurityLevel(cmd.Process.Pid, a.config.SecurityLevel); err != nil {
        cmd.Process.Kill()
        return err
    }
    
    // Wait for completion
    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("%w: %v", isolation.ErrExecutionFailed, err)
    }
    
    return nil
}
```

### IsAvailable

```go
func (a *WindowsPlatformAdapter) IsAvailable() bool {
    if runtime.GOOS != "windows" {
        return false
    }
    
    // Check Windows version
    if !isWindowsSupported() {
        return false
    }
    
    return true
}

func isWindowsSupported() bool {
    version := windows.RtlGetVersion()
    // Windows 8+ is required (version 6.2)
    return version.MajorVersion > 6 || (version.MajorVersion == 6 && version.MinorVersion >= 2)
}
```

## Testing

### Unit Tests

1. **Job Object Management**
   - Test job object creation
   - Test process addition to job
   - Test limit configuration
   - Test job object cleanup

2. **Security Levels**
   - Test security level application
   - Test integrity level setting
   - Test token configuration
   - Test privilege configuration

3. **AppContainer Isolation**
   - Test AppContainer creation
   - Test process creation in AppContainer
   - Test capability management
   - Test AppContainer cleanup

4. **Execution**
   - Test command execution with job object
   - Test command execution with security level
   - Test command execution in AppContainer
   - Test environment injection

### Integration Tests

1. **Job Object Enforcement**
   - Test memory limit enforcement
   - Test process limit enforcement
   - Test I/O limit enforcement
   - Test job termination

2. **Security Level Enforcement**
   - Test integrity level enforcement
   - Test privilege restrictions
   - Test access control

3. **AppContainer Isolation**
   - Test file system isolation
   - Test registry isolation
   - Test network isolation

## Security Considerations

### Job Object Security

1. **Limit Validation**
   - Validate all limit values
   - Enforce maximum limits
   - Prevent resource exhaustion

2. **Access Control**
   - Restrict job object access
   - Prevent job object hijacking
   - Validate process addition

### Security Level Security

1. **Token Security**
   - Validate integrity levels
   - Check privilege modifications
   - Prevent token impersonation

2. **Privilege Security**
   - Remove dangerous privileges
   - Validate privilege additions
   - Check privilege assignments

### AppContainer Security

1. **Capability Security**
   - Validate capability assignments
   - Check capability SID validity
   - Prevent capability escalation

2. **Isolation Security**
   - Verify file system isolation
   - Verify registry isolation
   - Verify network isolation

## Performance Considerations

1. **Job Object Overhead**
   - Minimize job object creation time
   - Reuse job objects where possible
   - Optimize limit application

2. **Security Level Overhead**
   - Cache security level configurations
   - Minimize token modifications
   - Optimize privilege checking

3. **AppContainer Overhead**
   - Pre-create AppContainers
   - Cache capability SIDs
   - Minimize profile creation

## Known Limitations

1. **AppContainer Requirements**
   - Requires Windows 8 or later
   - Some APIs not available in AppContainer
   - Limited file system access

2. **Job Object Limitations**
   - Cannot nest job objects
   - Breakaway behavior is complex
   - Some processes cannot be added to jobs

3. **Security Level Limitations**
   - Limited by user permissions
   - Some privileges cannot be removed
   - Integrity levels are advisory

## Future Enhancements

1. **Advanced Job Object Features**
   - Nested job objects (with breakaway)
   - Dynamic limit modification
   - Job object monitoring

2. **Enhanced AppContainer Support**
   - Multiple AppContainers per profile
   - Dynamic capability management
   - AppContainer resource limits

3. **Windows Defender Integration**
   - WDAC policy integration
   - Application allowlisting
   - Code signing enforcement

## Troubleshooting

### Common Issues

**Job object creation fails**
- Check Windows version
- Verify permissions
- Check for job object limits

**Security level application fails**
- Verify administrator privileges
- Check integrity level validity
- Review token configuration

**AppContainer creation fails**
- Check Windows version (8+)
- Verify AppContainer support
- Check capability SIDs

### Debug Mode

Enable debug logging:

```yaml
isolation:
  level: platform
  platform:
    name: windows-job-object
    debug: true
    log_level: debug
    log_file: C:\Temp\aps-windows-debug.log
```

## References

- [Job Objects](https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects)
- [Security Levels](https://docs.microsoft.com/en-us/windows/win32/secauthz/security-levels)
- [AppContainer](https://docs.microsoft.com/en-us/windows/win32/api/appcontainer/)
- [Windows Integrity Mechanism](https://docs.microsoft.com/en-us/windows/win32/secauthz/windows-integrity-mechanism)
