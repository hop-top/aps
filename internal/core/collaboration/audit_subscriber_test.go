package collaboration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/events"
	"hop.top/aps/internal/storage"
	"hop.top/kit/go/runtime/bus"

	"github.com/stretchr/testify/require"
)

// newAuditSubscriberFixture stands up an in-memory bus + audit log
// + the wired subscriber. Returns a publisher func and a query func.
func newAuditSubscriberFixture(t *testing.T) (publish func(topic string, src string, payload any), query func(workspaceID string) []collaboration.AuditEvent, b bus.Bus) {
	t.Helper()
	root := t.TempDir()
	store, err := storage.NewCollaborationStorage(root)
	require.NoError(t, err)
	log := collaboration.NewWorkspaceAuditLog(store)

	b = bus.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = b.Close(ctx)
	})
	unsub := collaboration.SubscribeAudit(b, log)
	t.Cleanup(unsub)

	pub := events.NewPublisher(b)

	publish = func(topic string, src string, payload any) {
		if err := pub.Publish(context.Background(), topic, src, payload); err != nil {
			t.Fatalf("publish %s: %v", topic, err)
		}
	}

	query = func(workspaceID string) []collaboration.AuditEvent {
		// Bus delivery is sync for non-async handlers, but spin briefly
		// in case future versions buffer.
		got, err := log.Query(context.Background(), workspaceID, collaboration.AuditQueryOptions{Limit: 100})
		require.NoError(t, err)
		return got
	}
	return publish, query, b
}

func TestSubscribeAudit_RecordsProfileCreated(t *testing.T) {
	publish, query, _ := newAuditSubscriberFixture(t)

	publish(string(events.TopicProfileCreated), "", events.ProfileCreatedPayload{
		ProfileID:   "noor",
		DisplayName: "Noor",
		Email:       "noor@example.com",
	})

	got := query(collaboration.GlobalAuditWorkspace)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(got), got)
	}
	ev := got[0]
	if ev.Event != string(events.TopicProfileCreated) {
		t.Errorf("event = %q, want %q", ev.Event, events.TopicProfileCreated)
	}
	if ev.Resource != "profile/noor" {
		t.Errorf("resource = %q, want profile/noor", ev.Resource)
	}
	if ev.Actor != "aps" {
		t.Errorf("actor = %q, want aps", ev.Actor)
	}
	if !strings.Contains(ev.Details, "Noor") {
		t.Errorf("details %q missing display name", ev.Details)
	}
}

func TestSubscribeAudit_RecordsSessionLifecycle(t *testing.T) {
	publish, query, _ := newAuditSubscriberFixture(t)

	publish(string(events.TopicSessionStarted), "", events.SessionStartedPayload{
		SessionID: "s1",
		ProfileID: "noor",
		Command:   "claude",
		PID:       1234,
		Tier:      "standard",
	})
	publish(string(events.TopicSessionStopped), "", events.SessionStoppedPayload{
		SessionID: "s1",
		ProfileID: "noor",
		Reason:    "unregister",
	})

	got := query(collaboration.GlobalAuditWorkspace)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(got), got)
	}
	if got[0].Event != string(events.TopicSessionStarted) || got[1].Event != string(events.TopicSessionStopped) {
		t.Errorf("event ordering wrong: %v / %v", got[0].Event, got[1].Event)
	}
	if got[1].Resource != "session/s1" || !strings.Contains(got[1].Details, "reason=unregister") {
		t.Errorf("stop event = %+v", got[1])
	}
}

func TestSubscribeAudit_RecordsAdapterEvents(t *testing.T) {
	publish, query, _ := newAuditSubscriberFixture(t)

	publish(string(events.TopicAdapterLinked), "", events.AdapterLinkedPayload{
		ProfileID:   "noor",
		AdapterType: "github",
		AdapterID:   "gh-123",
	})

	got := query(collaboration.GlobalAuditWorkspace)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Resource != "profile/noor/adapter/github" {
		t.Errorf("resource = %q", got[0].Resource)
	}
	if !strings.Contains(got[0].Details, "gh-123") {
		t.Errorf("details = %q", got[0].Details)
	}
}

func TestSubscribeAudit_IgnoresUnknownPayload(t *testing.T) {
	publish, query, _ := newAuditSubscriberFixture(t)

	// Topic prefixed aps. but unknown payload type — must not record.
	publish("aps.weird.event", "", struct{ X int }{X: 1})

	got := query(collaboration.GlobalAuditWorkspace)
	if len(got) != 0 {
		t.Errorf("got %d events, want 0", len(got))
	}
}

func TestSubscribeAudit_PreservesSourceAsActor(t *testing.T) {
	publish, query, _ := newAuditSubscriberFixture(t)

	publish(string(events.TopicProfileDeleted), "test-suite", events.ProfileDeletedPayload{
		ProfileID: "rami",
	})

	got := query(collaboration.GlobalAuditWorkspace)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Actor != "test-suite" {
		t.Errorf("actor = %q, want test-suite", got[0].Actor)
	}
}
