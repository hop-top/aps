package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionCommands(t *testing.T) {
	home := t.TempDir()
	env := append(os.Environ(), fmt.Sprintf("HOME=%s", home))

	// List empty
	cmd := exec.Command(apsBinary, "session", "list")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	// Output depends on formatting, typically headers
	if len(out) > 0 {
		assert.Contains(t, string(out), "No active sessions")
	}
}
