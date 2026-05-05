package collaboration

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"hop.top/kit/go/runtime/domain"
)

// GetID makes Session satisfy domain.Entity. Sessions are uniquely
// identified by their ID across all workspaces.
func (s Session) GetID() string { return s.ID }

// Bus topics emitted by SessionManager as aps-side aliases on top of
// the kit.runtime.entity.* events fired by domain.Service[Session].
//
// kit.runtime.entity.* (the canonical pre/post events) are kept on
// the kit defaults so the kit/runtime/policy engine has a stable
// veto seam to subscribe to (T-1192 style). The aliases below are
// best-effort, fire-and-forget publishes done from SessionManager
// AFTER the service-level write succeeds, so legacy aps subscribers
// matching aps.runtime.session.# keep receiving lifecycle events.
//
// 4-segment past-tense per kit topic spec.
const (
	TopicSessionCreated = "aps.runtime.session.created"
	TopicSessionUpdated = "aps.runtime.session.updated"
	TopicSessionDeleted = "aps.runtime.session.deleted"
	topicSessionSource  = "aps.collaboration.session"
)

// sessionRepository is the in-memory domain.Repository[Session]
// backing SessionManager. Storage semantics match the original
// SessionManager: a single map keyed by Session.ID with an RWMutex
// guarding concurrent access. Persistence is deliberately absent —
// collaboration sessions are ephemeral runtime state.
type sessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func newSessionRepository() *sessionRepository {
	return &sessionRepository{sessions: make(map[string]*Session)}
}

// Create stores a new session. Returns ErrConflict if the ID exists.
func (r *sessionRepository) Create(_ context.Context, entity *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[entity.ID]; exists {
		return fmt.Errorf("session %q: %w", entity.ID, domain.ErrConflict)
	}
	cp := *entity
	r.sessions[entity.ID] = &cp
	return nil
}

// Get returns a copy of the session for the given ID.
func (r *sessionRepository) Get(_ context.Context, id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q: %w", id, domain.ErrNotFound)
	}
	cp := *s
	return &cp, nil
}

// List returns all sessions. The Query filters (Limit/Offset) are
// honored on best effort: aps consumers query via the SessionManager
// helpers (ListSessions, ListActiveSessions) which apply their own
// per-workspace filters on top.
func (r *sessionRepository) List(_ context.Context, q domain.Query) ([]Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, *s)
	}
	if q.Offset > 0 && q.Offset < len(out) {
		out = out[q.Offset:]
	} else if q.Offset >= len(out) {
		return []Session{}, nil
	}
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	return out, nil
}

// Update replaces an existing session. Returns ErrNotFound if absent.
func (r *sessionRepository) Update(_ context.Context, entity *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[entity.ID]; !exists {
		return fmt.Errorf("session %q: %w", entity.ID, domain.ErrNotFound)
	}
	cp := *entity
	r.sessions[entity.ID] = &cp
	return nil
}

// Delete removes a session. Returns ErrNotFound if absent.
func (r *sessionRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[id]; !exists {
		return fmt.Errorf("session %q: %w", id, domain.ErrNotFound)
	}
	delete(r.sessions, id)
	return nil
}

// hasActiveSession reports whether an active session for the
// (workspace, profile) pair already exists. Used by SessionManager
// to enforce the no-duplicate-active-session invariant before the
// service.Create call.
func (r *sessionRepository) hasActiveSession(workspaceID, profileID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		if s.WorkspaceID == workspaceID && s.ProfileID == profileID && s.State == SessionActive {
			return true
		}
	}
	return false
}

// sessionValidator enforces the structural invariants on a Session
// before persistence. Cross-entity rules (no duplicate active session
// per (workspace, profile)) are enforced inline by SessionManager
// because they need precise error-message preservation that the
// generic ErrValidation wrapping doesn't carry.
type sessionValidator struct{}

func (sessionValidator) Validate(_ context.Context, s Session) error {
	if s.ID == "" {
		return fmt.Errorf("%w: session id required", domain.ErrValidation)
	}
	if s.WorkspaceID == "" {
		return fmt.Errorf("%w: workspace id required", domain.ErrValidation)
	}
	if s.ProfileID == "" {
		return fmt.Errorf("%w: profile id required", domain.ErrValidation)
	}
	switch s.State {
	case SessionJoining, SessionActive, SessionLeaving, SessionClosed:
	default:
		return fmt.Errorf("%w: invalid session state %q", domain.ErrValidation, s.State)
	}
	return nil
}

// errDuplicateActiveSession sentinels the legacy duplicate-active-session
// path so SessionManager.CreateSession can preserve the original error
// string (the existing test asserts on it).
var errDuplicateActiveSession = errors.New("already has an active session")

// newSessionService wires a domain.Service[Session] around the given
// repository and (optional) publisher. The service uses kit's
// DefaultTopics (kit.runtime.entity.*) so the kit/runtime/policy
// engine can subscribe to a stable veto seam. aps-prefixed alias
// topics (aps.runtime.session.*) are emitted from SessionManager
// after a successful service call — see sessionAliasPublish.
func newSessionService(repo *sessionRepository, pub domain.EventPublisher) *domain.Service[Session] {
	opts := []domain.Option[Session]{
		domain.WithValidation[Session](sessionValidator{}),
	}
	if pub != nil {
		opts = append(opts, domain.WithPublisher[Session](pub))
	}
	return domain.NewService[Session](repo, opts...)
}
