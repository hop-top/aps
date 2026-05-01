---
status: shipped
---

# Profile-Bound Capability Bundles (Nadia EA Worked Example)

**ID**: 053
**Feature**: Capability Management
**Persona**: [User](../personas/user.md)
**Related Personas**: [Maintainer](../personas/maintainer.md)
**Priority**: P2

## Story

As a profile owner, I want declarative capability bundles bound to my profile so external
flows (e.g. meeting-intake-pipeline) can invoke me through `aps adapter exec` without
hard-coding scripts or per-profile bypass logic.

Worked example: Nadia (Executive Office). Meeting-intake flow declares Nadia as the EA
identity; Nadia's profile YAML declares the capability set; adapter enforces. Same
mechanism reused for Sami, Rami, and other agents.

Nadia's declared set:

- `meeting:read-drops` — read files under `~/.ops/meetings/<slug>/`
- `meeting:capture-to-ctxt` — invoke `ctxt analyze` with `@meeting.<slug>` mention
- `meeting:create-tlc-task` — invoke `tlc task create`; assignee MUST differ from caller
- `email:draft-only` — invoke `aps adapter exec email draft` (not `send-direct`)
- `notify:founder` — invoke `aps adapter exec email send` only to whitelisted founder addr

## Acceptance Scenarios

1. **Given** Nadia profile YAML declares `capabilities: [meeting:*, email:draft-only,
   notify:founder]`, **When** flow runs `aps adapter exec email draft --profile nadia`,
   **Then** call succeeds (email:draft-only granted).
2. **Given** same profile, **When** flow runs `aps adapter exec email send-direct
   --profile nadia --to external@example.com`, **Then** call rejected; error cites
   missing `email:send-direct` capability.
3. **Given** Nadia tries `notify:founder` to a non-founder address, **When** adapter exec
   runs, **Then** rejected; error cites recipient not in founder whitelist.
4. **Given** an action whose required capability is not declared, **When** flow invokes
   it, **Then** rejected with helpful error listing capabilities Nadia DOES have.
5. **Given** new agent profile (e.g. Sami) declares the same set, **When** any of the
   above scenarios run against Sami, **Then** enforcement is identical (no
   Nadia-specific code paths).

## Tests

### E2E

`tests/e2e/capability/profile_capabilities_test.go`:

- `TestProfileCapabilities_DraftAllowed`
- `TestProfileCapabilities_SendDirectRejected`
- `TestProfileCapabilities_NotifyFounderWhitelist`
- `TestProfileCapabilities_UnknownCapability`
- `TestProfileCapabilities_PortableAcrossProfiles`
