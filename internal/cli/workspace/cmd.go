package workspace

import (
	"github.com/spf13/cobra"
)

// NewWorkspaceCmd creates the workspace command group.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
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
	cmd.AddCommand(NewUnarchiveCmd())
	cmd.AddCommand(NewDeleteCmd())

	// Multi-device workspace commands (Plan 7)
	cmd.AddCommand(NewActivityCmd())
	cmd.AddCommand(NewSyncCmd())

	return cmd
}
