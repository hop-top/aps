---
status: shipped
---

# Capability Environment

**ID**: 036
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want capabilities to inject environment variables into my profile's execution context so that tools and integrations are automatically configured when I run commands.

## Acceptance Scenarios

1. **Given** a profile with an installed capability that defines env vars, **When** I run a command, **Then** those env vars are present in the execution environment.
2. **Given** a capability not assigned to the profile, **When** I run a command, **Then** that capability's env vars are not injected.

## Tests

### E2E
- `tests/e2e/capability_test.go` — `TestCapabilityCommands`
