package core

import "context"

// EventPublisher is the seam used by core operations to emit lifecycle
// events. The CLI layer wires a concrete bus-backed publisher via
// SetEventPublisher during init; tests can swap in fakes.
//
// The interface mirrors hop.top/kit/go/runtime/domain.EventPublisher so
// that core stays decoupled from kit's bus package directly — anything
// implementing Publish(ctx, topic, source, payload) error works.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload any) error
}

// pkgPublisher is the package-level publisher set by SetEventPublisher.
// nil means events are dropped silently — core operations never block on
// or fail because of event delivery.
var pkgPublisher EventPublisher

// SetEventPublisher wires the publisher used by core lifecycle emits.
// Passing nil disables event emission. Safe to call multiple times.
func SetEventPublisher(p EventPublisher) { pkgPublisher = p }

// publish is the fire-and-forget helper used by core operations. Delivery
// errors are swallowed — events are advisory and must never break a
// successful core operation.
func publish(ctx context.Context, topic, source string, payload any) {
	if pkgPublisher == nil {
		return
	}
	_ = pkgPublisher.Publish(ctx, topic, source, payload)
}
