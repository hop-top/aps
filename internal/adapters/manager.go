// Package adapters wires aps protocol adapters into kit's ext.Manager
// while keeping the aps-specific HTTP-routing concern as a thin façade.
package adapters

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"hop.top/aps/internal/adapters/agentprotocol"
	"hop.top/aps/internal/adapters/messenger"
	"hop.top/aps/internal/core/protocol"
	"hop.top/kit/go/ai/ext"
)

// Manager wraps kit's ext.Manager with two aps-specific affordances:
//
//   - RegisterRoutes — fan out to every Extension that also implements
//     protocol.HTTPProtocolAdapter so HTTP-bound adapters can attach their
//     handlers to a shared mux.
//   - Names — flat list of registered extension names (handy for cobra
//     completion and operator-facing CLIs).
//
// All other lifecycle (InitAll, CloseAll, Add) is delegated to ext.Manager
// untouched.
type Manager struct {
	ext *ext.Manager

	mu    sync.RWMutex
	order []string // insertion order for stable Names() output
}

// NewManager returns a Manager backed by a fresh ext.Manager. Logging is
// disabled by default; callers wanting structured logs should construct
// their own ext.Manager directly.
func NewManager() *Manager {
	return &Manager{ext: ext.NewManager(nil)}
}

// Add registers an extension with the underlying ext.Manager.
func (m *Manager) Add(e ext.Extension) {
	m.ext.Add(e)
	m.mu.Lock()
	m.order = append(m.order, e.Meta().Name)
	m.mu.Unlock()
}

// InitAll forwards to ext.Manager.InitAll.
func (m *Manager) InitAll(ctx context.Context) error { return m.ext.InitAll(ctx) }

// CloseAll forwards to ext.Manager.CloseAll.
func (m *Manager) CloseAll() []error { return m.ext.CloseAll() }

// Extensions returns the registered extensions in insertion order.
func (m *Manager) Extensions() []ext.Extension { return m.ext.Extensions() }

// Names returns the registered extension names in insertion order.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

// SortedNames returns Names() sorted alphabetically.
func (m *Manager) SortedNames() []string {
	out := m.Names()
	sort.Strings(out)
	return out
}

// RegisterRoutes attaches HTTP handlers for every Extension that also
// implements protocol.HTTPProtocolAdapter. Extensions without HTTP routes
// are skipped silently — they participate in lifecycle but not routing.
func (m *Manager) RegisterRoutes(mux *http.ServeMux, core protocol.APSCore) error {
	for _, e := range m.ext.Extensions() {
		ha, ok := e.(protocol.HTTPProtocolAdapter)
		if !ok {
			continue
		}
		if err := ha.RegisterRoutes(mux, core); err != nil {
			return err
		}
	}
	return nil
}

// DefaultManager returns a Manager pre-populated with the default aps
// adapter set (currently just agent-protocol). Callers running the HTTP
// server should InitAll before RegisterRoutes.
func DefaultManager() *Manager {
	m := NewManager()
	m.Add(agentprotocol.NewAgentProtocolAdapter())
	m.Add(messenger.NewAdapter())
	return m
}
