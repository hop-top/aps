package e2e

import (
	"encoding/json"
	"testing"

	"oss-aps-cli/internal/acp"
	"oss-aps-cli/internal/core/protocol"
)

// TestACPFullWorkflow tests a complete ACP session workflow
func TestACPFullWorkflow(t *testing.T) {
	// Setup
	coreAdapter, _ := protocol.NewAPSAdapter()
	server, _ := acp.NewServer("test-profile", coreAdapter)

	sm := server.SessionManager()
	tm := server.TerminalManager()
	pm := server.PermissionManager()

	// Step 1: Initialize
	t.Log("1. Initialize connection...")
	if !server.IsInitialized() {
		// Simulate initialize request handling
		initReq := &acp.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "initialize",
			Params: acp.InitializeParams{
				ProtocolVersion: 1,
			},
			ID: 1,
		}

		if initReq.JSONRPC != "2.0" {
			t.Error("initialize request format invalid")
		}
	}

	// Step 2: Create session
	t.Log("2. Create session...")
	coreSession := &protocol.SessionState{
		SessionID: "sess_test_123",
		ProfileID: "test-profile",
		Metadata:  make(map[string]string),
	}

	session := sm.CreateSession(
		coreSession.SessionID,
		"test-profile",
		acp.SessionModeDefault,
		nil,
		coreSession,
	)

	if session.SessionID != "sess_test_123" {
		t.Fatalf("session creation failed")
	}
	t.Logf("✓ Session created: %s", session.SessionID)

	// Step 3: Test filesystem operations
	t.Log("3. Testing filesystem operations...")
	canRead := session.HasPermission("fs/read_text_file", "/tmp/test.txt")
	if !canRead {
		t.Error("should have read permission in default mode")
	}
	t.Log("✓ Read permission verified")

	// Step 4: Test terminal operations
	t.Log("4. Testing terminal operations...")
	term, err := tm.CreateTerminal("echo", []string{"hello"}, "", nil)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	t.Logf("✓ Terminal created: %s", term.ID)

	// Step 5: Test permissions
	t.Log("5. Testing permission system...")
	permReq, _ := pm.RequestPermission(session.SessionID, "fs/write", "/tmp/file.txt")
	if permReq == nil {
		t.Error("permission request creation failed")
	}
	t.Logf("✓ Permission request created: %s", permReq.ID)

	// Step 6: Grant permission
	t.Log("6. Testing permission grant...")
	pm.GrantPermission(permReq.ID)
	retrieved, _ := pm.GetPermissionRequest(permReq.ID)
	if !retrieved.Decision {
		t.Error("permission should be granted")
	}
	t.Log("✓ Permission granted")

	// Step 7: Change session mode
	t.Log("7. Testing mode change...")
	sm.SetSessionMode(session.SessionID, acp.SessionModeReadOnly)
	updated, _ := sm.GetSession(session.SessionID)
	if updated.Mode != acp.SessionModeReadOnly {
		t.Error("mode should be read_only")
	}
	t.Log("✓ Mode changed to read_only")

	// Step 8: Verify mode enforcement
	t.Log("8. Verifying mode enforcement...")
	canWrite := updated.HasPermission("fs/write", "/tmp/file.txt")
	if canWrite {
		t.Error("should not have write permission in read_only mode")
	}
	canRead = updated.HasPermission("fs/read", "/tmp/file.txt")
	if !canRead {
		t.Error("should still have read permission in read_only mode")
	}
	t.Log("✓ Mode enforcement verified")

	// Step 9: Test content blocks
	t.Log("9. Testing content blocks...")
	contentHandler := acp.NewContentBlockHandler()
	textBlock := contentHandler.CreateTextBlock("Hello, Agent!")
	if textBlock.Type != "text" {
		t.Error("content block type mismatch")
	}
	t.Log("✓ Content block created")

	// Step 10: Test execution plan
	t.Log("10. Testing execution plans...")
	planHandler := acp.NewExecutionPlanHandler()
	plan := planHandler.CreateExecutionPlan()
	planHandler.AddStep(plan, "Analyze code", "high", "pending")
	planHandler.AddStep(plan, "Run tests", "high", "pending")
	planHandler.AddStep(plan, "Deploy", "medium", "pending")

	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(plan.Steps))
	}
	t.Log("✓ Execution plan created with 3 steps")

	// Step 11: Update plan progress
	t.Log("11. Testing plan progress tracking...")
	planHandler.UpdateStepStatus(plan, 0, "completed")
	planHandler.UpdateStepStatus(plan, 1, "in_progress")

	if plan.Steps[0].Status != "completed" {
		t.Error("step 0 should be completed")
	}
	if plan.Steps[1].Status != "in_progress" {
		t.Error("step 1 should be in_progress")
	}
	if plan.Status != "in_progress" {
		t.Error("plan status should be in_progress")
	}
	t.Log("✓ Plan progress updated")

	// Step 12: Test notifications
	t.Log("12. Testing notification system...")
	notifHandler := acp.NewNotificationHandler()
	notif := notifHandler.CreateContentChunkNotification(
		session.SessionID,
		textBlock,
	)
	if notif.Method != "session/update" {
		t.Error("notification method should be session/update")
	}
	t.Log("✓ Notification created")

	// Step 13: Test MCP bridge
	t.Log("13. Testing MCP bridge...")
	bridge := acp.NewMCPBridge()
	bridge.RegisterMCPServer("my-tool", map[string]string{
		"type":    "stdio",
		"command": "my-tool",
	})

	servers := bridge.ListMCPServers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 MCP server, got %d", len(servers))
	}
	t.Log("✓ MCP server registered")

	// Step 14: Cleanup
	t.Log("14. Cleaning up resources...")
	tm.Release(term.ID)
	sm.DeleteSession(session.SessionID)
	pm.ClearPendingRequests(session.SessionID)
	t.Log("✓ Resources cleaned up")

	t.Log("\n✅ Complete ACP workflow test passed!")
}

// TestACPMessageSerialization tests JSON-RPC message handling
func TestACPMessageSerialization(t *testing.T) {
	t.Log("Testing JSON-RPC message serialization...")

	// Create a request
	request := acp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "session/new",
		Params: acp.SessionNewParams{
			ProfileID: "test-profile",
			Mode:      acp.SessionModeDefault,
		},
		ID: 1,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Unmarshal back
	var unmarshaled acp.JSONRPCRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if unmarshaled.Method != request.Method {
		t.Error("method mismatch after serialization")
	}

	t.Log("✓ JSON-RPC serialization works correctly")
}

// TestACPCapabilityNegotiation tests capability exchange
func TestACPCapabilityNegotiation(t *testing.T) {
	t.Log("Testing capability negotiation...")

	// Simulate agent capabilities
	agentCaps := map[string]interface{}{
		"filesystem": map[string]interface{}{
			"readTextFile":  true,
			"writeTextFile": true,
		},
		"terminal": map[string]interface{}{
			"create": true,
		},
	}

	// Simulate client request
	clientReq := acp.CapabilityRequest{
		Filesystem: map[string]bool{
			"readTextFile": true,
		},
		Terminal: true,
	}

	// Negotiate
	negotiated := acp.NegotiateCapabilities(agentCaps, clientReq)

	if _, ok := negotiated["filesystem"]; !ok {
		t.Error("filesystem capabilities should be negotiated")
	}

	if _, ok := negotiated["terminal"]; !ok {
		t.Error("terminal capabilities should be negotiated")
	}

	t.Log("✓ Capability negotiation successful")
}

// TestACPErrorHandling tests error responses
func TestACPErrorHandling(t *testing.T) {
	t.Log("Testing error handling...")

	// Create error response
	errResp := acp.ErrPermissionDenied

	if errResp.Code != acp.ErrCodePermissionDenied {
		t.Error("error code mismatch")
	}

	if errResp.Message != "permission denied" {
		t.Error("error message mismatch")
	}

	t.Log("✓ Error handling works correctly")
}

// TestACPSessionModeTransitions tests session mode changes
func TestACPSessionModeTransitions(t *testing.T) {
	t.Log("Testing session mode transitions...")

	coreSession := &protocol.SessionState{
		SessionID: "sess_mode_test",
		ProfileID: "test",
		Metadata:  make(map[string]string),
	}

	sm := acp.NewSessionManager()
	session := sm.CreateSession("sess_mode_test", "test", acp.SessionModeDefault, nil, coreSession)

	// Test transitions
	modes := []acp.SessionMode{
		acp.SessionModeAutoApprove,
		acp.SessionModeReadOnly,
		acp.SessionModeDefault,
	}

	for _, mode := range modes {
		sm.SetSessionMode(session.SessionID, mode)
		updated, _ := sm.GetSession(session.SessionID)

		if updated.Mode != mode {
			t.Errorf("mode transition to %s failed", mode)
		}

		t.Logf("✓ Transitioned to %s", mode)
	}

	t.Log("✓ All mode transitions successful")
}

