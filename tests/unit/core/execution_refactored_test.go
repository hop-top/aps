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

// makeProfileDir creates a profile directory under the XDG data path and
// returns both the profile directory and a cleanup function that restores
// XDG_DATA_HOME and APS_DATA_PATH.
func makeProfileDir(t *testing.T, tempDir, profileID string) string {
	t.Helper()
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	orig := os.Getenv("XDG_DATA_HOME")
	origAPS := os.Getenv("APS_DATA_PATH")
	os.Setenv("XDG_DATA_HOME", filepath.Join(tempDir, ".local", "share"))
	os.Unsetenv("APS_DATA_PATH")
	t.Cleanup(func() {
		os.Setenv("XDG_DATA_HOME", orig)
		if origAPS != "" {
			os.Setenv("APS_DATA_PATH", origAPS)
		}
	})

	return profileDir
}

func TestRunCommandWithProcessIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "test-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: test-profile
display_name: Test Profile
isolation:
  level: process
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("TEST_VAR=test_value"), 0600)
	require.NoError(t, err)

	err = core.RunCommand(profileID, "echo", []string{"hello"})
	assert.NoError(t, err)
}

func TestRunCommandWithDefaultIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "default-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: default-profile
display_name: Default Profile
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("TEST_VAR=default"), 0600)
	require.NoError(t, err)

	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.NoError(t, err)
}

func TestRunCommandPlatformIsolationNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "platform-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: platform-profile
display_name: Platform Profile
isolation:
  level: platform
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestRunCommandContainerIsolationNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "container-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: container-profile
display_name: Container Profile
isolation:
  level: container
  container:
    image: ubuntu:22.04
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestRunCommandInvalidIsolationLevel(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "invalid-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: invalid-profile
display_name: Invalid Profile
isolation:
  level: invalid_level
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	err = core.RunCommand(profileID, "echo", []string{"test"})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid isolation level") ||
		strings.Contains(err.Error(), "failed to load profile"))
}

func TestRunActionWithProcessIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "action-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: action-profile
display_name: Action Profile
isolation:
  level: process
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("ACTION_VAR=action_value"), 0600)
	require.NoError(t, err)

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

	err = core.RunAction(profileID, "test", nil)
	assert.NoError(t, err)
}

func TestRunActionWithPayload(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "payload-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: payload-profile
display_name: Payload Profile
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

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

	payload := []byte("test payload data")
	err = core.RunAction(profileID, "read-stdin", payload)
	assert.NoError(t, err)
}

func TestRunActionInvalidProfile(t *testing.T) {
	err := core.RunAction("nonexistent-profile", "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load profile")
}

func TestRunActionInvalidAction(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "no-action-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: no-action-profile
display_name: No Action Profile
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	err = core.RunAction(profileID, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get action")
}

func TestBackwardCompatibility_InjectEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))

	profileID := "compat-profile"
	profileDir := makeProfileDir(t, tempDir, profileID)

	profileContent := `id: compat-profile
display_name: Compatibility Profile
`
	err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("COMPAT_VAR=compat_value"), 0600)
	require.NoError(t, err)

	profile, err := core.LoadProfile(profileID)
	require.NoError(t, err)

	cmd := exec.Command("test")
	err = core.InjectEnvironment(cmd, profile)
	assert.NoError(t, err)

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
