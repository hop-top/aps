---
status: shipped
---

# Profile Presentation Tests

**ID**: 063
**Feature**: E2E Tests
**Persona**: [Maintainer](../personas/maintainer.md)
**Related Personas**: [User](../personas/user.md) (validates [062](062-profile-presentation.md))
**Priority**: P2

## Story

As a maintainer, I want automated verification of avatar/color generation,
config-driven defaults, and edit semantics so that visual identity stays
deterministic across releases and the kit/avatar facade remains decoupled
from any single provider.

## Acceptance Scenarios

1. **Given** a clean state, **When** `aps profile create <id> --auto-avatar --auto-color`
   is run, **Then** the resulting yaml contains a non-empty `avatar:` URL
   and a `#RRGGBB` `color`.
2. **Given** explicit `--avatar` and `--color` flags, **When** combined
   with `--auto-avatar --auto-color`, **Then** the explicit values win.
3. **Given** `~/.config/aps/config.yaml` with `profile.avatar.enabled: true`
   and `profile.color: true`, **When** `aps profile create <id>` is run
   with no flags, **Then** the profile is created with auto-generated
   avatar and color.
4. **Given** an existing profile, **When** `aps profile edit <id> --color`
   is run with a new value, **Then** color is updated and avatar is
   unchanged; the inverse holds for `--avatar`.
5. **Given** the same id run twice on different temp homes, **When**
   auto-generation runs, **Then** the same color and avatar URL are
   produced (no time-based or random seeding).
6. **Given** `--avatar-provider nonexistent`, **When** `--auto-avatar`
   is set, **Then** the avatar field is empty and the create succeeds
   (graceful — config errors don't block profile creation).
7. **Given** `--avatar-style bottts --avatar-size 128`, **When**
   `--auto-avatar` is set with the default dicebear provider, **Then**
   the generated URL contains `bottts` and `size=128`.

## Tests

### E2E
- `tests/e2e/profile_presentation_test.go`

### Related
- `internal/core/profile_presentation_test.go` (unit, story 062)
- `kit/go/core/avatar/avatar_test.go` (unit, kit-side)
