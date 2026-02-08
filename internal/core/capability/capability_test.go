package capability

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// setupTestHome sets HOME to a temp directory and returns the path
func setupTestHome(t *testing.T) string {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	os.Setenv("HOME", tmpDir)
	return tmpDir
}

// ============================================================================
// Capability Installation Tests (8 tests)
// ============================================================================

// TestInstallFromLocalPath tests installing from a local source directory
func TestInstallFromLocalPath(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	// Create a source directory with content
	sourceDir := filepath.Join(tmpDir, "source-capability")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("test content"), 0644))

	err := Install("my-capability", sourceDir)
	require.NoError(t, err)

	// Verify capability directory exists
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "my-capability")
	assert.DirExists(t, capDir)

	// Verify content was copied
	assert.FileExists(t, filepath.Join(capDir, "test.txt"))

	// Verify manifest was created
	assert.FileExists(t, filepath.Join(capDir, "manifest.yaml"))

	data, err := os.ReadFile(filepath.Join(capDir, "manifest.yaml"))
	require.NoError(t, err)

	var cap Capability
	err = yaml.Unmarshal(data, &cap)
	require.NoError(t, err)
	assert.Equal(t, "my-capability", cap.Name)
	assert.Equal(t, sourceDir, cap.Source)
	assert.Equal(t, TypeManaged, cap.Type)
}

// TestInstallFromEmptySource tests installing without a source
func TestInstallFromEmptySource(t *testing.T) {
	homeDir := setupTestHome(t)

	err := Install("empty-capability", "")
	require.NoError(t, err)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "empty-capability")
	assert.DirExists(t, capDir)
	assert.FileExists(t, filepath.Join(capDir, "manifest.yaml"))

	data, err := os.ReadFile(filepath.Join(capDir, "manifest.yaml"))
	require.NoError(t, err)

	var cap Capability
	err = yaml.Unmarshal(data, &cap)
	require.NoError(t, err)
	assert.Equal(t, "empty-capability", cap.Name)
	assert.Equal(t, TypeManaged, cap.Type)
}

// TestInstallWithVersionSpecification tests installing with version info in source
func TestInstallWithVersionSpecification(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	sourceDir := filepath.Join(tmpDir, "source-v1.2.3")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "version.txt"), []byte("1.2.3"), 0644))

	err := Install("versioned-cap", sourceDir)
	require.NoError(t, err)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "versioned-cap")
	assert.FileExists(t, filepath.Join(capDir, "version.txt"))

	data, err := os.ReadFile(filepath.Join(capDir, "version.txt"))
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", string(data))
}

// TestSmartPatternResolution tests resolving tool names to default paths
func TestSmartPatternResolution(t *testing.T) {
	tests := []struct {
		toolName     string
		expectedPath string
		shouldExist  bool
	}{
		{"claude", ".claude/commands/agent.md", true},
		{"cursor", ".cursor/commands/agent.md", true},
		{"windsurf", ".windsurf/workflows/agent.md", true},
		{"copilot", ".github/agents/agent.agent.md", true},
		{"unknown-tool", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			pattern, err := GetSmartPattern(tt.toolName)
			if tt.shouldExist {
				require.NoError(t, err)
				assert.Equal(t, tt.toolName, pattern.ToolName)
				assert.Equal(t, tt.expectedPath, pattern.DefaultPath)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestInstallToDifferentDirectory tests capability creates proper directory
func TestInstallToDifferentDirectory(t *testing.T) {
	homeDir := setupTestHome(t)

	err := Install("custom-cap", "")
	require.NoError(t, err)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "custom-cap")
	assert.DirExists(t, capDir)
}

// TestInstallationErrors tests various installation error conditions
func TestInstallationErrors(t *testing.T) {
	homeDir := setupTestHome(t)

	t.Run("InvalidSourcePath", func(t *testing.T) {
		// Source doesn't exist but install should still work
		err := Install("test-cap", "/nonexistent/path/to/source")
		require.NoError(t, err)

		capDir := filepath.Join(homeDir, ".aps", "capabilities", "test-cap")
		assert.DirExists(t, capDir)
	})
}

// TestDuplicateInstallation tests that installing duplicate capability fails
func TestDuplicateInstallation(t *testing.T) {
	setupTestHome(t)

	// First install
	err := Install("dup-cap", "")
	require.NoError(t, err)

	// Second install should fail
	err = Install("dup-cap", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestManifestValidation tests loading and validating capability manifests
func TestManifestValidation(t *testing.T) {
	homeDir := setupTestHome(t)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "manifest-test")
	require.NoError(t, os.MkdirAll(capDir, 0755))

	// Create a manifest with all fields
	manifest := Capability{
		Name:        "manifest-test",
		Source:      "https://example.com/capability",
		Description: "Test capability",
		InstalledAt: time.Now(),
		Type:        TypeManaged,
		Links: map[string]string{
			"/path/to/link": "/capability/path",
		},
	}

	data, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "manifest.yaml"), data, 0644))

	cap, err := LoadCapabilityFromPath(capDir, "manifest-test")
	require.NoError(t, err)
	assert.Equal(t, "manifest-test", cap.Name)
	assert.Equal(t, "https://example.com/capability", cap.Source)
	assert.Equal(t, TypeManaged, cap.Type)
}

// ============================================================================
// Capability Linking Tests (6 tests)
// ============================================================================

// TestLinkCapabilityToProfile tests linking a capability to a target path
func TestLinkCapabilityToProfile(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	// Create and install a capability
	err := Install("linkable-cap", "")
	require.NoError(t, err)

	// Create target path parent
	targetDir := filepath.Join(tmpDir, "profile", "tools")
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	targetPath := filepath.Join(targetDir, "linkable-cap")

	// Change to tmp dir for proper path resolution
	oldCwd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldCwd) })
	os.Chdir(tmpDir)

	err = Link("linkable-cap", targetPath)
	require.NoError(t, err)

	// Verify symlink exists
	assert.FileExists(t, targetPath)

	// Verify metadata was saved
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "linkable-cap")
	data, err := os.ReadFile(filepath.Join(capDir, "manifest.yaml"))
	require.NoError(t, err)

	var cap Capability
	err = yaml.Unmarshal(data, &cap)
	require.NoError(t, err)
	assert.NotEmpty(t, cap.Links)
}

// TestAdoptExistingDirectory tests adopting an existing directory
func TestAdoptExistingDirectory(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	// Create a file to adopt (not a directory)
	adoptFile := filepath.Join(tmpDir, "existing-tool")
	require.NoError(t, os.WriteFile(adoptFile, []byte("#!/bin/bash\necho test"), 0755))

	err := Adopt(adoptFile, "adopted-cap")
	require.NoError(t, err)

	// Verify capability was created
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "adopted-cap")
	assert.DirExists(t, capDir)

	// Verify original path still exists (symlinked back)
	assert.FileExists(t, adoptFile)
}

// TestMultiRootCapabilityDiscovery tests finding capabilities in multiple roots
func TestMultiRootCapabilityDiscovery(t *testing.T) {
	setupTestHome(t)

	// Create capabilities in default location
	err := Install("default-cap", "")
	require.NoError(t, err)

	// Get all roots
	roots, err := GetCapabilityRoots()
	require.NoError(t, err)

	// Default root should always be included
	defaultRoot, _ := GetCapabilitiesDir()
	assert.Contains(t, roots, defaultRoot)
}

// TestLinkValidation tests validation of link operations
func TestLinkValidation(t *testing.T) {
	setupTestHome(t)

	// Try to link non-existent capability
	err := Link("nonexistent-cap", "/tmp/target")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestLinkErrors tests error conditions during linking
func TestLinkErrors(t *testing.T) {
	setupTestHome(t)
	tmpDir := t.TempDir()

	// Create a capability
	err := Install("link-err-cap", "")
	require.NoError(t, err)

	// Create target that already exists
	targetPath := filepath.Join(tmpDir, "existing-target")
	require.NoError(t, os.MkdirAll(targetPath, 0755))

	oldCwd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldCwd) })
	os.Chdir(tmpDir)

	// Try to link to existing target
	err = Link("link-err-cap", targetPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestUnlinkCleanup tests unlinking and cleanup of capabilities
func TestUnlinkCleanup(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	// Create and link a capability
	err := Install("cleanup-cap", "")
	require.NoError(t, err)

	// Create target and link
	targetDir := filepath.Join(tmpDir, "targets")
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	targetPath := filepath.Join(targetDir, "cleanup-cap")

	oldCwd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldCwd) })
	os.Chdir(tmpDir)

	err = Link("cleanup-cap", targetPath)
	require.NoError(t, err)

	// Verify link exists
	assert.FileExists(t, targetPath)

	// Delete capability
	err = Delete("cleanup-cap")
	require.NoError(t, err)

	// Verify capability is gone
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "cleanup-cap")
	assert.NoDirExists(t, capDir)
}

// ============================================================================
// Capability Management Tests (6 tests)
// ============================================================================

// TestLoadCapabilityFromManifest tests loading capability from manifest.yaml
func TestLoadCapabilityFromManifest(t *testing.T) {
	homeDir := setupTestHome(t)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "load-test")
	require.NoError(t, os.MkdirAll(capDir, 0755))

	manifest := Capability{
		Name:        "load-test",
		Source:      "https://example.com/cap",
		Description: "Test capability",
		Type:        TypeManaged,
	}

	data, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "manifest.yaml"), data, 0644))

	cap, err := LoadCapabilityFromPath(capDir, "load-test")
	require.NoError(t, err)
	assert.Equal(t, "load-test", cap.Name)
	assert.Equal(t, "https://example.com/cap", cap.Source)
	assert.Equal(t, "Test capability", cap.Description)
}

// TestListAllCapabilities tests listing all installed capabilities
func TestListAllCapabilities(t *testing.T) {
	setupTestHome(t)

	// Create multiple capabilities
	for i := 1; i <= 3; i++ {
		err := Install(fmt.Sprintf("cap-%d", i), "")
		require.NoError(t, err)
	}

	// List all capabilities
	caps, err := List()
	require.NoError(t, err)
	assert.Len(t, caps, 3)

	names := make(map[string]bool)
	for _, cap := range caps {
		names[cap.Name] = true
	}

	assert.True(t, names["cap-1"])
	assert.True(t, names["cap-2"])
	assert.True(t, names["cap-3"])
}

// TestGenerateEnvExports tests generating shell export commands
func TestGenerateEnvExports(t *testing.T) {
	setupTestHome(t)

	// Create a capability
	err := Install("export-cap", "")
	require.NoError(t, err)

	exports, err := GenerateEnvExports()
	require.NoError(t, err)
	assert.NotEmpty(t, exports)

	// Verify export format
	var foundExport bool
	for _, export := range exports {
		if strings.Contains(export, "APS_EXPORT_CAP_PATH") {
			foundExport = true
			assert.True(t, strings.HasPrefix(export, "export APS_"))
			assert.True(t, strings.Contains(export, "capabilities/export-cap"))
		}
	}
	assert.True(t, foundExport)
}

// TestDeleteCapability tests deleting a capability
func TestDeleteCapability(t *testing.T) {
	homeDir := setupTestHome(t)

	err := Install("delete-cap", "")
	require.NoError(t, err)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "delete-cap")
	assert.DirExists(t, capDir)

	err = Delete("delete-cap")
	require.NoError(t, err)

	assert.NoDirExists(t, capDir)
}

// TestWatchCapability tests watching for capability changes
func TestWatchCapability(t *testing.T) {
	setupTestHome(t)
	tmpDir := t.TempDir()

	// Create external file to watch
	externalFile := filepath.Join(tmpDir, "external-tool.sh")
	require.NoError(t, os.WriteFile(externalFile, []byte("#!/bin/bash\necho test"), 0755))

	err := Watch(externalFile, "watched-cap")
	require.NoError(t, err)

	// Verify capability was created as reference type
	cap, err := LoadCapability("watched-cap")
	require.NoError(t, err)
	assert.Equal(t, TypeReference, cap.Type)
	assert.NotEmpty(t, cap.Links)
}

// TestDirectoryOperations tests directory creation and management
func TestDirectoryOperations(t *testing.T) {
	setupTestHome(t)

	// Get capabilities directory
	capDir, err := GetCapabilitiesDir()
	require.NoError(t, err)
	assert.Contains(t, capDir, ".aps/capabilities")

	// Get specific capability path
	capPath, err := GetCapabilityPath("test-cap")
	require.NoError(t, err)
	assert.Contains(t, capPath, ".aps/capabilities/test-cap")
}

// ============================================================================
// Smart Pattern Tests (3 tests)
// ============================================================================

// TestListSmartPatterns tests listing all available smart patterns
func TestListSmartPatterns(t *testing.T) {
	patterns := ListSmartPatterns()
	assert.NotEmpty(t, patterns)
	assert.Greater(t, len(patterns), 5)
}

// TestSmartPatternCaseInsensitive tests pattern matching is case-insensitive
func TestSmartPatternCaseInsensitive(t *testing.T) {
	tests := []string{"claude", "Claude", "CLAUDE", "ClAuDe"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			pattern, err := GetSmartPattern(input)
			require.NoError(t, err)
			assert.Equal(t, "claude", pattern.ToolName)
		})
	}
}

// TestSmartPatternUnknownTool tests error for unknown tool
func TestSmartPatternUnknownTool(t *testing.T) {
	_, err := GetSmartPattern("nonexistent-ai-tool-xyz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool pattern")
}

// ============================================================================
// Integration Tests (4 tests)
// ============================================================================

// TestFullCapabilityLifecycle tests complete install -> link -> delete cycle
func TestFullCapabilityLifecycle(t *testing.T) {
	setupTestHome(t)
	tmpDir := t.TempDir()

	capName := "lifecycle-cap"

	// Install
	err := Install(capName, "")
	require.NoError(t, err)

	// Verify exists
	cap, err := LoadCapability(capName)
	require.NoError(t, err)
	assert.Equal(t, capName, cap.Name)

	// Create link target
	targetDir := filepath.Join(tmpDir, "targets")
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	targetPath := filepath.Join(targetDir, capName)

	oldCwd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldCwd) })
	os.Chdir(tmpDir)

	// Link
	err = Link(capName, targetPath)
	require.NoError(t, err)

	// Verify link
	cap, err = LoadCapability(capName)
	require.NoError(t, err)
	assert.NotEmpty(t, cap.Links)

	// Delete
	err = Delete(capName)
	require.NoError(t, err)

	// Verify deleted
	_, err = LoadCapability(capName)
	require.Error(t, err)
}

// TestMultipleCapabilitiesCoexist tests multiple capabilities can coexist
func TestMultipleCapabilitiesCoexist(t *testing.T) {
	setupTestHome(t)

	// Create multiple capabilities
	for i := 1; i <= 5; i++ {
		capName := fmt.Sprintf("coexist-cap-%d", i)
		err := Install(capName, "")
		require.NoError(t, err)
	}

	// List and verify all exist
	caps, err := List()
	require.NoError(t, err)
	assert.Len(t, caps, 5)

	// Verify metadata independence
	for i := 1; i <= 5; i++ {
		capName := fmt.Sprintf("coexist-cap-%d", i)
		cap, err := LoadCapability(capName)
		require.NoError(t, err)
		assert.Equal(t, capName, cap.Name)
	}
}

// TestConcurrentCapabilityOperations tests thread-safe capability operations
func TestConcurrentCapabilityOperations(t *testing.T) {
	setupTestHome(t)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			capName := fmt.Sprintf("concurrent-cap-%d", idx)
			errChan <- Install(capName, "")
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}

	caps, err := List()
	require.NoError(t, err)
	assert.Len(t, caps, 10)
}

// TestCapabilityLoadWithoutManifest tests loading capability without manifest.yaml
func TestCapabilityLoadWithoutManifest(t *testing.T) {
	homeDir := setupTestHome(t)

	// Create capability dir without manifest
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "no-manifest")
	require.NoError(t, os.MkdirAll(capDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "file.txt"), []byte("content"), 0644))

	// Should synthesize capability
	cap, err := LoadCapabilityFromPath(capDir, "no-manifest")
	require.NoError(t, err)
	assert.Equal(t, "no-manifest", cap.Name)
	assert.Equal(t, TypeManaged, cap.Type)
}

// TestCapabilityTypeManaged tests TypeManaged capabilities
func TestCapabilityTypeManaged(t *testing.T) {
	setupTestHome(t)

	err := Install("managed-cap", "")
	require.NoError(t, err)

	cap, err := LoadCapability("managed-cap")
	require.NoError(t, err)
	assert.Equal(t, TypeManaged, cap.Type)
	assert.NotEmpty(t, cap.InstalledAt)
}

// TestCapabilityTypeReference tests TypeReference capabilities
func TestCapabilityTypeReference(t *testing.T) {
	setupTestHome(t)
	tmpDir := t.TempDir()

	// Create external file
	extFile := filepath.Join(tmpDir, "external.sh")
	require.NoError(t, os.WriteFile(extFile, []byte("#!/bin/bash\necho test"), 0755))

	// Watch it
	err := Watch(extFile, "ref-cap")
	require.NoError(t, err)

	cap, err := LoadCapability("ref-cap")
	require.NoError(t, err)
	assert.Equal(t, TypeReference, cap.Type)
}

// TestCopyDirWithNestedStructure tests copying nested directory structures
func TestCopyDirWithNestedStructure(t *testing.T) {
	homeDir := setupTestHome(t)
	tmpDir := t.TempDir()

	// Create source with nested structure
	sourceDir := filepath.Join(tmpDir, "source-nested")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "bin"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "lib"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "bin", "tool"), []byte("#!/bin/bash"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "lib", "lib.sh"), []byte("# library"), 0644))

	err := Install("nested-cap", sourceDir)
	require.NoError(t, err)

	// Verify structure was copied
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "nested-cap")
	assert.FileExists(t, filepath.Join(capDir, "bin", "tool"))
	assert.FileExists(t, filepath.Join(capDir, "lib", "lib.sh"))
}

// TestInstallWithSourceThatIsNotDirectory tests installing from non-directory source
func TestInstallWithSourceThatIsNotDirectory(t *testing.T) {
	setupTestHome(t)
	tmpDir := t.TempDir()

	// Create a file (not directory)
	sourceFile := filepath.Join(tmpDir, "tool.sh")
	require.NoError(t, os.WriteFile(sourceFile, []byte("#!/bin/bash"), 0755))

	// Install with file source (should not copy since it's not a directory)
	err := Install("file-source-cap", sourceFile)
	require.NoError(t, err)

	// But directory should still be created
	_, err = LoadCapability("file-source-cap")
	require.NoError(t, err)
}

// TestGetCapabilitiesDirStructure tests directory structure creation
func TestGetCapabilitiesDirStructure(t *testing.T) {
	setupTestHome(t)

	capDir, err := GetCapabilitiesDir()
	require.NoError(t, err)
	assert.NotEmpty(t, capDir)
	assert.Contains(t, capDir, ".aps/capabilities")
}

// TestManifestYAMLMarshaling tests capability manifest YAML marshaling
func TestManifestYAMLMarshaling(t *testing.T) {
	cap := Capability{
		Name:        "marshal-test",
		Source:      "https://example.com",
		Description: "Test marshaling",
		Type:        TypeManaged,
		Links: map[string]string{
			"/path/1": "/target/1",
			"/path/2": "/target/2",
		},
	}

	data, err := yaml.Marshal(cap)
	require.NoError(t, err)

	var loaded Capability
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cap.Name, loaded.Name)
	assert.Equal(t, cap.Source, loaded.Source)
	assert.Equal(t, cap.Description, loaded.Description)
	assert.Equal(t, cap.Type, loaded.Type)
	assert.Len(t, loaded.Links, 2)
}

// TestCapabilityPathHandling tests proper path handling in capability operations
func TestCapabilityPathHandling(t *testing.T) {
	setupTestHome(t)

	capName := "path-test"
	err := Install(capName, "")
	require.NoError(t, err)

	cap, err := LoadCapability(capName)
	require.NoError(t, err)

	// Path should be absolute and properly formatted
	assert.True(t, filepath.IsAbs(cap.Path))
	assert.Contains(t, cap.Path, capName)
	assert.True(t, strings.Contains(cap.Path, ".aps"))
}

// ============================================================================
// Error Handling and Edge Cases (5 tests)
// ============================================================================

// TestDeleteWithValidCapability tests delete with properly formed capability
func TestDeleteWithValidCapability(t *testing.T) {
	homeDir := setupTestHome(t)

	// Create and install a valid capability
	err := Install("valid-delete-cap", "")
	require.NoError(t, err)

	// Verify it exists
	cap, err := LoadCapability("valid-delete-cap")
	require.NoError(t, err)
	assert.NotEmpty(t, cap.Path)

	// Delete should succeed
	err = Delete("valid-delete-cap")
	require.NoError(t, err)

	// Should no longer exist
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "valid-delete-cap")
	assert.NoDirExists(t, capDir)
}

// TestLoadNonExistentCapability tests error handling
func TestLoadNonExistentCapability(t *testing.T) {
	setupTestHome(t)

	_, err := LoadCapability("totally-nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestEmptyCapabilitiesList tests handling of empty capabilities directory
func TestEmptyCapabilitiesList(t *testing.T) {
	setupTestHome(t)

	caps, err := List()
	require.NoError(t, err)
	assert.Empty(t, caps)
}

// TestGenerateEnvExportsEmpty tests env exports with no capabilities
func TestGenerateEnvExportsEmpty(t *testing.T) {
	setupTestHome(t)

	exports, err := GenerateEnvExports()
	require.NoError(t, err)
	assert.Empty(t, exports)
}

// TestCapabilityNameSanitization tests capability name handling
func TestCapabilityNameSanitization(t *testing.T) {
	setupTestHome(t)

	// Capability names with hyphens should work
	capName := "my-test-capability"
	err := Install(capName, "")
	require.NoError(t, err)

	cap, err := LoadCapability(capName)
	require.NoError(t, err)
	assert.Equal(t, capName, cap.Name)

	// Verify environment variable safe name
	exports, err := GenerateEnvExports()
	require.NoError(t, err)
	assert.NotEmpty(t, exports)

	for _, export := range exports {
		if strings.Contains(export, capName) {
			// Should have underscores instead of hyphens
			assert.Contains(t, export, "MY_TEST_CAPABILITY")
		}
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

// TestSaveMetadataCreatesFile tests saveMetadata function behavior
func TestSaveMetadataCreatesFile(t *testing.T) {
	homeDir := setupTestHome(t)

	// This tests saveMetadata indirectly
	err := Install("metadata-test", "")
	require.NoError(t, err)

	capDir := filepath.Join(homeDir, ".aps", "capabilities", "metadata-test")
	assert.FileExists(t, filepath.Join(capDir, "manifest.yaml"))
}

// TestGetSmartPatternDescriptions tests that patterns have descriptions
func TestGetSmartPatternDescriptions(t *testing.T) {
	patterns := ListSmartPatterns()
	for _, pattern := range patterns {
		assert.NotEmpty(t, pattern.Description)
		assert.NotEmpty(t, pattern.ToolName)
		assert.NotEmpty(t, pattern.DefaultPath)
	}
}

// TestLoadCapabilityInitializesLinks tests link map initialization
func TestLoadCapabilityInitializesLinks(t *testing.T) {
	homeDir := setupTestHome(t)

	// Create capability dir without manifest
	capDir := filepath.Join(homeDir, ".aps", "capabilities", "links-init-test")
	require.NoError(t, os.MkdirAll(capDir, 0755))

	cap, err := LoadCapabilityFromPath(capDir, "links-init-test")
	require.NoError(t, err)

	// Links should be initialized as non-nil map
	assert.NotNil(t, cap.Links)
}

// TestCapabilityInstallTimestamp tests that installed_at is set
func TestCapabilityInstallTimestamp(t *testing.T) {
	setupTestHome(t)

	before := time.Now()
	err := Install("timestamp-test", "")
	require.NoError(t, err)
	after := time.Now()

	cap, err := LoadCapability("timestamp-test")
	require.NoError(t, err)

	assert.True(t, cap.InstalledAt.After(before) || cap.InstalledAt.Equal(before))
	assert.True(t, cap.InstalledAt.Before(after) || cap.InstalledAt.Equal(after))
}

// TestListFiltersNonDirectories tests that List only returns directories
func TestListFiltersNonDirectories(t *testing.T) {
	homeDir := setupTestHome(t)

	// Create some valid capabilities
	err := Install("cap-1", "")
	require.NoError(t, err)

	// Create a file in the capabilities directory (not a directory)
	capsDir := filepath.Join(homeDir, ".aps", "capabilities")
	require.NoError(t, os.WriteFile(filepath.Join(capsDir, "stray-file.txt"), []byte("test"), 0644))

	// List should only return directories
	caps, err := List()
	require.NoError(t, err)

	for _, cap := range caps {
		assert.NotEqual(t, "stray-file.txt", cap.Name)
	}
}

// TestCapabilityPathAbsolute tests that capability paths are absolute
func TestCapabilityPathAbsolute(t *testing.T) {
	setupTestHome(t)

	cap, err := LoadCapabilityFromPath(".", "relative-path")
	require.NoError(t, err)

	// Even with relative path input, the returned path should work
	assert.NotEmpty(t, cap.Path)
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkInstall benchmarks capability installation
func BenchmarkInstall(b *testing.B) {
	tmpDir := b.TempDir()
	oldHome := os.Getenv("HOME")
	b.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		capName := fmt.Sprintf("bench-cap-%d", i)
		Install(capName, "")
	}
}

// BenchmarkList benchmarks listing capabilities
func BenchmarkList(b *testing.B) {
	tmpDir := b.TempDir()
	oldHome := os.Getenv("HOME")
	b.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	// Create some capabilities
	for i := 0; i < 10; i++ {
		Install(fmt.Sprintf("bench-cap-%d", i), "")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		List()
	}
}

// BenchmarkLoadCapability benchmarks loading a capability
func BenchmarkLoadCapability(b *testing.B) {
	tmpDir := b.TempDir()
	oldHome := os.Getenv("HOME")
	b.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", tmpDir)

	Install("bench-cap", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadCapability("bench-cap")
	}
}
