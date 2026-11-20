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
	return c.Get(ctx, name)
}

func (l *lazyCapabilityCache) Put(ctx context.Context, cp capability.Capability) error {
	c := l.open()
	if c == nil {
		return nil
	}
	return c.Put(ctx, cp)
}

func init() {
	capability.SetCache(&lazyCapabilityCache{})
}
