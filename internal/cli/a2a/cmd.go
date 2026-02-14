package a2a

import (
	"github.com/spf13/cobra"
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

	cmd.AddCommand(NewListTasksCmd())
	cmd.AddCommand(NewGetTaskCmd())
	cmd.AddCommand(NewSendTaskCmd())
	cmd.AddCommand(NewSendStreamCmd())
	cmd.AddCommand(NewCancelTaskCmd())
	cmd.AddCommand(NewSubscribeTaskCmd())
	cmd.AddCommand(NewShowCardCmd())
	cmd.AddCommand(NewFetchCardCmd())
	cmd.AddCommand(NewServerCmd())
	cmd.AddCommand(NewToggleCmd())

	return cmd
}
