package multidevice

import (
	"fmt"
	"time"
)

// ResolutionManager orchestrates conflict resolution by selecting the
// appropriate strategy for each conflict type and persisting results.
type ResolutionManager struct {
	workspaceID string
	lww         *LWWResolver
	store       *ConflictStore
}

// NewResolutionManager creates a manager for the given workspace.
func NewResolutionManager(workspaceID string) *ResolutionManager {
	return &ResolutionManager{
		workspaceID: workspaceID,
		lww:         NewLWWResolver(),
		store:       NewConflictStore(workspaceID),
	}
}

// ResolveConflict attempts automatic resolution or escalates to manual
// review depending on the conflict type:
//
//   - concurrent_write -> LWW
//   - metadata         -> LWW
//   - ordering         -> LWW (simplified; no OT for now)
//   - semantic         -> mark as manual
func (m *ResolutionManager) ResolveConflict(conflict *Conflict) error {
	if conflict == nil {
		return fmt.Errorf("conflict must not be nil")
	}

	switch conflict.Type {
	case ConflictConcurrentWrite, ConflictMetadata, ConflictOrdering:
		resolution, err := m.lww.Resolve(conflict)
		if err != nil {
			return fmt.Errorf("auto-resolving conflict %s: %w", conflict.ID, err)
		}
		conflict.Resolution = resolution
		// conflict.Status and conflict.ResolvedAt are set by the resolver.

	case ConflictSemantic:
		// Semantic conflicts require human judgement.
		conflict.Status = ConflictManual

	default:
		return fmt.Errorf("unknown conflict type: %s", conflict.Type)
	}

	if err := m.store.Save(conflict); err != nil {
		return fmt.Errorf("saving conflict %s: %w", conflict.ID, err)
	}

	return nil
}

// ResolveManually resolves a conflict with a user-chosen strategy and
// chosen values. This is used for conflicts that could not be
// automatically resolved (e.g., semantic conflicts).
func (m *ResolutionManager) ResolveManually(conflictID string, strategy string, choice map[string]interface{}) error {
	conflict, err := m.store.Load(conflictID)
	if err != nil {
		return fmt.Errorf("loading conflict %s: %w", conflictID, err)
	}

	if conflict.Status == ConflictResolved || conflict.Status == ConflictAutoResolved {
		return fmt.Errorf("conflict %s is already resolved", conflictID)
	}

	now := time.Now()
	conflict.Status = ConflictResolved
	conflict.ResolvedAt = &now
	conflict.Resolution = &ConflictResolution{
		Strategy:   strategy,
		Result:     choice,
		ResolvedBy: "manual",
	}

	if err := m.store.Save(conflict); err != nil {
		return fmt.Errorf("saving resolved conflict %s: %w", conflictID, err)
	}

	return nil
}

// ListConflicts returns conflicts for the workspace. When includeResolved
// is true, all conflicts (including resolved ones) are returned.
func (m *ResolutionManager) ListConflicts(workspaceID string, includeResolved bool) ([]*Conflict, error) {
	return m.store.List(includeResolved)
}

// GetConflict returns a specific conflict by ID.
func (m *ResolutionManager) GetConflict(conflictID string) (*Conflict, error) {
	return m.store.Load(conflictID)
}
