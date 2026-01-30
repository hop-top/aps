# APS Release Notes

## v0.2.x - macOS Platform Isolation (Released 2026-01-21)

### New Features
- **macOS Platform Sandbox (Tier 2)**
  - User account isolation via `dscl`
  - ACL configuration via `chmod +a`
  - Passwordless sudo setup
  - SSH key distribution for remote access
  - Session management with tmux integration
  - Graceful fallback to process isolation

- **Session Management**
  - List active sessions with filtering
  - Attach to sessions
  - Detach from sessions
  - Inspect session details
  - Capture session logs
  - Terminate sessions gracefully
  - Delete sessions

### Performance
- Process isolation: < 5ms overhead
- Platform sandbox: 150-300ms setup (first time), < 50ms (subsequent)
- Tmux overhead: < 10ms

### Security
- User-level isolation prevents accidental access
- Fine-grained ACLs for shared workspaces
- Restricted sudoers entries (host→sandbox only)
- SSH key management with proper permissions

### Documentation
- macOS platform setup guide
- Migration guide from process isolation
- Security audit
- Performance benchmarks

---

## v0.3.x - Linux Platform Isolation (Released 2026-01-21)

### New Features
- **Linux Platform Sandbox (Tier 2)**
  - User account isolation via `useradd`
  - ACL configuration via `setfacl`
  - Passwordless sudo setup
  - SSH key distribution for remote access
  - Session management with tmux integration
  - Optional user namespace support
  - Optional chroot filesystem isolation
  - Optional cgroups resource limiting

- **Cross-Platform Support**
  - Platform sandbox support on macOS and Linux
  - Consistent API across platforms
  - Feature parity between platforms

### Performance
- Process isolation: < 5ms overhead
- Platform sandbox: 200-400ms setup (first time), < 50ms (subsequent)
- Tmux overhead: < 10ms

### Security
- User-level isolation prevents accidental access
- Fine-grained ACLs for shared workspaces
- Restricted sudoers entries (host→sandbox only)
- SSH key management with proper permissions
- Optional kernel-level isolation (namespaces, cgroups)

### Documentation
- Linux platform setup guide
- Cross-platform testing results
- Troubleshooting guide

---

## v0.4.x - Container Isolation (Released 2026-01-21)

### New Features
- **Container Isolation (Tier 3)**
  - Docker-based container isolation
  - Automatic Dockerfile generation from profiles
  - Image building and caching
  - Container lifecycle management (create, start, stop, remove)
  - Volume mounting for host-container file sharing
  - Network configuration (bridge, host, none)
  - Resource limits (CPU, memory)
  - SSH server in containers for remote access
  - tmux in containers for session management

- **Docker CLI Integration**
  - Docker engine implementation using Docker CLI
  - No SDK dependency required
  - Supports all ContainerEngine interface methods

- **Enhanced Session Management**
  - Container session metadata (container ID, image, status)
  - SSH-based session attachment for containers
  - Container log streaming

### Performance
- Process isolation: < 5ms overhead
- Platform sandbox: < 500ms setup
- Container isolation: 2-5s cold start, 100-500ms warm start

### Security
- Kernel-level isolation (via Docker)
- Separate filesystem for each container
- Network isolation (bridge mode by default)
- Non-root user (appuser) in containers
- Resource limits prevent DoS
- No privileged containers by default

### Documentation
- Container isolation setup guide
- Docker requirements and installation
- Profile configuration examples
- Container troubleshooting
- SSH server setup in containers

---

## v0.5.x - Capability Management (Released 2026-01-30)

### New Features
- **Capability Management System**
  - Unified way to package and install external tools
  - Directory-based capabilities stored in `~/.aps/capabilities`
  - Lifecycle commands: `install`, `link`, `watch`, `adopt`, `delete`
  - Support for "managed" (owned) and "reference" (linked) capabilities

- **Smart Linking**
  - Automatic detection of common AI Agent tools (e.g., `copilot`, `windsurf`, `cursor`)
  - Auto-resolves correct configuration paths relative to workspace root
  - Simplifies setup: `aps capability link windsurf` just works

- **Environment Integration**
  - `aps env` command to generate shell exports
  - Dynamic `APS_<NAME>_PATH` variables pointing to installed tools
  - Shell integration via `eval $(aps env)` (zsh/bash compatible)

- **Configuration Enhancements**
  - Support for multiple `capability_sources` in `config.yaml`
  - Integration with existing XDG configuration patterns

### CLI Commands
- `aps capability install <source>`
- `aps capability list`
- `aps capability link <name>`
- `aps capability watch <path>`
- `aps capability adopt <path>`
- `aps env`

### Documentation
- Capability Management Design Guide
- Capability Management Requirements
- Updated README with environment setup instructions

---

## Breaking Changes

None. All releases maintain backward compatibility.

---

## Upgrading

### From v0.1.x to v0.2.x (macOS Platform Isolation)
1. No action required - process isolation remains default
2. Optionally migrate to platform isolation:
   ```bash
   # Update profile.yaml
   isolation:
     level: "platform"
   ```
3. Generate admin SSH key (optional):
   ```bash
   mkdir -p ~/.aps/keys
   ssh-keygen -t ed25519 -f ~/.aps/keys/admin_key -N ""
   ```

### From v0.2.x to v0.3.x (Linux Platform Isolation)
1. No action required for macOS users
2. Linux users can optionally enable platform isolation
3. Follow Linux platform setup guide

### From v0.3.x to v0.4.x (Container Isolation)
1. No action required - process/platform isolation remain default
2. Optionally migrate to container isolation:
   ```bash
   # Update profile.yaml
   isolation:
     level: "container"
     container:
       image: "ubuntu:22.04"
   ```
3. Install Docker if not available
4. Follow container isolation setup guide

---

## Contributors

- @jadb - Architecture and implementation

---

## Known Issues

### macOS
- Passwordless sudo may require manual approval on first run
- ACLs may not work on network filesystems

### Linux
- User namespace support requires kernel 3.8+
- Chroot requires manual setup if enabled

### Containers
- Docker Desktop required on macOS and Windows
- Container overhead (2-5s cold start)
- Windows support requires additional testing

---

## Security Notes

- Process isolation: Minimal isolation, suitable for trusted code
- Platform isolation: User-level isolation, suitable for multi-agent environments
- Container isolation: Kernel-level isolation, suitable for untrusted code
- SSH keys: Stored in plaintext (use secret manager integration)
- Sudo access: Required for platform isolation (restricted sudoers entries)

See `docs/dev/security/security-audit.md` for detailed security analysis.

---

## Performance Notes

See `docs/dev/testing/performance-benchmarks.md` for detailed performance benchmarks.

**Summary**:
- Process isolation: Minimal overhead, fastest
- Platform isolation: Moderate overhead, good balance
- Container isolation: Higher overhead, strongest isolation

---

## Next Steps

See `docs/dev/implementation/summaries/final-implementation-summary.md` for roadmap.

**Upcoming Features**:
- Container security scanning
- Secret manager integration
- Enhanced resource limiting
- Multi-factor SSH authentication

---

## Support

For issues, questions, or contributions:
- Documentation: https://github.com/oss-aps/cli-integration/docs
- Issues: https://github.com/oss-aps/cli-integration/issues
- Discussions: https://github.com/oss-aps/cli-integration/discussions

---

**Date**: 2026-01-21
