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
		Long: `Group device-permission operations against a workspace.
The single leaf under this parent is aps adapter permissions set,
which writes or inspects the permission bits a device holds in a
named workspace. Companion read is the device-level
aps adapter status / show.`,
	}
	// T-0648 — permissions is an intermediate grouping node (depth 2)
	// with a depth-3 `set` leaf underneath; required for the depth-
	// hierarchical signature check.
	kitcli.SetHierarchical(cmd)
	cmd.AddCommand(newSetPermissionsCmd())
	return cmd
}
