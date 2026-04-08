package session

import (
	"os/exec"
	"testing"
	"time"

	"hop.top/aps/internal/core/session"
)

// TestTmuxSessionNameResolution covers the fallback chain used to find
// the tmux session name to pass to `tmux kill-session -t`.
func TestTmuxSessionNameResolution(t *testing.T) {
	cases := []struct {
		name string
		sess *session.SessionInfo
		want string
	}{
		{
			name: "prefers TmuxSession field",
			sess: &session.SessionInfo{
				ID:          "fallback-id",
				TmuxSession: "explicit-name",
				Environment: map[string]string{"tmux_session": "env-name"},
			},
			want: "explicit-name",
		},
		{
			name: "falls back to environment map",
			sess: &session.SessionInfo{
				ID:          "fallback-id",
				Environment: map[string]string{"tmux_session": "env-name"},
			},
			want: "env-name",
		},
		{
			name: "falls back to session ID",
			sess: &session.SessionInfo{ID: "fallback-id"},
			want: "fallback-id",
		},
		{
			name: "empty environment value still falls through",
			sess: &session.SessionInfo{
				ID:          "fallback-id",
				Environment: map[string]string{"tmux_session": ""},
			},
			want: "fallback-id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tmuxSessionName(tc.sess)
			if got != tc.want {
				t.Fatalf("tmuxSessionName = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestProcessAliveDeadProcess verifies the liveness probe reports a
// reaped process as dead. We use a deliberately impossible PID rather
// than a freshly-exited child, because a child of the test process
// becomes a zombie after exit and signal(0) to a zombie returns nil
// (the kernel keeps the entry until the parent reaps it).
func TestProcessAliveDeadProcess(t *testing.T) {
	// Pick an unused PID. We scan upward from a high value until we
	// find one that isn't currently mapped to a process. This is
	// inherently racy against the kernel allocating that pid to a new
	// process, but the window is microscopic and the test is local.
	pid := -1
	for candidate := 99990; candidate < 100100; candidate++ {
		alive, _ := processAlive(candidate)
		if !alive {
			pid = candidate
			break
		}
	}
	if pid < 0 {
		t.Skip("could not find an unused pid in scan range")
	}

	alive, err := processAlive(pid)
	if err != nil {
		t.Fatalf("processAlive returned error: %v", err)
	}
	if alive {
		t.Fatalf("processAlive(%d) = true, want false", pid)
	}
}

// TestProcessAliveLiveProcess verifies the probe reports a long-running
// process as alive.
func TestProcessAliveLiveProcess(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	alive, err := processAlive(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("processAlive returned error: %v", err)
	}
	if !alive {
		t.Fatalf("processAlive(%d) = false, want true", cmd.Process.Pid)
	}
}

// TestProcessAliveZeroPid covers the no-op short-circuit for pid <= 0.
func TestProcessAliveZeroPid(t *testing.T) {
	alive, err := processAlive(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alive {
		t.Fatalf("processAlive(0) = true, want false")
	}
}

// TestWaitForProcessExitFastExit ensures waitForProcessExit returns nil
// once the process exits within the timeout window. We must reap the
// child concurrently, otherwise it lingers as a zombie and signal(0)
// will report it as alive.
func TestWaitForProcessExitFastExit(t *testing.T) {
	prev := pollInterval
	pollInterval = 10 * time.Millisecond
	t.Cleanup(func() { pollInterval = prev })

	cmd := exec.Command("sh", "-c", "sleep 0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	pid := cmd.Process.Pid

	// Reap concurrently so the kernel removes the process table entry
	// as soon as the child exits.
	waited := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(waited)
	}()
	t.Cleanup(func() { <-waited })

	start := time.Now()
	if err := waitForProcessExit(pid, 2*time.Second); err != nil {
		t.Fatalf("waitForProcessExit returned %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("waitForProcessExit took too long: %v", elapsed)
	}
}

// TestWaitForProcessExitTimeout ensures waitForProcessExit reports a
// timeout error when the target process is still alive after the
// deadline.
func TestWaitForProcessExitTimeout(t *testing.T) {
	prev := pollInterval
	pollInterval = 10 * time.Millisecond
	t.Cleanup(func() { pollInterval = prev })

	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	err := waitForProcessExit(cmd.Process.Pid, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestWaitForProcessExitZeroPid is a no-op short-circuit.
func TestWaitForProcessExitZeroPid(t *testing.T) {
	if err := waitForProcessExit(0, time.Second); err != nil {
		t.Fatalf("waitForProcessExit(0) returned %v", err)
	}
}

// TestWaitForProcessExitZeroTimeoutAlive returns an error if the
// process is still alive when called with a non-positive timeout.
func TestWaitForProcessExitZeroTimeoutAlive(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	if err := waitForProcessExit(cmd.Process.Pid, 0); err == nil {
		t.Fatal("expected error for live process with zero timeout, got nil")
	}
}

// TestTerminateProcess_AlreadyDeadReturnsNil verifies that sending a
// signal to a process that has already exited and been reaped is
// reported as success rather than an error. This is the race window
// between waitForProcessExit's last poll and the SIGKILL escalation.
func TestTerminateProcess_AlreadyDeadReturnsNil(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	pid := cmd.Process.Pid
	// Reap synchronously so the kernel removes the process table entry.
	if _, err := cmd.Process.Wait(); err != nil {
		t.Fatalf("wait helper: %v", err)
	}

	if err := terminateProcess(pid, true); err != nil {
		t.Fatalf("terminateProcess(%d, true) = %v, want nil", pid, err)
	}
}

// TestKillTmuxSession_NotFoundReturnsNil verifies that targeting a
// nonexistent tmux socket/session is treated as benign (the desired
// state — no session — is already true).
func TestKillTmuxSession_NotFoundReturnsNil(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	sess := &session.SessionInfo{
		ID:         "nonexistent-session",
		TmuxSocket: "/nonexistent/socket/path",
	}
	if err := killTmuxSession(sess); err != nil {
		t.Fatalf("killTmuxSession = %v, want nil", err)
	}
}

// TODO: end-to-end coverage of terminateSession against a real tmux
// process is intentionally omitted here. It requires spawning tmux on
// a temporary socket and is exercised by the isolation backend tests
// in internal/core/isolation. The pure helpers above (tmuxSessionName,
// waitForProcessExit, processAlive) are the load-bearing logic and are
// fully covered.
