package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"hop.top/aps/internal/events"
	"hop.top/kit/go/runtime/bus"
)

// TestPublisher_DeliversToSubscriber verifies the events.Publisher round-trips
// through an in-memory bus.Bus and reaches a subscriber matching the topic.
func TestPublisher_DeliversToSubscriber(t *testing.T) {
	t.Parallel()

	b := bus.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = b.Close(ctx)
	})

	pub := events.NewPublisher(b)

	var (
		got bus.Event
		mu  sync.Mutex
	)
	done := make(chan struct{})
	unsub := b.Subscribe(string(events.TopicProfileCreated), func(_ context.Context, e bus.Event) error {
		mu.Lock()
		got = e
		mu.Unlock()
		close(done)
		return nil
	})
	defer unsub()

	payload := events.ProfileCreatedPayload{ProfileID: "noor", DisplayName: "Noor"}
	if err := pub.Publish(context.Background(), string(events.TopicProfileCreated), "", payload); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscriber never received event")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Topic != events.TopicProfileCreated {
		t.Errorf("topic = %q, want %q", got.Topic, events.TopicProfileCreated)
	}
	if got.Source != "aps" {
		t.Errorf("source = %q, want %q", got.Source, "aps")
	}
	if p, ok := got.Payload.(events.ProfileCreatedPayload); !ok || p.ProfileID != "noor" {
		t.Errorf("payload = %#v, want ProfileCreatedPayload{noor}", got.Payload)
	}
}

// TestPublisher_PatternMatchesSessionTopics verifies aps.session.* subscribers
// receive both started and stopped events.
func TestPublisher_PatternMatchesSessionTopics(t *testing.T) {
	t.Parallel()

	b := bus.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = b.Close(ctx)
	})
	pub := events.NewPublisher(b)

	var (
		topics []bus.Topic
		mu     sync.Mutex
	)
	wg := make(chan struct{}, 2)
	unsub := b.Subscribe("aps.session.*", func(_ context.Context, e bus.Event) error {
		mu.Lock()
		topics = append(topics, e.Topic)
		mu.Unlock()
		wg <- struct{}{}
		return nil
	})
	defer unsub()

	if err := pub.Publish(context.Background(), string(events.TopicSessionStarted), "", events.SessionStartedPayload{SessionID: "s1"}); err != nil {
		t.Fatalf("publish started: %v", err)
	}
	if err := pub.Publish(context.Background(), string(events.TopicSessionStopped), "", events.SessionStoppedPayload{SessionID: "s1"}); err != nil {
		t.Fatalf("publish stopped: %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-wg:
		case <-time.After(time.Second):
			t.Fatalf("only got %d/2 events", i)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(topics) != 2 {
		t.Fatalf("topics = %v, want 2", topics)
	}
}
