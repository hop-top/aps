package acp

import (
	"github.com/spf13/cobra"
)

// NewACPCmd creates the acp command group.
//
// All leaves under `acp` sit at depth 2 (e.g. `aps acp server`), so
// no kit/hierarchical annotation is required on this node — the
// signature validator only enforces hierarchical markers for depth>=3
// chains. See go/console/cli/validate_signature.go.
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
	cmd.AddCommand(NewToggleCmd())

	return cmd
}
