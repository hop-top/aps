---
status: shipped
---

# Profile Presentation (Avatar + Color)

**ID**: 062
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md) (tested by [063](063-profile-presentation-tests.md))
**Priority**: P2

## Story

As a user, I want my agent profiles to carry an optional avatar image and
display color so that downstream surfaces (org chart, email signatures,
dashboards) can render them with visual identity, without me hand-picking
values for every profile.

## Acceptance Scenarios

1. **Given** an existing profile, **When** I run `aps profile show <id>`,
   **Then** I see optional `avatar` and `color` fields if set, and they
   are absent if unset.
2. **Given** a new profile, **When** I run
   `aps profile create <id> --avatar https://example.com/me.png --color "#3b82f6"`,
   **Then** both values are persisted to the profile yaml.
3. **Given** a new profile, **When** I run
   `aps profile create <id> --auto-avatar --auto-color`,
   **Then** a deterministic avatar URL and palette color are generated
   from the profile id.
4. **Given** the same profile id, **When** auto-avatar/auto-color are
   regenerated, **Then** the same values are produced (deterministic).
5. **Given** `~/.config/aps/config.yaml` with
   `profile: { color: true, avatar: { enabled: auto, provider: dicebear, style: bottts, size: 256 } }`,
   **When** I create a profile without flags, **Then** the configured
   provider/style/size are applied automatically.
6. **Given** an existing profile, **When** I run
   `aps profile edit <id> --color "#ff0000"`, **Then** only `color` is
   updated and other fields are preserved.
7. **Given** an existing profile with a color set, **When** I run
   `aps profile edit <id> --avatar ""`, **Then** the avatar is cleared
   while color remains.
8. **Given** any registered avatar provider, **When** I pass
   `--avatar-provider <name>`, **Then** the kit/avatar registry resolves
   the provider and generates accordingly; unknown provider yields an
   empty avatar (graceful fallback, not an error).

## Implementation Notes

- Avatar URL generation is delegated to `kit/go/core/avatar` (provider
  facade). aps does not hardcode dicebear; `dicebear` is the default
  registered provider. Other providers (gravatar, boring, custom) plug
  in via `avatar.RegisterProvider`.
- Color palette is a 12-entry curated set in `internal/core/profile_presentation.go`,
  deterministically indexed by SHA-256(id).
- Tri-state config (`true`|`false`|`auto`) for `color` and `avatar.enabled`
  via `core.AutoMode`.

## Tests

### E2E
- `tests/e2e/profile_presentation_test.go` —
  `TestProfilePresentation_AutoAvatarColor`,
  `TestProfilePresentation_ExplicitFlagsBeatAuto`,
  `TestProfilePresentation_ConfigDefaults`,
  `TestProfilePresentation_EditFields`,
  `TestProfilePresentation_DeterministicSeed`,
  `TestProfilePresentation_UnknownProviderGraceful`

### Unit
- `internal/core/profile_presentation_test.go` —
  `TestGenerateProfileColor_Deterministic`,
  `TestGenerateProfileColor_DifferentIDs`,
  `TestGenerateProfileAvatar`,
  `TestAutoMode_ShouldAutoAssign`
