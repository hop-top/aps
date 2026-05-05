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
)

// ProfileCreatedPayload is published after a profile is created.
//
// Note carries the optional --note|-n value supplied by the operator at
// the CLI layer (T-1291). Empty when the flag is unset. Policy engines
// can read it as `context.note` from CEL via policy.ContextAttrsKey.
type ProfileCreatedPayload struct {
	ProfileID    string
	DisplayName  string
	Email        string
	Department   string
	Capabilities []string
	Note         string
}

// ProfileUpdatedPayload is published after a profile is updated.
type ProfileUpdatedPayload struct {
	ProfileID  string
	Fields     []string // changed field names
	Department string
	Note       string
}

// ProfileDeletedPayload is published after a profile is deleted.
type ProfileDeletedPayload struct {
	ProfileID string
	Note      string
}

// AdapterLinkedPayload is published after an adapter is linked to a profile.
type AdapterLinkedPayload struct {
	ProfileID   string
	AdapterType string
	AdapterID   string
	Note        string
}

// AdapterUnlinkedPayload is published after an adapter is unlinked from a profile.
type AdapterUnlinkedPayload struct {
	ProfileID   string
	AdapterType string
	AdapterID   string
	Note        string
}

// SessionStartedPayload is published after a session is registered.
type SessionStartedPayload struct {
	SessionID string
	ProfileID string
	Command   string
	PID       int
	Tier      string
	Note      string
}

// SessionStoppedPayload is published after a session is unregistered or
// its status is set to inactive/errored.
type SessionStoppedPayload struct {
	SessionID string
	ProfileID string
	Reason    string // "unregister", "inactive", "errored"
	Note      string
}
