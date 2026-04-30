package session

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/prompt"
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
				confirmed, err := prompt.Confirm(
					fmt.Sprintf("Delete session %s?", sessionID))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			return deleteSession(sess)
		},
	}

	cmd.Flags().Bool("force", false, "Force delete without confirmation")

	return cmd
}

func deleteSession(sess *session.SessionInfo) error {
	if sess.TmuxSocket != "" {
		if err := killTmuxSession(sess); err != nil {
			// Non-benign tmux errors are logged but do not abort
			// the delete — the session is going away in the
			// registry regardless.
			fmt.Fprintf(os.Stderr, "warning: tmux teardown: %v\n", err)
		}
	}

	registry := session.GetRegistry()
	_ = registry.Unregister(sess.ID)

	fmt.Printf("Session %s deleted\n", sess.ID)
	return nil
}
