package events

import (
	"context"

	"hop.top/kit/go/runtime/bus"
)

const source = "aps"

// Publisher implements domain.EventPublisher backed by bus.Bus.
type Publisher struct {
	bus bus.Bus
}

// NewPublisher returns a Publisher wired to the given bus.
func NewPublisher(b bus.Bus) *Publisher {
	return &Publisher{bus: b}
}

// Publish sends an event on the bus.
func (p *Publisher) Publish(ctx context.Context, topic, src string, payload any) error {
	if src == "" {
		src = source
	}
	return p.bus.Publish(ctx, bus.NewEvent(bus.Topic(topic), src, payload))
}
