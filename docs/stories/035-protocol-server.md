---
status: shipped
---

# Protocol Server

**ID**: 035
**Feature**: Agent Protocol Adapter
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to start a protocol server for my profile so that external clients can interact with it via the Agent Protocol HTTP API.

## Acceptance Scenarios

1. **Given** a profile, **When** I start the protocol server, **Then** it listens on the configured port and serves Agent Protocol endpoints.
2. **Given** a running protocol server, **When** an external client sends a run request, **Then** the profile's command is executed and the result is returned.

## Tests

### E2E
- `tests/e2e/agent_protocol_test.go` — `TestAgentProtocol_UserStory1_StatelessRun`, `TestAgentProtocol_UserStory2_StreamingRun`, `TestAgentProtocol_UserStory4_AgentDiscovery`, `TestAgentProtocol_UserStory5_ThreadSessionManagement`, `TestAgentProtocol_BackgroundRun`, `TestAgentProtocol_ErrorHandling`
