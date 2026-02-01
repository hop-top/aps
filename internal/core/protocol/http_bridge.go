package protocol

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// DefaultHTTPBridge is a generic HTTP bridge that wraps any ProtocolServer
// and exposes it as JSON-RPC over HTTP
type DefaultHTTPBridge struct {
	server ProtocolServer
	mu     sync.RWMutex
}

var _ HTTPBridge = (*DefaultHTTPBridge)(nil)

// NewDefaultHTTPBridge creates a new HTTP bridge for a protocol server
func NewDefaultHTTPBridge(server ProtocolServer) *DefaultHTTPBridge {
	return &DefaultHTTPBridge{
		server: server,
	}
}

// Name returns the protocol name
func (b *DefaultHTTPBridge) Name() string {
	return b.server.Name() + "-http-bridge"
}

// Start initializes and starts the bridge (delegates to wrapped server)
func (b *DefaultHTTPBridge) Start(ctx context.Context, config interface{}) error {
	return b.server.Start(ctx, config)
}

// Stop gracefully stops the bridge (delegates to wrapped server)
func (b *DefaultHTTPBridge) Stop() error {
	return b.server.Stop()
}

// Status returns the current status of the bridge
func (b *DefaultHTTPBridge) Status() string {
	return b.server.Status()
}

// GetHTTPHandler returns an HTTP handler that exposes the protocol via HTTP
// Translates HTTP requests/responses to the protocol's native format
func (b *DefaultHTTPBridge) GetHTTPHandler() http.Handler {
	return http.HandlerFunc(b.handleHTTPRequest)
}

// handleHTTPRequest processes incoming HTTP requests and translates them
// to the protocol's native format
func (b *DefaultHTTPBridge) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// Set response header
	w.Header().Set("Content-Type", "application/json")

	// For now, return a simple bridge response
	// In a real implementation, this would:
	// 1. Parse the HTTP request into the protocol's format
	// 2. Forward to the wrapped server
	// 3. Translate the response back to HTTP
	response := map[string]interface{}{
		"protocol": b.server.Name(),
		"status":   b.server.Status(),
		"method":   r.Method,
		"path":     r.URL.Path,
		"message":  "HTTP bridge is active",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// JSONRPCHTTPBridge wraps a JSON-RPC protocol (like ACP) and exposes it via HTTP
// This allows stdio-based protocols to be accessed from HTTP clients
type JSONRPCHTTPBridge struct {
	server ProtocolServer
	mu     sync.RWMutex
}

var _ HTTPBridge = (*JSONRPCHTTPBridge)(nil)

// NewJSONRPCHTTPBridge creates a new HTTP bridge for JSON-RPC protocols
func NewJSONRPCHTTPBridge(server ProtocolServer) *JSONRPCHTTPBridge {
	return &JSONRPCHTTPBridge{
		server: server,
	}
}

// Name returns the protocol name
func (b *JSONRPCHTTPBridge) Name() string {
	return b.server.Name() + "-http"
}

// Start initializes and starts the bridge
func (b *JSONRPCHTTPBridge) Start(ctx context.Context, config interface{}) error {
	return b.server.Start(ctx, config)
}

// Stop gracefully stops the bridge
func (b *JSONRPCHTTPBridge) Stop() error {
	return b.server.Stop()
}

// Status returns the current status
func (b *JSONRPCHTTPBridge) Status() string {
	return b.server.Status()
}

// GetHTTPHandler returns an HTTP handler that translates HTTP to JSON-RPC
func (b *JSONRPCHTTPBridge) GetHTTPHandler() http.Handler {
	return http.HandlerFunc(b.handleJSONRPC)
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  interface{}       `json:"params,omitempty"`
	ID      interface{}       `json:"id,omitempty"`
	Error   *JSONRPCError     `json:"error,omitempty"`
	Result  interface{}       `json:"result,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// handleJSONRPC translates HTTP requests to JSON-RPC and forwards them
func (b *JSONRPCHTTPBridge) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse incoming request
	var req JSONRPCRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, -32700, "Parse error", nil)
		return
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, -32700, "Parse error", nil)
		return
	}

	// Validate JSON-RPC 2.0 format
	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, http.StatusBadRequest, -32600, "Invalid Request", nil)
		return
	}

	if req.Method == "" {
		writeJSONRPCError(w, http.StatusBadRequest, -32601, "Method not found", nil)
		return
	}

	// Build response (in a real implementation, forward to the actual protocol handler)
	response := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"method":   req.Method,
			"protocol": b.server.Name(),
			"status":   b.server.Status(),
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeJSONRPCError writes a JSON-RPC error response
func writeJSONRPCError(w http.ResponseWriter, httpStatus, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	errResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if data != nil {
		errResponse["error"].(map[string]interface{})["data"] = data
	}

	json.NewEncoder(w).Encode(errResponse)
}

// ProtocolServerAdapter adapts any ProtocolServer to be usable as HTTPProtocolAdapter
// This allows standalone servers to also register HTTP routes if needed
type ProtocolServerAdapter struct {
	server ProtocolServer
}

var _ HTTPProtocolAdapter = (*ProtocolServerAdapter)(nil)

// NewProtocolServerAdapter creates a new adapter
func NewProtocolServerAdapter(server ProtocolServer) *ProtocolServerAdapter {
	return &ProtocolServerAdapter{
		server: server,
	}
}

// Name returns the protocol name
func (a *ProtocolServerAdapter) Name() string {
	return a.server.Name()
}

// Start initializes and starts the protocol
func (a *ProtocolServerAdapter) Start(ctx context.Context, config interface{}) error {
	return a.server.Start(ctx, config)
}

// Stop gracefully stops the protocol
func (a *ProtocolServerAdapter) Stop() error {
	return a.server.Stop()
}

// Status returns the current status
func (a *ProtocolServerAdapter) Status() string {
	return a.server.Status()
}

// RegisterRoutes allows the protocol to register routes if supported
// For most standalone servers, this is a no-op
func (a *ProtocolServerAdapter) RegisterRoutes(mux *http.ServeMux, core APSCore) error {
	// Optional: if the server supports HTTP routing, it can implement this
	// For now, this is a no-op - the server manages its own HTTP server
	return nil
}
