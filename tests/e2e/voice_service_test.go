package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVoiceService_StatusStopped_FreshHome verifies `aps voice service
// status` prints "stopped" when no PID file exists. Exercises the
// kit/console/ps EntryFromPIDFile read path — when the file is absent,
// status falls through to "stopped".
func TestVoiceService_StatusStopped_FreshHome(t *testing.T) {
	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o700))

	stdout, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "status")
	require.NoError(t, err, "voice service status: stderr=%s", stderr)
	assert.Equal(t, "stopped\n", stdout)
}

// TestVoiceService_StatusRunning_FromPIDFile verifies `aps voice
// service status` reports "running" when a live PID file is present at
// the kit-mandated XDG runtime location ($XDG_RUNTIME_DIR/aps/voice.pid).
func TestVoiceService_StatusRunning_FromPIDFile(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}

	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	pidDir := filepath.Join(runtimeDir, "aps")
	require.NoError(t, os.MkdirAll(pidDir, 0o700))
	pidFile := filepath.Join(pidDir, "voice.pid")

	// Spawn a long-lived dummy process and pin its PID in the file.
	dummy := exec.Command(sleepBin, "30")
	require.NoError(t, dummy.Start())
	t.Cleanup(func() {
		_ = dummy.Process.Kill()
		_ = dummy.Wait()
	})
	require.NoError(t, os.WriteFile(pidFile,
		[]byte(strconv.Itoa(dummy.Process.Pid)+"\n"), 0o600))

	stdout, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "status")
	require.NoError(t, err, "voice service status: stderr=%s", stderr)
	assert.Equal(t, "running\n", stdout,
		"status should report running when pid file points to a live process")
}

// TestVoiceService_Stop_TerminatesAndCleansPIDFile verifies that
// `aps voice service stop` signals the live process recorded in the
// PID file and removes the file. This proves the migration: stop is
// driven entirely by the on-disk pid file (not in-memory exec.Cmd).
func TestVoiceService_Stop_TerminatesAndCleansPIDFile(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}

	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	pidDir := filepath.Join(runtimeDir, "aps")
	require.NoError(t, os.MkdirAll(pidDir, 0o700))
	pidFile := filepath.Join(pidDir, "voice.pid")

	dummy := exec.Command(sleepBin, "30")
	require.NoError(t, dummy.Start())
	pid := dummy.Process.Pid
	t.Cleanup(func() {
		// Best-effort: stop should already have killed it.
		_ = dummy.Process.Kill()
		_ = dummy.Wait()
	})
	require.NoError(t, os.WriteFile(pidFile,
		[]byte(strconv.Itoa(pid)+"\n"), 0o600))

	stdout, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "stop")
	require.NoError(t, err, "voice service stop: stderr=%s", stderr)
	assert.Contains(t, stdout, "Voice backend stopped.")

	// PID file removed.
	_, statErr := os.Stat(pidFile)
	assert.True(t, os.IsNotExist(statErr),
		"pid file should be removed after stop; got err=%v", statErr)

	// Status now reports stopped.
	stdout, stderr, err = runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "status")
	require.NoError(t, err, "voice service status: stderr=%s", stderr)
	assert.Equal(t, "stopped\n", stdout)

	// Process actually killed (give it a moment via Wait — which
	// returns once kill propagates).
	_ = dummy.Wait()
	require.False(t, processAlive(pid),
		"process %d should have been terminated by voice service stop", pid)
}

// TestVoiceService_Stop_NoOpWhenNotRunning matches story scenario 4:
// stop succeeds silently when no backend is running.
func TestVoiceService_Stop_NoOpWhenNotRunning(t *testing.T) {
	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o700))

	stdout, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "stop")
	require.NoError(t, err, "voice service stop: stderr=%s", stderr)
	assert.Contains(t, stdout, "Voice backend stopped.")
}

// TestVoiceService_Stop_ZombiePIDFile verifies that a stale pid file
// (pointing at a dead process) is silently swept by stop, with no
// error and no orphan file left behind.
func TestVoiceService_Stop_ZombiePIDFile(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not found, skipping")
	}

	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	pidDir := filepath.Join(runtimeDir, "aps")
	require.NoError(t, os.MkdirAll(pidDir, 0o700))
	pidFile := filepath.Join(pidDir, "voice.pid")

	// Spawn-and-reap to produce a guaranteed-dead PID.
	dead := exec.Command(sleepBin, "0")
	require.NoError(t, dead.Start())
	deadPID := dead.Process.Pid
	require.NoError(t, dead.Wait())
	require.NoError(t, os.WriteFile(pidFile,
		[]byte(strconv.Itoa(deadPID)+"\n"), 0o600))

	stdout, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "stop")
	require.NoError(t, err, "voice service stop: stderr=%s", stderr)
	assert.Contains(t, stdout, "Voice backend stopped.")

	_, statErr := os.Stat(pidFile)
	assert.True(t, os.IsNotExist(statErr), "stale pid file should be swept")
}

// TestVoiceService_Start_ErrorWhenNoBackendConfigured matches story
// scenario 2: start fails with a clear error when nothing is
// configured. CLI surface (exit code != 0, error to stderr) must match.
func TestVoiceService_Start_ErrorWhenNoBackendConfigured(t *testing.T) {
	home := t.TempDir()
	runtimeDir := filepath.Join(home, "run")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o700))

	_, stderr, err := runAPSWithEnv(t, home,
		map[string]string{"XDG_RUNTIME_DIR": runtimeDir},
		"voice", "service", "start")
	require.Error(t, err, "expected non-zero exit when no backend configured")
	assert.True(t,
		strings.Contains(stderr, "no voice backend binary configured") ||
			strings.Contains(stderr, "no binary configured"),
		"error message should explain missing backend; got: %s", stderr)
}

// processAlive is a syscall(0) liveness probe for the e2e harness.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(nil) == nil
}
