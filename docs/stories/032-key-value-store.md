---
status: shipped
---

# Key-Value Store

**ID**: 032
**Feature**: Agent Protocol Adapter
**Persona**: [External Client](../personas/external-client.md)
**Priority**: P2

## Story

As an external client, I want to store and retrieve key-value pairs through the Agent Protocol so that agents can persist state across runs.

## Acceptance Scenarios

1. **Given** a running agent protocol server, **When** I PUT a key-value pair, **Then** I can GET it back.
2. **Given** a stored key, **When** I DELETE it, **Then** subsequent GETs return not found.

## Tests

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_StoreOperations`
