package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"testing"
)

// destructiveToken mirrors kit's destructiveTokenSha (kit
// go/console/cli/policy_runE.go): the first 12 hex chars of
// sha256(cmd.CommandPath()). Tests pass the result via
// --confirm-token=<sha> to satisfy the kit/destructive-token gate
// (T-0654) on non-TTY exec. Keep this in sync with kit if the hash
// shape ever changes.
func destructiveToken(commandPath string) string {
	h := sha256.Sum256([]byte(commandPath))
	return hex.EncodeToString(h[:6])
}

// prepareAPS builds a sandboxed aps invocation: HOME and all XDG_*
// directories point at homeDir, APS_DATA_PATH is stripped from the
// parent, and APS_NO_BUS_WARN=1 keeps stderr clean. Extra env
// overrides take precedence (and may un-set APS_NO_BUS_WARN by
// passing it as ""). Tests that exercise the bus-token warning
// explicitly (webhook_gap) set APS_NO_BUS_WARN="" via extraEnv.
func prepareAPS(t *testing.T, homeDir string, extraEnv map[string]string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	if extraEnv == nil {
		extraEnv = map[string]string{}
	}
	if _, set := extraEnv["APS_NO_BUS_WARN"]; !set {
		extraEnv["APS_NO_BUS_WARN"] = "1"
	}
	cmd.Env = sandboxEnvWith(homeDir, extraEnv)
	return cmd
}

// runAPS executes the compiled binary with the given arguments and home directory
func runAPS(t *testing.T, homeDir string, args ...string) (string, string, error) {
	t.Helper()

	cmd := prepareAPS(t, homeDir, nil, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runAPSWithEnv executes the compiled binary with extra environment variables
func runAPSWithEnv(t *testing.T, homeDir string, env map[string]string, args ...string) (string, string, error) {
	t.Helper()

	cmd := prepareAPS(t, homeDir, env, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
