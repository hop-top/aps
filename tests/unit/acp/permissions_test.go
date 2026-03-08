package acp

import (
	"testing"

	"hop.top/aps/internal/acp"
)

// TestPermissionManager tests permission request management
func TestPermissionManagerCreate(t *testing.T) {
	pm := acp.NewPermissionManager()

	req, err := pm.RequestPermission("sess_123", "fs/write", "/tmp/file.txt")
	if err != nil {
		t.Fatalf("failed to create permission request: %v", err)
	}

	if req.SessionID != "sess_123" {
		t.Errorf("expected session ID 'sess_123', got '%s'", req.SessionID)
	}

	if req.Operation != "fs/write" {
		t.Errorf("expected operation 'fs/write', got '%s'", req.Operation)
	}

	if req.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", req.Status)
	}
}

// TestPermissionManagerGrant tests granting a permission
func TestPermissionManagerGrant(t *testing.T) {
	pm := acp.NewPermissionManager()

	req, _ := pm.RequestPermission("sess_123", "fs/write", "/tmp/file.txt")

	if err := pm.GrantPermission(req.ID); err != nil {
		t.Fatalf("failed to grant permission: %v", err)
	}

	retrieved, _ := pm.GetPermissionRequest(req.ID)
	if retrieved.Status != "approved" {
		t.Errorf("expected status 'approved', got '%s'", retrieved.Status)
	}

	if !retrieved.Decision {
		t.Error("expected decision to be true")
	}
}

// TestPermissionManagerDeny tests denying a permission
func TestPermissionManagerDeny(t *testing.T) {
	pm := acp.NewPermissionManager()

	req, _ := pm.RequestPermission("sess_123", "terminal/create", "bash")

	if err := pm.DenyPermission(req.ID); err != nil {
		t.Fatalf("failed to deny permission: %v", err)
	}

	retrieved, _ := pm.GetPermissionRequest(req.ID)
	if retrieved.Status != "denied" {
		t.Errorf("expected status 'denied', got '%s'", retrieved.Status)
	}

	if retrieved.Decision {
		t.Error("expected decision to be false")
	}
}

// TestPermissionRules tests permission rules
func TestPermissionRulesReadPath(t *testing.T) {
	rules := acp.NewDefaultPermissionRules()

	// Should allow reading from /home
	if !rules.CheckReadPermission("/home/user/file.txt") {
		t.Error("should allow reading from /home")
	}

	// Should deny reading /etc/shadow
	if rules.CheckReadPermission("/etc/shadow") {
		t.Error("should deny reading /etc/shadow")
	}
}

// TestPermissionRulesWritePath tests write path permissions
func TestPermissionRulesWritePath(t *testing.T) {
	rules := acp.NewDefaultPermissionRules()

	// Should allow writing to /tmp
	if !rules.CheckWritePermission("/tmp/file.txt") {
		t.Error("should allow writing to /tmp")
	}

	// Should deny writing to /etc
	if rules.CheckWritePermission("/etc/config.txt") {
		t.Error("should deny writing to /etc")
	}
}

// TestPermissionRulesCommand tests command permissions
func TestPermissionRulesCommand(t *testing.T) {
	rules := acp.NewDefaultPermissionRules()

	// Should allow ls command
	if !rules.CheckCommandPermission("ls") {
		t.Error("should allow 'ls' command")
	}

	// Should deny rm -rf
	if rules.CheckCommandPermission("rm -rf") {
		t.Error("should deny 'rm -rf' command")
	}
}

// TestClearPendingRequests tests clearing requests for a session
func TestClearPendingRequests(t *testing.T) {
	pm := acp.NewPermissionManager()

	pm.RequestPermission("sess_1", "fs/write", "/tmp/file1.txt")
	pm.RequestPermission("sess_1", "fs/write", "/tmp/file2.txt")
	pm.RequestPermission("sess_2", "fs/write", "/tmp/file3.txt")

	pm.ClearPendingRequests("sess_1")

	// sess_2 request should still be there
	req, err := pm.RequestPermission("sess_2", "fs/write", "/tmp/file3.txt")
	if err != nil {
		t.Fatalf("session 2 request should still be available: %v", err)
	}

	if req == nil {
		t.Error("expected to create new request for sess_2")
	}
}
