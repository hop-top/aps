package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withFakeActiveSessions swaps the package-level activeSessionsForProfile
// hook for the duration of the test and restores it via t.Cleanup.
//
// WARNING: this mutates package state and is NOT parallel-test safe.
// Do not call t.Parallel() in any test that uses this helper.
func withFakeActiveSessions(t *testing.T, fake func(string) ([]string, error)) {
	t.Helper()
	prev := activeSessionsForProfile
	activeSessionsForProfile = fake
	t.Cleanup(func() { activeSessionsForProfile = prev })
}

// newTestProfile creates a minimal profile in an isolated APS_DATA_PATH
// and returns its id.
func newTestProfile(t *testing.T, id string) string {
	t.Helper()
	if err := CreateProfile(id, Profile{DisplayName: id}); err != nil {
		t.Fatalf("CreateProfile(%q): %v", id, err)
	}
	dir, err := GetProfileDir(id)
	if err != nil {
		t.Fatalf("GetProfileDir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected profile dir %q to exist: %v", dir, err)
	}
	return id
}

func TestDeleteProfile_NotFound(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	withFakeActiveSessions(t, func(string) ([]string, error) { return nil, nil })

	err := DeleteProfile("nope-does-not-exist", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nope-does-not-exist") {
		t.Errorf("error should mention id, got: %v", err)
	}
}

func TestDeleteProfile_Success(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	withFakeActiveSessions(t, func(string) ([]string, error) { return nil, nil })

	id := newTestProfile(t, "to-delete")

	if err := DeleteProfile(id, false); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	dir, _ := GetProfileDir(id)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected profile dir to be removed, stat err: %v", err)
	}
}

func TestDeleteProfile_BlockedByActiveSession(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	withFakeActiveSessions(t, func(pid string) ([]string, error) {
		if pid == "blocked" {
			return []string{"sess-abc", "sess-def"}, nil
		}
		return nil, nil
	})

	id := newTestProfile(t, "blocked")

	err := DeleteProfile(id, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrProfileHasActiveSessions) {
		t.Errorf("expected errors.Is(err, ErrProfileHasActiveSessions), got: %v", err)
	}
	if !strings.Contains(err.Error(), "sess-abc") || !strings.Contains(err.Error(), "sess-def") {
		t.Errorf("error should list blocking session IDs, got: %v", err)
	}

	dir, _ := GetProfileDir(id)
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected profile dir to still exist, stat err: %v", err)
	}
}

func TestDeleteProfile_ForceIgnoresActiveSessions(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	withFakeActiveSessions(t, func(string) ([]string, error) {
		return []string{"sess-abc"}, nil
	})

	id := newTestProfile(t, "force-me")

	if err := DeleteProfile(id, true); err != nil {
		t.Fatalf("DeleteProfile force: %v", err)
	}

	dir, _ := GetProfileDir(id)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected profile dir to be removed, stat err: %v", err)
	}
}

// TestIntegration_DeleteProfileBlockedByRealRegistry exercises the full
// T6 + T7 path against a real on-disk session registry file (instead of
// the activeSessionsForProfile test seam) so that the default
// implementation is covered end to end.
func TestIntegration_DeleteProfileBlockedByRealRegistry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	// Make sure no previous test left activeSessionsForProfile swapped.
	// withFakeActiveSessions resets via t.Cleanup but explicit restore
	// here keeps the test self-contained.
	prev := activeSessionsForProfile
	activeSessionsForProfile = defaultActiveSessionsForProfile
	t.Cleanup(func() { activeSessionsForProfile = prev })

	id := newTestProfile(t, "p1")

	// Hand-write a registry file matching the on-disk JSON format
	// produced by SessionRegistry.saveToDiskLocked.
	sessionsDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	registryJSON := `{
  "id-1": {
    "id": "id-1",
    "profile_id": "p1",
    "command": "",
    "pid": 0,
    "status": "active",
    "created_at": "2026-04-07T00:00:00Z",
    "last_seen_at": "2026-04-07T00:00:00Z"
  }
}`
	if err := os.WriteFile(filepath.Join(sessionsDir, "registry.json"), []byte(registryJSON), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	err := DeleteProfile(id, false)
	if err == nil {
		t.Fatal("expected error from DeleteProfile, got nil")
	}
	if !errors.Is(err, ErrProfileHasActiveSessions) {
		t.Fatalf("expected errors.Is(err, ErrProfileHasActiveSessions), got: %v", err)
	}
	if !strings.Contains(err.Error(), "id-1") {
		t.Errorf("error should mention session id-1, got: %v", err)
	}

	dir, _ := GetProfileDir(id)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected profile dir to still exist, stat err: %v", err)
	}

	// Force bypass — should succeed even though registry still flags
	// an active session.
	if err := DeleteProfile(id, true); err != nil {
		t.Fatalf("DeleteProfile force: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected profile dir gone after force delete, stat err: %v", err)
	}
}

// TestDeleteProfile_RemovesEmbeddedWorkspaceLink verifies that when a
// profile has a workspace link stored in its YAML, deleting the profile
// removes the link along with the profile directory. This test does NOT
// exercise any external workspace registry (none exists today — links
// are purely embedded in the profile). If an external registry is
// introduced later, that cleanup must be tested separately.
func TestDeleteProfile_RemovesEmbeddedWorkspaceLink(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	withFakeActiveSessions(t, func(string) ([]string, error) { return nil, nil })

	id := newTestProfile(t, "with-ws")

	// Attach a workspace link directly via the profile API (avoids
	// pulling in the internal/workspace package and creating a test
	// import cycle).
	p, err := LoadProfile(id)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	p.Workspace = &WorkspaceLink{Name: "alpha", Scope: "session"}
	if err := SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	// Sanity check: the link is on disk.
	reloaded, err := LoadProfile(id)
	if err != nil {
		t.Fatalf("LoadProfile (sanity): %v", err)
	}
	if reloaded.Workspace == nil || reloaded.Workspace.Name != "alpha" {
		t.Fatalf("workspace link not persisted: %+v", reloaded.Workspace)
	}

	if err := DeleteProfile(id, false); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	dir, _ := GetProfileDir(id)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected profile dir (and its workspace link) to be removed, stat err: %v", err)
	}
}
