package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomPrefixConfig(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	// Setup custom config
	// On Darwin, os.UserConfigDir() usually returns ~/Library/Application Support
	// In our tests, we can override HOME or XDG_CONFIG_HOME
	xdgConfigHome := filepath.Join(home, ".config")
	os.MkdirAll(filepath.Join(xdgConfigHome, "aps"), 0755)
	configPath := filepath.Join(xdgConfigHome, "aps", "config.yaml")
	err := os.WriteFile(configPath, []byte("prefix: E2E"), 0644)
	require.NoError(t, err)

	// We need to pass XDG_CONFIG_HOME to the aps process
	// runAPS in helpers_test.go might not support custom env yet, let's check it.

	// Create profile
	_, _, err = runAPSWithEnv(t, home, map[string]string{"XDG_CONFIG_HOME": xdgConfigHome}, "profile", "create", "config-agent")
	require.NoError(t, err)

	// Run env
	stdout, _, err := runAPSWithEnv(t, home, map[string]string{"XDG_CONFIG_HOME": xdgConfigHome}, "run", "config-agent", "--", "env")
	require.NoError(t, err)

	// Verify custom prefix E2E_
	assert.Contains(t, stdout, "E2E_PROFILE_ID=config-agent")
}
