package a2a

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
)

func NewSubscribeTaskCmd() *cobra.Command {
	var (
		targetProfile string
		webhookURL    string
	)

	cmd := &cobra.Command{
		Use:   "subscribe-task <task-id>",
		Short: "Subscribe to push notifications for an A2A task",
		Long: `Subscribe to push notifications for task updates via webhook.

Example:
  aps a2a subscribe-task <task-id> --target worker --webhook http://localhost:9000/hook`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if err := client.SubscribeToTask(ctx, taskID, webhookURL); err != nil {
				return fmt.Errorf("failed to subscribe to task: %w", err)
			}

			fmt.Printf("Subscribed to task %s\n", taskID)
			fmt.Printf("Webhook URL: %s\n", webhookURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID (required)")
	cmd.Flags().StringVar(&webhookURL, "webhook", "", "Webhook URL for push notifications (required)")
	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("webhook")

	return cmd
}
