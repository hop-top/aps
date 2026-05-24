package session

import (
	"fmt"
	"os"
	"os/exec"

	"hop.top/aps/internal/core/session"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <session-id>",
		Short: "Show session logs (tmux capture)",
		Long: `Capture the tmux scrollback buffer for a session and write it
to stdout. The session must have a tmux socket recorded in the
registry; the command shells out to tmux capture-pane against
that socket. --tail "all" dumps the entire buffer, --tail <N>
limits to the last N lines, and the default captures the visible
pane plus escape sequences. --timestamps prepends tmux timestamp
metadata, and --follow re-attaches pipe-pane so new output
streams as it lands.

Read-only: no state mutation on the session or the buffer.
Idempotent on the same buffer state. Pair with aps session attach
when you want an interactive terminal rather than a one-shot
capture.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			registry := session.GetRegistry()
			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}

			if sess.TmuxSocket == "" {
				return fmt.Errorf("session %s does not have a tmux socket", sessionID)
			}

			follow, _ := cmd.Flags().GetBool("follow")
			tail, _ := cmd.Flags().GetString("tail")
			timestamps, _ := cmd.Flags().GetBool("timestamps")

			return captureTmuxLogs(sess, follow, tail, timestamps)
		},
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().String("tail", "", "Number of lines to show from the end (\"all\" for entire buffer)")
	cmd.Flags().Bool("timestamps", false, "Show timestamps")

	// T-0648 — kit 0.4 signature annotations. logs is read-only (it
	// drives tmux capture-pane to read the existing buffer).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)

	return cmd
}

func captureTmuxLogs(sess *session.SessionInfo, follow bool, tail string, timestamps bool) error {
	args := []string{"capture-pane", "-p"}

	if timestamps {
		args = append(args, "-t")
	}

	if tail == "all" {
		args = append(args, "-S", sess.TmuxSocket, "-t", sess.ID)
	} else if tail != "" {
		args = append(args, "-S", sess.TmuxSocket, "-t", sess.ID, "-E", tail)
	} else {
		args = append(args, "-S", sess.TmuxSocket, "-t", sess.ID, "-e")
	}

	cmd := exec.Command("tmux", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to capture tmux logs: %w", err)
	}

	if follow {
		args = []string{"-S", sess.TmuxSocket, "pipe-pane", "-t", sess.ID, "cat"}

		cmd := exec.Command("tmux", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to follow tmux logs: %w", err)
		}
	}

	return nil
}

func captureContainerLogs(sess *session.SessionInfo, follow bool, tail string, timestamps bool) error {
	if sess.ContainerID == "" {
		return fmt.Errorf("session does not have a container ID")
	}

	args := []string{"logs"}

	if follow {
		args = append(args, "-f")
	}

	if tail == "all" {
		args = append(args, "--tail", "all")
	} else if tail != "" {
		args = append(args, "--tail", tail)
	}

	if timestamps {
		args = append(args, "--timestamps")
	}

	args = append(args, sess.ContainerID)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to capture container logs: %w", err)
	}

	return nil
}

func attachToTmux(sess *session.SessionInfo, mode string) error {
	args := []string{"-S", sess.TmuxSocket}

	if mode == "view" {
		args = append(args, "attach", "-t", sess.ID, "-r")
	} else {
		args = append(args, "attach", "-t", sess.ID)
	}

	cmd := exec.Command("tmux", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func detachFromTmux(sess *session.SessionInfo) error {
	cmd := exec.Command("tmux", "-S", sess.TmuxSocket, "detach", "-t", sess.ID)
	return cmd.Run()
}
