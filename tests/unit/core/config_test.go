package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"
	kitconfig "hop.top/kit/go/core/config"

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

func TestLoadConfigLayered(t *testing.T) {
	t.Run("project layer overrides user layer", func(t *testing.T) {
		tempHome := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempHome)
		t.Cleanup(func() { os.Unsetenv("XDG_CONFIG_HOME") })

		userDir := filepath.Join(tempHome, "aps")
		assert.NoError(t, os.MkdirAll(userDir, 0o755))
		userYAML := []byte("prefix: USER\nisolation:\n  default_level: process\n")
		assert.NoError(t, os.WriteFile(filepath.Join(userDir, "config.yaml"), userYAML, 0o644))

		// Project-level layer (PWD/.aps.yaml).
		cwd, err := os.Getwd()
		assert.NoError(t, err)
		projectDir := t.TempDir()
		assert.NoError(t, os.Chdir(projectDir))
		t.Cleanup(func() { os.Chdir(cwd) })
		projectYAML := []byte("prefix: PROJECT\n")
		assert.NoError(t, os.WriteFile(filepath.Join(projectDir, ".aps.yaml"), projectYAML, 0o644))

		cfg, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "PROJECT", cfg.Prefix)
		assert.Equal(t, core.IsolationProcess, cfg.Isolation.DefaultLevel)
	})

	t.Run("malformed YAML returns defaults", func(t *testing.T) {
		tempHome := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempHome)
		t.Cleanup(func() { os.Unsetenv("XDG_CONFIG_HOME") })

		userDir := filepath.Join(tempHome, "aps")
		assert.NoError(t, os.MkdirAll(userDir, 0o755))
		assert.NoError(t, os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte("not: valid: yaml: ::"), 0o644))

		cfg, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, core.DefaultPrefix, cfg.Prefix)
	})
}

// TestLoadConfig_CLIOverrides verifies that core.SetConfigArgs threads
// kit cli's parsed -c/--config tokens into kitconfig.Load (T-0583).
func TestLoadConfig_CLIOverrides(t *testing.T) {
	t.Run("override wins over file", func(t *testing.T) {
		tempHome := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempHome)
		t.Cleanup(func() { os.Unsetenv("XDG_CONFIG_HOME") })

		userDir := filepath.Join(tempHome, "aps")
		assert.NoError(t, os.MkdirAll(userDir, 0o755))
		userYAML := []byte("prefix: USER\nisolation:\n  default_level: process\n")
		assert.NoError(t, os.WriteFile(filepath.Join(userDir, "config.yaml"), userYAML, 0o644))

		// Round-trip the raw token through ParseConfigArgs so the test
		// matches the wiring in internal/cli/root.go (which never hands
		// LoadConfig a flat dotted-key map).
		_, overrides, perr := kitconfig.ParseConfigArgs(
			[]string{"isolation.default_level=container"})
		assert.NoError(t, perr)
		core.SetConfigArgs(nil, overrides)
		t.Cleanup(func() { core.SetConfigArgs(nil, nil) })

		cfg, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "USER", cfg.Prefix)
		assert.Equal(t, core.IsolationContainer, cfg.Isolation.DefaultLevel)
	})

	t.Run("extra config path layers in", func(t *testing.T) {
		tempHome := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", tempHome)
		t.Cleanup(func() { os.Unsetenv("XDG_CONFIG_HOME") })

		extra := filepath.Join(tempHome, "extra.yaml")
		assert.NoError(t, os.WriteFile(extra, []byte("prefix: FROM_EXTRA\n"), 0o644))

		core.SetConfigArgs([]string{extra}, nil)
		t.Cleanup(func() { core.SetConfigArgs(nil, nil) })

		cfg, err := core.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "FROM_EXTRA", cfg.Prefix)
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
