package session

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

// freshRegistry returns an empty SessionRegistry independent of the
// GetRegistry singleton (which caches via sync.Once and would mask
// per-test data dirs set via t.Setenv).
func freshRegistry() *SessionRegistry {
	return &SessionRegistry{sessions: make(map[string]*SessionInfo)}
}

func TestRegister_PersistsToDisk(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1", ProfileID: "p1"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	if _, err := reloaded.Get("s1"); err != nil {
		t.Fatalf("session s1 not found after reload: %v", err)
	}
}

func TestUnregister_PersistsToDisk(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := r.Unregister("s1"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	if _, err := reloaded.Get("s1"); err == nil {
		t.Fatalf("session s1 should be absent after Unregister + reload")
	}
}

func TestUpdateStatus_PersistsToDisk(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1", Status: SessionActive}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := r.UpdateStatus("s1", SessionErrored); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	got, err := reloaded.Get("s1")
	if err != nil {
		t.Fatalf("Get after reload failed: %v", err)
	}
	if got.Status != SessionErrored {
		t.Fatalf("expected status %q after reload, got %q", SessionErrored, got.Status)
	}
}

func TestUpdateHeartbeat_PersistsToDisk(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	original, err := r.Get("s1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	originalSeen := original.LastSeenAt

	// Sleep a tick to guarantee a measurable time delta.
	time.Sleep(2 * time.Millisecond)

	if err := r.UpdateHeartbeat("s1"); err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	got, err := reloaded.Get("s1")
	if err != nil {
		t.Fatalf("Get after reload failed: %v", err)
	}
	if !got.LastSeenAt.After(originalSeen) {
		t.Fatalf("LastSeenAt should have advanced after UpdateHeartbeat; before=%v after=%v",
			originalSeen, got.LastSeenAt)
	}
}

func TestUpdateStatus_ErroredPersists(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1", Status: SessionActive}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := r.UpdateStatus("s1", SessionErrored); err != nil {
		t.Fatalf("UpdateStatus(SessionErrored) failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	got, err := reloaded.Get("s1")
	if err != nil {
		t.Fatalf("Get after reload failed: %v", err)
	}
	if got.Status != SessionErrored {
		t.Fatalf("expected status %q after reload, got %q", SessionErrored, got.Status)
	}

	// Errored sessions should be visible via ListByStatus so operators
	// running `aps session list` can see them.
	errored := reloaded.ListByStatus(SessionErrored)
	if len(errored) != 1 || errored[0].ID != "s1" {
		t.Fatalf("expected ListByStatus(SessionErrored) to return [s1], got %+v", errored)
	}
}

func TestUpdateSessionMetadata_PersistsAndRefreshes(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{
		ID:          "s1",
		Environment: map[string]string{"mode": "default"},
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	original, err := r.Get("s1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	originalSeen := original.LastSeenAt

	// Sleep a tick to guarantee a measurable time delta.
	time.Sleep(2 * time.Millisecond)

	if err := r.UpdateSessionMetadata("s1", map[string]string{"mode": "auto_approve", "extra": "v"}); err != nil {
		t.Fatalf("UpdateSessionMetadata failed: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	got, err := reloaded.Get("s1")
	if err != nil {
		t.Fatalf("Get after reload failed: %v", err)
	}
	if got.Environment["mode"] != "auto_approve" {
		t.Fatalf("expected mode=auto_approve, got %q", got.Environment["mode"])
	}
	if got.Environment["extra"] != "v" {
		t.Fatalf("expected extra=v, got %q", got.Environment["extra"])
	}
	if !got.LastSeenAt.After(originalSeen) {
		t.Fatalf("LastSeenAt should have advanced; before=%v after=%v", originalSeen, got.LastSeenAt)
	}
}

func TestUpdateSessionMetadata_MissingSession(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	err := r.UpdateSessionMetadata("does-not-exist", map[string]string{"k": "v"})
	if err == nil {
		t.Fatalf("expected error for missing session, got nil")
	}
	if len(r.sessions) != 0 {
		t.Fatalf("registry should remain empty, got %d entries", len(r.sessions))
	}
}

// TODO: rollback-on-save-failure test requires injecting a filesystem fault
// (read-only dir or fault-injecting fs). Left out for now.

func TestUnregister_MissingSessionIsNoOp(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := &SessionRegistry{
		sessions: make(map[string]*SessionInfo),
	}

	if err := r.Unregister("does-not-exist"); err != nil {
		t.Fatalf("Unregister of missing session should return nil, got: %v", err)
	}

	if len(r.sessions) != 0 {
		t.Fatalf("registry should remain empty, got %d entries", len(r.sessions))
	}
}

func TestUnregister_ExistingSessionRemoves(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := &SessionRegistry{
		sessions: make(map[string]*SessionInfo),
	}

	if err := r.Register(&SessionInfo{ID: "abc"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := r.Unregister("abc"); err != nil {
		t.Fatalf("Unregister of existing session failed: %v", err)
	}

	if _, exists := r.sessions["abc"]; exists {
		t.Fatalf("session abc should have been removed")
	}
}

func TestUnregister_IdempotentOnDoubleCall(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := &SessionRegistry{sessions: make(map[string]*SessionInfo)}

	if err := r.Register(&SessionInfo{ID: "abc"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := r.Unregister("abc"); err != nil {
		t.Fatalf("first Unregister failed: %v", err)
	}
	if err := r.Unregister("abc"); err != nil {
		t.Fatalf("second Unregister (idempotent) returned error: %v", err)
	}
}

// TODO: add a CleanupInactive save-failure rollback test once a
// fault-injecting filesystem is available.
func TestCleanupInactive_PersistsToDisk(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1"}); err != nil {
		t.Fatalf("Register s1 failed: %v", err)
	}
	if err := r.Register(&SessionInfo{ID: "s2"}); err != nil {
		t.Fatalf("Register s2 failed: %v", err)
	}

	// Force LastSeenAt far into the past so the timeout fires.
	past := time.Now().Add(-1 * time.Hour)
	r.sessions["s1"].LastSeenAt = past
	r.sessions["s2"].LastSeenAt = past

	expired, err := r.CleanupInactive(1 * time.Nanosecond)
	if err != nil {
		t.Fatalf("CleanupInactive failed: %v", err)
	}
	if len(expired) != 2 {
		t.Fatalf("expected 2 expired sessions, got %d (%v)", len(expired), expired)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	if _, err := reloaded.Get("s1"); err == nil {
		t.Fatalf("session s1 should be absent on disk after CleanupInactive")
	}
	if _, err := reloaded.Get("s2"); err == nil {
		t.Fatalf("session s2 should be absent on disk after CleanupInactive")
	}
}

func TestReaper_ReapsInactiveSessions(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := NewForTesting()

	// Seed a recent session.
	if err := r.Register(&SessionInfo{ID: "fresh"}); err != nil {
		t.Fatalf("Register fresh failed: %v", err)
	}
	// Seed a stale session, then backdate its LastSeenAt past
	// DefaultTimeout via direct field access (same package).
	if err := r.Register(&SessionInfo{ID: "stale"}); err != nil {
		t.Fatalf("Register stale failed: %v", err)
	}
	r.mu.Lock()
	r.sessions["stale"].LastSeenAt = time.Now().Add(-2 * DefaultTimeout)
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a short tick so the test runs quickly. The reaper checks
	// staleness against DefaultTimeout (30m), independent of tick
	// frequency.
	startReaper(ctx, r, 5*time.Millisecond)

	// Poll for the stale session's removal. Generous bound to
	// avoid flakes on a loaded CI machine.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := r.Get("stale"); err != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if _, err := r.Get("stale"); err == nil {
		t.Fatalf("expected stale session to be reaped")
	}
	if _, err := r.Get("fresh"); err != nil {
		t.Fatalf("expected fresh session to remain: %v", err)
	}
}

// TestIntegration_SessionLifecycleAcrossRestart exercises the full
// register -> heartbeat -> reap -> persist pipeline across multiple
// fresh registry instances simulating process restarts. It crosses
// T0/T1 (idempotent + write-through) and T5 (reaper persistence).
func TestIntegration_SessionLifecycleAcrossRestart(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	// Step 1: register on a fresh registry, prove it lands on disk.
	r1 := NewForTesting()
	if err := r1.Register(&SessionInfo{ID: "s1", ProfileID: "p1"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	r2 := NewForTesting()
	if err := r2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk r2 failed: %v", err)
	}
	if _, err := r2.Get("s1"); err != nil {
		t.Fatalf("expected s1 in r2 after disk reload: %v", err)
	}

	// Step 2: heartbeat on r1, then reload into r3 and assert
	// LastSeenAt matches.
	time.Sleep(2 * time.Millisecond)
	if err := r1.UpdateHeartbeat("s1"); err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}
	r1.mu.RLock()
	expectedSeen := r1.sessions["s1"].LastSeenAt
	r1.mu.RUnlock()

	r3 := NewForTesting()
	if err := r3.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk r3 failed: %v", err)
	}
	got, err := r3.Get("s1")
	if err != nil {
		t.Fatalf("Get s1 from r3 failed: %v", err)
	}
	if !got.LastSeenAt.Equal(expectedSeen) {
		t.Fatalf("LastSeenAt after reload mismatch: want %v got %v",
			expectedSeen, got.LastSeenAt)
	}

	// Step 3: backdate s1 LastSeenAt and reap.
	r1.mu.Lock()
	r1.sessions["s1"].LastSeenAt = time.Now().Add(-31 * time.Minute)
	r1.mu.Unlock()

	expired, err := r1.CleanupInactive(30 * time.Minute)
	if err != nil {
		t.Fatalf("CleanupInactive failed: %v", err)
	}
	if len(expired) != 1 || expired[0] != "s1" {
		t.Fatalf("expected expired=[s1], got %v", expired)
	}

	// Step 4: a fresh registry must NOT see s1 — proves CleanupInactive
	// auto-persists removals.
	r4 := NewForTesting()
	if err := r4.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk r4 failed: %v", err)
	}
	if _, err := r4.Get("s1"); err == nil {
		t.Fatalf("expected s1 absent from r4 after CleanupInactive persistence")
	}
}

// TestIntegration_ErroredStatePersistsAcrossRestart proves that a
// session marked SessionErrored survives a fresh registry load — the
// T3 contract that errored sessions remain in the registry for
// operator visibility.
func TestIntegration_ErroredStatePersistsAcrossRestart(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r1 := NewForTesting()
	if err := r1.Register(&SessionInfo{ID: "s1", ProfileID: "p1", Status: SessionActive}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := r1.UpdateStatus("s1", SessionErrored); err != nil {
		t.Fatalf("UpdateStatus(SessionErrored) failed: %v", err)
	}

	r2 := NewForTesting()
	if err := r2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	got, err := r2.Get("s1")
	if err != nil {
		t.Fatalf("Get after reload failed: %v", err)
	}
	if got.Status != SessionErrored {
		t.Fatalf("expected status %q after reload, got %q", SessionErrored, got.Status)
	}
	errored := r2.ListByStatus(SessionErrored)
	if len(errored) != 1 || errored[0].ID != "s1" {
		t.Fatalf("expected ListByStatus(SessionErrored)=[s1], got %+v", errored)
	}
}

// TestIntegration_ConcurrentUnregisterIsIdempotentAndPersistent proves
// T0 (idempotent Unregister) + T1 (auto-persist) handle concurrent
// callers correctly with no spurious errors and a persisted result.
func TestIntegration_ConcurrentUnregisterIsIdempotentAndPersistent(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := NewForTesting()
	if err := r.Register(&SessionInfo{ID: "s1", ProfileID: "p1"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	const callers = 10
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			if err := r.Unregister("s1"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent Unregister returned error: %v", err)
	}

	r.mu.RLock()
	n := len(r.sessions)
	r.mu.RUnlock()
	if n != 0 {
		t.Fatalf("expected empty registry, got %d entries", n)
	}

	reloaded := NewForTesting()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	if _, err := reloaded.Get("s1"); err == nil {
		t.Fatalf("expected s1 absent on disk after concurrent Unregister")
	}
}

func TestReaper_StopsOnContextCancel(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	r := NewForTesting()
	startReaper(ctx, r, 1*time.Millisecond)

	// Give the goroutine a moment to start.
	time.Sleep(10 * time.Millisecond)
	if runtime.NumGoroutine() <= baseline {
		t.Fatalf("reaper goroutine did not start: baseline=%d current=%d",
			baseline, runtime.NumGoroutine())
	}

	cancel()

	// Poll for goroutine exit with a bounded deadline.
	// runtime.NumGoroutine is inherently racy; give multiple samples.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline {
			return // clean exit
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("reaper goroutine did not exit after cancel: baseline=%d current=%d",
		baseline, runtime.NumGoroutine())
}

// TestCleanupInactive_SkipsErroredSessions verifies the T3 contract:
// errored sessions remain in the registry for operator visibility even
// when their LastSeenAt timestamp is older than the reaper timeout.
func TestCleanupInactive_SkipsErroredSessions(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()

	// Register two sessions with stale LastSeenAt, one errored and one not.
	past := time.Now().Add(-1 * time.Hour)
	if err := r.Register(&SessionInfo{ID: "stale-active", ProfileID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(&SessionInfo{ID: "stale-errored", ProfileID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateStatus("stale-errored", SessionErrored); err != nil {
		t.Fatal(err)
	}
	// Backdate both.
	r.mu.Lock()
	r.sessions["stale-active"].LastSeenAt = past
	r.sessions["stale-errored"].LastSeenAt = past
	r.mu.Unlock()

	expired, err := r.CleanupInactive(30 * time.Minute)
	if err != nil {
		t.Fatalf("CleanupInactive: %v", err)
	}
	if len(expired) != 1 || expired[0] != "stale-active" {
		t.Errorf("expected only stale-active to be reaped, got %v", expired)
	}
	if _, err := r.Get("stale-errored"); err != nil {
		t.Errorf("stale-errored should remain in registry: %v", err)
	}
}

// TestSessionType_DefaultsToStandardAndPersists ensures the new Type
// field round-trips through disk and that the zero value reads back
// as SessionTypeStandard so legacy registry files don't need migration.
func TestSessionType_DefaultsToStandardAndPersists(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "std", ProfileID: "p1"}); err != nil {
		t.Fatalf("Register std: %v", err)
	}
	if err := r.Register(&SessionInfo{ID: "voc", ProfileID: "p1", Type: SessionTypeVoice}); err != nil {
		t.Fatalf("Register voice: %v", err)
	}

	reloaded := freshRegistry()
	if err := reloaded.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk: %v", err)
	}
	std, err := reloaded.Get("std")
	if err != nil {
		t.Fatalf("Get std: %v", err)
	}
	if std.Type != SessionTypeStandard {
		t.Errorf("std.Type = %q, want SessionTypeStandard (empty)", std.Type)
	}
	voc, err := reloaded.Get("voc")
	if err != nil {
		t.Fatalf("Get voc: %v", err)
	}
	if voc.Type != SessionTypeVoice {
		t.Errorf("voc.Type = %q, want SessionTypeVoice", voc.Type)
	}
}

// TestListByType filters sessions by SessionType.
func TestListByType(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()

	if err := r.Register(&SessionInfo{ID: "a", ProfileID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(&SessionInfo{ID: "b", ProfileID: "p1", Type: SessionTypeVoice}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(&SessionInfo{ID: "c", ProfileID: "p1", Type: SessionTypeVoice}); err != nil {
		t.Fatal(err)
	}

	if got := r.ListByType(SessionTypeVoice); len(got) != 2 {
		t.Errorf("ListByType(voice) = %d, want 2", len(got))
	}
	if got := r.ListByType(SessionTypeStandard); len(got) != 1 {
		t.Errorf("ListByType(standard) = %d, want 1", len(got))
	}
}
