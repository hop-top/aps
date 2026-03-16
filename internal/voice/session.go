package voice

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionState string

const (
	SessionStateActive SessionState = "active"
	SessionStateClosed SessionState = "closed"
)

// Session tracks one active voice session.
type Session struct {
	ID          string
	ProfileID   string
	ChannelType string
	State       SessionState
	CreatedAt   time.Time
}

// SessionManager tracks all active voice sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

// Create registers a new active session and returns it.
func (sm *SessionManager) Create(profileID, channelType string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s := &Session{
		ID:          uuid.New().String(),
		ProfileID:   profileID,
		ChannelType: channelType,
		State:       SessionStateActive,
		CreatedAt:   time.Now(),
	}
	sm.sessions[s.ID] = s
	return s
}

// Get returns a session by ID.
func (sm *SessionManager) Get(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("voice session %q not found", id)
	}
	return s, nil
}

// List returns all sessions.
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		out = append(out, s)
	}
	return out
}

// Close marks a session as closed.
func (sm *SessionManager) Close(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("voice session %q not found", id)
	}
	s.State = SessionStateClosed
	return nil
}

// SwitchProfile updates the profile for an active session (mid-session switch).
func (sm *SessionManager) SwitchProfile(id, newProfileID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("voice session %q not found", id)
	}
	s.ProfileID = newProfileID
	return nil
}
