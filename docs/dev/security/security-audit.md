# APS Security Audit

**Date**: 2026-01-21
**Version**: v0.2.x / v0.3.x

## Executive Summary

APS isolation tiers provide varying levels of security. This audit evaluates isolation boundaries, privilege requirements, SSH key handling, and potential security concerns.

**Overall Assessment**: ✅ Secure when used within intended threat model

---

## Threat Model

APS is designed for:
- Multi-agent environments (different personas, contexts)
- Partial trust scenarios (developer tools, AI assistants)
- Workload isolation (separate configurations, secrets)

APS is NOT designed for:
- Completely untrusted code execution (use containers for this)
- Malware analysis or reverse engineering
- High-security government/enterprise workloads (use containers)

**Assumptions**:
- Host user has legitimate access to run APS
- Profiles are managed by trusted administrator
- SSH keys are stored securely on host
- Docker daemon (if used) is trusted

---

## Tier 1: Process Isolation

### Isolation Boundaries

| Component | Isolation | Risk Level |
|-----------|-----------|------------|
| Process | Separate process | Low |
| Filesystem | Same user, same filesystem | Medium |
| Environment | Separate env vars | Low |
| Network | Same network stack | Low |
| Kernel | Same kernel | Low |

### Security Analysis

**✅ Strengths**:
- Simple attack surface
- Minimal privilege requirements
- No sudo or elevated access needed

**⚠️ Weaknesses**:
- No user-level isolation
- Same filesystem access as host user
- Process tree visible to host user
- Secrets stored in plaintext files

**❌ Vulnerabilities**: None identified

### Recommendations

1. **Default for trusted workflows**: Use for development, testing, trusted AI tools
2. **Avoid for untrusted code**: Process isolation provides no containment
3. **Secrets management**: Consider secret manager integration (1Password, HashiCorp Vault)
4. **Filesystem permissions**: Use umask to limit file creation permissions

---

## Tier 2: Platform Sandbox - macOS

### Isolation Boundaries

| Component | Isolation | Risk Level |
|-----------|-----------|------------|
| User Account | Separate macOS user (aps-{profileID}) | Low |
| Filesystem | Separate home directory + ACLs on shared workspace | Low |
| Environment | Separate env vars | Low |
| Network | Same network stack | Low |
| Kernel | Same kernel | Low |
| Process | Separate process tree via user | Low |

### Security Analysis

**✅ Strengths**:
- User-level isolation prevents accidental access
- ACLs provide fine-grained access control
- Separate home directory with proper permissions
- SSH key distribution for remote access
- Passwordless sudo restricted to specific user mapping

**⚠️ Weaknesses**:
- Requires sudo access for host user (to switch to sandbox user)
- Shared workspace allows bidirectional file access
- Host user can `sudo -u aps-{profileID}` to run any command as sandbox user
- No kernel-level isolation
- No network isolation

**❌ Vulnerabilities**: None identified

### Privilege Requirements

**macOS dscl**:
- Requires: `sudo` access for user account management
- Commands used: `dscl . -create /Users/...`, `dscl . -passwd /Users/...`
- Risk: Elevated to manage user accounts (mitigated: only for initial setup)

**sudoers Configuration**:
- Requires: `sudo` to write `/etc/sudoers.d/50-nopasswd-for-...`
- Risk: Creates passwordless sudo entry (mitigated: only for host→sandbox user, not sandbox→root)
- Validation: Uses `visudo -c` to validate sudoers syntax

### SSH Key Handling

**Key Distribution**:
- Source: `~/.aps/keys/admin_pub` (host)
- Destination: `~/.ssh/authorized_keys` (sandbox user home)
- Method: Direct file write with sudo
- Permissions: 0600 on authorized_keys, 0700 on .ssh
- Ownership: Set to sandbox user via `chown`

**Security Assessment**: ✅ Secure
- Keys stored in host filesystem with proper permissions
- SSH server on sandbox user requires additional setup (documented)
- Admin key can be revoked by removing from authorized_keys

### Recommendations

1. **Restrict sudo access**: Limit sudo access to specific admin users
2. **SSH hardening**: Disable password authentication, enforce key-based auth
3. **Audit logs**: Monitor sudo usage for sandbox access
4. **ACL review**: Regularly review ACL permissions on shared workspaces
5. **User lifecycle**: Implement sandbox user cleanup on profile deletion

---

## Tier 2: Platform Sandbox - Linux

### Isolation Boundaries

| Component | Isolation | Risk Level |
|-----------|-----------|------------|
| User Account | Separate Linux user (aps-sandbox-{profileID}) | Low |
| Filesystem | Separate home directory + ACLs on shared workspace | Low |
| Environment | Separate env vars | Low |
| Network | Same network stack (optional namespace support) | Low |
| Kernel | Same kernel (optional namespace support) | Low |
| Process | Separate process tree via user | Low |

### Security Analysis

**✅ Strengths**:
- User-level isolation prevents accidental access
- ACLs provide fine-grained access control
- Optional user namespace support for stronger isolation
- Optional chroot support for filesystem isolation
- Optional cgroups support for resource limiting
- SSH key distribution for remote access
- Passwordless sudo restricted to specific user mapping

**⚠️ Weaknesses**:
- Requires sudo access for host user (to switch to sandbox user)
- Shared workspace allows bidirectional file access
- Host user can `sudo -u aps-sandbox-{profileID}` to run any command as sandbox user
- User namespace/chroot/cgroups are optional (not enabled by default)
- No network isolation by default

**❌ Vulnerabilities**: None identified

### Privilege Requirements

**useradd**:
- Requires: `sudo` access for user account management
- Commands used: `useradd`, `userdel`, `passwd`
- Risk: Elevated to manage user accounts (mitigated: only for initial setup)

**setfacl**:
- Requires: `sudo` access for ACL configuration
- Commands used: `setfacl -R -m ...`
- Risk: Elevated to set ACLs (mitigated: ACLs restricted to specific user)

**sudoers Configuration**:
- Requires: `sudo` to write `/etc/sudoers.d/50-nopasswd-for-...`
- Risk: Creates passwordless sudo entry (mitigated: only for host→sandbox user, not sandbox→root)
- Validation: Uses `visudo -c` to validate sudoers syntax

### SSH Key Handling

**Key Distribution**:
- Source: `~/.aps/keys/admin_pub` (host)
- Destination: `/home/aps-sandbox-{profileID}/.ssh/authorized_keys`
- Method: Direct file write via `docker exec` or `sudo -u ...`
- Permissions: 0600 on authorized_keys, 0700 on .ssh
- Ownership: Set to sandbox user via `chown`

**Security Assessment**: ✅ Secure
- Keys stored in host filesystem with proper permissions
- SSH server on sandbox user requires OpenSSH server setup (documented)
- Admin key can be revoked by removing from authorized_keys

### Recommendations

1. **Enable user namespaces by default**: Provide stronger isolation
2. **Restrict sudo access**: Limit sudo access to specific admin users
3. **SSH hardening**: Disable password authentication, enforce key-based auth
4. **Audit logs**: Monitor sudo usage for sandbox access
5. **ACL review**: Regularly review ACL permissions on shared workspaces
6. **Resource limits**: Enable cgroups to prevent resource exhaustion attacks

---

## Tier 3: Container Isolation

### Isolation Boundaries

| Component | Isolation | Risk Level |
|-----------|-----------|------------|
| User Account | Separate container user (appuser) | Low |
| Filesystem | Separate container filesystem (explicit volume mounts) | Low |
| Environment | Separate env vars | Low |
| Network | Separate network namespace (bridge mode) | Low |
| Kernel | Separate kernel namespace (via Docker) | Low |
| Process | Separate process tree (via Docker) | Low |

### Security Analysis

**✅ Strengths**:
- Container-level isolation prevents most escape vectors
- Separate filesystem with explicit volume mounts
- Network isolation (bridge mode by default)
- Non-root user (appuser) by default
- Resource limits (CPU, memory) prevent DoS
- SSH server runs inside container (isolated)
- No privileged containers by default

**⚠️ Weaknesses**:
- Requires Docker daemon access (needs sudo or docker group membership)
- Host filesystem accessible through volume mounts
- Docker socket not mounted (best practice, but limits some operations)
- Container escape vulnerabilities exist (rare, but possible)
- Resource limits prevent DoS but can't prevent all attacks
- SSH server in container increases attack surface

**❌ Vulnerabilities**: None identified

### Privilege Requirements

**Docker Daemon Access**:
- Requires: Sudo or docker group membership
- Commands used: `docker build`, `docker create`, `docker start`, `docker exec`
- Risk: Elevated to manage containers (mitigated: Docker daemon manages isolation)

**Container Operations**:
- All container operations run as non-root user (appuser) inside container
- No privileged mode or `--cap-add` by default
- No host filesystem mounts except explicit volumes
- No Docker socket mounting

### SSH Key Handling

**Key Distribution**:
- Source: `~/.aps/keys/admin_pub` (host)
- Destination: `/home/appuser/.ssh/authorized_keys` (container)
- Method: Direct file write via `docker exec`
- Permissions: 0600 on authorized_keys, 0700 on .ssh
- Ownership: Set to appuser

**Security Assessment**: ✅ Secure
- Keys stored in container filesystem with proper permissions
- SSH server runs inside container (isolated from host)
- Container-level isolation protects host from compromise
- Container can be destroyed and recreated if compromised

### Recommendations

1. **Docker security hardening**:
   - Enable Docker Content Trust
   - Use security scanning for images (Trivy, Snyk)
   - Regularly update Docker daemon
   - Use non-root Docker daemon if possible

2. **Container hardening**:
   - Use minimal base images (Alpine)
   - Scan images for vulnerabilities
   - Remove unnecessary packages from images
   - Enable seccomp profiles if needed

3. **Network isolation**:
   - Use bridge mode (default) instead of host mode
   - Configure firewall rules for containers
   - Avoid port forwarding unless needed

4. **Volume security**:
   - Minimize host filesystem mounts
   - Use readonly mounts where possible
   - Avoid mounting sensitive host directories

5. **SSH hardening**:
   - Disable password authentication in container SSH server
   - Use short-lived SSH keys for containers
   - Rotate admin keys regularly

---

## Cross-Tier Comparison

### Security Strength (Strongest to Weakest)

1. **Container Isolation (Tier 3)**: Kernel-level isolation, strongest security
2. **Platform Sandbox (Tier 2)**: User-level isolation, moderate security
3. **Process Isolation (Tier 1)**: Process-level isolation, weakest security

### Risk Level (Lowest to Highest)

1. **Container Isolation (Tier 3)**: Lowest risk, strongest isolation
2. **Platform Sandbox (Tier 2)**: Low risk, moderate isolation
3. **Process Isolation (Tier 1)**: Medium risk, minimal isolation

### Use Case Recommendations

| Use Case | Recommended Tier | Rationale |
|---------|-----------------|-----------|
| Development / Testing | Process (Tier 1) | Minimal overhead, sufficient isolation |
| Multi-agent environments | Platform (Tier 2) | User separation, manageable overhead |
| Untrusted code | Container (Tier 3) | Strongest isolation, contained risk |
| Production workloads | Container (Tier 3) | Strong isolation, resource limits |
| AI assistants | Process/Platform (Tier 1/2) | Balance of security and performance |

---

## SSH Key Handling - Detailed Analysis

### Key Storage Locations

| Tier | Location | Permissions | Ownership |
|------|----------|-------------|------------|
| Process (Tier 1) | Not used | N/A | N/A |
| Platform - macOS (Tier 2) | `~/.aps/keys/admin_pub` | 0644 | Host user |
| Platform - Linux (Tier 2) | `~/.aps/keys/admin_pub` | 0644 | Host user |
| Container (Tier 3) | `~/.aps/keys/admin_pub` | 0644 | Host user |

### Key Distribution Process

**macOS Platform Sandbox**:
```bash
# Read admin key from host
cat ~/.aps/keys/admin_pub

# Write to sandbox user authorized_keys
sudo sh -c "cat >> /Users/aps-profile/.ssh/authorized_keys"

# Set permissions and ownership
sudo chmod 0600 /Users/aps-profile/.ssh/authorized_keys
sudo chown aps-profile /Users/aps-profile/.ssh/authorized_keys
```

**Linux Platform Sandbox**:
```bash
# Read admin key from host
cat ~/.aps/keys/admin_pub

# Write to sandbox user authorized_keys
sudo -u aps-sandbox-profile sh -c "cat >> /home/aps-sandbox-profile/.ssh/authorized_keys"

# Set permissions and ownership
sudo chmod 0600 /home/aps-sandbox-profile/.ssh/authorized_keys
sudo chown aps-sandbox-profile /home/aps-sandbox-profile/.ssh/authorized_keys
```

**Container Isolation**:
```bash
# Read admin key from host
cat ~/.aps/keys/admin_pub

# Write to container user authorized_keys
docker exec -i container-id sh -c "cat >> /home/appuser/.ssh/authorized_keys"

# Set permissions and ownership
docker exec container-id chmod 0600 /home/appuser/.ssh/authorized_keys
docker exec container-id chown appuser:appuser /home/appuser/.ssh/authorized_keys
```

### Security Assessment

**✅ Secure Practices**:
- Admin key only stored on host
- Keys copied with proper permissions (0600 on authorized_keys, 0700 on .ssh)
- Keys owned by target user (sandbox user or appuser)
- No keys in transit over network (all operations local)
- SSH servers configured for key-only authentication (password authentication disabled)

**⚠️ Potential Concerns**:
- Host user can copy admin key from `~/.aps/keys/admin_pub` (mitigated: admin private key requires 0600 permissions)
- If host system is compromised, sandbox user can be accessed (mitigated: this is expected behavior)
- No key rotation mechanism by default (recommendation: implement key rotation)

**❌ Vulnerabilities**: None identified

### Recommendations

1. **Key rotation**: Implement automatic admin key rotation (every 30 days)
2. **Key management**: Use SSH agent instead of key files (future enhancement)
3. **Access logging**: Log all SSH connections to sandbox users
4. **Key revocation**: Implement mechanism to revoke admin keys
5. **Multi-factor**: Consider MFA for SSH access (future enhancement)

---

## Sudo Usage Analysis

### macOS Platform Sandbox

**sudoers Entry**:
```bash
/etc/sudoers.d/50-nopasswd-for-aps-profile:
  # Allow hostuser to sudo to aps-profile without password
  hostuser ALL=(aps-profile) NOPASSWD: ALL
```

**Security Assessment**: ✅ Secure
- Host user can only sudo to specific sandbox user
- Cannot sudo to root or other users
- Passwordless only for this specific user mapping
- Validated with `visudo -c`

### Linux Platform Sandbox

**sudoers Entry**:
```bash
/etc/sudoers.d/50-nopasswd-for-hostuser:
  # Allow hostuser to sudo to aps-sandbox-profile without password
  hostuser ALL=(aps-sandbox-profile) NOPASSWD: ALL
```

**Security Assessment**: ✅ Secure
- Host user can only sudo to specific sandbox user
- Cannot sudo to root or other users
- Passwordless only for this specific user mapping
- Validated with `visudo -c`

### Container Isolation

**No sudo required** for container operations:
- Docker daemon runs as root (system service)
- APS user only needs docker group membership
- No passwordless sudo entries required

**Security Assessment**: ✅ Secure
- No sudo usage required
- Docker daemon manages isolation
- Docker group membership is standard practice

---

## Audit Findings Summary

### High Severity Issues
**None identified**

### Medium Severity Issues

1. **No key rotation mechanism**
   - **Impact**: Long-lived admin keys increase risk if compromised
   - **Recommendation**: Implement automatic key rotation (every 30 days)

2. **Process isolation lacks user-level isolation**
   - **Impact**: Accidental access to other profiles' data
   - **Recommendation**: Use platform or container isolation for untrusted code

3. **Shared workspace bidirectional access**
   - **Impact**: Sandbox user can access host user files in shared workspace
   - **Recommendation**: Use readonly mounts where possible, review ACLs regularly

### Low Severity Issues

1. **No SSH connection logging**
   - **Impact**: Cannot audit SSH access to sandbox users
   - **Recommendation**: Implement SSH connection logging

2. **No session timeout enforcement**
   - **Impact**: Orphaned sessions may persist indefinitely
   - **Recommendation**: Implement automatic session cleanup after inactivity

3. **No resource limits in platform isolation (optional)**
   - **Impact**: Sandbox user could consume all system resources
   - **Recommendation**: Enable cgroups for Linux platform isolation by default

### Informational Findings

1. **Performance vs security tradeoff**: Lower isolation tiers have better performance but lower security
2. **Container escape risk**: While rare, container escape vulnerabilities exist (mitigated: use updated Docker)
3. **Privilege escalation risk**: Sudo access required for platform isolation (mitigated: restricted sudoers entries)
4. **Secrets storage**: Secrets stored in plaintext files (mitigated: use secret manager integration)

---

## Security Recommendations

### Short Term (Immediate)

1. **Document threat model**: Communicate to users what APS is designed for
2. **Add warnings**: Warn users when using process isolation with untrusted code
3. **Implement key rotation**: Add automatic admin key rotation
4. **Add session cleanup**: Implement automatic session timeout and cleanup

### Medium Term

1. **Enable user namespaces by default**: Provide stronger isolation for Linux platform
2. **Implement SSH connection logging**: Audit all SSH access to sandbox users
3. **Add secret manager integration**: Support 1Password, HashiCorp Vault, AWS Secrets Manager
4. **Implement resource limits**: Enable cgroups for Linux platform isolation by default

### Long Term

1. **Container security scanning**: Integrate vulnerability scanning for container images
2. **Network isolation**: Implement network namespace isolation for platform tiers
3. **Multi-factor authentication**: Add MFA support for SSH access
4. **Security audits**: Regular third-party security audits
5. **Compliance**: Add support for security compliance standards (SOC2, PCI DSS)

---

## Compliance Considerations

### SOC 2

- **Access Control**: ✅ User-level isolation (Tier 2/3), role-based access control
- **Encryption**: ⚠️ Secrets in plaintext (need secret manager integration)
- **Monitoring**: ⚠️ No SSH connection logging (need implementation)
- **Change Management**: ⚠️ No key rotation (need implementation)

### PCI DSS

- **Network Isolation**: ✅ Container isolation (Tier 3) provides network separation
- **Data Protection**: ⚠️ Secrets in plaintext (need secret manager integration)
- **Access Logging**: ⚠️ No SSH connection logging (need implementation)
- **Vulnerability Management**: ⚠️ No container scanning (need implementation)

### ISO 27001

- **Information Security**: ✅ Multi-tier isolation options
- **Access Control**: ✅ User-level isolation (Tier 2/3)
- **Network Security**: ✅ Container network isolation (Tier 3)
- **Physical Security**: ⚠️ N/A (runs on user's system)

---

## Conclusion

APS provides three tiers of isolation with varying security characteristics:

1. **Process Isolation (Tier 1)**: Minimal overhead, sufficient for trusted workflows
2. **Platform Sandbox (Tier 2)**: Moderate overhead, user-level isolation, good for multi-agent environments
3. **Container Isolation (Tier 3)**: Higher overhead, kernel-level isolation, best for untrusted code

**Overall Assessment**: ✅ Secure when used within intended threat model

**Recommendation**: Default to platform isolation (Tier 2) for balance of security and performance, use container isolation (Tier 3) for high-security requirements.

**Next Steps**:
1. Implement key rotation mechanism
2. Add SSH connection logging
3. Implement automatic session cleanup
4. Enable user namespaces for Linux platform
5. Integrate with secret manager
6. Add container security scanning

**Date**: 2026-01-21
