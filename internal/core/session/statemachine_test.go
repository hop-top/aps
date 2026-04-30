package session

import (
	"errors"
	"testing"

	"hop.top/kit/go/runtime/domain"
)

func TestSessionStateMachine_AllowedTransitions(t *testing.T) {
	cases := []struct {
		from SessionStatus
		to   SessionStatus
		ok   bool
	}{
		{SessionActive, SessionInactive, true},
		{SessionActive, SessionErrored, true},
		{SessionInactive, SessionActive, true},
		{SessionInactive, SessionErrored, true},
		{SessionErrored, SessionActive, false},   // terminal
		{SessionErrored, SessionInactive, false}, // terminal
		{SessionActive, SessionActive, true},     // self-transition is no-op (idempotent)
		{SessionInactive, SessionInactive, true},
		{SessionErrored, SessionErrored, true},
		// "" (initial) can transition to any of the three statuses.
		{"", SessionActive, true},
		{"", SessionInactive, true},
		{"", SessionErrored, true},
	}

	r := freshRegistry()
	for _, c := range cases {
		err := r.checkTransition(c.from, c.to)
		gotOK := err == nil
		if gotOK != c.ok {
			t.Errorf("from=%s to=%s ok=%v err=%v, want ok=%v", c.from, c.to, gotOK, err, c.ok)
		}
		if !c.ok && err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("from=%s to=%s err=%v, want errors.Is ErrInvalidTransition", c.from, c.to, err)
		}
	}
}

func TestUpdateStatus_RejectsInvalidTransition(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s1", ProfileID: "p1", Status: SessionErrored}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Errored is terminal — re-activation must be rejected.
	err := r.UpdateStatus("s1", SessionActive)
	if err == nil {
		t.Fatalf("UpdateStatus(errored→active) should fail")
	}
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}

	// Verify in-memory status was NOT changed.
	s, _ := r.Get("s1")
	if s.Status != SessionErrored {
		t.Errorf("status = %s, want still errored", s.Status)
	}
}

func TestUpdateStatus_AllowsValidTransition(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s2", ProfileID: "p2", Status: SessionActive}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := r.UpdateStatus("s2", SessionInactive); err != nil {
		t.Fatalf("UpdateStatus(active→inactive): %v", err)
	}
	if err := r.UpdateStatus("s2", SessionActive); err != nil {
		t.Fatalf("UpdateStatus(inactive→active): %v", err)
	}
	if err := r.UpdateStatus("s2", SessionErrored); err != nil {
		t.Fatalf("UpdateStatus(active→errored): %v", err)
	}
}
