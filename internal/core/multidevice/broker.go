package multidevice

import (
	"sync"
)

const (
	// subscriberBufferSize is the channel buffer size for each subscriber.
	subscriberBufferSize = 64
)

// Broker manages in-memory pub/sub for workspace events.
//
// Channel naming conventions:
//
//	workspace:{workspace_id}
//	workspace:{workspace_id}:device:{device_id}
//	device:{device_id}:presence
type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *WorkspaceEvent
}

// NewBroker creates a new Broker.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan *WorkspaceEvent),
	}
}

// Subscribe creates and returns a new channel that receives events published
// to the given channel name.
func (b *Broker) Subscribe(channel string) <-chan *WorkspaceEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *WorkspaceEvent, subscriberBufferSize)
	b.subscribers[channel] = append(b.subscribers[channel], ch)
	return ch
}

// Unsubscribe removes a subscriber channel from the given channel name and
// closes it.
func (b *Broker) Unsubscribe(channel string, ch <-chan *WorkspaceEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, ok := b.subscribers[channel]
	if !ok {
		return
	}

	var remaining []chan *WorkspaceEvent
	for _, sub := range subs {
		if sub == ch {
			close(sub)
			continue
		}
		remaining = append(remaining, sub)
	}

	if len(remaining) == 0 {
		delete(b.subscribers, channel)
	} else {
		b.subscribers[channel] = remaining
	}
}

// Publish sends an event to all subscribers of the given channel name.
// Non-blocking: if a subscriber's buffer is full, the event is dropped for
// that subscriber.
func (b *Broker) Publish(channel string, event *WorkspaceEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[channel]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Subscriber buffer full; drop event to avoid blocking.
		}
	}
}

// SubscriberCount returns the number of active subscribers on a channel.
func (b *Broker) SubscriberCount(channel string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers[channel])
}

// WorkspaceChannel returns the standard channel name for workspace-level events.
func WorkspaceChannel(workspaceID string) string {
	return "workspace:" + workspaceID
}

// DeviceChannel returns the standard channel name for device-specific events
// within a workspace.
func DeviceChannel(workspaceID, deviceID string) string {
	return "workspace:" + workspaceID + ":device:" + deviceID
}

// PresenceChannel returns the standard channel name for device presence events.
func PresenceChannel(deviceID string) string {
	return "device:" + deviceID + ":presence"
}
