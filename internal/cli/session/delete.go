package session

import (
	"context"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"
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

			// T-1291 — attach --note to ctx BEFORE the registry
			// mutation so the SessionStopped event payload carries the
			// audit note and policy engines can read it from CEL.
			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			return deleteSession(ctx, sess)
		},
	}

	cmd.Flags().Bool("force", false, "Force delete without confirmation")
	clinote.AddFlag(cmd) // T-1291

	return cmd
}

func deleteSession(ctx context.Context, sess *session.SessionInfo) error {
	if sess.TmuxSocket != "" {
		if err := killTmuxSession(sess); err != nil {
			// Non-benign tmux errors are logged but do not abort
			// the delete — the session is going away in the
			// registry regardless.
			fmt.Fprintf(os.Stderr, "warning: tmux teardown: %v\n", err)
		}
	}

	registry := session.GetRegistry()
	_ = registry.UnregisterWithContext(ctx, sess.ID)

	fmt.Printf("Session %s deleted\n", sess.ID)
	return nil
}
