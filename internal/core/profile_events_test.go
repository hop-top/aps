package core

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"hop.top/aps/internal/events"
)

// fakePublisher captures emitted topics + payloads for assertion.
type fakePublisher struct {
	mu     sync.Mutex
	topics []string
	got    []captured
}

type captured struct {
	topic   string
	payload any
}

func (f *fakePublisher) Publish(_ context.Context, topic, _ string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.topics = append(f.topics, topic)
	f.got = append(f.got, captured{topic: topic, payload: payload})
	return nil
}

// installFakePublisher swaps in a fresh fakePublisher for the duration of t,
// restoring the original on cleanup. Tests using this must NOT run in
// parallel with each other (they share the package-level publisher).
func installFakePublisher(t *testing.T) *fakePublisher {
	t.Helper()
	prev := pkgPublisher
	f := &fakePublisher{}
	SetEventPublisher(f)
	t.Cleanup(func() { SetEventPublisher(prev) })
	return f
}

// withTempData points APS_DATA_PATH at a per-test temp dir so file
// operations don't clobber real state.
func withTempData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dir)
	if err := os.MkdirAll(filepath.Join(dir, "profiles"), 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	return dir
}

func TestCreateProfile_EmitsProfileCreated(t *testing.T) {
	withTempData(t)
	f := installFakePublisher(t)

	err := CreateProfile("noor-test", Profile{DisplayName: "Noor", Email: "n@example.com"})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if len(f.got) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(f.got), f.got)
	}
	ev := f.got[0]
	if ev.topic != string(events.TopicProfileCreated) {
		t.Errorf("topic = %q, want %q", ev.topic, events.TopicProfileCreated)
	}
	p, ok := ev.payload.(events.ProfileCreatedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ProfileCreatedPayload", ev.payload)
	}
	if p.ProfileID != "noor-test" || p.DisplayName != "Noor" {
		t.Errorf("payload = %+v, want id=noor-test name=Noor", p)
	}
}

func TestDeleteProfile_EmitsProfileDeleted(t *testing.T) {
	withTempData(t)
	if err := CreateProfile("rami-test", Profile{DisplayName: "Rami"}); err != nil {
		t.Fatalf("setup CreateProfile: %v", err)
	}
	f := installFakePublisher(t)

	if err := DeleteProfile("rami-test", false); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	if len(f.got) != 1 || f.got[0].topic != string(events.TopicProfileDeleted) {
		t.Fatalf("got %+v, want one ProfileDeleted event", f.got)
	}
	p, ok := f.got[0].payload.(events.ProfileDeletedPayload)
	if !ok || p.ProfileID != "rami-test" {
		t.Errorf("payload = %#v, want ProfileDeletedPayload{rami-test}", f.got[0].payload)
	}
}

func TestAddCapabilityToProfile_EmitsProfileUpdated(t *testing.T) {
	withTempData(t)
	if err := CreateProfile("kai-test", Profile{DisplayName: "Kai"}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	f := installFakePublisher(t)

	if err := AddCapabilityToProfile("kai-test", "github"); err != nil {
		t.Fatalf("AddCapability: %v", err)
	}

	if len(f.got) != 1 {
		t.Fatalf("got %d events, want 1", len(f.got))
	}
	if f.got[0].topic != string(events.TopicProfileUpdated) {
		t.Errorf("topic = %q, want ProfileUpdated", f.got[0].topic)
	}
	p, ok := f.got[0].payload.(events.ProfileUpdatedPayload)
	if !ok {
		t.Fatalf("payload type = %T", f.got[0].payload)
	}
	if p.ProfileID != "kai-test" || len(p.Fields) != 1 || p.Fields[0] != "capabilities" {
		t.Errorf("payload = %+v, want id=kai-test fields=[capabilities]", p)
	}
}

func TestRemoveCapabilityFromProfile_EmitsProfileUpdated(t *testing.T) {
	withTempData(t)
	if err := CreateProfile("amir-test", Profile{DisplayName: "Amir", Capabilities: []string{"github"}}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	f := installFakePublisher(t)

	if err := RemoveCapabilityFromProfile("amir-test", "github"); err != nil {
		t.Fatalf("RemoveCapability: %v", err)
	}

	if len(f.got) != 1 || f.got[0].topic != string(events.TopicProfileUpdated) {
		t.Fatalf("got %+v, want one ProfileUpdated", f.got)
	}
}

// TestCorePublish_NilPublisher_NoPanic verifies that core operations work
// fine when no publisher is wired (default state).
func TestCorePublish_NilPublisher_NoPanic(t *testing.T) {
	withTempData(t)
	prev := pkgPublisher
	SetEventPublisher(nil)
	t.Cleanup(func() { SetEventPublisher(prev) })

	if err := CreateProfile("nil-pub-test", Profile{DisplayName: "X"}); err != nil {
		t.Fatalf("CreateProfile with nil publisher: %v", err)
	}
}
