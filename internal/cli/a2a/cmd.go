package a2a

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// NewA2ACmd creates the a2a command group
func NewA2ACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "a2a",
		Short: "Manage A2A (Agent-to-Agent) protocol operations",
		Long: `Manage A2A (Agent-to-Agent) protocol operations for inter-profile communication.

The a2a command group provides operations for:
- Creating and managing tasks
- Sending messages between profiles
- Subscribing to task updates
- Managing agent cards
- Discovering other agents`,
	}

	// T-0648 — mark as intermediate grouping node so depth>=3 leaves
	// under `a2a tasks ...` / `a2a card ...` pass the
	// kit/hierarchical signature check.
	kitcli.SetHierarchical(cmd)

	cmd.AddCommand(NewTasksCmd())
	cmd.AddCommand(NewCardCmd())
	cmd.AddCommand(NewServerCmd())
	cmd.AddCommand(NewToggleCmd())

	return cmd
}
