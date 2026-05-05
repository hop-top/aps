---
status: shipped
---

# 059 - Styled Tables on TTY

**ID**: 059
**Feature**: CLI Conventions §6.1 (Output formatting)
**Persona**: [User](../personas/user.md)
**Priority**: P2
**Status**: shipped
**Author**: jadb
**Track**: kit-styled-table-rollout

## Story

As an aps user reading list output in a terminal, I want every
`aps <noun> list` command and the migrated non-list tables (profile
trust, session inspect, policy list, workspace audit/members/tasks/
agents/ctx history, migrate dry-run) to render with theme-aware
borders and header emphasis when stdout is a TTY, while keeping
plain tabwriter output on pipes / files / CI logs. So scanning state
in the terminal is faster without breaking `mytool list | jq` or
golden-file diffs.

Convention parity with `gh repo list`, `kubectl get`, `tlc track list`,
and the kit upstream contract (`output.WithTableStyle`,
`cli.Root.TableStyle()`) shipped in kit/main commits ce64de3
(T-0252), 1a1834e (T-0253), 9babcd7 (T-0254). Aps was the first
adopter once those primitives landed.

## Acceptance Scenarios

1. **Given** I run `aps profile list` in a TTY, **When** stdout is
   a terminal, **Then** the output emits ANSI color escapes plus
   lipgloss box-drawing characters (`┌`, `─`, `│`) and rows are
   colored per the active kit/cli theme.

2. **Given** I run `aps profile list | cat` (pipe), **When** stdout
   is not a terminal, **Then** the output is plain tabwriter text
   with no ANSI escapes and no box-drawing runes — byte-stable for
   golden-file tests and downstream tools.

3. **Given** the same data, **When** I compare TTY output (after
   stripping ANSI) to non-TTY output, **Then** every row name and
   header label appears in both — visual chrome differs but cell
   text is identical.

4. **Given** any of the migrated non-list commands (`aps profile
   trust`, `aps workspace tasks`, `aps policy list`, `aps session
   inspect`, `aps migrate messengers --dry-run`, …), **When** I run
   it on a TTY, **Then** the table inherits the same styled renderer
   without per-callsite configuration.

5. **Given** I run any list command with `--format json` or
   `--format yaml`, **When** the formatter resolves, **Then** the
   styled path is bypassed entirely (kit/output gates `WithTableStyle`
   on the table format and TTY writer) and structured output is
   unchanged from before this change.

## Implementation Notes

- `internal/cli/listing.SetTableStyle(output.TableStyle)` installs a
  default style that `RenderList` forwards via
  `output.WithTableStyle`. Wired once during `internal/cli/root.go`
  init from `kitcli.Root.TableStyle()`.
- Five hand-rolled `tabwriter.NewWriter` callsites (audit
  `~/.ops/reviews/aps-kit-integration-audit-2026-05-04.md` §3) were
  migrated to typed rows + `listing.RenderList`; the shared
  `workspace/helpers.go:newTabWriter` factory and the `tableHeader`
  lipgloss vars in `policy/cmd.go`, `session/list.go`,
  `migrate/cmd.go`, and `workspace/helpers.go` were removed.
- Styled path is gated on `writerIsTTY` in kit/output (an
  `*os.File` + `isatty.IsTerminal` check) — non-TTY callers never
  see ANSI or box-drawing runes.
- Out of scope: migrating other tabwriter callsites in
  `internal/cli/{adapter,squad,workspace/activity}.go`. Tracked
  separately to keep this rollout scoped to the audit's 5 sites.
- Out of scope: removing `tlc/internal/cli/ttytable.go` in favor
  of the kit primitive. Filed as a follow-up tlc track via T-0455.

## Tests

### E2E
- `tests/e2e/profile/profile_list_styled_test.go`
  - `TestProfileList_Styled_TTYAndNonTTYContentIdentity` — runs
    `aps profile list` against a creack/pty pseudo-terminal pair,
    asserts ANSI + box-drawing on TTY output, plain text on
    non-TTY, and content identity after stripping ANSI.

### Unit (golden-style)
- `internal/cli/policy/list_test.go` — `policyRow` non-TTY shape.
- `internal/cli/migrate/cmd_test.go` — `migrateDryRunRow` shape.
- `internal/cli/session/inspect_test.go` — `sessionPropertyRow` +
  `sessionEnvRow` shapes.
- `internal/cli/profile_trust_test.go` — `trustScoreRow` +
  `trustHistoryRow` shapes.
- `internal/cli/workspace/styled_tables_test.go` — five workspace
  row types (`memberRow`, `taskRow`, `agentMatchRow`, `auditRow`,
  `ctxHistoryRow`).
- `internal/cli/listing/render_test.go` — added
  `TestRenderList_StyledNonTTY_FallsThroughToPlain` and
  `TestSetTableStyle_Replaces`.

## Dependencies

- `kit/go/console/output.{TableStyle, WithTableStyle, RowEmphasis,
  EmphasisKind}` (kit ce64de3, T-0252)
- `kit/go/console/cli.Root.TableStyle()` (kit 1a1834e, T-0253)
- `github.com/creack/pty` (promoted to direct dep for the TTY e2e)
- US-0007 in kit (Wave 0 story; this is the aps-side companion)
