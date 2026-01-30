package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	APSHomeDir     = ".aps"
	SessionsDir    = "sessions"
	RegistryFile   = "registry.json"
	DefaultTimeout = 30 * time.Minute
)

type SessionStatus string
type SessionTier string

const (
	SessionActive   SessionStatus = "active"
	SessionInactive SessionStatus = "inactive"
	SessionErrored  SessionStatus = "errored"
)

const (
	TierBasic    SessionTier = "basic"
	TierStandard SessionTier = "standard"
	TierPremium  SessionTier = "premium"
)

type SessionInfo struct {
	ID          string            `json:"id"`
	ProfileID   string            `json:"profile_id"`
	ProfileDir  string            `json:"profile_dir,omitempty"`
	Command     string            `json:"command"`
	PID         int               `json:"pid"`
	Status      SessionStatus     `json:"status"`
	Tier        SessionTier       `json:"tier,omitempty"`
	TmuxSocket  string            `json:"tmux_socket,omitempty"`
	TmuxSession string            `json:"tmux_session,omitempty"`
	ContainerID string            `json:"container_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	LastSeenAt  time.Time         `json:"last_seen_at"`
	Environment map[string]string `json:"environment,omitempty"`
}

type SessionRegistry struct {
	sessions map[string]*SessionInfo
	mu       sync.RWMutex
}

var registry *SessionRegistry
var once sync.Once

func GetRegistry() *SessionRegistry {
	once.Do(func() {
		registry = &SessionRegistry{
			sessions: make(map[string]*SessionInfo),
		}
		if err := registry.LoadFromDisk(); err != nil {
			fmt.Printf("Warning: failed to load session registry: %v\n", err)
		}
	})
	return registry
}

func (r *SessionRegistry) Register(session *SessionInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ID]; exists {
		return fmt.Errorf("session %s already exists", session.ID)
	}

	session.CreatedAt = time.Now()
	session.LastSeenAt = time.Now()
	r.sessions[session.ID] = session

	return nil
}

func (r *SessionRegistry) Unregister(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	delete(r.sessions, sessionID)
	return nil
}

func (r *SessionRegistry) Get(sessionID string) (*SessionInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

func (r *SessionRegistry) List() []*SessionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

func (r *SessionRegistry) ListByProfile(profileID string) []*SessionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*SessionInfo, 0)
	for _, session := range r.sessions {
		if session.ProfileID == profileID {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

func (r *SessionRegistry) UpdateStatus(sessionID string, status SessionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Status = status
	session.LastSeenAt = time.Now()

	return nil
}

func (r *SessionRegistry) UpdateHeartbeat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.LastSeenAt = time.Now()
	return nil
}

func (r *SessionRegistry) CleanupInactive(timeout time.Duration) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var expired []string
	now := time.Now()

	for id, session := range r.sessions {
		if now.Sub(session.LastSeenAt) > timeout {
			expired = append(expired, id)
			delete(r.sessions, id)
		}
	}

	return expired
}

// SaveToDisk persists the session registry to disk
func (r *SessionRegistry) SaveToDisk() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	apsDir := filepath.Join(home, APSHomeDir)
	sessionsDir := filepath.Join(apsDir, SessionsDir)

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	registryPath := filepath.Join(sessionsDir, RegistryFile)
	data, err := json.MarshalIndent(r.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// LoadFromDisk loads the session registry from disk
func (r *SessionRegistry) LoadFromDisk() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	registryPath := filepath.Join(home, APSHomeDir, SessionsDir, RegistryFile)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			r.sessions = make(map[string]*SessionInfo)
			return nil
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	if err := json.Unmarshal(data, &r.sessions); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	return nil
}

// ListByStatus filters sessions by status
func (r *SessionRegistry) ListByStatus(status SessionStatus) []*SessionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*SessionInfo, 0)
	for _, session := range r.sessions {
		if session.Status == status {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// ListByTier filters sessions by tier
func (r *SessionRegistry) ListByTier(tier SessionTier) []*SessionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*SessionInfo, 0)
	for _, session := range r.sessions {
		if session.Tier == tier {
			sessions = append(sessions, session)
		}
	}

	return sessions
}
