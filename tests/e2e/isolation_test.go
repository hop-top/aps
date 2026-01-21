package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolationManager_ProcessLevel(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "new", "iso-agent")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "run", "iso-agent", "--", "env")
	require.NoError(t, err)

	assert.Contains(t, stdout, "APS_PROFILE_ID=iso-agent")
	assert.Contains(t, stdout, filepath.Join(home, ".agents", "profiles", "iso-agent"))
}

func TestIsolationManager_InvalidProfile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, stderr, err := runAPS(t, home, "run", "nonexistent-agent", "--", "env")
	assert.Error(t, err)
	assert.Contains(t, stderr, "failed")
}

func TestIsolationManager_Sequence(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	profileIDs := []string{"agent-1", "agent-2", "agent-3"}

	for _, id := range profileIDs {
		_, _, err := runAPS(t, home, "profile", "new", id)
		require.NoError(t, err)

		stdout, _, err := runAPS(t, home, "run", id, "--", "echo", id)
		require.NoError(t, err)

		assert.Contains(t, stdout, id)
	}
}

func TestIsolationManager_ActionExecution(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "new", "action-agent")
	require.NoError(t, err)

	actionScript := `#!/bin/sh
echo "TEST_ACTION_OUTPUT=hello"
env | grep APS_PROFILE_ID
`
	actionPath := filepath.Join(home, ".agents", "profiles", "action-agent", "actions", "test.sh")
	err = os.WriteFile(actionPath, []byte(actionScript), 0755)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "action", "run", "action-agent", "test")
	require.NoError(t, err)

	assert.Contains(t, stdout, "TEST_ACTION_OUTPUT=hello")
	assert.Contains(t, stdout, "APS_PROFILE_ID=action-agent")
}

func TestIsolationManager_WithSecrets(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "new", "secret-agent")
	require.NoError(t, err)

	secretsPath := filepath.Join(home, ".agents", "profiles", "secret-agent", "secrets.env")
	err = os.WriteFile(secretsPath, []byte("CUSTOM_VAR=custom_value\n"), 0600)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "run", "secret-agent", "--", "env")
	require.NoError(t, err)

	assert.Contains(t, stdout, "CUSTOM_VAR=custom_value")
}
