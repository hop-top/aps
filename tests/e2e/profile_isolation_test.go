package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileIsolationConfig(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "new", "iso-test-profile", "--display-name", "Isolation Test Profile")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "iso-test-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "id: iso-test-profile")
	assert.Contains(t, stdout, "display_name: Isolation Test Profile")
}

func TestProfileWithContainerIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "container-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	content := `id: container-profile
display_name: Container Profile
isolation:
  level: container
  strict: true
  container:
    image: ubuntu:22.04
    network: bridge
`
	err = os.WriteFile(profilePath, []byte(content), 0644)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "container-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "level: container")
	assert.Contains(t, stdout, "image: ubuntu:22.04")
	assert.Contains(t, stdout, "strict: true")
}

func TestProfileWithPlatformIsolation(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "platform-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	content := `id: platform-profile
display_name: Platform Profile
isolation:
  level: platform
  platform:
    sandbox_id: sandbox-123
    name: test-sandbox
`
	err = os.WriteFile(profilePath, []byte(content), 0644)
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "profile", "show", "platform-profile")
	require.NoError(t, err)

	assert.Contains(t, stdout, "level: platform")
	assert.Contains(t, stdout, "sandbox_id: sandbox-123")
	assert.Contains(t, stdout, "name: test-sandbox")
}

func TestProfileInvalidIsolationLevel(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "invalid-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	content := `id: invalid-profile
display_name: Invalid Profile
isolation:
  level: invalid-level
`
	err = os.WriteFile(profilePath, []byte(content), 0644)
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "profile", "show", "invalid-profile")
	assert.Error(t, err)
	assert.Contains(t, stderr, "invalid isolation level")
}

func TestProfileContainerWithoutImage(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	profileDir := filepath.Join(home, ".local", "share", "aps", "profiles", "no-image-profile")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profilePath := filepath.Join(profileDir, "profile.yaml")
	content := `id: no-image-profile
display_name: No Image Profile
isolation:
  level: container
  container:
    network: bridge
`
	err = os.WriteFile(profilePath, []byte(content), 0644)
	require.NoError(t, err)

	_, stderr, err := runAPS(t, home, "profile", "show", "no-image-profile")
	assert.Error(t, err)
	assert.Contains(t, stderr, "requires an image")
}
