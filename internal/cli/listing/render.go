package listing

import (
	"io"

	"hop.top/kit/go/console/output"
)

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
func RenderList[T any](w io.Writer, format string, rows []T) error {
	if format == "" {
		format = output.Table
	}
	if rows == nil {
		rows = []T{}
	}
	return output.Render(w, format, rows)
}
