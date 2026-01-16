package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// prepareAPS creates the command with environment set up
func prepareAPS(t *testing.T, homeDir string, extraEnv map[string]string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)

	newEnv := []string{}
	newEnv = append(newEnv, fmt.Sprintf("HOME=%s", homeDir))
	newEnv = append(newEnv, fmt.Sprintf("USERPROFILE=%s", homeDir))

	// Add extra environment variables
	for k, v := range extraEnv {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, v))
	}

	for _, e := range os.Environ() {
		key := strings.Split(e, "=")[0]
		if key == "HOME" || key == "USERPROFILE" {
			continue
		}
		if _, ok := extraEnv[key]; ok {
			continue
		}
		newEnv = append(newEnv, e)
	}
	cmd.Env = newEnv
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
