# Install Capability

**ID**: 026
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want to install a capability (e.g., a specific set of snippets or a tool config) so that I can use it across my profiles.

## Acceptance Scenarios

1. **Given** a capability `gh-cli-extensions`, **When** I run `aps capability install gh-cli-extensions`, **Then** the artifacts are downloaded to `~/.aps/capabilities/gh-cli-extensions`.
2. **Given** an installed capability, **When** I list capabilities, **Then** it appears in the output.

## Tests

### Unit
- `tests/unit/core/capability/capability_test.go` — `TestCapabilityLifecycle`

### E2E
- `tests/e2e/capability_test.go` — `TestCapabilityCommands`
