package collaboration

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionManager tracks agent sessions within workspaces.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // session ID -> session
}

// Session represents an agent's active connection to a workspace.
type Session struct {
	ID          string       `json:"id" yaml:"id"`
	WorkspaceID string       `json:"workspace_id" yaml:"workspace_id"`
	ProfileID   string       `json:"profile_id" yaml:"profile_id"`
	State       SessionState `json:"state" yaml:"state"`
	JoinedAt    time.Time    `json:"joined_at" yaml:"joined_at"`
	LastSeen    time.Time    `json:"last_seen" yaml:"last_seen"`
	ExpiresAt   time.Time    `json:"expires_at" yaml:"expires_at"`
}

// NewSessionManager creates a session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session for an agent joining a workspace.
func (sm *SessionManager) CreateSession(workspaceID, profileID string, timeout time.Duration) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check for existing active session
	for _, s := range sm.sessions {
		if s.WorkspaceID == workspaceID && s.ProfileID == profileID && s.State == SessionActive {
			return nil, fmt.Errorf("agent %q already has an active session in workspace %q", profileID, workspaceID)
		}
	}

	now := time.Now()
	session := &Session{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		ProfileID:   profileID,
		State:       SessionActive,
		JoinedAt:    now,
		LastSeen:    now,
		ExpiresAt:   now.Add(timeout),
	}

	sm.sessions[session.ID] = session
	return session, nil
}

// Heartbeat updates a session's last-seen time and extends its expiry.
func (sm *SessionManager) Heartbeat(sessionID string, timeout time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}
	if session.State != SessionActive {
		return fmt.Errorf("session %q is not active (state: %s)", sessionID, session.State)
	}

	now := time.Now()
	session.LastSeen = now
	session.ExpiresAt = now.Add(timeout)
	return nil
}

// CloseSession marks a session as closed.
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	session.State = SessionClosed
	return nil
}

// GetSession returns a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	return session, nil
}

// GetAgentSession returns the active session for an agent in a workspace.
func (sm *SessionManager) GetAgentSession(workspaceID, profileID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, s := range sm.sessions {
		if s.WorkspaceID == workspaceID && s.ProfileID == profileID && s.State == SessionActive {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no active session for agent %q in workspace %q", profileID, workspaceID)
}

// ListSessions returns all sessions for a workspace.
func (sm *SessionManager) ListSessions(workspaceID string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*Session
	for _, s := range sm.sessions {
		if s.WorkspaceID == workspaceID {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// ListActiveSessions returns active sessions for a workspace.
func (sm *SessionManager) ListActiveSessions(workspaceID string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*Session
	for _, s := range sm.sessions {
		if s.WorkspaceID == workspaceID && s.State == SessionActive {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// ExpireSessions finds and closes sessions that have exceeded their timeout.
// Returns the profile IDs of expired agents.
func (sm *SessionManager) ExpireSessions(workspaceID string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	var expired []string

	for _, s := range sm.sessions {
		if s.WorkspaceID == workspaceID && s.State == SessionActive && now.After(s.ExpiresAt) {
			s.State = SessionClosed
			expired = append(expired, s.ProfileID)
		}
	}

	return expired
}

// CleanupClosed removes all closed sessions for a workspace.
func (sm *SessionManager) CleanupClosed(workspaceID string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	for id, s := range sm.sessions {
		if s.WorkspaceID == workspaceID && s.State == SessionClosed {
			delete(sm.sessions, id)
			count++
		}
	}
	return count
}
