package a2a

import "github.com/spf13/cobra"

// NewTasksCmd returns the `a2a tasks` mid-level command grouping
// task-oriented operations (list, show, send, cancel, subscribe, stream).
func NewTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Manage A2A tasks",
		Long:  `List, inspect, send, cancel, and subscribe to A2A tasks.`,
	}

	cmd.AddCommand(NewListTasksCmd())
	cmd.AddCommand(NewGetTaskCmd())
	cmd.AddCommand(NewSendTaskCmd())
	cmd.AddCommand(NewCancelTaskCmd())
	cmd.AddCommand(NewSubscribeTaskCmd())
	cmd.AddCommand(NewSendStreamCmd())

	return cmd
}
