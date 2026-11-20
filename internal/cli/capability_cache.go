package cli

import (
	"context"
	"fmt"
	"os"
	"sync"

	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/storage"
)

// lazyCapabilityCache is the process-wide capability.Cache. It opens the
// sqlstore-backed store on first Get or Put, so a process that never
// resolves a capability never touches the database. If the open fails
// it warns once and degrades to a no-op, so capability.LoadCapabilityCached
// falls through to the filesystem walk, as it did before the cache existed.
type lazyCapabilityCache struct {
	once  sync.Once
	cache *storage.CapabilityCache
}

func (l *lazyCapabilityCache) open() *storage.CapabilityCache {
	l.once.Do(func() {
		c, err := storage.NewCapabilityCache(storage.CapabilityCacheOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: capability cache: open failed (%v); falling back to filesystem\n", err)
			return
		}
		l.cache = c
	})
	return l.cache
}

func (l *lazyCapabilityCache) Get(ctx context.Context, name string) (capability.Capability, bool, error) {
	c := l.open()
	if c == nil {
		return capability.Capability{}, false, nil
	}
	got, hit, err := c.Get(ctx, name)
	if err != nil {
		return capability.Capability{}, false, fmt.Errorf("capability cache get: %w", err)
	}
	return got, hit, nil
}

func (l *lazyCapabilityCache) Put(ctx context.Context, cp capability.Capability) error {
	c := l.open()
	if c == nil {
		return nil
	}
	if err := c.Put(ctx, cp); err != nil {
		return fmt.Errorf("capability cache put: %w", err)
	}
	return nil
}

func init() {
	capability.SetCache(&lazyCapabilityCache{})
}
