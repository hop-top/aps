package collaboration

import (
	"context"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkspaceAuditLog records and queries workspace events.
type WorkspaceAuditLog struct {
	mu      sync.RWMutex
	storage Storage
}

// NewWorkspaceAuditLog creates an audit log backed by storage.
func NewWorkspaceAuditLog(storage Storage) *WorkspaceAuditLog {
	return &WorkspaceAuditLog{storage: storage}
}

// Record adds an audit event.
func (al *WorkspaceAuditLog) Record(_ context.Context, event AuditEvent) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	events, err := al.storage.LoadAuditEvents(event.WorkspaceID)
	if err != nil {
		events = []AuditEvent{}
	}

	events = append(events, event)
	return al.storage.SaveAuditEvents(event.WorkspaceID, events)
}

// Query returns audit events matching filters.
func (al *WorkspaceAuditLog) Query(_ context.Context, workspaceID string, opts AuditQueryOptions) ([]AuditEvent, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	events, err := al.storage.LoadAuditEvents(workspaceID)
	if err != nil {
		return nil, err
	}

	filtered := filterAuditEvents(events, opts)

	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	return paginate(filtered, opts.Offset, opts.Limit), nil
}

// filterAuditEvents applies query options to filter events.
func filterAuditEvents(events []AuditEvent, opts AuditQueryOptions) []AuditEvent {
	var filtered []AuditEvent

	var sinceTime, untilTime time.Time
	if opts.Since != "" {
		if d, err := time.ParseDuration(opts.Since); err == nil {
			sinceTime = time.Now().Add(-d)
		}
	}
	if opts.Until != "" {
		if d, err := time.ParseDuration(opts.Until); err == nil {
			untilTime = time.Now().Add(-d)
		}
	}

	for _, e := range events {
		if opts.Actor != "" && e.Actor != opts.Actor {
			continue
		}
		if opts.Event != "" && !matchEventPattern(e.Event, opts.Event) {
			continue
		}
		if !sinceTime.IsZero() && e.Timestamp.Before(sinceTime) {
			continue
		}
		if !untilTime.IsZero() && e.Timestamp.After(untilTime) {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered
}

// matchEventPattern matches an event name against a glob pattern.
// Supports "*" as wildcard, e.g. "conflict.*" matches "conflict.detect".
func matchEventPattern(event, pattern string) bool {
	matched, err := path.Match(pattern, event)
	if err != nil {
		return strings.Contains(event, strings.TrimSuffix(pattern, "*"))
	}
	return matched
}
