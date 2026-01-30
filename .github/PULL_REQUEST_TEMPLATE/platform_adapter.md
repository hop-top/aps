## Description
<!-- Brief description of changes -->

**Platform Adapter Type**: [ ] Linux [ ] macOS [ ] Windows [ ] Docker [ ] Other: ______

**Isolation Level**: [ ] platform [ ] container

## Type of Change
<!-- Mark all that apply -->
- [ ] New feature (platform adapter implementation)
- [ ] Bug fix
- [ ] Breaking change
- [ ] Documentation update

## Checklist

### Interface Compliance
- [ ] Implements `IsolationManager` interface from `internal/core/isolation/manager.go`
- [ ] All required methods implemented:
  - [ ] `PrepareContext(profileID string) (*ExecutionContext, error)`
  - [ ] `SetupEnvironment(cmd interface{}) error`
  - [ ] `Execute(command string, args []string) error`
  - [ ] `ExecuteAction(actionID string, payload []byte) error`
  - [ ] `Cleanup() error`
  - [ ] `Validate() error`
  - [ ] `IsAvailable() bool`
- [ ] Uses defined error types
- [ ] Follows error wrapping conventions

### Testing
- [ ] Unit tests in `tests/unit/core/isolation/<adapter>_test.go`
- [ ] Integration tests in `tests/e2e/isolation_<adapter>_test.go`
- [ ] All tests pass on target platform
- [ ] Tests cover success and failure paths
- [ ] Tests include `IsAvailable()` verification

### Code Quality
- [ ] Code formatted with `go fmt ./...`
- [ ] Passes `go vet ./...`
- [ ] Passes `golangci-lint`
- [ ] No secret values logged
- [ ] File permissions enforced (0600 for secrets)

### Documentation
- [ ] Platform prerequisites documented
- [ ] Required permissions documented
- [ ] Configuration options documented (if any)
- [ ] Limitations documented
- [ ] Example usage provided
- [ ] Updated `docs/PLATFORM_ADAPTERS.md` (if applicable)

### Security
- [ ] Secret values never logged
- [ ] Inputs validated before execution
- [ ] Command injection prevented
- [ ] Environment values sanitized

## Platform-Specific Checks

### For Linux Platform Adapter
- [ ] Namespace isolation implemented
- [ ] cgroup resource limits implemented
- [ ] seccomp filtering considered
- [ ] AppArmor profiles considered
- [ ] Tested on Linux (kernel 3.10+)

### For macOS Platform Adapter
- [ ] Sandbox integration implemented
- [ ] Resource limits via task_set_policy
- [ ] Process attribute restrictions
- [ ] Tested on macOS 10.15+

### For Windows Platform Adapter
- [ ] Job Objects implemented
- [ ] Windows Security Levels implemented
- [ ] AppContainer isolation (if applicable)
- [ ] Tested on Windows 8+ or Server 2012+

### For Container Adapter
- [ ] Docker container creation/execution
- [ ] Volume mounting for profiles
- [ ] Network isolation configured
- [ ] Resource limits (CPU, memory)
- [ ] Tested with Docker daemon

## Testing Evidence

### Unit Test Results
```
<!-- Paste unit test output -->
go test -v ./tests/unit/core/isolation/<adapter>_test.go
```

### Integration Test Results
```
<!-- Paste integration test output -->
go test -v ./tests/e2e/isolation_<adapter>_test.go
```

### Platform Test Results
<!-- List platforms tested on -->
- [ ] Linux (version: ______)
- [ ] macOS (version: ______)
- [ ] Windows (version: ______)
- [ ] Other: ______

## Related Issues

Closes #_____
Related to #_____

## How This Has Been Tested

<!-- Describe the tests you ran to verify your changes -->

### Manual Testing
- [ ] Profile creation with adapter's isolation level
- [ ] Command execution under isolation
- [ ] Action execution under isolation
- [ ] Environment injection verification
- [ ] Cleanup verification

### Example Commands Tested
```bash
<!-- Example commands used for testing -->
```

## Performance Impact

- `IsAvailable()` completes within 100ms: [ ] Yes [ ] No (_______ms)
- `Validate()` completes within 1s: [ ] Yes [ ] No (_______ms)
- `PrepareContext()` completes within 500ms: [ ] Yes [ ] No (_______ms)

## Breaking Changes

<!-- Describe any breaking changes and migration steps -->
N/A or describe changes

## Screenshots (if applicable)

<!-- Add screenshots for TUI or CLI changes -->

## Additional Notes

<!-- Any additional context, known issues, or future work -->
