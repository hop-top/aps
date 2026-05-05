package collaboration

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

// WorkspaceState represents the lifecycle state of a workspace.
type WorkspaceState string

const (
	StateCreating WorkspaceState = "creating"
	StateActive   WorkspaceState = "active"
	StateClosing  WorkspaceState = "closing"
	StateArchived WorkspaceState = "archived"
)

// ValidWorkspaceStates enumerates all valid workspace states.
var ValidWorkspaceStates = []WorkspaceState{StateCreating, StateActive, StateClosing, StateArchived}

// Validate checks if the workspace state is valid.
func (s WorkspaceState) Validate() error {
	switch s {
	case StateCreating, StateActive, StateClosing, StateArchived:
		return nil
	default:
		return fmt.Errorf("invalid workspace state: %q", s)
	}
}

// AgentRole represents an agent's role within a workspace.
type AgentRole string

const (
	RoleOwner       AgentRole = "owner"
	RoleContributor AgentRole = "contributor"
	RoleObserver    AgentRole = "observer"
)

// ValidAgentRoles enumerates all valid roles.
var ValidAgentRoles = []AgentRole{RoleOwner, RoleContributor, RoleObserver}

// Validate checks if the agent role is valid.
func (r AgentRole) Validate() error {
	switch r {
	case RoleOwner, RoleContributor, RoleObserver:
		return nil
	default:
		return fmt.Errorf("invalid agent role: %q", r)
	}
}

// CanWrite returns true if the role has write permissions.
func (r AgentRole) CanWrite() bool {
	return r == RoleOwner || r == RoleContributor
}

// CanAdmin returns true if the role has admin permissions.
func (r AgentRole) CanAdmin() bool {
	return r == RoleOwner
}

// TaskStatus represents the state of an inter-agent task.
type TaskStatus string

const (
	TaskSubmitted TaskStatus = "submitted"
	TaskWorking   TaskStatus = "working"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// ValidTaskStatuses enumerates all valid task statuses.
var ValidTaskStatuses = []TaskStatus{TaskSubmitted, TaskWorking, TaskCompleted, TaskFailed, TaskCancelled}

// Validate checks if the task status is valid.
func (s TaskStatus) Validate() error {
	switch s {
	case TaskSubmitted, TaskWorking, TaskCompleted, TaskFailed, TaskCancelled:
		return nil
	default:
		return fmt.Errorf("invalid task status: %q", s)
	}
}

// IsTerminal returns true if the task is in a final state.
func (s TaskStatus) IsTerminal() bool {
	return s == TaskCompleted || s == TaskFailed || s == TaskCancelled
}

// ConflictType represents the type of conflict detected.
type ConflictType string

const (
	ConflictWrite      ConflictType = "write"
	ConflictOrdering   ConflictType = "ordering"
	ConflictLock       ConflictType = "lock"
	ConflictCapability ConflictType = "capability"
)

// ValidConflictTypes enumerates all valid conflict types.
var ValidConflictTypes = []ConflictType{ConflictWrite, ConflictOrdering, ConflictLock, ConflictCapability}

// Validate checks if the conflict type is valid.
func (t ConflictType) Validate() error {
	switch t {
	case ConflictWrite, ConflictOrdering, ConflictLock, ConflictCapability:
		return nil
	default:
		return fmt.Errorf("invalid conflict type: %q", t)
	}
}

// SessionState represents the lifecycle state of an agent session.
type SessionState string

const (
	SessionJoining SessionState = "joining"
	SessionActive  SessionState = "active"
	SessionLeaving SessionState = "leaving"
	SessionClosed  SessionState = "closed"
)

// Permission represents a resource-level permission.
type Permission string

const (
	PermRead   Permission = "read"
	PermWrite  Permission = "write"
	PermDelete Permission = "delete"
	PermAdmin  Permission = "admin"
)

// ResolutionStrategy names a conflict resolution approach.
type ResolutionStrategy string

const (
	StrategyConsensus   ResolutionStrategy = "consensus"
	StrategyPriority    ResolutionStrategy = "priority"
	StrategyVoting      ResolutionStrategy = "voting"
	StrategyArbitration ResolutionStrategy = "arbitration"
	StrategyKeepFirst   ResolutionStrategy = "keep-first"
	StrategyKeepLast    ResolutionStrategy = "keep-last"
	StrategyRollback    ResolutionStrategy = "rollback"
	StrategyMerge       ResolutionStrategy = "merge"
)

// WorkspaceConfig holds the configuration for a collaboration workspace.
type WorkspaceConfig struct {
	Name              string             `json:"name" yaml:"name"`
	Description       string             `json:"description,omitempty" yaml:"description,omitempty"`
	OwnerProfileID    string             `json:"owner_profile_id" yaml:"owner_profile_id"`
	DefaultPolicy     ResolutionStrategy `json:"default_policy,omitempty" yaml:"default_policy,omitempty"`
	HeartbeatInterval time.Duration      `json:"heartbeat_interval,omitempty" yaml:"heartbeat_interval,omitempty"`
	SessionTimeout    time.Duration      `json:"session_timeout,omitempty" yaml:"session_timeout,omitempty"`
	MaxAgents         int                `json:"max_agents,omitempty" yaml:"max_agents,omitempty"`
}

// Validate checks required fields and sets defaults.
func (c *WorkspaceConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("workspace name is required")
	}
	if c.OwnerProfileID == "" {
		return fmt.Errorf("owner profile ID is required")
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 10 * time.Second
	}
	if c.SessionTimeout == 0 {
		c.SessionTimeout = 30 * time.Second
	}
	if c.MaxAgents == 0 {
		c.MaxAgents = 50
	}
	if c.DefaultPolicy == "" {
		c.DefaultPolicy = StrategyPriority
	}
	return nil
}

// AgentInfo holds metadata about an agent in a workspace.
type AgentInfo struct {
	ProfileID    string            `json:"profile_id" yaml:"profile_id"`
	DisplayName  string            `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Role         AgentRole         `json:"role" yaml:"role"`
	Capabilities []string          `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	JoinedAt     time.Time         `json:"joined_at" yaml:"joined_at"`
	LastSeen     time.Time         `json:"last_seen" yaml:"last_seen"`
	SessionID    string            `json:"session_id,omitempty" yaml:"session_id,omitempty"`
	Status       string            `json:"status" yaml:"status"` // "online", "offline"
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// TaskInfo represents an inter-agent task within a workspace.
type TaskInfo struct {
	ID           string          `json:"id" yaml:"id"`
	WorkspaceID  string          `json:"workspace_id" yaml:"workspace_id"`
	SenderID     string          `json:"sender_id" yaml:"sender_id"`
	RecipientID  string          `json:"recipient_id" yaml:"recipient_id"`
	Action       string          `json:"action" yaml:"action"`
	Input        json.RawMessage `json:"input,omitempty" yaml:"input,omitempty"`
	Output       json.RawMessage `json:"output,omitempty" yaml:"output,omitempty"`
	Status       TaskStatus      `json:"status" yaml:"status"`
	Error        string          `json:"error,omitempty" yaml:"error,omitempty"`
	Dependencies []string        `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	CreatedAt    time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" yaml:"updated_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Timeout      time.Duration   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Conflict represents a detected conflict in the workspace.
type Conflict struct {
	ID           string             `json:"id" yaml:"id"`
	WorkspaceID  string             `json:"workspace_id" yaml:"workspace_id"`
	Type         ConflictType       `json:"type" yaml:"type"`
	Resource     string             `json:"resource" yaml:"resource"`
	AgentIDs     []string           `json:"agent_ids" yaml:"agent_ids"`
	Description  string             `json:"description,omitempty" yaml:"description,omitempty"`
	Resolution   *ConflictResolution `json:"resolution,omitempty" yaml:"resolution,omitempty"`
	DetectedAt   time.Time          `json:"detected_at" yaml:"detected_at"`
	ResolvedAt   *time.Time         `json:"resolved_at,omitempty" yaml:"resolved_at,omitempty"`
}

// IsResolved returns true if the conflict has been resolved.
func (c *Conflict) IsResolved() bool {
	return c.Resolution != nil
}

// ConflictResolution records how a conflict was resolved.
type ConflictResolution struct {
	Strategy   ResolutionStrategy `json:"strategy" yaml:"strategy"`
	ResolvedBy string             `json:"resolved_by" yaml:"resolved_by"` // agent ID or "system"
	Details    string             `json:"details,omitempty" yaml:"details,omitempty"`
	Timestamp  time.Time          `json:"timestamp" yaml:"timestamp"`
}

// TaskDependency defines ordering between tasks.
type TaskDependency struct {
	TaskID       string `json:"task_id" yaml:"task_id"`
	DependsOnID  string `json:"depends_on_id" yaml:"depends_on_id"`
}

// ContextVisibility names the per-variable visibility scope.
//
// Default zero-value reads as VisibilityShared so every variable
// stored before T-1309 (which had no visibility field) round-trips
// as a workspace-wide variable — see NormalizeVisibility.
type ContextVisibility string

const (
	// VisibilityShared variables are visible to every member of the
	// workspace, gated only by the per-key ACL (T-1306). This is the
	// default and matches the pre-T-1309 behaviour.
	VisibilityShared ContextVisibility = "shared"
	// VisibilityPrivate variables are visible only to the writer
	// (the profile recorded in UpdatedBy at the time of the last
	// successful Set). Reads by other profiles behave as
	// "not found" so existence does not leak.
	VisibilityPrivate ContextVisibility = "private"
)

// NormalizeVisibility maps the zero-value to VisibilityShared. Used
// at every read seam so existing yaml/json without a visibility field
// reads as shared without a migration step.
func NormalizeVisibility(v ContextVisibility) ContextVisibility {
	if v == "" {
		return VisibilityShared
	}
	return v
}

// ContextVariable represents a key-value variable in workspace shared context.
type ContextVariable struct {
	Key       string    `json:"key" yaml:"key"`
	Value     string    `json:"value" yaml:"value"`
	Version   int       `json:"version" yaml:"version"`
	UpdatedBy string    `json:"updated_by" yaml:"updated_by"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// Visibility scopes the variable to the workspace ("shared",
	// default) or to the writer profile only ("private"). The zero
	// value reads as VisibilityShared via NormalizeVisibility, so
	// existing on-disk state without the field continues to work
	// unchanged. Set explicitly via WithVisibility on Set/SetWithContext.
	Visibility ContextVisibility `json:"visibility,omitempty" yaml:"visibility,omitempty"`
}

// EffectiveVisibility returns Visibility normalized to VisibilityShared
// when zero-valued. Use this at every read site that branches on
// visibility — never compare ContextVariable.Visibility directly.
func (v ContextVariable) EffectiveVisibility() ContextVisibility {
	return NormalizeVisibility(v.Visibility)
}

// IsVisibleTo returns true when profileID may observe v under T-1309
// visibility rules: shared variables are visible to every profile;
// private variables are visible only to UpdatedBy.
func (v ContextVariable) IsVisibleTo(profileID string) bool {
	if v.EffectiveVisibility() == VisibilityShared {
		return true
	}
	return v.UpdatedBy == profileID
}

// ContextMutation records a change to a context variable.
type ContextMutation struct {
	Key      string `json:"key" yaml:"key"`
	OldValue string `json:"old_value,omitempty" yaml:"old_value,omitempty"`
	NewValue string `json:"new_value" yaml:"new_value"`
	Version  int    `json:"version" yaml:"version"`
	AgentID  string `json:"agent_id" yaml:"agent_id"`
	// Note carries the optional --note|-n value supplied by the
	// operator at the CLI layer (T-1291). Empty when the flag is unset.
	Note      string    `json:"note,omitempty" yaml:"note,omitempty"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// AuditEvent records a state-changing operation in the workspace.
type AuditEvent struct {
	ID          string    `json:"id" yaml:"id"`
	WorkspaceID string    `json:"workspace_id" yaml:"workspace_id"`
	Actor       string    `json:"actor" yaml:"actor"` // agent ID or "system"
	Event       string    `json:"event" yaml:"event"` // "agent.join", "task.create", etc.
	Resource    string    `json:"resource,omitempty" yaml:"resource,omitempty"`
	Details     string    `json:"details,omitempty" yaml:"details,omitempty"`
	// Note carries the optional --note|-n value supplied by the
	// operator at the CLI layer (T-1291). Empty when the flag is unset.
	Note      string    `json:"note,omitempty" yaml:"note,omitempty"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// PolicyConfig holds conflict resolution policy for a workspace or resource.
type PolicyConfig struct {
	Default   ResolutionStrategy            `json:"default" yaml:"default"`
	Overrides map[string]ResolutionStrategy `json:"overrides,omitempty" yaml:"overrides,omitempty"` // resource key -> strategy
}

// GetStrategy returns the resolution strategy for a resource, falling back to default.
func (p *PolicyConfig) GetStrategy(resource string) ResolutionStrategy {
	if s, ok := p.Overrides[resource]; ok {
		return s
	}
	return p.Default
}

// ACLEntry defines access control for a context variable.
type ACLEntry struct {
	Key         string       `json:"key" yaml:"key"`
	Permissions map[AgentRole][]Permission `json:"permissions" yaml:"permissions"`
}

// DefaultACL returns the default ACL giving owners full access, contributors read/write, observers read-only.
func DefaultACL(key string) ACLEntry {
	return ACLEntry{
		Key: key,
		Permissions: map[AgentRole][]Permission{
			RoleOwner:       {PermRead, PermWrite, PermDelete, PermAdmin},
			RoleContributor: {PermRead, PermWrite},
			RoleObserver:    {PermRead},
		},
	}
}

// HasPermission checks if a role has a specific permission on this entry.
func (a *ACLEntry) HasPermission(role AgentRole, perm Permission) bool {
	perms, ok := a.Permissions[role]
	if !ok {
		return false
	}
	return slices.Contains(perms, perm)
}
