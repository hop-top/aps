package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalConfigIsolationDefaults(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: TEST
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "profile", "new", "default-iso-profile")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "default-iso-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: default-iso-profile")
}

func TestGlobalConfigIsolationCustom(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: MYAPP
isolation:
  default_level: container
  fallback_enabled: false
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "profile", "new", "custom-iso-profile")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "custom-iso-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: custom-iso-profile")
}

func TestGlobalConfigInvalidIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	configDir := filepath.Join(home, ".config", "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: INVALID
isolation:
  default_level: invalid_level
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, _, err = runAPS(t, home, "profile", "new", "invalid-iso-profile")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "invalid-iso-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: invalid-iso-profile")
}
