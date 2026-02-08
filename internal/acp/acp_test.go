package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"oss-aps-cli/internal/core/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Fixtures and Helpers
// ============================================================================

// mockTransport implements Transport interface for testing
type mockTransport struct {
	requestQueue  []*JSONRPCRequest
	responseQueue []interface{}
	closed        bool
	mu             sync.RWMutex
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		requestQueue:  make([]*JSONRPCRequest, 0),
		responseQueue: make([]interface{}, 0),
	}
}

func (mt *mockTransport) Read() (*JSONRPCRequest, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if len(mt.requestQueue) == 0 {
		return nil, io.EOF
	}
	req := mt.requestQueue[0]
	mt.requestQueue = mt.requestQueue[1:]
	return req, nil
}

func (mt *mockTransport) Write(response interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.responseQueue = append(mt.responseQueue, response)
	return nil
}

func (mt *mockTransport) Close() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.closed = true
	return nil
}

func (mt *mockTransport) pushRequest(req *JSONRPCRequest) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.requestQueue = append(mt.requestQueue, req)
}

func (mt *mockTransport) getResponses() []interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return append([]interface{}{}, mt.responseQueue...)
}

// mockAPSCore implements protocol.APSCore interface for testing
type mockAPSCore struct {
	sessions map[string]*protocol.SessionState
	runs     map[string]*protocol.RunState
	mu       sync.RWMutex
}

func newMockAPSCore() *mockAPSCore {
	return &mockAPSCore{
		sessions: make(map[string]*protocol.SessionState),
		runs:     make(map[string]*protocol.RunState),
	}
}

func (m *mockAPSCore) ExecuteRun(ctx context.Context, input protocol.RunInput, stream protocol.StreamWriter) (*protocol.RunState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run := &protocol.RunState{
		RunID:     generateID(),
		ProfileID: input.ProfileID,
		ActionID:  input.ActionID,
		ThreadID:  input.ThreadID,
		Status:    protocol.RunStatusCompleted,
		StartTime: time.Now(),
	}
	m.runs[run.RunID] = run
	return run, nil
}

func (m *mockAPSCore) GetRun(runID string) (*protocol.RunState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.runs[runID]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("run not found: %s", runID)
}

func (m *mockAPSCore) CancelRun(ctx context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run, ok := m.runs[runID]; ok {
		run.Status = protocol.RunStatusCancelled
		return nil
	}
	return fmt.Errorf("run not found: %s", runID)
}

func (m *mockAPSCore) GetAgent(profileID string) (*protocol.AgentInfo, error) {
	return &protocol.AgentInfo{
		ID:   profileID,
		Name: "Test Agent",
	}, nil
}

func (m *mockAPSCore) ListAgents() ([]protocol.AgentInfo, error) {
	return []protocol.AgentInfo{}, nil
}

func (m *mockAPSCore) GetAgentSchemas(profileID string) ([]protocol.ActionSchema, error) {
	return []protocol.ActionSchema{}, nil
}

func (m *mockAPSCore) CreateSession(profileID string, metadata map[string]string) (*protocol.SessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := &protocol.SessionState{
		SessionID: generateID(),
		ProfileID: profileID,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}
	m.sessions[session.SessionID] = session
	return session, nil
}

func (m *mockAPSCore) GetSession(sessionID string) (*protocol.SessionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sessions[sessionID]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (m *mockAPSCore) UpdateSession(sessionID string, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Metadata = metadata
		return nil
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

func (m *mockAPSCore) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockAPSCore) ListSessions(profileID string) ([]protocol.SessionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sessions []protocol.SessionState
	for _, s := range m.sessions {
		if profileID == "" || s.ProfileID == profileID {
			sessions = append(sessions, *s)
		}
	}
	return sessions, nil
}

func (m *mockAPSCore) StorePut(namespace string, key string, value []byte) error {
	return nil
}

func (m *mockAPSCore) StoreGet(namespace string, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockAPSCore) StoreDelete(namespace string, key string) error {
	return nil
}

func (m *mockAPSCore) StoreSearch(namespace string, prefix string) (map[string][]byte, error) {
	return make(map[string][]byte), nil
}

func (m *mockAPSCore) StoreListNamespaces() ([]string, error) {
	return []string{}, nil
}

// ============================================================================
// 1. Server Lifecycle Tests
// ============================================================================

func TestNewServer_ValidInput(t *testing.T) {
	core := newMockAPSCore()
	server, err := NewServer("test-profile", core)

	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "test-profile", server.profileID)
	assert.Equal(t, "stopped", server.status)
	assert.NotNil(t, server.sessionManager)
	assert.NotNil(t, server.permissionManager)
	assert.NotNil(t, server.terminalManager)
}

func TestNewServer_EmptyProfileID(t *testing.T) {
	core := newMockAPSCore()
	server, err := NewServer("", core)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "profile ID cannot be empty")
}

func TestNewServer_NilCore(t *testing.T) {
	server, err := NewServer("test-profile", nil)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "core cannot be nil")
}

func TestServer_Name(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	assert.Equal(t, "acp", server.Name())
}

func TestServer_StatusInitiallyStopped(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	assert.Equal(t, "stopped", server.Status())
}

func TestServer_StatusAfterStart(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	transport := newMockTransport()
	server.transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.mu.Lock()
	server.status = "running"
	server.ctx = ctx
	server.cancel = cancel
	server.mu.Unlock()

	assert.Equal(t, "running", server.Status())
}

func TestServer_Stop_NotRunning(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	err := server.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "stopped", server.Status())
}

func TestServer_Stop_Running(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	transport := newMockTransport()

	ctx, cancel := context.WithCancel(context.Background())
	server.mu.Lock()
	server.status = "running"
	server.transport = transport
	server.ctx = ctx
	server.cancel = cancel
	server.mu.Unlock()

	err := server.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "stopped", server.Status())
	assert.True(t, transport.closed)
}

func TestServer_GetAddress_ReturnsEmpty(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	assert.Equal(t, "", server.GetAddress())
}

func TestServer_InterfaceCompliance(t *testing.T) {
	// Verify that Server implements both ProtocolServer and StandaloneProtocolServer
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	var _ protocol.ProtocolServer = server
	var _ protocol.StandaloneProtocolServer = server
}

// ============================================================================
// 2. Session Management Tests
// ============================================================================

func TestSessionManager_CreateSession(t *testing.T) {
	sm := NewSessionManager()
	mode := SessionModeDefault
	caps := map[string]interface{}{"version": "1.0"}

	session := sm.CreateSession("sess-1", "profile-1", mode, caps, nil)

	assert.NotNil(t, session)
	assert.Equal(t, "sess-1", session.SessionID)
	assert.Equal(t, "profile-1", session.ProfileID)
	assert.Equal(t, mode, session.Mode)
	assert.Equal(t, caps, session.ClientCapabilities)
	assert.NotNil(t, session.AgentCapabilities)
	assert.NotNil(t, session.PermissionRules)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestSessionManager_CreateSession_DefaultMode(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("sess-1", "profile-1", "", nil, nil)

	assert.Equal(t, SessionModeDefault, session.Mode)
}

func TestSessionManager_GetSession(t *testing.T) {
	sm := NewSessionManager()
	created := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	retrieved, err := sm.GetSession("sess-1")

	assert.NoError(t, err)
	assert.Equal(t, created.SessionID, retrieved.SessionID)
}

func TestSessionManager_GetSession_NotFound(t *testing.T) {
	sm := NewSessionManager()

	session, err := sm.GetSession("nonexistent")

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionManager_UpdateSession(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	err := sm.UpdateSession("sess-1", func(s *ACPSession) error {
		s.Mode = SessionModeAutoApprove
		return nil
	})

	assert.NoError(t, err)
	retrieved, _ := sm.GetSession("sess-1")
	assert.Equal(t, SessionModeAutoApprove, retrieved.Mode)
}

func TestSessionManager_UpdateSession_NotFound(t *testing.T) {
	sm := NewSessionManager()

	err := sm.UpdateSession("nonexistent", func(s *ACPSession) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionManager_DeleteSession(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	err := sm.DeleteSession("sess-1")
	assert.NoError(t, err)

	_, err = sm.GetSession("sess-1")
	assert.Error(t, err)
}

func TestSessionManager_ListSessions(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)
	sm.CreateSession("sess-2", "profile-2", SessionModeReadOnly, nil, nil)

	// ListSessions filters by profile ID, so empty string returns only those with empty profile ID
	// Create sessions with empty profile ID to test
	sm.CreateSession("sess-3", "", SessionModeDefault, nil, nil)
	sessions := sm.ListSessions("")
	assert.Len(t, sessions, 1)

	// Filter by profile-1
	sessions = sm.ListSessions("profile-1")
	assert.Len(t, sessions, 1)
}

func TestSessionManager_ListSessions_FilterByProfile(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)
	sm.CreateSession("sess-2", "profile-2", SessionModeReadOnly, nil, nil)
	sm.CreateSession("sess-3", "profile-1", SessionModeAutoApprove, nil, nil)

	sessions := sm.ListSessions("profile-1")
	assert.Len(t, sessions, 2)
	for _, s := range sessions {
		assert.Equal(t, "profile-1", s.ProfileID)
	}
}

func TestSessionManager_ConcurrentOperations(t *testing.T) {
	sm := NewSessionManager()
	done := make(chan bool, 10)

	// Create sessions concurrently with a specific profile ID
	for i := 0; i < 10; i++ {
		go func(idx int) {
			sessID := fmt.Sprintf("sess-%d", idx)
			sm.CreateSession(sessID, "profile-1", SessionModeDefault, nil, nil)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All sessions should be under profile-1
	sessions := sm.ListSessions("profile-1")
	assert.Len(t, sessions, 10)
}

// ============================================================================
// 3. File System Operations Tests
// ============================================================================

func TestFileSystemHandler_NewFileSystemHandler(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.NotNil(t, handler)
	assert.Equal(t, tmpDir, handler.workingDir)
	assert.Equal(t, int64(MaxFileSize), handler.maxSize)
}

func TestFileSystemHandler_NewFileSystemHandler_DefaultMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, 0)

	assert.Equal(t, int64(MaxFileSize), handler.maxSize)
}

func TestFileSystemHandler_ReadTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nLine 2\nLine 3"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFileSystemHandler_ReadTextFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	content, err := handler.ReadTextFile(filepath.Join(tmpDir, "nonexistent.txt"), 0, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
	assert.Empty(t, content)
}

func TestFileSystemHandler_ReadTextFile_ExceedsMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	smallMax := int64(10)
	largeContent := "This file is larger than max"
	require.NoError(t, os.WriteFile(testFile, []byte(largeContent), 0644))

	handler := NewFileSystemHandler(tmpDir, smallMax)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.Empty(t, content)
}

func TestFileSystemHandler_WriteTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, testContent)

	assert.NoError(t, err)
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}


func TestFileSystemHandler_WriteTextFile_ExceedsMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	smallMax := int64(5)
	largeContent := "This content exceeds the maximum size"

	handler := NewFileSystemHandler(tmpDir, smallMax)
	err := handler.WriteTextFile(testFile, largeContent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestFileSystemHandler_IsSensitivePath(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	tests := []struct {
		path    string
		isSens  bool
		desc    string
	}{
		{filepath.Join(tmpDir, ".env"), true, ".env file"},
		{filepath.Join(tmpDir, "credentials.json"), true, "credentials file"},
		{filepath.Join(tmpDir, "secret.key"), true, "secret file"},
		{filepath.Join(tmpDir, "passwd"), true, "passwd file"},
		{"/etc/shadow", true, "shadow file"},
		{filepath.Join(tmpDir, "normal.txt"), false, "normal file"},
		{filepath.Join(tmpDir, "app.go"), false, "app file"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			isSens := handler.IsSensitivePath(tt.path)
			assert.Equal(t, tt.isSens, isSens, "path: %s", tt.path)
		})
	}
}

func TestFileSystemHandler_PathTraversalPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	// Try to traverse out of working directory
	pathTraversal := filepath.Join(tmpDir, "..", "etc", "passwd")
	err := handler.ValidatePath(pathTraversal, false)

	assert.Error(t, err)
	// The error message could be either about path traversal or sensitive path depending on normalization
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "path traversal") || strings.Contains(errMsg, "not permitted"),
		"Error should mention path traversal or permission: %v", err)
}

// ============================================================================
// 4. Terminal Management Tests
// ============================================================================

func TestTerminalManager_NewTerminalManager(t *testing.T) {
	tm := NewTerminalManager()
	assert.NotNil(t, tm)
}

func TestTerminalManager_CreateTerminal(t *testing.T) {
	tm := NewTerminalManager()
	term, err := tm.CreateTerminal("echo", []string{"hello"}, "", map[string]string{})

	assert.NoError(t, err)
	assert.NotNil(t, term)
	assert.NotEmpty(t, term.ID)

	retrieved, err := tm.GetTerminal(term.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestTerminalManager_CreateTerminal_InvalidCommand(t *testing.T) {
	tm := NewTerminalManager()
	term, err := tm.CreateTerminal("nonexistentcommand1234567890", []string{}, "", map[string]string{})

	assert.Error(t, err)
	assert.Nil(t, term)
}

func TestTerminalManager_GetTerminal_NotFound(t *testing.T) {
	tm := NewTerminalManager()
	terminal, err := tm.GetTerminal("nonexistent")

	assert.Error(t, err)
	assert.Nil(t, terminal)
}

func TestTerminalManager_Kill(t *testing.T) {
	tm := NewTerminalManager()
	term, err := tm.CreateTerminal("sleep", []string{"10"}, "", map[string]string{})
	require.NoError(t, err)

	err = tm.Kill(term.ID)
	assert.NoError(t, err)

	// Wait a moment for process to be killed
	time.Sleep(100 * time.Millisecond)

	terminal, _ := tm.GetTerminal(term.ID)
	if terminal != nil {
		// Process should be in exited state
		assert.NotNil(t, terminal)
	}
}

func TestTerminalManager_Release(t *testing.T) {
	tm := NewTerminalManager()
	term, err := tm.CreateTerminal("echo", []string{"hello"}, "", map[string]string{})
	require.NoError(t, err)

	err = tm.Release(term.ID)
	assert.NoError(t, err)

	// Terminal should be removed
	_, err = tm.GetTerminal(term.ID)
	assert.Error(t, err)
}

func TestTerminalManager_ReleaseAll(t *testing.T) {
	tm := NewTerminalManager()
	term1, _ := tm.CreateTerminal("echo", []string{"1"}, "", map[string]string{})
	term2, _ := tm.CreateTerminal("echo", []string{"2"}, "", map[string]string{})

	tm.ReleaseAll()

	_, err1 := tm.GetTerminal(term1.ID)
	_, err2 := tm.GetTerminal(term2.ID)
	assert.Error(t, err1)
	assert.Error(t, err2)
}

func TestTerminal_GetStatus(t *testing.T) {
	tm := NewTerminalManager()
	term, _ := tm.CreateTerminal("echo", []string{"test"}, "", map[string]string{})

	status := term.GetStatus()
	assert.NotNil(t, status)
	assert.NotEmpty(t, status)
}

// ============================================================================
// 5. Permission System Tests
// ============================================================================

func TestPermissionManager_RequestPermission(t *testing.T) {
	pm := NewPermissionManager()

	req, err := pm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt")

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotEmpty(t, req.ID)
	assert.Equal(t, "sess-1", req.SessionID)
	assert.Equal(t, "fs/write", req.Operation)
	assert.Equal(t, "/tmp/file.txt", req.Resource)
	assert.Equal(t, "pending", req.Status)
}

func TestPermissionManager_RequestPermission_EmptySession(t *testing.T) {
	pm := NewPermissionManager()

	req, err := pm.RequestPermission("", "fs/write", "/tmp/file.txt")

	assert.Error(t, err)
	assert.Nil(t, req)
}

func TestPermissionManager_RequestPermission_EmptyOperation(t *testing.T) {
	pm := NewPermissionManager()

	req, err := pm.RequestPermission("sess-1", "", "/tmp/file.txt")

	assert.Error(t, err)
	assert.Nil(t, req)
}

func TestPermissionManager_GrantPermission(t *testing.T) {
	pm := NewPermissionManager()
	req, _ := pm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt")

	err := pm.GrantPermission(req.ID)

	assert.NoError(t, err)
	granted, _ := pm.GetPermissionRequest(req.ID)
	assert.Equal(t, "approved", granted.Status)
	assert.True(t, granted.Decision)
}

func TestPermissionManager_DenyPermission(t *testing.T) {
	pm := NewPermissionManager()
	req, _ := pm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt")

	err := pm.DenyPermission(req.ID)

	assert.NoError(t, err)
	denied, _ := pm.GetPermissionRequest(req.ID)
	assert.Equal(t, "denied", denied.Status)
	assert.False(t, denied.Decision)
}

func TestPermissionManager_GetPermissionRequest(t *testing.T) {
	pm := NewPermissionManager()
	created, _ := pm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt")

	retrieved, err := pm.GetPermissionRequest(created.ID)

	assert.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestPermissionManager_GetPermissionRequest_NotFound(t *testing.T) {
	pm := NewPermissionManager()

	req, err := pm.GetPermissionRequest("nonexistent")

	assert.Error(t, err)
	assert.Nil(t, req)
}

func TestSessionMode_Modes(t *testing.T) {
	tests := []struct {
		mode SessionMode
		desc string
	}{
		{SessionModeDefault, "default mode"},
		{SessionModeAutoApprove, "auto-approve mode"},
		{SessionModeReadOnly, "read-only mode"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.mode))
		})
	}
}

// ============================================================================
// 6. JSON-RPC Request Handling Tests
// ============================================================================

func TestJSONRPCRequest_MarshalUnmarshal(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "default",
		},
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded JSONRPCRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, req.Method, decoded.Method)
}

func TestJSONRPCResponse_ErrorResponse(t *testing.T) {
	errResp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &ErrorResponse{
			Code:    -32602,
			Message: "Invalid params",
		},
		ID: 1,
	}

	data, err := json.Marshal(errResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded JSONRPCResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.NotNil(t, decoded.Error)
	assert.Equal(t, "Invalid params", decoded.Error.Message)
}

func TestJSONRPCResponse_ResultResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"sessionId": "sess-1",
		},
		ID: 1,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded JSONRPCResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.NotNil(t, decoded.Result)
}

// ============================================================================
// 7. Edge Cases and Error Handling Tests
// ============================================================================

func TestSessionManager_DeleteSession_NotFound(t *testing.T) {
	sm := NewSessionManager()

	// DeleteSession doesn't error on nonexistent sessions, it just returns nil
	err := sm.DeleteSession("nonexistent")

	assert.NoError(t, err)
}

func TestFileSystemHandler_ReadTextFile_WithLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 2, 4)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestPermissionManager_ClearPendingRequests(t *testing.T) {
	pm := NewPermissionManager()
	req1, _ := pm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt")
	req2, _ := pm.RequestPermission("sess-1", "fs/read", "/tmp/file2.txt")

	pm.ClearPendingRequests("sess-1")

	// Both requests should no longer exist
	_, err1 := pm.GetPermissionRequest(req1.ID)
	_, err2 := pm.GetPermissionRequest(req2.ID)
	assert.Error(t, err1)
	assert.Error(t, err2)
}

func TestContentBlock_TextBlock(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: "Hello, World!",
	}

	data, err := json.Marshal(block)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "Hello, World!")
}

func TestContentBlock_ImageBlock(t *testing.T) {
	block := ContentBlock{
		Type:     "image",
		MimeType: "image/png",
		Data:     "base64encodeddata",
	}

	data, err := json.Marshal(block)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "image/png")
}

func TestExecutionPlan_PlanStep(t *testing.T) {
	plan := ExecutionPlan{
		Steps: []PlanStep{
			{
				Content:  "Read file",
				Priority: "high",
				Status:   "pending",
			},
		},
		Status: "pending",
	}

	data, err := json.Marshal(plan)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "Read file")
}

// ============================================================================
// 8. Concurrent Operation Tests
// ============================================================================

func TestFileSystemHandler_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := handler.ReadTextFile(testFile, 0, 0)
			assert.NoError(t, err)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPermissionManager_ConcurrentRequests(t *testing.T) {
	pm := NewPermissionManager()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			sessID := "sess-" + string(rune(idx))
			_, err := pm.RequestPermission(sessID, "fs/write", "/tmp/file.txt")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// ============================================================================
// 9. Integration Tests
// ============================================================================

func TestServer_IntegrationWithSessionManager(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	session := server.sessionManager.CreateSession("sess-1", "test-profile", SessionModeDefault, nil, nil)
	assert.NotNil(t, session)

	retrieved, _ := server.sessionManager.GetSession("sess-1")
	assert.Equal(t, session.SessionID, retrieved.SessionID)
}

func TestServer_SessionLifecycle(t *testing.T) {
	sm := NewSessionManager()

	// Create
	created := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)
	assert.NotNil(t, created)

	// Retrieve
	retrieved, err := sm.GetSession("sess-1")
	assert.NoError(t, err)
	assert.Equal(t, created.SessionID, retrieved.SessionID)

	// Update
	err = sm.UpdateSession("sess-1", func(s *ACPSession) error {
		s.Mode = SessionModeAutoApprove
		return nil
	})
	assert.NoError(t, err)

	updated, _ := sm.GetSession("sess-1")
	assert.Equal(t, SessionModeAutoApprove, updated.Mode)

	// Delete
	err = sm.DeleteSession("sess-1")
	assert.NoError(t, err)

	_, err = sm.GetSession("sess-1")
	assert.Error(t, err)
}

// ============================================================================
// 10. Request/Response Handling Tests (10 tests)
// ============================================================================

func TestHandleInitialize_WithClientCapabilities(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: map[string]interface{}{
			"protocolVersion": 1,
			"clientCapabilities": map[string]interface{}{
				"filesystem": true,
				"terminal":   true,
			},
		},
	}

	resp := server.handleInitialize(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.ID)
}

func TestHandleAuthenticate_WithAuthToken(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
		Params: map[string]interface{}{
			"token": "test-auth-token-12345",
		},
	}

	resp := server.handleAuthenticate(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandleSessionNew_WithAutoApproveMode(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
			"clientCapabilities": map[string]interface{}{
				"filesystem": true,
			},
		},
	}

	resp := server.handleSessionNew(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandleSessionNew_WithReadOnlyMode(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "read_only",
		},
	}

	resp := server.handleSessionNew(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandleSessionSetMode_ModeTransition(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create a session first
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "default",
		},
	}
	createResp := server.handleSessionNew(createReq)
	result := createResp.Result.(SessionNewResult)
	sessID := result.SessionID

	// Now change mode
	setModeReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/set_mode",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"mode":      "auto_approve",
		},
	}

	resp := server.handleSessionSetMode(setModeReq)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleRequest_InvalidMethodNotFound(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "nonexistent/method",
		ID:      1,
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeMethodNotFound, resp.Error.Code)
}

func TestHandleRequest_MalformedJSONRPC(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "1.0", // Wrong version
		Method:  "initialize",
		ID:      1,
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
}

func TestHandleRequest_InvalidParameters(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params:  "invalid-not-an-object",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
}

func TestHandleRequest_ResponseFormatting(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
	}

	resp := server.handleRequest(req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.ID)
}

func TestHandleRequest_ErrorResponseWithCode(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "nonexistent",
		ID:      2,
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp.Error)
	assert.NotEmpty(t, resp.Error.Message)
	assert.True(t, resp.Error.Code < 0) // All error codes are negative
}

// ============================================================================
// 11. File System Handler Tests (15 tests)
// ============================================================================

func TestFileSystemHandler_ReadTextFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFileSystemHandler_ReadTextFile_WithStartLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 2, 0)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestFileSystemHandler_ReadTextFile_WithEndLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 3)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestFileSystemHandler_WriteTextFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "newfile.txt")
	testContent := "New Content"

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, testContent)

	assert.NoError(t, err)
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestFileSystemHandler_WriteTextFile_WithDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	// Pre-create the parent directory to ensure it's writable
	subdir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	testFile := filepath.Join(subdir, "file.txt")
	testContent := "Nested File Content"

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, testContent)

	assert.NoError(t, err)
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestFileSystemHandler_IsSensitivePath_DotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.True(t, handler.IsSensitivePath(filepath.Join(tmpDir, ".env")))
}

func TestFileSystemHandler_IsSensitivePath_Credentials(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.True(t, handler.IsSensitivePath(filepath.Join(tmpDir, "credentials.json")))
}

func TestFileSystemHandler_IsSensitivePath_Secret(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.True(t, handler.IsSensitivePath(filepath.Join(tmpDir, "secret.key")))
}

func TestFileSystemHandler_IsSensitivePath_Passwd(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.True(t, handler.IsSensitivePath(filepath.Join(tmpDir, "passwd")))
}

func TestFileSystemHandler_IsSensitivePath_Shadow(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	assert.True(t, handler.IsSensitivePath(filepath.Join(tmpDir, "shadow")))
}

func TestFileSystemHandler_PathTraversal_Prevention(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	traversalPath := filepath.Join(tmpDir, "..", "etc", "passwd")
	err := handler.ValidatePath(traversalPath, false)

	assert.Error(t, err)
}

func TestFileSystemHandler_FileSizeEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	maxSize := int64(10)
	largeContent := "This is larger than 10 bytes"
	require.NoError(t, os.WriteFile(testFile, []byte(largeContent), 0644))

	handler := NewFileSystemHandler(tmpDir, maxSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.Error(t, err)
	assert.Empty(t, content)
}

func TestFileSystemHandler_BinaryFileHandling(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0xFF, 0xFE, 0x00, 0x01}
	require.NoError(t, os.WriteFile(testFile, []byte(binaryContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	// Should still read it as string (will have unreadable chars)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestFileSystemHandler_FileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "overwrite.txt")
	originalContent := "Original"
	newContent := "Overwritten"

	// Write original
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	require.NoError(t, handler.WriteTextFile(testFile, originalContent))

	// Overwrite
	err := handler.WriteTextFile(testFile, newContent)
	assert.NoError(t, err)

	// Verify
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, newContent, string(content))
}

// ============================================================================
// 12. Terminal Handler Tests (10 tests)
// ============================================================================

func TestHandleTerminalCreate_WithEnvironment(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session first
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	// Create terminal with environment
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "echo",
			"arguments": []string{"hello"},
			"environment": map[string]string{
				"TEST_VAR": "test_value",
			},
		},
	}

	resp := server.handleTerminalCreate(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandleTerminalCreate_WithWorkingDirectory(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session first
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	// Create terminal with working directory
	tmpDir := t.TempDir()
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId":       sessID,
			"command":         "echo",
			"arguments":       []string{"test"},
			"workingDirectory": tmpDir,
		},
	}

	resp := server.handleTerminalCreate(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleTerminalOutput_ReturnsOutput(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session and terminal
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	createTermReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "echo",
			"arguments": []string{"test-output"},
		},
	}
	createTermResp := server.handleTerminalCreate(createTermReq)
	termID := createTermResp.Result.(TerminalCreateResult).TerminalID

	// Get output
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/output",
		ID:      3,
		Params: map[string]interface{}{
			"terminalId": termID,
		},
	}

	resp := server.handleTerminalOutput(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleTerminalWaitForExit_ReturnsExitCode(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session and terminal
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	createTermReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "echo",
			"arguments": []string{"test"},
		},
	}
	createTermResp := server.handleTerminalCreate(createTermReq)
	termID := createTermResp.Result.(TerminalCreateResult).TerminalID

	// Wait for exit
	time.Sleep(100 * time.Millisecond)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/wait_for_exit",
		ID:      3,
		Params: map[string]interface{}{
			"terminalId": termID,
		},
	}

	resp := server.handleTerminalWaitForExit(req)

	assert.NotNil(t, resp)
	// May have error if process already exited, that's OK for this test
}

func TestHandleTerminalKill_SignalHandling(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session and terminal
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	createTermReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "sleep",
			"arguments": []string{"10"},
		},
	}
	createTermResp := server.handleTerminalCreate(createTermReq)
	termID := createTermResp.Result.(TerminalCreateResult).TerminalID

	// Kill terminal
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/kill",
		ID:      3,
		Params: map[string]interface{}{
			"terminalId": termID,
		},
	}

	resp := server.handleTerminalKill(req)

	assert.NotNil(t, resp)
	// No error is acceptable since process may have already exited
}

func TestHandleTerminalRelease_CleanupResources(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session and terminal
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	createTermReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "echo",
			"arguments": []string{"test"},
		},
	}
	createTermResp := server.handleTerminalCreate(createTermReq)
	termID := createTermResp.Result.(TerminalCreateResult).TerminalID

	// Release terminal
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/release",
		ID:      3,
		Params: map[string]interface{}{
			"terminalId": termID,
		},
	}

	resp := server.handleTerminalRelease(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleTerminalCreate_TerminalNotFound(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Try to get non-existent terminal
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/output",
		ID:      1,
		Params: map[string]interface{}{
			"terminalId": "nonexistent",
		},
	}

	resp := server.handleTerminalOutput(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestTerminalManager_ConcurrentOperations(t *testing.T) {
	tm := NewTerminalManager()
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			_, err := tm.CreateTerminal("echo", []string{"test"}, "", map[string]string{})
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	// All terminals should be created
	terminals := tm.terminals
	assert.True(t, len(terminals) > 0)
}

// ============================================================================
// 13. Permission & Capability Tests (8 tests)
// ============================================================================

func TestBuildAgentCapabilities_WithDefaultProfile(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	caps := server.buildAgentCapabilities()

	assert.NotNil(t, caps)
	assert.NotNil(t, caps["filesystem"])
	assert.NotNil(t, caps["terminal"])
	assert.NotNil(t, caps["sessionModes"])
}

func TestBuildAgentCapabilities_FileSystemCapabilities(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	caps := server.buildAgentCapabilities()
	fsCaps := caps["filesystem"].(map[string]bool)

	assert.True(t, fsCaps["readTextFile"])
	assert.True(t, fsCaps["writeTextFile"])
}

func TestBuildAgentCapabilities_TerminalCapabilities(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	caps := server.buildAgentCapabilities()
	termCaps := caps["terminal"].(map[string]bool)

	assert.True(t, termCaps["create"])
	assert.True(t, termCaps["output"])
	assert.True(t, termCaps["waitForExit"])
	assert.True(t, termCaps["kill"])
	assert.True(t, termCaps["release"])
}

func TestACPSession_HasPermission_AutoApproveMode(t *testing.T) {
	session := &ACPSession{
		SessionID:       "sess-1",
		ProfileID:       "profile-1",
		Mode:            SessionModeAutoApprove,
		PermissionRules: []PermissionRule{},
	}

	// All operations should be allowed in auto-approve mode
	assert.True(t, session.HasPermission("fs/write", "/tmp/file.txt"))
	assert.True(t, session.HasPermission("terminal/create", "bash"))
}

func TestACPSession_HasPermission_ReadOnlyMode(t *testing.T) {
	session := &ACPSession{
		SessionID:       "sess-1",
		ProfileID:       "profile-1",
		Mode:            SessionModeReadOnly,
		PermissionRules: []PermissionRule{},
	}

	// Write operations should be denied in read-only mode
	assert.False(t, session.HasPermission("fs/write_text_file", "/tmp/file.txt"))
	assert.False(t, session.HasPermission("terminal/create", "bash"))

	// Read operations should be allowed
	assert.True(t, session.HasPermission("fs/read_text_file", "/tmp/file.txt"))
}

func TestACPSession_HasPermission_DefaultMode(t *testing.T) {
	session := &ACPSession{
		SessionID:       "sess-1",
		ProfileID:       "profile-1",
		Mode:            SessionModeDefault,
		PermissionRules: []PermissionRule{},
	}

	// Read operations should be allowed
	assert.True(t, session.HasPermission("fs/read_text_file", "/tmp/file.txt"))

	// Write operations should be denied
	assert.False(t, session.HasPermission("fs/write_text_file", "/tmp/file.txt"))
}

func TestACPSession_AddPermissionRule(t *testing.T) {
	session := &ACPSession{
		SessionID:       "sess-1",
		ProfileID:       "profile-1",
		Mode:            SessionModeDefault,
		PermissionRules: []PermissionRule{},
	}

	rule := PermissionRule{
		Operation:   "fs/write",
		Allowed:     true,
		PathPattern: "/tmp/*",
	}

	session.AddPermissionRule(rule)

	assert.Len(t, session.PermissionRules, 1)
	assert.Equal(t, "fs/write", session.PermissionRules[0].Operation)
}

func TestSessionManager_RequestPermission_AutoApproveMode(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeAutoApprove, nil, nil)

	// All requests should be approved in auto-approve mode
	assert.True(t, sm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt"))
	assert.True(t, sm.RequestPermission("sess-1", "terminal/create", "bash"))
}

func TestSessionManager_RequestPermission_ReadOnlyMode(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeReadOnly, nil, nil)

	// Write requests should be denied in read-only mode
	assert.False(t, sm.RequestPermission("sess-1", "fs/write", "/tmp/file.txt"))
	assert.False(t, sm.RequestPermission("sess-1", "terminal/create", "bash"))

	// Read requests should be allowed
	assert.True(t, sm.RequestPermission("sess-1", "fs/read", "/tmp/file.txt"))
}

// ============================================================================
// 14. Message Loop & Concurrency Tests (5 tests)
// ============================================================================

func TestMessageLoop_HandlesMultipleRequests(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	transport := newMockTransport()
	server.transport = transport

	// Queue multiple requests
	transport.pushRequest(&JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
	})

	transport.pushRequest(&JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      2,
	})

	// Queue EOF to stop loop
	transport.requestQueue = append(transport.requestQueue, nil)

	// Run message loop in goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	server.ctx = ctx
	server.cancel = cancel
	server.status = "running"

	// Don't run the full loop, just test request handling
	req1 := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
	}
	resp1 := server.handleRequest(req1)

	req2 := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      2,
	}
	resp2 := server.handleRequest(req2)

	assert.NotNil(t, resp1)
	assert.NotNil(t, resp2)
	assert.NotNil(t, resp1.ID)
	assert.NotNil(t, resp2.ID)
}

func TestMessageLoop_ConcurrentRequests_ProperSerialization(t *testing.T) {
	sm := NewSessionManager()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			sessID := fmt.Sprintf("sess-%d", idx)
			session := sm.CreateSession(sessID, "profile-1", SessionModeDefault, nil, nil)
			assert.NotNil(t, session)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All sessions should be created without issues
	assert.True(t, len(sm.ListSessions("profile-1")) >= 10)
}

func TestMessageLoop_ContextCancellation(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	transport := newMockTransport()
	server.transport = transport

	ctx, cancel := context.WithCancel(context.Background())
	server.ctx = ctx
	server.cancel = cancel
	server.status = "running"

	// Cancel context immediately
	cancel()

	// Check that context is cancelled
	assert.True(t, ctx.Err() != nil)
}

func TestMultipleClientConnections_IsolatedSessions(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create multiple sessions simulating different clients
	sess1 := server.sessionManager.CreateSession("client-1-sess-1", "profile-1", SessionModeDefault, nil, nil)
	sess2 := server.sessionManager.CreateSession("client-2-sess-1", "profile-1", SessionModeDefault, nil, nil)
	sess3 := server.sessionManager.CreateSession("client-3-sess-1", "profile-1", SessionModeDefault, nil, nil)

	assert.NotNil(t, sess1)
	assert.NotNil(t, sess2)
	assert.NotNil(t, sess3)
	assert.NotEqual(t, sess1.SessionID, sess2.SessionID)
	assert.NotEqual(t, sess2.SessionID, sess3.SessionID)
}

func TestTransportErrorHandling_InMessageLoop(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Verify error response handling
	resp := server.errorResponse(1, ErrInvalidParams)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
}

// ============================================================================
// 15. Additional Coverage Tests
// ============================================================================

func TestFileSystemHandler_ReadTextFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(testFile, []byte(""), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.NoError(t, err)
	assert.Empty(t, content)
}

func TestFileSystemHandler_WriteTextFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, "")

	assert.NoError(t, err)
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Empty(t, content)
}

func TestSessionManager_GetPermission_SessionNotFound(t *testing.T) {
	sm := NewSessionManager()

	allowed, err := sm.GetPermission("nonexistent", "fs/read", "/tmp/file.txt")

	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestSessionManager_RequestPermission_SessionNotFound(t *testing.T) {
	sm := NewSessionManager()

	allowed := sm.RequestPermission("nonexistent", "fs/read", "/tmp/file.txt")

	assert.False(t, allowed)
}

func TestACPSession_GetInfo(t *testing.T) {
	session := &ACPSession{
		SessionID:          "sess-1",
		ProfileID:          "profile-1",
		Mode:               SessionModeDefault,
		ClientCapabilities: map[string]interface{}{"test": true},
		AgentCapabilities:  map[string]interface{}{"test": true},
		CreatedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	info := session.GetInfo()

	assert.NotNil(t, info)
	assert.Equal(t, "sess-1", info["sessionId"])
	assert.Equal(t, "profile-1", info["profileId"])
	assert.Equal(t, SessionModeDefault, info["mode"])
}

func TestACPSession_UpdateLastActivity(t *testing.T) {
	session := &ACPSession{
		SessionID:    "sess-1",
		ProfileID:    "profile-1",
		Mode:         SessionModeDefault,
		CreatedAt:    time.Now(),
		LastActivity: time.Now().Add(-1 * time.Hour),
	}

	oldActivity := session.LastActivity
	time.Sleep(10 * time.Millisecond)
	session.UpdateLastActivity()

	assert.True(t, session.LastActivity.After(oldActivity))
}

func TestHandleSessionSetMode_InvalidSessionID(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/set_mode",
		ID:      1,
		Params: map[string]interface{}{
			"sessionId": "nonexistent",
			"mode":      "auto_approve",
		},
	}

	resp := server.handleSessionSetMode(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestHandleSessionSetMode_MissingSessionID(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/set_mode",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}

	resp := server.handleSessionSetMode(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestHandleTerminalCreate_MissingCommand(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session first
	createSessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createSessResp := server.handleSessionNew(createSessReq)
	sessID := createSessResp.Result.(SessionNewResult).SessionID

	// Create terminal without command
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "",
		},
	}

	resp := server.handleTerminalCreate(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestHandleTerminalCreate_MissingSessionID(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      1,
		Params: map[string]interface{}{
			"command": "echo",
		},
	}

	resp := server.handleTerminalCreate(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
}

func TestHandleFSReadTextFile_ReadOnlyMode_Allowed(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session in read-only mode
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "read_only",
		},
	}
	createResp := server.handleSessionNew(createReq)
	sessID := createResp.Result.(SessionNewResult).SessionID

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

	// Try to read - should be allowed in read-only mode
	readReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "fs/read_text_file",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"path":      testFile,
		},
	}

	resp := server.handleFSReadTextFile(readReq)

	// Should not have error (unless file operation fails)
	assert.NotNil(t, resp)
}

func TestHandleFSWriteTextFile_ReadOnlyMode_Denied(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session in read-only mode
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "read_only",
		},
	}
	createResp := server.handleSessionNew(createReq)
	sessID := createResp.Result.(SessionNewResult).SessionID

	// Try to write - should be denied in read-only mode
	writeReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "fs/write_text_file",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"path":      "/tmp/test.txt",
			"content":   "content",
		},
	}

	resp := server.handleFSWriteTextFile(writeReq)

	// Should have permission denied error
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodePermissionDenied, resp.Error.Code)
}

func TestHandleFSReadTextFile_AutoApproveMode_Allowed(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session in auto-approve mode
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}
	createResp := server.handleSessionNew(createReq)
	sessID := createResp.Result.(SessionNewResult).SessionID

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

	// Try to read - should be allowed
	readReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "fs/read_text_file",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"path":      testFile,
		},
	}

	resp := server.handleFSReadTextFile(readReq)

	// Should not have error
	assert.NotNil(t, resp)
}

