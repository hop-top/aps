package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandWithProcessIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile
	profileID := "test-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: test-profile
display_name: Test Profile
isolation:
  level: process
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("TEST_VAR=test_value"), 0600)
	require.NoError(t, err)

	// Run a simple command
	err = core.RunCommand(profileID, "echo", []string{"hello"})
	assert.NoError(t, err)
}

func TestRunCommandWithDefaultIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile without isolation level specified
	profileID := "default-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: default-profile
display_name: Default Profile
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("TEST_VAR=default"), 0600)
	require.NoError(t, err)

	// Run command should use default (process) isolation
	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.NoError(t, err)
}

func TestRunCommandPlatformIsolationNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile with platform isolation
	profileID := "platform-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: platform-profile
display_name: Platform Profile
isolation:
  level: platform
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Should fail with "not yet implemented"
	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestRunCommandContainerIsolationNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile with container isolation
	profileID := "container-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: container-profile
display_name: Container Profile
isolation:
  level: container
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Should fail with "not yet implemented"
	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestRunCommandInvalidIsolationLevel(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile with invalid isolation level
	profileID := "invalid-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: invalid-profile
display_name: Invalid Profile
isolation:
  level: invalid_level
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Should fail with error about invalid isolation level
	// Profile validation catches this before execution
	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	// Error could be about invalid isolation level or loading profile
	assert.True(t, strings.Contains(err.Error(), "invalid isolation level") ||
		strings.Contains(err.Error(), "failed to load profile"))
}

func TestRunActionWithProcessIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile with an action
	profileID := "action-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: action-profile
display_name: Action Profile
isolation:
  level: process
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("ACTION_VAR=action_value"), 0600)
	require.NoError(t, err)

	// Create an action
	actionsDir := filepath.Join(profileDir, "actions")
	err = os.MkdirAll(actionsDir, 0755)
	require.NoError(t, err)

	actionScript := `#!/bin/sh
echo "Action executed successfully"
`
	err = os.WriteFile(filepath.Join(actionsDir, "test.sh"), []byte(actionScript), 0755)
	require.NoError(t, err)

	actionYaml := `id: test
title: Test Action
type: sh
path: actions/test.sh
accepts_stdin: false
`
	err = os.WriteFile(filepath.Join(actionsDir, "test.yaml"), []byte(actionYaml), 0644)
	require.NoError(t, err)

	// Run the action
	err = core.RunAction(profileID, "test", nil)
	assert.NoError(t, err)
}

func TestRunActionWithPayload(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile with an action
	profileID := "payload-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: payload-profile
display_name: Payload Profile
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
cat > /dev/null
echo "Payload processed"
`
	err = os.WriteFile(filepath.Join(actionsDir, "read-stdin.sh"), []byte(actionScript), 0755)
	require.NoError(t, err)

	actionYaml := `id: read-stdin
title: Read Stdin Action
type: sh
path: actions/read-stdin.sh
accepts_stdin: true
`
	err = os.WriteFile(filepath.Join(actionsDir, "read-stdin.yaml"), []byte(actionYaml), 0644)
	require.NoError(t, err)

	// Run the action with payload
	payload := []byte("test payload data")
	err = core.RunAction(profileID, "read-stdin", payload)
	assert.NoError(t, err)
}

func TestRunActionInvalidProfile(t *testing.T) {
	// Try to run action on non-existent profile
	err := core.RunAction("nonexistent-profile", "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load profile")
}

func TestRunActionInvalidAction(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create a profile
	profileID := "no-action-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: no-action-profile
display_name: No Action Profile
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	// Try to run non-existent action
	err = core.RunAction(profileID, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get action")
}

func TestBackwardCompatibility_InjectEnvironment(t *testing.T) {
	// Ensure InjectEnvironment still works for backward compatibility
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "compat-profile"
	profileDir := filepath.Join(tempDir, ".agents", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: compat-profile
display_name: Compatibility Profile
`
	err = os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("COMPAT_VAR=compat_value"), 0600)
	require.NoError(t, err)

	profile, err := core.LoadProfile(profileID)
	require.NoError(t, err)

	// Test InjectEnvironment directly
	cmd := exec.Command("test")
	err = core.InjectEnvironment(cmd, profile)
	assert.NoError(t, err)

	// Verify environment was set
	foundID := false
	foundCompat := false
	for _, env := range cmd.Env {
		if env == "APS_PROFILE_ID=compat-profile" {
			foundID = true
		}
		if env == "COMPAT_VAR=compat_value" {
			foundCompat = true
		}
	}
	assert.True(t, foundID, "APS_PROFILE_ID not found in environment")
	assert.True(t, foundCompat, "COMPAT_VAR not found in environment")
}
