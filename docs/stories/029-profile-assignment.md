---
status: paper
---

# Profile Assignment

**ID**: 029
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As a user, I want to enable specific capabilities only for certain profiles so that my "Work" profile has different tools than my "Personal" profile.

## Acceptance Scenarios

1. **Given** profile `work-agent`, **When** I run `aps profile capability add work-agent git-enterprise`, **Then** the `git-enterprise` config is active when I run `aps run work-agent`.

## Tests

### E2E
- planned: `tests/e2e/profile_capability_test.go::TestProfileCapability_AddActivatesOnRun`

### Unit
- `tests/unit/core/profile_capability_test.go` — `TestAddCapabilityToProfile`, `TestRemoveCapabilityFromProfile`, `TestProfileHasCapability`, `TestProfilesUsingCapability`, `TestInjectEnvironment_PerProfileCaps`, `TestInjectEnvironment_NonEnabledCapsNotInjected`
