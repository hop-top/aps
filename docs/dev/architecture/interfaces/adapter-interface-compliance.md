# Platform Adapter Interface Compliance Requirements

This document defines the interface contract that all platform isolation adapters MUST implement.

## Required Interface

All platform adapters MUST implement the `IsolationManager` interface defined in `internal/core/isolation/manager.go`:

```go
type IsolationManager interface {
    PrepareContext(profileID string) (*ExecutionContext, error)
    SetupEnvironment(cmd interface{}) error
    Execute(command string, args []string) error
    ExecuteAction(actionID string, payload []byte) error
    Cleanup() error
    Validate() error
    IsAvailable() bool
}
```

## Method Specifications

### PrepareContext(profileID string) (*ExecutionContext, error)

**Purpose**: Initialize execution context for a profile

**Requirements**:
- MUST load and validate the profile from `<data>/profiles/<profile-id>/profile.yaml`
- MUST return an `ExecutionContext` with all required fields populated:
  - `ProfileID`: The profile identifier
  - `ProfileDir`: Absolute path to profile directory
  - `ProfileYaml`: Absolute path to profile.yaml
  - `SecretsPath`: Absolute path to secrets.env
  - `DocsDir`: Absolute path to docs directory
  - `Environment`: Map for environment variables (initially empty)
  - `WorkingDir`: Directory where commands will execute (typically profile directory)
- MUST return error if profile does not exist or is invalid
- MUST validate isolation configuration for the requested level

**Error Cases**:
- `ErrInvalidProfile`: Profile not found or invalid
- Any I/O errors reading profile files

### SetupEnvironment(cmd interface{}) error

**Purpose**: Configure environment variables for command execution

**Requirements**:
- MUST accept either `*exec.Cmd` (for direct execution) or adapter-specific command types
- MUST inject all required APS environment variables with configured prefix:
  - `<PREFIX>_PROFILE_ID`
  - `<PREFIX>_PROFILE_DIR`
  - `<PREFIX>_PROFILE_YAML`
  - `<PREFIX>_PROFILE_SECRETS`
  - `<PREFIX>_PROFILE_DOCS_DIR`
- MUST load and inject secrets from `secrets.env` (redacting values if logging)
- MUST handle git configuration injection if `git.enabled = true`
- MUST handle SSH key injection if `ssh.enabled = true`
- MUST preserve existing environment variables (no clobbering)
- MUST set working directory to `context.WorkingDir`

**Error Cases**:
- Error if `cmd` type is not supported
- Error if context not prepared
- Error loading secrets or configuration

### Execute(command string, args []string) error

**Purpose**: Execute a command under the profile context

**Requirements**:
- MUST use `PrepareContext` to initialize execution environment
- MUST use `SetupEnvironment` to configure the command
- MUST attach stdin/stdout/stderr from parent process
- MUST return the exact exit code from the invoked process
- MUST wrap execution errors with `ErrExecutionFailed`
- Platform adapters MAY use platform-specific execution mechanisms

**Error Cases**:
- `ErrExecutionFailed`: Command execution failed
- Errors from `PrepareContext` or `SetupEnvironment`

### ExecuteAction(actionID string, payload []byte) error

**Purpose**: Execute a profile action with optional payload

**Requirements**:
- MUST resolve action from `<data>/profiles/<profile-id>/actions/<action-id>.*`
- MUST detect action type based on extension (.sh, .py, .js)
- MUST use appropriate interpreter:
  - `.sh`: `sh <script-path>`
  - `.py`: `python3 <script-path>`
  - `.js`: `node <script-path>`
  - Default: Execute directly
- MUST inject payload to stdin if provided
- MUST attach stdin/stdout/stderr if no payload
- MUST use same environment injection as `Execute`
- MUST return the exact exit code from the action

**Error Cases**:
- Error if action not found
- `ErrExecutionFailed`: Action execution failed

### Cleanup() error

**Purpose**: Clean up resources after execution

**Requirements**:
- MUST release any platform-specific resources (containers, sandboxes, etc.)
- MUST clear context references if applicable
- SHOULD be idempotent (safe to call multiple times)
- MUST not error if called before execution

### Validate() error

**Purpose**: Validate adapter readiness

**Requirements**:
- MUST verify all required platform tools are available
- MUST verify profile directory structure is valid
- MUST verify required configuration files exist
- MUST check for platform-specific permissions

**Error Cases**:
- Error if platform dependencies are missing
- Error if profile structure is invalid

### IsAvailable() bool

**Purpose**: Check if adapter is available on current platform

**Requirements**:
- MUST return false if required platform tools are missing
- MUST return false if OS does not support this isolation level
- MUST be fast and non-blocking (no heavy initialization)
- MUST not modify any state

## Platform-Specific Considerations

### Process Isolation (Reference Implementation)

Located at: `internal/core/isolation/process.go`

- Always available on all platforms
- Uses standard Go `exec.Command`
- No special requirements

### Platform Isolation (Linux)

MUST implement:
- Linux namespace isolation (user, PID, mount, network)
- cgroup resource limits
- seccomp filtering (recommended)
- AppArmor profiles (recommended)

Prerequisites:
- Linux kernel 3.10+
- Required namespaces enabled in kernel

### Platform Isolation (macOS)

MUST implement:
- Sandbox integration (System Sandbox or similar)
- Resource limits via `task_set_policy`
- Process attribute restrictions

Prerequisites:
- macOS 10.15+
- App Sandbox entitlement (if using Apple sandbox)

### Platform Isolation (Windows)

MUST implement:
- Job Objects for process grouping
- Windows Security Levels
- AppContainer isolation (Windows 8+)

Prerequisites:
- Windows 8+ or Windows Server 2012+

### Container Isolation (Docker)

MUST implement:
- Docker container creation and execution
- Volume mounting for profile directories
- Network isolation configuration
- Resource limits (CPU, memory)

Prerequisites:
- Docker daemon running
- User has docker permissions

## Testing Requirements

### Unit Tests

Each adapter MUST have unit tests in `tests/unit/core/isolation/<adapter>_test.go`:

1. `TestPrepareContext_ValidProfile`
2. `TestPrepareContext_InvalidProfile`
3. `TestSetupEnvironment_Injection`
4. `TestExecute_Success`
5. `TestExecute_Failure`
6. `TestExecuteAction_WithPayload`
7. `TestExecuteAction_WithoutPayload`
8. `TestCleanup_Idempotent`
9. `TestValidate_Success`
10. `TestValidate_Failure`
11. `TestIsAvailable_True`
12. `TestIsAvailable_False`

### Integration Tests

Each adapter MUST have integration tests in `tests/e2e/isolation_<adapter>_test.go`:

1. Test profile creation with adapter's isolation level
2. Test command execution under isolation
3. Test action execution under isolation
4. Test environment injection
5. Test cleanup
6. Test fallback behavior

### Cross-Platform Tests

All adapters MUST pass:
- Unit tests on all supported platforms
- Integration tests on their native platform
- `go vet` and `go fmt` checks
- golangci-lint with default rules

## Security Requirements

1. MUST NOT log secret values (keys only)
2. MUST enforce file permissions (0600 for secrets)
3. MUST validate all inputs before execution
4. MUST sanitize environment variable values
5. MUST prevent command injection
6. MUST handle signal propagation appropriately

## Error Handling

All adapters MUST:
- Use defined error types (`ErrIsolationNotSupported`, `ErrInvalidProfile`, `ErrExecutionFailed`, `ErrStrictModeViolation`, `ErrNoAvailableAdapter`)
- Wrap errors with context using `fmt.Errorf("%w: %v", err, context)`
- Return meaningful error messages
- Clean up resources on error paths

## Performance Requirements

- `IsAvailable()` MUST complete within 100ms
- `Validate()` MUST complete within 1s
- `PrepareContext()` MUST complete within 500ms for typical profiles
- Resource cleanup MUST complete within 5s

## Documentation Requirements

Each adapter MUST:
1. Document platform prerequisites
2. Document required permissions
3. Document configuration options (if any)
4. Document limitations and known issues
5. Provide example usage in `docs/PLATFORM_ADAPTERS.md`
