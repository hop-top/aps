package cli

import (
	"context"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/adapter"
	"hop.top/aps/internal/events"
	"hop.top/kit/bus"
)

const defaultBusAddr = "ws://localhost:8080/ws/bus"

// eventBus is the process-wide bus for lifecycle events.
// Connects to dpkms hub when available; falls back to local-only.
var eventBus bus.Bus

// publisher wraps eventBus for structured event publishing.
var publisher *events.Publisher

func init() {
	addr := os.Getenv("APS_BUS_ADDR")
	if addr == "" {
		addr = defaultBusAddr
	}

	token := os.Getenv("APS_BUS_TOKEN")
	if token == "" {
		token = os.Getenv("BUS_TOKEN")
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "warn: bus auth: BUS_TOKEN or APS_BUS_TOKEN not set; bus disabled")
		return
	}

	eventBus = bus.New(
		bus.WithNetwork(addr),
		bus.WithNetworkOption(
			bus.WithAuth(&bus.StaticTokenAuth{Token_: token}),
		),
	)

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
