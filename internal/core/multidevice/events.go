package multidevice

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of workspace event.
type EventType string

const (
	EventProfileCreated         EventType = "profile.created"
	EventProfileUpdated         EventType = "profile.updated"
	EventProfileDeleted         EventType = "profile.deleted"
	EventProfileConfigChanged   EventType = "profile.config_changed"
	EventActionCreated          EventType = "action.created"
	EventActionUpdated          EventType = "action.updated"
	EventActionDeleted          EventType = "action.deleted"
	EventActionExecuted         EventType = "action.executed"
	EventWorkspaceAccessed      EventType = "workspace.accessed"
	EventWorkspaceConfigChanged EventType = "workspace.config_changed"
	EventDeviceLinked           EventType = "device.linked"
	EventDeviceUnlinked         EventType = "device.unlinked"
	EventDevicePermChanged      EventType = "device.permissions_changed"
	EventDevicePresenceChanged  EventType = "device.presence_changed"
	EventDeviceSyncFailed       EventType = "device.sync_failed"
	EventConflictDetected       EventType = "conflict.detected"
	EventConflictResolved       EventType = "conflict.resolved"
)

// Category returns the top-level category of the event type for filtering.
func (e EventType) Category() string {
	parts := strings.SplitN(string(e), ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// WorkspaceEvent represents a single event that occurred in a workspace.
type WorkspaceEvent struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	DeviceID    string                 `json:"device_id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   EventType              `json:"event_type"`
	Version     int64                  `json:"version"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	CausedBy    string                 `json:"caused_by,omitempty"`
	Signature   string                 `json:"signature,omitempty"`
}

// NewEvent creates a new WorkspaceEvent with a generated UUID and current timestamp.
func NewEvent(workspaceID, deviceID string, eventType EventType, payload map[string]interface{}) *WorkspaceEvent {
	return &WorkspaceEvent{
		ID:          fmt.Sprintf("evt-%s", uuid.New().String()),
		WorkspaceID: workspaceID,
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		EventType:   eventType,
		Payload:     payload,
	}
}
