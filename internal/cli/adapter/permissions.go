package adapter

import "github.com/spf13/cobra"

// newPermissionsCmd returns the `adapter permissions` mid-level command
// grouping device-permission operations (set).
func newPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Manage device permissions in a workspace",
	}
	cmd.AddCommand(newSetPermissionsCmd())
	return cmd
}
