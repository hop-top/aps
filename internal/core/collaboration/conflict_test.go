package collaboration_test

import (
	"testing"
	"time"

	"hop.top/aps/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictDetector_DetectWriteConflicts(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "v1", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "v2", "agent-b", collaboration.RoleContributor)

	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectWriteConflicts(ws, 5*time.Second)

	require.NotEmpty(t, conflicts)
	assert.Equal(t, "deploy-key", conflicts[0].Resource)
}

func TestConflictDetector_DetectWriteConflicts_NoConflict(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "v1", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "v2", "agent-a", collaboration.RoleOwner)

	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectWriteConflicts(ws, 5*time.Second)

	assert.Empty(t, conflicts)
}

func TestConflictDetector_DetectWriteConflicts_OutsideWindow(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "v1", "agent-a", collaboration.RoleOwner)
	ws.Context.Set("deploy-key", "v2", "agent-b", collaboration.RoleContributor)

	detector := collaboration.NewConflictDetector()
	// Use zero window so everything is outside
	conflicts := detector.DetectWriteConflicts(ws, 0)

	assert.Empty(t, conflicts)
}

func TestConflictDetector_DetectWriteConflicts_NilWorkspace(t *testing.T) {
	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectWriteConflicts(nil, 5*time.Second)

	assert.Nil(t, conflicts)
}

func TestConflictDetector_DetectWriteConflicts_SingleMutation(t *testing.T) {
	config := collaboration.WorkspaceConfig{Name: "test", OwnerProfileID: "owner"}
	ws, err := collaboration.NewWorkspace(config)
	require.NoError(t, err)

	ws.Context.Set("deploy-key", "v1", "agent-a", collaboration.RoleOwner)

	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectWriteConflicts(ws, 5*time.Second)

	assert.Empty(t, conflicts)
}

func TestConflictDetector_DetectOrderingConflicts(t *testing.T) {
	tasks := []collaboration.TaskInfo{
		{ID: "task-a", Dependencies: []string{"task-b"}},
		{ID: "task-b", Dependencies: []string{"task-a"}},
	}

	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectOrderingConflicts(tasks)

	require.NotEmpty(t, conflicts)
}

func TestConflictDetector_DetectOrderingConflicts_NoCycle(t *testing.T) {
	tasks := []collaboration.TaskInfo{
		{ID: "task-a", Dependencies: []string{}},
		{ID: "task-b", Dependencies: []string{"task-a"}},
		{ID: "task-c", Dependencies: []string{"task-b"}},
	}

	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectOrderingConflicts(tasks)

	assert.Empty(t, conflicts)
}

func TestConflictDetector_DetectOrderingConflicts_Empty(t *testing.T) {
	detector := collaboration.NewConflictDetector()
	conflicts := detector.DetectOrderingConflicts([]collaboration.TaskInfo{})

	assert.Empty(t, conflicts)
}

func TestConflictDetector_DetectLockConflicts(t *testing.T) {
	locks := map[string]string{
		"deploy-key": "agent-a",
	}

	detector := collaboration.NewConflictDetector()
	conflict := detector.DetectLockConflicts(locks, "agent-b", "deploy-key")

	require.NotNil(t, conflict)
	assert.Equal(t, "deploy-key", conflict.Resource)
}

func TestConflictDetector_DetectLockConflicts_SameAgent(t *testing.T) {
	locks := map[string]string{
		"deploy-key": "agent-a",
	}

	detector := collaboration.NewConflictDetector()
	conflict := detector.DetectLockConflicts(locks, "agent-a", "deploy-key")

	assert.Nil(t, conflict)
}

func TestConflictDetector_DetectLockConflicts_NotLocked(t *testing.T) {
	locks := map[string]string{
		"other-key": "agent-a",
	}

	detector := collaboration.NewConflictDetector()
	conflict := detector.DetectLockConflicts(locks, "agent-b", "deploy-key")

	assert.Nil(t, conflict)
}

func TestConflictDetector_DetectLockConflicts_NilLocks(t *testing.T) {
	detector := collaboration.NewConflictDetector()
	conflict := detector.DetectLockConflicts(nil, "agent-b", "deploy-key")

	assert.Nil(t, conflict)
}
