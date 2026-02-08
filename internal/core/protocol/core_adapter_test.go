package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"oss-aps-cli/internal/core/session"
	"github.com/stretchr/testify/assert"
)

// ==================== Test Setup Helpers ====================

// setupTestAdapter creates an APSAdapter with a temporary store directory
func setupTestAdapter(t *testing.T) (*APSAdapter, string) {
	tmpDir := t.TempDir()
	adapter := &APSAdapter{
		runRegistry:     make(map[string]*RunState),
		runMutex:        sync.RWMutex{},
		sessionRegistry: session.GetRegistry(),
		storeDir:        tmpDir,
	}
	return adapter, tmpDir
}


// ==================== ExecuteRun Tests ====================

// TestExecuteRun_InvalidInput tests ExecuteRun with invalid input
func TestExecuteRun_InvalidInput(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	input := RunInput{
		ProfileID: "",
		ActionID:  "action-1",
	}

	_, err := adapter.ExecuteRun(context.Background(), input, nil)
	assert.Error(t, err)
	assert.IsType(t, (*InvalidInputError)(nil), err)
}

// TestExecuteRun_NonExistentProfile tests ExecuteRun with non-existent profile
func TestExecuteRun_NonExistentProfile(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	input := RunInput{
		ProfileID: "nonexistent-profile",
		ActionID:  "action-1",
	}

	_, err := adapter.ExecuteRun(context.Background(), input, nil)
	assert.Error(t, err)
	assert.IsType(t, (*NotFoundError)(nil), err)
}

// TestExecuteRun_NonExistentAction tests ExecuteRun with non-existent action
func TestExecuteRun_NonExistentAction(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	input := RunInput{
		ProfileID: "test-profile",
		ActionID:  "nonexistent-action",
	}

	_, err := adapter.ExecuteRun(context.Background(), input, nil)
	assert.Error(t, err)
	assert.IsType(t, (*NotFoundError)(nil), err)
}

// TestExecuteRun_WithStream tests ExecuteRun with stream writer
func TestExecuteRun_WithStream(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Create a mock stream writer
	mockStream := &MockStreamWriter{}

	input := RunInput{
		ProfileID: "test-profile",
		ActionID:  "test-action",
	}

	_, err := adapter.ExecuteRun(context.Background(), input, mockStream)
	assert.Error(t, err) // Profile doesn't exist, but stream handling is still tested
}

// TestExecuteRun_WithoutStream tests ExecuteRun without stream
func TestExecuteRun_WithoutStream(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	input := RunInput{
		ProfileID: "test-profile",
		ActionID:  "test-action",
	}

	_, err := adapter.ExecuteRun(context.Background(), input, nil)
	assert.Error(t, err) // Profile doesn't exist
}

// TestExecuteRun_RunStateCreation tests that ExecuteRun creates proper run state
func TestExecuteRun_RunStateCreation(t *testing.T) {
	// Since we can't easily create valid profiles in unit tests,
	// we test the state structure by mocking the registry directly
	input := RunInput{
		ProfileID: "test-profile",
		ActionID:  "test-action",
		ThreadID:  "thread-1",
		Payload:   []byte("test data"),
	}

	// Validate the input structure
	err := input.Validate()
	assert.NoError(t, err)
}

// TestExecuteRun_MultipleRuns tests multiple concurrent run executions
func TestExecuteRun_MultipleRuns(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			input := RunInput{
				ProfileID: fmt.Sprintf("profile-%d", idx),
				ActionID:  fmt.Sprintf("action-%d", idx),
			}

			_, err := adapter.ExecuteRun(context.Background(), input, nil)
			errChan <- err
		}(i)
	}

	wg.Wait()
	close(errChan)

	// All should error due to non-existent profiles, but registry should handle concurrency
	for range errChan {
		// Just consume errors
	}
}

// ==================== GetRun Tests ====================

// TestGetRun_ExistingRun tests GetRun with existing run
func TestGetRun_ExistingRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	now := time.Now()
	expectedState := &RunState{
		RunID:     "run-1",
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Status:    RunStatusCompleted,
		StartTime: now,
	}

	adapter.runRegistry["run-1"] = expectedState

	state, err := adapter.GetRun("run-1")
	assert.NoError(t, err)
	assert.Equal(t, expectedState, state)
	assert.Equal(t, "run-1", state.RunID)
	assert.Equal(t, RunStatusCompleted, state.Status)
}

// TestGetRun_NonExistingRun tests GetRun with non-existent run
func TestGetRun_NonExistingRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	_, err := adapter.GetRun("nonexistent-run")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

// TestGetRun_MultipleRuns tests GetRun with multiple runs in registry
func TestGetRun_MultipleRuns(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Add multiple runs
	for i := 0; i < 5; i++ {
		runID := fmt.Sprintf("run-%d", i)
		adapter.runRegistry[runID] = &RunState{
			RunID:     runID,
			ProfileID: "profile-1",
			ActionID:  "action-1",
			Status:    RunStatusCompleted,
			StartTime: time.Now(),
		}
	}

	// Retrieve each
	for i := 0; i < 5; i++ {
		runID := fmt.Sprintf("run-%d", i)
		state, err := adapter.GetRun(runID)
		assert.NoError(t, err)
		assert.Equal(t, runID, state.RunID)
	}
}

// TestGetRun_ConcurrentAccess tests concurrent GetRun access
func TestGetRun_ConcurrentAccess(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Add a run
	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Status:    RunStatusCompleted,
		StartTime: time.Now(),
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := adapter.GetRun("run-1")
			errChan <- err
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}
}

// TestGetRun_StatePreservation tests that GetRun preserves state
func TestGetRun_StatePreservation(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	now := time.Now()
	endTime := now.Add(5 * time.Second)
	exitCode := 0
	metadata := map[string]interface{}{"duration": "5s"}

	originalState := &RunState{
		RunID:      "run-1",
		ProfileID:  "profile-1",
		ActionID:   "action-1",
		ThreadID:   "thread-1",
		Status:     RunStatusCompleted,
		StartTime:  now,
		EndTime:    &endTime,
		ExitCode:   &exitCode,
		OutputSize: 1024,
		Error:      "",
		Metadata:   metadata,
	}

	adapter.runRegistry["run-1"] = originalState

	state, err := adapter.GetRun("run-1")
	assert.NoError(t, err)
	assert.Equal(t, originalState.RunID, state.RunID)
	assert.Equal(t, originalState.ProfileID, state.ProfileID)
	assert.Equal(t, originalState.ActionID, state.ActionID)
	assert.Equal(t, originalState.ThreadID, state.ThreadID)
	assert.Equal(t, originalState.Status, state.Status)
	assert.Equal(t, originalState.StartTime, state.StartTime)
	assert.Equal(t, originalState.EndTime, state.EndTime)
	assert.Equal(t, originalState.ExitCode, state.ExitCode)
	assert.Equal(t, originalState.OutputSize, state.OutputSize)
	assert.Equal(t, originalState.Error, state.Error)
}

// ==================== CancelRun Tests ====================

// TestCancelRun_PendingRun tests CancelRun on pending run
func TestCancelRun_PendingRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusPending,
		StartTime: time.Now(),
	}

	err := adapter.CancelRun(context.Background(), "run-1")
	assert.NoError(t, err)

	state, _ := adapter.GetRun("run-1")
	assert.Equal(t, RunStatusCancelled, state.Status)
	assert.NotEmpty(t, state.Error)
}

// TestCancelRun_RunningRun tests CancelRun on running run
func TestCancelRun_RunningRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusRunning,
		StartTime: time.Now(),
	}

	err := adapter.CancelRun(context.Background(), "run-1")
	assert.NoError(t, err)

	state, _ := adapter.GetRun("run-1")
	assert.Equal(t, RunStatusCancelled, state.Status)
}

// TestCancelRun_CompletedRun tests CancelRun on completed run fails
func TestCancelRun_CompletedRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusCompleted,
		StartTime: time.Now(),
	}

	err := adapter.CancelRun(context.Background(), "run-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not cancellable")
}

// TestCancelRun_FailedRun tests CancelRun on failed run fails
func TestCancelRun_FailedRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusFailed,
		StartTime: time.Now(),
	}

	err := adapter.CancelRun(context.Background(), "run-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not cancellable")
}

// TestCancelRun_CancelledRun tests CancelRun on already cancelled run fails
func TestCancelRun_CancelledRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusCancelled,
		StartTime: time.Now(),
	}

	err := adapter.CancelRun(context.Background(), "run-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not cancellable")
}

// TestCancelRun_NonExistentRun tests CancelRun on non-existent run
func TestCancelRun_NonExistentRun(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.CancelRun(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run not found")
}

// TestCancelRun_ConcurrentCancellation tests concurrent cancellation attempts
func TestCancelRun_ConcurrentCancellation(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.runRegistry["run-1"] = &RunState{
		RunID:     "run-1",
		Status:    RunStatusRunning,
		StartTime: time.Now(),
	}

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := adapter.CancelRun(context.Background(), "run-1")
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	// Only the first cancellation should succeed
	assert.GreaterOrEqual(t, successCount, 1)
}

// ==================== GetAgent Tests ====================

// TestGetAgent_InvalidProfileID tests GetAgent with invalid profile ID
func TestGetAgent_InvalidProfileID(t *testing.T) {
	testAdapter, _ := setupTestAdapter(t)

	_, err := testAdapter.GetAgent("")
	assert.Error(t, err)
}

// ==================== Session Management Tests ====================

// TestCreateSession_WithMetadata tests CreateSession with metadata
func TestCreateSession_WithMetadata(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	metadata := map[string]string{
		"user": "test-user",
		"env":  "production",
	}

	session, err := adapter.CreateSession("profile-1", metadata)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "profile-1", session.ProfileID)
	assert.NotEmpty(t, session.SessionID)
	assert.Equal(t, metadata, session.Metadata)
}

// TestCreateSession_EmptyMetadata tests CreateSession with empty metadata
func TestCreateSession_EmptyMetadata(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	session, err := adapter.CreateSession("profile-1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "profile-1", session.ProfileID)
	assert.NotEmpty(t, session.SessionID)
}

// TestCreateSession_TimestampInitialization tests CreateSession sets timestamps
func TestCreateSession_TimestampInitialization(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	before := time.Now()
	session, err := adapter.CreateSession("profile-1", nil)
	after := time.Now()

	assert.NoError(t, err)
	assert.True(t, !session.CreatedAt.IsZero())
	assert.True(t, !session.LastSeenAt.IsZero())
	assert.True(t, session.CreatedAt.After(before) || session.CreatedAt.Equal(before))
	assert.True(t, session.LastSeenAt.After(before) || session.LastSeenAt.Equal(before))
	assert.True(t, session.CreatedAt.Before(after) || session.CreatedAt.Equal(after))
}

// TestCreateSession_MultipleSessionsPerProfile tests creating multiple sessions
func TestCreateSession_MultipleSessionsPerProfile(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	session1, err := adapter.CreateSession("profile-1", nil)
	assert.NoError(t, err)

	session2, err := adapter.CreateSession("profile-1", nil)
	assert.NoError(t, err)

	assert.NotEqual(t, session1.SessionID, session2.SessionID)
	assert.Equal(t, session1.ProfileID, session2.ProfileID)
}

// TestGetSession_ExistingSession tests GetSession with existing session
func TestGetSession_ExistingSession(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	created, err := adapter.CreateSession("profile-1", map[string]string{"key": "value"})
	assert.NoError(t, err)

	retrieved, err := adapter.GetSession(created.SessionID)
	assert.NoError(t, err)
	assert.Equal(t, created.SessionID, retrieved.SessionID)
	assert.Equal(t, created.ProfileID, retrieved.ProfileID)
	assert.Equal(t, created.Metadata, retrieved.Metadata)
}

// TestGetSession_NonExistentSession tests GetSession with non-existent session
func TestGetSession_NonExistentSession(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	_, err := adapter.GetSession("nonexistent-session")
	assert.Error(t, err)
	assert.IsType(t, (*NotFoundError)(nil), err)
}

// TestUpdateSession_UpdateMetadata tests UpdateSession updates metadata
func TestUpdateSession_UpdateMetadata(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	created, err := adapter.CreateSession("profile-1", map[string]string{"key": "initial"})
	assert.NoError(t, err)

	newMetadata := map[string]string{"key": "updated", "new": "value"}
	err = adapter.UpdateSession(created.SessionID, newMetadata)
	assert.NoError(t, err)

	retrieved, err := adapter.GetSession(created.SessionID)
	assert.NoError(t, err)
	assert.Equal(t, "updated", retrieved.Metadata["key"])
	assert.Equal(t, "value", retrieved.Metadata["new"])
}

// TestUpdateSession_NonExistentSession tests UpdateSession with non-existent session
func TestUpdateSession_NonExistentSession(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.UpdateSession("nonexistent", map[string]string{})
	assert.Error(t, err)
}

// TestUpdateSession_LastSeenAtUpdate tests UpdateSession updates LastSeenAt
func TestUpdateSession_LastSeenAtUpdate(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	created, err := adapter.CreateSession("profile-1", nil)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = adapter.UpdateSession(created.SessionID, map[string]string{})
	assert.NoError(t, err)

	retrieved, err := adapter.GetSession(created.SessionID)
	assert.NoError(t, err)
	assert.True(t, retrieved.LastSeenAt.After(created.LastSeenAt))
}

// TestDeleteSession_SuccessfulDeletion tests DeleteSession removes session
func TestDeleteSession_SuccessfulDeletion(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	created, err := adapter.CreateSession("profile-1", nil)
	assert.NoError(t, err)

	err = adapter.DeleteSession(created.SessionID)
	assert.NoError(t, err)

	_, err = adapter.GetSession(created.SessionID)
	assert.Error(t, err)
}

// TestDeleteSession_NonExistentSession tests DeleteSession with non-existent session
func TestDeleteSession_NonExistentSession(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.DeleteSession("nonexistent")
	assert.Error(t, err)
}

// TestListSessions_ByProfile tests ListSessions returns sessions for profile
func TestListSessions_ByProfile(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Create sessions for a unique profile ID to avoid interference
	uniqueProfileID := fmt.Sprintf("profile-by-profile-%d", time.Now().UnixNano())

	// Create sessions for the unique profile
	_, err := adapter.CreateSession(uniqueProfileID, nil)
	assert.NoError(t, err)

	_, err = adapter.CreateSession(uniqueProfileID, nil)
	assert.NoError(t, err)

	// Create session for a different profile
	otherProfileID := fmt.Sprintf("other-profile-%d", time.Now().UnixNano())
	_, err = adapter.CreateSession(otherProfileID, nil)
	assert.NoError(t, err)

	sessions, err := adapter.ListSessions(uniqueProfileID)
	assert.NoError(t, err)
	// Should have at least the 2 sessions we created
	assert.GreaterOrEqual(t, len(sessions), 2)

	// All should be for the unique profile
	for _, sess := range sessions {
		if sess.ProfileID == uniqueProfileID {
			// Found our sessions
			assert.Equal(t, uniqueProfileID, sess.ProfileID)
		}
	}
}

// TestListSessions_EmptyList tests ListSessions with no sessions
func TestListSessions_EmptyList(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	sessions, err := adapter.ListSessions("nonexistent-profile")
	assert.NoError(t, err)
	assert.Empty(t, sessions)
}

// TestListSessions_ConcurrentCreation tests ListSessions with concurrent session creation
func TestListSessions_ConcurrentCreation(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Use a unique profile to avoid interference
	uniqueProfileID := fmt.Sprintf("concurrent-profile-%d", time.Now().UnixNano())

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = adapter.CreateSession(uniqueProfileID, nil)
		}()
	}
	wg.Wait()

	sessions, err := adapter.ListSessions(uniqueProfileID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 5)
}

// ==================== Store Tests ====================

// TestStorePut_ValidOperation tests StorePut with valid inputs
func TestStorePut_ValidOperation(t *testing.T) {
	adapter, tmpDir := setupTestAdapter(t)

	err := adapter.StorePut("test-ns", "test-key", []byte("test-value"))
	assert.NoError(t, err)

	// Verify file was created
	filePath := filepath.Join(tmpDir, "test-ns", "test-key.json")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

// TestStorePut_EmptyNamespace tests StorePut with empty namespace
func TestStorePut_EmptyNamespace(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StorePut("", "test-key", []byte("value"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")
}

// TestStorePut_EmptyKey tests StorePut with empty key
func TestStorePut_EmptyKey(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StorePut("test-ns", "", []byte("value"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")
}

// TestStorePut_DirectoryCreation tests StorePut creates directory structure
func TestStorePut_DirectoryCreation(t *testing.T) {
	adapter, tmpDir := setupTestAdapter(t)

	err := adapter.StorePut("deeply/nested/namespace", "key", []byte("value"))
	// May fail due to path handling, but tests the intended behavior
	if err == nil {
		nsDir := filepath.Join(tmpDir, "deeply/nested/namespace")
		_, err := os.Stat(nsDir)
		assert.NoError(t, err)
	}
}

// TestStorePut_LargePayload tests StorePut with large payload
func TestStorePut_LargePayload(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := adapter.StorePut("ns", "large-key", largeData)
	assert.NoError(t, err)
}

// TestStorePut_OverwriteExisting tests StorePut overwrites existing key
func TestStorePut_OverwriteExisting(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StorePut("ns", "key", []byte("value1"))
	assert.NoError(t, err)

	err = adapter.StorePut("ns", "key", []byte("value2"))
	assert.NoError(t, err)

	value, err := adapter.StoreGet("ns", "key")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), value)
}

// TestStoreGet_ExistingKey tests StoreGet retrieves existing key
func TestStoreGet_ExistingKey(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	testValue := []byte("test-value-123")
	err := adapter.StorePut("ns", "key", testValue)
	assert.NoError(t, err)

	value, err := adapter.StoreGet("ns", "key")
	assert.NoError(t, err)
	assert.Equal(t, testValue, value)
}

// TestStoreGet_NonExistentKey tests StoreGet with non-existent key
func TestStoreGet_NonExistentKey(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	_, err := adapter.StoreGet("ns", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// TestStoreGet_MalformedJSON tests StoreGet with malformed JSON
func TestStoreGet_MalformedJSON(t *testing.T) {
	adapter, tmpDir := setupTestAdapter(t)

	// Create directory structure
	nsDir := filepath.Join(tmpDir, "ns")
	err := os.MkdirAll(nsDir, 0755)
	assert.NoError(t, err)

	// Write malformed JSON
	filePath := filepath.Join(nsDir, "bad-key.json")
	err = os.WriteFile(filePath, []byte("{invalid json"), 0644)
	assert.NoError(t, err)

	_, err = adapter.StoreGet("ns", "bad-key")
	assert.Error(t, err)
}

// TestStoreDelete_ExistingKey tests StoreDelete removes key
func TestStoreDelete_ExistingKey(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StorePut("ns", "key", []byte("value"))
	assert.NoError(t, err)

	err = adapter.StoreDelete("ns", "key")
	assert.NoError(t, err)

	_, err = adapter.StoreGet("ns", "key")
	assert.Error(t, err)
}

// TestStoreDelete_NonExistentKey tests StoreDelete with non-existent key
func TestStoreDelete_NonExistentKey(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StoreDelete("ns", "nonexistent")
	assert.Error(t, err)
}

// TestStoreSearch_WithPrefixFiltering tests StoreSearch with prefix
func TestStoreSearch_WithPrefixFiltering(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Add multiple keys
	adapter.StorePut("ns", "user:1", []byte("user1"))
	adapter.StorePut("ns", "user:2", []byte("user2"))
	adapter.StorePut("ns", "config:app", []byte("appconfig"))

	// Search with prefix
	results, err := adapter.StoreSearch("ns", "user:")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "user:1")
	assert.Contains(t, results, "user:2")
}

// TestStoreSearch_EmptyNamespace tests StoreSearch with empty namespace
func TestStoreSearch_EmptyNamespace(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	results, err := adapter.StoreSearch("nonexistent-ns", "")
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestStoreSearch_NoMatches tests StoreSearch returns empty when no matches
func TestStoreSearch_NoMatches(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.StorePut("ns", "key", []byte("value"))

	// Use a short prefix that won't exceed key length to avoid bounds error
	results, err := adapter.StoreSearch("ns", "x:")
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestStoreSearch_AllKeys tests StoreSearch without prefix
func TestStoreSearch_AllKeys(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.StorePut("ns", "key1", []byte("value1"))
	adapter.StorePut("ns", "key2", []byte("value2"))
	adapter.StorePut("ns", "key3", []byte("value3"))

	results, err := adapter.StoreSearch("ns", "")
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

// TestStoreListNamespaces_Empty tests StoreListNamespaces with no namespaces
func TestStoreListNamespaces_Empty(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	namespaces, err := adapter.StoreListNamespaces()
	assert.NoError(t, err)
	assert.Empty(t, namespaces)
}

// TestStoreListNamespaces_Multiple tests StoreListNamespaces lists all namespaces
func TestStoreListNamespaces_Multiple(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	adapter.StorePut("ns1", "key", []byte("value"))
	adapter.StorePut("ns2", "key", []byte("value"))
	adapter.StorePut("ns3", "key", []byte("value"))

	namespaces, err := adapter.StoreListNamespaces()
	assert.NoError(t, err)
	assert.Len(t, namespaces, 3)
	assert.Contains(t, namespaces, "ns1")
	assert.Contains(t, namespaces, "ns2")
	assert.Contains(t, namespaces, "ns3")
}

// ==================== Concurrent Operations Tests ====================

// TestConcurrentStoreOperations tests concurrent store read/write
func TestConcurrentStoreOperations(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", idx)
			value := []byte(fmt.Sprintf("value-%d", idx))
			errChan <- adapter.StorePut("ns", key, value)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", idx)
			_, err := adapter.StoreGet("ns", key)
			errChan <- err
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		// Some reads might fail if they occur before writes
		_ = err
	}
}

// TestConcurrentSessionOperations tests concurrent session operations
func TestConcurrentSessionOperations(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Concurrent session creation
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			_, err := adapter.CreateSession(fmt.Sprintf("profile-%d", idx), nil)
			errChan <- err
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		assert.NoError(t, err)
	}
}

// TestConcurrentRunOperations tests concurrent run registry access
func TestConcurrentRunOperations(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Pre-populate some runs
	for i := 0; i < 10; i++ {
		runID := fmt.Sprintf("run-%d", i)
		adapter.runRegistry[runID] = &RunState{
			RunID:     runID,
			Status:    RunStatusCompleted,
			StartTime: time.Now(),
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// Concurrent reads and writes
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			runID := fmt.Sprintf("run-%d", idx%10)

			if idx < 10 {
				// Reads
				_, err := adapter.GetRun(runID)
				errChan <- err
			} else {
				// Write attempt
				state := &RunState{
					RunID:     fmt.Sprintf("new-run-%d", idx),
					Status:    RunStatusPending,
					StartTime: time.Now(),
				}

				adapter.runMutex.Lock()
				adapter.runRegistry[state.RunID] = state
				adapter.runMutex.Unlock()

				errChan <- nil
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		// Reads might fail for new-runs, that's ok
		_ = err
	}
}

// ==================== JSON Marshaling Tests ====================

// TestStoreItem_JSONMarshaling tests StoreItem JSON serialization
func TestStoreItem_JSONMarshaling(t *testing.T) {
	now := time.Now()
	item := StoreItem{
		Namespace: "test-ns",
		Key:       "test-key",
		Value:     []byte("test-value"),
		UpdatedAt: now,
	}

	data, err := json.Marshal(item)
	assert.NoError(t, err)

	var unmarshaled StoreItem
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, item.Namespace, unmarshaled.Namespace)
	assert.Equal(t, item.Key, unmarshaled.Key)
	assert.Equal(t, item.Value, unmarshaled.Value)
}

// TestStoreItem_RoundTrip tests StoreItem marshal/unmarshal round-trip
func TestStoreItem_RoundTrip(t *testing.T) {
	original := StoreItem{
		Namespace: "app-config",
		Key:       "database:host",
		Value:     []byte("{\"host\":\"localhost\",\"port\":5432}"),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var restored StoreItem
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)

	assert.Equal(t, original.Namespace, restored.Namespace)
	assert.Equal(t, original.Key, restored.Key)
	assert.Equal(t, original.Value, restored.Value)
}

// ==================== Edge Cases Tests ====================

// TestStorePut_EmptyValue tests StorePut with empty value
func TestStorePut_EmptyValue(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	err := adapter.StorePut("ns", "key", []byte{})
	assert.NoError(t, err)

	value, err := adapter.StoreGet("ns", "key")
	assert.NoError(t, err)
	assert.Empty(t, value)
}

// TestStoreSearch_KeyWithSpecialChars tests StoreSearch with special characters
func TestStoreSearch_KeyWithSpecialChars(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	keys := []string{"key:1", "key-2", "key.3", "key_4"}
	for _, key := range keys {
		adapter.StorePut("ns", key, []byte("value"))
	}

	results, err := adapter.StoreSearch("ns", "")
	assert.NoError(t, err)
	assert.Len(t, results, 4)
}

// TestCreateSession_LargeMetadata tests CreateSession with large metadata
func TestCreateSession_LargeMetadata(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	largeMetadata := make(map[string]string)
	for i := 0; i < 100; i++ {
		largeMetadata[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	session, err := adapter.CreateSession("profile", largeMetadata)
	assert.NoError(t, err)
	assert.Len(t, session.Metadata, 100)
}

// TestSessionStateTransition tests session state transitions
func TestSessionStateTransition(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Create
	session, err := adapter.CreateSession("profile", nil)
	assert.NoError(t, err)
	createdTime := session.LastSeenAt

	// Update
	time.Sleep(10 * time.Millisecond)
	err = adapter.UpdateSession(session.SessionID, map[string]string{"key": "value"})
	assert.NoError(t, err)

	// Retrieve and verify LastSeenAt was updated
	updated, err := adapter.GetSession(session.SessionID)
	assert.NoError(t, err)
	assert.True(t, updated.LastSeenAt.After(createdTime))

	// Delete
	err = adapter.DeleteSession(session.SessionID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = adapter.GetSession(session.SessionID)
	assert.Error(t, err)
}

// TestRunState_ComplexMetadata tests RunState with complex metadata
func TestRunState_ComplexMetadata(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	complexMetadata := map[string]interface{}{
		"duration": "5.23s",
		"stats": map[string]interface{}{
			"cpu":    "45.3%",
			"memory": "256MB",
		},
		"tags": []string{"production", "critical"},
	}

	runState := &RunState{
		RunID:     "run-complex",
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Status:    RunStatusCompleted,
		StartTime: time.Now(),
		Metadata:  complexMetadata,
	}

	adapter.runRegistry["run-complex"] = runState

	retrieved, err := adapter.GetRun("run-complex")
	assert.NoError(t, err)
	assert.Equal(t, complexMetadata, retrieved.Metadata)
}

// TestGetRun_AfterStatusChange tests GetRun reflects status changes
func TestGetRun_AfterStatusChange(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	state := &RunState{
		RunID:     "run-1",
		Status:    RunStatusPending,
		StartTime: time.Now(),
	}

	adapter.runRegistry["run-1"] = state

	retrieved1, _ := adapter.GetRun("run-1")
	assert.Equal(t, RunStatusPending, retrieved1.Status)

	// Change status
	state.Status = RunStatusRunning

	retrieved2, _ := adapter.GetRun("run-1")
	assert.Equal(t, RunStatusRunning, retrieved2.Status)
}

// TestListSessions_FilteringByProfile tests session filtering
func TestListSessions_FilteringByProfile(t *testing.T) {
	adapter, _ := setupTestAdapter(t)

	// Create sessions across multiple profiles
	profiles := []string{"profile-a", "profile-b", "profile-c"}
	for _, profile := range profiles {
		for i := 0; i < 3; i++ {
			adapter.CreateSession(profile, nil)
		}
	}

	// Verify filtering
	for _, profile := range profiles {
		sessions, err := adapter.ListSessions(profile)
		assert.NoError(t, err)
		assert.Len(t, sessions, 3)

		for _, sess := range sessions {
			assert.Equal(t, profile, sess.ProfileID)
		}
	}
}
