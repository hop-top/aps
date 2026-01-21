package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"oss-aps-cli/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDefaultIsolation(t *testing.T) {
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("XDG_CONFIG_HOME", "")

	config, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, core.DefaultPrefix, config.Prefix)
	assert.Equal(t, core.IsolationProcess, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)
}

func TestConfigLoadWithIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: CUSTOM
isolation:
  default_level: container
  fallback_enabled: false
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "CUSTOM", config.Prefix)
	assert.Equal(t, core.IsolationContainer, config.Isolation.DefaultLevel)
	assert.False(t, config.Isolation.FallbackEnabled)
}

func TestConfigSaveWithIsolation(t *testing.T) {
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("XDG_CONFIG_HOME", "")

	config := &core.Config{
		Prefix: "MYAPP",
		Isolation: core.GlobalIsolationConfig{
			DefaultLevel:    core.IsolationPlatform,
			FallbackEnabled: false,
		},
	}

	err := core.SaveConfig(config)
	require.NoError(t, err)

	loadedConfig, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "MYAPP", loadedConfig.Prefix)
	assert.Equal(t, core.IsolationPlatform, loadedConfig.Isolation.DefaultLevel)
	assert.False(t, loadedConfig.Isolation.FallbackEnabled)
}

func TestConfigMigrate(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	oldConfig := `prefix: OLDAPP
`
	err = os.WriteFile(configPath, []byte(oldConfig), 0644)
	require.NoError(t, err)

	migrated, err := core.MigrateConfig()
	require.NoError(t, err)
	assert.True(t, migrated)

	config, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "OLDAPP", config.Prefix)
	assert.Equal(t, core.IsolationProcess, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "default_level")
	assert.Contains(t, string(content), "fallback_enabled")
}

func TestConfigMigrateNoExisting(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	migrated, err := core.MigrateConfig()
	require.NoError(t, err)
	assert.False(t, migrated)
}

func TestConfigInvalidIsolationLevel(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: TEST
isolation:
  default_level: invalid
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "TEST", config.Prefix)
	assert.Equal(t, core.IsolationProcess, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)
}

func TestConfigPartialIsolation(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	configDir := filepath.Join(tempDir, "aps")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `prefix: PARTIAL
isolation:
  default_level: platform
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := core.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "PARTIAL", config.Prefix)
	assert.Equal(t, core.IsolationPlatform, config.Isolation.DefaultLevel)
	assert.True(t, config.Isolation.FallbackEnabled)
}
