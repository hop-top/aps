package cli

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	apsjobs "hop.top/aps/internal/runtime/jobs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const execHelperEnv = "APS_TEST_EXEC_ARGS"

// TestExecuteHelperProcess is not a test: it is the child entry point
// for runAPS. It runs the real Execute() in a fresh process, exactly as
// the binary does, so the tree mutations Execute performs (group
// assignment, error classification) cannot leak into other tests.
func TestExecuteHelperProcess(t *testing.T) {
	raw := os.Getenv(execHelperEnv)
	if raw == "" {
		t.Skip("helper process only")
	}
	os.Args = append([]string{"aps"}, strings.Fields(raw)...)
	if err := Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// runAPS runs `aps <args>` in a child process with an isolated data
// dir and returns its stderr.
func runAPS(t *testing.T, dataHome string, args ...string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecuteHelperProcess$")
	cmd.Env = append(os.Environ(),
		execHelperEnv+"="+strings.Join(args, " "),
		"XDG_DATA_HOME="+dataHome,
		"XDG_STATE_HOME="+filepath.Join(dataHome, "state"),
		"APS_DATA_PATH="+filepath.Join(dataHome, "aps"),
		"APS_NO_BUS_WARN=1",
		"APS_JOBS_DISABLE=",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), stderr.String())
	return stderr.String()
}

// TestNonServingCommandStartsNoJobPoller pins that commands which do
// not serve jobs never open the job queue or start pollers, and
// therefore exit with a clean stderr. Starting pollers in every
// process made the shutdown cancel race a poll in flight.
func TestNonServingCommandStartsNoJobPoller(t *testing.T) {
	for _, args := range [][]string{
		{"version"},
		{"profile", "list"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			dataHome := t.TempDir()

			stderr := runAPS(t, dataHome, args...)

			require.NoFileExists(t, filepath.Join(dataHome, "aps", "jobs.db"),
				"non-serving command must not open the job queue")
			require.NotContains(t, stderr, "job poller")
			require.NotContains(t, stderr, "ERRO")
		})
	}
}

// TestStartJobs_OpensQueueForServingProcess is the counterpart: a
// serving process still gets the queue and its pollers via startJobs.
func TestStartJobs_OpensQueueForServingProcess(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("APS_JOBS_DISABLE", "")

	startJobs()
	t.Cleanup(drainJobs)

	require.NotNil(t, jobEngine)
	require.NotNil(t, apsjobs.Get())
	require.FileExists(t, filepath.Join(dataHome, "aps", "jobs.db"))
}

// freeAddr returns a loopback address with a free port.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// serveAndStop runs `aps serve` in a child process, waits for /health,
// lets the 500ms pollers run for settle, sends SIGTERM and returns the
// child's stderr after it exits.
func serveAndStop(t *testing.T, settle time.Duration) string {
	t.Helper()
	dataHome := t.TempDir()
	addr := freeAddr(t)

	cmd := exec.Command(os.Args[0], "-test.run=^TestExecuteHelperProcess$")
	cmd.Env = append(os.Environ(),
		execHelperEnv+"=serve --addr "+addr,
		"XDG_DATA_HOME="+dataHome,
		"XDG_STATE_HOME="+filepath.Join(dataHome, "state"),
		"APS_DATA_PATH="+filepath.Join(dataHome, "aps"),
		"APS_NO_BUS_WARN=1",
		"APS_JOBS_DISABLE=",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Start())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// The child's copy goroutine writes stderr until Wait returns, so the
	// buffer is only read after the child has exited.
	healthy := assert.Eventually(t, func() bool {
		resp, err := http.Get("http://" + addr + "/health")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 15*time.Second, 50*time.Millisecond)
	if !healthy {
		_ = cmd.Process.Kill()
		werr := <-done
		t.Fatalf("serve never became healthy (exit: %v): %s", werr, stderr.String())
	}

	// settle positions the signal inside a poll tick so a cancelled
	// in-flight query is exercised.
	time.Sleep(settle)
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))

	select {
	case err := <-done:
		require.NoError(t, err, stderr.String())
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		werr := <-done
		t.Fatalf("serve did not exit after SIGTERM (exit after kill: %v): %s", werr, stderr.String())
	}
	return stderr.String()
}

// TestServeShutdownLogsNoPollerError pins that stopping `aps serve`
// while the job pollers are polling does not log the cancelled poll
// as an error: shutdown cancellation is normal.
func TestServeShutdownLogsNoPollerError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM not supported")
	}
	// Pollers tick every 500ms from startJobs. Sweep the signal across
	// the second tick so some runs land mid-query.
	for i := 0; i < 17; i++ {
		settle := time.Duration(400+10*i) * time.Millisecond
		t.Run(fmt.Sprintf("settle_%dms", settle.Milliseconds()), func(t *testing.T) {
			stderr := serveAndStop(t, settle)
			require.NotContains(t, stderr, "ERRO job poller")
			require.NotContains(t, stderr, "context canceled")
		})
	}
}
