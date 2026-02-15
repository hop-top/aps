package collaboration_test

import (
	"context"
	"testing"

	"oss-aps-cli/internal/core/collaboration"
	"oss-aps-cli/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) (*collaboration.Registry, *collaboration.Manager) {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	return collaboration.NewRegistry(s), collaboration.NewManager(s)
}

func TestRegistry_ListWorkspaces(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	cfg1 := collaboration.WorkspaceConfig{Name: "alpha", OwnerProfileID: "owner-1"}
	_, err := mgr.Create(ctx, cfg1)
	require.NoError(t, err)

	cfg2 := collaboration.WorkspaceConfig{Name: "beta", OwnerProfileID: "owner-1"}
	_, err = mgr.Create(ctx, cfg2)
	require.NoError(t, err)

	workspaces, err := reg.ListWorkspaces(collaboration.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

func TestRegistry_ListWorkspaces_FilterByName(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	cfg1 := collaboration.WorkspaceConfig{Name: "alpha", OwnerProfileID: "owner-1"}
	_, err := mgr.Create(ctx, cfg1)
	require.NoError(t, err)

	cfg2 := collaboration.WorkspaceConfig{Name: "beta", OwnerProfileID: "owner-1"}
	_, err = mgr.Create(ctx, cfg2)
	require.NoError(t, err)

	workspaces, err := reg.ListWorkspaces(collaboration.ListOptions{
		Filters: map[string]string{"name": "beta"},
	})
	require.NoError(t, err)
	assert.Len(t, workspaces, 1)
	assert.Equal(t, "beta", workspaces[0].Config.Name)
}

func TestRegistry_ListWorkspaces_Pagination(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cfg := collaboration.WorkspaceConfig{
			Name:           "ws-" + string(rune('a'+i)),
			OwnerProfileID: "owner-1",
		}
		_, err := mgr.Create(ctx, cfg)
		require.NoError(t, err)
	}

	workspaces, err := reg.ListWorkspaces(collaboration.ListOptions{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, workspaces, 3)

	workspaces, err = reg.ListWorkspaces(collaboration.ListOptions{Offset: 3})
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

func TestRegistry_ListWorkspaces_Empty(t *testing.T) {
	reg, _ := newTestRegistry(t)

	workspaces, err := reg.ListWorkspaces(collaboration.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, workspaces)
}

func TestRegistry_SearchWorkspaces(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	configs := []collaboration.WorkspaceConfig{
		{Name: "frontend-ui", OwnerProfileID: "owner-1"},
		{Name: "backend-api", OwnerProfileID: "owner-1"},
		{Name: "frontend-tests", OwnerProfileID: "owner-1"},
	}
	for _, cfg := range configs {
		_, err := mgr.Create(ctx, cfg)
		require.NoError(t, err)
	}

	results, err := reg.SearchWorkspaces("frontend")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results, err = reg.SearchWorkspaces("api")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "backend-api", results[0].Config.Name)
}

func TestRegistry_SearchWorkspaces_CaseInsensitive(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	cfg := collaboration.WorkspaceConfig{Name: "MyProject", OwnerProfileID: "owner-1"}
	_, err := mgr.Create(ctx, cfg)
	require.NoError(t, err)

	results, err := reg.SearchWorkspaces("myproject")
	require.NoError(t, err)
	assert.Len(t, results, 1)

	results, err = reg.SearchWorkspaces("MYPROJECT")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestRegistry_SearchWorkspaces_NoMatch(t *testing.T) {
	reg, mgr := newTestRegistry(t)
	ctx := context.Background()

	cfg := collaboration.WorkspaceConfig{Name: "alpha", OwnerProfileID: "owner-1"}
	_, err := mgr.Create(ctx, cfg)
	require.NoError(t, err)

	results, err := reg.SearchWorkspaces("zzz")
	require.NoError(t, err)
	assert.Empty(t, results)
}
