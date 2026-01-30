package capability_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oss-aps-cli/internal/core/capability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityLifecycle(t *testing.T) {
	// Setup temp home dir
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a dummy source capability
	sourceDir := filepath.Join(tmpHome, "source-cap")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "config.yaml"), []byte("key: value"), 0644)
	require.NoError(t, err)

	// Test Install
	err = capability.Install("test-cap", sourceDir)
	require.NoError(t, err)

	// Verify Installation
	caps, err := capability.List()
	require.NoError(t, err)
	assert.Len(t, caps, 1)
	assert.Equal(t, "test-cap", caps[0].Name)
	assert.Equal(t, capability.TypeManaged, caps[0].Type)

	// Verify file content copied
	destContent, err := os.ReadFile(filepath.Join(caps[0].Path, "config.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "key: value", string(destContent))

	// Test GenerateEnvExports
	exports, err := capability.GenerateEnvExports()
	require.NoError(t, err)
	assert.NotEmpty(t, exports)
	foundEnv := false
	for _, exp := range exports {
		if strings.Contains(exp, "APS_TEST_CAP_PATH=") {
			foundEnv = true
			break
		}
	}
	assert.True(t, foundEnv, "Expected APS_TEST_CAP_PATH export")

	// Test Delete
	err = capability.Delete("test-cap")
	require.NoError(t, err)

	// Verify Deletion
	caps, err = capability.List()
	require.NoError(t, err)
	assert.Len(t, caps, 0)
}

func TestSmartLinking(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Stub Smart Pattern for testing?
	// Since GetSmartPattern relies on a hardcoded list or existing registry, we might verify existing known patterns.
	// We'll rely on "copilot" being present in the registry (assuming it is).

	// We can't easily install "copilot" unless we have a source, but Link relies on an installed capability.
	// So let's install a dummy cap named "copilot".
	err := os.MkdirAll(filepath.Join(tmpHome, "source"), 0755)
	require.NoError(t, err)
	err = capability.Install("copilot", filepath.Join(tmpHome, "source"))
	require.NoError(t, err)

	// Test Link with "copilot" target name (should resolve to .github/agents/agent.agent.md)
	// But we need to pretend we are in a workspace.
	cwd := t.TempDir()
	// Mock os.Getwd? capability.Link uses os.Getwd().
	// We can't mock os.Getwd easily in standard Go without deeper hacks or modifying the code to accept a wd.
	// Skipping Link test that depends on CWD for now, or we can Chdir (might be flaky in parallel tests but ok for unit).

	origWd, _ := os.Getwd()
	os.Chdir(cwd)
	defer os.Chdir(origWd)

	err = capability.Link("copilot", "copilot") // Name matches pattern
	require.NoError(t, err)

	// Verify link created at expected default path
	// Copilot pattern: .github/agents/agent.agent.md
	expectedLink := filepath.Join(cwd, ".github", "agents", "agent.agent.md")
	info, err := os.Lstat(expectedLink)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)
}
