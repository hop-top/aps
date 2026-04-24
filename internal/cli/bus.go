package cli

import (
	"context"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/adapter"
	"hop.top/aps/internal/events"
	"hop.top/kit/bus"
)

// eventBus is the process-wide in-memory bus for lifecycle events.
var eventBus bus.Bus

// publisher wraps eventBus for structured event publishing.
var publisher *events.Publisher

func init() {
	eventBus = bus.New()
	publisher = events.NewPublisher(eventBus)
	adapter.SetPublisher(publisher)
}

// publishEvent is a fire-and-forget helper; logs on failure but never
// blocks the caller.
func publishEvent(topic, source string, payload any) {
	if publisher == nil {
		return
	}
	if err := publisher.Publish(context.Background(), topic, source, payload); err != nil {
		fmt.Fprintf(os.Stderr, "warn: event publish (%s): %v\n", topic, err)
	}
}
