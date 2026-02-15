package storage_test

import (
	"testing"
	"time"

	collab "oss-aps-cli/internal/core/collaboration"
	"oss-aps-cli/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *storage.CollaborationStorage {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	return s
}

func TestCollaborationStorage_SaveAndLoadWorkspace(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "test-ws",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)

	err = s.SaveWorkspace(ws)
	require.NoError(t, err)

	loaded, err := s.LoadWorkspace(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, ws.ID, loaded.ID)
	assert.Equal(t, ws.Config.Name, loaded.Config.Name)
	assert.Equal(t, ws.Config.OwnerProfileID, loaded.Config.OwnerProfileID)
	assert.Equal(t, ws.State, loaded.State)
	assert.Len(t, loaded.Agents, 1)
	assert.Equal(t, "owner-1", loaded.Agents[0].ProfileID)
	assert.Equal(t, collab.RoleOwner, loaded.Agents[0].Role)
	assert.Equal(t, ws.Policy.Default, loaded.Policy.Default)
	assert.WithinDuration(t, ws.CreatedAt, loaded.CreatedAt, time.Second)
	assert.WithinDuration(t, ws.UpdatedAt, loaded.UpdatedAt, time.Second)
}

func TestCollaborationStorage_LoadWorkspace_NotFound(t *testing.T) {
	s := newTestStorage(t)

	_, err := s.LoadWorkspace("nonexistent")
	require.Error(t, err)

	var notFound *collab.WorkspaceNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestCollaborationStorage_ListWorkspaces(t *testing.T) {
	s := newTestStorage(t)

	ws1, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "ws-1",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws1))

	ws2, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "ws-2",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws2))

	ids, err := s.ListWorkspaces()
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, ws1.ID)
	assert.Contains(t, ids, ws2.ID)
}

func TestCollaborationStorage_ListWorkspaces_Empty(t *testing.T) {
	s := newTestStorage(t)

	ids, err := s.ListWorkspaces()
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestCollaborationStorage_DeleteWorkspace(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "to-delete",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	err = s.DeleteWorkspace(ws.ID)
	require.NoError(t, err)

	_, err = s.LoadWorkspace(ws.ID)
	require.Error(t, err)
}

func TestCollaborationStorage_DeleteWorkspace_NotFound(t *testing.T) {
	s := newTestStorage(t)

	err := s.DeleteWorkspace("nonexistent")
	require.Error(t, err)
}

func TestCollaborationStorage_SaveAndLoadTasks(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "task-ws",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	now := time.Now()
	tasks := []collab.TaskInfo{
		{
			ID:          "task-1",
			WorkspaceID: ws.ID,
			SenderID:    "agent-a",
			RecipientID: "agent-b",
			Action:      "review",
			Status:      collab.TaskSubmitted,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	err = s.SaveTasks(ws.ID, tasks)
	require.NoError(t, err)

	loaded, err := s.LoadTasks(ws.ID)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "task-1", loaded[0].ID)
	assert.Equal(t, collab.TaskSubmitted, loaded[0].Status)
}

func TestCollaborationStorage_LoadTasks_Empty(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "empty-tasks",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	tasks, err := s.LoadTasks(ws.ID)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestCollaborationStorage_SaveAndLoadConflicts(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "conflict-ws",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	conflicts := []collab.Conflict{
		{
			ID:          "conflict-1",
			WorkspaceID: ws.ID,
			Type:        collab.ConflictWrite,
			Resource:    "shared-var",
			AgentIDs:    []string{"agent-a", "agent-b"},
			DetectedAt:  time.Now(),
		},
	}

	err = s.SaveConflicts(ws.ID, conflicts)
	require.NoError(t, err)

	loaded, err := s.LoadConflicts(ws.ID)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "conflict-1", loaded[0].ID)
	assert.Equal(t, collab.ConflictWrite, loaded[0].Type)
}

func TestCollaborationStorage_LoadConflicts_Empty(t *testing.T) {
	s := newTestStorage(t)

	conflicts, err := s.LoadConflicts("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestCollaborationStorage_SaveAndLoadContext(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "context-ws",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	variables := []collab.ContextVariable{
		{
			Key:       "api_url",
			Value:     "https://api.example.com",
			Version:   1,
			UpdatedBy: "agent-a",
			UpdatedAt: time.Now(),
		},
	}

	err = s.SaveContext(ws.ID, variables)
	require.NoError(t, err)

	loaded, err := s.LoadContext(ws.ID)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "api_url", loaded[0].Key)
	assert.Equal(t, "https://api.example.com", loaded[0].Value)
}

func TestCollaborationStorage_LoadContext_Empty(t *testing.T) {
	s := newTestStorage(t)

	variables, err := s.LoadContext("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, variables)
}

func TestCollaborationStorage_SaveAndLoadAuditEvents(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "audit-ws",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	require.NoError(t, s.SaveWorkspace(ws))

	events := []collab.AuditEvent{
		{
			ID:          "event-1",
			WorkspaceID: ws.ID,
			Actor:       "owner-1",
			Event:       "workspace.create",
			Resource:    ws.ID,
			Timestamp:   time.Now(),
		},
	}

	err = s.SaveAuditEvents(ws.ID, events)
	require.NoError(t, err)

	loaded, err := s.LoadAuditEvents(ws.ID)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "event-1", loaded[0].ID)
	assert.Equal(t, "workspace.create", loaded[0].Event)
}

func TestCollaborationStorage_LoadAuditEvents_Empty(t *testing.T) {
	s := newTestStorage(t)

	events, err := s.LoadAuditEvents("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestCollaborationStorage_ActiveWorkspace(t *testing.T) {
	s := newTestStorage(t)

	err := s.SaveActiveWorkspace("ws-123")
	require.NoError(t, err)

	id, err := s.LoadActiveWorkspace()
	require.NoError(t, err)
	assert.Equal(t, "ws-123", id)
}

func TestCollaborationStorage_LoadActiveWorkspace_None(t *testing.T) {
	s := newTestStorage(t)

	id, err := s.LoadActiveWorkspace()
	require.NoError(t, err)
	assert.Empty(t, id)
}

func TestCollaborationStorage_ActiveWorkspace_Overwrite(t *testing.T) {
	s := newTestStorage(t)

	err := s.SaveActiveWorkspace("ws-1")
	require.NoError(t, err)

	err = s.SaveActiveWorkspace("ws-2")
	require.NoError(t, err)

	id, err := s.LoadActiveWorkspace()
	require.NoError(t, err)
	assert.Equal(t, "ws-2", id)
}

func TestCollaborationStorage_WorkspaceRoundTrip_MultipleAgents(t *testing.T) {
	s := newTestStorage(t)

	ws, err := collab.NewWorkspace(collab.WorkspaceConfig{
		Name:           "multi-agent",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)

	_, err = ws.AddAgent("contributor-1", collab.RoleContributor)
	require.NoError(t, err)

	_, err = ws.AddAgent("observer-1", collab.RoleObserver)
	require.NoError(t, err)

	require.NoError(t, s.SaveWorkspace(ws))

	loaded, err := s.LoadWorkspace(ws.ID)
	require.NoError(t, err)
	assert.Len(t, loaded.Agents, 3)

	roles := map[string]collab.AgentRole{}
	for _, a := range loaded.Agents {
		roles[a.ProfileID] = a.Role
	}
	assert.Equal(t, collab.RoleOwner, roles["owner-1"])
	assert.Equal(t, collab.RoleContributor, roles["contributor-1"])
	assert.Equal(t, collab.RoleObserver, roles["observer-1"])
}
