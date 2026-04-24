package adapter

import (
	"context"
	"fmt"
	"os"

	"hop.top/aps/internal/events"
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
		fmt.Fprintf(os.Stderr, "warn: event publish (%s): %v\n", topic, err)
	}
}
