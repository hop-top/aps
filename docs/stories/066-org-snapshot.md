---
status: paper
---

# 066 - Org Snapshot

**ID**: 066
**Feature**: Org Hierarchy
**Persona**: [User](../personas/user.md)
**Priority**: P2

## Story

As an aps user, I want to view and export the reporting structure my
profiles declare (story 065) so I can see who reports to whom, share
an org chart, and feed the structure to other tools.

```sh
aps org show <profile> [--depth N]
aps org snapshot [--all | --root <profile-id> | --squad <id>]
aps org snapshot --snapshot-format json|yaml|tree|mermaid
aps org snapshot --output org.mmd
```

## `aps org show <profile>`

Single-profile view:

- **Management chain up** — the profile's supervisors to the root.
- **Reports down** — direct and transitive reports.
- `--depth N` limits how many levels of reports are expanded.
- A channel summary column per row, derived from each profile's
  config: which of `a2a` / `acp` / `email` / `messengers` /
  `webhooks` the profile has enabled.

## `aps org snapshot`

Whole-org (or subtree) export. Scopes:

- `--all` (default) — every profile on disk.
- `--root <profile-id>` — the subtree rooted at the given profile.
- `--squad <id>` — profiles whose on-disk `Profile.Squads` membership
  includes the squad.

## Output formats

`--snapshot-format json|yaml|tree|mermaid` (default `tree`). This is a
local flag, deliberately distinct from kit's global `--format` — the
snapshot formats (`tree`, `mermaid`) are not generic output encodings.

- **json / yaml** — carry `nodes` (`id`, `display_name`, `type`,
  `channels`, `squads`), `edges`, and `captured_at`.
- **tree / mermaid** — carry no timestamp, so identical org state
  produces byte-identical output.
- **mermaid** — a `flowchart TD`; one edge per reporting pair,
  `manager --> report`; node labels are display names, with a
  `(human)` suffix for `type: human` profiles; node ids are sanitized
  to mermaid-safe identifiers.

`--output <file>` writes atomically (temp file + rename); without it,
output goes to stdout.

All formats are deterministic: nodes and edges are emitted in sorted
order, so repeated runs over unchanged profiles diff clean.

## Errors

- `--root <unknown-id>` — clear error naming the missing profile.
- `--squad <id>` with zero members — clear error, not empty output.
- Cyclic on-disk data — reported as an error; traversal must never
  hang (run `aps org check`, story 065, to locate the cycle).

## Acceptance Scenarios

1. **Given** a profile with managers above and reports below, **When**
   `aps org show <profile>` is run, **Then** the output contains the
   management chain up to the root and the direct and transitive
   reports below.
2. **Given** a deep report tree, **When** `--depth 1` is passed,
   **Then** only direct reports are expanded.
3. **Given** profiles with differing protocol config, **When**
   `aps org show <profile>` is run, **Then** each row's channel
   column reflects that profile's enabled channels
   (a2a/acp/email/messengers/webhooks).
4. **Given** several profiles, **When** `aps org snapshot` is run
   with no scope flag, **Then** every profile on disk appears
   (`--all` default).
5. **Given** `--root <profile-id>`, **When** the snapshot is run,
   **Then** only the subtree rooted at that profile appears.
6. **Given** profiles with `Squads` membership on disk, **When**
   `--squad <id>` is passed, **Then** only members of that squad
   appear.
7. **Given** `--snapshot-format json` (and `yaml`), **When** the
   snapshot is run, **Then** the document contains `nodes` with
   `id`, `display_name`, `type`, `channels`, `squads`, plus `edges`
   and `captured_at`.
8. **Given** `--snapshot-format tree` or `mermaid`, **When** the
   snapshot is run twice on unchanged profiles, **Then** the outputs
   are byte-identical (no timestamp).
9. **Given** `--snapshot-format mermaid`, **When** the snapshot is
   run, **Then** the output is a `flowchart TD` with
   `manager --> report` edges, display-name labels, a `(human)`
   suffix on human nodes, and sanitized node ids.
10. **Given** `--output <file>`, **When** the snapshot is run,
    **Then** the file is written atomically and contains the same
    bytes stdout would have carried.
11. **Given** the same org twice, **When** any format is emitted,
    **Then** nodes and edges are sorted and the outputs are stable.
12. **Given** `--root <unknown-id>`, **When** the snapshot is run,
    **Then** the command fails naming the missing profile.
13. **Given** `--squad <id>` with no members, **When** the snapshot
    is run, **Then** the command fails with a clear error.
14. **Given** cyclic `reports_to` data on disk, **When** `show` or
    `snapshot` is run, **Then** the command reports an error and
    terminates (never hangs).

## E2E Tests

- planned: `tests/e2e/org/org_show_test.go::TestOrgShow_ChainAndReports`
- planned: `tests/e2e/org/org_show_test.go::TestOrgShow_DepthLimit`
- planned: `tests/e2e/org/org_show_test.go::TestOrgShow_ChannelSummaryColumn`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_AllDefaultScope`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_RootScope`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_SquadScope`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_JSONNodesEdgesCapturedAt`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_YAMLNodesEdgesCapturedAt`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_TreeAndMermaidNoTimestamp`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_MermaidFlowchartShape`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_OutputFileAtomicWrite`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_DeterministicSortedOutput`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_UnknownRootFails`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_EmptySquadFails`
- planned: `tests/e2e/org/org_snapshot_test.go::TestOrgSnapshot_CyclicDataFailsWithoutHang`

## Dependencies

- Story 065 — `reports_to` + `type` fields and `aps org check`.
- Story 001 — profile management (profiles on disk).
