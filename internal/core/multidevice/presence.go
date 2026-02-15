package multidevice

import (
	"fmt"
	"sync"
	"time"
)

// PresenceTransition records a state change for a device.
type PresenceTransition struct {
	DeviceID    string
	WorkspaceID string
	From        PresenceState
	To          PresenceState
	At          time.Time
}

// PresenceTracker tracks device presence across workspaces.
type PresenceTracker struct {
	mu        sync.RWMutex
	presences map[string]*DevicePresence // key: workspaceID:deviceID
	config    PresenceConfig
}

// presenceKey builds a composite key for the presences map.
func presenceKey(workspaceID, deviceID string) string {
	return workspaceID + ":" + deviceID
}

// NewPresenceTracker creates a new PresenceTracker with the given config.
func NewPresenceTracker(config PresenceConfig) *PresenceTracker {
	return &PresenceTracker{
		presences: make(map[string]*DevicePresence),
		config:    config,
	}
}

// RecordHeartbeat updates the heartbeat timestamp for a device in a workspace.
// If no presence record exists, one is created in the online state.
func (t *PresenceTracker) RecordHeartbeat(deviceID, workspaceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device ID is required")
	}
	if workspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := presenceKey(workspaceID, deviceID)
	now := time.Now()

	p, exists := t.presences[key]
	if !exists {
		t.presences[key] = &DevicePresence{
			DeviceID:      deviceID,
			WorkspaceID:   workspaceID,
			State:         PresenceOnline,
			LastHeartbeat: now,
			LastActivity:  now,
		}
		return nil
	}

	p.LastHeartbeat = now
	// If the device was away or offline, transition back to online.
	if p.State == PresenceAway || p.State == PresenceOffline {
		p.State = PresenceOnline
	}

	return nil
}

// GetPresence returns the current presence record for a device in a workspace.
func (t *PresenceTracker) GetPresence(deviceID, workspaceID string) (*DevicePresence, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := presenceKey(workspaceID, deviceID)
	p, exists := t.presences[key]
	if !exists {
		return nil, fmt.Errorf("no presence record for device %s in workspace %s", deviceID, workspaceID)
	}

	// Return a copy to prevent data races.
	cp := *p
	return &cp, nil
}

// ListPresence returns all presence records for devices in a workspace.
func (t *PresenceTracker) ListPresence(workspaceID string) ([]*DevicePresence, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*DevicePresence
	for _, p := range t.presences {
		if p.WorkspaceID == workspaceID {
			cp := *p
			result = append(result, &cp)
		}
	}

	return result, nil
}

// IsOnline returns true if the device is currently online in the workspace.
func (t *PresenceTracker) IsOnline(deviceID, workspaceID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := presenceKey(workspaceID, deviceID)
	p, exists := t.presences[key]
	if !exists {
		return false
	}

	return p.State == PresenceOnline
}

// CheckTimeouts scans all presence records and transitions devices that have
// exceeded their configured timeouts. Returns a list of transitions that
// occurred.
func (t *PresenceTracker) CheckTimeouts() []PresenceTransition {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	var transitions []PresenceTransition

	for _, p := range t.presences {
		elapsed := now.Sub(p.LastHeartbeat)

		switch p.State {
		case PresenceOnline:
			if elapsed > t.config.OfflineTimeout {
				transitions = append(transitions, PresenceTransition{
					DeviceID:    p.DeviceID,
					WorkspaceID: p.WorkspaceID,
					From:        PresenceOnline,
					To:          PresenceOffline,
					At:          now,
				})
				p.State = PresenceOffline
			} else if elapsed > t.config.AwayTimeout {
				transitions = append(transitions, PresenceTransition{
					DeviceID:    p.DeviceID,
					WorkspaceID: p.WorkspaceID,
					From:        PresenceOnline,
					To:          PresenceAway,
					At:          now,
				})
				p.State = PresenceAway
			}
		case PresenceAway:
			if elapsed > t.config.OfflineTimeout {
				transitions = append(transitions, PresenceTransition{
					DeviceID:    p.DeviceID,
					WorkspaceID: p.WorkspaceID,
					From:        PresenceAway,
					To:          PresenceOffline,
					At:          now,
				})
				p.State = PresenceOffline
			}
		}
	}

	return transitions
}

// TransitionState manually moves a device to a new presence state.
func (t *PresenceTracker) TransitionState(deviceID, workspaceID string, to PresenceState) (*PresenceTransition, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := presenceKey(workspaceID, deviceID)
	p, exists := t.presences[key]
	if !exists {
		// Create a new record if none exists.
		now := time.Now()
		t.presences[key] = &DevicePresence{
			DeviceID:      deviceID,
			WorkspaceID:   workspaceID,
			State:         to,
			LastHeartbeat: now,
			LastActivity:  now,
		}
		return &PresenceTransition{
			DeviceID:    deviceID,
			WorkspaceID: workspaceID,
			From:        PresenceOffline,
			To:          to,
			At:          now,
		}, nil
	}

	if p.State == to {
		return nil, fmt.Errorf("device %s is already in state %s", deviceID, to)
	}

	now := time.Now()
	transition := &PresenceTransition{
		DeviceID:    deviceID,
		WorkspaceID: workspaceID,
		From:        p.State,
		To:          to,
		At:          now,
	}

	p.State = to
	p.LastActivity = now

	return transition, nil
}

// RemovePresence removes the presence record for a device in a workspace.
func (t *PresenceTracker) RemovePresence(deviceID, workspaceID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := presenceKey(workspaceID, deviceID)
	delete(t.presences, key)
}
