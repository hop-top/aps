package multidevice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// --- Policy types ---

// PolicyMode defines the access control mode for a workspace.
type PolicyMode string

const (
	// PolicyAllowAll allows all devices by default.
	PolicyAllowAll PolicyMode = "allow-all"
	// PolicyAllowList only allows explicitly listed devices.
	PolicyAllowList PolicyMode = "allow-list"
	// PolicyDenyList allows all except explicitly denied devices.
	PolicyDenyList PolicyMode = "deny-list"
)

// Policy defines workspace-level access policy.
type Policy struct {
	WorkspaceID  string     `json:"workspace_id"`
	Mode         PolicyMode `json:"mode"`
	AllowDevices []string   `json:"allow_devices,omitempty"`
	DenyDevices  []string   `json:"deny_devices,omitempty"`
}

// LoadPolicy reads a workspace's access policy from disk.
func LoadPolicy(workspaceID string) (*Policy, error) {
	wsDir, err := GetWorkspaceDir(workspaceID)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(wsDir, "policy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Policy{
				WorkspaceID: workspaceID,
				Mode:        PolicyAllowAll,
			}, nil
		}
		return nil, fmt.Errorf("reading policy: %w", err)
	}

	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy: %w", err)
	}

	return &p, nil
}

// SavePolicy writes a workspace's access policy to disk.
func SavePolicy(workspaceID string, policy *Policy) error {
	wsDir, err := GetWorkspaceDir(workspaceID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return fmt.Errorf("creating workspace directory: %w", err)
	}

	path := filepath.Join(wsDir, "policy.json")
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling policy: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// --- Audit types ---

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	DeviceID    string    `json:"device_id"`
	WorkspaceID string    `json:"workspace_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	Result      string    `json:"result"` // "allow" or "deny"
	Reason      string    `json:"reason,omitempty"`
}

// AuditLogger writes audit entries to a workspace's audit log.
type AuditLogger struct {
	workspaceID string
}

// NewAuditLogger creates a new AuditLogger for the given workspace.
func NewAuditLogger(workspaceID string) *AuditLogger {
	return &AuditLogger{workspaceID: workspaceID}
}

// auditLogPath returns the path to the workspace's audit log.
func (a *AuditLogger) auditLogPath() (string, error) {
	wsDir, err := GetWorkspaceDir(a.workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "audit.jsonl"), nil
}

// Log records an audit entry.
func (a *AuditLogger) Log(entry *AuditEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.WorkspaceID == "" {
		entry.WorkspaceID = a.workspaceID
	}

	path, err := a.auditLogPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// ListEntries returns audit log entries, optionally filtered.
func (a *AuditLogger) ListEntries(
	since time.Time, device, result, action string, limit int,
) ([]*AuditEntry, error) {
	path, err := a.auditLogPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []*AuditEntry
	decoder := json.NewDecoder(f)
	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			continue
		}

		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}
		if device != "" && entry.DeviceID != device {
			continue
		}
		if result != "" && entry.Result != result {
			continue
		}
		if action != "" && entry.Action != action {
			continue
		}

		entries = append(entries, &entry)
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// --- Manager ---

// Manager is the main entry point for multi-device workspace operations.
// It orchestrates linking, presence tracking, access control, event
// publishing, synchronization, and conflict resolution across workspaces.
type Manager struct {
	linker           *Linker
	presenceTracker  *PresenceTracker
	accessController *AccessController
	broker           *Broker

	stores     map[string]*EventStore
	publishers map[string]*Publisher
	syncMgrs   map[string]*SyncManager
	resolvers  map[string]*ResolutionManager
	auditors   map[string]*AuditLogger

	mu sync.RWMutex
}

// NewManager creates a new Manager with all subsystems initialized.
func NewManager() *Manager {
	linker := NewLinker()
	broker := NewBroker()
	return &Manager{
		linker:           linker,
		presenceTracker:  NewPresenceTracker(DefaultPresenceConfig()),
		accessController: NewAccessController(linker),
		broker:           broker,
		stores:           make(map[string]*EventStore),
		publishers:       make(map[string]*Publisher),
		syncMgrs:         make(map[string]*SyncManager),
		resolvers:        make(map[string]*ResolutionManager),
		auditors:         make(map[string]*AuditLogger),
	}
}

// --- Workspace-scoped component accessors ---

// GetPublisher returns (or lazily creates) the Publisher for a workspace.
func (m *Manager) GetPublisher(workspaceID string) *Publisher {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.publishers[workspaceID]; ok {
		return p
	}

	p := NewPublisherWithBroker(workspaceID, m.broker)
	m.publishers[workspaceID] = p
	return p
}

// GetEventStore returns (or lazily creates) the EventStore for a workspace.
func (m *Manager) GetEventStore(workspaceID string) *EventStore {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.stores[workspaceID]; ok {
		return s
	}

	s := NewEventStore(workspaceID)
	m.stores[workspaceID] = s
	return s
}

// GetSyncManager returns (or lazily creates) the SyncManager for a workspace.
func (m *Manager) GetSyncManager(workspaceID string) *SyncManager {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sm, ok := m.syncMgrs[workspaceID]; ok {
		return sm
	}

	sm := NewSyncManager(workspaceID)
	m.syncMgrs[workspaceID] = sm
	return sm
}

// GetResolutionManager returns (or lazily creates) the ResolutionManager
// for a workspace.
func (m *Manager) GetResolutionManager(workspaceID string) *ResolutionManager {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rm, ok := m.resolvers[workspaceID]; ok {
		return rm
	}

	rm := NewResolutionManager(workspaceID)
	m.resolvers[workspaceID] = rm
	return rm
}

// GetAuditLogger returns (or lazily creates) the AuditLogger for a workspace.
func (m *Manager) GetAuditLogger(workspaceID string) *AuditLogger {
	m.mu.Lock()
	defer m.mu.Unlock()

	if al, ok := m.auditors[workspaceID]; ok {
		return al
	}

	al := NewAuditLogger(workspaceID)
	m.auditors[workspaceID] = al
	return al
}

// --- Device operations (delegates to Linker) ---

// AttachDevice links a device to a workspace with the specified role.
func (m *Manager) AttachDevice(workspaceID, deviceID string, role DeviceRole) (*WorkspaceDeviceLink, error) {
	link, err := m.linker.AttachDevice(workspaceID, deviceID, role)
	if err != nil {
		return nil, fmt.Errorf("attaching device: %w", err)
	}

	// Record the linking event.
	pub := m.GetPublisher(workspaceID)
	event := NewEvent(workspaceID, deviceID, EventDeviceLinked, map[string]interface{}{
		"role": string(role),
	})
	_ = pub.PublishEvent(event)

	// Log the access decision.
	logger := m.GetAuditLogger(workspaceID)
	_ = logger.Log(&AuditEntry{
		DeviceID:    deviceID,
		WorkspaceID: workspaceID,
		Action:      "device.link",
		Result:      "allow",
	})

	return link, nil
}

// DetachDevice unlinks a device from a workspace.
func (m *Manager) DetachDevice(workspaceID, deviceID string) error {
	if err := m.linker.DetachDevice(workspaceID, deviceID); err != nil {
		return fmt.Errorf("detaching device: %w", err)
	}

	// Remove presence record.
	m.presenceTracker.RemovePresence(deviceID, workspaceID)

	// Record the unlinking event.
	pub := m.GetPublisher(workspaceID)
	event := NewEvent(workspaceID, deviceID, EventDeviceUnlinked, nil)
	_ = pub.PublishEvent(event)

	// Log the access decision.
	logger := m.GetAuditLogger(workspaceID)
	_ = logger.Log(&AuditEntry{
		DeviceID:    deviceID,
		WorkspaceID: workspaceID,
		Action:      "device.unlink",
		Result:      "allow",
	})

	return nil
}

// ListDeviceLinks returns all device links for a workspace.
func (m *Manager) ListDeviceLinks(workspaceID string) ([]*WorkspaceDeviceLink, error) {
	return m.linker.ListLinks(workspaceID)
}

// GetDeviceLink returns the link for a specific device in a workspace.
func (m *Manager) GetDeviceLink(workspaceID, deviceID string) (*WorkspaceDeviceLink, error) {
	return m.linker.GetLink(workspaceID, deviceID)
}

// UpdatePermissions updates the permissions for a device in a workspace.
func (m *Manager) UpdatePermissions(workspaceID, deviceID string, perms DevicePermissions) error {
	return m.linker.UpdatePermissions(workspaceID, deviceID, perms)
}

// SetRole updates the role for a device in a workspace.
func (m *Manager) SetRole(workspaceID, deviceID string, role DeviceRole) error {
	return m.linker.SetRole(workspaceID, deviceID, role)
}

// --- Presence operations ---

// RecordHeartbeat updates the heartbeat for a device in a workspace,
// using both the in-memory tracker and persisted state.
func (m *Manager) RecordHeartbeat(deviceID, workspaceID string) error {
	// Update in-memory tracker.
	if err := m.presenceTracker.RecordHeartbeat(deviceID, workspaceID); err != nil {
		return err
	}

	// Persist to disk for cross-process visibility.
	p, _ := loadPresenceFile(workspaceID, deviceID)
	if p == nil {
		p = &DevicePresence{
			DeviceID:    deviceID,
			WorkspaceID: workspaceID,
		}
	}

	p.State = PresenceOnline
	p.LastHeartbeat = time.Now()
	p.LastActivity = time.Now()

	return savePresenceFile(workspaceID, deviceID, p)
}

// GetDevicePresence returns presence info for all linked devices in a
// workspace, merging in-memory and persisted state.
func (m *Manager) GetDevicePresence(workspaceID string) ([]*DevicePresence, error) {
	links, err := m.linker.ListLinks(workspaceID)
	if err != nil {
		return nil, err
	}

	var presences []*DevicePresence
	for _, link := range links {
		p := &DevicePresence{
			DeviceID:      link.DeviceID,
			WorkspaceID:   link.WorkspaceID,
			State:         link.Status,
			LastHeartbeat: link.LastSyncAt,
			LastActivity:  link.LastSyncAt,
			SyncLag:       0,
			OfflineQueue:  0,
		}

		// Prefer in-memory tracker state if available.
		tracked, err := m.presenceTracker.GetPresence(link.DeviceID, workspaceID)
		if err == nil && tracked != nil {
			p.State = tracked.State
			p.LastHeartbeat = tracked.LastHeartbeat
			p.LastActivity = tracked.LastActivity
			p.SyncLag = tracked.SyncLag
			p.OfflineQueue = tracked.OfflineQueue
			p.ClientVersion = tracked.ClientVersion
		} else {
			// Fall back to persisted presence file.
			pf, err := loadPresenceFile(workspaceID, link.DeviceID)
			if err == nil && pf != nil {
				p.State = pf.State
				p.LastHeartbeat = pf.LastHeartbeat
				p.LastActivity = pf.LastActivity
				p.SyncLag = pf.SyncLag
				p.OfflineQueue = pf.OfflineQueue
				p.ClientVersion = pf.ClientVersion
			}
		}

		presences = append(presences, p)
	}

	return presences, nil
}

// --- Access control ---

// CheckAccess verifies that a device has permission to perform an action
// on a resource in a workspace. Uses the full AccessController evaluation
// chain (link, policy, role, schedule, rate limit).
func (m *Manager) CheckAccess(workspaceID, deviceID, action, resource string) error {
	err := m.accessController.EvaluatePermission(workspaceID, deviceID, action, resource)

	// Audit the access check.
	logger := m.GetAuditLogger(workspaceID)
	entry := &AuditEntry{
		DeviceID:    deviceID,
		WorkspaceID: workspaceID,
		Action:      action,
		Resource:    resource,
	}
	if err != nil {
		entry.Result = "deny"
		if ade, ok := err.(*AccessDeniedError); ok {
			entry.Reason = ade.Reason
		}
	} else {
		entry.Result = "allow"
	}
	_ = logger.Log(entry)

	return err
}

// --- Sync operations ---

// SyncDevice synchronizes a device with its workspace, processing any
// missed or offline events and resolving conflicts.
func (m *Manager) SyncDevice(deviceID, workspaceID string, lastVersion int64) (*SyncResult, error) {
	sm := m.GetSyncManager(workspaceID)
	result, err := sm.InitiateSync(deviceID, workspaceID, lastVersion)
	if err != nil {
		return nil, err
	}

	// Update the device link's last sync position.
	link, linkErr := m.linker.GetLink(workspaceID, deviceID)
	if linkErr == nil && link != nil {
		latest, verErr := m.GetEventStore(workspaceID).GetLatestVersion()
		if verErr == nil {
			link.LastSyncAt = time.Now()
			link.LastEventID = fmt.Sprintf("%d", latest)
			_ = SaveLink(link)
		}
	}

	return result, nil
}

// --- Conflict operations ---

// ListConflicts returns conflicts for a workspace.
func (m *Manager) ListConflicts(workspaceID string, includeResolved bool) ([]*Conflict, error) {
	rm := m.GetResolutionManager(workspaceID)
	return rm.ListConflicts(workspaceID, includeResolved)
}

// GetConflict loads a single conflict by ID.
func (m *Manager) GetConflict(workspaceID, conflictID string) (*Conflict, error) {
	rm := m.GetResolutionManager(workspaceID)
	return rm.GetConflict(conflictID)
}

// ResolveConflict resolves a conflict using the given strategy.
func (m *Manager) ResolveConflict(workspaceID, conflictID, strategy string, choice string) error {
	rm := m.GetResolutionManager(workspaceID)

	switch strategy {
	case "lww":
		conflict, err := rm.GetConflict(conflictID)
		if err != nil {
			return err
		}
		resolver := NewLWWResolver()
		if _, err := resolver.Resolve(conflict); err != nil {
			return fmt.Errorf("lww resolution failed: %w", err)
		}
		store := NewConflictStore(workspaceID)
		return store.Save(conflict)

	case "manual":
		choiceMap := map[string]interface{}{"chosen_by": choice}
		return rm.ResolveManually(conflictID, strategy, choiceMap)

	default:
		return fmt.Errorf("unknown resolution strategy: %s", strategy)
	}
}

// --- Event operations ---

// GetEvents returns events for a workspace filtered by criteria.
func (m *Manager) GetEvents(
	workspaceID string, since time.Time, limit int,
) ([]*WorkspaceEvent, error) {
	store := m.GetEventStore(workspaceID)
	events, err := store.GetSince(since)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	return events, nil
}

// --- Presence file helpers (for cross-process persistence) ---

func presenceFilePath(workspaceID, deviceID string) (string, error) {
	wsDir, err := GetWorkspaceDir(workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(wsDir, "presence", deviceID+".json"), nil
}

func loadPresenceFile(workspaceID, deviceID string) (*DevicePresence, error) {
	path, err := presenceFilePath(workspaceID, deviceID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p DevicePresence
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func savePresenceFile(workspaceID, deviceID string, p *DevicePresence) error {
	path, err := presenceFilePath(workspaceID, deviceID)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
