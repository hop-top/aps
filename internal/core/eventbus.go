package core

import (
	"context"

	"hop.top/kit/go/runtime/policy"
)

// NoteFromContext returns the audit note attached to ctx via
// policy.ContextAttrsKey, or "" when none is set. The CLI layer attaches
// it from the --note|-n flag (T-1291) before invoking core mutators so
// that event payloads can populate their Note field without changing
// every core function's signature.
func NoteFromContext(ctx context.Context) string {
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
