package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func TestLinkProfile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "link-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{ID: "link-test", DisplayName: "Link Test"}
	require.NoError(t, core.SaveProfile(profile))

	err := LinkProfile("link-test", "dev-project", "global")
	require.NoError(t, err)

	loaded, err := core.LoadProfile("link-test")
	require.NoError(t, err)
	require.NotNil(t, loaded.Workspace)
	assert.Equal(t, "dev-project", loaded.Workspace.Name)
	assert.Equal(t, "global", loaded.Workspace.Scope)
}

func TestUnlinkProfile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "unlink-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{
		ID:          "unlink-test",
		DisplayName: "Unlink Test",
		Workspace: &core.WorkspaceLink{
			Name:  "dev-project",
			Scope: "global",
		},
	}
	require.NoError(t, core.SaveProfile(profile))

	err := UnlinkProfile("unlink-test")
	require.NoError(t, err)

	loaded, err := core.LoadProfile("unlink-test")
	require.NoError(t, err)
	assert.Nil(t, loaded.Workspace)
}

func TestGetLinkedWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "linked-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{
		ID:          "linked-test",
		DisplayName: "Linked Test",
		Workspace: &core.WorkspaceLink{
			Name:  "my-workspace",
			Scope: "global",
		},
	}
	require.NoError(t, core.SaveProfile(profile))

	link, err := GetLinkedWorkspace("linked-test")
	require.NoError(t, err)
	require.NotNil(t, link)
	assert.Equal(t, "my-workspace", link.Name)
}

func TestGetLinkedWorkspaceNone(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "nolink-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{ID: "nolink-test", DisplayName: "No Link"}
	require.NoError(t, core.SaveProfile(profile))

	link, err := GetLinkedWorkspace("nolink-test")
	require.NoError(t, err)
	assert.Nil(t, link)
}
