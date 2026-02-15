package collaboration

import (
	"context"
	"encoding/json"
)

// WorkspaceManager orchestrates workspace lifecycle operations.
type WorkspaceManager interface {
	// Create creates a new collaboration workspace.
	Create(ctx context.Context, config WorkspaceConfig) (*Workspace, error)

	// Get returns a workspace by ID with ACL checks.
	Get(ctx context.Context, id string) (*Workspace, error)

	// List returns workspaces matching optional filters.
	List(ctx context.Context, opts ListOptions) ([]*Workspace, error)

	// Archive transitions a workspace to archived state.
	Archive(ctx context.Context, id string) error

	// Join registers an agent in the workspace.
	Join(ctx context.Context, workspaceID, profileID string) (*AgentInfo, error)

	// Leave removes an agent from the workspace.
	Leave(ctx context.Context, workspaceID, profileID string) error

	// Remove forcefully removes an agent from the workspace (owner action).
	Remove(ctx context.Context, workspaceID, targetProfileID, actorProfileID string) error

	// SetRole changes an agent's role in the workspace.
	SetRole(ctx context.Context, workspaceID, profileID string, role AgentRole) error

	// Members returns all agents in a workspace.
	Members(ctx context.Context, workspaceID string) ([]AgentInfo, error)

	// SetActiveWorkspace sets the active workspace for CLI context.
	SetActiveWorkspace(ctx context.Context, workspaceID string) error

	// GetActiveWorkspace returns the currently active workspace ID.
	GetActiveWorkspace(ctx context.Context) (string, error)
}

// TaskManager handles inter-agent task operations.
type TaskManager interface {
	// Send creates and dispatches a task to a recipient agent.
	Send(ctx context.Context, workspaceID string, task TaskInfo) (*TaskInfo, error)

	// Get returns a task by ID.
	Get(ctx context.Context, workspaceID, taskID string) (*TaskInfo, error)

	// List returns tasks in a workspace matching optional filters.
	List(ctx context.Context, workspaceID string, opts ListOptions) ([]TaskInfo, error)

	// UpdateStatus transitions a task's status.
	UpdateStatus(ctx context.Context, workspaceID, taskID string, status TaskStatus, output json.RawMessage) error

	// Cancel cancels a pending or working task.
	Cancel(ctx context.Context, workspaceID, taskID string) error

	// History returns task history for a workspace, optionally filtered by agent.
	History(ctx context.Context, workspaceID string, opts ListOptions) ([]TaskInfo, error)
}

// CapabilityDiscovery handles agent capability registration and matching.
type CapabilityDiscovery interface {
	// Register registers an agent's capabilities in the workspace index.
	Register(ctx context.Context, workspaceID, profileID string, capabilities []string) error

	// Refresh re-fetches capabilities for an agent from their Agent Card.
	Refresh(ctx context.Context, workspaceID, profileID string) error

	// FindAgents returns agents matching a capability query.
	FindAgents(ctx context.Context, workspaceID string, query CapabilityQuery) ([]AgentMatch, error)

	// ListCapabilities returns capabilities for a specific agent or all agents.
	ListCapabilities(ctx context.Context, workspaceID, profileID string) ([]string, error)
}

// CapabilityQuery specifies criteria for finding agents.
type CapabilityQuery struct {
	Capability string `json:"capability,omitempty"`
	Task       string `json:"task,omitempty"` // fuzzy task description
	Role       AgentRole `json:"role,omitempty"`
}

// AgentMatch represents a matched agent with a relevance score.
type AgentMatch struct {
	Agent AgentInfo `json:"agent"`
	Score float64   `json:"score"` // 0.0-1.0 relevance
	Match string    `json:"match"` // which capability matched
}

// ConflictResolver detects and resolves conflicts in the workspace.
type ConflictResolver interface {
	// Detect scans for conflicts in the workspace.
	Detect(ctx context.Context, workspaceID string) ([]Conflict, error)

	// Resolve applies a resolution strategy to a conflict.
	Resolve(ctx context.Context, workspaceID, conflictID string, resolution ConflictResolution) error

	// ListConflicts returns active (unresolved) conflicts.
	ListConflicts(ctx context.Context, workspaceID string) ([]Conflict, error)

	// SetPolicy sets the resolution policy for a workspace or resource.
	SetPolicy(ctx context.Context, workspaceID string, policy PolicyConfig) error

	// GetPolicy returns the current resolution policy.
	GetPolicy(ctx context.Context, workspaceID string) (*PolicyConfig, error)
}

// ContextStore manages shared workspace context variables.
type ContextStore interface {
	// Set sets a context variable with ACL checks.
	Set(ctx context.Context, workspaceID, key, value, agentID string) error

	// Get returns a context variable value.
	Get(ctx context.Context, workspaceID, key string) (*ContextVariable, error)

	// List returns all context variables in the workspace.
	List(ctx context.Context, workspaceID string) ([]ContextVariable, error)

	// Delete removes a context variable.
	Delete(ctx context.Context, workspaceID, key, agentID string) error

	// History returns the mutation history for a key.
	History(ctx context.Context, workspaceID, key string) ([]ContextMutation, error)

	// SetACL sets the access control for a context variable.
	SetACL(ctx context.Context, workspaceID string, acl ACLEntry) error

	// GetACL returns the ACL for a context variable.
	GetACL(ctx context.Context, workspaceID, key string) (*ACLEntry, error)
}

// AuditLog records and queries workspace events.
type AuditLog interface {
	// Record adds an audit event.
	Record(ctx context.Context, event AuditEvent) error

	// Query returns audit events matching filters.
	Query(ctx context.Context, workspaceID string, opts AuditQueryOptions) ([]AuditEvent, error)
}

// AuditQueryOptions controls audit log queries.
type AuditQueryOptions struct {
	Actor  string `json:"actor,omitempty"`
	Event  string `json:"event,omitempty"` // glob pattern, e.g. "conflict.*"
	Since  string `json:"since,omitempty"` // duration string, e.g. "1h"
	Until  string `json:"until,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// ListOptions provides common list/filter parameters.
type ListOptions struct {
	Limit   int               `json:"limit,omitempty"`
	Offset  int               `json:"offset,omitempty"`
	Sort    string            `json:"sort,omitempty"`
	Reverse bool              `json:"reverse,omitempty"`
	Filters map[string]string `json:"filters,omitempty"`
}

// Storage defines persistence operations for workspaces.
type Storage interface {
	// SaveWorkspace persists a workspace to disk.
	SaveWorkspace(ws *Workspace) error

	// LoadWorkspace loads a workspace from disk.
	LoadWorkspace(id string) (*Workspace, error)

	// ListWorkspaces returns all workspace IDs.
	ListWorkspaces() ([]string, error)

	// DeleteWorkspace removes a workspace from disk.
	DeleteWorkspace(id string) error

	// SaveTasks persists task list for a workspace.
	SaveTasks(workspaceID string, tasks []TaskInfo) error

	// LoadTasks loads tasks for a workspace.
	LoadTasks(workspaceID string) ([]TaskInfo, error)

	// SaveConflicts persists conflicts for a workspace.
	SaveConflicts(workspaceID string, conflicts []Conflict) error

	// LoadConflicts loads conflicts for a workspace.
	LoadConflicts(workspaceID string) ([]Conflict, error)

	// SaveContext persists context variables for a workspace.
	SaveContext(workspaceID string, variables []ContextVariable) error

	// LoadContext loads context variables for a workspace.
	LoadContext(workspaceID string) ([]ContextVariable, error)

	// SaveAuditEvents persists audit events for a workspace.
	SaveAuditEvents(workspaceID string, events []AuditEvent) error

	// LoadAuditEvents loads audit events for a workspace.
	LoadAuditEvents(workspaceID string) ([]AuditEvent, error)

	// SaveActiveWorkspace persists the active workspace ID.
	SaveActiveWorkspace(id string) error

	// LoadActiveWorkspace returns the active workspace ID.
	LoadActiveWorkspace() (string, error)
}
