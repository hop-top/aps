package cli

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"hop.top/aps/internal/cli/adapter"
	"hop.top/aps/internal/cli/policygate"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/core/session"
	"hop.top/aps/internal/events"
	"hop.top/aps/internal/storage"
	"github.com/spf13/cobra"
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

// Lazy bus state. The bus is built by ensureBus on first use, never at
// package init, so a command that does not touch the bus (version,
// profile list, ...) neither connects the network adapter, opens the
// collaboration store, loads the policy engine, nor warns about a
// missing token.
var (
	busOnce     sync.Once
	busErr      error
	busWarnOnce sync.Once
)

// lazyPublisher is the events seam handed to core, session and the
// adapter CLI layer. Each Publish resolves the bus through useBus, so
// the first event a command emits builds it.
type lazyPublisher struct{}

func (lazyPublisher) Publish(ctx context.Context, topic, source string, payload any) error {
	b, err := useBus()
	if err != nil {
		return err
	}
	if b == nil {
		return nil
	}
	return publisher.Publish(ctx, topic, source, payload)
}

func init() {
	lp := lazyPublisher{}
	adapter.SetPublisher(lp)
	core.SetEventPublisher(lp)
	session.SetEventPublisher(lp)
	// The pre_persisted veto path builds the bus on its first publish.
	policygate.SetBusProvider(useBus)
}

// busAtStartupCommands are the long-running servers that emit lifecycle
// events for the requests they handle. They build the bus before
// serving so the policy engine is wired and a missing token is reported
// once at startup rather than on the first event.
var busAtStartupCommands = []string{"aps serve", "aps webhook server"}

// usesBusAtStartup reports whether cmd is one of busAtStartupCommands.
func usesBusAtStartup(cmd *cobra.Command) bool {
	return slices.Contains(busAtStartupCommands, cmd.CommandPath())
}

// busToken returns the configured bus token, or "" when the bus is
// disabled.
func busToken() string {
	if t := os.Getenv("APS_BUS_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("BUS_TOKEN")
}

// ensureBus builds the process bus once. It returns a nil bus and nil
// error when no token is configured (bus disabled). A non-nil error
// means the policy engine failed to load; the bus is then left unset so
// no veto-gated path runs against an unenforced ruleset.
func ensureBus() (bus.Bus, error) {
	busOnce.Do(initBus)
	return eventBus, busErr
}

// useBus is ensureBus for call sites that actually use the bus: it also
// reports, once per process, that the bus is disabled or unusable.
// Callers that only inspect bus state (status) must not use it.
func useBus() (bus.Bus, error) {
	b, err := ensureBus()
	busWarnOnce.Do(func() {
		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "warn: bus: %v\n", err)
		case b == nil && os.Getenv("APS_NO_BUS_WARN") == "":
			// Tests that don't exercise the bus opt out via APS_NO_BUS_WARN to
			// keep stderr clean (e.g. progress-JSONL contract tests). The
			// webhook_gap test exercises this warning explicitly and leaves
			// the flag unset.
			fmt.Fprintln(os.Stderr, "warn: bus auth: BUS_TOKEN or APS_BUS_TOKEN not set; bus disabled")
		}
	})
	return b, err
}

func initBus() {
	token := busToken()
	if token == "" {
		return
	}
	addr := os.Getenv("APS_BUS_ADDR")
	if addr == "" {
		addr = defaultBusAddr
	}

	// Build the local memory bus and a network adapter explicitly so we
	// keep handles to both. bus.New(WithNetwork(...)) hides the inner
	// memBus inside an unexported networkedBus, which makes the correct
	// shutdown ordering impossible (see drainBus below).
	local := bus.New()

	// Load the policy engine and subscribe it before anything can
	// publish, so a pre_persisted veto is never missed. Misconfig (bad
	// YAML, unknown topic, broken CEL) is returned from the gate rather
	// than running a delete against an unenforced ruleset.
	if _, err := initPolicyEngine(local); err != nil {
		busErr = err
		_ = local.Close(context.Background())
		return
	}
	eventBus = local
	netAdapter = bus.NewNetworkAdapter(
		eventBus,
		bus.WithAuth(&bus.StaticTokenAuth{Token_: token}),
	)
	// Best-effort connect; the adapter retries on its own.
	_ = netAdapter.Connect(context.Background(), addr)

	publisher = events.NewPublisher(eventBus)

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
	if err := (lazyPublisher{}).Publish(context.Background(), topic, source, payload); err != nil {
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
