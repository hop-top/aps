package adapter

import (
	"context"

	"hop.top/aps/internal/logging"
)

// Publisher is the seam the adapter CLI layer emits lifecycle events
// through. *events.Publisher satisfies it.
type Publisher interface {
	Publish(ctx context.Context, topic, source string, payload any) error
}

// pub is set by SetPublisher from the parent cli package during init.
var pub Publisher

// SetPublisher wires the event publisher into the adapter CLI layer.
func SetPublisher(p Publisher) { pub = p }

func publishEvent(topic, source string, payload any) {
	if pub == nil {
		return
	}
	if err := pub.Publish(context.Background(), topic, source, payload); err != nil {
		logging.GetLogger().Warn("event publish failed", "topic", topic, "error", err)
	}
}
