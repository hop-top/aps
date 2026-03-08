package unit

import (
	"context"
	"testing"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/acp"
	"hop.top/aps/internal/adapters/agentprotocol"
	"hop.top/aps/internal/core/protocol"
)

// TestProtocolServerInterface tests that all protocols implement ProtocolServer
func TestProtocolServerInterface(t *testing.T) {
	// Test Agent Protocol implements HTTPProtocolAdapter
	agentAdapter := agentprotocol.NewAgentProtocolAdapter()
	if err := agentAdapter.Start(context.Background(), nil); err != nil {
		t.Fatalf("failed to start agent protocol adapter: %v", err)
	}

	if agentAdapter.Name() != "agent-protocol" {
		t.Errorf("expected name 'agent-protocol', got '%s'", agentAdapter.Name())
	}

	if agentAdapter.Status() != "running" {
		t.Errorf("expected status 'running', got '%s'", agentAdapter.Status())
	}

	if err := agentAdapter.Stop(); err != nil {
		t.Fatalf("failed to stop agent protocol adapter: %v", err)
	}

	if agentAdapter.Status() != "stopped" {
		t.Errorf("expected status 'stopped', got '%s'", agentAdapter.Status())
	}
}

// TestProtocolRegistry tests the unified protocol registry
func TestProtocolRegistry(t *testing.T) {
	registry := adapters.GetProtocolRegistry()

	// List available adapters
	httpAdapters := registry.ListHTTPAdapters()
	if len(httpAdapters) == 0 {
		t.Error("expected at least one HTTP adapter registered")
	}

	// Check that agent-protocol is registered
	found := false
	for _, name := range httpAdapters {
		if name == "agent-protocol" {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent-protocol not found in HTTP adapters")
	}
}

// TestACPServerBasics tests basic ACP server operations
func TestACPServerBasics(t *testing.T) {
	// Create a mock core adapter
	coreAdapter, err := protocol.NewAPSAdapter()
	if err != nil {
		t.Fatalf("failed to create core adapter: %v", err)
	}

	// Create ACP server
	server, err := acp.NewServer("test-profile", coreAdapter)
	if err != nil {
		t.Fatalf("failed to create ACP server: %v", err)
	}

	// Verify initial state
	if server.Name() != "acp" {
		t.Errorf("expected name 'acp', got '%s'", server.Name())
	}

	if server.Status() != "stopped" {
		t.Errorf("expected initial status 'stopped', got '%s'", server.Status())
	}

	if addr := server.GetAddress(); addr != "" {
		t.Errorf("expected empty address for stdio transport, got '%s'", addr)
	}

	// Note: Cannot fully test Start/Stop without mocking stdin/stdout
	// That will be covered in integration tests
}

// TestProtocolSeparation tests that protocols can be distinguished
func TestProtocolSeparation(t *testing.T) {
	registry := adapters.GetProtocolRegistry()

	// Agent Protocol is HTTP-based
	httpAdapters := registry.ListHTTPAdapters()
	hasAgentProtocol := false
	for _, name := range httpAdapters {
		if name == "agent-protocol" {
			hasAgentProtocol = true
			break
		}
	}
	if !hasAgentProtocol {
		t.Error("agent-protocol should be in HTTP adapters")
	}
}

// TestInterfaceImplementation tests that implementations match interfaces
func TestInterfaceImplementation(t *testing.T) {
	// Verify Agent Protocol implements HTTPProtocolAdapter
	var _ protocol.HTTPProtocolAdapter = agentprotocol.NewAgentProtocolAdapter()

	// Verify ACP server implements ProtocolServer
	coreAdapter, _ := protocol.NewAPSAdapter()
	server, _ := acp.NewServer("test", coreAdapter)
	var _ protocol.ProtocolServer = server
	var _ protocol.StandaloneProtocolServer = server

	t.Log("All protocol implementations verified")
}
