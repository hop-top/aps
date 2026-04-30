// Package storage hosts persistence adapters used by aps. CapabilityCache
// wraps a hop.top/kit/go/storage/sqlstore.Store keyed by capability name
// to avoid repeatedly walking the filesystem for capability metadata.
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hop.top/aps/internal/core/capability"
	"hop.top/kit/go/storage/sqlstore"
)

// DefaultCapabilityCacheTTL is the default TTL for cached capability
// entries. Short enough that on-disk truth wins eventually, long
// enough that repeat command invocations within a session avoid the
// filesystem walk.
const DefaultCapabilityCacheTTL = 5 * time.Minute

// CapabilityCache is a key-value cache of capability.Capability records
// backed by sqlstore. It is concurrency-safe (sqlstore is) and may be
// shared across goroutines.
//
// Keys are capability names (the same identity used by
// capability.LoadCapability). The cache is a write-through advisory
// layer — Get returning miss is never an error.
type CapabilityCache struct {
	store *sqlstore.Store
}

// CapabilityCacheOptions configures NewCapabilityCache.
type CapabilityCacheOptions struct {
	// Path is the SQLite database path. If empty, defaultCachePath is used.
	Path string
	// TTL caps the age of cached entries. Zero defaults to
	// DefaultCapabilityCacheTTL. A negative TTL disables expiry.
	TTL time.Duration
}

// NewCapabilityCache opens (or creates) a sqlstore-backed cache at the
// configured path. The returned cache must be Closed when finished.
func NewCapabilityCache(opts CapabilityCacheOptions) (*CapabilityCache, error) {
	path := opts.Path
	if path == "" {
		p, err := defaultCachePath()
		if err != nil {
			return nil, err
		}
		path = p
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = DefaultCapabilityCacheTTL
	}

	store, err := sqlstore.Open(path, sqlstore.Options{TTL: ttl})
	if err != nil {
		return nil, fmt.Errorf("open capability cache: %w", err)
	}
	return &CapabilityCache{store: store}, nil
}

// defaultCachePath returns the default path for the cache db. It uses
// XDG_CACHE_HOME when set, falling back to ~/.cache, and stores the db
// at <cache>/aps/cache.db.
func defaultCachePath() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate cache home: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "aps", "cache.db"), nil
}

// Put stores a capability under its Name. Returns an error if the
// capability name is empty (since Name is the cache key).
func (c *CapabilityCache) Put(ctx context.Context, cap capability.Capability) error {
	if cap.Name == "" {
		return errors.New("capability cache: cannot put record with empty Name")
	}
	return c.store.Put(ctx, cap.Name, cap)
}

// Get retrieves a cached capability by name. Returns (cap, true, nil)
// on a fresh hit, (zero, false, nil) on miss or TTL-expired entry, and
// (zero, false, err) on storage failure.
func (c *CapabilityCache) Get(ctx context.Context, name string) (capability.Capability, bool, error) {
	var cap capability.Capability
	hit, err := c.store.Get(ctx, name, &cap)
	if err != nil {
		return capability.Capability{}, false, err
	}
	return cap, hit, nil
}

// Close releases the underlying sqlstore resources.
func (c *CapabilityCache) Close() error { return c.store.Close() }
