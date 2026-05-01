---
status: shipped-no-e2e
---

# ACP Protocol Toggle

**ID**: 038
**Feature**: Agent Protocol Adapter
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to enable or disable the ACP (Agent Client Protocol) for my profile so that I can control when my profile is available to editor clients.

## Acceptance Scenarios

1. **Given** a profile without ACP enabled, **When** I run `aps acp toggle --profile <id>`, **Then** ACP is enabled with default configuration (stdio transport).
2. **Given** a profile with ACP enabled, **When** I run `aps acp toggle --profile <id>`, **Then** ACP is disabled and removed from the profile.
3. **Given** a profile, **When** I run `aps acp toggle --profile <id> --enabled=on --transport=http --port=9000`, **Then** ACP is enabled with custom configuration.
4. **Given** a profile with ACP enabled, **When** I run `aps acp server <id>`, **Then** the ACP server starts successfully with the configured settings.

## Tests

### E2E
- `tests/e2e/protocol_test.go` — `TestACPToggle_Enable`, `TestACPToggle_Disable`, `TestACPToggle_CustomConfig`, `TestACPToggle_ServerIntegration`
