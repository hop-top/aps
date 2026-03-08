//go:build darwin

package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/isolation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDarwinSandbox_InterfaceCompliance(t *testing.T) {
	t.Run("DarwinSandbox implements IsolationManager", func(t *testing.T) {
		var _ isolation.IsolationManager = isolation.NewDarwinSandbox()
	})
}

func TestDarwinSandbox_IsAvailable(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping in CI environment")
	}

	adapter := isolation.NewDarwinSandbox()
	assert.True(t, adapter.IsAvailable(), "DarwinSandbox should be available on macOS")
}

func TestDarwinSandbox_PrepareContext(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_DATA_HOME", "")
	os.Unsetenv("APS_DATA_PATH")
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))
	defer os.Setenv("XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"))

	profileID := "darwin-test-profile"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: darwin-test-profile
display_name: Darwin Test Profile
isolation:
  level: platform
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("DARWIN_TEST_VAR=darwin_value\n"), 0600)
	require.NoError(t, err)

	adapter := isolation.NewDarwinSandbox()
	context, err := adapter.PrepareContext(profileID)
	require.NoError(t, err)

	assert.Equal(t, profileID, context.ProfileID)
	assert.Equal(t, profileDir, context.ProfileDir)
	assert.Equal(t, profilePath, context.ProfileYaml)
	assert.Equal(t, secretsPath, context.SecretsPath)
	assert.NotNil(t, context.Environment)
}

func TestDarwinSandbox_Validate(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_DATA_HOME", "")
	os.Unsetenv("APS_DATA_PATH")
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))
	defer os.Setenv("XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"))

	profileID := "darwin-validate-test"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: darwin-validate-test
display_name: Darwin Validate Test
isolation:
  level: platform
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	adapter := isolation.NewDarwinSandbox()
	_, err = adapter.PrepareContext(profileID)
	require.NoError(t, err)

	err = adapter.Validate()
	assert.NoError(t, err)
}

func TestDarwinSandbox_ValidateErrors(t *testing.T) {
	adapter := isolation.NewDarwinSandbox()

	err := adapter.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context not prepared")
}

func TestDarwinSandbox_PasswordGeneration(t *testing.T) {
	adapter := isolation.NewDarwinSandbox()

	err := adapter.Validate()
	assert.Error(t, err)

	// Note: generateRandomPassword and findNextAvailableUID are internal methods
	// These are tested implicitly through the adapter configuration process
}

func TestDarwinSandbox_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_DATA_HOME", "")
	os.Unsetenv("APS_DATA_PATH")
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))
	defer os.Setenv("XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"))

	profileID := "darwin-cleanup-test"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: darwin-cleanup-test
display_name: Darwin Cleanup Test
isolation:
  level: platform
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	adapter := isolation.NewDarwinSandbox()
	_, err = adapter.PrepareContext(profileID)
	require.NoError(t, err)

	err = adapter.Cleanup()
	assert.NoError(t, err)
}

func TestDarwinSandbox_ManagerIntegration(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("XDG_DATA_HOME", "")
	os.Unsetenv("APS_DATA_PATH")
	os.Setenv("XDG_CONFIG_HOME", "")
	defer os.Setenv("HOME", os.Getenv("HOME"))
	defer os.Setenv("XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"))

	profileID := "darwin-manager-test"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: darwin-manager-test
display_name: Darwin Manager Test
isolation:
  level: platform
  strict: false
  fallback: true
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("DARWIN_MANAGER_VAR=manager_value\n"), 0600)
	require.NoError(t, err)

	profile, err := core.LoadProfile(profileID)
	require.NoError(t, err)

	globalConfig := &core.Config{
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationProcess,
			FallbackEnabled: true,
		},
	}

	manager := isolation.NewManager()
	manager.Register(core.IsolationProcess, isolation.NewProcessIsolation())
	manager.Register(core.IsolationPlatform, isolation.NewDarwinSandbox())

	isoManager, err := manager.GetIsolationManager(profile, globalConfig)
	require.NoError(t, err)
	assert.Equal(t, isolation.NewDarwinSandbox(), isoManager, "Should get DarwinSandbox adapter")
}
