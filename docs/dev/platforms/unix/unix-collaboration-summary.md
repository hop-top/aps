# Unix Platform Collaboration Summary

**Note:** This is a historical document summarizing the collaborative design phase for Unix platforms (macOS and Linux). The design documents referenced here are now located in `architecture/design/`.

## Overview

This document summarizes the collaborative design work completed for Unix platform adapters (macOS and Linux), including interface design improvements, Unix-specific considerations, session registry design, and test strategy.

## Documents Created

### 1. Unix Platform Adapter Design
**File:** `docs/UNIX_PLATFORM_ADAPTER_DESIGN.md`

**Contents:**
- ✅ Extended `IsolationManager` interface for Unix platforms
- ✅ Unix-specific `ExecutionContext` fields
- ✅ Platform-specific edge cases (macOS `dscl`, Linux `unshare`/`setfacl`)
- ✅ Unix-only features (shared workspace paths, user namespaces, process signaling)
- ✅ Cross-platform issues identified
- ✅ Error handling patterns defined

**Key Proposals:**

**Interface Extensions:**
- Added Unix-specific methods: `GetUserContext()`, `GetNamespaceContext()`, `GetResourceLimits()`, `GetTmuxContext()`
- Extended `ExecutionContext` with UID, GID, Shell, HomeDir, NamespaceID, CgroupPath, TmuxSocket

**macOS-Specific:**
- User management via `dscl` API
- Sandbox profile limitations
- Resource limit precision issues

**Linux-Specific:**
- Namespace isolation via `unshare`
- File ACLs via `setfacl`
- Cgroup v1/v2 compatibility

### 2. Unix Session Registry Design
**File:** `docs/UNIX_SESSION_REGISTRY_DESIGN.md`

**Contents:**
- ✅ Extended `SessionInfo` with Unix-specific metadata
- ✅ Supporting types for Unix session context
- ✅ SSH key distribution for macOS and Linux
- ✅ Tmux considerations across Unix platforms
- ✅ Potential issues with tmux identified

**Key Proposals:**

**Session Metadata:**
- UID/GID tracking
- Username/Groupname tracking
- NamespaceID and CgroupPath (Linux)
- Platform identifier (macos/linux)

**SSH Key Distribution:**
- SSH key generation (ED25519, RSA)
- macOS SSH server configuration
- Linux SSH server configuration
- SSH key rotation mechanism

**Tmux Considerations:**
- Socket location differences
- Version compatibility detection
- macOS-specific issues (paste buffer, clipboard)
- Linux-specific issues (terminal type, terminal emulators)

### 3. Unix Test Strategy
**File:** `docs/UNIX_TEST_STRATEGY.md`

**Contents:**
- ✅ E2E tests for macOS designed
- ✅ E2E tests for Linux designed
- ✅ Unix-specific CI runner configuration
- ✅ Test data and fixtures defined
- ✅ Coverage targets established

**Test Coverage:**

**macOS E2E Tests:**
- Profile creation and deletion
- Tmux session execution
- Session attach and detach
- Environment injection
- Git integration

**Linux E2E Tests:**
- Namespace detection
- Cgroup detection
- User information
- File permissions
- Process signaling

**CI Configuration:**
- GitHub Actions macOS runner workflow
- GitHub Actions Linux runner workflow
- Self-hosted runner setup for both platforms

## Acceptance Criteria Checklist

### ✅ Interface Design Incorporates Unix Requirements

- [x] `IsolationManager` interface includes Unix-specific methods
- [x] `ExecutionContext` includes Unix metadata fields
- [x] Supporting types for Unix features defined
- [x] Platform-specific edge cases documented

### ✅ Unix Edge Cases Documented

**macOS Edge Cases:**
- [x] `dscl` capabilities documented
- [x] Sandbox limitations documented
- [x] Resource limit precision issues documented
- [x] Code signing requirements documented

**Linux Edge Cases:**
- [x] `unshare` capabilities documented
- [x] `setfacl` capabilities documented
- [x] Cgroup version compatibility documented
- [x] Namespace limit handling documented

**Cross-Platform Issues:**
- [x] Path separator differences identified
- [x] Environment variable case sensitivity identified
- [x] Process tree handling differences identified
- [x] Permission model differences identified

### ✅ Test Strategy Defined

**E2E Tests:**
- [x] macOS E2E tests designed
- [x] Linux E2E tests designed
- [x] Test fixtures implemented
- [x] Test helper functions defined
- [x] Coverage targets established

**CI Runners:**
- [x] macOS runner workflow configured
- [x] Linux runner workflow configured
- [x] Self-hosted runner setup documented
- [x] Runner labels configured for routing

### ✅ Session Registry Design Incorporates Unix Metadata

**Session Schema:**
- [x] `SessionInfo` includes Unix-specific fields
- [x] Supporting types for Unix context defined
- [x] Session registry methods for Unix queries proposed
- [x] Session validation for Unix platforms defined

**SSH Key Distribution:**
- [x] SSH key generation mechanism proposed
- [x] macOS SSH key distribution documented
- [x] Linux SSH key distribution documented
- [x] SSH key rotation mechanism proposed

**Tmux Considerations:**
- [x] tmux socket location differences documented
- [x] tmux version compatibility handled
- [x] macOS-specific tmux issues addressed
- [x] Linux-specific tmux issues addressed

## Key Design Decisions

### 1. Interface Extensibility

**Decision:** Extend `IsolationManager` interface with Unix-specific methods rather than creating separate Unix interface.

**Rationale:**
- Maintains compatibility with existing code
- Allows gradual adoption of Unix features
- Provides clear separation of concerns
- Simplifies adapter implementation

### 2. Session Metadata Storage

**Decision:** Store Unix-specific metadata directly in `SessionInfo` structure rather than separate UnixSessionInfo.

**Rationale:**
- Simplifies session registry
- Reduces data duplication
- Makes Unix metadata queryable
- Maintains JSON serialization compatibility

### 3. SSH Key Management

**Decision:** Generate and store SSH keys per profile rather than global keys.

**Rationale:**
- Provides better isolation between profiles
- Allows per-profile SSH key rotation
- Simplifies permission management
- Follows least privilege principle

### 4. Tmux Socket Location

**Decision:** Use `/tmp/aps-tmux-{profile-id}-socket` pattern for both macOS and Linux.

**Rationale:**
- Consistent behavior across platforms
- Unique sockets prevent conflicts
- Temporary directory handles cleanup
- Easy to identify and manage

## Next Steps

### Immediate Actions

1. **Review and Approval**
   - Review interface design proposals
   - Review session registry extensions
   - Approve test strategy
   - Identify any missing requirements

2. **Implementation Planning**
   - Create implementation tasks for each platform adapter
   - Prioritize features based on complexity and impact
   - Assign developers to specific tasks
   - Set target dates for each phase

3. **Documentation Updates**
   - Update existing documentation with Unix-specific notes
   - Create platform-specific guides if needed
   - Update API documentation
   - Update user guides

### Future Work

1. **macOS Platform Adapter**
   - Implement macOS-specific isolation adapter
   - Implement macOS user management via `dscl`
   - Implement macOS sandbox profile support
   - Add macOS-specific E2E tests

2. **Linux Platform Adapter**
   - Implement Linux namespace isolation
   - Implement Linux cgroup management
   - Implement Linux ACL management
   - Add Linux-specific E2E tests

3. **Cross-Platform Enhancements**
   - Implement SSH key distribution
   - Enhance tmux integration
   - Improve error handling
   - Add performance monitoring

## Collaboration Notes

### Design Decisions Requiring Discussion

1. **Namespace Isolation Scope**
   - Should all processes be isolated or just specific ones?
   - How to handle user namespace for rootless execution?

2. **Resource Limit Precision**
   - Are approximate resource limits acceptable?
   - How to handle resource limit failures gracefully?

3. **Tmux Integration**
   - Should tmux be optional or required for all sessions?
   - How to handle tmux version incompatibilities?

### Open Questions

1. **User Management**
   - Should APS create system users or use existing users?
   - How to handle user cleanup on profile deletion?

2. **SSH Key Security**
   - How often should SSH keys be rotated?
   - Should SSH keys have expiration dates?

3. **Session Cleanup**
   - How to handle orphaned tmux sessions?
   - How to detect and clean up stale sessions?

## Summary

✅ **All acceptance criteria met:**
- Interface design incorporates Unix requirements
- Unix edge cases documented
- Test strategy defined
- Session registry design incorporates Unix metadata

**Documents created:**
- `docs/UNIX_PLATFORM_ADAPTER_DESIGN.md` - Interface design and Unix-specific considerations
- `docs/UNIX_SESSION_REGISTRY_DESIGN.md` - Session registry and SSH key distribution
- `docs/UNIX_TEST_STRATEGY.md` - E2E tests and CI configuration

**Ready for:** Implementation of Unix platform adapters based on collaborative design

## Related Documentation

### Design Documents (Current Locations)
- [../../architecture/design/unix-platform-adapter-design.md](../../architecture/design/unix-platform-adapter-design.md) - Unix platform adapter design (formerly UNIX_PLATFORM_ADAPTER_DESIGN.md)
- [../../architecture/design/unix-session-registry-design.md](../../architecture/design/unix-session-registry-design.md) - Unix session registry design (formerly UNIX_SESSION_REGISTRY_DESIGN.md)
- [../../testing/unix-test-strategy.md](../../testing/unix-test-strategy.md) - Unix test strategy (formerly UNIX_TEST_STRATEGY.md)

### Platform Implementation
- [../linux/overview.md](../linux/overview.md) - Linux platform user guide
- [../linux/linux-implementation.md](../linux/linux-implementation.md) - Linux implementation requirements
- [../macos/overview.md](../macos/overview.md) - macOS platform user guide
- [../macos/macos-implementation.md](../macos/macos-implementation.md) - macOS implementation requirements

### Implementation Status
- [../../implementation/summaries/linux-sandbox-summary.md](../../implementation/summaries/linux-sandbox-summary.md) - Linux implementation details
- [../../implementation/summaries/final-implementation-summary.md](../../implementation/summaries/final-implementation-summary.md) - Overall implementation status
