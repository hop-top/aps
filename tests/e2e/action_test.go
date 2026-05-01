package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionDiscovery(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "action-agent"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Create a script
	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	scriptPath := filepath.Join(actionsDir, "hello.sh")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho Hello from Action"), 0755)
	require.NoError(t, err)

	// List actions
	stdout, _, err := runAPS(t, home, "action", "list", profileID)
	require.NoError(t, err)
	assert.Contains(t, stdout, "hello")
}

func TestActionRun(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "run-agent"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Create script
	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	scriptPath := filepath.Join(actionsDir, "greet.sh")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho Greetings $APS_PROFILE_ID"), 0755)
	require.NoError(t, err)

	// Run action
	stdout, _, err := runAPS(t, home, "action", "run", profileID, "greet")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Greetings run-agent")
}

func TestActionPayload(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "payload-agent"

	// Create profile
	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Create script that echoes stdin
	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	scriptPath := filepath.Join(actionsDir, "echo.sh")
	// cat command reads from stdin
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\ncat"), 0755)
	require.NoError(t, err)

	// Create payload file
	payloadPath := filepath.Join(home, "payload.json")
	err = os.WriteFile(payloadPath, []byte(`{"msg":"hello"}`), 0644)
	require.NoError(t, err)

	// Run with payload file
	stdout, _, err := runAPS(t, home, "action", "run", profileID, "echo", "--payload-file", payloadPath)
	require.NoError(t, err)
	assert.Contains(t, stdout, `{"msg":"hello"}`)
}
