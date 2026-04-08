// Package session manages the runtime session registry for APS profiles.
//
// Write-through contract: all mutator methods (Register, Unregister,
// UpdateStatus, UpdateHeartbeat, UpdateSessionMetadata, CleanupInactive)
// persist the registry to disk before returning. Persistence failures
// are surfaced as errors and the in-memory mutation is rolled back so
// the in-memory state always matches what is on disk after any
// successful mutator return.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/logging"
)

// Directory and file constants for the on-disk session registry.
const (
	// APSHomeDir is the user-home subdirectory used by APS to store
	// per-user state when no explicit data path is configured.
	APSHomeDir = ".aps"
	// SessionsDir is the subdirectory under the APS data dir holding
	// session-related artifacts including the registry file.
	SessionsDir = "sessions"
	// RegistryFile is the JSON file inside SessionsDir that persists
	// the session registry between process invocations.
	RegistryFile = "registry.json"

	// DefaultTimeout is how long a session may be inactive (no heartbeat
	// activity) before the background reaper removes it from the registry.
	DefaultTimeout = 30 * time.Minute

	// ReaperTickInterval is how often the background reaper wakes to scan
	// for sessions past DefaultTimeout. Must be shorter than DefaultTimeout
	// so reaping is reasonably prompt after expiry.
	ReaperTickInterval = 5 * time.Minute
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
	WorkspaceID string            `json:"workspace_id,omitempty"`
}

type SessionRegistry struct {
	sessions map[string]*SessionInfo
	mu       sync.RWMutex
}

var registry *SessionRegistry
var once sync.Once

// NewForTesting returns a fresh, empty SessionRegistry that does not
// share state with the package singleton. It is intended for tests
// that need isolated registry state per test. The caller is
// responsible for persistence (the registry will still call
// saveToDiskLocked on mutations — set APS_DATA_PATH to a tmp dir).
func NewForTesting() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]*SessionInfo),
	}
}

func GetRegistry() *SessionRegistry {
	once.Do(func() {
		registry = &SessionRegistry{
			sessions: make(map[string]*SessionInfo),
		}
		if err := registry.LoadFromDisk(); err != nil {
			fmt.Printf("Warning: failed to load session registry: %v\n", err)
		}
		startReaper(context.Background(), registry, ReaperTickInterval)
	})
	return registry
}

// startReaper spawns a background goroutine that periodically calls
// CleanupInactive on the registry, removing any session whose
// LastSeenAt is older than DefaultTimeout.
//
// Cancellation contract: the production singleton (GetRegistry) calls
// this with context.Background() — the reaper runs for the lifetime
// of the process and is reaped by process exit. Tests that need to
// exercise the reaper should pass their own cancellable context (and
// a short tick interval) so they can stop the goroutine cleanly.
func startReaper(ctx context.Context, r *SessionRegistry, tick time.Duration) {
	go func() {
		logger := logging.GetLogger()
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				expired, err := r.CleanupInactive(DefaultTimeout)
				if err != nil {
					logger.Error("session reaper: cleanup failed", err)
					continue
				}
				if len(expired) > 0 {
					logger.Info("session reaper: removed inactive sessions",
						"count", len(expired),
						"ids", expired,
					)
				}
			}
		}
	}()
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

	if err := r.saveToDiskLocked(); err != nil {
		delete(r.sessions, session.ID)
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

func (r *SessionRegistry) Unregister(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	prev, existed := r.sessions[sessionID]
	delete(r.sessions, sessionID)

	if err := r.saveToDiskLocked(); err != nil {
		if existed {
			r.sessions[sessionID] = prev
		}
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
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

	prevStatus := session.Status
	prevSeen := session.LastSeenAt
	session.Status = status
	session.LastSeenAt = time.Now()

	if err := r.saveToDiskLocked(); err != nil {
		session.Status = prevStatus
		session.LastSeenAt = prevSeen
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

func (r *SessionRegistry) UpdateHeartbeat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	prevSeen := session.LastSeenAt
	session.LastSeenAt = time.Now()

	if err := r.saveToDiskLocked(); err != nil {
		session.LastSeenAt = prevSeen
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

// UpdateSessionMetadata merges the provided metadata into the session's
// Environment map and refreshes LastSeenAt. Persists to disk. Returns
// an error if the session does not exist or persistence fails, in
// which case the in-memory state is rolled back to its prior value.
func (r *SessionRegistry) UpdateSessionMetadata(sessionID string, metadata map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Capture for rollback.
	prevEnv := make(map[string]string, len(session.Environment))
	for k, v := range session.Environment {
		prevEnv[k] = v
	}
	prevSeen := session.LastSeenAt

	if session.Environment == nil {
		session.Environment = make(map[string]string)
	}
	for k, v := range metadata {
		session.Environment[k] = v
	}
	session.LastSeenAt = time.Now()

	if err := r.saveToDiskLocked(); err != nil {
		session.Environment = prevEnv
		session.LastSeenAt = prevSeen
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

// CleanupInactive removes any session whose LastSeenAt is older than
// the supplied timeout, persists the result to disk, and returns the
// IDs of the removed sessions. On persistence failure, all removals
// are rolled back and an error is returned alongside a nil expired
// slice so the caller cannot accidentally consume an inconsistent
// view.
//
// Sessions in the SessionErrored state are deliberately skipped: per
// the T3 design (docs/dev/agent-lifecycle.md), errored sessions remain
// in the registry indefinitely so operators can inspect them. They
// must be removed explicitly via Unregister.
//
// TODO: add a save-failure rollback test once a fault-injecting
// filesystem is available (see the TODO above the rollback tests).
func (r *SessionRegistry) CleanupInactive(timeout time.Duration) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var expired []string
	removed := make(map[string]*SessionInfo)
	now := time.Now()

	for id, session := range r.sessions {
		if session.Status == SessionErrored {
			continue
		}
		if now.Sub(session.LastSeenAt) > timeout {
			expired = append(expired, id)
			removed[id] = session
			delete(r.sessions, id)
		}
	}

	if err := r.saveToDiskLocked(); err != nil {
		for id, session := range removed {
			r.sessions[id] = session
		}
		return nil, fmt.Errorf("failed to persist session registry: %w", err)
	}
	return expired, nil
}

// SaveToDisk persists the session registry to disk. It acquires a read
// lock and delegates to saveToDiskLocked.
func (r *SessionRegistry) SaveToDisk() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.saveToDiskLocked()
}

// saveToDiskLocked writes the registry to disk WITHOUT acquiring r.mu.
// The caller MUST already hold r.mu (read or write). This exists so that
// mutator methods can persist while still holding their write lock,
// avoiding the deadlock that would occur if they called the public
// SaveToDisk (sync.RWMutex is not reentrant).
func (r *SessionRegistry) saveToDiskLocked() error {
	dataDir, err := core.GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	sessionsDir := filepath.Join(dataDir, SessionsDir)

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

	dataDir, err := core.GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	registryPath := filepath.Join(dataDir, SessionsDir, RegistryFile)

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
