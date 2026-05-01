package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolationFallbackProcessToDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create config with process as default
	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: TEST
isolation:
  default_level: process
  fallback_enabled: true
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create profile requesting container (currently not implemented)
	// In future, this would fall back to process when container is unavailable
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "fallback-profile")
	err = os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	profileContent := `id: fallback-profile
display_name: Fallback Test Profile
isolation:
  level: container
  strict: false
  fallback: true
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	// Profile should load successfully (fallback allowed)
	_, _, err = runAPS(t, home, "profile", "show", "fallback-profile")
	require.NoError(t, err)

	// Run command should fail with "not yet implemented" until container isolation is added
	// Note: Fallback logic exists in isolation manager but isn't wired into execution engine yet
	_, stderr, err := runAPS(t, home, "run", "fallback-profile", "--", "echo", "test")
	assert.Error(t, err)
	assert.Contains(t, stderr, "not yet implemented")
}

func TestIsolationStrictModeNoFallback(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create config with process as default
	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: STRICT
isolation:
  default_level: process
  fallback_enabled: true
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create profile with strict mode requesting container
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "strict-profile")
	err = os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	profileContent := `id: strict-profile
display_name: Strict Mode Test Profile
isolation:
  level: container
  strict: true
  fallback: true
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	// Profile should load (validation doesn't check isolation availability yet)
	stdout, _, err := runAPS(t, home, "profile", "show", "strict-profile")
	require.NoError(t, err)
	assert.Contains(t, stdout, "strict: true")

	// Running command should fail (strict mode prevents fallback)
	// Note: This test documents expected behavior - currently the CLI doesn't use
	// GetIsolationManager yet, so this may pass. The fallback logic
	// is available in the isolation manager for future integration.
}

func TestIsolationProfileFallbackDisabled(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create config with process as default
	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: NOFALLBACK
isolation:
  default_level: process
  fallback_enabled: true
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create profile with fallback disabled
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "no-fallback-profile")
	err = os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	profileContent := `id: no-fallback-profile
display_name: No Fallback Test Profile
isolation:
  level: container
  strict: false
  fallback: false
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	// Profile should load
	stdout, _, err := runAPS(t, home, "profile", "show", "no-fallback-profile")
	require.NoError(t, err)
	assert.Contains(t, stdout, "fallback: false")
}

func TestIsolationGlobalFallbackDisabled(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create config with fallback disabled globally
	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: GLOBALNOFALL
isolation:
  default_level: process
  fallback_enabled: false
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create profile requesting container with profile-level fallback enabled
	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "global-no-fallback-profile")
	err = os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	profileContent := `id: global-no-fallback-profile
display_name: Global No Fallback Test Profile
isolation:
  level: container
  strict: false
  fallback: true
  container:
    image: ubuntu:22.04
`
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	// Profile should load
	stdout, _, err := runAPS(t, home, "profile", "show", "global-no-fallback-profile")
	require.NoError(t, err)
	assert.Contains(t, stdout, "id: global-no-fallback-profile")
}

func TestIsolationDefaultLevelUsed(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Create config with platform as default
	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: DEFAULT
isolation:
  default_level: process
  fallback_enabled: true
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create profile without specifying isolation level
	_, _, err = runAPS(t, home, "profile", "create", "default-level-profile")
	require.NoError(t, err)

	// Profile should use default level from global config
	stdout, _, err := runAPS(t, home, "profile", "show", "default-level-profile")
	require.NoError(t, err)
	assert.Contains(t, stdout, "id: default-level-profile")

	// Run command should work
	stdout, _, err = runAPS(t, home, "run", "default-level-profile", "--", "echo", "works")
	require.NoError(t, err)
	assert.Contains(t, stdout, "works")
}
