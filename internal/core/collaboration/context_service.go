package collaboration

import (
	"context"
	"fmt"
	"sync"

	"hop.top/kit/go/runtime/domain"
)

// GetID makes ContextVariable satisfy domain.Entity. The key is the
// natural identifier within a single WorkspaceContext.
func (v ContextVariable) GetID() string { return v.Key }

// Bus topics emitted by WorkspaceContext as aps-side aliases on top
// of the kit.runtime.entity.* events fired by
// domain.Service[ContextVariable]. See session_service.go for the
// rationale: kit defaults stay canonical for the policy engine, and
// SessionManager-style alias publishes keep aps-prefixed subscribers
// notified.
const (
	TopicContextVariableCreated = "aps.runtime.context_variable.created"
	TopicContextVariableUpdated = "aps.runtime.context_variable.updated"
	TopicContextVariableDeleted = "aps.runtime.context_variable.deleted"
	topicContextSource          = "aps.collaboration.context"
)

// contextRepository is the in-memory domain.Repository[ContextVariable]
// backing WorkspaceContext. It owns ONLY the variables map; ACLs and
// the mutation log stay on WorkspaceContext because they are not
// part of ContextVariable's identity.
type contextRepository struct {
	mu        sync.RWMutex
	variables map[string]ContextVariable
}

func newContextRepository() *contextRepository {
	return &contextRepository{variables: make(map[string]ContextVariable)}
}

func (r *contextRepository) Create(_ context.Context, entity *ContextVariable) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.variables[entity.Key]; exists {
		return fmt.Errorf("context variable %q: %w", entity.Key, domain.ErrConflict)
	}
	r.variables[entity.Key] = *entity
	return nil
}

func (r *contextRepository) Get(_ context.Context, id string) (*ContextVariable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.variables[id]
	if !ok {
		return nil, fmt.Errorf("context variable %q: %w", id, domain.ErrNotFound)
	}
	return &v, nil
}

func (r *contextRepository) List(_ context.Context, q domain.Query) ([]ContextVariable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ContextVariable, 0, len(r.variables))
	for _, v := range r.variables {
		out = append(out, v)
	}
	if q.Offset > 0 && q.Offset < len(out) {
		out = out[q.Offset:]
	} else if q.Offset >= len(out) {
		return []ContextVariable{}, nil
	}
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	return out, nil
}

func (r *contextRepository) Update(_ context.Context, entity *ContextVariable) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.variables[entity.Key]; !exists {
		return fmt.Errorf("context variable %q: %w", entity.Key, domain.ErrNotFound)
	}
	r.variables[entity.Key] = *entity
	return nil
}

func (r *contextRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.variables[id]; !exists {
		return fmt.Errorf("context variable %q: %w", id, domain.ErrNotFound)
	}
	delete(r.variables, id)
	return nil
}

// snapshot returns a copy of the current variables map. Used by the
// snapshot/list helpers on WorkspaceContext.
func (r *contextRepository) snapshot() []ContextVariable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ContextVariable, 0, len(r.variables))
	for _, v := range r.variables {
		out = append(out, v)
	}
	return out
}

// contextValidator enforces the minimal invariants on a stored
// context variable. ACL checks are NOT here: they depend on
// (agentID, role) attributes that travel separately from the entity
// payload, and they're enforced by WorkspaceContext.Set/Delete
// before the service is invoked.
type contextValidator struct{}

func (contextValidator) Validate(_ context.Context, v ContextVariable) error {
	if v.Key == "" {
		return fmt.Errorf("%w: context variable key required", domain.ErrValidation)
	}
	if v.Version < 1 {
		return fmt.Errorf("%w: context variable %q version must be >= 1", domain.ErrValidation, v.Key)
	}
	if v.UpdatedBy == "" {
		return fmt.Errorf("%w: context variable %q updated_by required", domain.ErrValidation, v.Key)
	}
	return nil
}

// newContextService wires a domain.Service[ContextVariable] using
// kit's DefaultTopics for pre/post events. aps-prefixed aliases
// (aps.runtime.context_variable.*) are emitted by WorkspaceContext
// after a successful service call — see WorkspaceContext.publishAlias.
func newContextService(repo *contextRepository, pub domain.EventPublisher) *domain.Service[ContextVariable] {
	opts := []domain.Option[ContextVariable]{
		domain.WithValidation[ContextVariable](contextValidator{}),
	}
	if pub != nil {
		opts = append(opts, domain.WithPublisher[ContextVariable](pub))
	}
	return domain.NewService[ContextVariable](repo, opts...)
}
