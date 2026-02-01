package protocol

import (
	"context"
	"net/http"
)

// ProtocolServer represents a protocol server that can handle agent communication
// All agent communication protocols (Agent Protocol, A2A, ACP) implement this interface
type ProtocolServer interface {
	// Name returns the protocol name (e.g., "agent-protocol", "a2a", "acp")
	Name() string

	// Start initializes and starts the protocol server with the given context
	// config is protocol-specific configuration (can be nil)
	Start(ctx context.Context, config interface{}) error

	// Stop gracefully stops the protocol server
	Stop() error

	// Status returns the current status of the server
	// Returns a string like "running", "stopped", "error"
	Status() string
}

// HTTPProtocolAdapter is a marker interface for protocols that serve HTTP endpoints
// These protocols will register their routes via RegisterRoutes
type HTTPProtocolAdapter interface {
	ProtocolServer
	// RegisterRoutes registers HTTP routes on the provided mux
	// This is called during HTTP server setup
	RegisterRoutes(mux *http.ServeMux, core APSCore) error
}

// StandaloneProtocolServer is a marker interface for protocols that manage their own lifecycle
// These protocols start and stop their own servers (HTTP, stdio, etc)
type StandaloneProtocolServer interface {
	ProtocolServer
	// GetAddress returns the address where the server is listening (for HTTP servers)
	// Returns empty string for non-network servers (stdio)
	GetAddress() string
}

// HTTPBridge allows any ProtocolServer to be exposed via HTTP transport
// This is the adapter layer that bridges non-HTTP protocols (stdio) through HTTP
// Example: Expose ACP (stdio) protocol through HTTP endpoints for remote clients
type HTTPBridge interface {
	ProtocolServer
	// GetHTTPHandler returns an http.Handler that exposes this protocol server via HTTP
	// The handler should translate HTTP requests to the protocol's native format
	GetHTTPHandler() http.Handler
}
