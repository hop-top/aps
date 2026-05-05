---
status: shipped
---

# 060 - All aps Lists Render with Styled Tables

**ID**: 060
**Feature**: CLI Conventions §6.1 (Output formatting)
**Persona**: [User](../personas/user.md)
**Priority**: P2
**Status**: shipped
**Author**: jadb
**Track**: aps-tabwriter-sweep

## Story

As an aps user reading list output in a terminal, I want every
remaining tabular command (`aps workspace activity`, `aps device
presence`, `aps device pending`, `aps device channels`, `aps squad
check`) to render with theme-aware borders and header emphasis on
TTY, while keeping plain tabwriter output on pipes / files / CI
logs. So scanning state in the terminal is faster without breaking
`mytool list | jq` or golden-file diffs.

This completes the 2026-05-04 kit-integration audit's §1.1 scorecard
⚠️ row for "Tabular/JSON/YAML render". Story 059 migrated the 5
audit-cited callsites; this story sweeps the 5 additional sites the
Wave 2 agent flagged as scope creep:

  internal/cli/workspace/activity.go:114
  internal/cli/adapter/presence_cmd.go:101
  internal/cli/adapter/pending.go:58
  internal/cli/adapter/channels.go:121
  internal/cli/squad/check.go:33

After this lands, `grep -rn "tabwriter.NewWriter" internal/ cmd/`
returns zero — every tabular list in aps routes through
`listing.RenderList` and inherits the kit-themed styled renderer on
TTY writers.

## Acceptance Scenarios

1. **Given** I run `aps workspace activity <ws>` (or the four
   sibling commands) in a TTY, **When** stdout is a terminal,
   **Then** the output emits ANSI color escapes plus lipgloss
   box-drawing characters and rows are colored per the active
   kit/cli theme.

2. **Given** I run `aps workspace activity <ws> | cat` (pipe),
   **When** stdout is not a terminal, **Then** the output is plain
   tabwriter text with no ANSI escapes and no box-drawing runes —
   byte-stable for golden-file tests and downstream tools.

3. **Given** the same data, **When** I compare TTY output (after
   stripping ANSI) to non-TTY output, **Then** every row name and
   header label appears in both — visual chrome differs but cell
   text is identical.

4. **Given** I run `aps device pending --json`, **When** the
   formatter resolves, **Then** the styled path is bypassed and the
   structured field set is unchanged from before this change
   (`device_id`, `profile_id`, `device_name`, `device_os`,
   `requested_at` — exactly the keys the prior inline
   `pendingDevice` shape emitted).

5. **Given** I run `aps device presence --json`, **When** the
   formatter resolves, **Then** numeric fields (`sync_lag`,
   `offline_queue`) stay ints in the JSON — the suffixed strings
   ("3 events", "1 pending") only appear in the table path's
   `presenceTableRow` projection.

6. **Given** any of the migrated commands, **When** I run
   `grep -rn "tabwriter.NewWriter" internal/ cmd/`, **Then** zero
   matches return — the remaining tabwriter callsites are in
   third-party deps + the Wave 2 / Wave 3 archived implementations.

## Implementation Notes

- One commit per migrated callsite (so each commit builds + tests
  green individually). Conventional Commits, `refactor(cli/<pkg>)`
  scope, no co-author trailers.
- Pattern (per Wave 2 commit 970cbdc): typed row struct with
  `table:"COLNAME,priority=N"` + `json:"…"` + `yaml:"…"` tags,
  replace tabwriter block with
  `listing.RenderList(os.Stdout, output.Table, rows)`.
- `presence_cmd.go` introduces TWO row types — `presenceRow` for
  json/yaml (ints preserved) and `presenceTableRow` for the table
  (ints projected to suffixed strings). Same split for `pending.go`
  with `pendingTableRow` + `pendingJSONRow` so the existing JSON
  field set stays exactly stable.
- `squad/check.go` drops a hand-rolled `─── ─── ───` separator row
  — kit/output draws its own border on TTY, and the plain
  tabwriter renderer aligns columns without a manual divider.
- `internal/cli/adapter/styles.go` drops `tableHeader =
  lipgloss.NewStyle()…` (last consumer migrated) and the
  `charm.land/lipgloss/v2` import. Mirrors the Wave 2 cleanup in
  `policy/cmd.go`, `migrate/cmd.go`, and `workspace/helpers.go`.

## Tests

### E2E
- Reuses `tests/e2e/profile/profile_list_styled_test.go` from Wave
  2 (T-0457 in story 059) — the listing wrapper's TTY behavior is
  already e2e'd against a creack/pty pseudo-terminal pair. Per-
  callsite TTY tests would duplicate that contract.

### Unit (golden-style)
- `internal/cli/workspace/activity_test.go` — `activityRow`
  shape + JSON round-trip.
- `internal/cli/adapter/styled_tables_test.go` — `presenceRow`,
  `presenceTableRow`, `pendingTableRow`, `pendingJSONRow`,
  `channelRow` shapes + JSON round-trips, with the JSON field-set
  guard for `pendingJSONRow`.
- `internal/cli/squad/check_test.go` — `checkRow` shape + JSON
  round-trip.

## Dependencies

- Story 059 (`kit-styled-table-rollout`, T-0456) — the
  `listing.SetTableStyle` wiring + `kit/output.WithTableStyle`
  primitive that this story consumes.
- `kit/go/console/output.{TableStyle, WithTableStyle}` (kit
  ce64de3, T-0252) — already pinned via Wave 2.
