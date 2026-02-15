package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageRouter handles inter-agent task creation, routing, and lifecycle management.
type MessageRouter struct {
	mu      sync.RWMutex
	storage Storage
	audit   AuditLog
}

// NewMessageRouter creates a new message router.
func NewMessageRouter(storage Storage, audit AuditLog) *MessageRouter {
	return &MessageRouter{
		storage: storage,
		audit:   audit,
	}
}

// Send creates and dispatches a task to a recipient agent.
func (mr *MessageRouter) Send(ctx context.Context, workspaceID string, task TaskInfo) (*TaskInfo, error) {
	if task.Action == "" {
		return nil, fmt.Errorf("task action is required")
	}
	if task.RecipientID == "" {
		return nil, fmt.Errorf("task recipient is required")
	}
	if task.SenderID == "" {
		return nil, fmt.Errorf("task sender is required")
	}

	now := time.Now()
	task.ID = uuid.New().String()
	task.WorkspaceID = workspaceID
	task.Status = TaskSubmitted
	task.CreatedAt = now
	task.UpdatedAt = now

	if task.Timeout == 0 {
		task.Timeout = 5 * time.Minute
	}

	// Persist the task
	tasks, err := mr.storage.LoadTasks(workspaceID)
	if err != nil {
		tasks = []TaskInfo{}
	}
	tasks = append(tasks, task)
	if err := mr.storage.SaveTasks(workspaceID, tasks); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	if mr.audit != nil {
		mr.audit.Record(ctx, AuditEvent{
			ID:          uuid.New().String(),
			WorkspaceID: workspaceID,
			Actor:       task.SenderID,
			Event:       "task.create",
			Resource:    task.ID,
			Details:     fmt.Sprintf("%s -> %s: %s", task.SenderID, task.RecipientID, task.Action),
			Timestamp:   now,
		})
	}

	return &task, nil
}

// Get returns a task by ID.
func (mr *MessageRouter) Get(ctx context.Context, workspaceID, taskID string) (*TaskInfo, error) {
	tasks, err := mr.storage.LoadTasks(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	for i := range tasks {
		if tasks[i].ID == taskID {
			return &tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %q not found in workspace %q", taskID, workspaceID)
}

// List returns tasks in a workspace matching optional filters.
func (mr *MessageRouter) List(ctx context.Context, workspaceID string, opts ListOptions) ([]TaskInfo, error) {
	tasks, err := mr.storage.LoadTasks(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	filtered := filterTasks(tasks, opts)
	return paginate(filtered, opts.Offset, opts.Limit), nil
}

// UpdateStatus transitions a task's status.
func (mr *MessageRouter) UpdateStatus(ctx context.Context, workspaceID, taskID string, status TaskStatus, output json.RawMessage) error {
	if err := status.Validate(); err != nil {
		return err
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	tasks, err := mr.storage.LoadTasks(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	idx := -1
	for i := range tasks {
		if tasks[i].ID == taskID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("task %q not found", taskID)
	}

	task := &tasks[idx]
	if err := validateTransition(task.Status, status); err != nil {
		return err
	}

	now := time.Now()
	task.Status = status
	task.UpdatedAt = now
	if output != nil {
		task.Output = output
	}
	if status.IsTerminal() {
		task.CompletedAt = &now
	}

	if err := mr.storage.SaveTasks(workspaceID, tasks); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	if mr.audit != nil {
		mr.audit.Record(ctx, AuditEvent{
			ID:          uuid.New().String(),
			WorkspaceID: workspaceID,
			Actor:       task.RecipientID,
			Event:       "task." + string(status),
			Resource:    taskID,
			Timestamp:   now,
		})
	}

	return nil
}

// Cancel cancels a pending or working task.
func (mr *MessageRouter) Cancel(ctx context.Context, workspaceID, taskID string) error {
	return mr.UpdateStatus(ctx, workspaceID, taskID, TaskCancelled, nil)
}

// History returns task history for a workspace.
func (mr *MessageRouter) History(ctx context.Context, workspaceID string, opts ListOptions) ([]TaskInfo, error) {
	return mr.List(ctx, workspaceID, opts)
}

// validateTransition checks if a task status transition is valid.
func validateTransition(from, to TaskStatus) error {
	if from.IsTerminal() {
		return fmt.Errorf("cannot transition from terminal status %q", from)
	}

	valid := map[TaskStatus][]TaskStatus{
		TaskSubmitted: {TaskWorking, TaskCancelled},
		TaskWorking:   {TaskCompleted, TaskFailed, TaskCancelled},
	}

	allowed, ok := valid[from]
	if !ok {
		return fmt.Errorf("unknown status %q", from)
	}

	if slices.Contains(allowed, to) {
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", from, to)
}

// filterTasks applies list option filters to tasks.
func filterTasks(tasks []TaskInfo, opts ListOptions) []TaskInfo {
	if len(opts.Filters) == 0 {
		return tasks
	}

	var filtered []TaskInfo
	for _, t := range tasks {
		match := true
		if status, ok := opts.Filters["status"]; ok && string(t.Status) != status {
			match = false
		}
		if agent, ok := opts.Filters["agent"]; ok && t.SenderID != agent && t.RecipientID != agent {
			match = false
		}
		if action, ok := opts.Filters["action"]; ok && t.Action != action {
			match = false
		}
		if match {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// paginate applies offset and limit to a slice.
func paginate[T any](items []T, offset, limit int) []T {
	if offset > 0 && offset < len(items) {
		items = items[offset:]
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}
