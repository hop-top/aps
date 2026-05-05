package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"hop.top/aps/internal/cli/adapter"
	"hop.top/aps/internal/cli/policygate"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/events"
	"hop.top/aps/internal/storage"
	"hop.top/kit/go/runtime/bus"
)

const defaultBusAddr = "ws://localhost:8080/ws/bus"

// drainTimeout bounds the wall-clock budget for flushing in-flight async
// network forwarders before a short-lived CLI process exits. Generous
// enough to cover one WS write + ack on a slow link, tight enough that
// a wedged hub does not stall the user's shell. Override for tests via
// APS_BUS_DRAIN_TIMEOUT.
const drainTimeout = 3 * time.Second

// eventBus is the process-wide bus for lifecycle events.
// Connects to dpkms hub when available; falls back to local-only.
//
// Composition is explicit (memBus + NetworkAdapter, not bus.WithNetwork)
// so shutdown can drain in the correct order: wait for in-flight async
// forwarders on the local bus FIRST, THEN tear down the WebSocket peer
// connections. The bundled networkedBus.Close (kit/bus/bus.go) does it
// the other way around — it closes the WS first, which interrupts the
// forwarder's wc.conn.Write before the publish makes it to the hub.
// That race is the root cause of the cross-process drop documented in
// T-0176 (and surfaced by T-0162).
var (
	eventBus   bus.Bus             // local memory bus (no network wrap)
	netAdapter *bus.NetworkAdapter // outbound forwarder + inbound handler
)

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

	// Build the local memory bus and a network adapter explicitly so we
	// keep handles to both. bus.New(WithNetwork(...)) hides the inner
	// memBus inside an unexported networkedBus, which makes the correct
	// shutdown ordering impossible (see drainBus below).
	eventBus = bus.New()
	netAdapter = bus.NewNetworkAdapter(
		eventBus,
		bus.WithAuth(&bus.StaticTokenAuth{Token_: token}),
	)
	// Best-effort connect; the adapter retries on its own.
	_ = netAdapter.Connect(context.Background(), addr)

	publisher = events.NewPublisher(eventBus)
	adapter.SetPublisher(publisher)
	core.SetEventPublisher(publisher)
	session.SetEventPublisher(publisher)

	// Expose the bus to the policy gate so CLI delete handlers in the
	// session/ + workspace/ subpackages can fire pre_persisted vetoes
	// synchronously (T-1292). Subpackages can't import internal/cli
	// (parent-imports-child cycle), so the bus is published through
	// internal/cli/policygate as a side-channel seam.
	policygate.SetBus(eventBus)

	// Wire the workspace audit log as a bus subscriber. Storage init is
	// best-effort: if it fails (no data dir yet, etc.) we skip silently
	// since the bus itself works fine without a persistent audit sink.
	if store, err := storage.NewCollaborationStorage(""); err == nil {
		collaboration.SubscribeAudit(eventBus, collaboration.NewWorkspaceAuditLog(store))
	} else {
		fmt.Fprintf(os.Stderr, "warn: audit subscriber: storage init failed: %v\n", err)
	}
}

// publishEvent is a fire-and-forget helper; logs on failure but never
// blocks the caller. The actual drain happens in drainBus, called from
// the cobra Execute defer (see root.go).
func publishEvent(topic, source string, payload any) {
	if publisher == nil {
		return
	}
	if err := publisher.Publish(context.Background(), topic, source, payload); err != nil {
		fmt.Fprintf(os.Stderr, "warn: event publish (%s): %v\n", topic, err)
	}
}

// drainBus flushes in-flight async forwarders and tears down the network
// adapter. Safe to call when the bus is disabled (no token) or multiple
// times — each branch is a no-op when nothing is wired.
//
// Order matters: eventBus.Close blocks on the memBus async wg, which is
// the wg the network adapter's outbound forwarder goroutines run under
// (the adapter calls eventBus.SubscribeAsync("#", ...) at construction).
// Closing the network adapter first would close the WebSocket and break
// any in-flight wc.conn.Write before the publish reaches the hub —
// which is exactly the race T-0176 was filed for.
//
// If no publishes occurred, the wg.Wait inside Close returns immediately
// so the cost is a single bounded ctx alloc; no need to gate on a flag.
func drainBus() {
	if eventBus == nil {
		return
	}

	timeout := drainTimeout
	if v := os.Getenv("APS_BUS_DRAIN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := eventBus.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warn: bus drain: %v\n", err)
	}
	if netAdapter != nil {
		_ = netAdapter.Close()
	}
}
