# Styled tables audit

Audit of `kit/output` styled-table adoption across the aps CLI. Reference:
`~/.ops/docs/kit-conventions/reference/console-output.md`.

The integration is anchored at `internal/cli/listing`, a thin wrapper that
forwards every list command to `kit/output.Render` and threads a
process-wide `output.TableStyle` via `output.WithTableStyle` on TTY writers.
The styled path is gated on `writerIsTTY` inside kit, so non-TTY callers
(pipes, files, test buffers) keep emitting plain `tabwriter` output and
structured callers see no behavior change.

## Architecture

The seam is a single function pair in `internal/cli/listing/render.go`:

| Symbol                                       | Purpose                                              |
|----------------------------------------------|------------------------------------------------------|
| `RenderList[T any](w, format, rows) error`  | Sole public render entry — dispatches to kit/output  |
| `SetTableStyle(s output.TableStyle)`         | Install process-wide style; called once at root init |
| `activeTableStyle() (TableStyle, bool)`      | Internal accessor under `sync.RWMutex`               |

Plus the filter primitives (`All`, `Any`, `Not`, `MatchString`,
`MatchSlice`, `BoolFlag`, `Filter`, `Predicate`) in `predicate.go`. The
package contains 5 Go files totalling ~620 LOC including tests.

`internal/cli/root.go:246` calls `listing.SetTableStyle(root.TableStyle())`
once during `init`, lifting colors from the active kit/cli theme.

## Callsite coverage

`grep -rn "hop.top/aps/internal/cli/listing" --include="*.go"` finds
**32 non-test importers** under `internal/cli/` with **41 `RenderList`
invocations**:

| Sub-package                | Files |
|----------------------------|-------|
| `internal/cli` (top-level) | 7     |
| `internal/cli/a2a`         | 1     |
| `internal/cli/adapter`     | 6     |
| `internal/cli/bundle`      | 1     |
| `internal/cli/capability`  | 2     |
| `internal/cli/migrate`     | 1     |
| `internal/cli/policy`      | 1     |
| `internal/cli/session`     | 2     |
| `internal/cli/skill`       | 1     |
| `internal/cli/squad`       | 2     |
| `internal/cli/workspace`   | 9     |

Every callsite is the shape `listing.RenderList(w, format, rows)` where
`format` is either `globals.Format()` (full table/json/yaml dispatch from
the `--format` global), an explicit `output.Table` (fixed-format tables
like `aps squad check` or `aps profile trust score`), or a local `format`
variable read from `cmd.Flags()`. No callsite passes a per-call
`WithTableStyle` or otherwise bypasses the package-level seam.

## Non-list `output.Render` usage — intentional

`grep "output\.Render\b" internal/cli/ --include="*.go"` finds one
non-listing callsite:

- `internal/cli/version.go:32` — `output.Render(os.Stdout, format, info)`
  on a single `version.Info` struct (not a row slice). The table path
  prints `info.String()` instead; JSON/YAML round-trip through
  kit/output. Outside the listing seam by design — `RenderList` is
  shaped for `[]T`.

## Legacy renderer survey

- `grep "text/tabwriter" internal/cli/` — **0 matches** (production).
- `grep "tabwriter\.NewWriter" internal/cli/` — **0 matches**.
- `grep "output\.Dispatch" internal/cli/` — **0 matches** (aps wires the
  `--format` global via `kit/console/cli.New`'s default registration in
  `root.go` and reads it through `globals.Format()`; the newer
  `output.Dispatch` flag-aware path is unused but available).

The dry-run sites flagged historically in `internal/cli/migrate/cmd.go`
and `internal/cli/adapter/{channels,pending,presence_cmd,links}.go` are
all on `listing.RenderList` (refs in source comments).

## kit/output upstream — features available, not adopted

`hop.top/kit@v0.4.0-alpha.4/go/console/output` ships beyond what aps
currently consumes. Surfaced for future work; **not adopted in this
audit** (the task is verification + closure):

| API                                | Status in aps | Notes                                                                                   |
|------------------------------------|---------------|-----------------------------------------------------------------------------------------|
| `output.WithTableStyle(s)`         | Adopted       | Threaded through `listing.RenderList` when a style is installed.                        |
| `output.SetDefaultTableStyle(s)`   | Not used      | Process-wide setter on kit's side. aps owns its own mutex in `listing` (intentional indirection per `doc.go`). |
| `output.RowEmphasis(i, kind)`      | Not used      | Per-row Primary/Secondary/Muted coloring. Candidate for `aps squad check` failure rows, `profile trust history` low-score rows, `workspace conflicts` pending rows. |
| `output.EmphasisKind`              | Not used      | Companion enum.                                                                         |
| `output.Dispatch(cmd, v, data)`    | Not used      | Flag-aware shim; aps reads `--format` via `globals.Format()` instead.                   |
| `output.DisableOutputFlag()`       | Not used      | Suppresses `--output`/`-o`; aps does not register that alias.                           |

Recommendation: a follow-up task may evaluate `RowEmphasis` adoption
for the three callsites above. None of these gaps blocks the migration —
the seam is wired and every list flows through it.

## Test coverage

- `internal/cli/listing/render_test.go` — table/JSON/YAML formats, empty
  slice, unknown format, styled-non-TTY fallthrough (no ANSI / no
  box-drawing leakage), `SetTableStyle` last-write-wins.
- `internal/cli/listing/predicate_test.go` — `All`/`Any`/`Not` identity,
  empty-arg semantics, nil predicate handling, `MatchString`/`MatchSlice`
  empty-want short-circuit, `BoolFlag` Changed gating, `Filter` copy
  semantics.
- Per-sub-package `styled_tables_test.go` (in `adapter`, `workspace`) +
  golden checks in `migrate`, `policy`, `profile_trust`, `session`,
  `squad`, `a2a`, `bundle`, `capability`, `contact`, `skill` — assert
  non-TTY output has no ANSI / no box-drawing runes.

## Verification

```
/usr/bin/env go build -buildvcs=false ./...       # clean
/usr/bin/env go test ./internal/cli/listing/...   # ok
/usr/bin/env go test ./internal/cli/...           # ok (all sub-packages)
go test -run TestRootValidate_StrictGatesPass     # PASS
```

## Conclusion

Migration is complete. All 32 list-producing CLI files route through
`internal/cli/listing.RenderList`, which carries the styled-table style
forward via `output.WithTableStyle` when the writer is a TTY. The one
non-listing `output.Render` callsite (`version.go`) is a single-struct
render and intentionally bypasses the slice-shaped seam. No gaps to
fix; kit/output's `RowEmphasis` / `SetDefaultTableStyle` /
`output.Dispatch` are available for future enhancement work.
