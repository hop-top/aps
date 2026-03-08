package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/spf13/cobra"

	a2apkg "hop.top/aps/internal/a2a"
)

func NewSendTaskCmd() *cobra.Command {
	var (
		targetProfile string
		message       string
		taskID        string
		format        string
	)

	cmd := &cobra.Command{
		Use:   "send-task",
		Short: "Send a message to create or continue an A2A task",
		Long: `Send a message to create a new A2A task or continue an existing task.

Example:
  aps a2a send-task --target worker --message "Deploy application"
  aps a2a send-task --target worker --task-id <id> --message "Continue deployment"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			targetProf, err := loadProfile(targetProfile)
			if err != nil {
				return err
			}

			client, err := a2apkg.NewClient(targetProfile, targetProf)
			if err != nil {
				return fmt.Errorf("failed to create A2A client: %w", err)
			}

			msg := &a2a.Message{
				ID:   a2a.NewMessageID(),
				Role: a2a.MessageRoleUser,
				Parts: []a2a.Part{
					a2a.TextPart{Text: message},
				},
			}

			if taskID != "" {
				msg.TaskID = a2a.TaskID(taskID)
			}

			task, err := client.SendMessage(ctx, msg)
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(task)
			default:
				fmt.Printf("Task created/updated: %s\n", task.ID)
				fmt.Printf("Status: %s\n", task.Status.State)
				if len(task.History) > 0 {
					lastMsg := task.History[len(task.History)-1]
					fmt.Printf("Last message ID: %s\n", lastMsg.ID)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message text (required)")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Existing task ID (optional, creates new if not specified)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")
	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("message")

	return cmd
}
