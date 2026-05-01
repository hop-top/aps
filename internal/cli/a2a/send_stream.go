package a2a

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewSendStreamCmd() *cobra.Command {
	var (
		targetProfile string
		message       string
		taskID        string
	)

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Send a message with streaming updates (not yet supported)",
		Long:  `Send a message with streaming updates. This feature requires SDK support for streaming.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("send-stream not yet supported by a2a-go SDK v0.3.4")
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message text")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Existing task ID (optional)")

	return cmd
}
