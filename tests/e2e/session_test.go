package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionCommands(t *testing.T) {
	home := t.TempDir()

	// Use runAPS which strips APS_DATA_PATH/XDG_DATA_HOME from the
	// inherited environment so the test does not see the developer's
	// real session registry.
	stdout, _, err := runAPS(t, home, "session", "list")
	require.NoError(t, err)
	if len(stdout) > 0 {
		assert.Contains(t, stdout, "No active sessions")
	}
}
