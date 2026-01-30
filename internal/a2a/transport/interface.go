package transport

import (
	"context"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

// TransportType represents the type of A2A transport
type TransportType string

const (
	TransportIPC  TransportType = "ipc"
	TransportHTTP TransportType = "http"
	TransportGRPC TransportType = "grpc"
)

// Transport represents an A2A transport adapter
type Transport interface {
	// Type returns the transport type
	Type() TransportType

	// Send sends a message via the transport
	Send(ctx context.Context, message *a2a.Message) error

	// Receive receives a message from the transport
	Receive(ctx context.Context) (*a2a.Message, error)

	// Close closes the transport connection
	Close() error

	// IsHealthy checks if transport is operational
	IsHealthy() bool
}

// MessageHandler handles incoming messages from transport
type MessageHandler interface {
	// HandleMessage processes an incoming message
	HandleMessage(ctx context.Context, message *a2a.Message) error
}
