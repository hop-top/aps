package mobile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Registry manages the mobile adapter registry with atomic file operations
type Registry struct {
	mu   sync.RWMutex
	path string
}

// NewRegistry creates a new mobile adapter registry at the given path
func NewRegistry(registryDir string) (*Registry, error) {
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}

	return &Registry{
		path: filepath.Join(registryDir, "mobile-registry.json"),
	}, nil
}

// RegisterAdapter adds a new mobile adapter to the registry
func (r *Registry) RegisterAdapter(device *MobileAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	// Check for duplicate device ID
	for _, d := range data.Adapters {
		if d.AdapterID == device.AdapterID {
			return fmt.Errorf("device '%s' already registered", device.AdapterID)
		}
	}

	data.Adapters = append(data.Adapters, device)
	return r.saveLocked(data)
}

// GetAdapter returns a mobile adapter by ID
func (r *Registry) GetAdapter(deviceID string) (*MobileAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadLocked()
	if err != nil {
		return nil, err
	}

	for _, d := range data.Adapters {
		if d.AdapterID == deviceID {
			return d, nil
		}
	}

	return nil, ErrMobileAdapterNotFound(deviceID)
}

// ListAdapters returns all mobile adapters, optionally filtered by profile
func (r *Registry) ListAdapters(profileID string) ([]*MobileAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadLocked()
	if err != nil {
		return nil, err
	}

	if profileID == "" {
		return data.Adapters, nil
	}

	var filtered []*MobileAdapter
	for _, d := range data.Adapters {
		if d.ProfileID == profileID {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// ListPending returns devices in pending approval state
func (r *Registry) ListPending(profileID string) ([]*MobileAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadLocked()
	if err != nil {
		return nil, err
	}

	var pending []*MobileAdapter
	for _, d := range data.Adapters {
		if d.Status == PairingStatePending {
			if profileID == "" || d.ProfileID == profileID {
				pending = append(pending, d)
			}
		}
	}
	return pending, nil
}

// UpdateAdapter updates a device in the registry
func (r *Registry) UpdateAdapter(device *MobileAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	for i, d := range data.Adapters {
		if d.AdapterID == device.AdapterID {
			data.Adapters[i] = device
			return r.saveLocked(data)
		}
	}

	return ErrMobileAdapterNotFound(device.AdapterID)
}

// RevokeAdapter marks a device as revoked
func (r *Registry) RevokeAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	for _, d := range data.Adapters {
		if d.AdapterID == deviceID {
			d.Status = PairingStateRevoked
			return r.saveLocked(data)
		}
	}

	return ErrMobileAdapterNotFound(deviceID)
}

// ApproveAdapter marks a pending device as active
func (r *Registry) ApproveAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	for _, d := range data.Adapters {
		if d.AdapterID == deviceID {
			if d.Status != PairingStatePending {
				return fmt.Errorf("device '%s' is not pending approval (status: %s)", deviceID, d.Status)
			}
			d.Status = PairingStateActive
			now := time.Now()
			d.ApprovedAt = &now
			return r.saveLocked(data)
		}
	}

	return ErrMobileAdapterNotFound(deviceID)
}

// RejectAdapter marks a pending device as rejected and removes it
func (r *Registry) RejectAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	for i, d := range data.Adapters {
		if d.AdapterID == deviceID {
			if d.Status != PairingStatePending {
				return fmt.Errorf("device '%s' is not pending approval (status: %s)", deviceID, d.Status)
			}
			d.Status = PairingStateRejected
			data.Adapters = append(data.Adapters[:i], data.Adapters[i+1:]...)
			return r.saveLocked(data)
		}
	}

	return ErrMobileAdapterNotFound(deviceID)
}

// UpdateLastSeen updates the last seen timestamp for a device
func (r *Registry) UpdateLastSeen(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return err
	}

	for _, d := range data.Adapters {
		if d.AdapterID == deviceID {
			d.LastSeenAt = time.Now()
			return r.saveLocked(data)
		}
	}

	return ErrMobileAdapterNotFound(deviceID)
}

// CleanupExpired removes expired devices from the registry
func (r *Registry) CleanupExpired() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadLocked()
	if err != nil {
		return 0, err
	}

	var active []*MobileAdapter
	removed := 0
	now := time.Now()

	for _, d := range data.Adapters {
		if now.After(d.ExpiresAt) && d.Status == PairingStateActive {
			d.Status = PairingStateExpired
			removed++
		}
		// Keep all devices (including expired) for audit trail
		active = append(active, d)
	}

	if removed > 0 {
		data.Adapters = active
		return removed, r.saveLocked(data)
	}

	return 0, nil
}

// CountActive returns the number of active (non-revoked, non-expired) devices for a profile
func (r *Registry) CountActive(profileID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadLocked()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, d := range data.Adapters {
		if d.ProfileID == profileID && d.IsActive() {
			count++
		}
	}
	return count, nil
}

// loadLocked loads the registry data (caller must hold lock)
func (r *Registry) loadLocked() (*MobileAdapterRegistryData, error) {
	data := &MobileAdapterRegistryData{
		Version: "1.0",
		Devices: make([]*MobileAdapter, 0),
	}

	fileData, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(fileData, data); err != nil {
		return nil, &MobileError{
			Message: "failed to parse mobile adapter registry",
			Code:    ErrCodeRegistryCorrupt,
			Cause:   err,
		}
	}

	return data, nil
}

// saveLocked saves the registry data using atomic write (caller must hold lock)
func (r *Registry) saveLocked(data *MobileAdapterRegistryData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpPath := r.path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write registry temp file: %w", err)
	}

	if err := os.Rename(tmpPath, r.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename registry file: %w", err)
	}

	return nil
}
