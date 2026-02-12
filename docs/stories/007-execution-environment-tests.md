# Execution Environment Tests

**ID**: 007
**Feature**: E2E Tests
**Persona**: [Maintainer](../personas/maintainer.md)
**Related Personas**: [User](../personas/user.md) (validates [002](002-command-execution.md))
**Priority**: P1

## Story

As a maintainer, I want automated verification that `aps run` correctly injects secrets and variables so that the core value proposition of isolated execution environments is validated.

## Acceptance Scenarios

1. **Given** a profile with a known secret in `secrets.env`, **When** running `aps run ... -- env`, **Then** the secret is present in output.
2. **Given** a profile, **When** running `aps run ... -- env`, **Then** `AGENT_PROFILE_ID` and `AGENT_PROFILE_DIR` are set correctly.
3. **Given** a profile with `git.enabled=true`, **When** running `env`, **Then** `GIT_CONFIG_GLOBAL` is injected.

## Tests

### E2E
- `tests/e2e/run_test.go` — `TestExecutionInjection`, `TestSecretInjection`
- `tests/e2e/execution_engine_test.go` — `TestExecutionEngineWithProcessIsolation`, `TestExecutionEngineWithDefaultIsolation`
