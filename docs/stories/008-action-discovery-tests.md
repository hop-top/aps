# Action Discovery Tests

**ID**: 008
**Feature**: E2E Tests
**Persona**: [Maintainer](../personas/maintainer.md)
**Related Personas**: [User](../personas/user.md) (validates [003](003-action-execution.md))
**Priority**: P2

## Story

As a maintainer, I want verification that actions (scripts) are discovered and executed properly with payloads so that the scripting capability works.

## Acceptance Scenarios

1. **Given** a profile with a `.sh` script in `actions/`, **When** `aps action list` is run, **Then** the script is listed.
2. **Given** a simple echo script, **When** `aps action run` is called, **Then** output matches expected.
3. **Given** a script reading stdin, **When** `aps action run ... --payload-file` is used, **Then** script receives the content.

## Tests

### E2E
- `tests/e2e/action_test.go` — `TestActionDiscovery`, `TestActionRun`, `TestActionPayload`
- `tests/e2e/execution_engine_test.go` — `TestExecutionEngineActionExecution`, `TestExecutionEngineActionWithPayload`
