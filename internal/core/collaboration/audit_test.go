package collaboration_test

import (
	"context"
	"testing"
	"time"

	"oss-aps-cli/internal/core/collaboration"
	"oss-aps-cli/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuditLog(t *testing.T) (*collaboration.WorkspaceAuditLog, collaboration.Storage) {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	return collaboration.NewWorkspaceAuditLog(s), s
}

func TestWorkspaceAuditLog_Record(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	err := al.Record(ctx, collaboration.AuditEvent{
		ID:          "evt-1",
		WorkspaceID: "ws-1",
		Actor:       "agent-1",
		Event:       "task.create",
		Resource:    "task-123",
		Details:     "created new task",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "evt-1", events[0].ID)
	assert.Equal(t, "agent-1", events[0].Actor)
	assert.Equal(t, "task.create", events[0].Event)
	assert.Equal(t, "task-123", events[0].Resource)
}

func TestWorkspaceAuditLog_Record_SetsDefaults(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	// Record with empty ID and zero timestamp -- should be auto-populated.
	err := al.Record(ctx, collaboration.AuditEvent{
		WorkspaceID: "ws-1",
		Actor:       "agent-1",
		Event:       "agent.join",
	})
	require.NoError(t, err)

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{})
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.NotEmpty(t, events[0].ID, "ID should be auto-generated")
	assert.False(t, events[0].Timestamp.IsZero(), "Timestamp should be auto-set")
}

func TestWorkspaceAuditLog_Query_All(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	for i, evt := range []string{"agent.join", "task.create", "task.complete"} {
		err := al.Record(ctx, collaboration.AuditEvent{
			WorkspaceID: "ws-1",
			Actor:       "agent-1",
			Event:       evt,
			Timestamp:   time.Now().Add(time.Duration(i) * time.Second),
		})
		require.NoError(t, err)
	}

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{})
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestWorkspaceAuditLog_Query_FilterByActor(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	err := al.Record(ctx, collaboration.AuditEvent{
		WorkspaceID: "ws-1",
		Actor:       "agent-1",
		Event:       "task.create",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	err = al.Record(ctx, collaboration.AuditEvent{
		WorkspaceID: "ws-1",
		Actor:       "agent-2",
		Event:       "task.complete",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{
		Actor: "agent-1",
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "agent-1", events[0].Actor)
}

func TestWorkspaceAuditLog_Query_FilterByEvent(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	for _, evt := range []string{"task.create", "task.complete", "agent.join"} {
		err := al.Record(ctx, collaboration.AuditEvent{
			WorkspaceID: "ws-1",
			Actor:       "agent-1",
			Event:       evt,
			Timestamp:   time.Now(),
		})
		require.NoError(t, err)
	}

	// Use glob pattern "task.*" to match task events only.
	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{
		Event: "task.*",
	})
	require.NoError(t, err)
	assert.Len(t, events, 2)
	for _, e := range events {
		assert.Contains(t, e.Event, "task.")
	}
}

func TestWorkspaceAuditLog_Query_FilterBySince(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	// Record an old event.
	err := al.Record(ctx, collaboration.AuditEvent{
		WorkspaceID: "ws-1",
		Actor:       "agent-1",
		Event:       "old.event",
		Timestamp:   time.Now().Add(-2 * time.Hour),
	})
	require.NoError(t, err)

	// Record a recent event.
	err = al.Record(ctx, collaboration.AuditEvent{
		WorkspaceID: "ws-1",
		Actor:       "agent-1",
		Event:       "new.event",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	// Query with Since = "1h" should only return events from the last hour.
	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{
		Since: "1h",
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "new.event", events[0].Event)
}

func TestWorkspaceAuditLog_Query_Limit(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		err := al.Record(ctx, collaboration.AuditEvent{
			WorkspaceID: "ws-1",
			Actor:       "agent-1",
			Event:       "bulk.event",
			Timestamp:   time.Now(),
		})
		require.NoError(t, err)
	}

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestWorkspaceAuditLog_Query_DefaultLimit(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	// Record 5 events and query without limit; default is 100 so all should return.
	for i := 0; i < 5; i++ {
		err := al.Record(ctx, collaboration.AuditEvent{
			WorkspaceID: "ws-1",
			Actor:       "agent-1",
			Event:       "default.limit",
			Timestamp:   time.Now(),
		})
		require.NoError(t, err)
	}

	events, err := al.Query(ctx, "ws-1", collaboration.AuditQueryOptions{
		Limit: 0, // zero triggers the default of 100
	})
	require.NoError(t, err)
	assert.Len(t, events, 5)
}

func TestWorkspaceAuditLog_Query_Empty(t *testing.T) {
	al, _ := newTestAuditLog(t)
	ctx := context.Background()

	events, err := al.Query(ctx, "ws-empty", collaboration.AuditQueryOptions{})
	require.NoError(t, err)
	assert.Empty(t, events)
}
