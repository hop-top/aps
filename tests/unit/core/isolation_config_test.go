package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"hop.top/aps/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestProfile(t *testing.T, id string, content string) string {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", id)
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profilePath := filepath.Join(profileDir, "profile.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(content), 0644))

	return tempDir
}

func overrideXDGDataHome(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("XDG_DATA_HOME")
	origAPS := os.Getenv("APS_DATA_PATH")
	os.Setenv("XDG_DATA_HOME", filepath.Join(dir, ".local", "share"))
	os.Unsetenv("APS_DATA_PATH")
	t.Cleanup(func() {
		os.Setenv("XDG_DATA_HOME", orig)
		if origAPS != "" {
			os.Setenv("APS_DATA_PATH", origAPS)
		}
	})
}

func TestIsolationConfig_DefaultLevel(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	profile, err := core.LoadProfile("test-profile")
	require.NoError(t, err)
	assert.Equal(t, core.IsolationProcess, profile.Isolation.Level)
}

func TestIsolationConfig_ProcessLevel(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
isolation:
  level: process
  strict: false
  fallback: true
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	profile, err := core.LoadProfile("test-profile")
	require.NoError(t, err)
	assert.Equal(t, core.IsolationProcess, profile.Isolation.Level)
	assert.False(t, profile.Isolation.Strict)
	assert.True(t, profile.Isolation.Fallback)
}

func TestIsolationConfig_PlatformLevel(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
isolation:
  level: platform
  strict: true
  platform:
    sandbox_id: sandbox-123
    name: test-sandbox
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	profile, err := core.LoadProfile("test-profile")
	require.NoError(t, err)
	assert.Equal(t, core.IsolationPlatform, profile.Isolation.Level)
	assert.True(t, profile.Isolation.Strict)
	assert.Equal(t, "sandbox-123", profile.Isolation.Platform.SandboxID)
	assert.Equal(t, "test-sandbox", profile.Isolation.Platform.Name)
}

func TestIsolationConfig_ContainerLevel(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
isolation:
  level: container
  container:
    image: ubuntu:22.04
    network: bridge
    volumes:
      - /host/path:/container/path
    resources:
      memory_mb: 512
      cpu_quota: 100
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	profile, err := core.LoadProfile("test-profile")
	require.NoError(t, err)
	assert.Equal(t, core.IsolationContainer, profile.Isolation.Level)
	assert.Equal(t, "ubuntu:22.04", profile.Isolation.Container.Image)
	assert.Equal(t, "bridge", profile.Isolation.Container.Network)
	assert.Len(t, profile.Isolation.Container.Volumes, 1)
	assert.Equal(t, 512, profile.Isolation.Container.Resources.MemoryMB)
	assert.Equal(t, 100, profile.Isolation.Container.Resources.CPUQuota)
}

func TestIsolationConfig_InvalidLevel(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
isolation:
  level: invalid
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	_, err := core.LoadProfile("test-profile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid isolation level")
}

func TestIsolationConfig_ContainerWithoutImage(t *testing.T) {
	content := `id: test-profile
display_name: Test Profile
isolation:
  level: container
  container:
    network: bridge
`
	tempDir := setupTestProfile(t, "test-profile", content)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	_, err := core.LoadProfile("test-profile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires an image")
}

func TestIsolationConfig_ValidateMethod(t *testing.T) {
	profile := &core.Profile{
		ID:          "test",
		DisplayName: "Test",
		Isolation: core.IsolationConfig{
			Level: core.IsolationProcess,
		},
	}

	err := profile.ValidateIsolation()
	assert.NoError(t, err)
}

func TestIsolationConfig_ValidateInvalid(t *testing.T) {
	profile := &core.Profile{
		ID:          "test",
		DisplayName: "Test",
		Isolation: core.IsolationConfig{
			Level: "invalid-level",
		},
	}

	err := profile.ValidateIsolation()
	assert.Error(t, err)
}

func TestSaveProfileWithIsolation(t *testing.T) {
	tempDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	overrideXDGDataHome(t, tempDir)

	profile := &core.Profile{
		ID:          "iso-profile",
		DisplayName: "Isolation Test",
		Isolation: core.IsolationConfig{
			Level:    core.IsolationContainer,
			Strict:   true,
			Fallback: false,
			Container: core.ContainerConfig{
				Image:   "ubuntu:22.04",
				Network: "none",
			},
		},
	}

	err := core.SaveProfile(profile)
	require.NoError(t, err)

	loaded, err := core.LoadProfile("iso-profile")
	require.NoError(t, err)
	assert.Equal(t, core.IsolationContainer, loaded.Isolation.Level)
	assert.True(t, loaded.Isolation.Strict)
	assert.Equal(t, "ubuntu:22.04", loaded.Isolation.Container.Image)
}
