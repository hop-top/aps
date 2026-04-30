package events

import "hop.top/kit/go/runtime/bus"

// Bus topic constants for aps lifecycle events.
const (
	TopicProfileCreated  bus.Topic = "aps.profile.created"
	TopicProfileUpdated  bus.Topic = "aps.profile.updated"
	TopicProfileDeleted  bus.Topic = "aps.profile.deleted"
	TopicAdapterLinked   bus.Topic = "aps.adapter.linked"
	TopicAdapterUnlinked bus.Topic = "aps.adapter.unlinked"
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
