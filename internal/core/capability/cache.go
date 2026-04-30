package capability

import "context"

// Cache is the seam used by LoadCapabilityCached to short-circuit the
// filesystem walk in LoadCapability. The CLI wires a concrete
// sqlstore-backed cache via SetCache during init; tests can swap in
// fakes (or leave it nil to disable caching entirely).
type Cache interface {
	// Get returns (cap, true, nil) on a fresh hit, (zero, false, nil)
	// on miss or TTL-expired entry, and (zero, false, err) on storage
	// failure.
	Get(ctx context.Context, name string) (Capability, bool, error)

	// Put stores a capability under cap.Name. Errors are advisory —
	// callers should log but not fail.
	Put(ctx context.Context, cap Capability) error
}

// pkgCache is the package-level cache set by SetCache. nil means
// caching is disabled (and LoadCapabilityCached degenerates to
// LoadCapability).
var pkgCache Cache

// SetCache wires the cache used by LoadCapabilityCached. Passing nil
// disables caching. Safe to call multiple times.
func SetCache(c Cache) { pkgCache = c }

// LoadCapabilityCached is a cache-aware version of LoadCapability. It
// consults the configured Cache first and falls back to the
// filesystem walk on miss, populating the cache with the resolved
// record before returning.
//
// Cache errors are non-fatal: a Get failure logs through the bus
// emitter (TODO once a logger seam exists) and proceeds to the
// filesystem path; a Put failure after a successful filesystem load
// is silently discarded so a cache outage cannot break command
// execution.
func LoadCapabilityCached(ctx context.Context, name string) (Capability, error) {
	if pkgCache != nil {
		if cap, hit, err := pkgCache.Get(ctx, name); err == nil && hit {
			return cap, nil
		}
	}
	cap, err := LoadCapability(name)
	if err != nil {
		return cap, err
	}
	if pkgCache != nil {
		_ = pkgCache.Put(ctx, cap)
	}
	return cap, nil
}
