---
status: paper
---

# Smart Linking

**ID**: 028
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want specific tools to be automatically configured without remembering their paths so I can just say "link copilot" and have it work.

## Acceptance Scenarios

1. **Given** a pattern registry with `copilot -> .github/agents/agent.agent.md`, **When** I run `aps capability link copilot`, **Then** the capability is linked to `.github/agents/agent.agent.md` in the current directory.
2. **Given** I run `aps capability watch --tool windsurf`, **Then** it links the capability to `.windsurf/workflows/agent.md`.

## Tests

### E2E
- planned: `tests/e2e/capability_link_test.go::TestCapability_SmartLinkCopilot`
- planned: `tests/e2e/capability_link_test.go::TestCapability_WatchToolWindsurf`

### Unit
- `tests/unit/core/capability/capability_test.go` — `TestSmartLinking`
