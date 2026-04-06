# Windows Platform Isolation

**Platform**: Windows
**Isolation Level**: Platform Sandbox (Tier 2)
**Status**: Planned

## Overview

Windows platform isolation provides process isolation using Windows-specific security mechanisms:
- Job Objects for process grouping and resource limits
- Windows Security Levels for privilege restriction
- AppContainer isolation for sandboxing
- Optional Hyper-V for enhanced isolation

## Requirements

### System Requirements
- Windows 8 or Windows Server 2012 (minimum, for Job Objects)
- Windows 10 or Windows Server 2016 (recommended, for AppContainer)
- Administrator access for initial setup

### Required Windows Features
- Job Objects API (always available on Windows 8+)
- Windows Security Levels
- AppContainer isolation (Windows 8+)

### Optional Dependencies
- Hyper-V (for enhanced isolation via lightweight VMs)
- Windows Defender Application Control (WDAC) for additional security
- Windows Subsystem for Linux (WSL2) as an alternative isolation method

## Setup

### 1. Enable Required Windows Features

```powershell
# Check Windows version
winver

# Enable Hyper-V (optional, requires restart)
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All

# Enable Windows Defender Application Control (optional)
# Requires Group Policy configuration
```

### 2. Configure Job Objects

Job Objects are automatically available on Windows 8+ and require no additional setup. The APS Windows adapter will create and manage Job Objects for each isolated session.

### 3. AppContainer Setup

AppContainer isolation is available on Windows 8+ and provides:
- Network isolation
- Filesystem isolation
- Registry isolation
- Object namespace isolation

No manual setup required - the adapter handles AppContainer creation and management.

### 4. Security Configuration

```powershell
# Verify administrator access
net user %USERNAME% | findstr "Administrators"

# Configure firewall rules (if needed for SSH or remote access)
netsh advfirewall firewall add rule name="APS Session Access" dir=in action=allow protocol=TCP localport=22
```

## How It Works

### Process Isolation Flow

1. **Session Creation**: APS creates a new Job Object for the session
2. **Security Context**: Applies Windows Security Levels and optionally AppContainer
3. **Resource Limits**: Configures CPU, memory, I/O limits via Job Object
4. **Process Launch**: Spawns isolated process within the Job Object
5. **Monitoring**: Tracks process lifecycle via Job Object notifications

### Job Objects

Job Objects provide:
- **Process Grouping**: All child processes automatically included
- **Resource Limits**: CPU rate, memory caps, I/O throttling
- **UI Restrictions**: Limited desktop interaction
- **Termination**: Kill all processes in job atomically

### AppContainer Isolation

AppContainer provides:
- **Capability-Based Security**: Limited access to system resources
- **Network Isolation**: Separate network namespace
- **Filesystem Isolation**: Restricted file access
- **Registry Isolation**: Limited registry access

## Resource Limits

Configure resource limits in your APS configuration:

```yaml
isolation:
  platform: windows
  resource_limits:
    cpu_rate: 50          # 50% CPU cap
    memory_mb: 512        # 512 MB memory limit
    io_read_bps: 10485760 # 10 MB/s read limit
    io_write_bps: 5242880 # 5 MB/s write limit
```

## Usage

### Starting an Isolated Session

```bash
# Using APS CLI
aps create --platform windows --profile myprofile

# The Windows adapter will:
# 1. Create a Job Object
# 2. Apply configured limits
# 3. Launch process in isolated context
# 4. Return session ID
```

### Monitoring Sessions

```bash
# List active sessions
aps list

# Get session details
aps inspect <session-id>

# View Job Object statistics
aps stats <session-id>
```

### Cleanup

```bash
# Terminate session (kills all processes in Job Object)
aps terminate <session-id>

# The adapter automatically:
# - Closes Job Object handle
# - Cleans up AppContainer resources
# - Removes session from registry
```

## Security Considerations

### Job Object Security
- Job Objects provide good process isolation but share the same user context
- Processes in a job can still access user-level resources
- Not suitable for untrusted code without additional sandboxing

### AppContainer Security
- AppContainer provides strong isolation comparable to mobile app sandboxing
- Network access requires explicit capability grants
- Filesystem access limited to designated folders
- Recommended for running untrusted code

### Best Practices
1. Use AppContainer for untrusted workloads
2. Configure minimal required capabilities
3. Apply strict resource limits via Job Objects
4. Monitor for suspicious activity via Windows Event Log
5. Consider Hyper-V isolation for maximum security

## Troubleshooting

### Common Issues

**Job Object creation fails**
```
Error: Access denied creating Job Object
Solution: Run APS with administrator privileges
```

**AppContainer creation fails**
```
Error: AppContainer not supported
Solution: Verify Windows 8+ and ensure feature is enabled
```

**Process cannot access network**
```
Error: Network access denied in AppContainer
Solution: Grant internetClient capability in configuration
```

### Debugging

```powershell
# View active Job Objects
Get-Process | Select-Object Name, Id, @{Name="JobObject";Expression={$_.JobObject}}

# Check AppContainer status
Get-AppxPackage | Where-Object {$_.Name -like "*APS*"}

# Review Windows Event Log
Get-EventLog -LogName Application -Source APS -Newest 50
```

## Related Documentation

- [windows-implementation.md](windows-implementation.md) - Detailed implementation requirements
- [../../architecture/interfaces/adapter-interface-compliance.md](../../architecture/interfaces/adapter-interface-compliance.md) - Platform adapter interface
- [../../requirements/platform-adapter-merge-criteria.md](../../requirements/platform-adapter-merge-criteria.md) - Merge criteria and phases
- [../../testing/unix-test-strategy.md](../../testing/unix-test-strategy.md) - Testing approach (adapt for Windows)
- [../../security/security-audit.md](../../security/security-audit.md) - Security considerations

## Status

The Windows platform adapter is currently in **Phase 3** (planned). Implementation priority after Linux and macOS platforms are stable.

See [../../operations/releases/release-notes.md](../../operations/releases/release-notes.md) for current status and roadmap.
