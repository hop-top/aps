package capability

import (
	"os"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
)

// patternSummaryRow is the row shape for `aps capability patterns list`.
// Pattern data differs structurally from capabilities (no source/type),
// so we keep a distinct row type per task spec.
type patternSummaryRow struct {
	Tool        string `table:"TOOL,priority=9" json:"tool" yaml:"tool"`
	DefaultPath string `table:"DEFAULT PATH,priority=6" json:"default_path" yaml:"default_path"`
	Description string `table:"DESCRIPTION,priority=4" json:"description,omitempty" yaml:"description,omitempty"`
}

// newPatternsCmd builds the `capability patterns` noun-group.
// T-0396 — noun-list per CLI conventions §3.2: verbs under the noun.
func newPatternsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Smart patterns + builtin capabilities",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List smart patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			return runPatterns(format)
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func runPatterns(format string) error {
	rows := buildPatternRows()
	return listing.RenderList(os.Stdout, format, rows)
}

func buildPatternRows() []patternSummaryRow {
	patterns := capability.ListSmartPatterns()
	rows := make([]patternSummaryRow, 0, len(patterns))
	for _, p := range patterns {
		rows = append(rows, patternSummaryRow{
			Tool:        p.ToolName,
			DefaultPath: p.DefaultPath,
			Description: p.Description,
		})
	}
	return rows
}
