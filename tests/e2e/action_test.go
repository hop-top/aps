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

// TestActionList_TypeFilter exercises --type sh|py|js (T-0439). Seeds
// three actions of mixed runtimes and asserts only sh entries render
// when --type sh is passed.
func TestActionList_TypeFilter(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "type-agent"

	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0o755))
	for _, f := range []string{"build.sh", "report.py", "lint.js"} {
		require.NoError(t, os.WriteFile(filepath.Join(actionsDir, f), []byte("#!/bin/sh\n"), 0o755))
	}

	// Without filter: all three present.
	stdout, _, err := runAPS(t, home, "action", "list", profileID)
	require.NoError(t, err)
	for _, want := range []string{"build", "report", "lint"} {
		assert.Contains(t, stdout, want)
	}

	// --type sh: only build.
	stdout, _, err = runAPS(t, home, "action", "list", profileID, "--type", "sh")
	require.NoError(t, err)
	assert.Contains(t, stdout, "build")
	assert.NotContains(t, stdout, "report")
	assert.NotContains(t, stdout, "lint")
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
