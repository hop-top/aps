package collaboration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"
)

func timePtr(t time.Time) *time.Time { return &t }

func newTestCollector(t *testing.T) (*collaboration.MetricsCollector, *collaboration.Manager, collaboration.Storage) {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	mgr := collaboration.NewManager(s)
	mc := collaboration.NewMetricsCollector(s)
	return mc, mgr, s
}

func createTestWorkspace(t *testing.T, mgr *collaboration.Manager) *collaboration.Workspace {
	t.Helper()
	ws, err := mgr.Create(context.Background(), collaboration.WorkspaceConfig{
		Name:           "test-workspace",
		OwnerProfileID: "owner-1",
	})
	require.NoError(t, err)
	return ws
}

func TestMetricsCollector_Collect_EmptyWorkspace(t *testing.T) {
	mc, mgr, _ := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 1, metrics.TotalAgents)
	assert.Equal(t, 1, metrics.OnlineAgents)
	assert.Equal(t, 0, metrics.TasksSubmitted)
	assert.Equal(t, 0, metrics.TasksCompleted)
	assert.Equal(t, 0, metrics.TasksFailed)
	assert.Equal(t, 0, metrics.TasksCancelled)
	assert.Equal(t, 0, metrics.TasksWorking)
	assert.Equal(t, time.Duration(0), metrics.AvgDuration)
	assert.Equal(t, time.Duration(0), metrics.P95Duration)
	assert.Equal(t, 0, metrics.ConflictsDetected)
	assert.Equal(t, 0, metrics.ConflictsResolved)
	assert.Equal(t, 0, metrics.ConflictsUnresolved)
	assert.Equal(t, 0, metrics.ContextVariables)
	assert.Equal(t, 0, metrics.ContextMutations)
}

func TestMetricsCollector_Collect_WithTasks(t *testing.T) {
	mc, mgr, s := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	now := time.Now()
	tasks := []collaboration.TaskInfo{
		{ID: "t1", WorkspaceID: ws.ID, Status: collaboration.TaskSubmitted, CreatedAt: now},
		{ID: "t2", WorkspaceID: ws.ID, Status: collaboration.TaskCompleted, CreatedAt: now.Add(-10 * time.Minute), CompletedAt: timePtr(now)},
		{ID: "t3", WorkspaceID: ws.ID, Status: collaboration.TaskFailed, CreatedAt: now.Add(-5 * time.Minute), CompletedAt: timePtr(now)},
		{ID: "t4", WorkspaceID: ws.ID, Status: collaboration.TaskCancelled, CreatedAt: now},
		{ID: "t5", WorkspaceID: ws.ID, Status: collaboration.TaskWorking, CreatedAt: now},
	}
	err := s.SaveTasks(ws.ID, tasks)
	require.NoError(t, err)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 5, metrics.TasksSubmitted)
	assert.Equal(t, 1, metrics.TasksCompleted)
	assert.Equal(t, 1, metrics.TasksFailed)
	assert.Equal(t, 1, metrics.TasksCancelled)
	assert.Equal(t, 1, metrics.TasksWorking)
}

func TestMetricsCollector_Collect_TaskDurations(t *testing.T) {
	mc, mgr, s := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	now := time.Now()
	tasks := []collaboration.TaskInfo{
		{ID: "t1", WorkspaceID: ws.ID, Status: collaboration.TaskCompleted,
			CreatedAt: now.Add(-5 * time.Minute), CompletedAt: timePtr(now)},
		{ID: "t2", WorkspaceID: ws.ID, Status: collaboration.TaskCompleted,
			CreatedAt: now.Add(-10 * time.Minute), CompletedAt: timePtr(now)},
		{ID: "t3", WorkspaceID: ws.ID, Status: collaboration.TaskCompleted,
			CreatedAt: now.Add(-15 * time.Minute), CompletedAt: timePtr(now)},
	}
	err := s.SaveTasks(ws.ID, tasks)
	require.NoError(t, err)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, metrics.TasksCompleted)
	assert.True(t, metrics.AvgDuration > 0, "AvgDuration should be positive")
	assert.True(t, metrics.P95Duration > 0, "P95Duration should be positive")
	assert.True(t, metrics.P95Duration >= metrics.AvgDuration, "P95Duration should be >= AvgDuration")
}

func TestMetricsCollector_Collect_WithConflicts(t *testing.T) {
	mc, mgr, s := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	now := time.Now()
	conflicts := []collaboration.Conflict{
		{ID: "c1", WorkspaceID: ws.ID, Type: collaboration.ConflictWrite, Resource: "key-1",
			DetectedAt: now, ResolvedAt: timePtr(now),
			Resolution: &collaboration.ConflictResolution{
				Strategy: collaboration.StrategyPriority, ResolvedBy: "system", Timestamp: now,
			}},
		{ID: "c2", WorkspaceID: ws.ID, Type: collaboration.ConflictWrite, Resource: "key-2",
			DetectedAt: now, ResolvedAt: timePtr(now),
			Resolution: &collaboration.ConflictResolution{
				Strategy: collaboration.StrategyKeepLast, ResolvedBy: "system", Timestamp: now,
			}},
		{ID: "c3", WorkspaceID: ws.ID, Type: collaboration.ConflictWrite, Resource: "key-3",
			DetectedAt: now},
	}
	err := s.SaveConflicts(ws.ID, conflicts)
	require.NoError(t, err)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, metrics.ConflictsDetected)
	assert.Equal(t, 2, metrics.ConflictsResolved)
	assert.Equal(t, 1, metrics.ConflictsUnresolved)
}

func TestMetricsCollector_Collect_WithContext(t *testing.T) {
	mc, mgr, s := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	now := time.Now()
	ctx := []collaboration.ContextVariable{
		{Key: "env", Value: "production", UpdatedBy: "agent-1", Version: 1, UpdatedAt: now},
		{Key: "region", Value: "us-east-1", UpdatedBy: "agent-1", Version: 1, UpdatedAt: now},
		{Key: "debug", Value: "false", UpdatedBy: "agent-2", Version: 3, UpdatedAt: now},
	}
	err := s.SaveContext(ws.ID, ctx)
	require.NoError(t, err)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, metrics.ContextVariables)
}

func TestMetricsCollector_Collect_WithAuditEvents(t *testing.T) {
	mc, mgr, s := newTestCollector(t)

	ws := createTestWorkspace(t, mgr)

	now := time.Now()
	events := []collaboration.AuditEvent{
		{ID: "e1", WorkspaceID: ws.ID, Event: "agent.join", Actor: "agent-1", Timestamp: now},
		{ID: "e2", WorkspaceID: ws.ID, Event: "task.create", Actor: "agent-1", Timestamp: now},
		{ID: "e3", WorkspaceID: ws.ID, Event: "context.set", Actor: "agent-2", Timestamp: now},
	}
	err := s.SaveAuditEvents(ws.ID, events)
	require.NoError(t, err)

	metrics, err := mc.Collect(ws.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, metrics.AuditEvents)
}

func TestMetricsCollector_Collect_NotFound(t *testing.T) {
	mc, _, _ := newTestCollector(t)

	_, err := mc.Collect("nonexistent-workspace-id")
	assert.Error(t, err)
}
