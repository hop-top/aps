package acp

import (
	"github.com/spf13/cobra"
)

// NewACPCmd creates the acp command group
func NewACPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Manage ACP (Agent Client Protocol) server",
		Long: `Manage ACP (Agent Client Protocol) server for editor integrations.

The acp command group provides operations for:
- Starting an ACP server for a profile
- Managing ACP sessions
- Configuring ACP settings`,
	}

	cmd.AddCommand(NewServerCmd())

	return cmd
}
