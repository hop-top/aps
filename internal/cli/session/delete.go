package session

import (
	"fmt"
	"os/exec"

	"hop.top/aps/internal/core/session"

	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <session-id>",
		Short: "Delete a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			registry := session.GetRegistry()

			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				fmt.Printf("Are you sure you want to delete session %s? [y/N]: ", sessionID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			return deleteSession(sess)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	return cmd
}

func deleteSession(sess *session.SessionInfo) error {
	if sess.TmuxSocket != "" {
		cmd := exec.Command("tmux", "-S", sess.TmuxSocket, "kill-server")
		_ = cmd.Run()
	}

	registry := session.GetRegistry()
	if err := registry.Unregister(sess.ID); err != nil {
		return fmt.Errorf("failed to unregister session: %w", err)
	}

	fmt.Printf("Session %s deleted\n", sess.ID)
	return nil
}
