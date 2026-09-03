package bundle

import (
	"os"
	"path/filepath"
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

	// os.UserConfigDir() reads $XDG_CONFIG_HOME on Linux, $HOME on macOS,
	// and %AppData% on Windows — HOME/XDG_CONFIG_HOME are not consulted at
	// all there (see os.UserConfigDir godoc). Without AppData set, this
	// falls through to the real, shared %AppData%\aps\bundles on the
	// runner and can see state left behind by other tests/steps in the
	// same job. Set all three so isolation actually holds everywhere.
	t.Setenv("HOME", tmpDir)
	t.Setenv("AppData", tmpDir)

	bundles, err := LoadUserOverrides()
	require.NoError(t, err)
	assert.Empty(t, bundles)
}

func TestLoadUserOverrides_LoadsYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Set HOME (macOS/Linux) and AppData (Windows) so os.UserConfigDir()
	// returns a predictable, isolated path on every platform.
	t.Setenv("HOME", tmpDir)
	t.Setenv("AppData", tmpDir)

	// Determine the actual config dir as the implementation will see it.
	configDir, err := os.UserConfigDir()
	require.NoError(t, err)

	// Create the expected bundles directory under that config dir.
	bundlesDir := filepath.Join(configDir, "aps", "bundles")
	require.NoError(t, os.MkdirAll(bundlesDir, 0o755))

	bundleYAML := `name: custom-test
description: A user-defined test bundle
version: "1.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(bundlesDir, "custom-test.yaml"), []byte(bundleYAML), 0o644))

	bundles, err := LoadUserOverrides()
	require.NoError(t, err)
	require.Len(t, bundles, 1)
	assert.Equal(t, "custom-test", bundles[0].Name)
}
