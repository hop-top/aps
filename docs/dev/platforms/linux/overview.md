# Linux Platform Isolation

**Platform**: Linux
**Isolation Level**: Platform Sandbox (Tier 2)
**Status**: Supported

## Overview

Linux platform isolation provides user account isolation using:
- User namespaces via `unshare`
- Chroot filesystem isolation
- ACL configuration via `setfacl`
- Cgroups for resource limiting (optional)
- SSH access to sandbox user

## Requirements

### System Requirements
- Linux kernel 3.10+ (for user namespaces)
- sudo access with passwordless sudo configured
- OpenSSH server installed and running

### Required Tools
- `unshare` - User namespace creation
- `useradd` - User account management
- `setfacl` - ACL configuration
- `sudo` - Privilege escalation
- `tmux` - Terminal multiplexing (optional, for session management)

### Install Required Tools

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y util-linux acl sudo tmux

# RHEL/CentOS
sudo yum install -y util-linux acl sudo tmux

# Arch Linux
sudo pacman -S util-linux acl sudo tmux
```

### OpenSSH Server Setup

```bash
# Install OpenSSH server (if not already installed)
sudo apt-get install -y openssh-server  # Ubuntu/Debian
sudo yum install -y openssh-server       # RHEL/CentOS

# Ensure SSH service is running
sudo systemctl enable sshd
sudo systemctl start sshd

# Verify SSH server is listening
sudo netstat -tlnp | grep :22
```

## Setup

### 1. Admin SSH Key Generation

Generate admin SSH key for sandbox access:

```bash
# Create keys directory
mkdir -p ~/.aps/keys

# Generate admin key pair
ssh-keygen -t ed25519 -f ~/.aps/keys/admin_key -N ""

# Copy public key to admin_pub file
cp ~/.aps/keys/admin_key.pub ~/.aps/keys/admin_pub

# Set permissions
chmod 700 ~/.aps/keys
chmod 600 ~/.aps/keys/admin_key
chmod 644 ~/.aps/keys/admin_pub
```

### 2. Sudo Configuration

Configure passwordless sudo for the current user:

```bash
# Create sudoers file for APS
echo "$USER ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/99-aps-user

# Validate sudoers configuration
sudo visudo -c /etc/sudoers.d/99-aps-user
```

### 3. Create Profile with Linux Isolation

```bash
# Create new profile with platform isolation
aps profile new linux-sandbox --isolation-level platform

# Or create manually
aps profile new linux-sandbox
# Edit: ~/.local/share/aps/profiles/linux-sandbox/profile.yaml
# Add:
#   isolation:
#     level: "platform"
```

### Profile Configuration Example

```yaml
id: linux-sandbox
display_name: "Linux Sandbox Profile"

isolation:
  level: "platform"
  strict: false
  fallback: true

  platform:
    name: "Linux Sandbox"
    sandbox_id: "aps-linux-sandbox"
```

## Usage

### Running Commands in Sandbox

```bash
# Run command in sandbox
aps run linux-sandbox -- whoami

# Expected output: aps-sandbox-linux-sandbox

# Run command with args
aps run linux-sandbox -- ls -la /tmp

# Run interactive command
aps run linux-sandbox -- bash
```

### SSH Access to Sandbox

Connect to sandbox user via SSH:

```bash
# SSH to sandbox user
ssh aps-sandbox-linux-sandbox@localhost

# Or using admin key
ssh -i ~/.aps/keys/admin_key aps-sandbox-linux-sandbox@localhost
```

### Session Management

```bash
# List active sessions
aps session list

# Attach to session
aps session attach <session-id>

# Delete session
aps session delete <session-id>
```

## Architecture

### User Account Isolation

- **Sandbox User**: `aps-sandbox-{profileID}`
- **Home Directory**: `/home/aps-sandbox-{profileID}`
- **Shared Workspace**: `/tmp/aps-shared/{username}`

### Filesystem Layout

```
/tmp/aps-shared/                    # Shared workspace
├── {username}/                     # User's shared directory
│   ├── profile1/                   # Profile data
│   └── profile2/

/home/aps-sandbox-{profileID}/       # Sandbox user home
├── .ssh/
│   └── authorized_keys             # Admin SSH public key
└── .bashrc

/etc/sudoers.d/
└── 50-nopasswd-for-{username}     # Passwordless sudo config
```

### Access Control Lists (ACLs)

The shared workspace (`/tmp/aps-shared/{username}`) is configured with ACLs:

- **Owner**: Host user (read/write/execute)
- **Group**: Sandbox group (read/write/execute)
- **Other**: No access

ACL rules:
```
# Owner: {username}:rwX
# Group: aps-sandbox-{profileID}:rwX
# Default: No access
```

### Resource Limits (Optional)

Cgroups can be configured for resource limiting:

```yaml
isolation:
  level: "platform"
  platform:
    limits:
      memory_mb: 512
      cpu_quota: 100000
      cpu_period: 100000
```

**Note**: Cgroups support is optional and depends on system configuration.

## Security

### Isolation Boundaries

1. **User Account**: Separate Linux user account
2. **Filesystem**: Chroot environment (optional)
3. **Network**: Same network namespace as host (can be configured with `unshare`)
4. **Process**: Separate process tree via user namespaces

### Security Considerations

- **Sudo Access**: Host user can sudo to sandbox user
- **Shared Workspace**: Both host and sandbox user have read/write access
- **SSH Keys**: Admin public key copied to sandbox user's `~/.ssh/authorized_keys`
- **Passwordless Sudo**: Configured for host → sandbox user only

### Recommended Use Cases

✅ **Suitable for**:
- Multi-tenant agent environments
- Partial trust scenarios
- Code isolation with shared workspace
- Development and testing environments

❌ **Not suitable for**:
- Untrusted code execution
- High-security requirements
- Production workload isolation

## Troubleshooting

### Issue: "unshare not available"

**Problem**: `unshare` command not found

**Solution**:
```bash
# Install util-linux
sudo apt-get install -y util-linux  # Ubuntu/Debian
sudo yum install -y util-linux      # RHEL/CentOS
```

### Issue: "setfacl not available"

**Problem**: `setfacl` command not found

**Solution**:
```bash
# Install ACL utilities
sudo apt-get install -y acl  # Ubuntu/Debian
sudo yum install -y acl       # RHEL/CentOS
```

### Issue: "failed to create user"

**Problem**: User creation failed

**Solution**:
```bash
# Check if user already exists
id aps-sandbox-{profileID}

# Manually remove user if needed
sudo userdel -r aps-sandbox-{profileID}

# Retry profile creation
aps run linux-sandbox -- whoami
```

### Issue: "failed to set ACL"

**Problem**: ACL configuration failed

**Solution**:
```bash
# Check filesystem supports ACLs
tune2fs -l /dev/sda1 | grep "Filesystem features"

# Remount with ACL support if needed
sudo mount -o remount,acl /tmp

# Verify ACL support
getfacl /tmp/aps-shared
```

### Issue: SSH connection fails

**Problem**: Cannot SSH to sandbox user

**Solution**:
```bash
# Check SSH server is running
sudo systemctl status sshd

# Check if admin key is in authorized_keys
sudo cat /home/aps-sandbox-{profileID}/.ssh/authorized_keys

# Check SSH server config
sudo cat /etc/ssh/sshd_config | grep AllowUsers

# Restart SSH server
sudo systemctl restart sshd
```

### Issue: "permission denied" on shared workspace

**Problem**: Cannot access shared workspace directory

**Solution**:
```bash
# Check directory permissions
ls -la /tmp/aps-shared/

# Check ACL settings
getfacl /tmp/aps-shared/{username}/

# Fix permissions manually
sudo chown -R {username}:aps-sandbox-{profileID} /tmp/aps-shared/{username}/
sudo chmod -R 0770 /tmp/aps-shared/{username}/
sudo setfacl -R -m u:aps-sandbox-{profileID}:rwX /tmp/aps-shared/{username}/
```

### Issue: "sudo: no tty present"

**Problem**: Sudo requires TTY for password input

**Solution**:
```bash
# Ensure passwordless sudo is configured
sudo visudo -c /etc/sudoers.d/99-aps-user

# Use -n flag to avoid password prompt
sudo -n true
```

## Cleanup

### Remove Sandbox User

```bash
# Remove user and home directory
sudo userdel -r aps-sandbox-{profileID}

# Remove sudoers entry
sudo rm /etc/sudoers.d/50-nopasswd-for-{profileID}
```

### Remove Profile

```bash
# Delete profile
aps profile delete linux-sandbox

# Or manually
rm -rf ~/.local/share/aps/profiles/linux-sandbox
```

### Remove Shared Workspace

```bash
# Remove shared workspace directory
sudo rm -rf /tmp/aps-shared/{username}
```

## Advanced Configuration

### User Namespace Isolation

For stronger isolation, enable user namespaces:

```bash
# Check kernel support
cat /proc/sys/user/max_user_namespaces

# Enable if not already set
sudo sysctl -w user.max_user_namespaces=10000

# Persist across reboots
echo "user.max_user_namespaces=10000" | sudo tee -a /etc/sysctl.conf
```

### Chroot Environment

Enable chroot for filesystem isolation:

```bash
# Create chroot directory
sudo mkdir -p /tmp/aps-chroot-{profileID}/{bin,lib,etc,home,tmp,dev}

# Mount necessary filesystems
sudo mount --bind /bin /tmp/aps-chroot-{profileID}/bin
sudo mount --bind /lib /tmp/aps-chroot-{profileID}/lib
sudo mount --bind /etc /tmp/aps-chroot-{profileID}/etc
sudo mount --bind /dev /tmp/aps-chroot-{profileID}/dev
sudo mount -t proc proc /tmp/aps-chroot-{profileID}/proc
```

**Note**: Chroot requires additional setup and is disabled by default.

### Cgroups Resource Limiting

Configure Cgroups for resource limits:

```bash
# Create cgroup
sudo mkdir -p /sys/fs/cgroup/aps-{profileID}

# Set memory limit (512MB)
echo 536870912 | sudo tee /sys/fs/cgroup/aps-{profileID}/memory.limit_in_bytes

# Set CPU quota (100% of 1 CPU)
echo 100000 | sudo tee /sys/fs/cgroup/aps-{profileID}/cpu.cfs_quota_us
echo 100000 | sudo tee /sys/fs/cgroup/aps-{profileID}/cpu.cfs_period_us

# Add process to cgroup
echo <PID> | sudo tee /sys/fs/cgroup/aps-{profileID}/cgroup.procs
```

## Kernel Version Requirements

| Feature | Minimum Kernel |
|---------|---------------|
| User Namespaces | 3.8 |
| Set ACLs | 2.6 |
| Cgroups v1 | 2.6.24 |
| Cgroups v2 | 4.5 |

Check kernel version:
```bash
uname -r
```

## References

- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [unshare(1)](https://man7.org/linux/man-pages/man1/unshare.1.html)
- [setfacl(1)](https://man7.org/linux/man-pages/man1/setfacl.1.html)
- [cgroups(7)](https://man7.org/linux/man-pages/man7/cgroups.7.html)
- [OpenSSH Server](https://www.openssh.com/manual.html)

## Support

For issues or questions:
- Check troubleshooting section above
- Review logs in `~/.aps/logs/`
- Open an issue on GitHub
