package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/storage"
)

// testTTL is far above sqlstore's one-second timestamp granularity.
const testTTL = time.Minute

func newCache(t *testing.T) *storage.CapabilityCache {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	c, err := storage.NewCapabilityCache(storage.CapabilityCacheOptions{
		Path: dbPath,
		TTL:  testTTL,
	})
	if err != nil {
		t.Fatalf("NewCapabilityCache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestCapabilityCache_PutGet_Roundtrips(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	cap := capability.Capability{
		Name:        "github",
		Path:        "/tmp/cap/github",
		Type:        capability.TypeManaged,
		Description: "github cli",
		InstalledAt: time.Now().UTC().Truncate(time.Second),
		Links:       map[string]string{"a": "b"},
	}
	if err := c.Put(ctx, cap); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, hit, err := c.Get(ctx, "github")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatalf("expected hit, got miss")
	}
	if got.Name != "github" || got.Path != cap.Path || got.Type != capability.TypeManaged {
		t.Errorf("got = %+v", got)
	}
	if got.Links["a"] != "b" {
		t.Errorf("links = %+v, want a=b", got.Links)
	}
}

func TestCapabilityCache_Get_MissReturnsHitFalse(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	_, hit, err := c.Get(ctx, "nope")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Errorf("expected miss, got hit")
	}
}

func TestCapabilityCache_Put_EmptyNameRejected(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	err := c.Put(ctx, capability.Capability{Name: ""})
	if err == nil {
		t.Error("expected error for empty name, got nil")
	}
}

func TestCapabilityCache_TTLExpiresEntries(t *testing.T) {
	// sqlstore stamps stored_at at second precision (truncated), so a
	// short TTL cannot be tested with real time: an entry's apparent age
	// is inflated by up to 1s and even an immediate Get can miss. Use a
	// TTL far above that granularity and backdate the stored timestamp
	// instead of sleeping.
	const ttl = testTTL
	c := newCache(t)
	ctx := context.Background()

	cap := capability.Capability{Name: "ttl-test", Path: "/x", Type: capability.TypeManaged}
	if err := c.Put(ctx, cap); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Fresh: should hit.
	if _, hit, err := c.Get(ctx, "ttl-test"); err != nil || !hit {
		t.Fatalf("immediate Get: hit=%v err=%v, want hit=true", hit, err)
	}

	// Older than most of the TTL but still inside it: should hit.
	if err := c.BackdateForTest(ctx, "ttl-test", ttl/2); err != nil {
		t.Fatal(err)
	}
	if _, hit, err := c.Get(ctx, "ttl-test"); err != nil || !hit {
		t.Fatalf("Get inside TTL: hit=%v err=%v, want hit=true", hit, err)
	}

	// Past the TTL: should miss.
	if err := c.BackdateForTest(ctx, "ttl-test", 2*ttl); err != nil {
		t.Fatal(err)
	}
	_, hit, err := c.Get(ctx, "ttl-test")
	if err != nil {
		t.Fatalf("Get after TTL: %v", err)
	}
	if hit {
		t.Errorf("expected miss after TTL expiry, got hit")
	}
}

func TestCapabilityCache_PutOverwrites(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	if err := c.Put(ctx, capability.Capability{Name: "x", Path: "/a"}); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if err := c.Put(ctx, capability.Capability{Name: "x", Path: "/b"}); err != nil {
		t.Fatalf("second Put: %v", err)
	}

	got, hit, err := c.Get(ctx, "x")
	if err != nil || !hit {
		t.Fatalf("Get: hit=%v err=%v", hit, err)
	}
	if got.Path != "/b" {
		t.Errorf("path = %q, want /b", got.Path)
	}
}

func TestCapabilityCache_DefaultPathRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	// No Path set — relies on XDG default. Just verify it opens.
	c, err := storage.NewCapabilityCache(storage.CapabilityCacheOptions{TTL: time.Minute})
	if err != nil {
		t.Fatalf("NewCapabilityCache (default path): %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Put(ctx, capability.Capability{Name: "default", Path: "/p"}); err != nil {
		t.Fatalf("Put: %v", err)
	}
}
