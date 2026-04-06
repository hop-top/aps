# A2A Client

**ID**: 018
**Feature**: A2A Protocol
**Persona**: [User](../personas/user.md)
**Priority**: P3

## Story

As a user, I want to send tasks from one APS profile to another using the A2A protocol so that my agents can collaborate.

## Acceptance Scenarios

1. **Given** two profiles with A2A enabled, **When** I create a task from profile A to profile B, **Then** the task is delivered and processed.
2. **Given** an A2A client, **When** I send a message, **Then** I can track task status through completion or cancellation.

## Tests

### Unit
- `tests/unit/a2a/client_test.go` — `TestNewClient_InvalidProfileID`, `TestNewClient_NilProfile`, `TestNewClient_A2ADisabled`, `TestNewClient_ValidProfile`
- `tests/unit/a2a/client_message_test.go` — `TestClient_SendMessage_InvalidMessage`, `TestClient_SendMessage_ValidMessage`

### E2E
- `tests/e2e/a2a_client_test.go` — `TestClient_ProfileToProfileCommunication`, `TestClient_ListTasks`, `TestClient_CancelTask`
