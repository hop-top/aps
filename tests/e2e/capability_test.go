package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityCommands(t *testing.T) {
	// Setup isolated HOME
	tmpHome := t.TempDir()
	env := append(os.Environ(), fmt.Sprintf("HOME=%s", tmpHome))

	// 1. Create a dummy capability source
	sourceDir := filepath.Join(tmpHome, "test-source")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "tool.txt"), []byte("tool"), 0644)
	require.NoError(t, err)

	// 2. Install
	// aps capability install <source> --name mytool
	cmd := exec.Command(apsBinary, "capability", "install", sourceDir, "--name", "mytool")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Install failed: %s", string(out))
	assert.Contains(t, string(out), "installed successfully")

	// 3. List
	cmd = exec.Command(apsBinary, "capability", "list")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "List failed: %s", string(out))
	assert.Contains(t, string(out), "mytool")
	assert.Contains(t, string(out), "managed")

	// 4. Env
	cmd = exec.Command(apsBinary, "env")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "Env failed: %s", string(out))
	assert.Contains(t, string(out), "export APS_MYTOOL_PATH=")

	// 5. Delete
	cmd = exec.Command(apsBinary, "capability", "delete", "mytool")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "Delete failed: %s", string(out))
	assert.Contains(t, string(out), "deleted")

	// 6. Verify List empty
	cmd = exec.Command(apsBinary, "capability", "list")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "No capabilities installed")
}
