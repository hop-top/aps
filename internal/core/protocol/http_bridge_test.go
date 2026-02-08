package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== DefaultHTTPBridge Tests ====================

// TestDefaultHTTPBridge_Name tests Name method returns correct name
func TestDefaultHTTPBridge_Name(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	name := bridge.Name()
	assert.Equal(t, "test-protocol-http-bridge", name)
	assert.Contains(t, name, "http-bridge")
	assert.Contains(t, name, "test-protocol")
}

// TestDefaultHTTPBridge_NameWithDifferentProtocols tests Name with various protocol names
func TestDefaultHTTPBridge_NameWithDifferentProtocols(t *testing.T) {
	tests := []struct {
		name             string
		protocolName     string
		expectedContains string
	}{
		{"acp protocol", "acp", "acp-http-bridge"},
		{"a2a protocol", "a2a", "a2a-http-bridge"},
		{"agent-protocol", "agent-protocol", "agent-protocol-http-bridge"},
		{"custom protocol", "custom", "custom-http-bridge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockProtocolServer{NameValue: tt.protocolName}
			bridge := NewDefaultHTTPBridge(mock)
			assert.Equal(t, tt.expectedContains, bridge.Name())
		})
	}
}

// TestDefaultHTTPBridge_Start delegates to wrapped server
func TestDefaultHTTPBridge_Start(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	ctx := context.Background()
	err := bridge.Start(ctx, nil)

	assert.NoError(t, err)
	assert.True(t, mock.StartCalled)
}

// TestDefaultHTTPBridge_StartWithConfig tests Start with config
func TestDefaultHTTPBridge_StartWithConfig(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	ctx := context.Background()
	config := map[string]interface{}{"port": 8080}
	err := bridge.Start(ctx, config)

	assert.NoError(t, err)
	assert.True(t, mock.StartCalled)
}

// TestDefaultHTTPBridge_StartWithError tests Start error propagation
func TestDefaultHTTPBridge_StartWithError(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue:  "test-protocol",
		StartError: fmt.Errorf("failed to start"),
	}
	bridge := NewDefaultHTTPBridge(mock)

	ctx := context.Background()
	err := bridge.Start(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start")
}

// TestDefaultHTTPBridge_StartWithContext tests Start respects context
func TestDefaultHTTPBridge_StartWithContext(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context

	bridge.Start(ctx, nil)
	// The mock doesn't actually check context, but verify call was made
	assert.True(t, mock.StartCalled)
}

// TestDefaultHTTPBridge_Stop delegates to wrapped server
func TestDefaultHTTPBridge_Stop(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	err := bridge.Stop()

	assert.NoError(t, err)
	assert.True(t, mock.StopCalled)
}

// TestDefaultHTTPBridge_StopWithError tests Stop error propagation
func TestDefaultHTTPBridge_StopWithError(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		StopError: fmt.Errorf("failed to stop"),
	}
	bridge := NewDefaultHTTPBridge(mock)

	err := bridge.Stop()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop")
}

// TestDefaultHTTPBridge_Status delegates to wrapped server
func TestDefaultHTTPBridge_Status(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		Status_:   "running",
	}
	bridge := NewDefaultHTTPBridge(mock)

	status := bridge.Status()

	assert.Equal(t, "running", status)
}

// TestDefaultHTTPBridge_StatusVariations tests Status with different statuses
func TestDefaultHTTPBridge_StatusVariations(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"running status", "running", "running"},
		{"stopped status", "stopped", "stopped"},
		{"error status", "error", "error"},
		{"initializing status", "initializing", "initializing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockProtocolServer{
				NameValue: "test-protocol",
				Status_:   tt.status,
			}
			bridge := NewDefaultHTTPBridge(mock)
			assert.Equal(t, tt.expected, bridge.Status())
		})
	}
}

// TestDefaultHTTPBridge_GetHTTPHandler returns handler
func TestDefaultHTTPBridge_GetHTTPHandler(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	handler := bridge.GetHTTPHandler()

	assert.NotNil(t, handler)
}

// TestDefaultHTTPBridge_HTTPHandlerIsHTTPHandler validates handler type
func TestDefaultHTTPBridge_HTTPHandlerIsHTTPHandler(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	handler := bridge.GetHTTPHandler()

	// Verify it's an http.Handler
	var _ http.Handler = handler
	assert.NotNil(t, handler)
}

// TestDefaultHTTPBridge_HandleHTTPRequest_GET tests GET request handling
func TestDefaultHTTPBridge_HandleHTTPRequest_GET(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		Status_:   "running",
	}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "test-protocol", response["protocol"])
	assert.Equal(t, "running", response["status"])
	assert.Equal(t, "GET", response["method"])
	assert.Equal(t, "/", response["path"])
}

// TestDefaultHTTPBridge_HandleHTTPRequest_POST tests POST request handling
func TestDefaultHTTPBridge_HandleHTTPRequest_POST(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		Status_:   "running",
	}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("POST", "/api/test", strings.NewReader(`{"test":"data"}`))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "POST", response["method"])
	assert.Equal(t, "/api/test", response["path"])
}

// TestDefaultHTTPBridge_HandleHTTPRequest_PUT tests PUT request handling
func TestDefaultHTTPBridge_HandleHTTPRequest_PUT(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("PUT", "/resource/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "PUT", response["method"])
}

// TestDefaultHTTPBridge_HandleHTTPRequest_DELETE tests DELETE request handling
func TestDefaultHTTPBridge_HandleHTTPRequest_DELETE(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("DELETE", "/resource/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "DELETE", response["method"])
}

// TestDefaultHTTPBridge_HandleHTTPRequest_IncludesProtocolInfo tests response includes protocol info
func TestDefaultHTTPBridge_HandleHTTPRequest_IncludesProtocolInfo(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "custom-protocol",
		Status_:   "initializing",
	}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "custom-protocol", response["protocol"])
	assert.Equal(t, "initializing", response["status"])
	assert.Equal(t, "HTTP bridge is active", response["message"])
}

// TestDefaultHTTPBridge_HandleHTTPRequest_VariousPath tests various paths
func TestDefaultHTTPBridge_HandleHTTPRequest_VariousPath(t *testing.T) {
	paths := []string{"/", "/api/v1/resource", "/health", "/deep/nested/path"}

	for _, path := range paths {
		t.Run(fmt.Sprintf("path %s", path), func(t *testing.T) {
			mock := &MockProtocolServer{NameValue: "test-protocol"}
			bridge := NewDefaultHTTPBridge(mock)
			handler := bridge.GetHTTPHandler()

			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, path, response["path"])
		})
	}
}

// TestDefaultHTTPBridge_ConcurrentRequests tests concurrent request handling
func TestDefaultHTTPBridge_ConcurrentRequests(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/path%d", index), nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()
}

// TestDefaultHTTPBridge_IsHTTPBridge validates HTTPBridge implementation
func TestDefaultHTTPBridge_IsHTTPBridge(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	var bridge HTTPBridge
	bridge = NewDefaultHTTPBridge(mock)

	assert.NotNil(t, bridge)
	assert.NotEmpty(t, bridge.Name())
	assert.NotNil(t, bridge.GetHTTPHandler())
}

// ==================== JSONRPCHTTPBridge Tests ====================

// TestJSONRPCHTTPBridge_Name tests Name method
func TestJSONRPCHTTPBridge_Name(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)

	name := bridge.Name()
	assert.Equal(t, "acp-http", name)
	assert.Contains(t, name, "-http")
}

// TestJSONRPCHTTPBridge_NameVariations tests Name with different protocols
func TestJSONRPCHTTPBridge_NameVariations(t *testing.T) {
	tests := []struct {
		name             string
		protocolName     string
		expectedContains string
	}{
		{"acp", "acp", "acp-http"},
		{"a2a", "a2a", "a2a-http"},
		{"agent-protocol", "agent-protocol", "agent-protocol-http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockProtocolServer{NameValue: tt.protocolName}
			bridge := NewJSONRPCHTTPBridge(mock)
			assert.Equal(t, tt.expectedContains, bridge.Name())
		})
	}
}

// TestJSONRPCHTTPBridge_Start delegates to wrapped server
func TestJSONRPCHTTPBridge_Start(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)

	ctx := context.Background()
	err := bridge.Start(ctx, nil)

	assert.NoError(t, err)
	assert.True(t, mock.StartCalled)
}

// TestJSONRPCHTTPBridge_StartWithError tests Start error propagation
func TestJSONRPCHTTPBridge_StartWithError(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue:  "acp",
		StartError: fmt.Errorf("startup failed"),
	}
	bridge := NewJSONRPCHTTPBridge(mock)

	ctx := context.Background()
	err := bridge.Start(ctx, nil)

	assert.Error(t, err)
}

// TestJSONRPCHTTPBridge_Stop delegates to wrapped server
func TestJSONRPCHTTPBridge_Stop(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)

	err := bridge.Stop()

	assert.NoError(t, err)
	assert.True(t, mock.StopCalled)
}

// TestJSONRPCHTTPBridge_Status delegates to wrapped server
func TestJSONRPCHTTPBridge_Status(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "acp",
		Status_:   "running",
	}
	bridge := NewJSONRPCHTTPBridge(mock)

	status := bridge.Status()

	assert.Equal(t, "running", status)
}

// TestJSONRPCHTTPBridge_GetHTTPHandler returns handler
func TestJSONRPCHTTPBridge_GetHTTPHandler(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)

	handler := bridge.GetHTTPHandler()

	assert.NotNil(t, handler)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_ValidRequest tests valid JSON-RPC request
func TestJSONRPCHTTPBridge_HandleJSONRPC_ValidRequest(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "acp",
		Status_:   "running",
	}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response JSONRPCRequest
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, float64(1), response.ID)
	assert.NotNil(t, response.Result)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_ValidRequest_WithParams tests request with params
func TestJSONRPCHTTPBridge_HandleJSONRPC_ValidRequest_WithParams(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		Params: map[string]interface{}{
			"key": "value",
		},
		ID: 1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_NoID tests request without ID (notification)
func TestJSONRPCHTTPBridge_HandleJSONRPC_NoID(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_InvalidVersion tests invalid JSON-RPC version
func TestJSONRPCHTTPBridge_HandleJSONRPC_InvalidVersion(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "1.0", // Invalid version
		Method:  "test_method",
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["error"])
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_MissingMethod tests missing method
func TestJSONRPCHTTPBridge_HandleJSONRPC_MissingMethod(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "", // Empty method
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_InvalidJSON tests malformed JSON
func TestJSONRPCHTTPBridge_HandleJSONRPC_InvalidJSON(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{invalid json`))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_EmptyBody tests empty request body
func TestJSONRPCHTTPBridge_HandleJSONRPC_EmptyBody(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_ErrorResponse tests error response format
func TestJSONRPCHTTPBridge_HandleJSONRPC_ErrorResponse(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "1.0", // Invalid
		Method:  "test",
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.NotNil(t, response["error"])
	errMap := response["error"].(map[string]interface{})
	assert.NotNil(t, errMap["code"])
	assert.NotNil(t, errMap["message"])
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_ResponseIncludesProtocolInfo tests response includes protocol info
func TestJSONRPCHTTPBridge_HandleJSONRPC_ResponseIncludesProtocolInfo(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "custom-acp",
		Status_:   "running",
	}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response JSONRPCRequest
	json.Unmarshal(w.Body.Bytes(), &response)
	resultMap := response.Result.(map[string]interface{})
	assert.Equal(t, "custom-acp", resultMap["protocol"])
	assert.Equal(t, "running", resultMap["status"])
}

// TestJSONRPCHTTPBridge_HandleJSONRPC_ConcurrentRequests tests concurrent JSON-RPC requests
func TestJSONRPCHTTPBridge_HandleJSONRPC_ConcurrentRequests(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			request := JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  fmt.Sprintf("method_%d", index),
				ID:      index,
			}
			body, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()
}

// TestJSONRPCHTTPBridge_IsHTTPBridge validates HTTPBridge implementation
func TestJSONRPCHTTPBridge_IsHTTPBridge(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "acp"}
	var bridge HTTPBridge
	bridge = NewJSONRPCHTTPBridge(mock)

	assert.NotNil(t, bridge)
	assert.NotEmpty(t, bridge.Name())
	assert.NotNil(t, bridge.GetHTTPHandler())
}

// ==================== Error Handler Tests ====================

// TestWriteJSONRPCError_BasicError tests basic error response
func TestWriteJSONRPCError_BasicError(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSONRPCError(w, http.StatusBadRequest, -32700, "Parse error", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.NotNil(t, response["error"])
	errMap := response["error"].(map[string]interface{})
	assert.Equal(t, float64(-32700), errMap["code"])
	assert.Equal(t, "Parse error", errMap["message"])
}

// TestWriteJSONRPCError_WithData tests error with data field
func TestWriteJSONRPCError_WithData(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"details": "additional info"}

	writeJSONRPCError(w, http.StatusBadRequest, -32700, "Parse error", data)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	errMap := response["error"].(map[string]interface{})
	assert.NotNil(t, errMap["data"])
}

// TestWriteJSONRPCError_VariousHTTPStatus tests various HTTP status codes
func TestWriteJSONRPCError_VariousHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		httpStatus int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONRPCError(w, tt.httpStatus, -32700, "Error", nil)
			assert.Equal(t, tt.httpStatus, w.Code)
		})
	}
}

// TestWriteJSONRPCError_StandardErrorCodes tests JSON-RPC standard error codes
func TestWriteJSONRPCError_StandardErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
	}{
		{"Parse error", -32700, "Parse error"},
		{"Invalid Request", -32600, "Invalid Request"},
		{"Method not found", -32601, "Method not found"},
		{"Invalid params", -32602, "Invalid params"},
		{"Internal error", -32603, "Internal error"},
		{"Server error", -32000, "Server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONRPCError(w, http.StatusBadRequest, tt.code, tt.message, nil)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			errMap := response["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.code), errMap["code"])
			assert.Equal(t, tt.message, errMap["message"])
		})
	}
}

// ==================== JSONRPCRequest Tests ====================

// TestJSONRPCRequest_Marshaling tests JSON-RPC request marshaling
func TestJSONRPCRequest_Marshaling(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		Params: map[string]interface{}{
			"key": "value",
		},
		ID: 1,
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "2.0")
	assert.Contains(t, string(data), "test_method")
}

// TestJSONRPCRequest_Unmarshaling tests JSON-RPC request unmarshaling
func TestJSONRPCRequest_Unmarshaling(t *testing.T) {
	jsonData := []byte(`{
		"jsonrpc": "2.0",
		"method": "test_method",
		"params": {"key": "value"},
		"id": 1
	}`)

	var req JSONRPCRequest
	err := json.Unmarshal(jsonData, &req)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, "test_method", req.Method)
	assert.NotNil(t, req.Params)
}

// TestJSONRPCRequest_WithoutID tests request without ID (notification)
func TestJSONRPCRequest_WithoutID(t *testing.T) {
	jsonData := []byte(`{
		"jsonrpc": "2.0",
		"method": "test_method"
	}`)

	var req JSONRPCRequest
	err := json.Unmarshal(jsonData, &req)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, "test_method", req.Method)
	assert.Nil(t, req.ID)
}

// TestJSONRPCRequest_WithError tests request with error field
func TestJSONRPCRequest_WithError(t *testing.T) {
	jsonData := []byte(`{
		"jsonrpc": "2.0",
		"error": {"code": -32700, "message": "Parse error"},
		"id": 1
	}`)

	var req JSONRPCRequest
	err := json.Unmarshal(jsonData, &req)
	assert.NoError(t, err)
	assert.NotNil(t, req.Error)
	assert.Equal(t, -32700, req.Error.Code)
	assert.Equal(t, "Parse error", req.Error.Message)
}

// TestJSONRPCRequest_StringID tests request with string ID
func TestJSONRPCRequest_StringID(t *testing.T) {
	jsonData := []byte(`{
		"jsonrpc": "2.0",
		"method": "test_method",
		"id": "string-id"
	}`)

	var req JSONRPCRequest
	err := json.Unmarshal(jsonData, &req)
	assert.NoError(t, err)
	assert.Equal(t, "string-id", req.ID)
}

// ==================== ProtocolServerAdapter Tests ====================

// TestProtocolServerAdapter_Name tests Name method
func TestProtocolServerAdapter_Name(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	adapter := NewProtocolServerAdapter(mock)

	name := adapter.Name()
	assert.Equal(t, "test-protocol", name)
}

// TestProtocolServerAdapter_Start tests Start delegation
func TestProtocolServerAdapter_Start(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	adapter := NewProtocolServerAdapter(mock)

	ctx := context.Background()
	err := adapter.Start(ctx, nil)

	assert.NoError(t, err)
	assert.True(t, mock.StartCalled)
}

// TestProtocolServerAdapter_Stop tests Stop delegation
func TestProtocolServerAdapter_Stop(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	adapter := NewProtocolServerAdapter(mock)

	err := adapter.Stop()

	assert.NoError(t, err)
	assert.True(t, mock.StopCalled)
}

// TestProtocolServerAdapter_Status tests Status delegation
func TestProtocolServerAdapter_Status(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		Status_:   "running",
	}
	adapter := NewProtocolServerAdapter(mock)

	status := adapter.Status()

	assert.Equal(t, "running", status)
}

// TestProtocolServerAdapter_RegisterRoutes is no-op
func TestProtocolServerAdapter_RegisterRoutes(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	adapter := NewProtocolServerAdapter(mock)

	mux := http.NewServeMux()
	err := adapter.RegisterRoutes(mux, nil)

	assert.NoError(t, err)
}

// TestProtocolServerAdapter_RegisterRoutesWithCore tests RegisterRoutes with APSCore
func TestProtocolServerAdapter_RegisterRoutesWithCore(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	adapter := NewProtocolServerAdapter(mock)

	mux := http.NewServeMux()
	mockCore := &MockAPSCore{}
	err := adapter.RegisterRoutes(mux, mockCore)

	assert.NoError(t, err)
}

// TestProtocolServerAdapter_IsHTTPProtocolAdapter validates HTTPProtocolAdapter
func TestProtocolServerAdapter_IsHTTPProtocolAdapter(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	var adapter HTTPProtocolAdapter
	adapter = NewProtocolServerAdapter(mock)

	assert.NotNil(t, adapter)
	assert.NotEmpty(t, adapter.Name())
	assert.NoError(t, adapter.Start(context.Background(), nil))
	assert.NoError(t, adapter.Stop())
}

// ==================== Integration Tests ====================

// TestBridges_LifecycleSequence tests complete lifecycle
func TestBridges_LifecycleSequence(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "test-protocol",
		Status_:   "stopped",
	}
	bridge := NewDefaultHTTPBridge(mock)

	// Start
	ctx := context.Background()
	err := bridge.Start(ctx, nil)
	require.NoError(t, err)
	assert.True(t, mock.StartCalled)

	// Get handler
	handler := bridge.GetHTTPHandler()
	assert.NotNil(t, handler)

	// Verify status
	mock.Status_ = "running"
	assert.Equal(t, "running", bridge.Status())

	// Stop
	err = bridge.Stop()
	assert.NoError(t, err)
	assert.True(t, mock.StopCalled)
}

// TestJSONRPCBridge_FullWorkflow tests complete JSON-RPC workflow
func TestJSONRPCBridge_FullWorkflow(t *testing.T) {
	mock := &MockProtocolServer{
		NameValue: "acp",
		Status_:   "running",
	}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	// Valid request
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		ID:      1,
	}
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Invalid request
	badBody := []byte(`{invalid`)
	req = httptest.NewRequest("POST", "/", bytes.NewReader(badBody))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestMultipleBridges_IndependentServers tests multiple bridges with different servers
func TestMultipleBridges_IndependentServers(t *testing.T) {
	mock1 := &MockProtocolServer{NameValue: "protocol-1"}
	mock2 := &MockProtocolServer{NameValue: "protocol-2"}

	bridge1 := NewDefaultHTTPBridge(mock1)
	bridge2 := NewJSONRPCHTTPBridge(mock2)

	assert.Equal(t, "protocol-1-http-bridge", bridge1.Name())
	assert.Equal(t, "protocol-2-http", bridge2.Name())

	handler1 := bridge1.GetHTTPHandler()
	handler2 := bridge2.GetHTTPHandler()

	// Both should be valid handlers but different instances
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

// TestHTTPBridge_RequestBodyValidation tests request body validation
func TestHTTPBridge_RequestBodyValidation(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		expectedCode  int
		expectedError bool
	}{
		{
			name:          "valid JSON-RPC",
			body:          `{"jsonrpc":"2.0","method":"test","id":1}`,
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "invalid JSON",
			body:          `{invalid}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
		{
			name:          "empty body",
			body:          ``,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
		{
			name:          "malformed JSON",
			body:          `{"jsonrpc":"2.0"`,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockProtocolServer{NameValue: "test"}
			bridge := NewJSONRPCHTTPBridge(mock)
			handler := bridge.GetHTTPHandler()

			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

// TestHTTPBridge_ResponseHeaders tests response headers
func TestHTTPBridge_ResponseHeaders(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	bridge := NewDefaultHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Body.String())
}

// TestHTTPBridge_LargeRequest tests handling of large requests
func TestHTTPBridge_LargeRequest(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	largeParams := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeParams[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test_method",
		Params:  largeParams,
		ID:      1,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHTTPBridge_SpecialCharactersInMethod tests special characters in method name
func TestHTTPBridge_SpecialCharactersInMethod(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	methods := []string{
		"method_with_underscore",
		"methodWithCamelCase",
		"method-with-dash",
		"method.with.dots",
	}

	for _, method := range methods {
		t.Run(fmt.Sprintf("method %s", method), func(t *testing.T) {
			request := JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  method,
				ID:      1,
			}
			body, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestDefaultHTTPBridge_ThreadSafety tests thread safety with multiple goroutines
func TestDefaultHTTPBridge_ThreadSafety(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	bridge := NewDefaultHTTPBridge(mock)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Call Name multiple times
			for j := 0; j < 10; j++ {
				name := bridge.Name()
				if name == "" {
					errors <- fmt.Errorf("empty name")
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestJSONRPCHTTPBridge_ResponseFormat tests response format compliance
func TestJSONRPCHTTPBridge_ResponseFormat(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	bridge := NewJSONRPCHTTPBridge(mock)
	handler := bridge.GetHTTPHandler()

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "test",
		ID:      123,
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Verify required fields
	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.NotNil(t, response["id"])
	assert.NotNil(t, response["result"])

	// Verify no error field in success response
	assert.Nil(t, response["error"])
}
