package session

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A reaper tick that lands after Close must get an error, not a nil
// store dereference.
func TestClose_CleanupAfterCloseReturnsError(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()
	if err := r.SaveToDisk(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := r.CleanupInactive(DefaultTimeout); !errors.Is(err, ErrRegistryClosed) {
		t.Fatalf("CleanupInactive after Close = %v, want ErrRegistryClosed", err)
	}
}

// Store access past the ensureStore gate (the window a concurrent
// Close opens) must not dereference the cleared store.
func TestClose_KVAccessAfterCloseDoesNotPanic(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()
	if err := r.SaveToDisk(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx := context.Background()
	if _, err := r.kvListSessions(ctx); !errors.Is(err, ErrRegistryClosed) {
		t.Errorf("kvListSessions = %v, want ErrRegistryClosed", err)
	}
	if _, err := r.kvGetSession(ctx, "x"); !errors.Is(err, ErrRegistryClosed) {
		t.Errorf("kvGetSession = %v, want ErrRegistryClosed", err)
	}
	if err := r.kvDeleteSession(ctx, "x"); !errors.Is(err, ErrRegistryClosed) {
		t.Errorf("kvDeleteSession = %v, want ErrRegistryClosed", err)
	}
	if err := r.kvPutSessionLocked(ctx, &SessionInfo{ID: "x"}); !errors.Is(err, ErrRegistryClosed) {
		t.Errorf("kvPutSessionLocked = %v, want ErrRegistryClosed", err)
	}
}

func TestClose_OperationsAfterCloseFail(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()
	if err := r.Register(&SessionInfo{ID: "a", ProfileID: "p"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := r.Register(&SessionInfo{ID: "b", ProfileID: "p"}); !errors.Is(err, ErrRegistryClosed) {
		t.Errorf("Register after Close = %v, want ErrRegistryClosed", err)
	}
	if got := r.ListByStatus(SessionActive); got != nil {
		t.Errorf("ListByStatus after Close = %v, want nil", got)
	}
}

func TestClose_Idempotent(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()
	if err := r.SaveToDisk(); err != nil {
		t.Fatalf("open: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := r.Close(); err != nil {
			t.Fatalf("Close #%d: %v", i+1, err)
		}
	}
	// Never-opened registry too.
	if err := NewForTesting().Close(); err != nil {
		t.Fatalf("Close unopened: %v", err)
	}
}

// Close must stop the reaper and wait for it, even when the caller's
// context is never cancelled; a reaper started afterwards is a no-op.
func TestClose_WaitsForReaper(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := NewForTesting()
	if err := r.SaveToDisk(); err != nil {
		t.Fatalf("open: %v", err)
	}
	startReaper(context.Background(), r, time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- r.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return (reaper not stopped?)")
	}

	waited := make(chan struct{})
	go func() { r.reaperWG.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("reaper goroutine still running after Close")
	}

	startReaper(context.Background(), r, time.Millisecond)
	r.smu.RLock()
	n := len(r.reaperStops)
	r.smu.RUnlock()
	if n != 0 {
		t.Fatalf("startReaper on closed registry registered %d reapers", n)
	}
}
