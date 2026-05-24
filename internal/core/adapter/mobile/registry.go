package mobile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"hop.top/kit/go/storage/kv"
	"hop.top/kit/go/storage/kv/sqlite"
)

// adapterKeyPrefix scopes mobile adapter rows under a fixed namespace
// inside the shared kv table. Distinct from any future per-package
// namespaces so prefix scans don't cross domains.
const adapterKeyPrefix = "adapter/"

// adapterDBFile is the on-disk sqlite filename inside the caller-
// supplied registryDir. Sits alongside the legacy json file during
// the migration window; the json file is removed once its contents
// land in kv.
const adapterDBFile = "mobile.db"

// legacyRegistryFile is the pre-kv mobile adapter file produced by
// earlier aps releases. Kept exported here as a const so the
// migration helper can locate it deterministically.
const legacyRegistryFile = "mobile-registry.json"

func adapterKey(id string) string { return adapterKeyPrefix + id }

func idFromAdapterKey(key string) string {
	return strings.TrimPrefix(key, adapterKeyPrefix)
}

// Registry manages the mobile adapter registry backed by a sqlite
// kit/storage/kv store at <registryDir>/mobile.db. The mutex guards
// compound operations (register-if-absent, the various read-modify-
// write update flows, CleanupExpired); single-key reads delegate to
// the kv backend's own locking.
type Registry struct {
	mu    sync.Mutex
	store kv.Store
}

// NewRegistry opens (or creates) a kv-backed mobile adapter registry
// at the supplied directory. Any legacy mobile-registry.json file in
// the same directory is migrated into kv on first open and removed.
func NewRegistry(registryDir string) (*Registry, error) {
	if err := os.MkdirAll(registryDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}
	store, err := sqlite.New(filepath.Join(registryDir, adapterDBFile))
	if err != nil {
		return nil, fmt.Errorf("open mobile adapter kv: %w", err)
	}
	r := &Registry{store: store}
	if err := r.migrateLegacyJSON(context.Background(), registryDir); err != nil {
		_ = store.Close()
		return nil, err
	}
	return r, nil
}

// Close releases the underlying kv store. Idempotent.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == nil {
		return nil
	}
	err := r.store.Close()
	r.store = nil
	if err != nil {
		return fmt.Errorf("kv close: %w", err)
	}
	return nil
}

// migrateLegacyJSON imports any pre-existing mobile-registry.json into
// the kv store, then removes the legacy file. Rows already present in
// kv (e.g. from a retried partial migration) are not overwritten.
func (r *Registry) migrateLegacyJSON(ctx context.Context, dir string) error {
	legacyPath := filepath.Join(dir, legacyRegistryFile)
	// #nosec G304 -- dir is supplied by the operator-provided data dir, not user input
	raw, err := os.ReadFile(legacyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read legacy mobile registry: %w", err)
	}
	if len(raw) == 0 {
		_ = os.Remove(legacyPath)
		return nil
	}
	var legacy MobileAdapterRegistryData
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return &MobileError{
			Message: "failed to parse mobile adapter registry",
			Code:    ErrCodeRegistryCorrupt,
			Cause:   err,
		}
	}
	for _, d := range legacy.Adapters {
		if d == nil || d.AdapterID == "" {
			continue
		}
		if _, ok, err := r.store.Get(ctx, adapterKey(d.AdapterID)); err != nil {
			return fmt.Errorf("kv get during legacy migration: %w", err)
		} else if ok {
			continue
		}
		if err := r.putAdapter(ctx, d); err != nil {
			return err
		}
	}
	if err := os.Remove(legacyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy mobile registry: %w", err)
	}
	return nil
}

// putAdapter writes a single adapter to the kv store. Does not acquire
// r.mu — callers that already hold the mutex for a compound op pass
// through.
func (r *Registry) putAdapter(ctx context.Context, d *MobileAdapter) error {
	buf, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal adapter: %w", err)
	}
	if err := r.store.Put(ctx, adapterKey(d.AdapterID), buf); err != nil {
		return fmt.Errorf("kv put: %w", err)
	}
	return nil
}

// getAdapter returns the adapter row keyed by id, or nil if absent.
func (r *Registry) getAdapter(ctx context.Context, id string) (*MobileAdapter, error) {
	raw, ok, err := r.store.Get(ctx, adapterKey(id))
	if err != nil {
		return nil, fmt.Errorf("kv get: %w", err)
	}
	if !ok {
		return nil, nil
	}
	var d MobileAdapter
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, &MobileError{
			Message: "failed to parse mobile adapter registry",
			Code:    ErrCodeRegistryCorrupt,
			Cause:   err,
		}
	}
	return &d, nil
}

// listAdapters returns every adapter row in the store. Used by all
// the List*/Count* methods to drive in-memory filters; the dataset is
// expected to remain small (operator-scale, not user-scale).
func (r *Registry) listAdapters(ctx context.Context) ([]*MobileAdapter, error) {
	keys, err := r.store.List(ctx, adapterKeyPrefix)
	if err != nil {
		return nil, fmt.Errorf("kv list: %w", err)
	}
	out := make([]*MobileAdapter, 0, len(keys))
	for _, k := range keys {
		raw, ok, err := r.store.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("kv get during list: %w", err)
		}
		if !ok {
			continue
		}
		var d MobileAdapter
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, &MobileError{
				Message: fmt.Sprintf("failed to parse mobile adapter %q", idFromAdapterKey(k)),
				Code:    ErrCodeRegistryCorrupt,
				Cause:   err,
			}
		}
		out = append(out, &d)
	}
	return out, nil
}

// RegisterAdapter adds a new mobile adapter to the registry.
func (r *Registry) RegisterAdapter(device *MobileAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	existing, err := r.getAdapter(ctx, device.AdapterID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("adapter '%s' already registered", device.AdapterID)
	}
	return r.putAdapter(ctx, device)
}

// GetAdapter returns a mobile adapter by ID.
func (r *Registry) GetAdapter(deviceID string) (*MobileAdapter, error) {
	d, err := r.getAdapter(context.Background(), deviceID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrMobileAdapterNotFound(deviceID)
	}
	return d, nil
}

// ListAdapters returns all mobile adapters, optionally filtered by profile.
func (r *Registry) ListAdapters(profileID string) ([]*MobileAdapter, error) {
	all, err := r.listAdapters(context.Background())
	if err != nil {
		return nil, err
	}
	if profileID == "" {
		return all, nil
	}
	var filtered []*MobileAdapter
	for _, d := range all {
		if d.ProfileID == profileID {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// ListPending returns devices in pending approval state.
func (r *Registry) ListPending(profileID string) ([]*MobileAdapter, error) {
	all, err := r.listAdapters(context.Background())
	if err != nil {
		return nil, err
	}
	var pending []*MobileAdapter
	for _, d := range all {
		if d.Status != PairingStatePending {
			continue
		}
		if profileID == "" || d.ProfileID == profileID {
			pending = append(pending, d)
		}
	}
	return pending, nil
}

// UpdateAdapter updates a device in the registry.
func (r *Registry) UpdateAdapter(device *MobileAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	existing, err := r.getAdapter(ctx, device.AdapterID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrMobileAdapterNotFound(device.AdapterID)
	}
	return r.putAdapter(ctx, device)
}

// RevokeAdapter marks a device as revoked.
func (r *Registry) RevokeAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	d, err := r.getAdapter(ctx, deviceID)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrMobileAdapterNotFound(deviceID)
	}
	d.Status = PairingStateRevoked
	return r.putAdapter(ctx, d)
}

// ApproveAdapter marks a pending device as active.
func (r *Registry) ApproveAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	d, err := r.getAdapter(ctx, deviceID)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrMobileAdapterNotFound(deviceID)
	}
	if d.Status != PairingStatePending {
		return fmt.Errorf("adapter '%s' is not pending approval (status: %s)", deviceID, d.Status)
	}
	d.Status = PairingStateActive
	now := time.Now()
	d.ApprovedAt = &now
	return r.putAdapter(ctx, d)
}

// RejectAdapter marks a pending device as rejected and removes it.
func (r *Registry) RejectAdapter(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	d, err := r.getAdapter(ctx, deviceID)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrMobileAdapterNotFound(deviceID)
	}
	if d.Status != PairingStatePending {
		return fmt.Errorf("adapter '%s' is not pending approval (status: %s)", deviceID, d.Status)
	}
	if err := r.store.Delete(ctx, adapterKey(deviceID)); err != nil {
		return fmt.Errorf("kv delete: %w", err)
	}
	return nil
}

// UpdateLastSeen updates the last seen timestamp for a device.
func (r *Registry) UpdateLastSeen(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	d, err := r.getAdapter(ctx, deviceID)
	if err != nil {
		return err
	}
	if d == nil {
		return ErrMobileAdapterNotFound(deviceID)
	}
	d.LastSeenAt = time.Now()
	return r.putAdapter(ctx, d)
}

// CleanupExpired transitions active adapters past their ExpiresAt to
// PairingStateExpired. Expired adapters are KEPT in the registry for
// audit purposes — the rename-only sweep matches the prior in-memory
// behaviour, which is why kit's native ttl is not used here.
func (r *Registry) CleanupExpired() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()

	all, err := r.listAdapters(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	removed := 0
	for _, d := range all {
		if now.After(d.ExpiresAt) && d.Status == PairingStateActive {
			d.Status = PairingStateExpired
			if err := r.putAdapter(ctx, d); err != nil {
				return removed, err
			}
			removed++
		}
	}
	return removed, nil
}

// CountActive returns the number of active (non-revoked, non-expired)
// devices for a profile.
func (r *Registry) CountActive(profileID string) (int, error) {
	all, err := r.listAdapters(context.Background())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, d := range all {
		if d.ProfileID == profileID && d.IsActive() {
			count++
		}
	}
	return count, nil
}
