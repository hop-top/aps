# Thread/Session Management

**ID**: 025
**Feature**: Agent Protocol Adapter (specs/006)
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P3

## Story

As an external client, I want to maintain session state across multiple runs so that I can build multi-turn interactions.

## Acceptance Scenarios

1. **Given** no existing threads, **When** I POST to `/threads` with `{"agent_id": "myagent"}`, **Then** a session is created and ID returned.
2. **Given** an existing thread, **When** I POST `/threads/{id}/runs`, **Then** the run executes within that session context.

## Tests

### Unit
- `tests/unit/adapters/agentprotocol_test.go` — `TestAgentProtocol_ThreadsEndpoint`
- `tests/unit/core/protocol_test.go` — `TestAPSAdapter_CreateSession`
- `tests/unit/core/session_test.go` — `TestGetRegistry`, `TestSSHKeyManager`

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_UserStory5_ThreadSessionManagement`
- `tests/e2e/session_test.go` — `TestSessionCommands`
