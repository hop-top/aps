package adapters

import (
	"fmt"
	"net/http"
	"sync"

	"oss-aps-cli/internal/adapters/agentprotocol"
	"oss-aps-cli/internal/core/protocol"
)

type ProtocolAdapter interface {
	Name() string
	RegisterRoutes(mux *http.ServeMux, core protocol.APSCore) error
}

type AdapterRegistry struct {
	adapters map[string]ProtocolAdapter
	mu       sync.RWMutex
}

var registry *AdapterRegistry
var once sync.Once

func GetRegistry() *AdapterRegistry {
	once.Do(func() {
		registry = &AdapterRegistry{
			adapters: make(map[string]ProtocolAdapter),
		}
	})
	return registry
}

func (r *AdapterRegistry) Register(name string, adapter ProtocolAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter %s already registered", name)
	}

	r.adapters[name] = adapter
	return nil
}

func (r *AdapterRegistry) GetAdapter(name string) (ProtocolAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}

	return adapter, nil
}

func (r *AdapterRegistry) ListAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}

	return names
}

func (r *AdapterRegistry) RegisterAll(mux *http.ServeMux, core protocol.APSCore) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, adapter := range r.adapters {
		if err := adapter.RegisterRoutes(mux, core); err != nil {
			return fmt.Errorf("failed to register adapter %s: %w", name, err)
		}
	}

	return nil
}

func RegisterDefaults() error {
	reg := GetRegistry()
	agentProto := agentprotocol.NewAgentProtocolAdapter()
	return reg.Register("agent-protocol", agentProto)
}
