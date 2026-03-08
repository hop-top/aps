package adapters

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"hop.top/aps/internal/core/protocol"
)

// ProtocolRegistry manages all protocol servers (both HTTP adapters and standalone servers)
type ProtocolRegistry struct {
	mu              sync.RWMutex
	httpAdapters    map[string]protocol.HTTPProtocolAdapter
	standaloneServers map[string]protocol.StandaloneProtocolServer
	runningServers  map[string]protocol.ProtocolServer // Tracks what's running
}

var globalRegistry = &ProtocolRegistry{
	httpAdapters:    make(map[string]protocol.HTTPProtocolAdapter),
	standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
	runningServers:  make(map[string]protocol.ProtocolServer),
}

// GetProtocolRegistry returns the global protocol registry
func GetProtocolRegistry() *ProtocolRegistry {
	return globalRegistry
}

// RegisterHTTPAdapter registers an HTTP protocol adapter
// HTTP adapters register their routes with the HTTP mux during server startup
func (r *ProtocolRegistry) RegisterHTTPAdapter(name string, adapter protocol.HTTPProtocolAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.httpAdapters[name]; exists {
		return fmt.Errorf("HTTP adapter %q already registered", name)
	}

	r.httpAdapters[name] = adapter
	return nil
}

// RegisterStandaloneServer registers a standalone protocol server
// Standalone servers manage their own lifecycle and can be started/stopped independently
func (r *ProtocolRegistry) RegisterStandaloneServer(name string, server protocol.StandaloneProtocolServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.standaloneServers[name]; exists {
		return fmt.Errorf("standalone server %q already registered", name)
	}

	r.standaloneServers[name] = server
	return nil
}

// RegisterHTTPRoutes registers all HTTP adapter routes on the provided mux
// This is called once during HTTP server startup to set up all HTTP-based protocols
func (r *ProtocolRegistry) RegisterHTTPRoutes(mux *http.ServeMux, core protocol.APSCore) error {
	r.mu.RLock()
	adapters := make(map[string]protocol.HTTPProtocolAdapter)
	for k, v := range r.httpAdapters {
		adapters[k] = v
	}
	r.mu.RUnlock()

	for name, adapter := range adapters {
		if err := adapter.RegisterRoutes(mux, core); err != nil {
			return fmt.Errorf("failed to register HTTP adapter %q: %w", name, err)
		}
	}

	return nil
}

// StartStandaloneServer starts a standalone protocol server
func (r *ProtocolRegistry) StartStandaloneServer(ctx context.Context, name string, config interface{}) error {
	r.mu.Lock()
	server, exists := r.standaloneServers[name]
	r.mu.Unlock()

	if !exists {
		return fmt.Errorf("standalone server %q not registered", name)
	}

	if err := server.Start(ctx, config); err != nil {
		return fmt.Errorf("failed to start %q server: %w", name, err)
	}

	r.mu.Lock()
	r.runningServers[name] = server
	r.mu.Unlock()

	return nil
}

// StopServer stops a running server
func (r *ProtocolRegistry) StopServer(name string) error {
	r.mu.Lock()
	server, exists := r.runningServers[name]
	r.mu.Unlock()

	if !exists {
		return fmt.Errorf("server %q not running", name)
	}

	if err := server.Stop(); err != nil {
		return fmt.Errorf("failed to stop %q server: %w", name, err)
	}

	r.mu.Lock()
	delete(r.runningServers, name)
	r.mu.Unlock()

	return nil
}

// GetServerStatus returns the status of a server
func (r *ProtocolRegistry) GetServerStatus(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check running servers first
	if server, exists := r.runningServers[name]; exists {
		return server.Status(), nil
	}

	// Check standalone servers
	if server, exists := r.standaloneServers[name]; exists {
		return server.Status(), nil
	}

	// Check HTTP adapters
	if adapter, exists := r.httpAdapters[name]; exists {
		return adapter.Status(), nil
	}

	return "", fmt.Errorf("server %q not registered", name)
}

// ListHTTPAdapters returns the names of all registered HTTP adapters
func (r *ProtocolRegistry) ListHTTPAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.httpAdapters))
	for name := range r.httpAdapters {
		names = append(names, name)
	}
	return names
}

// ListStandaloneServers returns the names of all registered standalone servers
func (r *ProtocolRegistry) ListStandaloneServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.standaloneServers))
	for name := range r.standaloneServers {
		names = append(names, name)
	}
	return names
}

// ListRunningServers returns the names of all currently running servers
func (r *ProtocolRegistry) ListRunningServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.runningServers))
	for name := range r.runningServers {
		names = append(names, name)
	}
	return names
}

// GetStandaloneServerAddress returns the address of a standalone server
func (r *ProtocolRegistry) GetStandaloneServerAddress(name string) (string, error) {
	r.mu.RLock()
	server, exists := r.standaloneServers[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("standalone server %q not registered", name)
	}

	return server.GetAddress(), nil
}
