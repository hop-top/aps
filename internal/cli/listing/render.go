package listing

import (
	"io"
	"sync"

	"hop.top/kit/go/console/output"
)

// tableStyleMu guards the package-level default TableStyle. Reads happen
// on every RenderList call; writes happen once during root init via
// SetTableStyle. The mutex keeps the contract race-free for tests that
// configure styles concurrently with rendering goroutines.
var (
	tableStyleMu  sync.RWMutex
	tableStyle    output.TableStyle
	tableStyleSet bool
)

// SetTableStyle installs the default TableStyle that RenderList forwards
// to output.Render via output.WithTableStyle. Intended to be called once
// during CLI root init from internal/cli with kitcli.Root.TableStyle().
//
// kit/output gates the styled renderer on a TTY writer, so passing a
// style here is safe for tests that pipe RenderList into a bytes.Buffer:
// the styled path falls back to the plain tabwriter renderer when the
// writer is not a *os.File terminal.
//
// Calling SetTableStyle a second time replaces the previous style.
func SetTableStyle(s output.TableStyle) {
	tableStyleMu.Lock()
	tableStyle = s
	tableStyleSet = true
	tableStyleMu.Unlock()
}

// activeTableStyle returns the installed style and whether one was set.
// Internal helper so render.go does not lock directly.
func activeTableStyle() (output.TableStyle, bool) {
	tableStyleMu.RLock()
	defer tableStyleMu.RUnlock()
	return tableStyle, tableStyleSet
}

// RenderList writes rows as a list in the requested format.
//
// format is one of output.Table, output.JSON, output.YAML. An empty
// format defaults to output.Table. Unknown formats are forwarded to
// kit/output which returns a descriptive error listing the registered
// formats.
//
// rows is any slice — typically []SomeSummaryRow with kit/output
// table tags. The Table formatter inspects the row type's struct tags
// to derive headers and priorities; JSON and YAML serialize the slice
// directly using the json/yaml struct tags.
//
// Callers MUST pass an empty slice (not nil) when there are zero
// matches — kit/output's Table renderer prints headers + a hint line
// for empty slices, while nil produces a less-friendly error.
//
// When SetTableStyle has installed a default TableStyle, RenderList
// forwards it via output.WithTableStyle. The styled path activates only
// on TTY writers; non-TTY writers (pipes, files, bytes.Buffer in tests)
// keep emitting the plain tabwriter output, so structured callers see
// no behavior change.
func RenderList[T any](w io.Writer, format string, rows []T) error {
	if format == "" {
		format = output.Table
	}
	if rows == nil {
		rows = []T{}
	}
	if style, ok := activeTableStyle(); ok {
		// RenderList is a thin pass-through to kit/output so callers
		// see kit's typed errors directly; wrapping here would mask
		// the descriptive "unknown format" error that kit/output.Render
		// produces. wrapcheck.ignore-sigs in .golangci.yml exempts the
		// kit/output return; we don't need a //nolint here.
		return output.Render(w, format, rows, output.WithTableStyle(style))
	}
	return output.Render(w, format, rows)
}
