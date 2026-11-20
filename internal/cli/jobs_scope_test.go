package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	apsjobs "hop.top/aps/internal/runtime/jobs"

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
