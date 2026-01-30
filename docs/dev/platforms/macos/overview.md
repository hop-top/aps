# macOS Platform Isolation

## Overview

APS provides macOS platform isolation using user account sandboxing, giving each profile its own isolated user account with restricted filesystem access and process separation.

## Requirements

### System Requirements
- macOS 10.15 (Catalina) or later
- Administrator/sudo access
- Xcode Command Line Tools (for dscl, launchctl)

### OpenSSH Server
The macOS platform adapter requires OpenSSH Server for remote access to sandbox users.

To enable OpenSSH on macOS:
```bash
# Enable Remote Login
sudo systemsetup -setremotelogin on

# Or enable via System Preferences:
# System Preferences > Sharing > Remote Login: ON
```

### SSH Key Setup
Admin public key must be available at `~/.aps/keys/admin_pub`:
```bash
# Create keys directory
mkdir -p ~/.aps/keys

# Copy admin public key (if not already there)
cp ~/.ssh/id_rsa.pub ~/.aps/keys/admin_pub
```

## Architecture

### User Account Management

For each profile with platform isolation, APS creates a dedicated sandbox user:

```
Username:  aps-{profileID}
Home Dir:  /Users/aps-{profileID}
UID:       Auto-assigned (next available above 500)
```

The sandbox user is:
- Hidden from login window (`IsHidden=1`)
- Removed from `staff` group
- Has random password (not needed due to passwordless sudo)

### Shared Workspace

Both the host user and sandbox user have read/write access to a shared directory:

```
Path: /Users/Shared/aps-{HOST_USER}
Permissions: 0770 (rwxrwx---)
ACL: group:aps-{profileID} allow [full access]
```

### Passwordless Sudo

APS configures passwordless sudo for the host user to execute commands as the sandbox user:

```bash
# Sudoers file: /etc/sudoers.d/50-nopasswd-for-{username}
{HOST_USER} ALL=({sandbox_user}) NOPASSWD: ALL
```

### SSH Key Distribution

Admin's SSH public key is added to sandbox user's `~/.ssh/authorized_keys`:

```
Permissions:
~/.ssh/         0700
authorized_keys   0600
```

This allows passwordless SSH from host user to sandbox user.

## Usage

### Creating a Profile with macOS Isolation

```yaml
# ~/.agents/profiles/secure-agent/profile.yaml
id: secure-agent
display_name: Secure Agent

isolation:
  level: platform
  strict: false
  fallback: true
```

### Running Commands

```bash
# Execute command in macOS sandbox user
aps run secure-agent -- whoami

# Expected output: aps-secure-agent
```

### Executing Actions

```bash
# Run a defined action
aps action secure-agent run my-script

# The action runs as the sandbox user with:
# - Isolated home directory
# - Access to shared workspace
# - Passwordless sudo available
```

## Session Management

When executing commands, APS tracks sessions:

```bash
# List active sessions
aps session list

# Attach to a session (via SSH to sandbox user)
aps session attach {session_id}
```

## Security Features

### Process Isolation
- Each profile runs under its own user account
- Process tree is isolated from other profiles
- No signal interference between profiles

### Filesystem Isolation
- Sandbox user has limited filesystem access
- Only profile directory and shared workspace are writable
- System directories remain read-only

### ACL Enforcement
- Access Control Lists (ACLs) restrict shared workspace access
- Fine-grained permissions for multiple sandbox users
- No cross-profile data leakage

### Resource Tracking
- Session registry tracks all running processes
- Automatic cleanup on process termination
- PID-based resource monitoring

## Configuration

### Profile-Level Settings

```yaml
isolation:
  level: platform      # Enable macOS platform sandbox
  strict: false        # Don't fail if unavailable
  fallback: true       # Fall back to process isolation
```

### Global Settings

```yaml
# ~/.config/aps/config.yaml
prefix: APS

isolation:
  default_level: process    # Default isolation for all profiles
  fallback_enabled: true    # Allow graceful degradation
```

## Troubleshooting

### "dscl command not found"
Install Xcode Command Line Tools:
```bash
xcode-select --install
```

### "Permission denied" when creating user
Ensure you have sudo/admin privileges:
```bash
# Verify sudo access
sudo -v

# Run with sudo if needed
sudo aps run profile-id -- command
```

### Sandbox user already exists
APS detects existing sandbox users and reuses them. To recreate:
```bash
# Delete sandbox user
sudo dscl . -delete /Users/aps-{profileID}

# Delete home directory
sudo rm -rf /Users/aps-{profileID}

# Then run APS again (will recreate)
```

### SSH connection fails
1. Verify OpenSSH server is running:
   ```bash
   sudo launchctl list | grep ssh
   ```

2. Check admin public key exists:
   ```bash
   ls -la ~/.aps/keys/admin_pub
   ```

3. Verify authorized_keys in sandbox user home:
   ```bash
   sudo cat /Users/aps-{profileID}/.ssh/authorized_keys
   ```

4. Test SSH manually:
   ```bash
   ssh aps-{profileID}@localhost whoami
   ```

### ACL configuration fails
If ACL rules cannot be set:
```bash
# Verify filesystem supports ACLs
ls -lde /Users/Shared

# Manually set ACL if needed
sudo chmod -h +a "group:aps-{profileID} allow full" /Users/Shared/aps-{USER}
```

### Process cleanup not working
If sandbox processes don't terminate:
```bash
# List sandbox user processes
ps -u aps-{profileID}

# Kill manually if needed
sudo pkill -u aps-{profileID}
```

## Limitations

### iOS Simulator Access
iOS simulator access from sandbox users is **not currently supported**. This is a known limitation tracked for future implementation.

### Filesystem Access
Sandbox users have limited filesystem access:
- **Read-only**: System directories (/System, /Library, /usr, /bin, etc.)
- **Write access**: Profile home directory and shared workspace only
- **No access**: Other users' home directories, admin-only directories

### Network Access
Sandbox users inherit host user's network permissions. Network restrictions must be configured at the system level (firewall, parental controls, etc.).

## Known Issues

1. **First Run Slow**: Creating sandbox user takes ~2-5 seconds on first run
2. **Sudoers Validation**: Requires valid sudoers syntax; invalid entries are rejected
3. **ACL on APFS**: Some macOS volumes may have ACL limitations

## Best Practices

1. **Use Shared Workspace for Data Exchange**: Store shared files in `/Users/Shared/aps-{USER}`
2. **Profile-Specific Secrets**: Use `secrets.env` in each profile for isolated credentials
3. **Action Scripts**: Use shell scripts (`*.sh`) for complex workflows in sandbox
4. **Session Cleanup**: Always run `aps session list` and cleanup old sessions
5. **Backup Configuration**: Periodically backup `~/.agents` directory

## Migration from Process Isolation

To migrate existing profiles from process to platform isolation:

1. Update `profile.yaml`:
   ```yaml
   isolation:
     level: platform  # Changed from: process
   ```

2. Test the profile:
   ```bash
   aps run profile-id -- echo "test"
   ```

3. Verify user isolation:
   ```bash
   # Should show sandbox user, not host user
   aps run profile-id -- whoami
   ```

## API Reference

### DarwinSandbox Struct

```go
type DarwinSandbox struct {
    context        *ExecutionContext
    username       string           // sandbox user name
    password       string           // random password (internal)
    homeDir        string           // sandbox user home directory
    sharedDir      string           // shared workspace path
    tmuxSocket     string           // tmux socket for session
    tmuxSession    string           // tmux session ID
    useTmux        bool             // whether tmux is in use
    sessionPID     int              // process PID
    configured     bool             // whether sandbox is configured
    adminPublicKey []byte           // admin SSH public key
}
```

### Key Methods

- `PrepareContext(profileID)`: Initialize sandbox context for profile
- `SetupEnvironment(cmd)`: Inject environment variables for execution
- `Execute(command, args)`: Execute command as sandbox user
- `ExecuteAction(actionID, payload)`: Execute action as sandbox user
- `Cleanup()`: Cleanup resources and terminate processes
- `Validate()`: Verify sandbox configuration
- `IsAvailable()`: Check if macOS sandbox is available

## References

- [Isolation Architecture](../../specs/001-build-cli-core/isolation-architecture.md)
- [AGENTS.md](../../AGENTS.md)
- [macOS dscl Documentation](https://ss64.com/mac/dscl)
- [macOS chmod ACL Documentation](https://ss64.com/mac/chmod)
