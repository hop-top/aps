# Isolation Levels

Complete guide to configuring and using APS isolation levels for secure profile execution.

## What is Isolation?

Isolation in APS defines how securely commands and actions are executed within a profile. APS provides three tiers of isolation, ranging from simple environment separation to full containerized execution.

## Isolation Levels

### Process Isolation (Default)

**Security Level**: Low  
**Performance**: < 50ms overhead  
**Platform Support**: ✅ All (Linux, macOS, Windows)

Process isolation provides environment-only isolation using the same user process. This is the current baseline behavior and is always available.

**Features:**
- Environment variable injection
- Git config isolation
- SSH key management
- Profile-specific secrets
- Separate working directory

**Use Cases:**
- Development environments
- Quick command execution
- Resource-constrained systems
- Cross-platform compatibility required

**Configuration:**
```yaml
isolation:
  level: process
  strict: false
  fallback: true
```

### Platform Sandbox

**Security Level**: Medium  
**Performance**: 100-500ms overhead  
**Platform Support**:
- ✅ macOS: User account isolation, ACLs, launchctl management
- ✅ Linux: User namespaces, chroot, setfacl, cgroups
- 🚧 Windows: Coming soon (Restricted tokens, job objects)

Platform sandbox uses OS-native sandboxing features for enhanced security without full containerization.

**Features:**
- All process isolation features
- Filesystem restrictions
- Network isolation options
- User/group separation
- Resource limits (cgroups)

**Use Cases:**
- Production environments
- Security-sensitive operations
- Multi-tenant systems
- Running untrusted code

**Configuration:**
```yaml
isolation:
  level: platform
  strict: false
  fallback: true
  platform:
    name: "production-sandbox"
    sandbox_id: "aps-sandbox-001"
```

### Container Isolation

**Security Level**: High  
**Performance**: 1-5s overhead  
**Platform Support**:
- ✅ Linux: Docker/Podman (native)
- ✅ macOS: Docker Desktop or Colima (via VM)
- 🚧 Windows: Docker Desktop via WSL2 (coming soon)

Container isolation provides full containerization for maximum security and reproducibility.

**Features:**
- All platform sandbox features
- Complete filesystem isolation
- Network isolation
- Resource quotas (CPU, memory)
- Custom images and environments

**Use Cases:**
- Maximum security required
- Reproducible builds
- CI/CD pipelines
- Running untrusted code
- Multi-cloud deployments

**Configuration:**
```yaml
isolation:
  level: container
  strict: true
  fallback: false
  container:
    image: "ubuntu:22.04"
    network: "bridge"
    volumes:
      - "/host/path:/container/path"
    resources:
      memory_mb: 512
      cpu_quota: 1000
```

## Configuration

### Global Configuration

Configure default isolation in `~/.config/aps/config.yaml`:

```yaml
prefix: APS
isolation:
  default_level: process      # process | platform | container
  fallback_enabled: true       # Allow fallback to lower isolation levels
```

### Profile Configuration

Configure per-profile isolation in `~/.agents/profiles/<id>/profile.yaml`:

```yaml
id: myagent
display_name: "My AI Agent"

isolation:
  level: process              # Requested isolation level
  strict: false               # Fail if requested level is unavailable
  fallback: true              # Allow fallback to lower isolation levels
  
  # Platform-specific options (level: platform)
  platform:
    name: "sandbox-name"
    sandbox_id: "unique-id"
  
  # Container-specific options (level: container)
  container:
    image: "ubuntu:22.04"
    network: "bridge"
    volumes: []
    resources:
      memory_mb: 512
      cpu_quota: 1000
```

## Fallback Behavior

When the requested isolation level is unavailable, APS can gracefully degrade to a lower isolation level.

### Fallback Rules

1. **Container → Platform → Process**: If container is unavailable, try platform, then process
2. **Platform → Process**: If platform is unavailable, try process
3. **Process**: Always available (no fallback possible)

### Configuration

**Enable fallback (default):**
```yaml
isolation:
  level: container
  fallback: true
  strict: false
```

**Disable fallback:**
```yaml
isolation:
  level: container
  fallback: false
  strict: false
```

If fallback is disabled and requested level is unavailable, execution fails with an error.

**Strict mode:**
```yaml
isolation:
  level: container
  fallback: true
  strict: true
```

In strict mode, even if fallback is enabled, the system will fail if the exact requested level is unavailable.

### Global Fallback Control

Disable fallback globally in `~/.config/aps/config.yaml`:

```yaml
isolation:
  default_level: process
  fallback_enabled: false
```

## Security Considerations

### Process Isolation

- **Threats**: Commands run with your user permissions
- **Mitigations**: Profile separation, secrets isolation
- **When to Use**: Development, trusted code, cross-platform needs

### Platform Sandbox

- **Threats**: Escaped sandbox processes
- **Mitigations**: OS-level security, filesystem restrictions
- **When to Use**: Production, semi-trusted code, enhanced security

### Container Isolation

- **Threats**: Container breakout, resource exhaustion
- **Mitigations**: Resource quotas, network isolation, hardened images
- **When to Use**: Untrusted code, maximum security, reproducibility

## Performance Comparison

| Level | Setup Time | Overhead | CPU/Memory |
|-------|-----------|----------|------------|
| Process | < 10ms | Minimal | None |
| Platform | 100-500ms | Low | Low |
| Container | 1-5s | Medium | Configurable |

## Examples

### Development Profile

```bash
aps profile new dev-agent --display-name "Development Agent"

# Use default process isolation
cat > ~/.agents/profiles/dev-agent/profile.yaml << 'EOF'
id: dev-agent
display_name: "Development Agent"

isolation:
  level: process
  strict: false
  fallback: true

git:
  enabled: true
EOF
```

### Production Profile

```bash
aps profile new prod-agent --display-name "Production Agent"

# Use platform isolation with strict mode
cat > ~/.agents/profiles/prod-agent/profile.yaml << 'EOF'
id: prod-agent
display_name: "Production Agent"

isolation:
  level: platform
  strict: true
  fallback: false

platform:
  name: "production-sandbox"
  sandbox_id: "aps-prod-001"

limits:
  max_runtime_minutes: 60
  max_concurrency: 5
EOF
```

### Maximum Security Profile

```bash
aps profile new secure-agent --display-name "Secure Agent"

# Use container isolation with resource limits
cat > ~/.agents/profiles/secure-agent/profile.yaml << 'EOF'
id: secure-agent
display_name: "Secure Agent"

isolation:
  level: container
  strict: true
  fallback: false

container:
  image: "ubuntu:22.04"
  network: "none"
  resources:
    memory_mb: 1024
    cpu_quota: 2000

limits:
  max_runtime_minutes: 30
  max_concurrency: 2
EOF
```

## Troubleshooting

### "isolation level not supported"

**Cause**: Requested isolation level not available on your platform

**Solutions**:
1. Enable fallback: `fallback: true`
2. Use available level: `level: process`
3. Install required tools: Docker for container isolation

### "strict mode violation: requested isolation level not available"

**Cause**: Strict mode enabled and exact level unavailable

**Solutions**:
1. Disable strict mode: `strict: false`
2. Install required isolation tools
3. Use a different isolation level

### "no available isolation adapter after fallback"

**Cause**: All requested isolation levels unavailable

**Solutions**:
1. Ensure `level: process` is in fallback chain
2. Verify isolation tools are installed
3. Check global configuration: `fallback_enabled: true`

## Best Practices

1. **Development**: Use process isolation for speed
2. **Production**: Use platform or container isolation for security
3. **Testing**: Enable fallback for flexibility
4. **Critical Systems**: Use strict mode with appropriate level
5. **Resource Limits**: Combine isolation with profile limits
6. **Monitor Performance**: Adjust isolation based on usage patterns
7. **Document Choices**: Note isolation requirements in profile notes.md
8. **Test Fallback**: Verify fallback behavior works as expected

## Platform-Specific Notes

### Linux

- All three levels fully supported
- Container isolation uses Docker/Podman natively
- Platform sandbox uses user namespaces and cgroups

### macOS

- Process and container isolation supported
- Platform sandbox uses user account isolation and ACLs
- Container isolation requires Docker Desktop or Colima

### Windows

- Process isolation fully supported
- Platform sandbox: Coming soon
- Container isolation: Coming soon (via WSL2)

## Migrating from Default Behavior

Existing profiles without explicit isolation configuration will use:

- **Level**: `process` (default)
- **Fallback**: `true`
- **Strict**: `false`

To upgrade profiles:

1. Add isolation section to profile.yaml
2. Test with fallback enabled
3. Gradually move to higher isolation levels
4. Enable strict mode once stable

## Related Documentation

- [Profiles](PROFILES.md) - Profile management
- [Sessions](SESSIONS.md) - Session management
- [Security](SECURITY.md) - Security best practices
- [Examples](EXAMPLES.md) - Practical isolation examples
