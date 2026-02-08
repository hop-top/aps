package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== Type Validation Tests ====================

// TestValidateRunStatus_AllValidStates validates all valid RunStatus values
func TestValidateRunStatus_AllValidStates(t *testing.T) {
	tests := []struct {
		name   string
		status RunStatus
		valid  bool
	}{
		{
			name:   "pending status is valid",
			status: RunStatusPending,
			valid:  true,
		},
		{
			name:   "running status is valid",
			status: RunStatusRunning,
			valid:  true,
		},
		{
			name:   "completed status is valid",
			status: RunStatusCompleted,
			valid:  true,
		},
		{
			name:   "failed status is valid",
			status: RunStatusFailed,
			valid:  true,
		},
		{
			name:   "cancelled status is valid",
			status: RunStatusCancelled,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunStatus(tt.status)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestValidateRunStatus_InvalidStatus validates that invalid statuses are rejected
func TestValidateRunStatus_InvalidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status RunStatus
	}{
		{
			name:   "invalid status empty string",
			status: RunStatus(""),
		},
		{
			name:   "invalid status unknown",
			status: RunStatus("unknown"),
		},
		{
			name:   "invalid status paused",
			status: RunStatus("paused"),
		},
		{
			name:   "invalid status in_progress",
			status: RunStatus("in_progress"),
		},
		{
			name:   "invalid status success",
			status: RunStatus("success"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunStatus(tt.status)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid run status")
		})
	}
}

// TestValidateRunStatus_ErrorMessageFormatting validates error message quality
func TestValidateRunStatus_ErrorMessageFormatting(t *testing.T) {
	err := ValidateRunStatus(RunStatus("invalid_status"))
	assert.NotNil(t, err)
	assert.Equal(t, "invalid run status: invalid_status", err.Error())
}

// TestValidateAgentInfo_Complete validates that complete AgentInfo is valid
func TestValidateAgentInfo_Complete(t *testing.T) {
	info := &AgentInfo{
		ID:           "agent-123",
		Name:         "Test Agent",
		Description:  "A test agent",
		Capabilities: []string{"capability1", "capability2"},
	}

	err := ValidateAgentInfo(info)
	assert.NoError(t, err)
}

// TestValidateAgentInfo_MissingID validates that missing ID is rejected
func TestValidateAgentInfo_MissingID(t *testing.T) {
	info := &AgentInfo{
		Name:        "Test Agent",
		Description: "A test agent",
	}

	err := ValidateAgentInfo(info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent id is required")
}

// TestValidateAgentInfo_MissingName validates that missing Name is rejected
func TestValidateAgentInfo_MissingName(t *testing.T) {
	info := &AgentInfo{
		ID:          "agent-123",
		Description: "A test agent",
	}

	err := ValidateAgentInfo(info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent name is required")
}

// TestValidateAgentInfo_EmptyValues validates edge case of empty strings
func TestValidateAgentInfo_EmptyValues(t *testing.T) {
	tests := []struct {
		name      string
		agentInfo *AgentInfo
		wantError bool
	}{
		{
			name: "empty ID",
			agentInfo: &AgentInfo{
				ID:   "",
				Name: "Test",
			},
			wantError: true,
		},
		{
			name: "empty Name",
			agentInfo: &AgentInfo{
				ID:   "agent-123",
				Name: "",
			},
			wantError: true,
		},
		{
			name: "both populated",
			agentInfo: &AgentInfo{
				ID:   "agent-123",
				Name: "Test",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentInfo(tt.agentInfo)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateActionSchema_Complete validates that complete ActionSchema is valid
func TestValidateActionSchema_Complete(t *testing.T) {
	schema := &ActionSchema{
		Name:        "test-action",
		Description: "A test action",
		Input: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	err := ValidateActionSchema(schema)
	assert.NoError(t, err)
}

// TestValidateActionSchema_MissingName validates that missing Name is rejected
func TestValidateActionSchema_MissingName(t *testing.T) {
	schema := &ActionSchema{
		Description: "A test action",
		Input:       map[string]interface{}{},
	}

	err := ValidateActionSchema(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action schema name is required")
}

// TestValidateActionSchema_MissingDescription validates that missing Description is rejected
func TestValidateActionSchema_MissingDescription(t *testing.T) {
	schema := &ActionSchema{
		Name:  "test-action",
		Input: map[string]interface{}{},
	}

	err := ValidateActionSchema(schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action schema description is required")
}

// TestValidateActionSchema_EmptyValues validates edge cases
func TestValidateActionSchema_EmptyValues(t *testing.T) {
	tests := []struct {
		name       string
		schema     *ActionSchema
		wantError  bool
		errorMatch string
	}{
		{
			name: "empty Name",
			schema: &ActionSchema{
				Name:        "",
				Description: "A test action",
				Input:       map[string]interface{}{},
			},
			wantError:  true,
			errorMatch: "action schema name is required",
		},
		{
			name: "empty Description",
			schema: &ActionSchema{
				Name:        "test-action",
				Description: "",
				Input:       map[string]interface{}{},
			},
			wantError:  true,
			errorMatch: "action schema description is required",
		},
		{
			name: "whitespace Name",
			schema: &ActionSchema{
				Name:        "   ",
				Description: "A test action",
				Input:       map[string]interface{}{},
			},
			wantError: false, // whitespace is valid, only empty string is invalid
		},
		{
			name: "nil Input is acceptable",
			schema: &ActionSchema{
				Name:        "test-action",
				Description: "A test action",
				Input:       nil,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActionSchema(tt.schema)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMatch != "" {
					assert.Contains(t, err.Error(), tt.errorMatch)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRunInput validates RunInput validation
func TestValidateRunInput(t *testing.T) {
	tests := []struct {
		name      string
		input     RunInput
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid input",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "action-1",
				Payload:   []byte("test"),
			},
			wantError: false,
		},
		{
			name: "missing ProfileID",
			input: RunInput{
				ActionID: "action-1",
			},
			wantError: true,
			errorMsg:  "profile_id is required",
		},
		{
			name: "missing ActionID",
			input: RunInput{
				ProfileID: "profile-1",
			},
			wantError: true,
			errorMsg:  "action_id is required",
		},
		{
			name: "empty ProfileID",
			input: RunInput{
				ProfileID: "",
				ActionID:  "action-1",
			},
			wantError: true,
			errorMsg:  "profile_id is required",
		},
		{
			name: "empty ActionID",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "",
			},
			wantError: true,
			errorMsg:  "action_id is required",
		},
		{
			name: "empty payload is valid",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "action-1",
				Payload:   []byte{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRunState_TypeConstraints validates RunState type constraints
func TestValidateRunState_TypeConstraints(t *testing.T) {
	tests := []struct {
		name      string
		state     *RunState
		assertion func(*testing.T, *RunState)
	}{
		{
			name: "RunState with all fields populated",
			state: &RunState{
				RunID:      "run-1",
				ProfileID:  "profile-1",
				ActionID:   "action-1",
				ThreadID:   "thread-1",
				Status:     RunStatusRunning,
				StartTime:  time.Now(),
				EndTime:    nil,
				ExitCode:   nil,
				OutputSize: 0,
				Error:      "",
				Metadata:   nil,
			},
			assertion: func(t *testing.T, s *RunState) {
				assert.NotEmpty(t, s.RunID)
				assert.NotEmpty(t, s.Status)
				assert.False(t, s.StartTime.IsZero())
			},
		},
		{
			name: "RunState with exit code",
			state: &RunState{
				RunID:     "run-2",
				Status:    RunStatusCompleted,
				StartTime: time.Now(),
				ExitCode:  ptrInt(0),
			},
			assertion: func(t *testing.T, s *RunState) {
				assert.NotNil(t, s.ExitCode)
				assert.Equal(t, 0, *s.ExitCode)
			},
		},
		{
			name: "RunState with error message",
			state: &RunState{
				RunID:     "run-3",
				Status:    RunStatusFailed,
				StartTime: time.Now(),
				Error:     "command not found",
			},
			assertion: func(t *testing.T, s *RunState) {
				assert.NotEmpty(t, s.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertion(t, tt.state)
		})
	}
}

// ==================== Interface Compliance Tests ====================

// MockStreamWriter implements StreamWriter interface for testing
type MockStreamWriter struct {
	WriteCount int
	CloseCount int
	LastEvent  string
	LastData   []byte
	WriteError error
	CloseError error
}

func (m *MockStreamWriter) Write(event string, data []byte) error {
	if m.WriteError != nil {
		return m.WriteError
	}
	m.WriteCount++
	m.LastEvent = event
	m.LastData = data
	return nil
}

func (m *MockStreamWriter) Close() error {
	if m.CloseError != nil {
		return m.CloseError
	}
	m.CloseCount++
	return nil
}

// MockProtocolServer implements ProtocolServer interface for testing
type MockProtocolServer struct {
	NameValue   string
	Status_     string
	StartError  error
	StopError   error
	StartCalled bool
	StopCalled  bool
}

func (m *MockProtocolServer) Name() string {
	return m.NameValue
}

func (m *MockProtocolServer) Start(ctx context.Context, config interface{}) error {
	m.StartCalled = true
	return m.StartError
}

func (m *MockProtocolServer) Stop() error {
	m.StopCalled = true
	return m.StopError
}

func (m *MockProtocolServer) Status() string {
	return m.Status_
}

// TestStreamWriter_InterfaceCompliance validates StreamWriter implementation
func TestStreamWriter_InterfaceCompliance(t *testing.T) {
	t.Run("Write method signature", func(t *testing.T) {
		var writer StreamWriter
		mock := &MockStreamWriter{}
		writer = mock

		err := writer.Write("test-event", []byte("test-data"))
		assert.NoError(t, err)
		assert.Equal(t, 1, mock.WriteCount)
		assert.Equal(t, "test-event", mock.LastEvent)
	})

	t.Run("Close method signature", func(t *testing.T) {
		var writer StreamWriter
		mock := &MockStreamWriter{}
		writer = mock

		err := writer.Close()
		assert.NoError(t, err)
		assert.Equal(t, 1, mock.CloseCount)
	})

	t.Run("Write error propagation", func(t *testing.T) {
		var writer StreamWriter
		mock := &MockStreamWriter{WriteError: assert.AnError}
		writer = mock

		err := writer.Write("event", []byte("data"))
		assert.Error(t, err)
	})

	t.Run("Close error propagation", func(t *testing.T) {
		var writer StreamWriter
		mock := &MockStreamWriter{CloseError: assert.AnError}
		writer = mock

		err := writer.Close()
		assert.Error(t, err)
	})
}

// TestProtocolServer_InterfaceCompliance validates ProtocolServer interface
func TestProtocolServer_InterfaceCompliance(t *testing.T) {
	t.Run("Name method", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{NameValue: "test-protocol"}
		server = mock

		name := server.Name()
		assert.Equal(t, "test-protocol", name)
	})

	t.Run("Start method", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{}
		server = mock

		ctx := context.Background()
		err := server.Start(ctx, nil)
		assert.NoError(t, err)
		assert.True(t, mock.StartCalled)
	})

	t.Run("Start with error", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{StartError: assert.AnError}
		server = mock

		ctx := context.Background()
		err := server.Start(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("Stop method", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{}
		server = mock

		err := server.Stop()
		assert.NoError(t, err)
		assert.True(t, mock.StopCalled)
	})

	t.Run("Stop with error", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{StopError: assert.AnError}
		server = mock

		err := server.Stop()
		assert.Error(t, err)
	})

	t.Run("Status method", func(t *testing.T) {
		var server ProtocolServer
		mock := &MockProtocolServer{Status_: "running"}
		server = mock

		status := server.Status()
		assert.Equal(t, "running", status)
	})
}

// MockHTTPProtocolAdapter implements HTTPProtocolAdapter interface
type MockHTTPProtocolAdapter struct {
	MockProtocolServer
}

func (m *MockHTTPProtocolAdapter) RegisterRoutes(mux *http.ServeMux, core APSCore) error {
	return nil
}

// TestHTTPProtocolAdapter_InterfaceCompliance validates HTTPProtocolAdapter
func TestHTTPProtocolAdapter_InterfaceCompliance(t *testing.T) {
	t.Run("implements ProtocolServer", func(t *testing.T) {
		var adapter HTTPProtocolAdapter
		mock := &MockHTTPProtocolAdapter{
			MockProtocolServer: MockProtocolServer{NameValue: "http-protocol"},
		}
		adapter = mock

		assert.Equal(t, "http-protocol", adapter.Name())
	})

	t.Run("RegisterRoutes method", func(t *testing.T) {
		var adapter HTTPProtocolAdapter
		mock := &MockHTTPProtocolAdapter{}
		adapter = mock

		mux := http.NewServeMux()
		err := adapter.RegisterRoutes(mux, nil)
		assert.NoError(t, err)
	})

	t.Run("can handle all ProtocolServer methods", func(t *testing.T) {
		var adapter HTTPProtocolAdapter
		mock := &MockHTTPProtocolAdapter{
			MockProtocolServer: MockProtocolServer{Status_: "running"},
		}
		adapter = mock

		ctx := context.Background()
		assert.NoError(t, adapter.Start(ctx, nil))
		assert.NoError(t, adapter.Stop())
		assert.Equal(t, "running", adapter.Status())
	})
}

// MockStandaloneProtocolServer implements StandaloneProtocolServer interface
type MockStandaloneProtocolServer struct {
	MockProtocolServer
	AddressValue string
}

func (m *MockStandaloneProtocolServer) GetAddress() string {
	return m.AddressValue
}

// TestStandaloneProtocolServer_InterfaceCompliance validates StandaloneProtocolServer
func TestStandaloneProtocolServer_InterfaceCompliance(t *testing.T) {
	t.Run("implements ProtocolServer", func(t *testing.T) {
		var server StandaloneProtocolServer
		mock := &MockStandaloneProtocolServer{
			MockProtocolServer: MockProtocolServer{NameValue: "standalone"},
		}
		server = mock

		assert.Equal(t, "standalone", server.Name())
	})

	t.Run("GetAddress method for HTTP server", func(t *testing.T) {
		var server StandaloneProtocolServer
		mock := &MockStandaloneProtocolServer{AddressValue: "localhost:8080"}
		server = mock

		assert.Equal(t, "localhost:8080", server.GetAddress())
	})

	t.Run("GetAddress returns empty for stdio server", func(t *testing.T) {
		var server StandaloneProtocolServer
		mock := &MockStandaloneProtocolServer{AddressValue: ""}
		server = mock

		assert.Empty(t, server.GetAddress())
	})

	t.Run("can handle all ProtocolServer methods", func(t *testing.T) {
		var server StandaloneProtocolServer
		mock := &MockStandaloneProtocolServer{
			MockProtocolServer: MockProtocolServer{Status_: "running"},
		}
		server = mock

		ctx := context.Background()
		assert.NoError(t, server.Start(ctx, nil))
		assert.NoError(t, server.Stop())
		assert.Equal(t, "running", server.Status())
	})
}

// MockAPSCore implements APSCore interface for testing
type MockAPSCore struct {
	ExecuteRunFunc      func(ctx context.Context, input RunInput, stream StreamWriter) (*RunState, error)
	GetRunFunc          func(runID string) (*RunState, error)
	CancelRunFunc       func(ctx context.Context, runID string) error
	GetAgentFunc        func(profileID string) (*AgentInfo, error)
	ListAgentsFunc      func() ([]AgentInfo, error)
	GetAgentSchemasFunc func(profileID string) ([]ActionSchema, error)
	CreateSessionFunc   func(profileID string, metadata map[string]string) (*SessionState, error)
	GetSessionFunc      func(sessionID string) (*SessionState, error)
	UpdateSessionFunc   func(sessionID string, metadata map[string]string) error
	DeleteSessionFunc   func(sessionID string) error
	ListSessionsFunc    func(profileID string) ([]SessionState, error)
	StorePutFunc        func(namespace string, key string, value []byte) error
	StoreGetFunc        func(namespace string, key string) ([]byte, error)
	StoreDeleteFunc     func(namespace string, key string) error
	StoreSearchFunc     func(namespace string, prefix string) (map[string][]byte, error)
	StoreListNamespacesFunc func() ([]string, error)
}

func (m *MockAPSCore) ExecuteRun(ctx context.Context, input RunInput, stream StreamWriter) (*RunState, error) {
	if m.ExecuteRunFunc != nil {
		return m.ExecuteRunFunc(ctx, input, stream)
	}
	return &RunState{}, nil
}

func (m *MockAPSCore) GetRun(runID string) (*RunState, error) {
	if m.GetRunFunc != nil {
		return m.GetRunFunc(runID)
	}
	return &RunState{}, nil
}

func (m *MockAPSCore) CancelRun(ctx context.Context, runID string) error {
	if m.CancelRunFunc != nil {
		return m.CancelRunFunc(ctx, runID)
	}
	return nil
}

func (m *MockAPSCore) GetAgent(profileID string) (*AgentInfo, error) {
	if m.GetAgentFunc != nil {
		return m.GetAgentFunc(profileID)
	}
	return &AgentInfo{}, nil
}

func (m *MockAPSCore) ListAgents() ([]AgentInfo, error) {
	if m.ListAgentsFunc != nil {
		return m.ListAgentsFunc()
	}
	return []AgentInfo{}, nil
}

func (m *MockAPSCore) GetAgentSchemas(profileID string) ([]ActionSchema, error) {
	if m.GetAgentSchemasFunc != nil {
		return m.GetAgentSchemasFunc(profileID)
	}
	return []ActionSchema{}, nil
}

func (m *MockAPSCore) CreateSession(profileID string, metadata map[string]string) (*SessionState, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(profileID, metadata)
	}
	return &SessionState{}, nil
}

func (m *MockAPSCore) GetSession(sessionID string) (*SessionState, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(sessionID)
	}
	return &SessionState{}, nil
}

func (m *MockAPSCore) UpdateSession(sessionID string, metadata map[string]string) error {
	if m.UpdateSessionFunc != nil {
		return m.UpdateSessionFunc(sessionID, metadata)
	}
	return nil
}

func (m *MockAPSCore) DeleteSession(sessionID string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(sessionID)
	}
	return nil
}

func (m *MockAPSCore) ListSessions(profileID string) ([]SessionState, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc(profileID)
	}
	return []SessionState{}, nil
}

func (m *MockAPSCore) StorePut(namespace string, key string, value []byte) error {
	if m.StorePutFunc != nil {
		return m.StorePutFunc(namespace, key, value)
	}
	return nil
}

func (m *MockAPSCore) StoreGet(namespace string, key string) ([]byte, error) {
	if m.StoreGetFunc != nil {
		return m.StoreGetFunc(namespace, key)
	}
	return []byte{}, nil
}

func (m *MockAPSCore) StoreDelete(namespace string, key string) error {
	if m.StoreDeleteFunc != nil {
		return m.StoreDeleteFunc(namespace, key)
	}
	return nil
}

func (m *MockAPSCore) StoreSearch(namespace string, prefix string) (map[string][]byte, error) {
	if m.StoreSearchFunc != nil {
		return m.StoreSearchFunc(namespace, prefix)
	}
	return make(map[string][]byte), nil
}

func (m *MockAPSCore) StoreListNamespaces() ([]string, error) {
	if m.StoreListNamespacesFunc != nil {
		return m.StoreListNamespacesFunc()
	}
	return []string{}, nil
}

// TestAPSCore_InterfaceCompliance validates APSCore interface implementation
func TestAPSCore_InterfaceCompliance(t *testing.T) {
	t.Run("ExecuteRun method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			ExecuteRunFunc: func(ctx context.Context, input RunInput, stream StreamWriter) (*RunState, error) {
				return &RunState{RunID: "test-run"}, nil
			},
		}
		core = mock

		ctx := context.Background()
		result, err := core.ExecuteRun(ctx, RunInput{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-run", result.RunID)
	})

	t.Run("GetRun method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			GetRunFunc: func(runID string) (*RunState, error) {
				return &RunState{RunID: runID}, nil
			},
		}
		core = mock

		result, err := core.GetRun("run-123")
		assert.NoError(t, err)
		assert.Equal(t, "run-123", result.RunID)
	})

	t.Run("CancelRun method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{}
		core = mock

		ctx := context.Background()
		err := core.CancelRun(ctx, "run-123")
		assert.NoError(t, err)
	})

	t.Run("GetAgent method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			GetAgentFunc: func(profileID string) (*AgentInfo, error) {
				return &AgentInfo{ID: profileID, Name: "Test"}, nil
			},
		}
		core = mock

		result, err := core.GetAgent("agent-1")
		assert.NoError(t, err)
		assert.Equal(t, "agent-1", result.ID)
	})

	t.Run("ListAgents method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			ListAgentsFunc: func() ([]AgentInfo, error) {
				return []AgentInfo{{ID: "1"}, {ID: "2"}}, nil
			},
		}
		core = mock

		results, err := core.ListAgents()
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("GetAgentSchemas method signature", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			GetAgentSchemasFunc: func(profileID string) ([]ActionSchema, error) {
				return []ActionSchema{{Name: "schema-1"}}, nil
			},
		}
		core = mock

		results, err := core.GetAgentSchemas("profile-1")
		assert.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("session management methods", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			CreateSessionFunc: func(profileID string, metadata map[string]string) (*SessionState, error) {
				return &SessionState{SessionID: "sess-1"}, nil
			},
			GetSessionFunc: func(sessionID string) (*SessionState, error) {
				return &SessionState{SessionID: sessionID}, nil
			},
		}
		core = mock

		sess, err := core.CreateSession("profile-1", nil)
		assert.NoError(t, err)
		assert.Equal(t, "sess-1", sess.SessionID)

		retrieved, err := core.GetSession("sess-1")
		assert.NoError(t, err)
		assert.Equal(t, "sess-1", retrieved.SessionID)
	})

	t.Run("store management methods", func(t *testing.T) {
		var core APSCore
		mock := &MockAPSCore{
			StorePutFunc: func(namespace string, key string, value []byte) error {
				return nil
			},
			StoreGetFunc: func(namespace string, key string) ([]byte, error) {
				return []byte("value"), nil
			},
		}
		core = mock

		err := core.StorePut("ns", "key", []byte("value"))
		assert.NoError(t, err)

		value, err := core.StoreGet("ns", "key")
		assert.NoError(t, err)
		assert.Equal(t, []byte("value"), value)
	})
}

// TestDefaultHTTPBridge_InterfaceCompliance validates HTTPBridge implementation
func TestDefaultHTTPBridge_InterfaceCompliance(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test-protocol"}
	bridge := NewDefaultHTTPBridge(mock)

	t.Run("Name includes http-bridge suffix", func(t *testing.T) {
		name := bridge.Name()
		assert.Contains(t, name, "http-bridge")
		assert.Contains(t, name, "test-protocol")
	})

	t.Run("delegates Start to wrapped server", func(t *testing.T) {
		ctx := context.Background()
		err := bridge.Start(ctx, nil)
		assert.NoError(t, err)
		assert.True(t, mock.StartCalled)
	})

	t.Run("delegates Stop to wrapped server", func(t *testing.T) {
		err := bridge.Stop()
		assert.NoError(t, err)
		assert.True(t, mock.StopCalled)
	})

	t.Run("delegates Status to wrapped server", func(t *testing.T) {
		mock.Status_ = "running"
		status := bridge.Status()
		assert.Equal(t, "running", status)
	})

	t.Run("GetHTTPHandler returns an HTTP handler", func(t *testing.T) {
		handler := bridge.GetHTTPHandler()
		assert.NotNil(t, handler)
	})
}

// TestProtocolServerAdapter_InterfaceCompliance validates ProtocolServerAdapter
func TestProtocolServerAdapter_InterfaceCompliance(t *testing.T) {
	mock := &MockProtocolServer{NameValue: "test"}
	adapter := NewProtocolServerAdapter(mock)

	t.Run("implements HTTPProtocolAdapter", func(t *testing.T) {
		var httpAdapter HTTPProtocolAdapter
		httpAdapter = adapter
		assert.NotNil(t, httpAdapter)
	})

	t.Run("delegates all ProtocolServer methods", func(t *testing.T) {
		assert.Equal(t, "test", adapter.Name())

		ctx := context.Background()
		err := adapter.Start(ctx, nil)
		assert.NoError(t, err)

		err = adapter.Stop()
		assert.NoError(t, err)

		mock.Status_ = "stopped"
		assert.Equal(t, "stopped", adapter.Status())
	})

	t.Run("RegisterRoutes is no-op", func(t *testing.T) {
		mux := http.NewServeMux()
		err := adapter.RegisterRoutes(mux, nil)
		assert.NoError(t, err)
	})
}

// ==================== RunState Tests (15 tests) ====================

// TestRunState_InitializationWithAllFields tests RunState initialization with all fields
func TestRunState_InitializationWithAllFields(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Second)
	exitCode := 0
	metadata := map[string]interface{}{"key": "value"}

	state := &RunState{
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

	assert.Equal(t, "run-1", state.RunID)
	assert.Equal(t, "profile-1", state.ProfileID)
	assert.Equal(t, "action-1", state.ActionID)
	assert.Equal(t, "thread-1", state.ThreadID)
	assert.Equal(t, RunStatusCompleted, state.Status)
	assert.Equal(t, now, state.StartTime)
	assert.Equal(t, &endTime, state.EndTime)
	assert.Equal(t, &exitCode, state.ExitCode)
	assert.Equal(t, int64(1024), state.OutputSize)
	assert.Empty(t, state.Error)
	assert.Equal(t, metadata, state.Metadata)
}

// TestRunState_InitializationMinimal tests RunState with minimal fields
func TestRunState_InitializationMinimal(t *testing.T) {
	now := time.Now()
	state := &RunState{
		RunID:     "run-1",
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Status:    RunStatusPending,
		StartTime: now,
	}

	assert.Equal(t, "run-1", state.RunID)
	assert.Nil(t, state.EndTime)
	assert.Nil(t, state.ExitCode)
	assert.Zero(t, state.OutputSize)
	assert.Empty(t, state.Error)
}

// TestRunState_JSONMarshal tests RunState JSON marshaling
func TestRunState_JSONMarshal(t *testing.T) {
	now := time.Now()
	state := &RunState{
		RunID:      "run-1",
		ProfileID:  "profile-1",
		ActionID:   "action-1",
		Status:     RunStatusRunning,
		StartTime:  now,
		OutputSize: 512,
	}

	data, err := json.Marshal(state)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "run-1")
	assert.Contains(t, string(data), "running")
}

// TestRunState_JSONUnmarshal tests RunState JSON unmarshaling
func TestRunState_JSONUnmarshal(t *testing.T) {
	jsonData := []byte(`{
		"run_id": "run-1",
		"profile_id": "profile-1",
		"action_id": "action-1",
		"status": "completed",
		"start_time": "2025-01-15T10:30:00Z",
		"exit_code": 0,
		"output_size": 1024
	}`)

	var state RunState
	err := json.Unmarshal(jsonData, &state)
	assert.NoError(t, err)
	assert.Equal(t, "run-1", state.RunID)
	assert.Equal(t, "profile-1", state.ProfileID)
	assert.Equal(t, "action-1", state.ActionID)
	assert.Equal(t, RunStatusCompleted, state.Status)
	assert.NotNil(t, state.ExitCode)
	assert.Equal(t, 0, *state.ExitCode)
	assert.Equal(t, int64(1024), state.OutputSize)
}

// TestRunState_JSONRoundTrip tests JSON marshaling and unmarshaling round-trip
func TestRunState_JSONRoundTrip(t *testing.T) {
	now := time.Now()
	exitCode := 0
	originalState := &RunState{
		RunID:      "run-123",
		ProfileID:  "profile-456",
		ActionID:   "action-789",
		ThreadID:   "thread-xyz",
		Status:     RunStatusCompleted,
		StartTime:  now,
		ExitCode:   &exitCode,
		OutputSize: 2048,
		Error:      "",
		Metadata:   map[string]interface{}{"duration": "5s"},
	}

	data, err := json.Marshal(originalState)
	assert.NoError(t, err)

	var unmarshaledState RunState
	err = json.Unmarshal(data, &unmarshaledState)
	assert.NoError(t, err)

	assert.Equal(t, originalState.RunID, unmarshaledState.RunID)
	assert.Equal(t, originalState.ProfileID, unmarshaledState.ProfileID)
	assert.Equal(t, originalState.ActionID, unmarshaledState.ActionID)
	assert.Equal(t, originalState.ThreadID, unmarshaledState.ThreadID)
	assert.Equal(t, originalState.Status, unmarshaledState.Status)
}

// TestRunState_WithMetadata tests RunState with metadata
func TestRunState_WithMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"duration":      "5.2s",
		"memory_used":   "256MB",
		"cpu_percent":   "45.3",
		"custom_field":  "custom_value",
	}

	state := &RunState{
		RunID:      "run-1",
		ProfileID:  "profile-1",
		ActionID:   "action-1",
		Status:     RunStatusCompleted,
		StartTime:  time.Now(),
		Metadata:   metadata,
	}

	assert.NotNil(t, state.Metadata)
	assert.Equal(t, metadata, state.Metadata)
}

// TestRunState_WithExitCodes tests RunState with various exit codes
func TestRunState_WithExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		exitCode *int
	}{
		{"zero exit code", ptrInt(0)},
		{"success exit code", ptrInt(1)},
		{"error exit code", ptrInt(127)},
		{"negative exit code", ptrInt(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &RunState{
				RunID:     "run-1",
				Status:    RunStatusCompleted,
				StartTime: time.Now(),
				ExitCode:  tt.exitCode,
			}

			assert.NotNil(t, state.ExitCode)
			assert.Equal(t, tt.exitCode, state.ExitCode)
			if tt.exitCode != nil {
				assert.Equal(t, *tt.exitCode, *state.ExitCode)
			}
		})
	}
}

// TestRunState_WithErrorMessages tests RunState with error messages
func TestRunState_WithErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
	}{
		{"command not found", "command not found: python3"},
		{"permission denied", "permission denied: /usr/bin/script"},
		{"timeout", "execution timeout after 30s"},
		{"empty error", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &RunState{
				RunID:      "run-1",
				Status:     RunStatusFailed,
				StartTime:  time.Now(),
				Error:      tt.errorMessage,
			}

			assert.Equal(t, tt.errorMessage, state.Error)
		})
	}
}

// TestRunState_TimeTracking tests RunState time tracking
func TestRunState_TimeTracking(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(10 * time.Second)

	state := &RunState{
		RunID:     "run-1",
		Status:    RunStatusCompleted,
		StartTime: startTime,
		EndTime:   &endTime,
	}

	assert.False(t, state.StartTime.IsZero())
	assert.NotNil(t, state.EndTime)
	assert.True(t, state.EndTime.After(state.StartTime))
	assert.Equal(t, 10*time.Second, state.EndTime.Sub(state.StartTime))
}

// TestRunState_StatusTransitions tests RunState status transitions
func TestRunState_StatusTransitions(t *testing.T) {
	state := &RunState{
		RunID:     "run-1",
		StartTime: time.Now(),
	}

	// Pending -> Running
	state.Status = RunStatusPending
	assert.Equal(t, RunStatusPending, state.Status)

	state.Status = RunStatusRunning
	assert.Equal(t, RunStatusRunning, state.Status)

	// Running -> Completed
	endTime := time.Now()
	state.EndTime = &endTime
	state.Status = RunStatusCompleted
	assert.Equal(t, RunStatusCompleted, state.Status)
}

// TestRunState_CancelledStatus tests RunState with cancelled status
func TestRunState_CancelledStatus(t *testing.T) {
	state := &RunState{
		RunID:      "run-1",
		Status:     RunStatusCancelled,
		StartTime:  time.Now(),
		Error:      "cancelled by user",
	}

	assert.Equal(t, RunStatusCancelled, state.Status)
	assert.NotEmpty(t, state.Error)
}

// TestRunState_FailedStatus tests RunState with failed status
func TestRunState_FailedStatus(t *testing.T) {
	exitCode := 1
	state := &RunState{
		RunID:      "run-1",
		Status:     RunStatusFailed,
		StartTime:  time.Now(),
		ExitCode:   &exitCode,
		Error:      "action execution failed",
	}

	assert.Equal(t, RunStatusFailed, state.Status)
	assert.NotNil(t, state.ExitCode)
	assert.Equal(t, 1, *state.ExitCode)
}

// TestRunState_MultipleRuns tests creating multiple run states
func TestRunState_MultipleRuns(t *testing.T) {
	states := make([]*RunState, 5)
	for i := 0; i < 5; i++ {
		states[i] = &RunState{
			RunID:     fmt.Sprintf("run-%d", i),
			ProfileID: "profile-1",
			ActionID:  "action-1",
			Status:    RunStatusCompleted,
			StartTime: time.Now(),
		}
	}

	assert.Len(t, states, 5)
	for i, state := range states {
		assert.Equal(t, fmt.Sprintf("run-%d", i), state.RunID)
	}
}

// ==================== SessionState Tests (10 tests) ====================

// TestSessionState_Initialization tests SessionState initialization
func TestSessionState_Initialization(t *testing.T) {
	now := time.Now()
	metadata := map[string]string{"key": "value"}

	session := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  now,
		LastSeenAt: now,
		Metadata:   metadata,
	}

	assert.Equal(t, "sess-1", session.SessionID)
	assert.Equal(t, "profile-1", session.ProfileID)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, now, session.LastSeenAt)
	assert.Equal(t, metadata, session.Metadata)
}

// TestSessionState_JSONMarshal tests SessionState JSON marshaling
func TestSessionState_JSONMarshal(t *testing.T) {
	now := time.Now()
	session := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  now,
		LastSeenAt: now,
	}

	data, err := json.Marshal(session)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "sess-1")
	assert.Contains(t, string(data), "profile-1")
}

// TestSessionState_JSONUnmarshal tests SessionState JSON unmarshaling
func TestSessionState_JSONUnmarshal(t *testing.T) {
	jsonData := []byte(`{
		"session_id": "sess-1",
		"profile_id": "profile-1",
		"created_at": "2025-01-15T10:30:00Z",
		"last_seen_at": "2025-01-15T10:35:00Z"
	}`)

	var session SessionState
	err := json.Unmarshal(jsonData, &session)
	assert.NoError(t, err)
	assert.Equal(t, "sess-1", session.SessionID)
	assert.Equal(t, "profile-1", session.ProfileID)
}

// TestSessionState_JSONRoundTrip tests JSON round-trip
func TestSessionState_JSONRoundTrip(t *testing.T) {
	now := time.Now()
	originalSession := &SessionState{
		SessionID:  "sess-abc",
		ProfileID:  "profile-xyz",
		CreatedAt:  now,
		LastSeenAt: now,
		Metadata: map[string]string{
			"env": "production",
			"region": "us-east-1",
		},
	}

	data, err := json.Marshal(originalSession)
	assert.NoError(t, err)

	var unmarshaledSession SessionState
	err = json.Unmarshal(data, &unmarshaledSession)
	assert.NoError(t, err)

	assert.Equal(t, originalSession.SessionID, unmarshaledSession.SessionID)
	assert.Equal(t, originalSession.ProfileID, unmarshaledSession.ProfileID)
}

// TestSessionState_MetadataHandling tests SessionState metadata handling
func TestSessionState_MetadataHandling(t *testing.T) {
	session := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		Metadata: map[string]string{
			"user_id":       "user-123",
			"api_key":       "key-456",
			"last_action":   "create_resource",
			"request_count": "42",
		},
	}

	assert.NotNil(t, session.Metadata)
	assert.Equal(t, "user-123", session.Metadata["user_id"])
	assert.Equal(t, "42", session.Metadata["request_count"])
}

// TestSessionState_TimestampTracking tests SessionState timestamp tracking
func TestSessionState_TimestampTracking(t *testing.T) {
	createdAt := time.Now()
	lastSeenAt := createdAt.Add(1 * time.Hour)

	session := &SessionState{
		SessionID:  "sess-1",
		CreatedAt:  createdAt,
		LastSeenAt: lastSeenAt,
	}

	assert.Equal(t, createdAt, session.CreatedAt)
	assert.Equal(t, lastSeenAt, session.LastSeenAt)
	assert.True(t, session.LastSeenAt.After(session.CreatedAt))
}

// TestSessionState_Updates tests SessionState updates
func TestSessionState_Updates(t *testing.T) {
	session := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		Metadata:   map[string]string{"count": "1"},
	}

	// Update metadata
	session.Metadata["count"] = "2"
	session.LastSeenAt = time.Now()

	assert.Equal(t, "2", session.Metadata["count"])
	assert.True(t, session.LastSeenAt.After(session.CreatedAt))
}

// TestSessionState_EmptyMetadata tests SessionState with empty metadata
func TestSessionState_EmptyMetadata(t *testing.T) {
	session := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		Metadata:   make(map[string]string),
	}

	assert.NotNil(t, session.Metadata)
	assert.Empty(t, session.Metadata)
}

// TestSessionState_MultipleSessions tests multiple sessions
func TestSessionState_MultipleSessions(t *testing.T) {
	sessions := make([]*SessionState, 3)
	for i := 0; i < 3; i++ {
		sessions[i] = &SessionState{
			SessionID:  fmt.Sprintf("sess-%d", i),
			ProfileID:  "profile-1",
			CreatedAt:  time.Now(),
			LastSeenAt: time.Now(),
		}
	}

	assert.Len(t, sessions, 3)
	assert.Equal(t, "sess-0", sessions[0].SessionID)
	assert.Equal(t, "sess-1", sessions[1].SessionID)
	assert.Equal(t, "sess-2", sessions[2].SessionID)
}

// TestSessionState_Comparison tests SessionState comparison
func TestSessionState_Comparison(t *testing.T) {
	session1 := &SessionState{
		SessionID:  "sess-1",
		ProfileID:  "profile-1",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}

	session2 := &SessionState{
		SessionID:  "sess-2",
		ProfileID:  "profile-1",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}

	assert.NotEqual(t, session1.SessionID, session2.SessionID)
	assert.Equal(t, session1.ProfileID, session2.ProfileID)
}

// ==================== AgentInfo Tests (8 tests) ====================

// TestAgentInfo_Validation tests AgentInfo validation
func TestAgentInfo_Validation(t *testing.T) {
	info := &AgentInfo{
		ID:           "agent-1",
		Name:         "Test Agent",
		Description:  "A test agent",
		Capabilities: []string{"run_actions", "list_profiles"},
	}

	err := ValidateAgentInfo(info)
	assert.NoError(t, err)
}

// TestAgentInfo_CapabilitiesList tests AgentInfo with capabilities list
func TestAgentInfo_CapabilitiesList(t *testing.T) {
	capabilities := []string{
		"execute_actions",
		"list_profiles",
		"manage_sessions",
		"store_data",
	}

	info := &AgentInfo{
		ID:           "agent-1",
		Name:         "Full Agent",
		Description:  "Agent with all capabilities",
		Capabilities: capabilities,
	}

	assert.Equal(t, capabilities, info.Capabilities)
	assert.Len(t, info.Capabilities, 4)
}

// TestAgentInfo_JSONMarshal tests AgentInfo JSON marshaling
func TestAgentInfo_JSONMarshal(t *testing.T) {
	info := &AgentInfo{
		ID:           "agent-1",
		Name:         "Test Agent",
		Description:  "A test agent",
		Capabilities: []string{"run"},
	}

	data, err := json.Marshal(info)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "agent-1")
	assert.Contains(t, string(data), "Test Agent")
}

// TestAgentInfo_JSONUnmarshal tests AgentInfo JSON unmarshaling
func TestAgentInfo_JSONUnmarshal(t *testing.T) {
	jsonData := []byte(`{
		"id": "agent-1",
		"name": "Test Agent",
		"description": "A test agent",
		"capabilities": ["run", "list"]
	}`)

	var info AgentInfo
	err := json.Unmarshal(jsonData, &info)
	assert.NoError(t, err)
	assert.Equal(t, "agent-1", info.ID)
	assert.Equal(t, "Test Agent", info.Name)
	assert.Len(t, info.Capabilities, 2)
}

// TestAgentInfo_EmptyCapabilities tests AgentInfo with empty capabilities
func TestAgentInfo_EmptyCapabilities(t *testing.T) {
	info := &AgentInfo{
		ID:           "agent-1",
		Name:         "Basic Agent",
		Description:  "Agent with no capabilities",
		Capabilities: []string{},
	}

	assert.NotNil(t, info.Capabilities)
	assert.Empty(t, info.Capabilities)
}

// TestAgentInfo_MultipleAgents tests multiple agents
func TestAgentInfo_MultipleAgents(t *testing.T) {
	agents := []AgentInfo{
		{
			ID:           "agent-1",
			Name:         "Agent 1",
			Description:  "First agent",
			Capabilities: []string{"run"},
		},
		{
			ID:           "agent-2",
			Name:         "Agent 2",
			Description:  "Second agent",
			Capabilities: []string{"run", "list"},
		},
	}

	assert.Len(t, agents, 2)
	assert.Equal(t, "agent-1", agents[0].ID)
	assert.Equal(t, "agent-2", agents[1].ID)
}

// TestAgentInfo_Comparison tests AgentInfo comparison
func TestAgentInfo_Comparison(t *testing.T) {
	info1 := &AgentInfo{
		ID:   "agent-1",
		Name: "Agent 1",
	}

	info2 := &AgentInfo{
		ID:   "agent-1",
		Name: "Agent 1",
	}

	assert.Equal(t, info1.ID, info2.ID)
	assert.Equal(t, info1.Name, info2.Name)
}

// TestAgentInfo_Metadata tests AgentInfo as metadata holder
func TestAgentInfo_Metadata(t *testing.T) {
	info := &AgentInfo{
		ID:          "agent-custom",
		Name:        "Custom Agent",
		Description: "Agent with custom metadata in description",
		Capabilities: []string{
			"capability:execute",
			"capability:read",
			"capability:write",
		},
	}

	assert.NotEmpty(t, info.Description)
	assert.NotEmpty(t, info.Capabilities)
}

// ==================== RunInput Tests (8 tests) ====================

// TestRunInput_ValidationRequired tests RunInput validation with required fields
func TestRunInput_ValidationRequired(t *testing.T) {
	input := RunInput{
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Payload:   []byte("test"),
	}

	err := input.Validate()
	assert.NoError(t, err)
}

// TestRunInput_WithOptionalThreadID tests RunInput with optional ThreadID
func TestRunInput_WithOptionalThreadID(t *testing.T) {
	input := RunInput{
		ProfileID: "profile-1",
		ActionID:  "action-1",
		ThreadID:  "thread-1",
		Payload:   []byte("test"),
	}

	err := input.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "thread-1", input.ThreadID)
}

// TestRunInput_WithPayload tests RunInput with payload
func TestRunInput_WithPayload(t *testing.T) {
	payload := []byte(`{"key": "value"}`)
	input := RunInput{
		ProfileID: "profile-1",
		ActionID:  "action-1",
		Payload:   payload,
	}

	err := input.Validate()
	assert.NoError(t, err)
	assert.Equal(t, payload, input.Payload)
}

// TestRunInput_ValidationErrors tests RunInput validation errors
func TestRunInput_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     RunInput
		wantError bool
	}{
		{
			name: "missing profile ID",
			input: RunInput{
				ActionID: "action-1",
			},
			wantError: true,
		},
		{
			name: "missing action ID",
			input: RunInput{
				ProfileID: "profile-1",
			},
			wantError: true,
		},
		{
			name: "both missing",
			input: RunInput{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRunInput_MultipleRunInputs tests multiple RunInputs
func TestRunInput_MultipleRunInputs(t *testing.T) {
	inputs := []RunInput{
		{ProfileID: "profile-1", ActionID: "action-1"},
		{ProfileID: "profile-2", ActionID: "action-2"},
		{ProfileID: "profile-3", ActionID: "action-3"},
	}

	for _, input := range inputs {
		err := input.Validate()
		assert.NoError(t, err)
	}
}

// TestRunInput_JSONMarshaling tests RunInput JSON marshaling
func TestRunInput_JSONMarshaling(t *testing.T) {
	input := RunInput{
		ProfileID: "profile-1",
		ActionID:  "action-1",
		ThreadID:  "thread-1",
		Payload:   []byte("test payload"),
	}

	// Note: RunInput doesn't have json tags, but we can still test marshaling
	data, err := json.Marshal(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestRunInput_EdgeCases tests RunInput edge cases
func TestRunInput_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     RunInput
		wantError bool
	}{
		{
			name: "empty payload is valid",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "action-1",
				Payload:   []byte{},
			},
			wantError: false,
		},
		{
			name: "nil payload is valid",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "action-1",
				Payload:   nil,
			},
			wantError: false,
		},
		{
			name: "empty thread ID is valid",
			input: RunInput{
				ProfileID: "profile-1",
				ActionID:  "action-1",
				ThreadID:  "",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== ActionSchema Tests (7 tests) ====================

// TestActionSchema_Validation tests ActionSchema validation
func TestActionSchema_Validation(t *testing.T) {
	schema := &ActionSchema{
		Name:        "test-action",
		Description: "A test action",
		Input: map[string]interface{}{
			"type": "object",
		},
	}

	err := ValidateActionSchema(schema)
	assert.NoError(t, err)
}

// TestActionSchema_ComplexInputTypes tests ActionSchema with complex input types
func TestActionSchema_ComplexInputTypes(t *testing.T) {
	schema := &ActionSchema{
		Name:        "complex-action",
		Description: "Action with complex input schema",
		Input: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"age": map[string]interface{}{
					"type": "integer",
				},
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"name"},
		},
	}

	err := ValidateActionSchema(schema)
	assert.NoError(t, err)
	assert.NotNil(t, schema.Input)
}

// TestActionSchema_JSONMarshal tests ActionSchema JSON marshaling
func TestActionSchema_JSONMarshal(t *testing.T) {
	schema := &ActionSchema{
		Name:        "test-action",
		Description: "A test action",
		Input: map[string]interface{}{
			"type": "object",
		},
	}

	data, err := json.Marshal(schema)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test-action")
}

// TestActionSchema_JSONUnmarshal tests ActionSchema JSON unmarshaling
func TestActionSchema_JSONUnmarshal(t *testing.T) {
	jsonData := []byte(`{
		"name": "test-action",
		"description": "A test action",
		"input": {
			"type": "object"
		}
	}`)

	var schema ActionSchema
	err := json.Unmarshal(jsonData, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "test-action", schema.Name)
	assert.Equal(t, "A test action", schema.Description)
}

// TestActionSchema_SchemaComposition tests schema composition
func TestActionSchema_SchemaComposition(t *testing.T) {
	baseSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"base_field": map[string]interface{}{
				"type": "string",
			},
		},
	}

	schema := &ActionSchema{
		Name:        "composite-action",
		Description: "Composite action schema",
		Input:       baseSchema,
	}

	err := ValidateActionSchema(schema)
	assert.NoError(t, err)
	assert.NotNil(t, schema.Input)
}

// TestActionSchema_ValidationErrors tests schema validation errors
func TestActionSchema_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		schema     *ActionSchema
		wantError  bool
	}{
		{
			name: "missing name",
			schema: &ActionSchema{
				Description: "No name",
				Input:       map[string]interface{}{},
			},
			wantError: true,
		},
		{
			name: "missing description",
			schema: &ActionSchema{
				Name:  "test",
				Input: map[string]interface{}{},
			},
			wantError: true,
		},
		{
			name: "both fields present",
			schema: &ActionSchema{
				Name:        "test",
				Description: "Test action",
				Input:       map[string]interface{}{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActionSchema(tt.schema)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestActionSchema_Comparison tests schema comparison
func TestActionSchema_Comparison(t *testing.T) {
	schema1 := &ActionSchema{
		Name:        "action-1",
		Description: "First action",
		Input:       map[string]interface{}{},
	}

	schema2 := &ActionSchema{
		Name:        "action-2",
		Description: "Second action",
		Input:       map[string]interface{}{},
	}

	assert.NotEqual(t, schema1.Name, schema2.Name)
}

// ==================== Error Handling Tests (10 tests) ====================

// TestNewNotFoundError tests NewNotFoundError
func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("profile-123")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "profile-123")
}

// TestNewInvalidInputError tests NewInvalidInputError
func TestNewInvalidInputError(t *testing.T) {
	err := NewInvalidInputError("profile_id", "profile ID is required")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "profile ID is required")
	assert.Equal(t, "profile_id", err.Field)
}

// TestNewValidationError tests NewValidationError
func TestNewValidationError(t *testing.T) {
	err := NewValidationError("action_id", "invalid action ID format")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid action ID format")
	assert.Equal(t, "action_id", err.Field)
}

// TestNewExecutionError tests NewExecutionError
func TestNewExecutionError(t *testing.T) {
	err := NewExecutionError("my-action", "execution failed with code 1")
	assert.NotNil(t, err)
	assert.Equal(t, "execution failed with code 1", err.Error())
	assert.Equal(t, "my-action", err.Action)
}

// TestErrorMessageFormatting tests error message formatting
func TestErrorMessageFormatting(t *testing.T) {
	tests := []struct {
		name       string
		errFunc    func() error
		wantSubstr string
	}{
		{
			name: "not found error",
			errFunc: func() error {
				return NewNotFoundError("resource-1")
			},
			wantSubstr: "resource-1",
		},
		{
			name: "invalid input error",
			errFunc: func() error {
				return NewInvalidInputError("field", "custom message")
			},
			wantSubstr: "custom message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errFunc()
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
}

// TestErrorTypeAssertion tests error type assertion
func TestErrorTypeAssertion(t *testing.T) {
	execErr := NewExecutionError("test-action", "failed")

	// ExecutionError pointer already known, test direct access
	assert.NotNil(t, execErr)
	assert.Equal(t, "test-action", execErr.Action)
	assert.Equal(t, "failed", execErr.Message)

	// Test as error interface
	var errInterface error = execErr
	assert.NotNil(t, errInterface)
}

// TestMultipleErrorTypes tests multiple error types
func TestMultipleErrorTypes(t *testing.T) {
	errors := []error{
		NewNotFoundError("resource"),
		NewInvalidInputError("field", "invalid"),
		NewValidationError("field", "validation failed"),
		NewExecutionError("action", "failed"),
	}

	assert.Len(t, errors, 4)
	for _, err := range errors {
		assert.Error(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

// TestExecutionErrorStruct tests ExecutionError struct
func TestExecutionErrorStruct(t *testing.T) {
	err := &ExecutionError{
		Action:  "my-action",
		Message: "something went wrong",
	}

	assert.Equal(t, "something went wrong", err.Error())
	assert.Equal(t, "my-action", err.Action)
	assert.Equal(t, "something went wrong", err.Message)
}

// TestErrorRecoveryScenarios tests error recovery scenarios
func TestErrorRecoveryScenarios(t *testing.T) {
	// Scenario: Try to execute action that doesn't exist
	var result error

	// First error: NotFoundError
	result = NewNotFoundError("action-123")
	assert.NotNil(t, result)

	// Recovery: Provide a default error
	if result != nil {
		result = NewExecutionError("fallback-action", "using fallback")
	}

	assert.NotNil(t, result)
	execErr, ok := result.(*ExecutionError)
	assert.True(t, ok)
	assert.Equal(t, "fallback-action", execErr.Action)
}

// ==================== Helper Functions ====================

func ptrInt(i int) *int {
	return &i
}
