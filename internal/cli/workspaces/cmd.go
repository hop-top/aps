package workspaces

import (
	"github.com/spf13/cobra"
)

// NewWorkspacesCmd creates the workspaces command group.
func NewWorkspacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"ws"},
		Short:   "Manage workspaces",
		Long: `Manage workspaces for organizing agent work.

Workspaces provide logical groupings with configuration.
Profiles can be linked to workspaces for context awareness.`,
	}

	cmd.AddCommand(NewNewCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewLinkCmd())
	cmd.AddCommand(NewUnlinkCmd())
	cmd.AddCommand(NewArchiveCmd())
	cmd.AddCommand(NewDeleteCmd())

	return cmd
}
