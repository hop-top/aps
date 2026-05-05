package session

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/session"
)

func NewDetachCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "detach [session-id]",
		Short: "Detach from a running session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := session.GetRegistry()

			if all {
				return detachAll(registry)
			}

			if len(args) == 0 {
				return fmt.Errorf("session ID required or use --all flag")
			}

			sessionID := args[0]
			return detachFromSession(registry, sessionID)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Detach from all sessions")
	clinote.AddFlag(cmd) // T-1291

	return cmd
}

func detachFromSession(registry *session.SessionRegistry, sessionID string) error {
	sess, err := registry.Get(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if sess.TmuxSocket == "" {
		return fmt.Errorf("session does not have a tmux socket")
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	if sess.Status != session.SessionActive {
		return fmt.Errorf("session %s is not active (status: %s)", sessionID, sess.Status)
	}

	fmt.Printf("Detaching from session %s\n", sessionID)

	cmd := exec.Command(tmuxPath, "-S", sess.TmuxSocket, "detach-client", "-s", sess.ID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to detach: %w", err)
	}

	return nil
}

func detachAll(registry *session.SessionRegistry) error {
	sessions := registry.List()
	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return nil
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	var detached int
	for _, sess := range sessions {
		if sess.Status == session.SessionActive && sess.TmuxSocket != "" {
			cmd := exec.Command(tmuxPath, "-S", sess.TmuxSocket, "detach-client", "-s", sess.ID)
			if err := cmd.Run(); err == nil {
				detached++
				fmt.Printf("Detached from session %s\n", sess.ID)
			}
		}
	}

	fmt.Printf("Detached from %d session(s)\n", detached)
	return nil
}
