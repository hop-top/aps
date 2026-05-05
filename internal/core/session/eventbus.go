package session

import (
	"context"

	"hop.top/kit/go/runtime/policy"
)

// noteFromContext returns the audit note attached to ctx via
// policy.ContextAttrsKey by the CLI layer (T-1291). Empty when none is
// set. Used by event publishers in this package to surface the note in
// SessionStarted / SessionStopped payloads without changing every
// registry method's signature.
func noteFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	attrs, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := attrs["note"].(string); ok {
		return v
	}
	return ""
}

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
