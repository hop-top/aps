# Streaming Run

**ID**: 022
**Feature**: Agent Protocol Adapter (specs/006)
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P1

## Story

As an external client, I want to stream action output in real-time so that I can display progress to users.

## Acceptance Scenarios

1. **Given** a profile `myagent` with action `longrun`, **When** I POST to `/runs/stream`, **Then** I receive SSE events as the action produces output.
2. **Given** a streaming run, **When** the action completes, **Then** I receive a final event with `status: completed`.

## Tests

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_UserStory2_StreamingRun`
