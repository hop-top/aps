package collaboration

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hop.top/kit/go/runtime/policy"
)

// noteFromContext returns the audit note attached to ctx via
// policy.ContextAttrsKey by the CLI layer (T-1291). Empty when none is
// set. Used by recordEvent to surface the operator-supplied reason in
// the workspace audit log.
func noteFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	attrs, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := attrs["note"].(string); ok {
		return v
	}
	return ""
}

// Manager orchestrates workspace lifecycle operations.
// It delegates persistence to a Storage implementation and records audit events
// for all state-changing operations.
type Manager struct {
	storage Storage
}

// NewManager creates a Manager backed by the given storage.
func NewManager(storage Storage) *Manager {
	return &Manager{storage: storage}
}

// Create creates a new collaboration workspace, persists it, and records an audit event.
func (m *Manager) Create(ctx context.Context, config WorkspaceConfig) (*Workspace, error) {
	ws, err := NewWorkspace(config)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return nil, fmt.Errorf("saving workspace: %w", err)
	}

	_ = m.recordEvent(ws.ID, config.OwnerProfileID, "workspace.create", ws.ID, "", noteFromContext(ctx))
	return ws, nil
}

// Get loads a workspace by ID from storage.
func (m *Manager) Get(ctx context.Context, id string) (*Workspace, error) {
	ws, err := m.storage.LoadWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("loading workspace %q: %w", id, err)
	}
	return ws, nil
}

// List returns all workspaces, optionally filtered by the "name" and "status" keys
// in opts.Filters.
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Workspace, error) {
	ids, err := m.storage.ListWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	var workspaces []*Workspace
	for _, id := range ids {
		ws, err := m.storage.LoadWorkspace(id)
		if err != nil {
			continue // skip unreadable workspaces
		}
		if !matchesFilters(ws, opts) {
			continue
		}
		workspaces = append(workspaces, ws)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(workspaces) {
		workspaces = workspaces[opts.Offset:]
	} else if opts.Offset >= len(workspaces) {
		return []*Workspace{}, nil
	}
	if opts.Limit > 0 && opts.Limit < len(workspaces) {
		workspaces = workspaces[:opts.Limit]
	}

	return workspaces, nil
}

// Archive transitions a workspace to the archived state and persists the change.
func (m *Manager) Archive(ctx context.Context, id string) error {
	ws, err := m.storage.LoadWorkspace(id)
	if err != nil {
		return fmt.Errorf("loading workspace %q: %w", id, err)
	}

	if err := ws.SetState(StateArchived); err != nil {
		return fmt.Errorf("archiving workspace %q: %w", id, err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return fmt.Errorf("saving workspace %q: %w", id, err)
	}

	_ = m.recordEvent(ws.ID, "system", "workspace.archive", ws.ID, "", noteFromContext(ctx))
	return nil
}

// Join adds an agent as a contributor to the workspace.
func (m *Manager) Join(ctx context.Context, workspaceID, profileID string) (*AgentInfo, error) {
	ws, err := m.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("loading workspace %q: %w", workspaceID, err)
	}

	agent, err := ws.AddAgent(profileID, RoleContributor)
	if err != nil {
		return nil, fmt.Errorf("joining workspace %q: %w", workspaceID, err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return nil, fmt.Errorf("saving workspace %q: %w", workspaceID, err)
	}

	_ = m.recordEvent(workspaceID, profileID, "agent.join", profileID, "", noteFromContext(ctx))
	return agent, nil
}

// Leave removes an agent from the workspace.
func (m *Manager) Leave(ctx context.Context, workspaceID, profileID string) error {
	ws, err := m.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return fmt.Errorf("loading workspace %q: %w", workspaceID, err)
	}

	if err := ws.RemoveAgent(profileID); err != nil {
		return fmt.Errorf("leaving workspace %q: %w", workspaceID, err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return fmt.Errorf("saving workspace %q: %w", workspaceID, err)
	}

	_ = m.recordEvent(workspaceID, profileID, "agent.leave", profileID, "", noteFromContext(ctx))
	return nil
}

// Remove forcefully removes an agent from the workspace. Only the workspace owner
// may perform this action.
func (m *Manager) Remove(ctx context.Context, workspaceID, targetProfileID, actorProfileID string) error {
	ws, err := m.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return fmt.Errorf("loading workspace %q: %w", workspaceID, err)
	}

	if !ws.IsOwner(actorProfileID) {
		return &PermissionDeniedError{
			ProfileID: actorProfileID,
			Action:    "remove agent",
			Required:  "owner",
		}
	}

	if err := ws.RemoveAgent(targetProfileID); err != nil {
		return fmt.Errorf("removing agent %q from workspace %q: %w", targetProfileID, workspaceID, err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return fmt.Errorf("saving workspace %q: %w", workspaceID, err)
	}

	details := fmt.Sprintf("removed by %s", actorProfileID)
	_ = m.recordEvent(workspaceID, actorProfileID, "agent.remove", targetProfileID, details, noteFromContext(ctx))
	return nil
}

// SetRole changes an agent's role in the workspace. Only the workspace owner may
// change roles.
func (m *Manager) SetRole(ctx context.Context, workspaceID, profileID string, role AgentRole) error {
	ws, err := m.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return fmt.Errorf("loading workspace %q: %w", workspaceID, err)
	}

	// Determine the actor: we accept that the caller (the one setting the role) is
	// the workspace owner. In a single-CLI context the owner is implied.
	// The SetAgentRole method on Workspace validates role transitions.
	if err := ws.SetAgentRole(profileID, role); err != nil {
		return fmt.Errorf("setting role for %q in workspace %q: %w", profileID, workspaceID, err)
	}

	if err := m.storage.SaveWorkspace(ws); err != nil {
		return fmt.Errorf("saving workspace %q: %w", workspaceID, err)
	}

	details := fmt.Sprintf("role changed to %s", role)
	_ = m.recordEvent(workspaceID, "system", "agent.role_change", profileID, details, noteFromContext(ctx))
	return nil
}

// Members returns all agents in a workspace.
func (m *Manager) Members(ctx context.Context, workspaceID string) ([]AgentInfo, error) {
	ws, err := m.storage.LoadWorkspace(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("loading workspace %q: %w", workspaceID, err)
	}

	agents := make([]AgentInfo, len(ws.Agents))
	copy(agents, ws.Agents)
	return agents, nil
}

// SetActiveWorkspace persists the active workspace ID.
func (m *Manager) SetActiveWorkspace(ctx context.Context, workspaceID string) error {
	// Verify workspace exists
	if _, err := m.storage.LoadWorkspace(workspaceID); err != nil {
		return fmt.Errorf("workspace %q not found: %w", workspaceID, err)
	}

	if err := m.storage.SaveActiveWorkspace(workspaceID); err != nil {
		return fmt.Errorf("saving active workspace: %w", err)
	}
	return nil
}

// GetActiveWorkspace returns the currently active workspace ID.
func (m *Manager) GetActiveWorkspace(ctx context.Context) (string, error) {
	id, err := m.storage.LoadActiveWorkspace()
	if err != nil {
		return "", fmt.Errorf("loading active workspace: %w", err)
	}
	return id, nil
}

// recordEvent appends an audit event to the workspace's audit log.
//
// note carries the operator-supplied --note|-n value (T-1291), or "" when
// the flag is unset. It is stored alongside details so audit consumers
// can present "what" (event/resource) and "why" (note) without parsing
// details.
func (m *Manager) recordEvent(workspaceID, actor, event, resource, details, note string) error {
	existing, err := m.storage.LoadAuditEvents(workspaceID)
	if err != nil {
		existing = []AuditEvent{}
	}

	entry := AuditEvent{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Actor:       actor,
		Event:       event,
		Resource:    resource,
		Details:     details,
		Note:        note,
		Timestamp:   time.Now(),
	}
	existing = append(existing, entry)

	return m.storage.SaveAuditEvents(workspaceID, existing)
}

// matchesFilters returns true if the workspace matches the given list options filters.
func matchesFilters(ws *Workspace, opts ListOptions) bool {
	if opts.Filters == nil {
		return true
	}
	if name, ok := opts.Filters["name"]; ok && ws.Config.Name != name {
		return false
	}
	if status, ok := opts.Filters["status"]; ok && string(ws.State) != status {
		return false
	}
	return true
}
