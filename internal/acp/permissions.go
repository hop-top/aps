package acp

import (
	"fmt"
	"sync"
)

// PermissionManager handles permission requests and decisions
type PermissionManager struct {
	pendingRequests map[string]*PermissionRequest
	mu              sync.RWMutex
}

// PermissionRequest represents a pending permission request
type PermissionRequest struct {
	ID        string
	SessionID string
	Operation string
	Resource  string
	Status    string // "pending", "approved", "denied"
	Decision  bool
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		pendingRequests: make(map[string]*PermissionRequest),
	}
}

// RequestPermission creates a new permission request
func (pm *PermissionManager) RequestPermission(sessionID string, operation string, resource string) (*PermissionRequest, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if operation == "" {
		return nil, fmt.Errorf("operation cannot be empty")
	}

	reqID := generatePermissionRequestID()
	req := &PermissionRequest{
		ID:        reqID,
		SessionID: sessionID,
		Operation: operation,
		Resource:  resource,
		Status:    "pending",
	}

	pm.pendingRequests[reqID] = req
	return req, nil
}

// GrantPermission grants a pending permission request
func (pm *PermissionManager) GrantPermission(requestID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	req, exists := pm.pendingRequests[requestID]
	if !exists {
		return fmt.Errorf("permission request not found: %s", requestID)
	}

	req.Status = "approved"
	req.Decision = true
	return nil
}

// DenyPermission denies a pending permission request
func (pm *PermissionManager) DenyPermission(requestID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	req, exists := pm.pendingRequests[requestID]
	if !exists {
		return fmt.Errorf("permission request not found: %s", requestID)
	}

	req.Status = "denied"
	req.Decision = false
	return nil
}

// GetPermissionRequest retrieves a permission request
func (pm *PermissionManager) GetPermissionRequest(requestID string) (*PermissionRequest, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	req, exists := pm.pendingRequests[requestID]
	if !exists {
		return nil, fmt.Errorf("permission request not found: %s", requestID)
	}

	return req, nil
}

// ClearPendingRequests removes all pending requests for a session
func (pm *PermissionManager) ClearPendingRequests(sessionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for id, req := range pm.pendingRequests {
		if req.SessionID == sessionID {
			delete(pm.pendingRequests, id)
		}
	}
}

// PermissionRules defines default permission rules
type PermissionRules struct {
	// Filesystem rules
	AllowedReadPaths  []string
	DeniedReadPaths   []string
	AllowedWritePaths []string
	DeniedWritePaths  []string

	// Terminal rules
	AllowedCommands []string
	DeniedCommands  []string

	// General
	RequireApproval bool
}

// NewDefaultPermissionRules creates default permission rules
func NewDefaultPermissionRules() *PermissionRules {
	return &PermissionRules{
		AllowedReadPaths: []string{
			"/home",
			"/tmp",
		},
		DeniedReadPaths: []string{
			"/etc/passwd",
			"/etc/shadow",
			"/.env",
			"/credentials",
		},
		AllowedWritePaths: []string{
			"/tmp",
		},
		DeniedWritePaths: []string{
			"/etc",
			"/sys",
			"/proc",
			"/.env",
		},
		AllowedCommands: []string{
			"ls",
			"cat",
			"echo",
		},
		DeniedCommands: []string{
			"rm -rf",
			"mkfs",
			"dd",
		},
		RequireApproval: true,
	}
}

// CheckReadPermission checks if a path can be read
func (pr *PermissionRules) CheckReadPermission(path string) bool {
	// Check denied paths first
	for _, denied := range pr.DeniedReadPaths {
		if pathMatches(path, denied) {
			return false
		}
	}

	// Check allowed paths
	if len(pr.AllowedReadPaths) > 0 {
		for _, allowed := range pr.AllowedReadPaths {
			if pathMatches(path, allowed) {
				return true
			}
		}
		return false
	}

	return true // Default allow if no rules
}

// CheckWritePermission checks if a path can be written
func (pr *PermissionRules) CheckWritePermission(path string) bool {
	// Check denied paths first
	for _, denied := range pr.DeniedWritePaths {
		if pathMatches(path, denied) {
			return false
		}
	}

	// Check allowed paths
	if len(pr.AllowedWritePaths) > 0 {
		for _, allowed := range pr.AllowedWritePaths {
			if pathMatches(path, allowed) {
				return true
			}
		}
		return false
	}

	return true // Default allow if no rules
}

// CheckCommandPermission checks if a command can be executed
func (pr *PermissionRules) CheckCommandPermission(cmd string) bool {
	// Check denied commands first
	for _, denied := range pr.DeniedCommands {
		if cmd == denied {
			return false
		}
	}

	// Check allowed commands
	if len(pr.AllowedCommands) > 0 {
		for _, allowed := range pr.AllowedCommands {
			if cmd == allowed {
				return true
			}
		}
		return false
	}

	return true // Default allow if no rules
}

// pathMatches checks if a path matches a pattern (simple implementation)
func pathMatches(path string, pattern string) bool {
	// Simple prefix matching
	if pattern == "*" {
		return true
	}

	// Exact match
	if path == pattern {
		return true
	}

	// Prefix match
	if len(path) > len(pattern) && path[:len(pattern)+1] == pattern+"/" {
		return true
	}

	return false
}

// generatePermissionRequestID generates a unique permission request ID
func generatePermissionRequestID() string {
	return "perm_" + generateID()
}
