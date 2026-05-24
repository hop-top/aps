// Package session manages the runtime session registry for APS profiles.
//
// Write-through contract: all mutator methods (Register, Unregister,
// UpdateStatus, UpdateHeartbeat, UpdateSessionMetadata, CleanupInactive)
// persist the registry to disk before returning. Persistence failures
// are surfaced as errors and the in-memory representation is rolled
// back so subsequent reads always see what is on disk after any
// successful mutator return.
//
// State is durably stored in a sqlite-backed kit/storage/kv table at
// <dataDir>/sessions/registry.db. Legacy registry.json files written
// by earlier aps releases are migrated into kv on first open and then
// removed.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hop.top/aps/internal/events"
	"hop.top/aps/internal/logging"
	"hop.top/kit/go/runtime/domain"
	"hop.top/kit/go/storage/kv"
)

// statusRules defines valid SessionStatus transitions enforced by
// SessionRegistry.UpdateStatus. The rules are deliberately strict:
//   - "" (initial) → any non-terminal status (sessions registered
//     without a status default to empty and need a first set)
//   - active   ↔ inactive (resume / pause)
//   - active/inactive → errored (terminal)
//   - errored is terminal: any further status change must go through
//     Unregister + Register
//
// Self-transitions are not allowed (a status set is meaningful only
// when it actually changes).
var statusRules = map[domain.State][]domain.State{
	domain.State(""):              {domain.State(SessionActive), domain.State(SessionInactive), domain.State(SessionErrored)},
	domain.State(SessionActive):   {domain.State(SessionInactive), domain.State(SessionErrored)},
	domain.State(SessionInactive): {domain.State(SessionActive), domain.State(SessionErrored)},
	domain.State(SessionErrored):  {}, // terminal
}

// statusMachine is the package-level state machine that enforces
// statusRules. Constructed once at init and used by checkTransition.
// Publisher is intentionally nil — aps emits its own richer
// aps.session.* events from the registry methods directly, so we
// don't need the generic domain.state.pre/post-transition events.
var statusMachine = domain.NewStateMachine(statusRules, nil)

// Directory and file constants for the on-disk session registry.
const (
	// APSHomeDir is the user-home subdirectory used by APS to store
	// per-user state when no explicit data path is configured.
	APSHomeDir = ".aps"
	// SessionsDir is the subdirectory under the APS data dir holding
	// session-related artifacts including the registry kv database.
	SessionsDir = "sessions"
	// RegistryFile is the legacy JSON file produced by pre-kv aps
	// releases. Kept exported so the migration helper can locate it.
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
	Type        SessionType       `json:"type,omitempty"`
	TmuxSocket  string            `json:"tmux_socket,omitempty"`
	TmuxSession string            `json:"tmux_session,omitempty"`
	ContainerID string            `json:"container_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	LastSeenAt  time.Time         `json:"last_seen_at"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
}

// SessionRegistry is the runtime view of all known sessions, backed
// by a sqlite kv store. The embedded mutex guards compound operations
// that must observe a consistent slice of the store (register-if-
// absent, the CleanupInactive sweep, the metadata-merge read-modify-
// write); single-key reads delegate to the kv backend's own locking.
type SessionRegistry struct {
	store     kv.Store
	storeOnce sync.Once
	storeErr  error
	mu        sync.Mutex
}

var registry *SessionRegistry
var once sync.Once

// NewForTesting returns a fresh SessionRegistry that does not share
// state with the package singleton. The kv store is lazily opened on
// first use and honours APS_DATA_PATH so tests that set
// `t.Setenv("APS_DATA_PATH", t.TempDir())` get isolated state.
func NewForTesting() *SessionRegistry {
	return &SessionRegistry{}
}

func GetRegistry() *SessionRegistry {
	once.Do(func() {
		registry = &SessionRegistry{}
		if err := registry.ensureStore(); err != nil {
			fmt.Printf("Warning: failed to open session registry: %v\n", err)
		}
		startReaper(context.Background(), registry, ReaperTickInterval)
	})
	return registry
}

// reaperDisabled lets the CLI layer opt the in-process reaper out when
// it has wired the kit/runtime/job-driven sweep instead. The reaper
// goroutine and the job-poll sweep would otherwise race over the same
// session set. The setter is package-level (mirroring SetEventPublisher)
// and is read once at startReaper start time.
var reaperDisabled bool

// DisableInlineReaper turns the goroutine reaper into a no-op for any
// future GetRegistry call. Idempotent. The CLI calls this before the
// first registry access; library / test consumers leave it alone and
// get the in-process ticker for free.
func DisableInlineReaper() { reaperDisabled = true }

// startReaper spawns a background goroutine that periodically calls
// CleanupInactive on the registry, removing any session whose
// LastSeenAt is older than DefaultTimeout.
//
// Cancellation contract: the production singleton (GetRegistry) calls
// this with context.Background() — the reaper runs for the lifetime
// of the process and is reaped by process exit. Tests that need to
// exercise the reaper should pass their own cancellable context (and
// a short tick interval) so they can stop the goroutine cleanly.
//
// When the CLI has wired the durabletask job runner via
// DisableInlineReaper, this function returns without spawning the
// goroutine; the kit poller drives the sweep instead.
func startReaper(ctx context.Context, r *SessionRegistry, tick time.Duration) {
	if reaperDisabled {
		return
	}
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
	return r.RegisterWithContext(context.Background(), session)
}

// RegisterWithContext is the ctx-aware variant of Register; reads the
// audit note attached via policy.ContextAttrsKey by the CLI layer
// (T-1291) and surfaces it in the SessionStarted bus payload.
func (r *SessionRegistry) RegisterWithContext(ctx context.Context, session *SessionInfo) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.kvGetSession(ctx, session.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("session %s already exists", session.ID)
	}

	session.CreatedAt = time.Now()
	session.LastSeenAt = time.Now()
	if err := r.kvPutSessionLocked(ctx, session); err != nil {
		return fmt.Errorf("failed to persist session registry: %w", err)
	}

	publish(ctx, string(events.TopicSessionStarted), "", events.SessionStartedPayload{
		SessionID: session.ID,
		ProfileID: session.ProfileID,
		Command:   session.Command,
		PID:       session.PID,
		Tier:      string(session.Tier),
		Note:      noteFromContext(ctx),
	})
	return nil
}

func (r *SessionRegistry) Unregister(sessionID string) error {
	return r.UnregisterWithContext(context.Background(), sessionID)
}

// UnregisterWithContext is the ctx-aware variant; reads the audit note
// attached via policy.ContextAttrsKey by the CLI layer (T-1291) and
// surfaces it in the SessionStopped bus payload.
func (r *SessionRegistry) UnregisterWithContext(ctx context.Context, sessionID string) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	prev, err := r.kvGetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := r.kvDeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to persist session registry: %w", err)
	}

	if prev != nil {
		publish(ctx, string(events.TopicSessionStopped), "", events.SessionStoppedPayload{
			SessionID: sessionID,
			ProfileID: prev.ProfileID,
			Reason:    "unregister",
			Note:      noteFromContext(ctx),
		})
	}
	return nil
}

func (r *SessionRegistry) Get(sessionID string) (*SessionInfo, error) {
	if err := r.ensureStore(); err != nil {
		return nil, err
	}
	info, err := r.kvGetSession(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return info, nil
}

func (r *SessionRegistry) List() []*SessionInfo {
	if err := r.ensureStore(); err != nil {
		return nil
	}
	sessions, err := r.kvListSessions(context.Background())
	if err != nil {
		return nil
	}
	return sessions
}

func (r *SessionRegistry) ListByProfile(profileID string) []*SessionInfo {
	if err := r.ensureStore(); err != nil {
		return nil
	}
	all, err := r.kvListSessions(context.Background())
	if err != nil {
		return nil
	}
	out := make([]*SessionInfo, 0, len(all))
	for _, s := range all {
		if s.ProfileID == profileID {
			out = append(out, s)
		}
	}
	return out
}

// checkTransition validates a SessionStatus transition against the
// package state machine. Returns nil if allowed (including the no-op
// case where from==to — idempotent status sets are not state changes
// and should not error). Otherwise returns an error that wraps
// domain.ErrInvalidTransition (testable via errors.Is).
func (r *SessionRegistry) checkTransition(from, to SessionStatus) error {
	if from == to {
		return nil
	}
	// Pass nil context — domain.StateMachine.Transition only uses ctx
	// when a publisher is wired (which it isn't here).
	return statusMachine.Transition(nil, domain.State(from), domain.State(to), false) //nolint:staticcheck
}

func (r *SessionRegistry) UpdateStatus(sessionID string, status SessionStatus) error {
	return r.UpdateStatusWithContext(context.Background(), sessionID, status)
}

// UpdateStatusWithContext is the ctx-aware variant of UpdateStatus
// (T-1291). When the transition lands in a terminal state, the audit
// note attached to ctx via policy.ContextAttrsKey is surfaced in the
// SessionStopped payload.
func (r *SessionRegistry) UpdateStatusWithContext(ctx context.Context, sessionID string, status SessionStatus) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	session, err := r.kvGetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if err := r.checkTransition(session.Status, status); err != nil {
		return fmt.Errorf("session %s: %w", sessionID, err)
	}

	prevStatus := session.Status
	session.Status = status
	session.LastSeenAt = time.Now()

	if err := r.kvPutSessionLocked(ctx, session); err != nil {
		return fmt.Errorf("failed to persist session registry: %w", err)
	}

	if prevStatus != status && (status == SessionInactive || status == SessionErrored) {
		reason := "inactive"
		if status == SessionErrored {
			reason = "errored"
		}
		publish(ctx, string(events.TopicSessionStopped), "", events.SessionStoppedPayload{
			SessionID: sessionID,
			ProfileID: session.ProfileID,
			Reason:    reason,
			Note:      noteFromContext(ctx),
		})
	}
	return nil
}

func (r *SessionRegistry) UpdateHeartbeat(sessionID string) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	ctx := context.Background()
	r.mu.Lock()
	defer r.mu.Unlock()

	session, err := r.kvGetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.LastSeenAt = time.Now()

	if err := r.kvPutSessionLocked(ctx, session); err != nil {
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

// UpdateSessionMetadata merges the provided metadata into the session's
// Environment map and refreshes LastSeenAt. Persists to disk. Returns
// an error if the session does not exist or persistence fails.
func (r *SessionRegistry) UpdateSessionMetadata(sessionID string, metadata map[string]string) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	ctx := context.Background()
	r.mu.Lock()
	defer r.mu.Unlock()

	session, err := r.kvGetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if session.Environment == nil {
		session.Environment = make(map[string]string)
	}
	for k, v := range metadata {
		session.Environment[k] = v
	}
	session.LastSeenAt = time.Now()

	if err := r.kvPutSessionLocked(ctx, session); err != nil {
		return fmt.Errorf("failed to persist session registry: %w", err)
	}
	return nil
}

// CleanupInactive removes any session whose LastSeenAt is older than
// the supplied timeout, persists the result to disk, and returns the
// IDs of the removed sessions.
//
// Sessions in the SessionErrored state are deliberately skipped: per
// the T3 design (docs/dev/agent-lifecycle.md), errored sessions remain
// in the registry indefinitely so operators can inspect them. They
// must be removed explicitly via Unregister.
func (r *SessionRegistry) CleanupInactive(timeout time.Duration) ([]string, error) {
	if err := r.ensureStore(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	r.mu.Lock()
	defer r.mu.Unlock()

	sessions, err := r.kvListSessions(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var expired []string
	expiredInfos := make([]*SessionInfo, 0)
	for _, s := range sessions {
		if s.Status == SessionErrored {
			continue
		}
		if now.Sub(s.LastSeenAt) > timeout {
			if err := r.kvDeleteSession(ctx, s.ID); err != nil {
				return nil, fmt.Errorf("failed to persist session registry: %w", err)
			}
			expired = append(expired, s.ID)
			expiredInfos = append(expiredInfos, s)
		}
	}

	for _, s := range expiredInfos {
		publish(ctx, string(events.TopicSessionStopped), "", events.SessionStoppedPayload{
			SessionID: s.ID,
			ProfileID: s.ProfileID,
			Reason:    "expired",
		})
	}
	return expired, nil
}

// SaveToDisk is a no-op retained for backwards compatibility — the kv
// store persists every mutation synchronously, so explicit saves are
// unnecessary.
func (r *SessionRegistry) SaveToDisk() error {
	return r.ensureStore()
}

// LoadFromDisk is a no-op retained for backwards compatibility — the
// kv store is opened lazily and reflects the on-disk state on every
// read.
func (r *SessionRegistry) LoadFromDisk() error {
	return r.ensureStore()
}

// ListByStatus filters sessions by status.
func (r *SessionRegistry) ListByStatus(status SessionStatus) []*SessionInfo {
	if err := r.ensureStore(); err != nil {
		return nil
	}
	all, err := r.kvListSessions(context.Background())
	if err != nil {
		return nil
	}
	out := make([]*SessionInfo, 0, len(all))
	for _, s := range all {
		if s.Status == status {
			out = append(out, s)
		}
	}
	return out
}

// ListByTier filters sessions by tier.
func (r *SessionRegistry) ListByTier(tier SessionTier) []*SessionInfo {
	if err := r.ensureStore(); err != nil {
		return nil
	}
	all, err := r.kvListSessions(context.Background())
	if err != nil {
		return nil
	}
	out := make([]*SessionInfo, 0, len(all))
	for _, s := range all {
		if s.Tier == tier {
			out = append(out, s)
		}
	}
	return out
}

// ListByType filters sessions by SessionType. The empty SessionType
// (SessionTypeStandard) matches entries with no explicit type set —
// i.e. sessions persisted before the Type field existed are treated
// as standard.
func (r *SessionRegistry) ListByType(t SessionType) []*SessionInfo {
	if err := r.ensureStore(); err != nil {
		return nil
	}
	all, err := r.kvListSessions(context.Background())
	if err != nil {
		return nil
	}
	out := make([]*SessionInfo, 0, len(all))
	for _, s := range all {
		if s.Type == t {
			out = append(out, s)
		}
	}
	return out
}

// setLastSeenForTest backdates a session's LastSeenAt in the store.
// Exposed via the same package for tests that need to age sessions
// for the reaper without sleeping. NOT part of the public API.
func (r *SessionRegistry) setLastSeenForTest(id string, ts time.Time) error {
	if err := r.ensureStore(); err != nil {
		return err
	}
	ctx := context.Background()
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := r.kvGetSession(ctx, id)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("session %s not found", id)
	}
	info.LastSeenAt = ts
	return r.kvPutSessionLocked(ctx, info)
}
