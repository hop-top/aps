package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ap "hop.top/aps/internal/adapters/agentprotocol"
	"hop.top/aps/internal/core/protocol"

	"github.com/stretchr/testify/assert"
)

type mockCore struct{}

func (m *mockCore) ExecuteRun(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error) {
	return &protocol.RunState{
		RunID:    "test-run-id",
		Status:   protocol.RunStatusCompleted,
		ExitCode: func() *int { i := 0; return &i }(),
	}, nil
}

func (m *mockCore) GetRun(runID string) (*protocol.RunState, error) {
	return &protocol.RunState{
		RunID:    runID,
		Status:   protocol.RunStatusCompleted,
		ExitCode: func() *int { i := 0; return &i }(),
	}, nil
}

func (m *mockCore) CancelRun(ctx context.Context, runID string) error {
	return nil
}

func (m *mockCore) GetAgent(profileID string) (*protocol.AgentInfo, error) {
	return &protocol.AgentInfo{
		ID:           profileID,
		Name:         "Test Agent",
		Description:  "Test Description",
		Capabilities: []string{"test"},
	}, nil
}

func (m *mockCore) ListAgents() ([]protocol.AgentInfo, error) {
	return []protocol.AgentInfo{
		{
			ID:           "agent-1",
			Name:         "Agent 1",
			Description:  "Description 1",
			Capabilities: []string{"cap1"},
		},
	}, nil
}

func (m *mockCore) GetAgentSchemas(profileID string) ([]protocol.ActionSchema, error) {
	return []protocol.ActionSchema{
		{
			Name:        "action-1",
			Description: "Action 1",
			Input:       map[string]interface{}{"type": "string"},
		},
	}, nil
}

func (m *mockCore) CreateSession(profileID string, metadata map[string]string) (*protocol.SessionState, error) {
	return &protocol.SessionState{
		SessionID: "test-session",
		ProfileID: profileID,
		Metadata:  metadata,
	}, nil
}

func (m *mockCore) GetSession(sessionID string) (*protocol.SessionState, error) {
	return &protocol.SessionState{
		SessionID: sessionID,
		ProfileID: "test-profile",
		Metadata:  map[string]string{"key": "value"},
	}, nil
}

func (m *mockCore) UpdateSession(sessionID string, metadata map[string]string) error {
	return nil
}

func (m *mockCore) HeartbeatSession(_ string) error {
	return nil
}

func (m *mockCore) DeleteSession(sessionID string) error {
	return nil
}

func (m *mockCore) ListSessions(profileID string) ([]protocol.SessionState, error) {
	return []protocol.SessionState{
		{
			SessionID: "session-1",
			ProfileID: profileID,
			Metadata:  map[string]string{},
		},
	}, nil
}

func (m *mockCore) StorePut(namespace string, key string, value []byte) error {
	return nil
}

func (m *mockCore) StoreGet(namespace string, key string) ([]byte, error) {
	return []byte("value"), nil
}

func (m *mockCore) StoreDelete(namespace string, key string) error {
	return nil
}

func (m *mockCore) StoreSearch(namespace string, prefix string) (map[string][]byte, error) {
	return map[string][]byte{"key1": []byte("value1")}, nil
}

func (m *mockCore) StoreListNamespaces() ([]string, error) {
	return []string{"ns1"}, nil
}

func TestAgentProtocolAdapter_Name(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	assert.Equal(t, "agent-protocol", adapter.Name())
}

func TestAgentProtocolAdapter_RegisterRoutes(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()

	err := adapter.RegisterRoutes(mux, mockCore)
	assert.NoError(t, err)

	t.Run("health check returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAgentProtocol_CreateRunEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("successful run creation", func(t *testing.T) {
		body := ap.CreateRunRequest{
			AgentID:  "test-profile",
			ActionID: "test-action",
			Input:    map[string]interface{}{"input": "test"},
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp ap.RunResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-run-id", resp.RunID)
		assert.Equal(t, "completed", resp.Status)
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/runs", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAgentProtocol_RunWaitEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("successful run wait", func(t *testing.T) {
		body := ap.RunWaitRequest{
			AgentID:  "test-profile",
			ActionID: "test-action",
			Input:    map[string]interface{}{"input": "test"},
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/v1/runs/wait", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.RunResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-run-id", resp.RunID)
	})
}

func TestAgentProtocol_GetRunEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("get existing run", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/runs/test-run-id", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.RunResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-run-id", resp.RunID)
	})

	t.Run("get run without id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/runs/", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAgentProtocol_AgentsEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("list agents", func(t *testing.T) {
		body, _ := json.Marshal(ap.AgentSearchRequest{})
		req := httptest.NewRequest("POST", "/v1/agents/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.AgentSearchResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Agents, 1)
		assert.Equal(t, "agent-1", resp.Agents[0].ID)
	})

	t.Run("get agent detail", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/agents/test-profile", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.AgentDetailResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-profile", resp.ID)
		assert.Len(t, resp.Schemas, 1)
	})
}

func TestAgentProtocol_ThreadsEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("create thread", func(t *testing.T) {
		body := ap.CreateThreadRequest{
			AgentID:  "test-profile",
			Metadata: map[string]interface{}{"key": "value"},
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/v1/threads", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp ap.ThreadResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-session", resp.ThreadID)
	})

	t.Run("get thread", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/threads/test-session", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.ThreadResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-session", resp.ThreadID)
	})
}

func TestAgentProtocol_StoreEndpoints(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	t.Run("store put", func(t *testing.T) {
		body := ap.StorePutRequest{
			Namespace: "test-ns",
			Key:       "test-key",
			Value:     "test-value",
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/v1/store", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("store get", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/store/test-ns/test-key", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp ap.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "test-ns", resp.Namespace)
		assert.Equal(t, "test-key", resp.Key)
	})

	t.Run("store delete", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/v1/store/test-ns/test-key", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestAgentProtocol_RunCancelEndpoint(t *testing.T) {
	adapter := ap.NewAgentProtocolAdapter()
	mockCore := &mockCore{}
	mux := http.NewServeMux()
	_ = adapter.RegisterRoutes(mux, mockCore)

	req := httptest.NewRequest("POST", "/v1/runs/test-run/cancel", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
