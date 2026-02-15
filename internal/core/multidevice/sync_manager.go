package multidevice

import (
	"fmt"
	"time"
)

// SyncResult describes the outcome of a sync operation.
type SyncResult struct {
	EventsSynced      int           `json:"events_synced"`
	EventsProcessed   int           `json:"events_processed"`
	ConflictsDetected int           `json:"conflicts_detected"`
	Conflicts         int           `json:"conflicts"`
	AutoResolved      int           `json:"auto_resolved"`
	ManualPending     int           `json:"manual_pending"`
	LatestVersion     int64         `json:"latest_version"`
	Duration          time.Duration `json:"duration"`
}

// SyncManager handles device synchronization by coordinating the event
// store, conflict detector, and resolution manager.
type SyncManager struct {
	workspaceID      string
	store            *EventStore
	conflictDetector *ConflictDetector
	resolver         *ResolutionManager
}

// NewSyncManager creates a sync manager for the given workspace. It wires
// up an EventStore, ConflictDetector, and ResolutionManager internally.
func NewSyncManager(workspaceID string) *SyncManager {
	store := NewEventStore(workspaceID)
	return &SyncManager{
		workspaceID:      workspaceID,
		store:            store,
		conflictDetector: NewConflictDetector(store),
		resolver:         NewResolutionManager(workspaceID),
	}
}

// InitiateSync synchronizes a device that has reconnected. It retrieves
// all events the device has missed (since lastVersion), checks for
// conflicts between those events and any offline events the device may
// have queued, and resolves what it can automatically.
func (m *SyncManager) InitiateSync(deviceID, workspaceID string, lastVersion int64) (*SyncResult, error) {
	start := time.Now()

	// Get the current latest version to bound our range query.
	latestVersion, err := m.store.GetLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("getting latest version: %w", err)
	}

	if lastVersion >= latestVersion {
		// Device is already up to date.
		return &SyncResult{
			LatestVersion: latestVersion,
			Duration:      time.Since(start),
		}, nil
	}

	// Retrieve all events since the device's last known version.
	missedEvents, err := m.store.GetRange(lastVersion+1, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("getting missed events: %w", err)
	}

	// Load and process any offline events the device has queued.
	queue := NewOfflineQueue(deviceID, workspaceID)
	offlineEvents, err := queue.Dequeue()
	if err != nil {
		return nil, fmt.Errorf("dequeuing offline events: %w", err)
	}

	totalEvents := len(missedEvents) + len(offlineEvents)
	result := &SyncResult{
		EventsSynced:    totalEvents,
		EventsProcessed: totalEvents,
		LatestVersion:   latestVersion,
	}

	// Check each offline event for conflicts against stored events.
	for _, event := range offlineEvents {
		conflict, err := m.conflictDetector.Detect(event)
		if err != nil {
			return nil, fmt.Errorf("detecting conflict for event %s: %w", event.ID, err)
		}

		if conflict != nil {
			result.ConflictsDetected++

			if err := m.resolver.ResolveConflict(conflict); err != nil {
				return nil, fmt.Errorf("resolving conflict %s: %w", conflict.ID, err)
			}

			switch conflict.Status {
			case ConflictAutoResolved:
				result.AutoResolved++
			case ConflictManual:
				result.ManualPending++
			}
		}

		// Store the event regardless of conflict (the resolution
		// determines which value takes precedence at read time).
		if err := m.store.Store(event); err != nil {
			return nil, fmt.Errorf("storing offline event %s: %w", event.ID, err)
		}
	}

	result.Conflicts = result.ConflictsDetected
	result.Duration = time.Since(start)

	// Update latest version after storing offline events.
	if newLatest, err := m.store.GetLatestVersion(); err == nil {
		result.LatestVersion = newLatest
	}

	return result, nil
}

// GetMissedEvents returns events a device has missed since the given
// version.
func (m *SyncManager) GetMissedEvents(workspaceID string, sinceVersion int64) ([]*WorkspaceEvent, error) {
	latestVersion, err := m.store.GetLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("getting latest version: %w", err)
	}

	if sinceVersion >= latestVersion {
		return nil, nil
	}

	events, err := m.store.GetRange(sinceVersion+1, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("getting events since version %d: %w", sinceVersion, err)
	}

	return events, nil
}

// ProcessOfflineEvents processes events from a device's offline queue.
// Each event is checked for conflicts, resolved if possible, and stored.
func (m *SyncManager) ProcessOfflineEvents(deviceID, workspaceID string, events []*WorkspaceEvent) (*SyncResult, error) {
	start := time.Now()

	result := &SyncResult{
		EventsSynced:    len(events),
		EventsProcessed: len(events),
	}

	for _, event := range events {
		// Ensure the event is attributed to the correct device and workspace.
		event.DeviceID = deviceID
		event.WorkspaceID = workspaceID

		conflict, err := m.conflictDetector.Detect(event)
		if err != nil {
			return nil, fmt.Errorf("detecting conflict for event %s: %w", event.ID, err)
		}

		if conflict != nil {
			result.ConflictsDetected++

			if err := m.resolver.ResolveConflict(conflict); err != nil {
				return nil, fmt.Errorf("resolving conflict %s: %w", conflict.ID, err)
			}

			switch conflict.Status {
			case ConflictAutoResolved:
				result.AutoResolved++
			case ConflictManual:
				result.ManualPending++
			}
		}

		if err := m.store.Store(event); err != nil {
			return nil, fmt.Errorf("storing event %s: %w", event.ID, err)
		}
	}

	result.Conflicts = result.ConflictsDetected
	result.Duration = time.Since(start)

	if newLatest, err := m.store.GetLatestVersion(); err == nil {
		result.LatestVersion = newLatest
	}

	return result, nil
}
