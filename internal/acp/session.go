package acp

import (
	"fmt"
	"sync"
	"time"

	"hop.top/aps/internal/core/protocol"
)

// SessionManager manages ACP sessions
type SessionManager struct {
	sessions map[string]*ACPSession
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ACPSession),
	}
}

// CreateSession creates a new ACP session
func (sm *SessionManager) CreateSession(sessionID string, profileID string, mode SessionMode, clientCaps map[string]interface{}, coreSession *protocol.SessionState) *ACPSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if mode == "" {
		mode = SessionModeDefault
	}

	session := &ACPSession{
		SessionID:          sessionID,
		ProfileID:          profileID,
		Mode:               mode,
		CoreSession:        coreSession,
		ClientCapabilities: clientCaps,
		AgentCapabilities:  make(map[string]interface{}),
		PermissionRules:    make([]PermissionRule, 0),
		CreatedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	sm.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*ACPSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// UpdateSession updates a session
func (sm *SessionManager) UpdateSession(sessionID string, fn func(*ACPSession) error) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if err := fn(session); err != nil {
		return err
	}

	session.LastActivity = time.Now()
	return nil
}

// SetSessionMode updates the session mode
func (sm *SessionManager) SetSessionMode(sessionID string, mode SessionMode) error {
	return sm.UpdateSession(sessionID, func(s *ACPSession) error {
		s.Mode = mode
		return nil
	})
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
	return nil
}

// ListSessions returns all sessions for a profile
func (sm *SessionManager) ListSessions(profileID string) []*ACPSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*ACPSession
	for _, session := range sm.sessions {
		if session.ProfileID == profileID {
			result = append(result, session)
		}
	}
	return result
}

// GetPermission checks if an operation is allowed
func (sm *SessionManager) GetPermission(sessionID string, operation string, resource string) (bool, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return false, err
	}

	return session.HasPermission(operation, resource), nil
}

// RequestPermission checks if permission is needed and records it
func (sm *SessionManager) RequestPermission(sessionID string, operation string, resource string) bool {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return false
	}

	// Read-only mode denies all write operations
	if session.Mode == SessionModeReadOnly {
		switch operation {
		case "fs/write", "fs/write_text_file", "terminal/create", "terminal/kill":
			return false
		}
	}

	// Auto-approve mode allows everything
	if session.Mode == SessionModeAutoApprove {
		return true
	}

	// Default mode checks rules
	return session.HasPermission(operation, resource)
}

// HasPermission checks if a session has permission for an operation
func (s *ACPSession) HasPermission(operation string, resource string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check permission rules first
	for _, rule := range s.PermissionRules {
		if rule.Operation == operation {
			// Check path pattern if applicable
			if rule.PathPattern != "" && resource != "" {
				// Simple substring match for now
				if rule.PathPattern == "*" || rule.PathPattern == resource {
					return rule.Allowed
				}
			} else {
				return rule.Allowed
			}
		}
	}

	// Mode-based permission logic
	switch s.Mode {
	case SessionModeAutoApprove:
		// Allow all operations
		return true

	case SessionModeReadOnly:
		// Allow only read operations
		switch operation {
		case "fs/read", "fs/read_text_file":
			return true
		case "terminal/output", "terminal/wait_for_exit":
			return true
		default:
			return false
		}

	case SessionModeDefault:
		// Allow non-destructive operations
		switch operation {
		case "fs/read", "fs/read_text_file":
			return true
		case "terminal/output", "terminal/wait_for_exit":
			return true
		default:
			return false
		}
	}

	return false
}

// AddPermissionRule adds a permission rule to the session
func (s *ACPSession) AddPermissionRule(rule PermissionRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PermissionRules = append(s.PermissionRules, rule)
}

// UpdateLastActivity updates the session's last activity timestamp
func (s *ACPSession) UpdateLastActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivity = time.Now()
}

// GetInfo returns public session information
func (s *ACPSession) GetInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"sessionId":             s.SessionID,
		"profileId":             s.ProfileID,
		"mode":                  s.Mode,
		"createdAt":             s.CreatedAt,
		"lastActivity":          s.LastActivity,
		"clientCapabilities":    s.ClientCapabilities,
		"agentCapabilities":     s.AgentCapabilities,
	}
}
