package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestKVStore_DurabilityAcrossClose proves the kv-backed registry
// survives a full Close + reopen against the same on-disk sqlite
// file. Mirrors the operator scenario: aps process exit, then a
// fresh process opens the same APS_DATA_PATH.
func TestKVStore_DurabilityAcrossClose(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)

	// First "process": write and close.
	r1 := &SessionRegistry{}
	if err := r1.Register(&SessionInfo{ID: "persist-1", ProfileID: "p1", Status: SessionActive}); err != nil {
		t.Fatalf("r1 Register: %v", err)
	}
	if err := r1.Register(&SessionInfo{ID: "persist-2", ProfileID: "p2", Status: SessionInactive}); err != nil {
		t.Fatalf("r1 Register: %v", err)
	}
	if err := r1.Close(); err != nil {
		t.Fatalf("r1 Close: %v", err)
	}

	// Second "process": fresh registry against the same dir must see
	// both rows verbatim.
	r2 := &SessionRegistry{}
	t.Cleanup(func() { _ = r2.Close() })

	got1, err := r2.Get("persist-1")
	if err != nil {
		t.Fatalf("r2 Get persist-1: %v", err)
	}
	if got1.ProfileID != "p1" || got1.Status != SessionActive {
		t.Errorf("persist-1 = %+v, want profile=p1 status=active", got1)
	}
	got2, err := r2.Get("persist-2")
	if err != nil {
		t.Fatalf("r2 Get persist-2: %v", err)
	}
	if got2.ProfileID != "p2" || got2.Status != SessionInactive {
		t.Errorf("persist-2 = %+v, want profile=p2 status=inactive", got2)
	}
}

// TestKVStore_LegacyJSONMigration verifies that a registry.json file
// produced by a pre-kv aps release is imported into kv on first open
// and the legacy file is then removed. This is the upgrade path —
// without it operators would lose every paired session after
// installing the kv-backed binary.
func TestKVStore_LegacyJSONMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)

	// Seed the legacy json file as a prior aps release would have.
	sessionsDir := filepath.Join(dir, SessionsDir)
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions dir: %v", err)
	}
	legacy := map[string]*SessionInfo{
		"legacy-1": {ID: "legacy-1", ProfileID: "p1", Status: SessionActive, LastSeenAt: time.Now()},
		"legacy-2": {ID: "legacy-2", ProfileID: "p2", Status: SessionErrored, LastSeenAt: time.Now()},
	}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	legacyPath := filepath.Join(sessionsDir, RegistryFile)
	if err := os.WriteFile(legacyPath, raw, 0o600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	// First open migrates and removes the legacy file.
	r := freshRegistry(t)
	if _, err := r.Get("legacy-1"); err != nil {
		t.Errorf("legacy-1 should have migrated: %v", err)
	}
	if _, err := r.Get("legacy-2"); err != nil {
		t.Errorf("legacy-2 should have migrated: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Errorf("legacy registry file should be removed, stat err = %v", err)
	}
}

// TestKVStore_CleanupInactiveDropsRows confirms the application-
// managed sweep actually deletes rows from kv (kit's sqlite backend
// supports native TTL but we deliberately do not use it for sessions
// because errored sessions must survive cleanup indefinitely — see
// the T3 contract in CleanupInactive).
func TestKVStore_CleanupInactiveDropsRows(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry(t)
	if err := r.Register(&SessionInfo{ID: "old", ProfileID: "p"}); err != nil {
		t.Fatalf("Register old: %v", err)
	}
	if err := r.setLastSeenForTest("old", time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatalf("backdate old: %v", err)
	}

	expired, err := r.CleanupInactive(time.Minute)
	if err != nil {
		t.Fatalf("CleanupInactive: %v", err)
	}
	if len(expired) != 1 || expired[0] != "old" {
		t.Fatalf("expired = %v, want [old]", expired)
	}

	// Direct kv probe: the row must be physically gone, not just
	// hidden behind an in-memory map.
	if err := r.ensureStore(); err != nil {
		t.Fatalf("ensureStore: %v", err)
	}
	if _, ok, err := r.store.Get(context.Background(), sessionKey("old")); err != nil {
		t.Fatalf("kv get: %v", err)
	} else if ok {
		t.Fatalf("session row 'old' still present in kv after CleanupInactive")
	}
}
