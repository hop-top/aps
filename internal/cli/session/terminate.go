package session

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"oss-aps-cli/internal/core/session"
)

func NewTerminateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terminate <session-id>",
		Short: "Terminate a session gracefully",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			registry := session.GetRegistry()
			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}

			force, _ := cmd.Flags().GetBool("force")
			timeout, _ := cmd.Flags().GetInt("timeout")

			return terminateSession(sess, force, timeout)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force terminate without graceful shutdown")
	cmd.Flags().Int("timeout", 10, "Graceful shutdown timeout in seconds")

	return cmd
}

func terminateSession(sess *session.SessionInfo, force bool, timeout int) error {
	fmt.Printf("Terminating session %s...\n", sess.ID)

	if sess.TmuxSocket != "" {
		if err := terminateTmuxSession(sess, force, timeout); err != nil {
			return fmt.Errorf("failed to terminate tmux session: %w", err)
		}
	}

	if sess.PID > 0 {
		if err := terminateProcess(sess.PID, force); err != nil {
			fmt.Printf("Warning: failed to terminate process: %v\n", err)
		}
	}

	registry := session.GetRegistry()
	if err := registry.UpdateStatus(sess.ID, session.SessionInactive); err != nil {
		fmt.Printf("Warning: failed to update session status: %v\n", err)
	}

	fmt.Printf("Session %s terminated\n", sess.ID)
	return nil
}

func terminateProcess(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}

	var signal os.Signal
	if force {
		signal = os.Kill
	} else {
		signal = os.Interrupt
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(signal); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	return nil
}
