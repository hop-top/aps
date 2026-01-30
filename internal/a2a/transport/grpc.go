package transport

import (
	"context"
	"fmt"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"
)

// GRPCConfig holds gRPC transport configuration
type GRPCConfig struct {
	Endpoint    string
	Timeout     time.Duration
	MTLSEnabled bool
}

// DefaultGRPCConfig returns default gRPC configuration
func DefaultGRPCConfig(endpoint string) *GRPCConfig {
	return &GRPCConfig{
		Endpoint:    endpoint,
		Timeout:     30 * time.Second,
		MTLSEnabled: false,
	}
}

// GRPCTransport implements A2A transport via gRPC
type GRPCTransport struct {
	config  *GRPCConfig
	handler MessageHandler
	msgChan chan *a2a.Message
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

var _ Transport = (*GRPCTransport)(nil)

// NewGRPCTransport creates a new gRPC transport instance
func NewGRPCTransport(config *GRPCConfig, handler MessageHandler) (*GRPCTransport, error) {
	if config == nil {
		return nil, fmt.Errorf("grpc config cannot be nil")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCTransport{
		config:  config,
		handler: handler,
		msgChan: make(chan *a2a.Message, 100),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}, nil
}

// Type returns the transport type
func (t *GRPCTransport) Type() TransportType {
	return TransportGRPC
}

// Send sends a message via gRPC transport
func (t *GRPCTransport) Send(ctx context.Context, message *a2a.Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	return fmt.Errorf("gRPC transport not yet fully implemented")
}

// Receive receives a message from gRPC transport
func (t *GRPCTransport) Receive(ctx context.Context) (*a2a.Message, error) {
	select {
	case msg := <-t.msgChan:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes gRPC transport
func (t *GRPCTransport) Close() error {
	t.cancel()
	t.running = false
	return nil
}

// IsHealthy checks if gRPC transport is operational
func (t *GRPCTransport) IsHealthy() bool {
	return t.running
}

// HandleServerResponse processes incoming messages from gRPC server
func (t *GRPCTransport) HandleServerResponse(message *a2a.Message) error {
	select {
	case t.msgChan <- message:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("transport is closed")
	}
}
