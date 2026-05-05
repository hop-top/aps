package collaboration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hop.top/kit/go/runtime/domain"
)

// SessionManager tracks agent sessions within workspaces.
//
// As of T-1290 SessionManager is a thin facade over a
// domain.Service[Session] that owns CRUD-shaped lifecycle (Create /
// Update / Delete) and fires kit's canonical pre-event seams
// (kit.runtime.entity.pre_validated, kit.runtime.entity.pre_persisted)
// so the kit/runtime/policy engine can veto.
//
// Domain-specific operations whose semantics don't fit the generic
// CRUD verbs — Heartbeat, CloseSession, ExpireSessions — are layered
// on top: each reads the current entity, applies the field/state
// change in memory, and persists through service.Update so the same
// pre/post seams still fire.
//
// CleanupClosed routes through service.Delete one-by-one so each
// removed session emits a deleted event.
//
// Topic strategy: kit.runtime.entity.* are the authoritative pre/post
// events. SessionManager additionally emits aps.runtime.session.*
// aliases (see TopicSessionCreated / Updated / Deleted) on success
// for legacy aps subscribers — same approach wsm T-1286 takes for
// Workspace via wsm.runtime.workspace.*.
//
// The composition with internal/core/session.statusMachine is
// orthogonal: that package's StateMachine guards SessionInfo (a
// distinct entity for the per-process session registry). The
// collaboration Session has its own state set
// (joining/active/leaving/closed) enforced by sessionValidator.
type SessionManager struct {
	repo    *sessionRepository
	service *domain.Service[Session]
	pub     domain.EventPublisher
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

// SessionManagerOption configures a SessionManager.
type SessionManagerOption func(*SessionManager)

// WithSessionPublisher attaches an event publisher to the underlying
// domain.Service[Session]. Pre-events (pre_validated, pre_persisted)
// fire synchronously; subscriber errors veto the operation. Post
// events (created, updated, deleted) are best-effort. The same
// publisher receives the aps.runtime.session.* aliases emitted by
// SessionManager after a successful service call. When no publisher
// is configured (the default) the service still validates and
// persists but emits no events — matching pre-T-1290 behaviour for
// callers that don't wire a bus.
func WithSessionPublisher(p domain.EventPublisher) SessionManagerOption {
	return func(sm *SessionManager) {
		sm.pub = p
		sm.service = newSessionService(sm.repo, p)
	}
}

// NewSessionManager creates a session manager. Without options the
// underlying domain.Service runs without an event publisher; pass
// WithSessionPublisher(...) at boot to wire bus events.
func NewSessionManager(opts ...SessionManagerOption) *SessionManager {
	repo := newSessionRepository()
	sm := &SessionManager{
		repo:    repo,
		service: newSessionService(repo, nil),
	}
	for _, o := range opts {
		o(sm)
	}
	return sm
}

// publishAlias is a fire-and-forget alias publisher. Errors from
// subscribers on alias topics never veto — vetoes belong on the
// kit.runtime.entity.pre_* seams, not the aps notifications.
func (sm *SessionManager) publishAlias(topic string, payload any) {
	if sm.pub == nil {
		return
	}
	_ = sm.pub.Publish(context.Background(), topic, topicSessionSource, payload)
}

// CreateSession creates a new session for an agent joining a workspace.
//
// The duplicate-active-session check runs BEFORE service.Create so
// the original error message ("already has an active session …") is
// preserved verbatim — the existing test asserts on this substring.
// The actual write goes through service.Create so pre_validated +
// pre_persisted seams fire.
func (sm *SessionManager) CreateSession(workspaceID, profileID string, timeout time.Duration) (*Session, error) {
	if sm.repo.hasActiveSession(workspaceID, profileID) {
		return nil, fmt.Errorf("agent %q already has an active session in workspace %q: %w",
			profileID, workspaceID, errDuplicateActiveSession)
	}

	now := time.Now()
	session := Session{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		ProfileID:   profileID,
		State:       SessionActive,
		JoinedAt:    now,
		LastSeen:    now,
		ExpiresAt:   now.Add(timeout),
	}

	if err := sm.service.Create(context.Background(), &session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	got, err := sm.repo.Get(context.Background(), session.ID)
	if err != nil {
		return nil, fmt.Errorf("create session: load post-create: %w", err)
	}
	sm.publishAlias(TopicSessionCreated, *got)
	return got, nil
}

// UpdateSession persists an in-place update through the domain
// service. Pre-events fire and may veto. Validation runs.
//
// The wider SessionManager API uses this internally for Heartbeat,
// CloseSession, and ExpireSessions; it is also exported so external
// callers can mutate fields not covered by those helpers (metadata,
// expiry adjustments, …) while still flowing through the policy
// seams.
func (sm *SessionManager) UpdateSession(s *Session) error {
	if s == nil {
		return fmt.Errorf("update: session is nil")
	}
	if err := sm.service.Update(context.Background(), s); err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	sm.publishAlias(TopicSessionUpdated, *s)
	return nil
}

// DeleteSession removes a session by ID through the domain service,
// firing pre_validated + pre_persisted (with nil entity) and
// emitting a deleted post-event.
func (sm *SessionManager) DeleteSession(sessionID string) error {
	if err := sm.service.Delete(context.Background(), sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	sm.publishAlias(TopicSessionDeleted, map[string]string{"id": sessionID})
	return nil
}

// Heartbeat updates a session's last-seen time and extends its expiry.
func (sm *SessionManager) Heartbeat(sessionID string, timeout time.Duration) error {
	s, err := sm.repo.Get(context.Background(), sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("session %q not found", sessionID)
		}
		return err
	}
	if s.State != SessionActive {
		return fmt.Errorf("session %q is not active (state: %s)", sessionID, s.State)
	}

	now := time.Now()
	s.LastSeen = now
	s.ExpiresAt = now.Add(timeout)
	if err := sm.service.Update(context.Background(), s); err != nil {
		return fmt.Errorf("heartbeat session: %w", err)
	}
	sm.publishAlias(TopicSessionUpdated, *s)
	return nil
}

// CloseSession marks a session as closed.
func (sm *SessionManager) CloseSession(sessionID string) error {
	s, err := sm.repo.Get(context.Background(), sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("session %q not found", sessionID)
		}
		return fmt.Errorf("load session: %w", err)
	}

	s.State = SessionClosed
	if err := sm.service.Update(context.Background(), s); err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	sm.publishAlias(TopicSessionUpdated, *s)
	return nil
}

// GetSession returns a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	s, err := sm.repo.Get(context.Background(), sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("session %q not found", sessionID)
		}
		return nil, err
	}
	return s, nil
}

// GetAgentSession returns the active session for an agent in a workspace.
func (sm *SessionManager) GetAgentSession(workspaceID, profileID string) (*Session, error) {
	all, _ := sm.repo.List(context.Background(), domain.Query{})
	for i := range all {
		s := all[i]
		if s.WorkspaceID == workspaceID && s.ProfileID == profileID && s.State == SessionActive {
			cp := s
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("no active session for agent %q in workspace %q", profileID, workspaceID)
}

// ListSessions returns all sessions for a workspace.
func (sm *SessionManager) ListSessions(workspaceID string) []*Session {
	all, _ := sm.repo.List(context.Background(), domain.Query{})
	out := make([]*Session, 0, len(all))
	for i := range all {
		s := all[i]
		if s.WorkspaceID == workspaceID {
			cp := s
			out = append(out, &cp)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ListActiveSessions returns active sessions for a workspace.
func (sm *SessionManager) ListActiveSessions(workspaceID string) []*Session {
	all, _ := sm.repo.List(context.Background(), domain.Query{})
	out := make([]*Session, 0)
	for i := range all {
		s := all[i]
		if s.WorkspaceID == workspaceID && s.State == SessionActive {
			cp := s
			out = append(out, &cp)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ExpireSessions finds and closes sessions that have exceeded their timeout.
// Returns the profile IDs of expired agents. Each expiry is persisted
// through service.Update so the pre/post seams fire per session.
func (sm *SessionManager) ExpireSessions(workspaceID string) []string {
	all, _ := sm.repo.List(context.Background(), domain.Query{})
	now := time.Now()
	var expired []string
	for i := range all {
		s := all[i]
		if s.WorkspaceID == workspaceID && s.State == SessionActive && now.After(s.ExpiresAt) {
			s.State = SessionClosed
			if err := sm.service.Update(context.Background(), &s); err != nil {
				// On veto/persistence error, skip this session;
				// callers see a partial expired list. Matches the
				// pre-refactor behaviour where map mutation was
				// best-effort under the lock.
				continue
			}
			sm.publishAlias(TopicSessionUpdated, s)
			expired = append(expired, s.ProfileID)
		}
	}
	return expired
}

// CleanupClosed removes all closed sessions for a workspace.
func (sm *SessionManager) CleanupClosed(workspaceID string) int {
	all, _ := sm.repo.List(context.Background(), domain.Query{})
	count := 0
	for i := range all {
		s := all[i]
		if s.WorkspaceID == workspaceID && s.State == SessionClosed {
			if err := sm.service.Delete(context.Background(), s.ID); err != nil {
				continue
			}
			sm.publishAlias(TopicSessionDeleted, map[string]string{"id": s.ID})
			count++
		}
	}
	return count
}
