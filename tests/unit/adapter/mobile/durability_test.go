package mobile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hop.top/aps/internal/core/adapter/mobile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistry_DurabilityAcrossClose proves that closing a Registry
// and reopening one rooted at the same directory surfaces every
// previously-written adapter row. Mirrors the operator scenario
// where aps exits and restarts; the kv-backed sqlite file must be
// the source of truth across the process boundary.
func TestRegistry_DurabilityAcrossClose(t *testing.T) {
	dir := t.TempDir()

	r1, err := mobile.NewRegistry(dir)
	require.NoError(t, err)

	dev := &mobile.MobileAdapter{
		AdapterID:    "persist-1",
		ProfileID:    "p1",
		AdapterName:  "Persistent",
		AdapterOS:    "iOS",
		RegisteredAt: time.Now(),
		LastSeenAt:   time.Now(),
		ExpiresAt:    time.Now().Add(14 * 24 * time.Hour),
		TokenHash:    mobile.HashToken("persist-token"),
		Status:       mobile.PairingStateActive,
		Capabilities: []string{"run:stateless"},
	}
	require.NoError(t, r1.RegisterAdapter(dev))
	require.NoError(t, r1.Close())

	r2, err := mobile.NewRegistry(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r2.Close() })

	got, err := r2.GetAdapter("persist-1")
	require.NoError(t, err)
	assert.Equal(t, "persist-1", got.AdapterID)
	assert.Equal(t, "p1", got.ProfileID)
	assert.Equal(t, mobile.PairingStateActive, got.Status)
}

// TestRegistry_LegacyJSONMigration verifies the upgrade path from
// the pre-kv mobile-registry.json file: legacy rows are imported
// into kv on first NewRegistry call and the legacy file is removed.
// Without it operators would lose their paired devices after
// installing the kv-backed binary.
func TestRegistry_LegacyJSONMigration(t *testing.T) {
	dir := t.TempDir()

	// Seed the legacy file as a prior aps release would have done.
	legacy := mobile.MobileAdapterRegistryData{
		Version: "1.0",
		Adapters: []*mobile.MobileAdapter{
			{
				AdapterID:    "legacy-1",
				ProfileID:    "p1",
				AdapterName:  "Legacy",
				AdapterOS:    "Android",
				RegisteredAt: time.Now(),
				LastSeenAt:   time.Now(),
				ExpiresAt:    time.Now().Add(14 * 24 * time.Hour),
				TokenHash:    mobile.HashToken("legacy-token"),
				Status:       mobile.PairingStateActive,
				Capabilities: []string{"run:stateless"},
			},
		},
	}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	legacyPath := filepath.Join(dir, "mobile-registry.json")
	require.NoError(t, os.WriteFile(legacyPath, raw, 0o644))

	r, err := mobile.NewRegistry(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })

	got, err := r.GetAdapter("legacy-1")
	require.NoError(t, err)
	assert.Equal(t, "legacy-1", got.AdapterID)
	assert.Equal(t, "p1", got.ProfileID)

	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Errorf("legacy mobile-registry.json should be removed after migration, stat err = %v", err)
	}
}
