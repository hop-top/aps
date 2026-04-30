package session

import "context"

// EventPublisher is the seam used by session lifecycle methods to emit
// events. The CLI layer wires a concrete bus-backed publisher via
// SetEventPublisher during init; tests can swap in fakes.
//
// Mirrors hop.top/kit/go/runtime/domain.EventPublisher so this package
// stays decoupled from kit's bus implementation.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload any) error
}

// pkgPublisher is the package-level publisher set by SetEventPublisher.
// nil means events are dropped silently — registry mutations never block
// on or fail because of event delivery.
var pkgPublisher EventPublisher

// SetEventPublisher wires the publisher used by session lifecycle emits.
// Passing nil disables event emission. Safe to call multiple times.
func SetEventPublisher(p EventPublisher) { pkgPublisher = p }

// publish is the fire-and-forget helper used by session mutators.
// Delivery errors are swallowed — events are advisory and must never
// break a successful registry operation.
func publish(ctx context.Context, topic, source string, payload any) {
	if pkgPublisher == nil {
		return
	}
	_ = pkgPublisher.Publish(ctx, topic, source, payload)
}
