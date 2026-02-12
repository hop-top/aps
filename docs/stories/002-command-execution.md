# Command Execution

**ID**: 002
**Feature**: CLI Core (specs/001)
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md) (tested by [007](007-execution-environment-tests.md))
**Priority**: P1

## Story

As a user, I want to run arbitrary commands within a profile's context so that I can utilize the profile's secrets and environment.

## Acceptance Scenarios

1. **Given** a profile `agent-a` with a secret `FOO=BAR`, **When** I run `aps run agent-a -- env`, **Then** the output contains `FOO=BAR`.
2. **Given** a profile `agent-a`, **When** I run `aps run agent-a -- whoami`, **Then** the command executes successfully.
3. **Given** a non-existent profile, **When** I run `aps run fake -- cmd`, **Then** it returns an error.

## Tests

### E2E
- `tests/e2e/run_test.go` — `TestExecutionInjection`, `TestSecretInjection`
- `tests/e2e/execution_engine_test.go` — `TestExecutionEngineWithProcessIsolation`, `TestExecutionEngineWithDefaultIsolation`, `TestExecutionEngineBackwardCompatibility`

### Unit
- `tests/unit/core/execution_test.go` — `TestInjectEnvironment`
- `tests/unit/core/execution_refactored_test.go` — `TestRunCommandWithProcessIsolation`, `TestRunCommandWithDefaultIsolation`, `TestBackwardCompatibility_InjectEnvironment`
