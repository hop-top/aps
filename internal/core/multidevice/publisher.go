package multidevice

import (
	"fmt"
)

// Publisher publishes events to the event store and notifies subscribers
// through the broker.
type Publisher struct {
	store       *EventStore
	broker      *Broker
	workspaceID string
}

// NewPublisher creates a new Publisher for the given workspace.
// It initializes its own EventStore and Broker.
func NewPublisher(workspaceID string) *Publisher {
	return &Publisher{
		store:       NewEventStore(workspaceID),
		broker:      NewBroker(),
		workspaceID: workspaceID,
	}
}

// NewPublisherWithBroker creates a new Publisher using a shared broker.
func NewPublisherWithBroker(workspaceID string, broker *Broker) *Publisher {
	return &Publisher{
		store:       NewEventStore(workspaceID),
		broker:      broker,
		workspaceID: workspaceID,
	}
}

// PublishEvent stores an event and broadcasts it to subscribers.
func (p *Publisher) PublishEvent(event *WorkspaceEvent) error {
	if err := p.store.Store(event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Broadcast to workspace-level subscribers.
	channel := WorkspaceChannel(p.workspaceID)
	p.broker.Publish(channel, event)

	// Broadcast to device-specific subscribers if a device ID is present.
	if event.DeviceID != "" {
		deviceCh := DeviceChannel(p.workspaceID, event.DeviceID)
		p.broker.Publish(deviceCh, event)
	}

	return nil
}

// PublishProfileUpdate creates and publishes a profile update event.
func (p *Publisher) PublishProfileUpdate(workspaceID string, profileID string, changes map[string]interface{}) error {
	payload := map[string]interface{}{
		"profile_id": profileID,
	}
	for k, v := range changes {
		payload[k] = v
	}

	event := NewEvent(workspaceID, "", EventProfileUpdated, payload)

	return p.PublishEvent(event)
}

// PublishActionExecution creates and publishes an action execution event.
func (p *Publisher) PublishActionExecution(workspaceID string, actionName string, result map[string]interface{}) error {
	payload := map[string]interface{}{
		"action_name": actionName,
	}
	for k, v := range result {
		payload[k] = v
	}

	event := NewEvent(workspaceID, "", EventActionExecuted, payload)

	return p.PublishEvent(event)
}

// Store returns the underlying event store.
func (p *Publisher) Store() *EventStore {
	return p.store
}

// Broker returns the underlying broker.
func (p *Publisher) Broker() *Broker {
	return p.broker
}
