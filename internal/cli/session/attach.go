package session

import (
	"fmt"

	"github.com/spf13/cobra"
	"oss-aps-cli/internal/core/session"
)

func NewAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session-id>",
		Short: "Attach to a running session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			registry := session.GetRegistry()

			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("session not found: %w", err)
			}

			fmt.Printf("Attaching to session %s (PID: %d)\n", sessionID, sess.PID)
			fmt.Println("Attach functionality not yet implemented")

			return nil
		},
	}
}
