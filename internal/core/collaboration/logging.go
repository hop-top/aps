package collaboration

import (
	"hop.top/aps/internal/logging"
)

// EventLogger provides structured operational logging for collaboration events.
// It uses the global charm.land/log/v2 logger for stderr output.
// This is distinct from WorkspaceAuditLog which provides persistent, queryable records.
type EventLogger struct {
	logger *logging.Logger
}

// NewEventLogger creates an event logger using the global logger.
func NewEventLogger() *EventLogger {
	return &EventLogger{logger: logging.GetLogger()}
}

// WorkspaceCreated logs a workspace creation event.
func (el *EventLogger) WorkspaceCreated(workspaceID, name, owner string) {
	el.logger.Info("workspace created",
		"workspace_id", workspaceID,
		"name", name,
		"owner", owner,
	)
}

// WorkspaceArchived logs a workspace archival event.
func (el *EventLogger) WorkspaceArchived(workspaceID string) {
	el.logger.Info("workspace archived", "workspace_id", workspaceID)
}

// AgentJoined logs an agent joining a workspace.
func (el *EventLogger) AgentJoined(workspaceID, profileID string, role AgentRole) {
	el.logger.Info("agent joined workspace",
		"workspace_id", workspaceID,
		"profile_id", profileID,
		"role", string(role),
	)
}

// AgentLeft logs an agent leaving a workspace.
func (el *EventLogger) AgentLeft(workspaceID, profileID string) {
	el.logger.Info("agent left workspace",
		"workspace_id", workspaceID,
		"profile_id", profileID,
	)
}

// AgentRemoved logs a forced agent removal.
func (el *EventLogger) AgentRemoved(workspaceID, targetID, actorID string) {
	el.logger.Warn("agent removed from workspace",
		"workspace_id", workspaceID,
		"target", targetID,
		"actor", actorID,
	)
}

// RoleChanged logs an agent role change.
func (el *EventLogger) RoleChanged(workspaceID, profileID string, role AgentRole) {
	el.logger.Info("agent role changed",
		"workspace_id", workspaceID,
		"profile_id", profileID,
		"role", string(role),
	)
}

// TaskCreated logs a task creation event.
func (el *EventLogger) TaskCreated(workspaceID, taskID, sender, recipient, action string) {
	el.logger.Debug("task created",
		"workspace_id", workspaceID,
		"task_id", taskID,
		"sender", sender,
		"recipient", recipient,
		"action", action,
	)
}

// TaskStatusChanged logs a task status transition.
func (el *EventLogger) TaskStatusChanged(workspaceID, taskID string, status TaskStatus) {
	el.logger.Debug("task status changed",
		"workspace_id", workspaceID,
		"task_id", taskID,
		"status", string(status),
	)
}

// ConflictDetected logs a detected conflict.
func (el *EventLogger) ConflictDetected(workspaceID, conflictID string, conflictType ConflictType, resource string) {
	el.logger.Warn("conflict detected",
		"workspace_id", workspaceID,
		"conflict_id", conflictID,
		"type", string(conflictType),
		"resource", resource,
	)
}

// ConflictResolved logs a resolved conflict.
func (el *EventLogger) ConflictResolved(workspaceID, conflictID string, strategy ResolutionStrategy) {
	el.logger.Info("conflict resolved",
		"workspace_id", workspaceID,
		"conflict_id", conflictID,
		"strategy", string(strategy),
	)
}

// ContextSet logs a context variable being set.
func (el *EventLogger) ContextSet(workspaceID, key, agentID string, version int) {
	el.logger.Debug("context variable set",
		"workspace_id", workspaceID,
		"key", key,
		"agent_id", agentID,
		"version", version,
	)
}

// ContextDeleted logs a context variable being deleted.
func (el *EventLogger) ContextDeleted(workspaceID, key, agentID string) {
	el.logger.Debug("context variable deleted",
		"workspace_id", workspaceID,
		"key", key,
		"agent_id", agentID,
	)
}

// OperationError logs an error during a collaboration operation.
func (el *EventLogger) OperationError(operation string, err error, fields ...any) {
	el.logger.Error(operation, err, fields...)
}
