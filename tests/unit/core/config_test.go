package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Run("default configuration when file missing", func(t *testing.T) {
		// Ensure config file doesn't exist in a temp home
		tempHome := t.TempDir()
		os.Setenv("HOME", tempHome)
		os.Setenv("XDG_CONFIG_HOME", "")

		config, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, core.DefaultPrefix, config.Prefix)
	})

	t.Run("custom prefix from config file", func(t *testing.T) {
		tempHome := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempHome)

		configDir := filepath.Join(tempHome, "aps")
		err := os.MkdirAll(configDir, 0755)
		assert.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		err = os.WriteFile(configPath, []byte("prefix: CUSTOM"), 0644)
		assert.NoError(t, err)

		config, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "CUSTOM", config.Prefix)
	})
}

func TestGetConfigDir(t *testing.T) {
	t.Run("respects XDG_CONFIG_HOME", func(t *testing.T) {
		tempDir := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempDir)

		dir, err := core.GetConfigDir()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(tempDir, "aps"), dir)
	})

	t.Run("falls back to UserConfigDir", func(t *testing.T) {
		os.Setenv("XDG_CONFIG_HOME", "")
		// os.UserConfigDir() behavior depends on OS, but we can verify it's not empty
		dir, err := core.GetConfigDir()
		assert.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.True(t, filepath.IsAbs(dir))
	})
}
