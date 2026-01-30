# Platform Adapter Merge Criteria

This document defines the merge criteria for platform adapter PRs across different implementation phases.

## Overview

Platform adapters are implemented across multiple phases to ensure incremental delivery and risk management. Each phase has specific merge requirements that must be satisfied before code can be merged into the main branch.

## Phase Definitions

### Phase 2: Foundational Infrastructure
**Goal**: Establish core interfaces and shared infrastructure for all platform adapters

**Deliverables**:
- `IsolationManager` interface definition
- `ExecutionContext` structure
- `Manager` registry for adapter registration
- Error types and handling patterns
- Base testing utilities

### Phase 3: Platform-Specific Adapters (Linux, macOS, Windows)
**Goal**: Implement platform-specific isolation mechanisms for each major platform

**Deliverables**:
- Linux platform adapter
- macOS platform adapter
- Windows platform adapter

### Phase 4: Container Isolation
**Goal**: Implement Docker-based container isolation

**Deliverables**:
- Container adapter implementation
- Docker integration
- Volume mounting and networking configuration

---

## Phase 2 Merge Criteria

### Must-Have Requirements

1. **Interface Definition Complete**
   - [ ] `IsolationManager` interface defined with all 7 methods
   - [ ] `ExecutionContext` struct with all required fields
   - [ ] `Manager` registry with `Register()` and `Get()` methods
   - [ ] Fallback logic implemented in `GetIsolationManager()`

2. **Error Handling**
   - [ ] All error types defined (`ErrIsolationNotSupported`, `ErrInvalidProfile`, etc.)
   - [ ] Error wrapping conventions established
   - [ ] Error messages are descriptive

3. **Reference Implementation**
   - [ ] `ProcessIsolation` adapter implemented (baseline reference)
   - [ ] All methods implemented correctly
   - [ ] Code passes `go fmt` and `go vet`

4. **Testing Infrastructure**
   - [ ] Unit test framework in place
   - [ ] Test helper functions for common test patterns
   - [ ] Mock implementations for testing adapters

5. **Documentation**
   - [ ] Interface contract documented
   - [ ] Method specifications complete
   - [ ] Example usage provided

### Code Quality Gates

- [ ] All tests pass: `go test ./tests/unit/core/isolation/...`
- [ ] No linting errors: `golangci-lint run`
- [ ] No compilation errors
- [ ] Code coverage >= 70% for isolation package

### Merge Blockers

- Failing tests
- Missing interface methods
- Breaking changes to existing APIs
- Incomplete error handling

### Pre-Merge Checklist

- [ ] CI/CD pipeline passes on all platforms (Linux, macOS, Windows)
- [ ] At least one maintainer approval
- [ ] Documentation updated
- [ ] No merge conflicts

---

## Phase 3 Merge Criteria

### Platform-Specific Requirements

#### Linux Platform Adapter

**Must-Have**:
- [ ] Implements `IsolationManager` interface
- [ ] Uses Linux namespaces (user, PID, mount, network)
- [ ] Implements cgroup resource limits
- [ ] `IsAvailable()` returns false on non-Linux platforms
- [ ] Unit tests cover all methods
- [ ] Integration tests on Linux

**Should-Have**:
- [ ] seccomp filtering
- [ ] AppArmor profile support
- [ ] Performance benchmarks
- [ ] Configuration options for namespace isolation

#### macOS Platform Adapter

**Must-Have**:
- [ ] Implements `IsolationManager` interface
- [ ] Uses macOS Sandbox or equivalent
- [ ] Implements resource limits via `task_set_policy`
- [ ] `IsAvailable()` returns false on non-macOS platforms
- [ ] Unit tests cover all methods
- [ ] Integration tests on macOS

**Should-Have**:
- [ ] Code signing support
- [ ] Entitlement management
- [ ] Performance benchmarks

#### Windows Platform Adapter

**Must-Have**:
- [ ] Implements `IsolationManager` interface
- [ ] Uses Job Objects for process grouping
- [ ] Implements Windows Security Levels
- [ ] `IsAvailable()` returns false on non-Windows platforms
- [ ] Unit tests cover all methods
- [ ] Integration tests on Windows

**Should-Have**:
- [ ] AppContainer isolation (Windows 8+)
- [ ] Windows Defender exclusion handling
- [ ] Performance benchmarks

### Cross-Platform Requirements

1. **Interface Compliance**
   - [ ] All methods from `IsolationManager` implemented
   - [ ] Uses defined error types
   - [ ] Follows error wrapping conventions
   - [ ] Environment injection matches process adapter

2. **Testing Requirements**
   - [ ] Unit tests in `tests/unit/core/isolation/<platform>_test.go`
   - [ ] Integration tests in `tests/e2e/isolation_<platform>_test.go`
   - [ ] All tests pass on native platform
   - [ ] Code coverage >= 80%

3. **Code Quality**
   - [ ] Passes `go fmt ./...`
   - [ ] Passes `go vet ./...`
   - [ ] Passes `golangci-lint`
   - [ ] No secret values logged
   - [ ] Security audit passed (no known vulnerabilities)

4. **Documentation**
   - [ ] Platform prerequisites documented
   - [ ] Required permissions documented
   - [ ] Known limitations documented
   - [ ] Example usage provided
   - [ ] Updated `docs/PLATFORM_ADAPTERS.md`

### Performance Requirements

- [ ] `IsAvailable()` <= 100ms
- [ ] `Validate()` <= 1s
- [ ] `PrepareContext()` <= 500ms
- [ ] `Cleanup()` <= 5s

### Security Requirements

- [ ] Secret values never logged
- [ ] Inputs validated before execution
- [ ] Command injection prevented
- [ ] Environment values sanitized
- [ ] File permissions enforced

### Code Quality Gates

- [ ] All tests pass: `go test ./tests/unit/core/isolation/... && go test ./tests/e2e/...`
- [ ] No linting errors: `golangci-lint run`
- [ ] Security scan passes
- [ ] Code coverage >= 80% for adapter

### Merge Blockers

- Failing tests on native platform
- Security vulnerabilities
- Interface violations
- Missing error handling
- Poor performance (>2x slower than requirements)

### Pre-Merge Checklist

- [ ] CI/CD pipeline passes on all platforms
- [ ] Tests pass on native platform
- [ ] Two maintainer approvals
- [ ] Documentation complete
- [ ] No merge conflicts
- [ ] Changelog entry added

---

## Phase 4 Merge Criteria

### Must-Have Requirements

1. **Container Adapter Implementation**
   - [ ] Implements `IsolationManager` interface
   - [ ] Docker container creation and execution
   - [ ] Volume mounting for profile directories
   - [ ] Network isolation configuration
   - [ ] Resource limits (CPU, memory)
   - [ ] `IsAvailable()` checks for Docker daemon
   - [ ] Graceful cleanup of containers

2. **Docker Integration**
   - [ ] Uses Docker API (not CLI commands)
   - [ ] Handles Docker daemon connectivity
   - [ ] Validates Docker version compatibility
   - [ ] Handles container lifecycle (create, start, stop, remove)

3. **Configuration**
   - [ ] Reads container configuration from profile
   - [ ] Supports custom images
   - [ ] Supports volume mounts
   - [ ] Supports network configuration
   - [ ] Supports resource limits

4. **Testing Requirements**
   - [ ] Unit tests in `tests/unit/core/isolation/container_test.go`
   - [ ] Integration tests with actual Docker
   - [ ] Tests cover all Docker API operations
   - [ ] Tests handle Docker daemon failures gracefully
   - [ ] Code coverage >= 80%

5. **Code Quality**
   - [ ] Passes `go fmt ./...`
   - [ ] Passes `go vet ./...`
   - [ ] Passes `golangci-lint`
   - [ ] No hardcoded paths or credentials
   - [ ] Proper error handling for Docker API

6. **Documentation**
   - [ ] Docker prerequisites documented
   - [ ] Docker configuration options documented
   - [ ] Example Docker profiles provided
   - [ ] Troubleshooting guide for Docker issues
   - [ ] Updated `docs/PLATFORM_ADAPTERS.md`

### Should-Have Features

- [ ] Docker Compose integration
- [ ] Support for Docker Swarm
- [ ] Container health checks
- [ ] Custom entrypoint support
- [ ] Environment variable override support

### Performance Requirements

- [ ] `IsAvailable()` <= 100ms (daemon check)
- [ ] `Validate()` <= 2s (includes Docker API call)
- [ ] `PrepareContext()` <= 500ms
- [ ] Container creation <= 5s
- [ ] `Cleanup()` <= 10s (includes container removal)

### Security Requirements

- [ ] Secrets not passed as environment variables (use secrets mounts)
- [ ] Containers run as non-root user
- [ ] Read-only filesystem where possible
- [ ] No privileged mode unless required
- [ ] Resource limits enforced

### Code Quality Gates

- [ ] All tests pass: `go test ./tests/unit/core/isolation/... && go test ./tests/e2e/...`
- [ ] No linting errors: `golangci-lint run`
- [ ] Security scan passes
- [ ] Code coverage >= 80% for container adapter

### Merge Blockers

- Failing tests
- Docker API misuse
- Security vulnerabilities
- Interface violations
- Missing cleanup (orphaned containers)

### Pre-Merge Checklist

- [ ] CI/CD pipeline passes on all platforms with Docker
- [ ] Tests pass with Docker daemon
- [ ] Two maintainer approvals
- [ ] Documentation complete
- [ ] No merge conflicts
- [ ] Changelog entry added
- [ ] Known limitations documented

---

## General Merge Criteria (All Phases)

### Code Review

- [ ] At least one maintainer approves
- [ ] No outstanding review comments
- [ ] All suggested changes addressed or justified

### Testing

- [ ] All automated tests pass
- [ ] Manual testing performed (if applicable)
- [ ] Performance benchmarks meet requirements
- [ ] Security scan passes

### Documentation

- [ ] Code is well-commented
- [ ] Public APIs documented
- [ ] User-facing documentation updated
- [ ] Changelog entry added

### Compatibility

- [ ] No breaking changes without justification
- [ ] Backward compatibility maintained where possible
- [ ] Migration guide provided for breaking changes

### CI/CD

- [ ] All CI checks pass
- [ ] Build succeeds on all target platforms
- [ ] No merge conflicts with main branch

---

## Rollback Criteria

If any of the following issues are discovered within 7 days of merge:

1. **Critical Security Vulnerability**
   - Immediate rollback required
   - Hotfix priority

2. **Platform-Specific Regressions**
   - Evaluate impact vs. rollback
   - May require hotfix if critical

3. **Performance Degradation > 50%**
   - Evaluate impact
   - May require optimization or rollback

4. **Failing Production Tests**
   - Immediate investigation
   - Rollback if cannot fix quickly

---

## Post-Merge Requirements

1. **Monitoring**
   - Monitor for errors in production
   - Track performance metrics
   - Collect user feedback

2. **Bug Fixes**
   - Address critical bugs within 48 hours
   - Address non-critical bugs in next release

3. **Documentation Updates**
   - Update FAQs based on common issues
   - Add troubleshooting guides
   - Improve example usage

4. **Future Enhancements**
   - Document known limitations
   - Track enhancement requests
   - Plan for future phases
