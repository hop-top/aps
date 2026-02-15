package collaboration

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ConflictDetector scans workspaces for various types of conflicts.
type ConflictDetector struct{}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectWriteConflicts scans workspace context mutations for concurrent writes
// to the same key by different agents within the given time window.
func (cd *ConflictDetector) DetectWriteConflicts(workspace *Workspace, window time.Duration) []Conflict {
	if workspace == nil || workspace.Context == nil {
		return nil
	}

	mutations := workspace.Context.Mutations()
	if len(mutations) < 2 {
		return nil
	}

	cutoff := time.Now().Add(-window)

	// Group recent mutations by key.
	byKey := make(map[string][]ContextMutation)
	for _, m := range mutations {
		if m.Timestamp.Before(cutoff) {
			continue
		}
		byKey[m.Key] = append(byKey[m.Key], m)
	}

	var conflicts []Conflict
	for key, keyMutations := range byKey {
		if len(keyMutations) < 2 {
			continue
		}

		// Collect distinct agents that wrote to this key.
		agents := make(map[string]bool)
		for _, m := range keyMutations {
			agents[m.AgentID] = true
		}
		if len(agents) < 2 {
			continue
		}

		agentIDs := make([]string, 0, len(agents))
		for a := range agents {
			agentIDs = append(agentIDs, a)
		}

		conflicts = append(conflicts, Conflict{
			ID:          uuid.New().String(),
			WorkspaceID: workspace.ID,
			Type:        ConflictWrite,
			Resource:    key,
			AgentIDs:    agentIDs,
			Description: fmt.Sprintf("concurrent writes to %q by %d agents within %s", key, len(agentIDs), window),
			DetectedAt:  time.Now(),
		})
	}

	return conflicts
}

// DetectOrderingConflicts checks for circular dependencies among the given tasks.
// A circular dependency means no valid execution order exists.
func (cd *ConflictDetector) DetectOrderingConflicts(tasks []TaskInfo) []Conflict {
	if len(tasks) == 0 {
		return nil
	}

	graph := NewDependencyGraph()
	for _, t := range tasks {
		graph.AddTask(t.ID, t.Dependencies)
	}

	cycle := graph.DetectCycles()
	if cycle == nil {
		return nil
	}

	return []Conflict{
		{
			ID:          uuid.New().String(),
			Type:        ConflictOrdering,
			Resource:    "task-dependencies",
			AgentIDs:    cycle,
			Description: fmt.Sprintf("circular task dependency detected: %v", cycle),
			DetectedAt:  time.Now(),
		},
	}
}

// DetectLockConflicts checks whether the requestor can acquire a lock on the
// given resource. The locks map holds resource -> current holder mappings.
// Returns a conflict if the resource is already held by a different agent.
func (cd *ConflictDetector) DetectLockConflicts(locks map[string]string, requestor string, resource string) *Conflict {
	if locks == nil {
		return nil
	}

	holder, held := locks[resource]
	if !held || holder == requestor {
		return nil
	}

	return &Conflict{
		ID:       uuid.New().String(),
		Type:     ConflictLock,
		Resource: resource,
		AgentIDs: []string{holder, requestor},
		Description: fmt.Sprintf(
			"lock conflict on %q: held by %q, requested by %q",
			resource, holder, requestor,
		),
		DetectedAt: time.Now(),
	}
}
