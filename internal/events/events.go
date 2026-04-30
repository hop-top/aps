// Package events defines the bus topic constants and event payload types for
// aps lifecycle events. The shared bus instance lives in internal/cli/bus.go
// as a process-wide singleton; events here are published by core via the
// EventPublisher interface (see internal/core/eventbus.go) so core stays
// decoupled from the bus implementation.
package events

import "hop.top/kit/go/runtime/bus"

// Bus topic constants for aps lifecycle events.
//
// Topic naming follows kit's dot-separated convention: <namespace>.<entity>.<action>.
// The "aps." prefix scopes events to this CLI; subscribers can match patterns
// like "aps.profile.*" or "aps.#" to receive all aps events.
const (
	// Profile lifecycle.
	TopicProfileCreated bus.Topic = "aps.profile.created"
	TopicProfileUpdated bus.Topic = "aps.profile.updated"
	TopicProfileDeleted bus.Topic = "aps.profile.deleted"

	// Adapter lifecycle.
	TopicAdapterLinked   bus.Topic = "aps.adapter.linked"
	TopicAdapterUnlinked bus.Topic = "aps.adapter.unlinked"

	// Session lifecycle.
	TopicSessionStarted bus.Topic = "aps.session.started"
	TopicSessionStopped bus.Topic = "aps.session.stopped"

	// Webhook + action surfaces. Currently reserved: aps does not yet
	// emit on webhook receipt or action run from a single chokepoint
	// (webhook handlers live in internal/cli/webhook, actions are
	// dispatched through internal/core/action.go but lack a unified
	// post-run hook). Constants are defined here so subscribers can
	// register handlers ahead of the emit wiring landing — see
	// docs/plans/2026-04-29-kit-reorg-adoption/domain-mapping.md.
	TopicWebhookReceived bus.Topic = "aps.webhook.received"
	TopicActionRan       bus.Topic = "aps.action.ran"
)

// ProfileCreatedPayload is published after a profile is created.
type ProfileCreatedPayload struct {
	ProfileID    string
	DisplayName  string
	Email        string
	Department   string
	Capabilities []string
}

// ProfileUpdatedPayload is published after a profile is updated.
type ProfileUpdatedPayload struct {
	ProfileID  string
	Fields     []string // changed field names
	Department string
}

// ProfileDeletedPayload is published after a profile is deleted.
type ProfileDeletedPayload struct {
	ProfileID string
}

// AdapterLinkedPayload is published after an adapter is linked to a profile.
type AdapterLinkedPayload struct {
	ProfileID   string
	AdapterType string
	AdapterID   string
}

// AdapterUnlinkedPayload is published after an adapter is unlinked from a profile.
type AdapterUnlinkedPayload struct {
	ProfileID   string
	AdapterType string
	AdapterID   string
}

// SessionStartedPayload is published after a session is registered.
type SessionStartedPayload struct {
	SessionID string
	ProfileID string
	Command   string
	PID       int
	Tier      string
}

// SessionStoppedPayload is published after a session is unregistered or
// its status is set to inactive/errored.
type SessionStoppedPayload struct {
	SessionID string
	ProfileID string
	Reason    string // "unregister", "inactive", "errored"
}

// WebhookReceivedPayload is reserved for future webhook emit wiring.
type WebhookReceivedPayload struct {
	ProfileID string
	Event     string
	Source    string
}

// ActionRanPayload is reserved for future action emit wiring.
type ActionRanPayload struct {
	ProfileID string
	ActionID  string
	ExitCode  int
}
