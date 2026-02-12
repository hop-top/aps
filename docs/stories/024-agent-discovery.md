# Agent Discovery

**ID**: 024
**Feature**: Agent Protocol Adapter (specs/006)
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P2

## Story

As an external client, I want to list available agents and their capabilities so that I can build dynamic UIs.

## Acceptance Scenarios

1. **Given** profiles `agent-a` and `agent-b`, **When** I POST to `/agents/search`, **Then** I receive both agents with metadata.
2. **Given** profile `agent-a` with actions `foo` and `bar`, **When** I GET `/agents/agent-a/schemas`, **Then** I receive JSON Schema for each action.

## Tests

### Unit
- `tests/unit/adapters/agentprotocol_test.go` — `TestAgentProtocol_AgentsEndpoint`
- `tests/unit/core/protocol_test.go` — `TestAPSAdapter_GetAgent`

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_UserStory4_AgentDiscovery`
