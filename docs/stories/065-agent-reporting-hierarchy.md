---
status: paper
---

# 065 - Agent Reporting Hierarchy

**ID**: 065
**Feature**: Org Hierarchy
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As an aps user running a fleet of agent profiles, I want each profile to
optionally declare who it reports to — another agent or a human manager —
so the org structure lives in the profiles themselves and can be
validated, queried, and visualized instead of existing only in my head.

```sh
aps profile create <id> --reports-to <manager-id>
aps profile edit <id> --reports-to <manager-id>
aps profile edit <id> --reports-to ""        # clear
aps profile create <id> --type human
aps org check
```

## `reports_to`

A profile MAY carry a `reports_to:` field naming the profile id of its
supervisor. The supervisor can be an agent or a human profile. The field
is optional; profiles without it are roots (or unattached).

Clearing is explicit: `--reports-to ""` removes the field. Omitting the
flag on `edit` leaves the current value untouched.

## Profile `type` discriminator

Human managers are representable as profiles via a new `type:` field:

- Allowed values: `agent` (default) | `human`.
- Write-time strict: `create`/`edit` reject any other value with an
  error listing the allowed values.
- Read-time tolerant: an on-disk profile with an unknown `type:` still
  loads for `list`/`show`; `aps org check` reports it (see below).
- `aps run <profile>` and session start reject a `type: human` profile
  with a clear error — humans are representable, not executable.

## Integrity — `aps org check`

Validates the reporting graph across all profiles on disk and reports:

- **Cycles** — with the full path (`a → b → c → a`).
- **Dangling refs** — `reports_to` naming a profile that does not exist.
- **Self-references** — a profile reporting to itself.
- **Unknown `type:` values** — anything other than `agent`/`human`.
- **Unloadable profile files** — yaml that fails to parse is reported,
  not silently skipped.

Exit 0 on a clean org; non-zero when any finding is reported.

## Events

Changing `reports_to` publishes the existing `aps.profile.updated`
event with `fields: ["reports_to"]`. No new event type.

## Delete safety

`aps profile delete <id>` warns when other profiles report to the
target, listing them, so the operator knows the delete creates
dangling refs.

## Acceptance Scenarios

1. **Given** two profiles, **When** `aps profile create <id>
   --reports-to <manager>` is run, **Then** the resulting yaml
   contains `reports_to: <manager>`.
2. **Given** an existing profile, **When** `aps profile edit <id>
   --reports-to <manager>` is run, **Then** `reports_to` is updated
   and an `aps.profile.updated` event is published with
   `fields: ["reports_to"]`.
3. **Given** a profile with `reports_to` set, **When**
   `--reports-to ""` is passed to `edit`, **Then** the field is
   removed from the yaml.
4. **Given** `--type human`, **When** `create` is run, **Then** the
   yaml contains `type: human`.
5. **Given** `--type robot`, **When** `create` or `edit` is run,
   **Then** the command fails listing the allowed values
   `agent`/`human`.
6. **Given** an on-disk profile with `type: cyborg`, **When**
   `aps profile show <id>` is run, **Then** the profile loads
   (read-time tolerant).
7. **Given** a `type: human` profile, **When** `aps run <profile>`
   or a session start targets it, **Then** the command fails with a
   clear "human profiles are not executable" error.
8. **Given** `a → b → c → a` on disk, **When** `aps org check` is
   run, **Then** the cycle is reported with the full path and the
   command exits non-zero.
9. **Given** a `reports_to` naming a missing profile, **When**
   `aps org check` is run, **Then** the dangling ref is reported.
10. **Given** a profile reporting to itself, **When** `aps org check`
    is run, **Then** the self-reference is reported.
11. **Given** an on-disk profile with an unknown `type:`, **When**
    `aps org check` is run, **Then** the unknown value is reported.
12. **Given** an unparseable profile yaml, **When** `aps org check`
    is run, **Then** the file is reported as unloadable, not skipped.
13. **Given** a clean org, **When** `aps org check` is run, **Then**
    the command exits 0.
14. **Given** profiles reporting to `<id>`, **When**
    `aps profile delete <id>` is run, **Then** a warning lists the
    profiles left dangling.

## E2E Tests

- planned: `tests/e2e/profile/profile_reports_to_test.go::TestProfileCreate_ReportsTo`
- planned: `tests/e2e/profile/profile_reports_to_test.go::TestProfileEdit_ReportsToPublishesUpdatedEvent`
- planned: `tests/e2e/profile/profile_reports_to_test.go::TestProfileEdit_ClearReportsTo`
- planned: `tests/e2e/profile/profile_type_test.go::TestProfileCreate_TypeHuman`
- planned: `tests/e2e/profile/profile_type_test.go::TestProfileCreate_UnknownTypeRejected`
- planned: `tests/e2e/profile/profile_type_test.go::TestProfileShow_ToleratesUnknownTypeOnDisk`
- planned: `tests/e2e/profile/profile_type_test.go::TestRun_RejectsHumanProfile`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_DetectsCycle`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_DanglingReportsTo`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_SelfReference`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_UnknownTypeValue`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_UnloadableProfile`
- planned: `tests/e2e/org/org_check_test.go::TestOrgCheck_CleanOrgPasses`
- planned: `tests/e2e/profile/profile_delete_test.go::TestProfileDelete_WarnsWhenReportedTo`

## Dependencies

- Story 001 — profile management (create/edit/delete/show).
- Story 002 — command execution (`aps run` gate for `type: human`).
- Story 066 — org snapshot consumes the graph this story defines.
