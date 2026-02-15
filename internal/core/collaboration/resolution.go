package collaboration

import (
	"context"
	"fmt"
	"time"
)

// ConflictPolicy defines a strategy for resolving a conflict within a workspace.
type ConflictPolicy interface {
	// Resolve applies the policy to resolve a conflict, returning the resolution details.
	Resolve(ctx context.Context, conflict Conflict, workspace *Workspace) (*ConflictResolution, error)
}

// NewPolicy creates a ConflictPolicy for the given strategy.
func NewPolicy(strategy ResolutionStrategy) (ConflictPolicy, error) {
	switch strategy {
	case StrategyPriority:
		return &PriorityPolicy{}, nil
	case StrategyKeepFirst:
		return &KeepFirstPolicy{}, nil
	case StrategyKeepLast:
		return &KeepLastPolicy{}, nil
	case StrategyRollback:
		return &RollbackPolicy{}, nil
	default:
		return nil, fmt.Errorf("unsupported resolution strategy: %q", strategy)
	}
}

// rolePriority returns a numeric priority for an agent role.
// Higher values indicate higher priority.
func rolePriority(role AgentRole) int {
	switch role {
	case RoleOwner:
		return 3
	case RoleContributor:
		return 2
	case RoleObserver:
		return 1
	default:
		return 0
	}
}

// PriorityPolicy resolves conflicts by selecting the agent with the highest role.
// Role hierarchy: owner > contributor > observer.
type PriorityPolicy struct{}

// Resolve picks the value written by the highest-priority agent.
func (p *PriorityPolicy) Resolve(_ context.Context, conflict Conflict, workspace *Workspace) (*ConflictResolution, error) {
	if workspace == nil {
		return nil, fmt.Errorf("workspace is required for priority resolution")
	}
	if len(conflict.AgentIDs) == 0 {
		return nil, fmt.Errorf("conflict has no agents")
	}

	var winnerID string
	winnerPriority := -1

	for _, agentID := range conflict.AgentIDs {
		agent, err := workspace.GetAgent(agentID)
		if err != nil {
			continue
		}
		pri := rolePriority(agent.Role)
		if pri > winnerPriority {
			winnerPriority = pri
			winnerID = agentID
		}
	}

	if winnerID == "" {
		return nil, fmt.Errorf("no valid agents found in workspace for conflict resolution")
	}

	return &ConflictResolution{
		Strategy:   StrategyPriority,
		ResolvedBy: winnerID,
		Details:    fmt.Sprintf("agent %q wins by role priority", winnerID),
		Timestamp:  time.Now(),
	}, nil
}

// KeepFirstPolicy resolves conflicts by keeping the first (earliest) write.
type KeepFirstPolicy struct{}

// Resolve keeps the value from the earliest mutation.
func (p *KeepFirstPolicy) Resolve(_ context.Context, conflict Conflict, workspace *Workspace) (*ConflictResolution, error) {
	if workspace == nil || workspace.Context == nil {
		return nil, fmt.Errorf("workspace with context is required for keep-first resolution")
	}

	mutations := workspace.Context.MutationsForKey(conflict.Resource)
	if len(mutations) == 0 {
		return nil, fmt.Errorf("no mutations found for resource %q", conflict.Resource)
	}

	// Find the earliest mutation among conflicting agents.
	agentSet := make(map[string]bool, len(conflict.AgentIDs))
	for _, id := range conflict.AgentIDs {
		agentSet[id] = true
	}

	var earliest *ContextMutation
	for i := range mutations {
		if !agentSet[mutations[i].AgentID] {
			continue
		}
		if earliest == nil || mutations[i].Timestamp.Before(earliest.Timestamp) {
			earliest = &mutations[i]
		}
	}

	if earliest == nil {
		return nil, fmt.Errorf("no matching mutations found for conflicting agents on %q", conflict.Resource)
	}

	// Restore the first writer's value.
	_, err := workspace.Context.Set(conflict.Resource, earliest.NewValue, "system", RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to restore first writer value: %w", err)
	}

	return &ConflictResolution{
		Strategy:   StrategyKeepFirst,
		ResolvedBy: "system",
		Details:    fmt.Sprintf("kept value from first writer %q", earliest.AgentID),
		Timestamp:  time.Now(),
	}, nil
}

// KeepLastPolicy resolves conflicts by keeping the last (most recent) write.
type KeepLastPolicy struct{}

// Resolve keeps the value from the most recent mutation. This is effectively
// a no-op since the last write is already the current value, but we record
// the resolution for auditability.
func (p *KeepLastPolicy) Resolve(_ context.Context, conflict Conflict, workspace *Workspace) (*ConflictResolution, error) {
	if workspace == nil || workspace.Context == nil {
		return nil, fmt.Errorf("workspace with context is required for keep-last resolution")
	}

	mutations := workspace.Context.MutationsForKey(conflict.Resource)
	if len(mutations) == 0 {
		return nil, fmt.Errorf("no mutations found for resource %q", conflict.Resource)
	}

	// Find the latest mutation among conflicting agents.
	agentSet := make(map[string]bool, len(conflict.AgentIDs))
	for _, id := range conflict.AgentIDs {
		agentSet[id] = true
	}

	var latest *ContextMutation
	for i := range mutations {
		if !agentSet[mutations[i].AgentID] {
			continue
		}
		if latest == nil || mutations[i].Timestamp.After(latest.Timestamp) {
			latest = &mutations[i]
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no matching mutations found for conflicting agents on %q", conflict.Resource)
	}

	return &ConflictResolution{
		Strategy:   StrategyKeepLast,
		ResolvedBy: "system",
		Details:    fmt.Sprintf("kept value from last writer %q", latest.AgentID),
		Timestamp:  time.Now(),
	}, nil
}

// RollbackPolicy resolves conflicts by reverting to the value before the conflict.
type RollbackPolicy struct{}

// Resolve reverts the resource to its pre-conflict value.
func (p *RollbackPolicy) Resolve(_ context.Context, conflict Conflict, workspace *Workspace) (*ConflictResolution, error) {
	if workspace == nil || workspace.Context == nil {
		return nil, fmt.Errorf("workspace with context is required for rollback resolution")
	}

	mutations := workspace.Context.MutationsForKey(conflict.Resource)
	if len(mutations) == 0 {
		return nil, fmt.Errorf("no mutations found for resource %q", conflict.Resource)
	}

	// Find the earliest conflicting mutation and use its OldValue.
	agentSet := make(map[string]bool, len(conflict.AgentIDs))
	for _, id := range conflict.AgentIDs {
		agentSet[id] = true
	}

	var earliest *ContextMutation
	for i := range mutations {
		if !agentSet[mutations[i].AgentID] {
			continue
		}
		if earliest == nil || mutations[i].Timestamp.Before(earliest.Timestamp) {
			earliest = &mutations[i]
		}
	}

	if earliest == nil {
		return nil, fmt.Errorf("no matching mutations found for conflicting agents on %q", conflict.Resource)
	}

	// Restore the pre-conflict value.
	_, err := workspace.Context.Set(conflict.Resource, earliest.OldValue, "system", RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback value: %w", err)
	}

	return &ConflictResolution{
		Strategy:   StrategyRollback,
		ResolvedBy: "system",
		Details:    fmt.Sprintf("rolled back %q to pre-conflict value", conflict.Resource),
		Timestamp:  time.Now(),
	}, nil
}
