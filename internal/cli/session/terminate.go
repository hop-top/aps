package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/cmdrun"
	"hop.top/aps/internal/core/session"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/progress"
	"hop.top/kit/go/core/uxp/invoke"
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
		Long: `Tear down a running session and its tmux server. The default
graceful path asks tmux to kill the session (which sends SIGHUP to
the inner shell), waits up to --timeout seconds (default 10) for
the inner PID to exit, and escalates to SIGKILL if the deadline
elapses. With --force the inner PID is SIGKILLed immediately and
tmux is killed without waiting.

Registry status is updated to inactive even when tmux teardown
returns warnings, so the session entry never strands in an
ambiguous in-between state. Pair with aps session delete to remove
the registry entry afterwards, or skip terminate entirely and call
aps session delete which tears down and unregisters in one step.

Destructive: in-flight work in the session is lost. The
destructive-token confirmation flow gates the apply path, and
--dry-run is opted out because preview would have to fake the OS
signal path that defines the operation. Idempotent on the registry
status — terminated sessions stay inactive. Use --note to attach an
audit reason that flows to the SessionStopped event payload.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			registry := session.GetRegistry()
			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("failed to get session: %w", err)
			}

			force, _ := cmd.Flags().GetBool("force")
			timeout, _ := cmd.Flags().GetInt("timeout")

			// T-1291 — attach --note to ctx BEFORE the registry status
			// update so the SessionStopped event payload carries the
			// audit reason and policy engines can read it from CEL.
			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd))
			return terminateSession(ctx, sess, force, timeout)
		},
	}

	cmd.Flags().Bool("force", false, "Force terminate without graceful shutdown")
	cmd.Flags().Int("timeout", 10, "Graceful shutdown timeout in seconds")
	clinote.AddFlag(cmd) // T-1291

	// T-0654 — terminate kills the tmux server and process tree;
	// in-flight work is lost. Idempotent on the registry status
	// (terminated sessions stay inactive).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	kitcli.SetDestructiveToken(cmd)
	// T-0656 — terminate sends SIGTERM and waits for the tmux server
	// to exit; preview would have to fake the OS signal path.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "terminate sends SIGTERM to the tmux server and waits for cleanup; previewing would have to fake the OS signal path that defines the operation."); err != nil {
		panic(err)
	}

	return cmd
}

func terminateSession(ctx context.Context, sess *session.SessionInfo, force bool, timeout int) error {
	fmt.Printf("Terminating session %s...\n", sess.ID)

	r := progress.FromContext(ctx)
	phase := "terminate"
	if force {
		phase = "kill"
	}
	r.Emit(ctx, progress.Event{Phase: phase, Item: sess.ID})

	var errs []error
	if force {
		errs = forceTeardown(sess)
	} else {
		errs = gracefulTeardown(sess, timeout)
	}

	// Always update status, even on partial failure above.
	registry := session.GetRegistry()
	if err := registry.UpdateStatusWithContext(ctx, sess.ID, session.SessionInactive); err != nil {
		errs = append(errs, fmt.Errorf("update session status: %w", err))
	}

	if len(errs) > 0 {
		okFalse := false
		r.Emit(ctx, progress.Event{Phase: phase, Item: sess.ID, OK: &okFalse})
		fmt.Printf("Session %s terminated with warnings\n", sess.ID)
		return errors.Join(errs...)
	}

	okTrue := true
	r.Emit(ctx, progress.Event{Phase: phase, Item: sess.ID, OK: &okTrue})
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
	return killTmuxSessionWith(ctx, cmdrun.Exec(), sess.TmuxSocket, name)
}

// tmuxKillSpec builds the `tmux kill-session` command line. The socket
// is a server option and precedes the subcommand.
func tmuxKillSpec(socket, name string) invoke.CommandSpec {
	return invoke.CommandSpec{
		Path: binTmux,
		Args: []string{"-S", socket, "kill-session", "-t", name},
	}
}

// killTmuxSessionWith is the runner-injected form, so tests can assert
// the command line and drive the benign-error classification without a
// live tmux server.
func killTmuxSessionWith(
	ctx context.Context,
	runner cmdrun.Runner,
	socket, name string,
) error {
	res, err := runner.Run(ctx, tmuxKillSpec(socket, name))
	if err != nil {
		return fmt.Errorf("tmux kill-session failed: %w", err)
	}
	if res.Code == 0 {
		return nil
	}
	// Tmux returns non-zero when the session/server is already gone.
	// That's a benign race — the session is definitely not running,
	// which is the desired end state.
	msg := string(res.Stderr)
	if session.IsBenignTmuxError(msg) {
		return nil
	}
	return fmt.Errorf("tmux kill-session failed: exit %d: %s",
		res.Code, strings.TrimSpace(msg))
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
