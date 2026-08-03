package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"hop.top/aps/internal/core/cmdrun"
	"hop.top/aps/internal/core/session"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/core/uxp/invoke"
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
pane plus escape sequences. --follow re-attaches pipe-pane so new
output streams as it lands.

--timestamps has no effect on tmux sessions: capture-pane exposes
no timestamp option. The flag is accepted (it applies to
container-backed sessions) and a warning is printed.

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

			if timestamps && !tmuxSupportsTimestamps() {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
					"warn: --timestamps is not supported for tmux sessions "+
						"(capture-pane has no timestamp option); ignoring")
			}

			return captureTmuxLogs(cmd.Context(), cmdrun.Exec(),
				sess, follow, tail, timestamps)
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

// tmuxCaptureSpec builds the `tmux capture-pane` command line.
//
// Socket selection (-S <path>) is a server option and must precede the
// subcommand, as `tmux -S <sock> capture-pane ...`. Inside
// capture-pane, -S means start-line and -t means target-pane, so the
// socket cannot be passed after the verb.
//
// tail selects the line range: "all" captures the whole history
// (-S -), a number captures that many lines back from the end, and
// empty captures the visible pane with escape sequences (-e).
func tmuxCaptureSpec(socket, target, tail string, timestamps bool) invoke.CommandSpec {
	args := []string{"-S", socket, "capture-pane", "-p", "-t", target}

	switch {
	case tail == "all":
		// -S - is capture-pane's "start of history".
		args = append(args, "-S", "-")
	case tail != "":
		// N lines back from the visible pane's start.
		args = append(args, "-S", "-"+tail)
	default:
		args = append(args, "-e")
	}

	// timestamps is intentionally not mapped to a flag.
	//
	// capture-pane has no timestamp option (tmux 3.7b: -aeFHLpPqCJMN,
	// -b, -E, -S, -t). The previous implementation appended "-t" for
	// it, which is target-pane — it consumed the following "-S" as its
	// target and the socket was never applied. Silently emitting some
	// other flag would repeat that class of bug, so the flag is
	// accepted and reported as unsupported by the caller instead.
	_ = timestamps

	return invoke.CommandSpec{Path: "tmux", Args: args}
}

// tmuxSupportsTimestamps reports whether the tmux capture path can
// honour --timestamps. It cannot; the caller warns rather than
// silently dropping the operator's flag.
func tmuxSupportsTimestamps() bool { return false }

// tmuxPipePaneSpec builds the follow-mode command line.
func tmuxPipePaneSpec(socket, target string) invoke.CommandSpec {
	return invoke.CommandSpec{
		Path: "tmux",
		Args: []string{"-S", socket, "pipe-pane", "-t", target, "cat"},
	}
}

func captureTmuxLogs(
	ctx context.Context,
	runner cmdrun.Runner,
	sess *session.SessionInfo,
	follow bool,
	tail string,
	timestamps bool,
) error {
	spec := tmuxCaptureSpec(sess.TmuxSocket, sess.ID, tail, timestamps)
	res, err := runner.Run(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to capture tmux logs: %w", err)
	}
	_, _ = os.Stdout.Write(res.Stdout)
	_, _ = os.Stderr.Write(res.Stderr)
	if res.Code != 0 {
		return fmt.Errorf("failed to capture tmux logs: tmux exited %d: %s",
			res.Code, strings.TrimSpace(string(res.Stderr)))
	}

	if follow {
		followRes, err := runner.Run(ctx, tmuxPipePaneSpec(sess.TmuxSocket, sess.ID))
		if err != nil {
			return fmt.Errorf("failed to follow tmux logs: %w", err)
		}
		_, _ = os.Stdout.Write(followRes.Stdout)
		_, _ = os.Stderr.Write(followRes.Stderr)
		if followRes.Code != 0 {
			return fmt.Errorf("failed to follow tmux logs: tmux exited %d: %s",
				followRes.Code, strings.TrimSpace(string(followRes.Stderr)))
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
