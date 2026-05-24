package adapter

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// newPermissionsCmd returns the `adapter permissions` mid-level command
// grouping device-permission operations (set).
func newPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Manage device permissions in a workspace",
	}
	// T-0648 — permissions is an intermediate grouping node (depth 2)
	// with a depth-3 `set` leaf underneath; required for the depth-
	// hierarchical signature check.
	kitcli.SetHierarchical(cmd)
	cmd.AddCommand(newSetPermissionsCmd())
	return cmd
}
