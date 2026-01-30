# Migration Guide: Process to Platform Isolation

**Version**: v0.2.x / v0.3.x
**Date**: 2026-01-21

## Overview

This guide helps you migrate from process isolation (Tier 1) to platform sandbox isolation (Tier 2) for better security.

## Why Migrate?

### Benefits of Platform Isolation

1. **User-Level Isolation**: Each profile runs as separate user
2. **Filesystem Separation**: Separate home directories
3. **Shared Workspaces**: Controlled access via ACLs
4. **SSH Access**: Remote access to sandbox user
5. **Resource Control**: Optional cgroups for resource limiting
6. **Multi-Tenant**: Multiple agents isolated from each other

### When to Migrate

✅ **Migrate to Platform if**:
- Running multiple AI assistants with different contexts
- Need separation between different development workflows
- Want better isolation than process but lower overhead than containers
- Need SSH access to sandboxed environments
- Managing multiple developers/teams

❌ **Stay on Process if**:
- Single profile for personal use
- Development only, no security concerns
- Maximum performance required (minimal overhead)
- Running only trusted code/tools

## Migration Steps

### 1. Verify Platform Support

**macOS**:
```bash
# Check if dscl is available
which dscl

# Check if setfacl is available (if needed)
which chmod  # macOS uses chmod +a
```

**Linux**:
```bash
# Check if useradd is available
which useradd

# Check if setfacl is available
which setfacl
```

### 2. Generate Admin SSH Key (Optional but Recommended)

```bash
# Create keys directory
mkdir -p ~/.aps/keys

# Generate admin key pair
ssh-keygen -t ed25519 -f ~/.aps/keys/admin_key -N ""

# Copy public key to admin_pub
cp ~/.aps/keys/admin_key.pub ~/.aps/keys/admin_pub

# Set permissions
chmod 700 ~/.aps/keys
chmod 600 ~/.aps/keys/admin_key
chmod 644 ~/.aps/keys/admin_pub
```

### 3. Update Profile Configuration

**Existing Profile (Process Isolation)**:
```yaml
# ~/.agents/profiles/my-profile/profile.yaml
id: my-profile
display_name: "My Profile"

# No isolation configuration (defaults to process)
```

**Updated Profile (Platform Isolation)**:
```yaml
# ~/.agents/profiles/my-profile/profile.yaml
id: my-profile
display_name: "My Profile"

isolation:
  level: "platform"
  strict: false
  fallback: true
  
  platform:
    name: "Platform Sandbox"
    sandbox_id: "aps-my-profile"
```

### 4. Test Platform Isolation

```bash
# Test with a simple command
aps run my-profile -- whoami

# Expected output:
# macOS: aps-my-profile
# Linux: aps-sandbox-my-profile

# Test with a more complex command
aps run my-profile -- ls -la ~/.aps

# Test with interactive command
aps run my-profile -- bash

# Test session management
aps session list

# Test SSH connection (if SSH is set up)
ssh aps-my-profile@localhost whoami
```

### 5. Verify File Access

```bash
# Create a file in your home directory
echo "test content" > ~/test-file.txt

# Access from sandbox (shared workspace)
# macOS: /Users/Shared/aps-$USER
# Linux: /tmp/aps-shared/$USER
aps run my-profile -- cat /Users/Shared/aps-$USER/test-file.txt  # macOS
aps run my-profile -- cat /tmp/aps-shared/$USER/test-file.txt     # Linux

# Verify file exists in sandbox
aps run my-profile -- ls -la
```

### 6. Clean Up Process Isolation Artifacts

Process isolation doesn't leave persistent artifacts, so no cleanup is needed.

### 7. Update Automation/Scripts

If you have automation or scripts that assume process isolation, update them:

**Before (Process Isolation)**:
```bash
aps run my-profile -- command

# Commands run as same user
# File access is same as host user
```

**After (Platform Isolation)**:
```bash
aps run my-profile -- command

# Commands run as sandbox user
# File access in shared workspace
# SSH access to sandbox user available
```

## Common Migration Issues

### Issue 1: Command Fails After Migration

**Symptom**: `aps run profile -- command` fails with "permission denied"

**Solution**:
1. Check if sandbox user was created:
   ```bash
   # macOS
   dscl . -list /Users | grep aps-
   
   # Linux
   id aps-sandbox-profile
   ```

2. Check if shared workspace exists:
   ```bash
   # macOS
   ls -la /Users/Shared/aps-$USER
   
   # Linux
   ls -la /tmp/aps-shared/$USER
   ```

3. Verify profile isolation level:
   ```bash
   cat ~/.agents/profiles/profile/profile.yaml | grep "level:"
   ```

### Issue 2: SSH Connection Fails

**Symptom**: Cannot SSH to sandbox user

**Solution**:
1. Verify admin key exists:
   ```bash
   ls -la ~/.aps/keys/admin_key
   ```

2. Verify admin key in authorized_keys:
   ```bash
   # macOS
   sudo cat /Users/aps-profile/.ssh/authorized_keys
   
   # Linux
   sudo cat /home/aps-sandbox-profile/.ssh/authorized_keys
   ```

3. Check SSH server is running (if using platform sandbox):
   ```bash
   # macOS
   # SSH server runs as sandbox user, check if user session is active
   ps aux | grep sshd | grep aps-
   
   # Linux
   sudo systemctl status sshd
   ```

### Issue 3: File Access Issues in Shared Workspace

**Symptom**: Cannot access files in shared workspace

**Solution**:
1. Check ACL permissions:
   ```bash
   # macOS
   ls -le /Users/Shared/aps-$USER
   
   # Linux
   getfacl /tmp/aps-shared/$USER
   ```

2. Check ownership:
   ```bash
   # macOS
   ls -ld /Users/Shared/aps-$USER
   
   # Linux
   ls -ld /tmp/aps-shared/$USER
   ```

3. Fix permissions if needed:
   ```bash
   # macOS
   sudo chmod 0770 /Users/Shared/aps-$USER
   sudo chmod -a "+a:aps-profile allow:rwX" /Users/Shared/aps-$USER
   
   # Linux
   sudo chmod 0770 /tmp/aps-shared/$USER
   sudo setfacl -m "u:aps-sandbox-profile:rwX" /tmp/aps-shared/$USER
   ```

### Issue 4: Performance Degradation

**Symptom**: Commands run slower after migration

**Solution**:
1. **First time setup**: Slow (150-300ms) is expected for initial user creation
2. **Subsequent runs**: Fast (< 50ms) after first run
3. **Profile-specific caching**: Each profile has its own sandbox user (one-time cost)

If performance issues persist:
1. Check if fallback to process isolation is happening:
   ```bash
   aps run my-profile -- whoami
   # If output is your username, it fell back to process
   ```

2. Check system resources:
   ```bash
   # macOS
   top -l 1

   # Linux
   top -bn1
   ```

## Rollback Procedure

If you need to rollback to process isolation:

1. Update profile configuration:
   ```yaml
   # ~/.agents/profiles/my-profile/profile.yaml
   id: my-profile
   display_name: "My Profile"

   # Remove isolation section or set level to process
   isolation:
     level: "process"
   ```

2. Restart APS (if needed):
   ```bash
   # No restart needed, changes take effect immediately
   ```

3. Verify process isolation is active:
   ```bash
   aps run my-profile -- whoami
   # Expected: your username
   ```

## Advanced Migration

### Migrating Multiple Profiles

```bash
# List all profiles
aps profile list

# Migrate each profile
for profile in $(aps profile list); do
  echo "Migrating $profile..."
  
  # Update profile.yaml
  profile_file=~/.agents/profiles/$profile/profile.yaml
  
  # Add isolation configuration
  echo "isolation:" >> $profile_file
  echo "  level: \"platform\"" >> $profile_file
  echo "  strict: false" >> $profile_file
  echo "  fallback: true" >> $profile_file
  
  # Test the migration
  aps run $profile -- whoami
done
```

### Using Profiles with Different Isolation Levels

You can use different isolation levels for different profiles:

```yaml
# Profile 1: Process isolation (development)
isolation:
  level: "process"

# Profile 2: Platform isolation (multi-tenant)
isolation:
  level: "platform"

# Profile 3: Container isolation (untrusted code)
isolation:
  level: "container"
  container:
    image: "ubuntu:22.04"
```

## Tips for Smooth Migration

1. **Test one profile at a time**: Migrate one profile, verify it works, then migrate the rest
2. **Use fallback**: Keep fallback enabled so APS can fall back to process isolation if issues arise
3. **Document your profiles**: Keep notes on which profiles use which isolation level
4. **Backup profiles**: Back up your profile directories before migration:
   ```bash
   cp -r ~/.agents/profiles ~/.agents/profiles.backup
   ```
5. **Gradual adoption**: Start new profiles with platform isolation, migrate existing profiles later

## Migration Checklist

- [ ] Verify platform support (dscl/useradd, setfacl)
- [ ] Generate admin SSH key (optional but recommended)
- [ ] Update profile configuration for platform isolation
- [ ] Test command execution
- [ ] Verify file access in shared workspace
- [ ] Test SSH connection (if configured)
- [ ] Test session management
- [ ] Update automation/scripts
- [ ] Verify performance is acceptable
- [ ] Rollback plan documented (if needed)

## Support

For migration issues:
1. Check troubleshooting section in platform documentation
   - macOS: `docs/dev/platforms/macos/overview.md`
   - Linux: `docs/dev/platforms/linux/overview.md`
2. Review logs in `~/.aps/logs/`
3. Open an issue on GitHub

## Conclusion

Migrating to platform isolation provides better security and user-level isolation with minimal performance overhead. The migration is straightforward and can be done one profile at a time.

**Recommendation**: Start with one profile, verify it works, then migrate the rest gradually.

**Date**: 2026-01-21
