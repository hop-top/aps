package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"hop.top/aps/internal/core/session"

	"github.com/spf13/cobra"
)

// tmuxKillTimeout bounds how long a tmux kill-session invocation can
// run. Tmux should respond near-instantly; anything longer is a sign
// tmux itself is wedged and we prefer to return an error than hang.
const tmuxKillTimeout = 5 * time.Second

// pollInterval is how often waitForProcessExit checks process liveness.
// Exposed as a package var so tests can shorten it.
var pollInterval = 100 * time.Millisecond

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

	cmd.Flags().Bool("force", false, "Force terminate without graceful shutdown")
	cmd.Flags().Int("timeout", 10, "Graceful shutdown timeout in seconds")

	return cmd
}

func terminateSession(sess *session.SessionInfo, force bool, timeout int) error {
	fmt.Printf("Terminating session %s...\n", sess.ID)

	var errs []error
	if force {
		errs = forceTeardown(sess)
	} else {
		errs = gracefulTeardown(sess, timeout)
	}

	// Always update status, even on partial failure above.
	registry := session.GetRegistry()
	if err := registry.UpdateStatus(sess.ID, session.SessionInactive); err != nil {
		errs = append(errs, fmt.Errorf("update session status: %w", err))
	}

	if len(errs) > 0 {
		fmt.Printf("Session %s terminated with warnings\n", sess.ID)
		return errors.Join(errs...)
	}

	fmt.Printf("Session %s terminated\n", sess.ID)
	return nil
}

// forceTeardown SIGKILLs the inner process immediately and then kills
// the tmux session. It does not wait for graceful exit.
func forceTeardown(sess *session.SessionInfo) []error {
	var errs []error
	if sess.PID > 0 {
		if err := terminateProcess(sess.PID, true); err != nil {
			errs = append(errs, fmt.Errorf("force kill process: %w", err))
		}
	}
	if sess.TmuxSocket != "" {
		if err := killTmuxSession(sess); err != nil {
			errs = append(errs, fmt.Errorf("kill tmux session: %w", err))
		}
	}
	return errs
}

// gracefulTeardown asks tmux to kill the session (sending HUP to the
// inner shell), waits up to `timeout` seconds for the PID to exit, and
// escalates to SIGKILL if it doesn't.
func gracefulTeardown(sess *session.SessionInfo, timeout int) []error {
	var errs []error
	if sess.TmuxSocket != "" {
		if err := killTmuxSession(sess); err != nil {
			// Non-fatal: tmux may already be gone. Record and
			// continue so we still update registry status.
			errs = append(errs, fmt.Errorf("kill tmux session: %w", err))
		}
	}
	if sess.PID > 0 {
		waitDuration := time.Duration(timeout) * time.Second
		if err := waitForProcessExit(sess.PID, waitDuration); err != nil {
			// Process didn't exit gracefully — escalate.
			if killErr := terminateProcess(sess.PID, true); killErr != nil {
				errs = append(errs, fmt.Errorf("escalate to SIGKILL: %w", killErr))
			}
		}
	}
	return errs
}

// terminateProcess sends a signal to the given pid. When force is true
// it sends SIGKILL, otherwise SIGINT. A pid <= 0 is a no-op.
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
		// If the process died between the last liveness probe and now,
		// the signal call surfaces ErrProcessDone or ESRCH. That's the
		// desired outcome — the process is gone — not a failure.
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("failed to send signal: %w", err)
	}

	return nil
}

// killTmuxSession runs `tmux -S <socket> kill-session -t <name>` to
// terminate a single tmux session without affecting other sessions on
// the same socket. The session name is taken from sess.TmuxSession when
// populated, otherwise from sess.Environment["tmux_session"], otherwise
// from sess.ID (the process backend uses the tmux session name as the
// SessionInfo ID).
func killTmuxSession(sess *session.SessionInfo) error {
	name := tmuxSessionName(sess)
	if name == "" {
		return fmt.Errorf("no tmux session name available for session %s", sess.ID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), tmuxKillTimeout)
	defer cancel()
	var stderr bytes.Buffer
	// #nosec G204 -- tmux args come from the session registry, not user input
	cmd := exec.CommandContext(ctx, "tmux", "-S", sess.TmuxSocket, "kill-session", "-t", name)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Tmux returns non-zero when the session/server is already
		// gone. That's a benign race — the session is definitely not
		// running, which is the desired end state.
		msg := stderr.String()
		if session.IsBenignTmuxError(msg) {
			return nil
		}
		return fmt.Errorf("tmux kill-session failed: %w: %s", err, strings.TrimSpace(msg))
	}
	return nil
}

// tmuxSessionName resolves the tmux session name for a SessionInfo,
// preferring the dedicated field and falling back to the environment
// map and finally the SessionInfo ID.
func tmuxSessionName(sess *session.SessionInfo) string {
	if sess.TmuxSession != "" {
		return sess.TmuxSession
	}
	if v, ok := sess.Environment["tmux_session"]; ok && v != "" {
		return v
	}
	return sess.ID
}

// waitForProcessExit polls process liveness until the process exits or
// the timeout elapses. Returns nil if the process exited within the
// timeout, or an error describing why it gave up.
//
// Liveness is tested by sending signal 0, which performs the kernel's
// permission check without actually delivering a signal. ESRCH means
// the process is gone.
func waitForProcessExit(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}
	if timeout <= 0 {
		if alive, _ := processAlive(pid); alive {
			return fmt.Errorf("process %d still alive (no wait requested)", pid)
		}
		return nil
	}

	deadline := time.Now().Add(timeout)
	for {
		alive, err := processAlive(pid)
		if err != nil {
			return err
		}
		if !alive {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("process %d still alive after %s", pid, timeout)
		}
		remaining := time.Until(deadline)
		sleep := pollInterval
		if remaining < sleep {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
}

// processAlive returns whether the given pid currently maps to a live
// process by sending it signal 0. Errors other than ESRCH (no such
// process) are surfaced to the caller.
func processAlive(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("failed to find process: %w", err)
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return false, nil
		}
		// EPERM means the process exists but we can't signal it; treat
		// as alive so the caller doesn't loop forever assuming exit.
		if errors.Is(err, syscall.EPERM) {
			return true, nil
		}
		return false, fmt.Errorf("liveness check failed: %w", err)
	}
	return true, nil
}
