package bundle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuiltins_ReturnsAll(t *testing.T) {
	bundles, err := LoadBuiltins()
	require.NoError(t, err)
	assert.Len(t, bundles, 7)
}

func TestLoadBuiltins_DeveloperHasRequires(t *testing.T) {
	bundles, err := LoadBuiltins()
	require.NoError(t, err)

	var dev *Bundle
	for i := range bundles {
		if bundles[i].Name == "developer" {
			dev = &bundles[i]
			break
		}
	}
	require.NotNil(t, dev, "developer bundle not found in builtins")
	assert.GreaterOrEqual(t, len(dev.Requires), 3)
}

func TestLoadUserOverrides_EmptyDirReturnsEmpty(t *testing.T) {
	// Point the user config dir at a temp dir that has no bundles subdir.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// On macOS, os.UserConfigDir() uses $HOME/Library/Application Support when
	// XDG_CONFIG_HOME is unset. On Linux it uses $XDG_CONFIG_HOME. Set both to
	// be safe across platforms.
	t.Setenv("HOME", tmpDir)

	bundles, err := LoadUserOverrides()
	require.NoError(t, err)
	assert.Empty(t, bundles)
}

func TestLoadUserOverrides_LoadsYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Set HOME so os.UserConfigDir() returns a predictable path.
	t.Setenv("HOME", tmpDir)

	// Determine the actual config dir as the implementation will see it.
	configDir, err := os.UserConfigDir()
	require.NoError(t, err)

	// Create the expected bundles directory under that config dir.
	bundlesDir := configDir + "/aps/bundles"
	require.NoError(t, os.MkdirAll(bundlesDir, 0755))

	bundleYAML := `name: custom-test
description: A user-defined test bundle
version: "1.0"
`
	require.NoError(t, os.WriteFile(bundlesDir+"/custom-test.yaml", []byte(bundleYAML), 0644))

	bundles, err := LoadUserOverrides()
	require.NoError(t, err)
	require.Len(t, bundles, 1)
	assert.Equal(t, "custom-test", bundles[0].Name)
}
