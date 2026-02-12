# Profile Lifecycle Tests

**ID**: 006
**Feature**: E2E Tests (specs/002)
**Persona**: [Maintainer](../personas/maintainer.md)
**Related Personas**: [User](../personas/user.md) (validates [001](001-profile-management.md))
**Priority**: P1

## Story

As a maintainer, I want automated verification of profile management commands so that the basic data model works correctly across changes.

## Acceptance Scenarios

1. **Given** a clean state, **When** `aps profile list` is run, **Then** output is empty.
2. **Given** a new profile `e2e-agent`, **When** created via `aps profile new`, **Then** it appears in `aps profile list`.
3. **Given** an existing profile, **When** `aps profile show` is run, **Then** valid YAML matches expectations.
4. **Given** an existing profile, **When** attempting to overwrite without `--force`, **Then** it fails.
5. **Given** an existing profile, **When** overwritten with `--force`, **Then** it succeeds.

## Tests

### E2E
- `tests/e2e/profile_test.go` — `TestProfileLifecycle`, `TestProfileOverwrite`
