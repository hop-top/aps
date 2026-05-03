package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveAPSContext_Workspace asserts -C my-workspace resolves to the
// workspace directory (data/workspaces/<id>).
func TestResolveAPSContext_Workspace(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	wsID := "my-workspace"
	wsDir := filepath.Join(dataDir, "workspaces", wsID)
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	got, err := resolveAPSContext(wsID)
	require.NoError(t, err)

	want, _ := filepath.EvalSymlinks(wsDir)
	gotReal, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, want, gotReal)
}

// TestResolveAPSContext_Profile asserts -C my-profile resolves to the
// profile directory (data/profiles/<id>) when no workspace exists with
// that name.
func TestResolveAPSContext_Profile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	profileID := "my-profile"
	profileDir := filepath.Join(dataDir, "profiles", profileID)
	require.NoError(t, os.MkdirAll(profileDir, 0o755))

	got, err := resolveAPSContext(profileID)
	require.NoError(t, err)

	want, _ := filepath.EvalSymlinks(profileDir)
	gotReal, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, want, gotReal)
}

// TestResolveAPSContext_LiteralPathFallthrough asserts that a path the
// resolver doesn't recognise returns an error so kit's resolveChdir can
// emit its literal "not a directory" message. Existing literal paths
// are stat-handled by kit before the resolver is invoked, so this only
// covers the fall-through contract.
func TestResolveAPSContext_LiteralPathFallthrough(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	// "/tmp" exists as a real directory; kit's resolveChdir handles it
	// via os.Stat before calling our resolver. Simulate the resolver
	// being called with a non-workspace, non-profile target.
	_, err := resolveAPSContext("/tmp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"/tmp"`)
}

// TestResolveAPSContext_WorkspaceWinsOverProfile asserts that when a
// name exists as both a workspace and a profile, the workspace wins
// (matching the documented resolution order).
func TestResolveAPSContext_WorkspaceWinsOverProfile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	name := "shared-name"
	wsDir := filepath.Join(dataDir, "workspaces", name)
	pfDir := filepath.Join(dataDir, "profiles", name)
	require.NoError(t, os.MkdirAll(wsDir, 0o755))
	require.NoError(t, os.MkdirAll(pfDir, 0o755))

	got, err := resolveAPSContext(name)
	require.NoError(t, err)

	want, _ := filepath.EvalSymlinks(wsDir)
	gotReal, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, want, gotReal)
}
