---
status: paper
---

# 055 - Structured Progress on Long-Running Ops

**ID**: 055
**Feature**: CLI Core
**Persona**: [User](../personas/user.md)
**Priority**: P2
**Status**: paper
**Author**: jadb
**Tasks**: T-0462, T-0463

## Story

As an aps user (human or agent) running an operation that blocks for
more than two seconds, I want structured progress events on stderr so
that I can tell whether the tool is alive, what phase it is in, and
when it finished — without changing the data envelope on stdout.

Per `cli-conventions-with-kit.md` §6.5 (the §6.5 progress mandate),
long-running commands SHOULD emit progress events. Default rendering
is human-friendly stderr lines; agents opt into JSONL on stderr via
`--progress-format json` (or implicitly via `--format json`).

Today aps does not import `hop.top/kit/go/console/progress`. Several
commands flagged in
[docs/cli/progress-inventory.md](../cli/progress-inventory.md) block
for >2s in real use without any progress feedback.

## Scope

T-0463 wires the top 5 sites identified by the inventory:

1. `aps a2a tasks send`
2. `aps a2a card fetch`
3. `aps a2a tasks cancel`
4. `aps a2a tasks subscribe`
5. `aps run`

Out of scope for this story: device pairing UI (has bespoke QR
rendering), the long-running `aps listen` daemon (no completion
boundary), and the AGNTCY directory commands (stubs today, no
actual network round-trip).

## Acceptance Scenarios

1. **Given** `aps a2a tasks send --target peer --message hi` runs
   over a peer that takes 3s to respond, **When** stderr is
   captured, **Then** stderr contains at least one progress line
   per phase (`connect`, `send`, `ack`) before the command returns.

2. **Given** the same command with `--progress-format json`,
   **When** stderr is captured, **Then** every progress line is a
   single-line JSON object containing `phase`, `at` (RFC 3339 with
   timezone), and (for `ack`) an `ok` boolean.

3. **Given** the same command with `--quiet`, **When** stderr is
   captured, **Then** stderr contains no progress lines.

4. **Given** `aps a2a card fetch --url <slow>` with default flags
   on a TTY, **When** stderr is captured, **Then** stderr contains
   `[fetch]` and `[parse]` lines (Human reporter format).

5. **Given** any of the wired commands invoked over a non-TTY
   stderr (CI), **When** stderr is captured, **Then** progress
   lines are emitted but contain no ANSI cursor-control escape
   sequences (Human format is plain text; JSONL stays JSONL).

6. **Given** a network error on `aps a2a tasks send`, **When** the
   command exits non-zero, **Then** the final progress event has
   `OK: false` so agents reading the JSONL stream see the failure
   without parsing the error envelope.

7. **Given** `aps run <profile> -- <cmd>`, **When** the user
   subprocess runs, **Then** aps emits exactly two progress events
   (`exec` start, `exit` end) and does not interfere with the
   child's stdout/stderr.

## Implementation Notes

- kit/cli (`hop.top/kit/go/console/cli`) auto-wires the active
  `progress.Reporter` into `cmd.Context()` based on `--quiet`,
  `--progress-format`, and `--format`. Adopters call
  `progress.FromContext(cmd.Context()).Emit(ctx, event)`.
- aps already inherits `--quiet` and `--format` from kit/cli's
  default flag set; `--progress-format` is registered by kit/cli
  too. No new flag plumbing needed in aps.
- Phase names are lowercase nouns (`connect`, `send`, `ack`,
  `fetch`, `parse`, `cancel`, `register`, `exec`, `exit`). The
  closing event of a phase carries `OK *bool` for success/failure.
- Stdout stays purely the data envelope. Progress goes to stderr.

## Tests

### E2E

- `tests/e2e/progress_test.go`
  - `TestProgress_RunEmitsExecExit` — `aps run` emits exec + exit
  - `TestProgress_RunQuietSilent` — `--quiet` suppresses
  - `TestProgress_RunJSONLOnStderr` — `--progress-format json`
    emits valid JSONL with required fields
  - `TestProgress_RunNonTTYNoANSI` — non-TTY stderr is plain
  - `TestProgress_RunFailureEmitsOKFalse` — exit code !=0 →
    final event has `OK:false`

`aps run` is the test target because it is the only wired site
that produces deterministic progress without a peer (it shells
out to a known stub like `true` or `false`).

## Dependencies

- Builds on: kit/console/progress (already merged in kit).
- Related: [docs/cli/progress-inventory.md](../cli/progress-inventory.md)
  (T-0462 audit output).
