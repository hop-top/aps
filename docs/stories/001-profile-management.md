# Profile Management

**ID**: 001
**Feature**: CLI Core (specs/001)
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md) (tested by [006](006-profile-lifecycle-tests.md))
**Priority**: P1

## Story

As a user, I want to create and manage agent profiles so that I can maintain isolated identities and configurations.

## Acceptance Scenarios

1. **Given** no existing profiles, **When** I run `aps profile list`, **Then** I see no output (or empty list).
2. **Given** no profile `agent-a`, **When** I run `aps profile new agent-a`, **Then** the profile directory structure is created with default files.
3. **Given** an existing profile `agent-a`, **When** I run `aps profile list`, **Then** `agent-a` is listed.
4. **Given** an existing profile `agent-a`, **When** I run `aps profile show agent-a`, **Then** I see the profile configuration and modules.
5. **Given** an existing profile `agent-a`, **When** I run `aps profile new agent-a` without force, **Then** it fails to overwrite.

## Tests

### E2E
- `tests/e2e/profile_test.go` — `TestProfileLifecycle`, `TestProfileOverwrite`

### Unit
- `tests/unit/core/profile_bundle_test.go` — `TestProfileBundle_ExportImport`
