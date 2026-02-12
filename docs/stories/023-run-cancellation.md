# Run Cancellation

**ID**: 023
**Feature**: Agent Protocol Adapter (specs/006)
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P2

## Story

As an external client, I want to cancel a running action so that I can stop runaway processes.

## Acceptance Scenarios

1. **Given** a running action, **When** I POST to `/runs/{run_id}/cancel`, **Then** the process is terminated and status becomes `cancelled`.
2. **Given** an already-completed run, **When** I POST to cancel, **Then** I receive HTTP 400.

## Tests

### Unit
- `tests/unit/adapters/agentprotocol_test.go` — `TestAgentProtocol_RunCancelEndpoint`

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_BackgroundRun`
