package adapter

import (
	"context"

	"hop.top/aps/internal/events"
	"hop.top/aps/internal/logging"
)

// pub is set by SetPublisher from the parent cli package during init.
var pub *events.Publisher

// SetPublisher wires the event publisher into the adapter CLI layer.
func SetPublisher(p *events.Publisher) { pub = p }

func publishEvent(topic, source string, payload any) {
	if pub == nil {
		return
	}
	if err := pub.Publish(context.Background(), topic, source, payload); err != nil {
		logging.GetLogger().Warn("event publish failed", "topic", topic, "error", err)
	}
}
