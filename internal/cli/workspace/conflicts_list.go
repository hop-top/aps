package workspace

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core/multidevice"
)

// conflictSummaryRow is the per-conflict row rendered by
// `aps workspace conflicts list`. Field tags drive kit/output table
// columns (priority hints survive narrow terminals) plus json/yaml.
type conflictSummaryRow struct {
	ID         string `table:"ID,priority=10" json:"id" yaml:"id"`
	Workspace  string `table:"WORKSPACE,priority=8" json:"workspace" yaml:"workspace"`
	Type       string `table:"TYPE,priority=7" json:"type" yaml:"type"`
	Status     string `table:"STATUS,priority=9" json:"status" yaml:"status"`
	Resource   string `table:"RESOURCE,priority=5" json:"resource" yaml:"resource"`
	Devices    int    `table:"DEVICES,priority=3" json:"devices" yaml:"devices"`
	DetectedAt string `table:"DETECTED,priority=4" json:"detected_at" yaml:"detected_at"`
}

func newConflictsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspace conflicts",
		Long: `List conflicts detected in a workspace.

By default, all detected conflicts are listed (pending, auto-resolved
awaiting review, manual, and fully resolved). Use --unresolved to
narrow to only conflicts still requiring attention.

The --workspace flag is a global (T-0376) and inherits from the active
workspace when not supplied.`,
		RunE: runConflictsList,
	}

	cmd.Flags().Bool("unresolved", false,
		"Show only conflicts not yet fully resolved")

	return cmd
}

func runConflictsList(cmd *cobra.Command, args []string) error {
	wsID, err := resolveWorkspace(cmd, args)
	if err != nil {
		return err
	}

	mgr := multidevice.NewManager()
	// Always pass includeResolved=true to the manager and filter via
	// the listing predicate below — keeps the data path uniform and
	// gives callers consistent JSON/YAML output regardless of flag.
	conflicts, err := mgr.ListConflicts(wsID, true)
	if err != nil {
		return err
	}

	rows := buildConflictRows(conflicts)

	pred := listing.All(
		listing.BoolFlag(
			cmd.Flags().Changed("unresolved"),
			func(r conflictSummaryRow) bool {
				return r.Status != string(multidevice.ConflictResolved) &&
					r.Status != string(multidevice.ConflictAutoResolved)
			},
			mustBool(cmd, "unresolved"),
		),
	)

	rows = listing.Filter(rows, pred)

	return listing.RenderList(os.Stdout, globals.Format(), rows)
}

func buildConflictRows(conflicts []*multidevice.Conflict) []conflictSummaryRow {
	rows := make([]conflictSummaryRow, len(conflicts))
	for i, c := range conflicts {
		deviceSet := make(map[string]bool)
		for _, evt := range c.Events {
			deviceSet[evt.DeviceID] = true
		}
		rows[i] = conflictSummaryRow{
			ID:         c.ID,
			Workspace:  c.WorkspaceID,
			Type:       string(c.Type),
			Status:     string(c.Status),
			Resource:   c.Resource,
			Devices:    len(deviceSet),
			DetectedAt: c.DetectedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return rows
}

// mustBool reads a bool flag; the lookup never fails for flags
// declared on the command, so a panic here is a programming error.
func mustBool(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		panic(fmt.Sprintf("flag %q not declared as bool: %v", name, err))
	}
	return v
}
