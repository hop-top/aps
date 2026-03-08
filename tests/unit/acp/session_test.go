package acp

import (
	"testing"
	"time"

	"hop.top/aps/internal/acp"
	"hop.top/aps/internal/core/protocol"
)

// TestSessionManagerCreate tests session creation
func TestSessionManagerCreate(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
		Metadata: map[string]string{
			"acp_mode": "default",
		},
	}

	session := sm.CreateSession(
		coreSession.SessionID,
		coreSession.ProfileID,
		acp.SessionModeDefault,
		nil,
		coreSession,
	)

	if session.SessionID != "sess_123" {
		t.Errorf("expected session ID 'sess_123', got '%s'", session.SessionID)
	}

	if session.Mode != acp.SessionModeDefault {
		t.Errorf("expected mode 'default', got '%s'", session.Mode)
	}
}

// TestSessionManagerGet tests session retrieval
func TestSessionManagerGet(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	sm.CreateSession("sess_123", "test-profile", acp.SessionModeDefault, nil, coreSession)

	retrieved, err := sm.GetSession("sess_123")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if retrieved.SessionID != "sess_123" {
		t.Errorf("expected session ID 'sess_123', got '%s'", retrieved.SessionID)
	}
}

// TestSessionManagerSetMode tests mode updates
func TestSessionManagerSetMode(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	sm.CreateSession("sess_123", "test-profile", acp.SessionModeDefault, nil, coreSession)

	if err := sm.SetSessionMode("sess_123", acp.SessionModeAutoApprove); err != nil {
		t.Fatalf("failed to set mode: %v", err)
	}

	session, _ := sm.GetSession("sess_123")
	if session.Mode != acp.SessionModeAutoApprove {
		t.Errorf("expected mode 'auto_approve', got '%s'", session.Mode)
	}
}

// TestSessionManagerDelete tests session deletion
func TestSessionManagerDelete(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	sm.CreateSession("sess_123", "test-profile", acp.SessionModeDefault, nil, coreSession)

	if err := sm.DeleteSession("sess_123"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	_, err := sm.GetSession("sess_123")
	if err == nil {
		t.Error("expected error getting deleted session")
	}
}

// TestSessionManagerListSessions tests session listing
func TestSessionManagerListSessions(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession1 := &protocol.SessionState{
		SessionID: "sess_1",
		ProfileID: "profile_a",
	}
	coreSession2 := &protocol.SessionState{
		SessionID: "sess_2",
		ProfileID: "profile_a",
	}
	coreSession3 := &protocol.SessionState{
		SessionID: "sess_3",
		ProfileID: "profile_b",
	}

	sm.CreateSession("sess_1", "profile_a", acp.SessionModeDefault, nil, coreSession1)
	sm.CreateSession("sess_2", "profile_a", acp.SessionModeDefault, nil, coreSession2)
	sm.CreateSession("sess_3", "profile_b", acp.SessionModeDefault, nil, coreSession3)

	sessions := sm.ListSessions("profile_a")
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions for profile_a, got %d", len(sessions))
	}
}

// TestSessionReadOnlyMode tests read-only mode enforcement
func TestSessionReadOnlyMode(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	session := sm.CreateSession("sess_123", "test-profile", acp.SessionModeReadOnly, nil, coreSession)

	// Read operations should be allowed
	if !session.HasPermission("fs/read", "") {
		t.Error("read operation should be allowed in read-only mode")
	}

	// Write operations should be denied
	if session.HasPermission("fs/write", "") {
		t.Error("write operation should be denied in read-only mode")
	}
}

// TestSessionAutoApproveMode tests auto-approve mode
func TestSessionAutoApproveMode(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	sm.CreateSession("sess_123", "test-profile", acp.SessionModeAutoApprove, nil, coreSession)

	// All operations should be allowed
	allowed := sm.RequestPermission("sess_123", "fs/write", "/tmp/file.txt")
	if !allowed {
		t.Error("write operation should be allowed in auto-approve mode")
	}
}

// TestSessionLastActivity tests activity tracking
func TestSessionLastActivity(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	session := sm.CreateSession("sess_123", "test-profile", acp.SessionModeDefault, nil, coreSession)
	createdAt := session.LastActivity

	// Wait a tiny bit
	time.Sleep(10 * time.Millisecond)

	// Update activity
	session.UpdateLastActivity()

	if session.LastActivity.Equal(createdAt) {
		t.Error("last activity should be updated")
	}
}

// TestSessionInfo tests GetInfo method
func TestSessionInfo(t *testing.T) {
	sm := acp.NewSessionManager()

	coreSession := &protocol.SessionState{
		SessionID: "sess_123",
		ProfileID: "test-profile",
	}

	session := sm.CreateSession("sess_123", "test-profile", acp.SessionModeDefault, nil, coreSession)

	info := session.GetInfo()

	if info["sessionId"] != "sess_123" {
		t.Error("info should contain session ID")
	}

	if info["mode"] != acp.SessionModeDefault {
		t.Error("info should contain session mode")
	}
}
