---
status: shipped
---

# A2A Server

**ID**: 017
**Feature**: A2A Protocol
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to expose my APS profiles as A2A agents via a server so that other agents can submit tasks and receive results.

## Acceptance Scenarios

1. **Given** a profile with A2A enabled, **When** I start the A2A server, **Then** it responds to task requests per the A2A protocol.
2. **Given** a running A2A server, **When** a valid task is submitted, **Then** the profile's action is executed and the result is returned.

## Tests

### Unit
- `internal/a2a/server_test.go` — `TestNewServer_A2ADisabled`, `TestNewServer_NilProfile`, `TestNewServer_NilConfig`, `TestServer_Start`, `TestServer_Stop`, `TestServer_GetAddress`

### E2E
- `tests/e2e/a2a_server_test.go` — `TestA2AServer_TaskSubmission`, `TestA2AServer_StreamingTaskSubmission`, `TestA2AServer_TaskCancellation`, `TestA2AServer_PushNotificationConfig`
