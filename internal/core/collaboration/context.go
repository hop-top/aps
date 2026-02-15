package collaboration

import (
	"fmt"
	"maps"
	"sync"
	"time"
)

// WorkspaceContext holds shared key-value state for a collaboration workspace.
// All operations are thread-safe.
type WorkspaceContext struct {
	mu        sync.RWMutex
	variables map[string]ContextVariable
	mutations []ContextMutation
	acls      map[string]ACLEntry
}

// NewWorkspaceContext creates an empty workspace context.
func NewWorkspaceContext() *WorkspaceContext {
	return &WorkspaceContext{
		variables: make(map[string]ContextVariable),
		acls:      make(map[string]ACLEntry),
	}
}

// NewWorkspaceContextFromState restores context from persisted state.
func NewWorkspaceContextFromState(variables []ContextVariable, acls map[string]ACLEntry) *WorkspaceContext {
	ctx := NewWorkspaceContext()
	for _, v := range variables {
		ctx.variables[v.Key] = v
	}
	if acls != nil {
		ctx.acls = acls
	}
	return ctx
}

// Set sets a context variable, checking ACL permissions.
func (wc *WorkspaceContext) Set(key, value, agentID string, role AgentRole) (*ContextVariable, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if err := wc.checkPermission(key, role, PermWrite); err != nil {
		return nil, err
	}

	now := time.Now()
	existing, exists := wc.variables[key]

	version := 1
	var oldValue string
	if exists {
		version = existing.Version + 1
		oldValue = existing.Value
	}

	v := ContextVariable{
		Key:       key,
		Value:     value,
		Version:   version,
		UpdatedBy: agentID,
		UpdatedAt: now,
	}
	wc.variables[key] = v

	wc.mutations = append(wc.mutations, ContextMutation{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  value,
		Version:   version,
		AgentID:   agentID,
		Timestamp: now,
	})

	return &v, nil
}

// Get returns a context variable by key.
func (wc *WorkspaceContext) Get(key string) (*ContextVariable, bool) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	v, ok := wc.variables[key]
	if !ok {
		return nil, false
	}
	return &v, true
}

// Delete removes a context variable, checking ACL permissions.
func (wc *WorkspaceContext) Delete(key, agentID string, role AgentRole) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if err := wc.checkPermission(key, role, PermDelete); err != nil {
		return err
	}

	existing, exists := wc.variables[key]
	if !exists {
		return fmt.Errorf("context variable %q not found", key)
	}

	delete(wc.variables, key)

	wc.mutations = append(wc.mutations, ContextMutation{
		Key:       key,
		OldValue:  existing.Value,
		NewValue:  "",
		Version:   existing.Version + 1,
		AgentID:   agentID,
		Timestamp: time.Now(),
	})

	return nil
}

// List returns all context variables.
func (wc *WorkspaceContext) List() []ContextVariable {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	vars := make([]ContextVariable, 0, len(wc.variables))
	for _, v := range wc.variables {
		vars = append(vars, v)
	}
	return vars
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

	vars := make([]ContextVariable, 0, len(wc.variables))
	for _, v := range wc.variables {
		vars = append(vars, v)
	}

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
