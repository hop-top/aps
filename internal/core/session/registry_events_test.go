package session

import (
	"context"
	"sync"
	"testing"

	"hop.top/aps/internal/events"
)

// fakePublisher captures emitted topics + payloads for assertion.
type fakePublisher struct {
	mu     sync.Mutex
	events []captured
}

type captured struct {
	topic   string
	payload any
}

func (f *fakePublisher) Publish(_ context.Context, topic, _ string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, captured{topic: topic, payload: payload})
	return nil
}

// installFakePublisher installs a fresh fakePublisher for the duration of
// t, restoring the prior publisher on cleanup. Tests must NOT run in
// parallel with each other (they share the package-level publisher).
func installFakePublisher(t *testing.T) *fakePublisher {
	t.Helper()
	prev := pkgPublisher
	f := &fakePublisher{}
	SetEventPublisher(f)
	t.Cleanup(func() { SetEventPublisher(prev) })
	return f
}

func TestRegister_EmitsSessionStarted(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	f := installFakePublisher(t)

	r := freshRegistry()
	if err := r.Register(&SessionInfo{
		ID:        "s-start",
		ProfileID: "noor",
		Command:   "claude",
		PID:       4242,
		Tier:      TierStandard,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if len(f.events) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(f.events), f.events)
	}
	ev := f.events[0]
	if ev.topic != string(events.TopicSessionStarted) {
		t.Errorf("topic = %q, want %q", ev.topic, events.TopicSessionStarted)
	}
	p, ok := ev.payload.(events.SessionStartedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want SessionStartedPayload", ev.payload)
	}
	if p.SessionID != "s-start" || p.ProfileID != "noor" || p.Command != "claude" || p.PID != 4242 || p.Tier != "standard" {
		t.Errorf("payload = %+v", p)
	}
}

func TestUnregister_EmitsSessionStopped(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s-stop", ProfileID: "rami"}); err != nil {
		t.Fatalf("setup Register: %v", err)
	}
	f := installFakePublisher(t)

	if err := r.Unregister("s-stop"); err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	if len(f.events) != 1 || f.events[0].topic != string(events.TopicSessionStopped) {
		t.Fatalf("got %+v, want one SessionStopped", f.events)
	}
	p, ok := f.events[0].payload.(events.SessionStoppedPayload)
	if !ok {
		t.Fatalf("payload type = %T", f.events[0].payload)
	}
	if p.SessionID != "s-stop" || p.ProfileID != "rami" || p.Reason != "unregister" {
		t.Errorf("payload = %+v, want id=s-stop profile=rami reason=unregister", p)
	}
}

func TestUnregister_MissingSession_NoEvent(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	f := installFakePublisher(t)

	r := freshRegistry()
	if err := r.Unregister("nope"); err != nil {
		t.Fatalf("Unregister missing should be no-op: %v", err)
	}
	if len(f.events) != 0 {
		t.Errorf("got %d events for missing-session unregister, want 0", len(f.events))
	}
}

func TestUpdateStatus_Inactive_EmitsSessionStopped(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s-inact", ProfileID: "kai", Status: SessionActive}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	f := installFakePublisher(t)

	if err := r.UpdateStatus("s-inact", SessionInactive); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if len(f.events) != 1 || f.events[0].topic != string(events.TopicSessionStopped) {
		t.Fatalf("got %+v, want one SessionStopped", f.events)
	}
	p := f.events[0].payload.(events.SessionStoppedPayload)
	if p.Reason != "inactive" {
		t.Errorf("reason = %q, want inactive", p.Reason)
	}
}

func TestUpdateStatus_Errored_EmitsSessionStopped(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s-err", ProfileID: "amir", Status: SessionActive}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	f := installFakePublisher(t)

	if err := r.UpdateStatus("s-err", SessionErrored); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if len(f.events) != 1 {
		t.Fatalf("got %d events, want 1", len(f.events))
	}
	p := f.events[0].payload.(events.SessionStoppedPayload)
	if p.Reason != "errored" {
		t.Errorf("reason = %q, want errored", p.Reason)
	}
}

func TestUpdateStatus_StillActive_NoEvent(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "s-stay", ProfileID: "x", Status: SessionInactive}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	f := installFakePublisher(t)

	// Active is a "back to live" transition — not a stop.
	if err := r.UpdateStatus("s-stay", SessionActive); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if len(f.events) != 0 {
		t.Errorf("got %d events for active transition, want 0", len(f.events))
	}
}

func TestCleanupInactive_EmitsSessionStoppedPerExpired(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "old1", ProfileID: "p1", Status: SessionActive}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := r.Register(&SessionInfo{ID: "old2", ProfileID: "p2", Status: SessionActive}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Force LastSeenAt far in the past on both.
	r.mu.Lock()
	for _, s := range r.sessions {
		s.LastSeenAt = s.LastSeenAt.Add(-1000 * 1e9 * 60) // 1000 minutes back
	}
	r.mu.Unlock()

	f := installFakePublisher(t)

	expired, err := r.CleanupInactive(0)
	if err != nil {
		t.Fatalf("CleanupInactive: %v", err)
	}
	if len(expired) != 2 {
		t.Fatalf("expired = %v, want 2", expired)
	}
	if len(f.events) != 2 {
		t.Fatalf("got %d events, want 2", len(f.events))
	}
	for _, ev := range f.events {
		if ev.topic != string(events.TopicSessionStopped) {
			t.Errorf("topic = %q, want SessionStopped", ev.topic)
		}
		p := ev.payload.(events.SessionStoppedPayload)
		if p.Reason != "expired" {
			t.Errorf("reason = %q, want expired", p.Reason)
		}
	}
}

func TestRegister_NilPublisher_NoPanic(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	prev := pkgPublisher
	SetEventPublisher(nil)
	t.Cleanup(func() { SetEventPublisher(prev) })

	r := freshRegistry()
	if err := r.Register(&SessionInfo{ID: "nilpub"}); err != nil {
		t.Fatalf("Register with nil publisher: %v", err)
	}
}
