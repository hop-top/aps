package capability

import (
	"context"
	"errors"
	"testing"
)

// fakeCache implements Cache for unit tests.
type fakeCache struct {
	store    map[string]Capability
	getCalls int
	putCalls int
	getErr   error
	putErr   error
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: make(map[string]Capability)}
}

func (f *fakeCache) Get(_ context.Context, name string) (Capability, bool, error) {
	f.getCalls++
	if f.getErr != nil {
		return Capability{}, false, f.getErr
	}
	cap, ok := f.store[name]
	return cap, ok, nil
}

func (f *fakeCache) Put(_ context.Context, cap Capability) error {
	f.putCalls++
	if f.putErr != nil {
		return f.putErr
	}
	f.store[cap.Name] = cap
	return nil
}

func swapCache(t *testing.T, c Cache) {
	t.Helper()
	prev := pkgCache
	SetCache(c)
	t.Cleanup(func() { SetCache(prev) })
}

func TestLoadCapabilityCached_HitShortsCircuit(t *testing.T) {
	c := newFakeCache()
	c.store["preinstalled"] = Capability{Name: "preinstalled", Path: "/cached/path"}
	swapCache(t, c)

	got, err := LoadCapabilityCached(context.Background(), "preinstalled")
	if err != nil {
		t.Fatalf("LoadCapabilityCached: %v", err)
	}
	if got.Path != "/cached/path" {
		t.Errorf("path = %q, want /cached/path (cached value)", got.Path)
	}
	if c.getCalls != 1 {
		t.Errorf("getCalls = %d, want 1", c.getCalls)
	}
	if c.putCalls != 0 {
		t.Errorf("putCalls = %d, want 0 (hit shouldn't repopulate)", c.putCalls)
	}
}

func TestLoadCapabilityCached_MissFallsBackToFS(t *testing.T) {
	// No filesystem capability exists, so LoadCapability returns an
	// error — but we still expect the cache to have been consulted.
	c := newFakeCache()
	swapCache(t, c)

	_, err := LoadCapabilityCached(context.Background(), "definitely-not-installed-xyz")
	if err == nil {
		t.Fatalf("expected fs-load error for missing capability")
	}
	if c.getCalls != 1 {
		t.Errorf("getCalls = %d, want 1", c.getCalls)
	}
	// Put NOT called because the load failed.
	if c.putCalls != 0 {
		t.Errorf("putCalls = %d, want 0 on fs miss", c.putCalls)
	}
}

func TestLoadCapabilityCached_NilCacheBypasses(t *testing.T) {
	swapCache(t, nil)

	// With no cache, just verify it falls through to LoadCapability and
	// errors on a missing capability — same observable behaviour as the
	// non-cached path. No panic.
	_, err := LoadCapabilityCached(context.Background(), "missing")
	if err == nil {
		t.Errorf("expected error from underlying LoadCapability")
	}
}

func TestLoadCapabilityCached_CacheGetErrorIsNonFatal(t *testing.T) {
	c := newFakeCache()
	c.getErr = errors.New("boom")
	swapCache(t, c)

	// Cache errors should not propagate — we should fall through to FS.
	_, err := LoadCapabilityCached(context.Background(), "missing")
	// FS load also fails (capability doesn't exist) but the error
	// returned should be the FS one, not the cache "boom".
	if err == nil {
		t.Fatal("expected fs-load error")
	}
	if errors.Is(err, c.getErr) {
		t.Errorf("err = %v, want fs-load error not cache error", err)
	}
}
