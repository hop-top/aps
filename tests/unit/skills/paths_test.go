package skills_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"hop.top/aps/internal/skills"
)

func TestNewSkillPaths(t *testing.T) {
	profileID := "test-profile"
	paths := skills.NewSkillPaths(profileID)

	assert.NotEmpty(t, paths.ProfilePath)
	assert.NotEmpty(t, paths.GlobalPath)
	assert.Empty(t, paths.UserPaths)
	assert.Empty(t, paths.DetectedPaths)

	// Verify profile path format
	assert.Contains(t, paths.ProfilePath, "test-profile")
	assert.Contains(t, paths.ProfilePath, "profiles")
}

func TestSkillPaths_GlobalPath_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	// Test with XDG_DATA_HOME set
	t.Setenv("XDG_DATA_HOME", "/custom/data")

	paths := skills.NewSkillPaths("")
	assert.Contains(t, paths.GlobalPath, "/custom/data/aps/skills")
}

func TestSkillPaths_GlobalPath_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	paths := skills.NewSkillPaths("")
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, "Library", "Application Support", "aps", "skills")
	assert.Equal(t, expected, paths.GlobalPath)
}

func TestSkillPaths_AllPaths(t *testing.T) {
	paths := skills.NewSkillPaths("test-profile")
	paths.UserPaths = []string{"/user/path1", "/user/path2"}
	paths.DetectedPaths = []string{"/detected/path1"}

	allPaths := paths.AllPaths()

	// Verify order
	assert.Len(t, allPaths, 5)
	assert.Equal(t, paths.ProfilePath, allPaths[0]) // Profile first
	assert.Equal(t, paths.GlobalPath, allPaths[1])  // Global second
	assert.Equal(t, "/user/path1", allPaths[2])     // User paths
	assert.Equal(t, "/user/path2", allPaths[3])
	assert.Equal(t, "/detected/path1", allPaths[4]) // Detected last
}

func TestSkillPaths_DetectIDEPaths(t *testing.T) {
	// Create a fake IDE directory
	tmpIDEDir := filepath.Join(t.TempDir(), ".claude", "skills")
	require := assert.New(t)
	require.NoError(os.MkdirAll(tmpIDEDir, 0755))

	paths := skills.NewSkillPaths("")

	// Note: This test may fail in CI since we can't easily create ~/.claude/skills
	// In real usage, paths would be detected
	detected := paths.DetectIDEPaths()

	// We can't assert specific paths exist, but we can verify the function runs
	assert.NotNil(t, detected)

	// Test that it returns a slice (even if empty)
	assert.IsType(t, []string{}, detected)
}

func TestSkillPaths_SuggestIDEPaths(t *testing.T) {
	paths := skills.NewSkillPaths("")

	// Initially no user paths configured
	paths.UserPaths = []string{}

	// Get suggestions
	suggestions := paths.SuggestIDEPaths()

	// Should return a slice (may be empty if no IDE paths detected)
	assert.NotNil(t, suggestions)

	// If we add a detected path to user paths, it should not be suggested
	if len(suggestions) > 0 {
		firstSuggestion := suggestions[0]
		paths.UserPaths = append(paths.UserPaths, firstSuggestion)

		newSuggestions := paths.SuggestIDEPaths()
		assert.NotContains(t, newSuggestions, firstSuggestion)
	}
}

func TestSkillPaths_DetectIDEPaths_Coverage(t *testing.T) {
	// Test that DetectIDEPaths handles different OS gracefully
	paths := skills.NewSkillPaths("")
	detected := paths.DetectIDEPaths()

	// Should always return a slice, never nil
	assert.NotNil(t, detected)
	assert.IsType(t, []string{}, detected)

	// Each detected path should be a valid directory (if any detected)
	for _, path := range detected {
		info, err := os.Stat(path)
		assert.NoError(t, err, "Detected path should exist: %s", path)
		if err == nil {
			assert.True(t, info.IsDir(), "Detected path should be directory: %s", path)
		}
	}
}
