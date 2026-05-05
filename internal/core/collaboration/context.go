package collaboration

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"hop.top/kit/go/runtime/domain"
)

// WorkspaceContext holds shared key-value state for a collaboration workspace.
// All operations are thread-safe.
//
// As of T-1290 the variable store is backed by a
// domain.Service[ContextVariable]. Set / Delete delegate to the
// service so the pre_validated + pre_persisted veto seams fire on
// every mutation. ACL checks remain on this type because they depend
// on (agentID, role) attributes that aren't part of the entity
// payload — they're enforced BEFORE the service call so a denied
// permission short-circuits without burning publisher cycles.
//
// The mutations log and ACL map remain plain in-memory state on
// WorkspaceContext: they are not entities themselves and don't
// benefit from being CRUD-shaped.
type WorkspaceContext struct {
	mu        sync.RWMutex
	repo      *contextRepository
	service   *domain.Service[ContextVariable]
	pub       domain.EventPublisher
	mutations []ContextMutation
	acls      map[string]ACLEntry
}

// WorkspaceContextOption configures a WorkspaceContext.
type WorkspaceContextOption func(*WorkspaceContext)

// WithContextPublisher attaches an event publisher to the underlying
// domain.Service[ContextVariable]. See WithSessionPublisher for the
// pre/post-event contract. The same publisher receives the
// aps.runtime.context_variable.* aliases emitted on success.
func WithContextPublisher(p domain.EventPublisher) WorkspaceContextOption {
	return func(wc *WorkspaceContext) {
		wc.pub = p
		wc.service = newContextService(wc.repo, p)
	}
}

// NewWorkspaceContext creates an empty workspace context.
func NewWorkspaceContext(opts ...WorkspaceContextOption) *WorkspaceContext {
	repo := newContextRepository()
	wc := &WorkspaceContext{
		repo:    repo,
		service: newContextService(repo, nil),
		acls:    make(map[string]ACLEntry),
	}
	for _, o := range opts {
		o(wc)
	}
	return wc
}

// publishAlias is a fire-and-forget alias publisher for the
// aps.runtime.context_variable.* topics. Errors from subscribers on
// alias topics never veto — vetoes belong on the
// kit.runtime.entity.pre_* seams, not the aps notifications.
func (wc *WorkspaceContext) publishAlias(topic string, payload any) {
	if wc.pub == nil {
		return
	}
	_ = wc.pub.Publish(context.Background(), topic, topicContextSource, payload)
}

// NewWorkspaceContextFromState restores context from persisted state.
// Restored variables bypass the domain.Service event seams (loading
// is not a mutation); subsequent Set/Delete operations flow through
// the service as usual.
func NewWorkspaceContextFromState(variables []ContextVariable, acls map[string]ACLEntry, opts ...WorkspaceContextOption) *WorkspaceContext {
	wc := NewWorkspaceContext(opts...)
	for _, v := range variables {
		wc.repo.variables[v.Key] = v
	}
	if acls != nil {
		wc.acls = acls
	}
	return wc
}

// SetOption configures an individual Set/SetWithContext call. Used
// instead of a positional parameter so the public Set signature stays
// stable as new per-write attributes are introduced (T-1309).
type SetOption func(*setOptions)

// setOptions is the resolved bag for a single Set call.
type setOptions struct {
	visibility    ContextVisibility
	visibilitySet bool
}

// WithVisibility opts the resulting ContextVariable into a specific
// visibility scope. When unset the existing variable's visibility is
// preserved (or VisibilityShared on first write); explicitly passing
// VisibilityShared on a previously private variable promotes it back
// to shared.
func WithVisibility(v ContextVisibility) SetOption {
	return func(o *setOptions) {
		o.visibility = v
		o.visibilitySet = true
	}
}

// Set sets a context variable, checking ACL permissions.
//
// Routing: a first write for a key is dispatched to service.Create
// (firing pre/post created events); subsequent writes go to
// service.Update (firing pre/post updated events). Both paths
// validate. ACL is checked on this side because the (agentID, role)
// pair is not part of the entity payload and a permission denial
// must short-circuit before any publisher subscriber sees the event.
func (wc *WorkspaceContext) Set(key, value, agentID string, role AgentRole, opts ...SetOption) (*ContextVariable, error) {
	return wc.SetWithContext(context.Background(), key, value, agentID, role, opts...)
}

// SetWithContext is the ctx-aware variant of Set; reads the audit note
// from policy.ContextAttrsKey (T-1291) and stores it on the resulting
// ContextMutation.
//
// T-1309: accepts SetOption values (e.g. WithVisibility) without
// breaking existing positional-only callers.
func (wc *WorkspaceContext) SetWithContext(ctx context.Context, key, value, agentID string, role AgentRole, opts ...SetOption) (*ContextVariable, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	now := time.Now()
	existing, exists := wc.repo.variables[key]

	// T-1309: writes to a private variable owned by another profile
	// behave as "not found" so the caller cannot probe for the
	// variable's existence (and cannot clobber it). The check sits
	// before the ACL gate intentionally — a permission-denied error
	// would itself leak existence.
	if exists && !existing.IsVisibleTo(agentID) {
		return nil, fmt.Errorf("context variable %q not found", key)
	}

	if err := wc.checkPermission(key, role, PermWrite); err != nil {
		return nil, err
	}

	var sopts setOptions
	for _, o := range opts {
		o(&sopts)
	}

	version := 1
	var oldValue string
	if exists {
		version = existing.Version + 1
		oldValue = existing.Value
	}

	// Visibility resolution rule (T-1309):
	//   1. If the caller passed WithVisibility, honour it verbatim.
	//   2. Otherwise on update preserve the existing variable's
	//      Visibility (no implicit demotion / promotion).
	//   3. On first write fall through to the zero value, which reads
	//      as VisibilityShared via NormalizeVisibility — and we do NOT
	//      set the field on the entity so existing yaml without
	//      visibility round-trips byte-stable until an operator opts in.
	visibility := ContextVisibility("")
	if sopts.visibilitySet {
		visibility = sopts.visibility
	} else if exists {
		visibility = existing.Visibility
	}

	v := ContextVariable{
		Key:        key,
		Value:      value,
		Version:    version,
		UpdatedBy:  agentID,
		UpdatedAt:  now,
		Visibility: visibility,
	}

	if exists {
		if err := wc.service.Update(ctx, &v); err != nil {
			return nil, fmt.Errorf("update context variable: %w", err)
		}
		wc.publishAlias(TopicContextVariableUpdated, v)
	} else {
		if err := wc.service.Create(ctx, &v); err != nil {
			return nil, fmt.Errorf("create context variable: %w", err)
		}
		wc.publishAlias(TopicContextVariableCreated, v)
	}

	wc.mutations = append(wc.mutations, ContextMutation{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  value,
		Version:   version,
		AgentID:   agentID,
		Note:      noteFromContext(ctx),
		Timestamp: now,
	})

	return &v, nil
}

// Get returns a context variable by key WITHOUT a visibility filter.
//
// This is the raw access path used by storage round-trips, metrics,
// and tests. CLI handlers that act on behalf of a specific profile
// MUST use GetForProfile so private variables don't leak across
// profiles (T-1309).
func (wc *WorkspaceContext) Get(key string) (*ContextVariable, bool) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	v, ok := wc.repo.variables[key]
	if !ok {
		return nil, false
	}
	return &v, true
}

// GetForProfile returns a context variable visible to profileID
// (T-1309). Private variables written by another profile behave as
// "not found" so existence does not leak across profiles. Pass an
// empty profileID to opt out of the visibility filter (equivalent to
// Get).
func (wc *WorkspaceContext) GetForProfile(key, profileID string) (*ContextVariable, bool) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	v, ok := wc.repo.variables[key]
	if !ok {
		return nil, false
	}
	if profileID != "" && !v.IsVisibleTo(profileID) {
		return nil, false
	}
	return &v, true
}

// Delete removes a context variable, checking ACL permissions.
func (wc *WorkspaceContext) Delete(key, agentID string, role AgentRole) error {
	return wc.DeleteWithContext(context.Background(), key, agentID, role)
}

// DeleteWithContext is the ctx-aware variant of Delete; reads the audit
// note from policy.ContextAttrsKey (T-1291) and stores it on the
// resulting ContextMutation.
//
// T-1309: when the variable is private and the caller is not the
// writer, the call surfaces "not found" rather than "permission
// denied" so existence does not leak across profiles.
func (wc *WorkspaceContext) DeleteWithContext(ctx context.Context, key, agentID string, role AgentRole) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	existing, exists := wc.repo.variables[key]
	if exists && !existing.IsVisibleTo(agentID) {
		// Private var owned by another profile: behave as
		// "not found" so the caller cannot probe for existence.
		return fmt.Errorf("context variable %q not found", key)
	}

	if err := wc.checkPermission(key, role, PermDelete); err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("context variable %q not found", key)
	}

	if err := wc.service.Delete(ctx, key); err != nil {
		// Translate the kit ErrNotFound back to the legacy message
		// so existing callers and tests don't see a surface change.
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("context variable %q not found", key)
		}
		return fmt.Errorf("delete context variable: %w", err)
	}
	wc.publishAlias(TopicContextVariableDeleted, map[string]string{"id": key})

	wc.mutations = append(wc.mutations, ContextMutation{
		Key:       key,
		OldValue:  existing.Value,
		NewValue:  "",
		Version:   existing.Version + 1,
		AgentID:   agentID,
		Note:      noteFromContext(ctx),
		Timestamp: time.Now(),
	})

	return nil
}

// List returns all context variables WITHOUT a visibility filter.
//
// Like Get, this is the raw access path used by storage round-trips,
// metrics, and tests. CLI handlers acting on behalf of a profile MUST
// use ListForProfile so private variables stay scoped to their
// writer (T-1309).
func (wc *WorkspaceContext) List() []ContextVariable {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return wc.repo.snapshot()
}

// ListForProfile returns context variables visible to profileID
// (T-1309). Private variables written by other profiles are omitted.
// Pass an empty profileID to disable the filter (equivalent to List).
func (wc *WorkspaceContext) ListForProfile(profileID string) []ContextVariable {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	all := wc.repo.snapshot()
	if profileID == "" {
		return all
	}
	out := make([]ContextVariable, 0, len(all))
	for _, v := range all {
		if v.IsVisibleTo(profileID) {
			out = append(out, v)
		}
	}
	return out
}

// Mutations returns the full mutation history.
func (wc *WorkspaceContext) Mutations() []ContextMutation {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	out := make([]ContextMutation, len(wc.mutations))
	copy(out, wc.mutations)
	return out
}

// MutationsForKey returns mutation history for a specific key.
func (wc *WorkspaceContext) MutationsForKey(key string) []ContextMutation {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	var out []ContextMutation
	for _, m := range wc.mutations {
		if m.Key == key {
			out = append(out, m)
		}
	}
	return out
}

// SetACL sets the access control entry for a key.
func (wc *WorkspaceContext) SetACL(acl ACLEntry) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	wc.acls[acl.Key] = acl
}

// GetACL returns the ACL for a key, or the default ACL.
func (wc *WorkspaceContext) GetACL(key string) ACLEntry {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if acl, ok := wc.acls[key]; ok {
		return acl
	}
	return DefaultACL(key)
}

// Snapshot returns a read-only copy of all variables and ACLs.
func (wc *WorkspaceContext) Snapshot() ([]ContextVariable, map[string]ACLEntry) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	vars := wc.repo.snapshot()

	acls := make(map[string]ACLEntry, len(wc.acls))
	maps.Copy(acls, wc.acls)

	return vars, acls
}

// checkPermission validates an agent has the required permission on a key.
func (wc *WorkspaceContext) checkPermission(key string, role AgentRole, perm Permission) error {
	acl, ok := wc.acls[key]
	if !ok {
		acl = DefaultACL(key)
	}
	if !acl.HasPermission(role, perm) {
		return fmt.Errorf("permission denied: role %q lacks %q permission on %q", role, perm, key)
	}
	return nil
}

// DetectWriteConflict checks if two agents wrote to the same key within a window.
func (wc *WorkspaceContext) DetectWriteConflict(key string, window time.Duration) *Conflict {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	var recentMutations []ContextMutation
	cutoff := time.Now().Add(-window)

	for i := len(wc.mutations) - 1; i >= 0; i-- {
		m := wc.mutations[i]
		if m.Key != key {
			continue
		}
		if m.Timestamp.Before(cutoff) {
			break
		}
		recentMutations = append(recentMutations, m)
	}

	if len(recentMutations) < 2 {
		return nil
	}

	agents := make(map[string]bool)
	for _, m := range recentMutations {
		agents[m.AgentID] = true
	}
	if len(agents) < 2 {
		return nil
	}

	agentIDs := make([]string, 0, len(agents))
	for a := range agents {
		agentIDs = append(agentIDs, a)
	}

	return &Conflict{
		Type:        ConflictWrite,
		Resource:    key,
		AgentIDs:    agentIDs,
		Description: fmt.Sprintf("concurrent writes to %q by %d agents", key, len(agentIDs)),
		DetectedAt:  time.Now(),
	}
}
