package collaboration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"oss-aps-cli/internal/core/collaboration"
	"oss-aps-cli/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) collaboration.Storage {
	t.Helper()
	root := t.TempDir()
	s, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	return s
}

func TestMessageRouter_Send(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}

	result, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.ID, "task ID should be populated")
	assert.Equal(t, collaboration.TaskSubmitted, result.Status)
	assert.Equal(t, "agent-a", result.SenderID)
	assert.Equal(t, "agent-b", result.RecipientID)
	assert.Equal(t, "analyze", result.Action)
	assert.False(t, result.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, result.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

func TestMessageRouter_Send_MissingAction(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Input:       json.RawMessage(`{"key":"value"}`),
	}

	result, err := router.Send(ctx, "ws-1", task)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMessageRouter_Send_MissingRecipient(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID: "agent-a",
		Action:   "analyze",
		Input:    json.RawMessage(`{"key":"value"}`),
	}

	result, err := router.Send(ctx, "ws-1", task)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMessageRouter_Send_MissingSender(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}

	result, err := router.Send(ctx, "ws-1", task)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMessageRouter_Send_DefaultTimeout(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}

	result, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Default timeout should be 5 minutes
	assert.Equal(t, 5*time.Minute, result.Timeout, "default timeout should be 5 minutes")
}

func TestMessageRouter_Get(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}

	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	got, err := router.Get(ctx, "ws-1", sent.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, sent.ID, got.ID)
	assert.Equal(t, sent.SenderID, got.SenderID)
	assert.Equal(t, sent.RecipientID, got.RecipientID)
	assert.Equal(t, sent.Action, got.Action)
	assert.Equal(t, sent.Status, got.Status)
}

func TestMessageRouter_Get_NotFound(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	got, err := router.Get(ctx, "ws-1", "nonexistent-id")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestMessageRouter_List(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	for range 3 {
		task := collaboration.TaskInfo{
			SenderID:    "agent-a",
			RecipientID: "agent-b",
			Action:      "analyze",
			Input:       json.RawMessage(`{"key":"value"}`),
		}
		_, err := router.Send(ctx, "ws-1", task)
		require.NoError(t, err)
	}

	tasks, err := router.List(ctx, "ws-1", collaboration.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestMessageRouter_List_FilterByStatus(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	// Send two tasks
	task1 := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent1, err := router.Send(ctx, "ws-1", task1)
	require.NoError(t, err)

	task2 := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "review",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	_, err = router.Send(ctx, "ws-1", task2)
	require.NoError(t, err)

	// Transition task1 to working
	err = router.UpdateStatus(ctx, "ws-1", sent1.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)

	// Filter by working status
	opts := collaboration.ListOptions{
		Filters: map[string]string{"status": string(collaboration.TaskWorking)},
	}
	tasks, err := router.List(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, sent1.ID, tasks[0].ID)
	assert.Equal(t, collaboration.TaskWorking, tasks[0].Status)
}

func TestMessageRouter_List_FilterByAgent(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task1 := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	_, err := router.Send(ctx, "ws-1", task1)
	require.NoError(t, err)

	task2 := collaboration.TaskInfo{
		SenderID:    "agent-c",
		RecipientID: "agent-d",
		Action:      "review",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	_, err = router.Send(ctx, "ws-1", task2)
	require.NoError(t, err)

	// Filter by agent (sender)
	opts := collaboration.ListOptions{
		Filters: map[string]string{"agent": "agent-a"},
	}
	tasks, err := router.List(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "agent-a", tasks[0].SenderID)
}

func TestMessageRouter_List_Pagination(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	// Send 5 tasks
	for range 5 {
		task := collaboration.TaskInfo{
			SenderID:    "agent-a",
			RecipientID: "agent-b",
			Action:      "analyze",
			Input:       json.RawMessage(`{"key":"value"}`),
		}
		_, err := router.Send(ctx, "ws-1", task)
		require.NoError(t, err)
	}

	// Get first page (limit 2, offset 0)
	opts := collaboration.ListOptions{Limit: 2, Offset: 0}
	page1, err := router.List(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page (limit 2, offset 2)
	opts = collaboration.ListOptions{Limit: 2, Offset: 2}
	page2, err := router.List(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Get third page (limit 2, offset 4) - only 1 remaining
	opts = collaboration.ListOptions{Limit: 2, Offset: 4}
	page3, err := router.List(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, page3, 1)

	// Ensure pages do not overlap
	allIDs := make(map[string]bool)
	for _, task := range page1 {
		allIDs[task.ID] = true
	}
	for _, task := range page2 {
		assert.False(t, allIDs[task.ID], "page2 should not overlap with page1")
		allIDs[task.ID] = true
	}
	for _, task := range page3 {
		assert.False(t, allIDs[task.ID], "page3 should not overlap with previous pages")
	}
}

func TestMessageRouter_UpdateStatus_SubmittedToWorking(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskSubmitted, sent.Status)

	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)

	got, err := router.Get(ctx, "ws-1", sent.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskWorking, got.Status)
}

func TestMessageRouter_UpdateStatus_WorkingToCompleted(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	// Transition to working first
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)

	// Transition to completed with output
	output := json.RawMessage(`{"result":"done"}`)
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskCompleted, output)
	require.NoError(t, err)

	got, err := router.Get(ctx, "ws-1", sent.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskCompleted, got.Status)
	assert.False(t, got.CompletedAt.IsZero(), "CompletedAt should be set when task completes")
	assert.JSONEq(t, `{"result":"done"}`, string(got.Output))
}

func TestMessageRouter_UpdateStatus_WorkingToFailed(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)

	output := json.RawMessage(`{"error":"something went wrong"}`)
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskFailed, output)
	require.NoError(t, err)

	got, err := router.Get(ctx, "ws-1", sent.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskFailed, got.Status)
}

func TestMessageRouter_UpdateStatus_InvalidTransition(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	// Submitted -> Completed is not a valid transition (must go through Working)
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskCompleted, nil)
	assert.Error(t, err, "submitted -> completed should be an invalid transition")
}

func TestMessageRouter_UpdateStatus_FromTerminal(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	// Move to working then completed
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskCompleted, json.RawMessage(`{"result":"done"}`))
	require.NoError(t, err)

	// Completed is terminal; transitioning to working should fail
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	assert.Error(t, err, "completed -> working should fail because completed is a terminal state")
}

func TestMessageRouter_UpdateStatus_InvalidStatus(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskStatus("invalid-status"), nil)
	assert.Error(t, err, "invalid status string should cause an error")
}

func TestMessageRouter_Cancel(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskSubmitted, sent.Status)

	err = router.Cancel(ctx, "ws-1", sent.ID)
	require.NoError(t, err)

	got, err := router.Get(ctx, "ws-1", sent.ID)
	require.NoError(t, err)
	assert.Equal(t, collaboration.TaskCancelled, got.Status)
}

func TestMessageRouter_Cancel_AlreadyCompleted(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	task := collaboration.TaskInfo{
		SenderID:    "agent-a",
		RecipientID: "agent-b",
		Action:      "analyze",
		Input:       json.RawMessage(`{"key":"value"}`),
	}
	sent, err := router.Send(ctx, "ws-1", task)
	require.NoError(t, err)

	// Move to working then completed
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
	require.NoError(t, err)
	err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskCompleted, json.RawMessage(`{"result":"done"}`))
	require.NoError(t, err)

	// Cancelling a completed task should fail
	err = router.Cancel(ctx, "ws-1", sent.ID)
	assert.Error(t, err, "cancelling an already completed task should fail")
}

func TestMessageRouter_History(t *testing.T) {
	store := newTestStorage(t)
	router := collaboration.NewMessageRouter(store, nil)
	ctx := context.Background()

	// Send multiple tasks with different outcomes
	for i := range 3 {
		task := collaboration.TaskInfo{
			SenderID:    "agent-a",
			RecipientID: "agent-b",
			Action:      "analyze",
			Input:       json.RawMessage(`{"key":"value"}`),
		}
		sent, err := router.Send(ctx, "ws-1", task)
		require.NoError(t, err)

		// Move first task to completed
		if i == 0 {
			err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskWorking, nil)
			require.NoError(t, err)
			err = router.UpdateStatus(ctx, "ws-1", sent.ID, collaboration.TaskCompleted, json.RawMessage(`{"result":"done"}`))
			require.NoError(t, err)
		}
	}

	tasks, err := router.History(ctx, "ws-1", collaboration.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, tasks, 3, "history should return all tasks")

	// Verify we can filter history as well
	opts := collaboration.ListOptions{
		Filters: map[string]string{"status": string(collaboration.TaskCompleted)},
	}
	completed, err := router.History(ctx, "ws-1", opts)
	require.NoError(t, err)
	assert.Len(t, completed, 1, "history filtered by completed should return 1 task")
}
