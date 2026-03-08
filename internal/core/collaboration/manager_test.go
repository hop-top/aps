package collaboration_test

import (
	"context"
	"testing"

	"hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *collaboration.Manager {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	return collaboration.NewManager(s)
}

func testConfig() collaboration.WorkspaceConfig {
	return collaboration.WorkspaceConfig{
		Name:           "test-workspace",
		OwnerProfileID: "owner-1",
	}
}

func TestManager_Create(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)
	assert.NotEmpty(t, ws.ID)
	assert.Equal(t, "test-workspace", ws.Config.Name)
	assert.Equal(t, collaboration.StateActive, ws.State)
	assert.Len(t, ws.Agents, 1)
	assert.Equal(t, "owner-1", ws.Agents[0].ProfileID)
	assert.Equal(t, collaboration.RoleOwner, ws.Agents[0].Role)
}

func TestManager_Create_InvalidConfig(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	_, err := m.Create(ctx, collaboration.WorkspaceConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace name is required")
}

func TestManager_Get(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	loaded, err := m.Get(ctx, ws.ID)
	require.NoError(t, err)
	assert.Equal(t, ws.ID, loaded.ID)
	assert.Equal(t, ws.Config.Name, loaded.Config.Name)
}

func TestManager_Get_NotFound(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	_, err := m.Get(ctx, "nonexistent")
	require.Error(t, err)
}

func TestManager_List(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	cfg1 := testConfig()
	cfg1.Name = "alpha"
	_, err := m.Create(ctx, cfg1)
	require.NoError(t, err)

	cfg2 := testConfig()
	cfg2.Name = "beta"
	_, err = m.Create(ctx, cfg2)
	require.NoError(t, err)

	workspaces, err := m.List(ctx, collaboration.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

func TestManager_List_FilterByName(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	cfg1 := testConfig()
	cfg1.Name = "alpha"
	_, err := m.Create(ctx, cfg1)
	require.NoError(t, err)

	cfg2 := testConfig()
	cfg2.Name = "beta"
	_, err = m.Create(ctx, cfg2)
	require.NoError(t, err)

	workspaces, err := m.List(ctx, collaboration.ListOptions{
		Filters: map[string]string{"name": "alpha"},
	})
	require.NoError(t, err)
	assert.Len(t, workspaces, 1)
	assert.Equal(t, "alpha", workspaces[0].Config.Name)
}

func TestManager_List_FilterByStatus(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	err = m.Archive(ctx, ws.ID)
	require.NoError(t, err)

	cfg2 := testConfig()
	cfg2.Name = "still-active"
	_, err = m.Create(ctx, cfg2)
	require.NoError(t, err)

	workspaces, err := m.List(ctx, collaboration.ListOptions{
		Filters: map[string]string{"status": "active"},
	})
	require.NoError(t, err)
	assert.Len(t, workspaces, 1)
	assert.Equal(t, "still-active", workspaces[0].Config.Name)
}

func TestManager_List_Pagination(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cfg := testConfig()
		cfg.Name = "ws-" + string(rune('a'+i))
		_, err := m.Create(ctx, cfg)
		require.NoError(t, err)
	}

	workspaces, err := m.List(ctx, collaboration.ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

func TestManager_Archive(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	err = m.Archive(ctx, ws.ID)
	require.NoError(t, err)

	loaded, err := m.Get(ctx, ws.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.StateArchived, loaded.State)
}

func TestManager_Archive_NotFound(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	err := m.Archive(ctx, "nonexistent")
	require.Error(t, err)
}

func TestManager_Join(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	agent, err := m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)
	assert.Equal(t, "agent-2", agent.ProfileID)
	assert.Equal(t, collaboration.RoleContributor, agent.Role)

	members, err := m.Members(ctx, ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestManager_Join_AlreadyMember(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "owner-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in workspace")
}

func TestManager_Leave(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	err = m.Leave(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	members, err := m.Members(ctx, ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestManager_Leave_LastOwner(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	err = m.Leave(ctx, ws.ID, "owner-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last owner")
}

func TestManager_Remove(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	err = m.Remove(ctx, ws.ID, "agent-2", "owner-1")
	require.NoError(t, err)

	members, err := m.Members(ctx, ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestManager_Remove_NotOwner(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-3")
	require.NoError(t, err)

	err = m.Remove(ctx, ws.ID, "agent-3", "agent-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestManager_SetRole(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	err = m.SetRole(ctx, ws.ID, "agent-2", collaboration.RoleObserver)
	require.NoError(t, err)

	members, err := m.Members(ctx, ws.ID)
	require.NoError(t, err)
	for _, a := range members {
		if a.ProfileID == "agent-2" {
			assert.Equal(t, collaboration.RoleObserver, a.Role)
		}
	}
}

func TestManager_SetRole_InvalidRole(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	_, err = m.Join(ctx, ws.ID, "agent-2")
	require.NoError(t, err)

	err = m.SetRole(ctx, ws.ID, "agent-2", "invalid-role")
	require.Error(t, err)
}

func TestManager_Members(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	members, err := m.Members(ctx, ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "owner-1", members[0].ProfileID)
}

func TestManager_ActiveWorkspace(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	ws, err := m.Create(ctx, testConfig())
	require.NoError(t, err)

	err = m.SetActiveWorkspace(ctx, ws.ID)
	require.NoError(t, err)

	active, err := m.GetActiveWorkspace(ctx)
	require.NoError(t, err)
	assert.Equal(t, ws.ID, active)
}

func TestManager_SetActiveWorkspace_NotFound(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	err := m.SetActiveWorkspace(ctx, "nonexistent")
	require.Error(t, err)
}

func TestManager_GetActiveWorkspace_None(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()

	active, err := m.GetActiveWorkspace(ctx)
	require.NoError(t, err)
	assert.Empty(t, active)
}
