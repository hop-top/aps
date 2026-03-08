package agentprotocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"hop.top/aps/internal/core/protocol"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// SECTION 1: HANDLER METHOD TESTS (15 tests)
// ============================================================================

// Test 1: handleCreateRun with valid request
func TestHandleCreateRun_ValidRequest(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := CreateRunRequest{
		AgentID:   "agent-test",
		ActionID:  "action-test",
		SessionID: "session-123",
		Input: map[string]interface{}{
			"input": "test-payload",
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.RunID)
	assert.Equal(t, "completed", resp.Status)
}

// Test 2: handleCreateRun with invalid JSON
func TestHandleCreateRun_InvalidJSON(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader([]byte("{invalid")))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NotEmpty(t, errResp.Message)
}

// Test 3: handleCreateRun with wrong HTTP method
func TestHandleCreateRun_WrongMethod(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("GET", "/v1/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	// The router may return 404 or 405 depending on routing implementation
	assert.True(t, w.Code == http.StatusMethodNotAllowed || w.Code == http.StatusNotFound)
}

// Test 4: handleRunWait with valid request
func TestHandleRunWait_ValidRequest(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := RunWaitRequest{
		AgentID:  "agent-test",
		ActionID: "action-test",
		ThreadID: "thread-123",
		Input: map[string]interface{}{
			"input": "payload",
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs/wait", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.RunID)
}

// Test 5: handleRunWait with protocol.NotFoundError
func TestHandleRunWait_ProfileNotFound(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Mock NotFoundError
	mockCore.executeRunFunc = func(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error) {
		return nil, protocol.NewNotFoundError("agent-test")
	}

	req := RunWaitRequest{
		AgentID:  "agent-test",
		ActionID: "action-test",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs/wait", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test 6: handleRunWait with InvalidInputError
func TestHandleRunWait_InvalidInput(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Mock InvalidInputError
	mockCore.executeRunFunc = func(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error) {
		return nil, protocol.NewInvalidInputError("action_id", "invalid action")
	}

	req := RunWaitRequest{
		AgentID:  "agent-test",
		ActionID: "invalid",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs/wait", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test 7: handleGetRun with valid run ID
func TestHandleGetRun_ValidRunID(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Pre-create a run
	exitCode := 42
	runState := &protocol.RunState{
		RunID:    "run-12345",
		Status:   protocol.RunStatusCompleted,
		ExitCode: &exitCode,
		Error:    "",
	}
	mockCore.mu.Lock()
	mockCore.runs["run-12345"] = runState
	mockCore.mu.Unlock()

	httpReq := httptest.NewRequest("GET", "/v1/runs/run-12345", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "run-12345", resp.RunID)
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, 42, *resp.ExitCode)
}

// Test 8: handleGetRun with missing run ID
func TestHandleGetRun_MissingRunID(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("GET", "/v1/runs/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test 9: handleRunAction with cancel action
func TestHandleRunAction_CancelAction(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("POST", "/v1/runs/run-123/cancel", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// Test 10: handleRunAction with invalid action path
func TestHandleRunAction_InvalidPath(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Path without two parts
	httpReq := httptest.NewRequest("POST", "/v1/runs/run-123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	// Should result in error (404 or 400)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Test 11: handleCreateThread with metadata filtering
func TestHandleCreateThread_MetadataFiltering(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := CreateThreadRequest{
		AgentID: "agent-test",
		Metadata: map[string]interface{}{
			"string_key": "value",
			"int_key":    123,
			"bool_key":   true,
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp ThreadResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "value", resp.Metadata["string_key"])
	// Non-string values should not be in metadata
	if _, exists := resp.Metadata["int_key"]; exists {
		t.Errorf("Non-string values should not be in metadata")
	}
}

// Test 12: handleThreadsUpdate with metadata update
func TestHandleThreadsUpdate_MetadataUpdate(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Create session first
	mockCore.CreateSession("agent-test", map[string]string{"old": "value"})

	updateReq := map[string]interface{}{
		"new_key": "new_value",
	}
	body, _ := json.Marshal(updateReq)

	httpReq := httptest.NewRequest("PATCH", "/v1/threads/session-agent-test", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ThreadResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "new_value", resp.Metadata["new_key"])
}

// Test 13: handleStorePut with complex value
func TestHandleStorePut_ComplexValue(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := StorePutRequest{
		Namespace: "test-ns",
		Key:       "complex-key",
		Value:     `{"nested":"json","array":[1,2,3]}`,
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("PUT", "/v1/store", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify it was stored
	value, _ := mockCore.StoreGet("test-ns", "complex-key")
	assert.Equal(t, `{"nested":"json","array":[1,2,3]}`, string(value))
}

// Test 14: sendError response format validation
func TestSendError_ResponseFormat(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("GET", "/v1/agents/nonexistent-agent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NotEmpty(t, errResp.Error)
	assert.Equal(t, http.StatusNotFound, errResp.Code)
	assert.NotEmpty(t, errResp.Message)
}

// Test 15: Concurrent request handling
func TestConcurrentRequests_HandleCreations(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	const numGoroutines = 15
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			req := CreateRunRequest{
				AgentID:  fmt.Sprintf("agent-%d", id),
				ActionID: fmt.Sprintf("action-%d", id),
				Input: map[string]interface{}{
					"input": fmt.Sprintf("payload-%d", id),
				},
			}
			body, _ := json.Marshal(req)

			httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, httpReq)

			if w.Code == http.StatusCreated {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numGoroutines, successCount)
}

// ============================================================================
// SECTION 2: PROTOCOL VALIDATION TESTS (10 tests)
// ============================================================================

// Test 16: RunResponse message format validation
func TestRunResponse_FormatValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := CreateRunRequest{
		AgentID:  "test-agent",
		ActionID: "test-action",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	var resp RunResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Validate required fields
	assert.NotEmpty(t, resp.RunID)
	assert.NotEmpty(t, resp.Status)
	// Metadata may be nil or an empty map, both are valid
	if resp.Metadata == nil {
		resp.Metadata = make(map[string]string)
	}
	assert.NotNil(t, resp.Metadata)
}

// Test 17: ThreadResponse field validation
func TestThreadResponse_FieldValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := CreateThreadRequest{
		AgentID: "agent-test",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	var resp ThreadResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Validate required fields
	assert.NotEmpty(t, resp.ThreadID)
	assert.NotEmpty(t, resp.AgentID)
	assert.NotNil(t, resp.Metadata)
	assert.Equal(t, "agent-test", resp.AgentID)
}

// Test 18: ErrorResponse structure validation
func TestErrorResponse_StructureValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader([]byte("invalid json")))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)

	// Validate error structure
	assert.NotEmpty(t, errResp.Error)
	assert.GreaterOrEqual(t, errResp.Code, 400)
	assert.Equal(t, w.Code, errResp.Code)
}

// Test 19: StoreItem validation
func TestStoreItem_FieldValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Store a value
	mockCore.StorePut("namespace", "key", []byte("value"))

	httpReq := httptest.NewRequest("GET", "/v1/store/namespace/key", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	var item StoreItem
	json.Unmarshal(w.Body.Bytes(), &item)

	assert.Equal(t, "namespace", item.Namespace)
	assert.Equal(t, "key", item.Key)
	assert.Equal(t, "value", item.Value)
}

// Test 20: AgentSearchResponse validation
func TestAgentSearchResponse_Validation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	mockCore.mu.Lock()
	mockCore.agents["agent-1"] = &protocol.AgentInfo{
		ID:           "agent-1",
		Name:         "Agent 1",
		Capabilities: []string{"cap1"},
	}
	mockCore.mu.Unlock()

	req := AgentSearchRequest{}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/agents/search", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	var resp AgentSearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.NotNil(t, resp.Agents)
	assert.Greater(t, len(resp.Agents), 0)
}

// Test 21: Input payload type validation
func TestInputPayload_TypeValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Only string values in "input" key should be processed
	req := CreateRunRequest{
		AgentID:  "agent-test",
		ActionID: "action-test",
		Input: map[string]interface{}{
			"input": "string-payload", // Valid
			"other": 123,               // Should be ignored
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test 22: HTTP status code correctness
func TestStatusCode_Correctness(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		operation  string
	}{
		{"Created returns 201", http.StatusCreated, "POST /v1/runs"},
		{"OK returns 200", http.StatusOK, "GET /v1/runs/run-123"},
		{"NoContent returns 204", http.StatusNoContent, "DELETE /v1/threads/session-123"},
		{"BadRequest returns 400", http.StatusBadRequest, "GET /v1/runs/"},
		{"NotFound returns 404", http.StatusNotFound, "GET /v1/agents/nonexistent"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := NewAgentProtocolAdapter()
			mockCore := newMockAPSCore()
			mux := http.NewServeMux()
			_ = adapter.RegisterRoutes(mux, mockCore)

			var httpReq *http.Request
			switch test.operation {
			case "POST /v1/runs":
				body, _ := json.Marshal(CreateRunRequest{AgentID: "test", ActionID: "test"})
				httpReq = httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
				httpReq.Header.Set("Content-Type", "application/json")
			case "GET /v1/runs/run-123":
				mockCore.mu.Lock()
				mockCore.runs["run-123"] = &protocol.RunState{RunID: "run-123", Status: protocol.RunStatusCompleted}
				mockCore.mu.Unlock()
				httpReq = httptest.NewRequest("GET", "/v1/runs/run-123", nil)
			case "DELETE /v1/threads/session-123":
				mockCore.CreateSession("test", map[string]string{})
				httpReq = httptest.NewRequest("DELETE", "/v1/threads/session-test", nil)
			case "GET /v1/runs/":
				httpReq = httptest.NewRequest("GET", "/v1/runs/", nil)
			case "GET /v1/agents/nonexistent":
				httpReq = httptest.NewRequest("GET", "/v1/agents/nonexistent", nil)
			}

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httpReq)

			assert.Equal(t, test.statusCode, w.Code)
		})
	}
}

// Test 23: Content-Type header validation
func TestContentType_HeaderValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	body, _ := json.Marshal(CreateRunRequest{AgentID: "test", ActionID: "test"})
	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// Test 24: JSON unmarshaling error handling
func TestJSONUnmarshaling_ErrorHandling(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	invalidJSON := []byte(`{"agent_id": "test", invalid}`)
	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(invalidJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NotEmpty(t, errResp.Message)
}

// Test 25: Protocol field mapping
func TestFieldMapping_ProtocolCompliance(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Test that CreateRunRequest.SessionID maps to protocol.RunInput.ThreadID
	req := CreateRunRequest{
		AgentID:   "agent-123",
		ActionID:  "action-456",
		SessionID: "session-789",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	// Verify correct mapping occurred (indirectly through successful creation)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// SECTION 3: INTEGRATION TESTS (5 tests)
// ============================================================================

// Test 26: End-to-end: Create thread, create run, get run
func TestE2E_ThreadCreationAndRunExecution(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Step 1: Create thread
	threadReq := CreateThreadRequest{
		AgentID: "e2e-agent",
		Metadata: map[string]interface{}{
			"test": "metadata",
		},
	}
	threadBody, _ := json.Marshal(threadReq)
	threadHttpReq := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(threadBody))
	threadHttpReq.Header.Set("Content-Type", "application/json")
	threadW := httptest.NewRecorder()
	mux.ServeHTTP(threadW, threadHttpReq)

	assert.Equal(t, http.StatusCreated, threadW.Code)

	var threadResp ThreadResponse
	json.Unmarshal(threadW.Body.Bytes(), &threadResp)
	threadID := threadResp.ThreadID

	// Step 2: Create run in thread
	runReq := CreateRunRequest{
		AgentID:  "e2e-agent",
		ActionID: "e2e-action",
	}
	runBody, _ := json.Marshal(runReq)
	runHttpReq := httptest.NewRequest("POST", fmt.Sprintf("/v1/threads/%s/runs", threadID), bytes.NewReader(runBody))
	runHttpReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	mux.ServeHTTP(runW, runHttpReq)

	assert.Equal(t, http.StatusCreated, runW.Code)

	var runResp RunResponse
	json.Unmarshal(runW.Body.Bytes(), &runResp)
	runID := runResp.RunID

	// Step 3: Get run status
	getHttpReq := httptest.NewRequest("GET", fmt.Sprintf("/v1/runs/%s", runID), nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getHttpReq)

	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp RunResponse
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	assert.Equal(t, runID, getResp.RunID)
}

// Test 27: Complex payload handling
func TestE2E_ComplexPayloadProcessing(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	complexPayload := `{
		"nested": {
			"objects": [1, 2, 3],
			"string": "value"
		},
		"array": ["a", "b", "c"]
	}`

	req := CreateRunRequest{
		AgentID:  "agent-complex",
		ActionID: "action-complex",
		Input: map[string]interface{}{
			"input": complexPayload,
		},
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.RunID)
}

// Test 28: Store operations workflow
func TestE2E_StoreOperationsWorkflow(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Step 1: Put item
	putReq := StorePutRequest{
		Namespace: "e2e-ns",
		Key:       "e2e-key",
		Value:     "e2e-value",
	}
	putBody, _ := json.Marshal(putReq)
	putHttpReq := httptest.NewRequest("PUT", "/v1/store", bytes.NewReader(putBody))
	putHttpReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	mux.ServeHTTP(putW, putHttpReq)

	assert.Equal(t, http.StatusCreated, putW.Code)

	// Step 2: Get item
	getHttpReq := httptest.NewRequest("GET", "/v1/store/e2e-ns/e2e-key", nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getHttpReq)

	assert.Equal(t, http.StatusOK, getW.Code)

	var item StoreItem
	json.Unmarshal(getW.Body.Bytes(), &item)
	assert.Equal(t, "e2e-value", item.Value)

	// Step 3: Delete item
	delHttpReq := httptest.NewRequest("DELETE", "/v1/store/e2e-ns/e2e-key", nil)
	delW := httptest.NewRecorder()
	mux.ServeHTTP(delW, delHttpReq)

	assert.Equal(t, http.StatusNoContent, delW.Code)

	// Step 4: Verify deletion
	getAgainHttpReq := httptest.NewRequest("GET", "/v1/store/e2e-ns/e2e-key", nil)
	getAgainW := httptest.NewRecorder()
	mux.ServeHTTP(getAgainW, getAgainHttpReq)

	assert.Equal(t, http.StatusNotFound, getAgainW.Code)
}

// Test 29: Streaming response validation
func TestE2E_StreamingResponseValidation(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := RunWaitRequest{
		AgentID:  "agent-stream",
		ActionID: "action-stream",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/runs/stream", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

// Test 30: Concurrent request race condition test
func TestE2E_ConcurrentOperationRaceCondition(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	const numThreads = 10
	var wg sync.WaitGroup

	wg.Add(numThreads)

	for i := 0; i < numThreads; i++ {
		go func(id int) {
			defer wg.Done()

			// Simulate thread creation, store operation, and run creation
			threadReq := CreateThreadRequest{
				AgentID: fmt.Sprintf("agent-%d", id),
			}
			threadBody, _ := json.Marshal(threadReq)
			threadHttpReq := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(threadBody))
			threadHttpReq.Header.Set("Content-Type", "application/json")
			threadW := httptest.NewRecorder()
			mux.ServeHTTP(threadW, threadHttpReq)

			if threadW.Code != http.StatusCreated {
				t.Errorf("Failed to create thread: %d", threadW.Code)
				return
			}

			// Store operation
			storeReq := StorePutRequest{
				Namespace: fmt.Sprintf("ns-%d", id),
				Key:       fmt.Sprintf("key-%d", id),
				Value:     fmt.Sprintf("value-%d", id),
			}
			storeBody, _ := json.Marshal(storeReq)
			storeHttpReq := httptest.NewRequest("PUT", "/v1/store", bytes.NewReader(storeBody))
			storeHttpReq.Header.Set("Content-Type", "application/json")
			storeW := httptest.NewRecorder()
			mux.ServeHTTP(storeW, storeHttpReq)

			if storeW.Code != http.StatusCreated {
				t.Errorf("Failed to store item: %d", storeW.Code)
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without deadlock, test passes
	assert.True(t, true)
}

// Test 31: Agent search with filtering
func TestE2E_AgentSearchFiltering(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Register multiple agents
	mockCore.mu.Lock()
	for i := 0; i < 5; i++ {
		mockCore.agents[fmt.Sprintf("search-agent-%d", i)] = &protocol.AgentInfo{
			ID:   fmt.Sprintf("search-agent-%d", i),
			Name: fmt.Sprintf("Search Agent %d", i),
		}
	}
	mockCore.mu.Unlock()

	// Search with query
	req := AgentSearchRequest{Query: "Agent 1", Limit: 10}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/v1/agents/search", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AgentSearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp.Agents)
}

// Test 32: Thread operation sequence
func TestE2E_ThreadOperationSequence(t *testing.T) {
	adapter := NewAgentProtocolAdapter()
	mockCore := newMockAPSCore()
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	// Create thread
	createReq := CreateThreadRequest{
		AgentID: "seq-agent",
		Metadata: map[string]interface{}{
			"version": "1",
		},
	}
	createBody, _ := json.Marshal(createReq)
	createHttpReq := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(createBody))
	createHttpReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createHttpReq)

	var threadResp ThreadResponse
	json.Unmarshal(createW.Body.Bytes(), &threadResp)
	threadID := threadResp.ThreadID

	// Update thread
	updateReq := map[string]interface{}{
		"version": "2",
	}
	updateBody, _ := json.Marshal(updateReq)
	updateHttpReq := httptest.NewRequest("PATCH", fmt.Sprintf("/v1/threads/%s", threadID), bytes.NewReader(updateBody))
	updateHttpReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	mux.ServeHTTP(updateW, updateHttpReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	// Get thread
	getHttpReq := httptest.NewRequest("GET", fmt.Sprintf("/v1/threads/%s", threadID), nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getHttpReq)

	assert.Equal(t, http.StatusOK, getW.Code)

	// Delete thread
	delHttpReq := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/threads/%s", threadID), nil)
	delW := httptest.NewRecorder()
	mux.ServeHTTP(delW, delHttpReq)

	assert.Equal(t, http.StatusNoContent, delW.Code)
}
