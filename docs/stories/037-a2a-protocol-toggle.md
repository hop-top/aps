---
status: shipped-no-e2e
---

# A2A Protocol Toggle

**ID**: 037
**Feature**: A2A Protocol
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to enable or disable the A2A (Agent-to-Agent) protocol for my profile so that I can control when my profile is exposed as an A2A agent.

## Acceptance Scenarios

1. **Given** a profile without A2A enabled, **When** I run `aps a2a toggle --profile <id>`, **Then** A2A is enabled with default configuration (jsonrpc/127.0.0.1:8081).
2. **Given** a profile with A2A enabled, **When** I run `aps a2a toggle --profile <id>`, **Then** A2A is disabled and removed from the profile.
3. **Given** a profile, **When** I run `aps a2a toggle --profile <id> --enabled=on --protocol=grpc --port=9000`, **Then** A2A is enabled with custom configuration.
4. **Given** a profile with A2A enabled, **When** I run `aps a2a server --profile <id>`, **Then** the A2A server starts successfully with the configured settings.

## Tests

### E2E
- `tests/e2e/protocol_test.go` — `TestA2AToggle_Enable`, `TestA2AToggle_Disable`, `TestA2AToggle_CustomConfig`, `TestA2AToggle_ServerIntegration`
