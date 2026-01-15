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
func prepareAPS(t *testing.T, homeDir string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	
	cmd.Env = os.Environ()
	newEnv := []string{}
	newEnv = append(newEnv, fmt.Sprintf("HOME=%s", homeDir))
	newEnv = append(newEnv, fmt.Sprintf("USERPROFILE=%s", homeDir))
	
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "USERPROFILE=") {
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

	cmd := prepareAPS(t, homeDir, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
