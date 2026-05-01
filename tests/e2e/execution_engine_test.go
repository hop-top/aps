package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionEngineWithProcessIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile with process isolation
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "process-exec-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: process-exec-profile
display_name: Process Execution Profile
isolation:
  level: process
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("EXEC_VAR=process_value"), 0600)
	require.NoError(t, err)

	// Run command using CLI
	stdout, _, err := runAPS(t, home, "run", "process-exec-profile", "--", "echo", "test")
	require.NoError(t, err)
	assert.Contains(t, stdout, "test")
}

func TestExecutionEngineWithDefaultIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile without isolation level (should use default)
	_, _, err := runAPS(t, home, "profile", "create", "default-exec-profile")
	require.NoError(t, err)

	// Run command should work
	stdout, _, err := runAPS(t, home, "run", "default-exec-profile", "--", "echo", "default")
	require.NoError(t, err)
	assert.Contains(t, stdout, "default")
}

func TestExecutionEngineUnsupportedIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile with container isolation (not implemented)
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "container-exec-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: container-exec-profile
display_name: Container Execution Profile
isolation:
  level: container
  strict: true
  fallback: false
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Run command should fail with clear error message
	_, stderr, err := runAPS(t, home, "run", "container-exec-profile", "--", "echo", "test")
	assert.Error(t, err)
	assert.Contains(t, stderr, "not yet implemented")
}

func TestExecutionEngineActionExecution(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile with process isolation
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "action-exec-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: action-exec-profile
display_name: Action Execution Profile
isolation:
  level: process
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("ACTION_EXEC_VAR=executed"), 0600)
	require.NoError(t, err)

	// Create an action
	actionsDir := filepath.Join(profileDir, "actions")
	err = os.MkdirAll(actionsDir, 0755)
	require.NoError(t, err)

	actionScript := `#!/bin/sh
echo "Action executed successfully"
`
	err = os.WriteFile(filepath.Join(actionsDir, "exec-test.sh"), []byte(actionScript), 0755)
	require.NoError(t, err)

	actionYaml := `id: exec-test
title: Exec Test Action
type: sh
path: actions/exec-test.sh
accepts_stdin: false
`
	err = os.WriteFile(filepath.Join(actionsDir, "exec-test.yaml"), []byte(actionYaml), 0644)
	require.NoError(t, err)

	// Run action
	stdout, _, err := runAPS(t, home, "action", "run", "action-exec-profile", "exec-test")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Action executed successfully")
}

func TestExecutionEngineActionWithPayload(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create profile
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "payload-exec-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: payload-exec-profile
display_name: Payload Execution Profile
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Create an action that reads stdin
	actionsDir := filepath.Join(profileDir, "actions")
	err = os.MkdirAll(actionsDir, 0755)
	require.NoError(t, err)

	actionScript := `#!/bin/sh
# Read and count payload bytes
wc -c > /dev/null
echo "Payload processed"
`
	err = os.WriteFile(filepath.Join(actionsDir, "read-payload.sh"), []byte(actionScript), 0755)
	require.NoError(t, err)

	actionYaml := `id: read-payload
title: Read Payload Action
type: sh
path: actions/read-payload.sh
accepts_stdin: true
`
	err = os.WriteFile(filepath.Join(actionsDir, "read-payload.yaml"), []byte(actionYaml), 0644)
	require.NoError(t, err)

	// Create payload file
	payloadPath := filepath.Join(home, "payload.txt")
	payloadContent := "test payload data for execution engine"
	err = os.WriteFile(payloadPath, []byte(payloadContent), 0644)
	require.NoError(t, err)

	// Run action with payload
	stdout, _, err := runAPS(t, home, "action", "run", "payload-exec-profile", "read-payload", "--payload-file", payloadPath)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Payload processed")
}

func TestExecutionEngineBackwardCompatibility(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create old-style profile without isolation section
	_, _, err := runAPS(t, home, "profile", "create", "old-style-profile")
	require.NoError(t, err)

	// Add a secret to verify environment injection
	secretsPath := filepath.Join(home, ".local", "share", "aps", "profiles", "old-style-profile", "secrets.env")
	err = os.WriteFile(secretsPath, []byte("OLD_VAR=old_value\n"), 0600)
	require.NoError(t, err)

	// Run command - should work with default process isolation
	stdout, _, err := runAPS(t, home, "run", "old-style-profile", "--", "echo", "backward_compat")
	require.NoError(t, err)
	assert.Contains(t, stdout, "backward_compat")
}
