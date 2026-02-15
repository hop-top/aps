package collaboration

import "fmt"

// WorkspaceNotFoundError indicates a workspace does not exist.
type WorkspaceNotFoundError struct {
	ID string
}

func (e *WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace not found: %s", e.ID)
}

// AgentNotInWorkspaceError indicates an agent is not a member.
type AgentNotInWorkspaceError struct {
	ProfileID   string
	WorkspaceID string
}

func (e *AgentNotInWorkspaceError) Error() string {
	return fmt.Sprintf("agent %q is not in workspace %q", e.ProfileID, e.WorkspaceID)
}

// PermissionDeniedError indicates insufficient permissions.
type PermissionDeniedError struct {
	ProfileID  string
	Action     string
	Required   string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: agent %q cannot %s (requires %s)", e.ProfileID, e.Action, e.Required)
}

// ConflictError indicates a conflict was detected.
type ConflictError struct {
	ConflictID string
	Resource   string
	Message    string
}

func (e *ConflictError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("conflict on resource %q (id: %s)", e.Resource, e.ConflictID)
}

// TaskTimeoutError indicates a task exceeded its timeout.
type TaskTimeoutError struct {
	TaskID  string
	Timeout string
}

func (e *TaskTimeoutError) Error() string {
	return fmt.Sprintf("task %q timed out after %s", e.TaskID, e.Timeout)
}

// WorkspaceCapacityError indicates a workspace has reached max agents.
type WorkspaceCapacityError struct {
	WorkspaceID string
	MaxAgents   int
}

func (e *WorkspaceCapacityError) Error() string {
	return fmt.Sprintf("workspace %q at capacity (%d agents)", e.WorkspaceID, e.MaxAgents)
}
