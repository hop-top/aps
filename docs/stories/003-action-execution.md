---
status: shipped
---

# Action Execution

**ID**: 003
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md) (tested by [008](008-action-discovery-tests.md))
**Priority**: P2

## Story

As a user, I want to discover and execute defined actions so that I can perform repeatable tasks.

## Acceptance Scenarios

1. **Given** a profile with actions, **When** I run `aps action list <profile>`, **Then** I see the available actions.
2. **Given** an action `hello.sh`, **When** I run `aps action run <profile> hello`, **Then** the script executes.
3. **Given** an action accepting payload, **When** I run `aps action run <profile> action --payload-file data.json`, **Then** the action receives the data on stdin.

## Tests

### E2E
- `tests/e2e/action_test.go` — `TestActionDiscovery`, `TestActionRun`, `TestActionPayload`
- `tests/e2e/execution_engine_test.go` — `TestExecutionEngineActionExecution`, `TestExecutionEngineActionWithPayload`

### Unit
- `tests/unit/core/execution_refactored_test.go` — `TestRunActionWithProcessIsolation`, `TestRunActionWithPayload`, `TestRunActionInvalidProfile`, `TestRunActionInvalidAction`
