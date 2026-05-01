---
status: shipped
---

# Stateless Run

**ID**: 021
**Feature**: Agent Protocol Adapter
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P1

## Story

As an external client, I want to trigger an action via HTTP and receive the result so that I can integrate APS with other systems.

## Acceptance Scenarios

1. **Given** a profile `myagent` with action `hello`, **When** I POST to `/runs/wait` with `{"agent_id": "myagent", "input": {"action": "hello"}}`, **Then** I receive the action output in the response body.
2. **Given** a non-existent profile, **When** I POST to `/runs/wait`, **Then** I receive HTTP 404.
3. **Given** an action that fails, **When** I POST to `/runs/wait`, **Then** I receive HTTP 200 with `status: failed` and error details.

## Tests

### Unit
- `tests/unit/adapters/agentprotocol_test.go` — `TestAgentProtocol_RunWaitEndpoint`, `TestAgentProtocol_CreateRunEndpoint`, `TestAgentProtocol_GetRunEndpoint`
- `tests/unit/core/protocol_test.go` — `TestAPSAdapter_ExecuteRun_InvalidInput`

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_UserStory1_StatelessRun`, `TestAgentProtocol_ErrorHandling`
