package adapters

import (
	"context"
	"net/http"
	"testing"

	"hop.top/aps/internal/adapters/agentprotocol"
	"hop.top/kit/go/ai/ext"
)

// TestAgentProtocolAdapter_ImplementsExt verifies the agentprotocol adapter
// fulfils kit's ext.Extension contract: Meta, Capabilities, Init, Close.
func TestAgentProtocolAdapter_ImplementsExt(t *testing.T) {
	var _ ext.Extension = (*agentprotocol.AgentProtocolAdapter)(nil)

	a := agentprotocol.NewAgentProtocolAdapter()
	meta := a.Meta()
	if meta.Name != "agent-protocol" {
		t.Fatalf("Meta.Name = %q, want agent-protocol", meta.Name)
	}
	if a.Capabilities() == 0 {
		t.Fatal("Capabilities should be non-zero")
	}
}

// TestNewManager_LifecycleAndRouting checks the new ext.Manager-backed
// registry initialises adapters, routes HTTP via Routable, and closes cleanly.
func TestNewManager_LifecycleAndRouting(t *testing.T) {
	mgr := NewManager()
	a := agentprotocol.NewAgentProtocolAdapter()
	mgr.Add(a)

	ctx := context.Background()
	if err := mgr.InitAll(ctx); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	if a.Status() != "running" {
		t.Fatalf("status after InitAll = %q, want running", a.Status())
	}

	mux := http.NewServeMux()
	if err := mgr.RegisterRoutes(mux, nil); err != nil {
		t.Fatalf("RegisterRoutes: %v", err)
	}

	if errs := mgr.CloseAll(); len(errs) != 0 {
		t.Fatalf("CloseAll: %v", errs)
	}
	if a.Status() != "stopped" {
		t.Fatalf("status after CloseAll = %q, want stopped", a.Status())
	}
}

// TestNewManager_ListNames mirrors the legacy AdapterRegistry.ListAdapters
// behaviour for any caller that relied on enumerating registered names.
func TestNewManager_ListNames(t *testing.T) {
	mgr := NewManager()
	mgr.Add(agentprotocol.NewAgentProtocolAdapter())

	names := mgr.Names()
	if len(names) != 1 || names[0] != "agent-protocol" {
		t.Fatalf("Names = %v, want [agent-protocol]", names)
	}
}
