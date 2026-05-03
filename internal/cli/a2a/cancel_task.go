package a2a

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
	"hop.top/aps/internal/cli/globals"
)

func NewCancelTaskCmd() *cobra.Command {
	var targetProfile string

	cmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a running A2A task",
		Long:  `Cancel a running A2A task on a target profile.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0411 — A2A cancel is a network call to the target peer.
			if globals.IsOffline() {
				return fmt.Errorf("a2a tasks cancel: %w", globals.ErrOffline)
			}

			ctx := context.Background()
			taskID := a2a.TaskID(args[0])

			targetProf, err := loadProfile(targetProfile)
			if err != nil {
				return err
			}

			client, err := a2apkg.NewClient(targetProfile, targetProf)
			if err != nil {
				return fmt.Errorf("failed to create A2A client: %w", err)
			}

			if err := client.CancelTask(ctx, taskID); err != nil {
				return fmt.Errorf("failed to cancel task: %w", err)
			}

			fmt.Printf("Task %s cancelled successfully\n", taskID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID (required)")
	cmd.MarkFlagRequired("target")

	return cmd
}
