package isolation_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"oss-aps-cli/internal/core/isolation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestProfile(t *testing.T) string {
	tempDir := t.TempDir()
	profileID := "test-profile"

	agentsDir := filepath.Join(tempDir, ".agents")
	profileDir := filepath.Join(agentsDir, "profiles", profileID)
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profileYaml := `id: test-profile
display_name: Test Profile
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(profileYaml), 0644))

	secretsPath := filepath.Join(profileDir, "secrets.env")
	defaultSecrets := "TEST_SECRET=value"
	require.NoError(t, os.WriteFile(secretsPath, []byte(defaultSecrets), 0600))

	return tempDir
}

func TestProcessIsolation_PrepareContext(t *testing.T) {
	tempDir := setupTestProfile(t)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", originalHome)

	process := isolation.NewProcessIsolation()
	context, err := process.PrepareContext("test-profile")

	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "test-profile", context.ProfileID)
	assert.Contains(t, context.ProfileDir, "test-profile")
}

func TestProcessIsolation_SetupEnvironment(t *testing.T) {
	tempDir := setupTestProfile(t)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", originalHome)

	process := isolation.NewProcessIsolation()
	_, err := process.PrepareContext("test-profile")
	require.NoError(t, err)

	cmd := exec.Command("env")
	err = process.SetupEnvironment(cmd)
	assert.NoError(t, err)
	assert.NotNil(t, cmd.Env)
	assert.Contains(t, cmd.Env, "APS_PROFILE_ID=test-profile")
	assert.Contains(t, cmd.Env, "TEST_SECRET=value")
}

func TestProcessIsolation_Validate(t *testing.T) {
	tempDir := setupTestProfile(t)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", originalHome)

	process := isolation.NewProcessIsolation()

	_, err := process.PrepareContext("test-profile")
	require.NoError(t, err)

	err = process.Validate()
	assert.NoError(t, err)
}

func TestProcessIsolation_Cleanup(t *testing.T) {
	process := isolation.NewProcessIsolation()

	err := process.Cleanup()
	assert.NoError(t, err)

	context, err := process.PrepareContext("test-profile")
	assert.Error(t, err)
	assert.Nil(t, context)

	tempDir := setupTestProfile(t)
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", originalHome)

	_, err = process.PrepareContext("test-profile")
	assert.NoError(t, err)

	assert.NotNil(t, process)

	err = process.Cleanup()
	assert.NoError(t, err)
}
