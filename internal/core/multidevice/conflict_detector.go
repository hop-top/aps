package multidevice

import (
	"fmt"
	"time"
)

// ConflictType classifies the kind of conflict detected between events.
type ConflictType string

const (
	// ConflictConcurrentWrite indicates two devices modified the same
	// resource without knowledge of each other's changes.
	ConflictConcurrentWrite ConflictType = "concurrent_write"
	// ConflictOrdering indicates events arrived out of their expected
	// causal order.
	ConflictOrdering ConflictType = "ordering"
	// ConflictSemantic indicates logically incompatible operations on the
	// same resource (e.g., delete after update).
	ConflictSemantic ConflictType = "semantic"
	// ConflictMetadata indicates conflicting metadata changes (e.g.,
	// permission updates from different devices).
	ConflictMetadata ConflictType = "metadata"
)

// ConflictStatus tracks the lifecycle of a conflict.
type ConflictStatus string

const (
	// ConflictPending means the conflict has been detected but not yet resolved.
	ConflictPending ConflictStatus = "pending"
	// ConflictAutoResolved means the conflict was automatically resolved
	// (e.g., via last-write-wins).
	ConflictAutoResolved ConflictStatus = "auto_resolved"
	// ConflictManual means the conflict requires manual resolution by a user.
	ConflictManual ConflictStatus = "manual"
	// ConflictResolved means the conflict has been fully resolved.
	ConflictResolved ConflictStatus = "resolved"
)

// Conflict represents a detected conflict between workspace events.
type Conflict struct {
	ID          string              `json:"id"`
	WorkspaceID string              `json:"workspace_id"`
	Type        ConflictType        `json:"type"`
	Status      ConflictStatus      `json:"status"`
	Resource    string              `json:"resource"`
	Events      []*WorkspaceEvent   `json:"events"`
	DetectedAt  time.Time           `json:"detected_at"`
	ResolvedAt  *time.Time          `json:"resolved_at,omitempty"`
	Resolution  *ConflictResolution `json:"resolution,omitempty"`
}

// ConflictResolution records how a conflict was resolved.
type ConflictResolution struct {
	Strategy    string                 `json:"strategy"`                // "lww", "manual", "ot"
	WinnerEvent string                 `json:"winner_event,omitempty"` // ID of the winning event
	Result      map[string]interface{} `json:"result,omitempty"`       // merged/chosen values
	ResolvedBy  string                 `json:"resolved_by,omitempty"`  // device ID or "auto"
}

// ConflictDetector examines incoming events against the event store to find
// conflicts -- situations where two or more devices have concurrently
// modified the same resource.
type ConflictDetector struct {
	store *EventStore
}

// NewConflictDetector creates a detector backed by the given event store.
func NewConflictDetector(store *EventStore) *ConflictDetector {
	return &ConflictDetector{store: store}
}

// Detect checks whether an incoming event conflicts with any previously
// stored events. The detection algorithm:
//
//  1. Extract the resource identifier from the event payload.
//  2. Retrieve recent events that modified the same resource.
//  3. Identify events from different devices with overlapping versions
//     (concurrent modifications).
//  4. Classify the conflict type based on the nature of modifications.
//
// Returns nil if no conflict is detected.
func (d *ConflictDetector) Detect(event *WorkspaceEvent) (*Conflict, error) {
	if event == nil {
		return nil, fmt.Errorf("event must not be nil")
	}

	resource := extractResourceFromEvent(event)
	if resource == "" {
		// Events without a resource identifier cannot conflict.
		return nil, nil
	}

	// Query recent events on the same resource within the same workspace.
	recentEvents, err := d.store.QueryByResource(event.WorkspaceID, resource)
	if err != nil {
		return nil, fmt.Errorf("querying events for resource %q: %w", resource, err)
	}

	// Gather events from other devices that might conflict.
	var conflicting []*WorkspaceEvent
	for _, stored := range recentEvents {
		if stored.DeviceID == event.DeviceID {
			continue
		}
		// Two events conflict when they share a version or when the
		// incoming event does not causally follow the stored one.
		if stored.Version >= event.Version || isOverlapping(stored, event) {
			conflicting = append(conflicting, stored)
		}
	}

	if len(conflicting) == 0 {
		return nil, nil
	}

	conflictType := classifyConflict(event, conflicting)

	conflict := &Conflict{
		ID:          fmt.Sprintf("cnfl_%d", time.Now().UnixNano()),
		WorkspaceID: event.WorkspaceID,
		Type:        conflictType,
		Status:      ConflictPending,
		Resource:    resource,
		Events:      append(conflicting, event),
		DetectedAt:  time.Now(),
	}

	return conflict, nil
}

// isOverlapping determines whether two events represent concurrent
// modifications (neither causally precedes the other).
func isOverlapping(a, b *WorkspaceEvent) bool {
	// If events share the same version they are concurrent by definition.
	if a.Version == b.Version {
		return true
	}

	// If events occurred very close together (within 5 seconds) across
	// different devices, treat them as potentially concurrent.
	const concurrencyWindow = 5 * time.Second
	timeDiff := a.Timestamp.Sub(b.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	return timeDiff < concurrencyWindow && a.DeviceID != b.DeviceID
}

// classifyConflict determines the type of conflict based on the events
// involved.
func classifyConflict(incoming *WorkspaceEvent, existing []*WorkspaceEvent) ConflictType {
	incomingCategory := incoming.EventType.Category()

	for _, e := range existing {
		existingCategory := e.EventType.Category()

		// Different event categories on the same resource indicate a
		// semantic conflict (e.g., update vs delete).
		if incomingCategory != existingCategory {
			return ConflictSemantic
		}

		// Metadata-specific events get their own conflict type.
		if incoming.EventType == EventDevicePermChanged ||
			incoming.EventType == EventWorkspaceConfigChanged {
			return ConflictMetadata
		}

		// Version ordering issues.
		if e.Version > incoming.Version {
			return ConflictOrdering
		}
	}

	return ConflictConcurrentWrite
}
