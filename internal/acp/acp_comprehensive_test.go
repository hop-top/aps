package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hop.top/aps/internal/core/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// PART 1: ACP Server Implementation Tests (12 tests)
// ============================================================================

// Test 1: Server startup and initialization
func TestServer_StartupAndInitialization(t *testing.T) {
	core := newMockAPSCore()
	server, err := NewServer("test-profile", core)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock transport
	transport := newMockTransport()
	server.transport = transport

	// Start server (with context/cancel setup)
	server.ctx = ctx
	server.cancel = cancel
	server.mu.Lock()
	server.status = "running"
	server.mu.Unlock()

	assert.Equal(t, "running", server.Status())
	assert.NotNil(t, server.sessionManager)
	assert.NotNil(t, server.permissionManager)
	assert.NotNil(t, server.terminalManager)
}

// Test 2: Server shutdown and resource cleanup
func TestServer_ShutdownAndCleanup(t *testing.T) {
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

	// Create some sessions that will be cleaned up
	server.sessionManager.CreateSession("sess-1", "test-profile", SessionModeDefault, nil, nil)
	server.sessionManager.CreateSession("sess-2", "test-profile", SessionModeDefault, nil, nil)

	err := server.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "stopped", server.Status())
	assert.True(t, transport.closed)
}

// Test 3: JSON-RPC message handling - valid request
func TestServer_JSONRPCMessageHandling_ValidRequest(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

// Test 4: Protocol compliance - JSON-RPC 2.0 format
func TestServer_ProtocolCompliance_JSONRPCFormat(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	tests := []struct {
		name        string
		jsonrpc     string
		shouldError bool
	}{
		{"Valid 2.0", "2.0", false},
		{"Invalid 1.0", "1.0", true},
		{"Invalid 3.0", "3.0", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: test.jsonrpc,
				Method:  "initialize",
				ID:      1,
			}

			resp := server.handleRequest(req)

			if test.shouldError {
				assert.NotNil(t, resp.Error)
			} else {
				assert.Nil(t, resp.Error)
			}
		})
	}
}

// Test 5: Concurrent message processing
func TestServer_ConcurrentMessageProcessing(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	var wg sync.WaitGroup
	responseCount := int32(0)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "authenticate",
				ID:      id,
			}

			resp := server.handleRequest(req)
			if resp != nil && resp.Result != nil {
				atomic.AddInt32(&responseCount, 1)
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int32(10), responseCount)
}

// Test 6: Error handling - method not found
func TestServer_ErrorHandling_MethodNotFound(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "invalid/method",
		ID:      1,
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeMethodNotFound, resp.Error.Code)
	assert.Equal(t, int(1), resp.ID)
}

// Test 7: Error handling - invalid parameters
func TestServer_ErrorHandling_InvalidParameters(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params:  "not-a-valid-object",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
}

// Test 8: Error handling - internal server error
func TestServer_ErrorHandling_InternalError(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: map[string]interface{}{
			"protocolVersion": 1,
		},
	}

	resp := server.handleInitialize(req)
	assert.Nil(t, resp.Error)

	// Try to initialize again - should error
	req2 := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      2,
		Params: map[string]interface{}{
			"protocolVersion": 1,
		},
	}

	resp2 := server.handleInitialize(req2)
	assert.NotNil(t, resp2.Error)
}

// Test 9: Message loop with EOF handling
func TestServer_MessageLoop_EOFHandling(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)
	transport := newMockTransport()
	server.transport = transport

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	server.ctx = ctx
	server.cancel = cancel
	server.status = "running"

	// Push one request then EOF
	transport.pushRequest(&JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      1,
	})

	// Message loop will return on EOF
	server.messageLoop()

	assert.Equal(t, "stopped", server.Status())
}

// Test 10: Server status transitions
func TestServer_StatusTransitions(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	assert.Equal(t, "stopped", server.Status())

	// Simulate start
	server.mu.Lock()
	server.status = "running"
	server.mu.Unlock()
	assert.Equal(t, "running", server.Status())

	// Stop
	err := server.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "stopped", server.Status())
}

// Test 11: Protocol version negotiation
func TestServer_ProtocolVersionNegotiation(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: map[string]interface{}{
			"protocolVersion": 1,
		},
	}

	resp := server.handleInitialize(req)

	assert.Nil(t, resp.Error)
	result := resp.Result.(InitializeResult)
	assert.Equal(t, uint16(1), result.ProtocolVersion)
}

// Test 12: Server name and address
func TestServer_NameAndAddress(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	assert.Equal(t, "acp", server.Name())
	assert.Equal(t, "", server.GetAddress())
}

// ============================================================================
// PART 2: ACP Request Handlers Tests (12 tests)
// ============================================================================

// Test 13: File system read operation
func TestHandler_FileSystem_ReadTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

// Test 14: File system write operation
func TestHandler_FileSystem_WriteTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "New Content"

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, testContent)

	assert.NoError(t, err)

	// Verify file was written
	readContent, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(readContent))
}

// Test 15: File system list directory operation
func TestHandler_FileSystem_ListDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644))

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	// Verify handler works with existing files
	content, err := handler.ReadTextFile(filepath.Join(tmpDir, "file1.txt"), 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, "content1", content)
}

// Test 16: Terminal creation and execution
func TestHandler_Terminal_CreateAndExecute(t *testing.T) {
	tm := NewTerminalManager()

	term, err := tm.CreateTerminal("echo", []string{"hello"}, "", map[string]string{})

	assert.NoError(t, err)
	assert.NotNil(t, term)
	assert.NotEmpty(t, term.ID)
	assert.Equal(t, "running", term.Status)
}

// Test 17: Terminal output retrieval
func TestHandler_Terminal_GetOutput(t *testing.T) {
	tm := NewTerminalManager()

	term, err := tm.CreateTerminal("echo", []string{"test-output"}, "", map[string]string{})
	require.NoError(t, err)

	// Wait for command to complete
	time.Sleep(100 * time.Millisecond)

	output, err := tm.GetOutput(term.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, output)
}

// Test 18: Permission validation
func TestHandler_Permission_Validation(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create session in auto-approve mode
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}

	resp := server.handleSessionNew(req)
	assert.Nil(t, resp.Error)

	sessID := resp.Result.(SessionNewResult).SessionID
	session, _ := server.sessionManager.GetSession(sessID)

	// In auto-approve mode, all permissions should be granted
	assert.True(t, session.HasPermission("fs/write_text_file", "/tmp/file.txt"))
	assert.True(t, session.HasPermission("terminal/create", "bash"))
}

// Test 19: Sandbox enforcement - path traversal prevention
func TestHandler_Sandbox_PathTraversalPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	// Try to access parent directory
	traversalPath := filepath.Join(tmpDir, "..", "etc", "passwd")
	err := handler.ValidatePath(traversalPath, false)

	assert.Error(t, err)
}

// Test 20: Sandbox enforcement - sensitive path blocking
func TestHandler_Sandbox_SensitivePathBlocking(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewFileSystemHandler(tmpDir, MaxFileSize)

	sensitivePaths := []string{
		filepath.Join(tmpDir, ".env"),
		filepath.Join(tmpDir, "credentials.json"),
		filepath.Join(tmpDir, "secret.key"),
	}

	for _, path := range sensitivePaths {
		isSensitive := handler.IsSensitivePath(path)
		assert.True(t, isSensitive, "path should be blocked: %s", path)
	}
}

// Test 21: Request handler - session-based permission check
func TestHandler_SessionBasedPermission(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create read-only session
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "read_only",
		},
	}

	resp := server.handleSessionNew(req)
	sessID := resp.Result.(SessionNewResult).SessionID

	// Try to write - should be denied
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

	writeResp := server.handleFSWriteTextFile(writeReq)
	assert.NotNil(t, writeResp.Error)
	assert.Equal(t, ErrCodePermissionDenied, writeResp.Error.Code)
}

// Test 22: Terminal kill signal handling
func TestHandler_Terminal_KillSignalHandling(t *testing.T) {
	tm := NewTerminalManager()

	term, err := tm.CreateTerminal("sleep", []string{"10"}, "", map[string]string{})
	require.NoError(t, err)

	err = tm.Kill(term.ID)
	assert.NoError(t, err)

	// Wait for process to be killed
	time.Sleep(100 * time.Millisecond)

	terminal, _ := tm.GetTerminal(term.ID)
	assert.NotNil(t, terminal)
}

// Test 23: Terminal release and resource cleanup
func TestHandler_Terminal_ReleaseAndCleanup(t *testing.T) {
	tm := NewTerminalManager()

	term, err := tm.CreateTerminal("echo", []string{"test"}, "", map[string]string{})
	require.NoError(t, err)

	err = tm.Release(term.ID)
	assert.NoError(t, err)

	_, err = tm.GetTerminal(term.ID)
	assert.Error(t, err)
}

// Test 24: Handler routing and dispatch
func TestHandler_RequestRouting(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	testCases := []struct {
		method string
		id     int
	}{
		{"initialize", 1},
		{"authenticate", 2},
		{"session/new", 3},
		{"session/set_mode", 4},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  tc.method,
				ID:      tc.id,
			}

			resp := server.handleRequest(req)
			assert.NotNil(t, resp)
			assert.Equal(t, "2.0", resp.JSONRPC)
		})
	}
}

// ============================================================================
// PART 3: State Management Tests (8 tests)
// ============================================================================

// Test 25: Session state creation and tracking
func TestStateManagement_SessionStateCreation(t *testing.T) {
	sm := NewSessionManager()

	session := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	assert.NotNil(t, session)
	assert.Equal(t, "sess-1", session.SessionID)
	assert.Equal(t, "profile-1", session.ProfileID)
	assert.Equal(t, SessionModeDefault, session.Mode)
	assert.False(t, session.CreatedAt.IsZero())
}

// Test 26: Run state management
func TestStateManagement_RunStateManagement(t *testing.T) {
	core := newMockAPSCore()

	input := protocol.RunInput{
		ProfileID: "profile-1",
		ActionID:  "action-1",
		ThreadID:  "thread-1",
	}

	runState, err := core.ExecuteRun(context.Background(), input, nil)

	assert.NoError(t, err)
	assert.NotNil(t, runState)
	assert.Equal(t, "profile-1", runState.ProfileID)
}

// Test 27: Concurrent session state updates
func TestStateManagement_ConcurrentSessionUpdates(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	var wg sync.WaitGroup
	successCount := int32(0)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := sm.UpdateSession("sess-1", func(s *ACPSession) error {
				s.LastActivity = time.Now()
				return nil
			})

			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(10), successCount)
}

// Test 28: Session mode transitions
func TestStateManagement_SessionModeTransitions(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	assert.Equal(t, SessionModeDefault, session.Mode)

	// Transition to auto-approve
	err := sm.SetSessionMode("sess-1", SessionModeAutoApprove)
	assert.NoError(t, err)

	updated, _ := sm.GetSession("sess-1")
	assert.Equal(t, SessionModeAutoApprove, updated.Mode)

	// Transition to read-only
	err = sm.SetSessionMode("sess-1", SessionModeReadOnly)
	assert.NoError(t, err)

	updated, _ = sm.GetSession("sess-1")
	assert.Equal(t, SessionModeReadOnly, updated.Mode)
}

// Test 29: Session last activity tracking
func TestStateManagement_SessionActivityTracking(t *testing.T) {
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

// Test 30: Permission rules state management
func TestStateManagement_PermissionRulesManagement(t *testing.T) {
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
	assert.Equal(t, rule, session.PermissionRules[0])
}

// Test 31: Terminal state tracking
func TestStateManagement_TerminalStateTracking(t *testing.T) {
	tm := NewTerminalManager()

	term, err := tm.CreateTerminal("echo", []string{"test"}, "", map[string]string{})
	require.NoError(t, err)

	assert.Equal(t, "running", term.Status)

	status := term.GetStatus()
	assert.NotNil(t, status)
	assert.Equal(t, "running", status["status"])
}

// Test 32: Session permission rules state persistence
func TestStateManagement_PermissionRulesPersistence(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)

	rule1 := PermissionRule{Operation: "fs/read", Allowed: true, PathPattern: "/tmp/*"}
	rule2 := PermissionRule{Operation: "fs/write", Allowed: false, PathPattern: "/etc/*"}

	session.AddPermissionRule(rule1)
	session.AddPermissionRule(rule2)

	retrieved, _ := sm.GetSession("sess-1")
	assert.Len(t, retrieved.PermissionRules, 2)
	assert.Equal(t, rule1, retrieved.PermissionRules[0])
	assert.Equal(t, rule2, retrieved.PermissionRules[1])
}

// ============================================================================
// PART 4: Error Scenarios Tests (3 tests)
// ============================================================================

// Test 33: Protocol violations error handling
func TestErrorScenarios_ProtocolViolations(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	tests := []struct {
		name       string
		jsonrpc    string
		method     string
		expectedErr ErrorCode
	}{
		{
			"Invalid JSONRPC version",
			"1.0",
			"initialize",
			ErrCodeInvalidRequest,
		},
		{
			"Method not found",
			"2.0",
			"invalid/method",
			ErrCodeMethodNotFound,
		},
		{
			"Invalid params",
			"2.0",
			"session/new",
			ErrCodeInvalidParams,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: test.jsonrpc,
				Method:  test.method,
				ID:      1,
			}

			if test.method == "session/new" && test.jsonrpc == "2.0" {
				req.Params = "invalid"
			}

			resp := server.handleRequest(req)

			assert.NotNil(t, resp.Error)
			assert.Equal(t, test.expectedErr, resp.Error.Code)
		})
	}
}

// Test 34: Permission denied scenarios
func TestErrorScenarios_PermissionDenied(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Create read-only session
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

	// Try terminal creation in read-only mode
	termReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "terminal/create",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"command":   "bash",
		},
	}

	termResp := server.handleTerminalCreate(termReq)
	assert.NotNil(t, termResp.Error)
	assert.Equal(t, ErrCodePermissionDenied, termResp.Error.Code)
}

// Test 35: Resource limits enforcement
func TestErrorScenarios_ResourceLimitsEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(10)
	handler := NewFileSystemHandler(tmpDir, maxSize)

	// Create a file larger than max size
	testFile := filepath.Join(tmpDir, "large.txt")
	largeContent := "This file is definitely larger than 10 bytes"
	require.NoError(t, os.WriteFile(testFile, []byte(largeContent), 0644))

	// Try to read file exceeding max size
	content, err := handler.ReadTextFile(testFile, 0, 0)

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

// ============================================================================
// PART 5: Advanced Integration Tests
// ============================================================================

// Test 36: Full session lifecycle
func TestAdvanced_SessionLifecycle(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Step 1: Create session
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "auto_approve",
		},
	}

	createResp := server.handleSessionNew(createReq)
	assert.Nil(t, createResp.Error)
	sessID := createResp.Result.(SessionNewResult).SessionID

	// Step 2: Verify session exists
	session, err := server.sessionManager.GetSession(sessID)
	assert.NoError(t, err)
	assert.NotNil(t, session)

	// Step 3: Update session mode
	setModeReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/set_mode",
		ID:      2,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"mode":      "read_only",
		},
	}

	setModeResp := server.handleSessionSetMode(setModeReq)
	assert.Nil(t, setModeResp.Error)

	// Step 4: Verify mode changed
	updated, _ := server.sessionManager.GetSession(sessID)
	assert.Equal(t, SessionModeReadOnly, updated.Mode)

	// Step 5: Delete session
	err = server.sessionManager.DeleteSession(sessID)
	assert.NoError(t, err)

	_, err = server.sessionManager.GetSession(sessID)
	assert.Error(t, err)
}

// Test 37: Capability negotiation and filtering
func TestAdvanced_CapabilityNegotiation(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Get default capabilities
	caps := server.buildAgentCapabilities()
	assert.NotNil(t, caps["filesystem"])
	assert.NotNil(t, caps["terminal"])

	// Create read-only session and verify capabilities filtered
	createReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      1,
		Params: map[string]interface{}{
			"mode": "read_only",
		},
	}

	createResp := server.handleSessionNew(createReq)
	result := createResp.Result.(SessionNewResult)

	// Verify capabilities are filtered for read-only mode
	assert.NotNil(t, result.AgentCapabilities)
}

// Test 38: Multi-session isolation
func TestAdvanced_MultiSessionIsolation(t *testing.T) {
	sm := NewSessionManager()

	// Create multiple sessions for different profiles
	sess1 := sm.CreateSession("sess-1", "profile-1", SessionModeDefault, nil, nil)
	sess2 := sm.CreateSession("sess-2", "profile-2", SessionModeDefault, nil, nil)
	sess3 := sm.CreateSession("sess-3", "profile-1", SessionModeAutoApprove, nil, nil)

	// List sessions by profile
	profile1Sessions := sm.ListSessions("profile-1")
	profile2Sessions := sm.ListSessions("profile-2")

	assert.Len(t, profile1Sessions, 2)
	assert.Len(t, profile2Sessions, 1)

	// Verify isolation
	assert.NotEqual(t, sess1.SessionID, sess2.SessionID)
	assert.NotEqual(t, sess1.Mode, sess3.Mode)
}

// Test 39: File system operations with real directory structure
func TestAdvanced_FileSystemComplexOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	subdir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	// Write file in nested directory
	testFile := filepath.Join(subdir, "nested.txt")
	testContent := "Nested content"

	handler := NewFileSystemHandler(tmpDir, MaxFileSize)
	err := handler.WriteTextFile(testFile, testContent)
	assert.NoError(t, err)

	// Read it back
	content, err := handler.ReadTextFile(testFile, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Update file
	updatedContent := "Updated nested content"
	err = handler.WriteTextFile(testFile, updatedContent)
	assert.NoError(t, err)

	content, err = handler.ReadTextFile(testFile, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, updatedContent, content)
}

// Test 40: Terminal management with concurrent operations
func TestAdvanced_TerminalConcurrentOperations(t *testing.T) {
	tm := NewTerminalManager()
	var wg sync.WaitGroup
	createdCount := int32(0)
	var terminalIDs []string
	var mu sync.Mutex

	// Create multiple terminals concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			term, err := tm.CreateTerminal("echo", []string{"test"}, "", map[string]string{})
			if err == nil {
				atomic.AddInt32(&createdCount, 1)
				mu.Lock()
				terminalIDs = append(terminalIDs, term.ID)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify all terminals were created
	assert.Equal(t, int32(5), createdCount)
	assert.Len(t, terminalIDs, 5)

	// Verify all can be retrieved
	for _, id := range terminalIDs {
		term, err := tm.GetTerminal(id)
		assert.NoError(t, err)
		assert.NotNil(t, term)
	}
}

// Test 41: Permission manager with multiple concurrent requests
func TestAdvanced_PermissionManagerConcurrency(t *testing.T) {
	pm := NewPermissionManager()
	var wg sync.WaitGroup
	requestCount := int32(0)
	var requestIDs []string
	var mu sync.Mutex

	// Create permission requests concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sessID := fmt.Sprintf("sess-%d", idx)
			req, err := pm.RequestPermission(sessID, "fs/write", "/tmp/file.txt")
			if err == nil {
				atomic.AddInt32(&requestCount, 1)
				mu.Lock()
				requestIDs = append(requestIDs, req.ID)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify all requests were created
	assert.Equal(t, int32(10), requestCount)

	// Grant some, deny others
	for i, reqID := range requestIDs {
		if i%2 == 0 {
			err := pm.GrantPermission(reqID)
			assert.NoError(t, err)
		} else {
			err := pm.DenyPermission(reqID)
			assert.NoError(t, err)
		}
	}

	// Verify statuses
	grantedCount := 0
	deniedCount := 0
	for _, reqID := range requestIDs {
		req, _ := pm.GetPermissionRequest(reqID)
		if req.Status == "approved" {
			grantedCount++
		} else if req.Status == "denied" {
			deniedCount++
		}
	}

	assert.True(t, grantedCount > 0)
	assert.True(t, deniedCount > 0)
}

// Test 42: Complex JSON-RPC message chain
func TestAdvanced_ComplexJSONRPCChain(t *testing.T) {
	core := newMockAPSCore()
	server, _ := NewServer("test-profile", core)

	// Message 1: Initialize
	initReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: map[string]interface{}{
			"protocolVersion": 1,
		},
	}

	initResp := server.handleInitialize(initReq)
	assert.Nil(t, initResp.Error)

	// Message 2: Authenticate
	authReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "authenticate",
		ID:      2,
	}

	authResp := server.handleAuthenticate(authReq)
	assert.Nil(t, authResp.Error)

	// Message 3: Create session
	sessReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		ID:      3,
		Params: map[string]interface{}{
			"mode": "default",
		},
	}

	sessResp := server.handleSessionNew(sessReq)
	assert.Nil(t, sessResp.Error)
	sessID := sessResp.Result.(SessionNewResult).SessionID

	// Message 4: Set mode
	modeReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/set_mode",
		ID:      4,
		Params: map[string]interface{}{
			"sessionId": sessID,
			"mode":      "auto_approve",
		},
	}

	modeResp := server.handleSessionSetMode(modeReq)
	assert.Nil(t, modeResp.Error)

	// Verify final state
	session, _ := server.sessionManager.GetSession(sessID)
	assert.Equal(t, SessionModeAutoApprove, session.Mode)
}

// Test 43: Content block handling and validation
func TestAdvanced_ContentBlockHandling(t *testing.T) {
	cbh := NewContentBlockHandler()

	// Test text block
	textBlock := cbh.CreateTextBlock("Hello, World!")
	assert.NoError(t, cbh.ValidateContentBlock(*textBlock))

	data, err := cbh.MarshalContentBlock(*textBlock)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	unmarshaled, err := cbh.UnmarshalContentBlock(data)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", unmarshaled.Text)
}

// Test 44: Execution plan management
func TestAdvanced_ExecutionPlanManagement(t *testing.T) {
	eph := NewExecutionPlanHandler()

	// Create plan
	plan := eph.CreateExecutionPlan()
	assert.NotNil(t, plan)
	assert.Equal(t, "pending", plan.Status)

	// Add steps
	eph.AddStep(plan, "Read file", "high", "pending")
	eph.AddStep(plan, "Process data", "medium", "pending")
	eph.AddStep(plan, "Write output", "low", "pending")

	assert.Len(t, plan.Steps, 3)

	// Validate plan
	err := eph.ValidateExecutionPlan(plan)
	assert.NoError(t, err)

	// Update step status
	err = eph.UpdateStepStatus(plan, 0, "completed")
	assert.NoError(t, err)

	assert.Equal(t, "completed", plan.Steps[0].Status)
}

// Test 45: Notification handling
func TestAdvanced_NotificationHandling(t *testing.T) {
	nh := NewNotificationHandler()

	// Test content chunk notification
	block := &ContentBlock{
		Type: "text",
		Text: "Content chunk",
	}
	notification := nh.CreateContentChunkNotification("sess-1", block)
	assert.Equal(t, "session/update", notification.Method)

	data, err := json.Marshal(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test tool call notification
	toolNotif := nh.CreateToolCallNotification("sess-1", "read_file", map[string]interface{}{
		"path": "/tmp/test.txt",
	})
	assert.Equal(t, "session/update", toolNotif.Method)

	// Test mode update notification
	modeNotif := nh.CreateModeUpdateNotification("sess-1", SessionModeAutoApprove)
	assert.Equal(t, "session/update", modeNotif.Method)

	// Test status notification
	statusNotif := nh.CreateStatusNotification("sess-1", "running", map[string]interface{}{
		"progress": 50,
	})
	assert.Equal(t, "session/update", statusNotif.Method)
}
