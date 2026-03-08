package a2a

import (
	"context"
	"testing"

	a2a "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	eventqueue "github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
)

func TestNewExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	assert.NotNil(t, executor)
}

func TestExecutor_GetProfile(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	retrieved := executor.GetProfile()
	assert.Equal(t, profile, retrieved)
	assert.Equal(t, "test-profile", retrieved.ID)
}

func TestExecutor_Execute_NoMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, a2a.NewTaskID())
	require.NoError(t, err)
	defer q.Close()

	reqCtx := &a2asrv.RequestContext{
		TaskID:     a2a.NewTaskID(),
		Message:    nil,
		StoredTask: nil,
	}

	err = executor.Execute(ctx, reqCtx, q)
	assert.Error(t, err)
}

func TestExecutor_Execute_WithMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    message,
		StoredTask: nil,
	}

	err = executor.Execute(ctx, reqCtx, q)
	assert.NoError(t, err)
}

func TestExecutor_Execute_WithStoredTask(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})
	storedTask := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    message,
		StoredTask: storedTask,
	}

	err = executor.Execute(ctx, reqCtx, q)
	assert.NoError(t, err)
}

func TestExecutor_Execute_EmitsEvents(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(context.Background(), taskID)
	require.NoError(t, err)

	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "test message"})

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    message,
		StoredTask: nil,
	}

	// Execute the task (blocking operation)
	err = executor.Execute(context.Background(), reqCtx, q)
	assert.NoError(t, err)

	q.Close()
}

func TestExecutor_Cancel_NoMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    nil,
		StoredTask: nil,
	}

	err = executor.Cancel(ctx, reqCtx, q)
	assert.NoError(t, err)
}

func TestExecutor_Cancel_WithTask(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	storedTask := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateWorking},
	}

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    nil,
		StoredTask: storedTask,
	}

	err = executor.Cancel(ctx, reqCtx, q)
	assert.NoError(t, err)
}

func TestExecutor_Cancel_EmitsEvent(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(context.Background(), taskID)
	require.NoError(t, err)

	storedTask := &a2a.Task{
		ID:     taskID,
		Status: a2a.TaskStatus{State: a2a.TaskStateWorking},
	}

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    nil,
		StoredTask: storedTask,
	}

	// Cancel the task
	err = executor.Cancel(context.Background(), reqCtx, q)
	assert.NoError(t, err)

	q.Close()
}

func TestExecutor_Execute_MultipleTextParts(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID: "test-profile",
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	// Message with multiple parts
	message := a2a.NewMessage(a2a.MessageRoleUser,
		a2a.TextPart{Text: "part1"},
		a2a.TextPart{Text: "part2"},
	)

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    message,
		StoredTask: nil,
	}

	err = executor.Execute(ctx, reqCtx, q)
	assert.NoError(t, err)
}

func TestExecutor_Execute_ComplexMessage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &StorageConfig{
		BasePath: tmpDir,
	}

	storage, err := NewStorage(config)
	require.NoError(t, err)

	profile := &core.Profile{
		ID:           "test-profile",
		DisplayName:  "Test Agent",
		Capabilities: []string{"execute", "search"},
		A2A: &core.A2AConfig{},
	}

	executor := NewExecutor(profile, storage)
	ctx := context.Background()

	taskID := a2a.NewTaskID()
	queue := eventqueue.NewInMemoryManager()
	q, err := queue.GetOrCreate(ctx, taskID)
	require.NoError(t, err)
	defer q.Close()

	message := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "execute test command"})

	reqCtx := &a2asrv.RequestContext{
		TaskID:     taskID,
		Message:    message,
		StoredTask: nil,
	}

	err = executor.Execute(ctx, reqCtx, q)
	assert.NoError(t, err)
}
