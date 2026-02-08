package adapters

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"oss-aps-cli/internal/acp"
	"oss-aps-cli/internal/adapters/agentprotocol"
	"oss-aps-cli/internal/core/protocol"
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
	registry := GetProtocolRegistry()

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
	registry := GetProtocolRegistry()

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

// ============================================================================
// HTTP Adapter Registration Tests (5 tests)
// ============================================================================

// TestRegisterHTTPAdapterSuccess tests that an HTTP adapter can be registered
func TestRegisterHTTPAdapterSuccess(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	adapter := &MockHTTPAdapter{
		name:   "test-http",
		status: "stopped",
	}

	err := registry.RegisterHTTPAdapter("test-http", adapter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(registry.httpAdapters))

	// Verify adapter is stored correctly
	registered, exists := registry.httpAdapters["test-http"]
	assert.True(t, exists)
	assert.Equal(t, adapter, registered)
}

// TestRegisterHTTPAdapterDuplicate tests that duplicate HTTP adapter registration fails
func TestRegisterHTTPAdapterDuplicate(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	adapter1 := &MockHTTPAdapter{name: "test", status: "stopped"}
	adapter2 := &MockHTTPAdapter{name: "test", status: "stopped"}

	err1 := registry.RegisterHTTPAdapter("test", adapter1)
	assert.NoError(t, err1)

	err2 := registry.RegisterHTTPAdapter("test", adapter2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "already registered")
}

// TestListHTTPAdapters tests that all registered HTTP adapters are listed
func TestListHTTPAdapters(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Register multiple adapters
	adapters := []string{"adapter-1", "adapter-2", "adapter-3"}
	for _, name := range adapters {
		adapter := &MockHTTPAdapter{name: name, status: "stopped"}
		registry.RegisterHTTPAdapter(name, adapter)
	}

	// List and verify
	listed := registry.ListHTTPAdapters()
	assert.Equal(t, 3, len(listed))

	// Verify all adapters are listed
	for _, name := range adapters {
		assert.Contains(t, listed, name)
	}
}

// TestRegisterHTTPRoutes tests that HTTP routes are registered on a mux
func TestRegisterHTTPRoutes(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Create mock adapters
	mockAdapter1 := new(MockHTTPAdapter)
	mockAdapter1.On("RegisterRoutes", mock.Anything, mock.Anything).Return(nil)

	mockAdapter2 := new(MockHTTPAdapter)
	mockAdapter2.On("RegisterRoutes", mock.Anything, mock.Anything).Return(nil)

	registry.RegisterHTTPAdapter("adapter-1", mockAdapter1)
	registry.RegisterHTTPAdapter("adapter-2", mockAdapter2)

	// Create a test mux
	mux := http.NewServeMux()
	coreAdapter, _ := protocol.NewAPSAdapter()

	// Register routes
	err := registry.RegisterHTTPRoutes(mux, coreAdapter)
	assert.NoError(t, err)

	// Verify RegisterRoutes was called on all adapters
	mockAdapter1.AssertCalled(t, "RegisterRoutes", mux, coreAdapter)
	mockAdapter2.AssertCalled(t, "RegisterRoutes", mux, coreAdapter)
}

// TestRegisterHTTPRoutesError tests that RegisterHTTPRoutes handles adapter errors
func TestRegisterHTTPRoutesError(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Create a mock adapter that returns an error
	mockAdapter := new(MockHTTPAdapter)
	mockAdapter.On("RegisterRoutes", mock.Anything, mock.Anything).
		Return(assert.AnError)

	registry.RegisterHTTPAdapter("failing-adapter", mockAdapter)

	mux := http.NewServeMux()
	coreAdapter, _ := protocol.NewAPSAdapter()

	err := registry.RegisterHTTPRoutes(mux, coreAdapter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failing-adapter")
}

// ============================================================================
// Standalone Server Management Tests (5 tests)
// ============================================================================

// TestRegisterStandaloneServer tests that a standalone server can be registered
func TestRegisterStandaloneServer(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	server := &MockStandaloneServer{
		name:    "test-server",
		status:  "stopped",
		address: "localhost:9000",
	}

	err := registry.RegisterStandaloneServer("test-server", server)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(registry.standaloneServers))

	registered, exists := registry.standaloneServers["test-server"]
	assert.True(t, exists)
	assert.Equal(t, server, registered)
}

// TestRegisterStandaloneServerDuplicate tests duplicate standalone server registration fails
func TestRegisterStandaloneServerDuplicate(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	server1 := &MockStandaloneServer{name: "test", status: "stopped", address: "localhost:9000"}
	server2 := &MockStandaloneServer{name: "test", status: "stopped", address: "localhost:9001"}

	err1 := registry.RegisterStandaloneServer("test", server1)
	assert.NoError(t, err1)

	err2 := registry.RegisterStandaloneServer("test", server2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "already registered")
}

// TestStartStandaloneServer tests that a standalone server can be started
func TestStartStandaloneServer(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	mockServer := new(MockStandaloneServer)
	mockServer.On("Start", mock.Anything, mock.Anything).Return(nil)
	mockServer.On("Status").Return("running")
	mockServer.On("GetAddress").Return("localhost:9000")
	mockServer.On("Name").Return("test-server")

	registry.RegisterStandaloneServer("test-server", mockServer)

	ctx := context.Background()
	err := registry.StartStandaloneServer(ctx, "test-server", nil)
	assert.NoError(t, err)

	// Verify server is in running servers
	assert.Equal(t, 1, len(registry.runningServers))
	assert.Contains(t, registry.runningServers, "test-server")
}

// TestStartStandaloneServerNotFound tests starting non-existent standalone server
func TestStartStandaloneServerNotFound(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	ctx := context.Background()
	err := registry.StartStandaloneServer(ctx, "non-existent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

// TestStopServerCleanup tests that stopping a server removes it from running servers
func TestStopServerCleanup(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	mockServer := new(MockStandaloneServer)
	mockServer.On("Start", mock.Anything, mock.Anything).Return(nil)
	mockServer.On("Stop").Return(nil)
	mockServer.On("Status").Return("running")
	mockServer.On("GetAddress").Return("localhost:9000")
	mockServer.On("Name").Return("test-server")

	registry.RegisterStandaloneServer("test-server", mockServer)

	// Start the server
	ctx := context.Background()
	registry.StartStandaloneServer(ctx, "test-server", nil)
	assert.Equal(t, 1, len(registry.runningServers))

	// Stop the server
	err := registry.StopServer("test-server")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(registry.runningServers))

	mockServer.AssertCalled(t, "Stop")
}

// TestGetServerStatus tests retrieving server status
func TestGetServerStatus(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Test HTTP adapter status
	httpAdapter := &MockHTTPAdapter{name: "http-adapter", status: "running"}
	registry.RegisterHTTPAdapter("http-adapter", httpAdapter)

	status, err := registry.GetServerStatus("http-adapter")
	assert.NoError(t, err)
	assert.Equal(t, "running", status)

	// Test standalone server status
	standaloneServer := &MockStandaloneServer{
		name:    "standalone-server",
		status:  "stopped",
		address: "localhost:9000",
	}
	registry.RegisterStandaloneServer("standalone-server", standaloneServer)

	status, err = registry.GetServerStatus("standalone-server")
	assert.NoError(t, err)
	assert.Equal(t, "stopped", status)
}

// TestGetServerStatusNotFound tests status of non-existent server
func TestGetServerStatusNotFound(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	status, err := registry.GetServerStatus("non-existent")
	assert.Error(t, err)
	assert.Equal(t, "", status)
	assert.Contains(t, err.Error(), "not registered")
}

// TestMultipleServerInstances tests managing multiple server instances
func TestMultipleServerInstances(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Register multiple standalone servers
	for i := 1; i <= 3; i++ {
		server := new(MockStandaloneServer)
		serverName := "server-" + string(rune('0'+i))
		server.On("Start", mock.Anything, mock.Anything).Return(nil)
		server.On("Status").Return("running")
		server.On("GetAddress").Return("localhost:900" + string(rune('0'+i)))
		server.On("Name").Return(serverName)

		registry.RegisterStandaloneServer(serverName, server)
	}

	assert.Equal(t, 3, len(registry.ListStandaloneServers()))

	// Start all servers
	ctx := context.Background()
	for _, name := range registry.ListStandaloneServers() {
		err := registry.StartStandaloneServer(ctx, name, nil)
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, len(registry.runningServers))
}

// ============================================================================
// Registry Operations Tests (5 tests)
// ============================================================================

// TestGetHTTPAdapterRetrieval tests retrieving an HTTP adapter
func TestGetHTTPAdapterRetrieval(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	adapter := &MockHTTPAdapter{name: "test-adapter", status: "stopped"}
	registry.RegisterHTTPAdapter("test-adapter", adapter)

	// Verify we can retrieve the adapter
	registry.mu.RLock()
	retrieved, exists := registry.httpAdapters["test-adapter"]
	registry.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, adapter, retrieved)
}

// TestGetStandaloneServerRetrieval tests retrieving a standalone server
func TestGetStandaloneServerRetrieval(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	server := &MockStandaloneServer{
		name:    "test-server",
		status:  "stopped",
		address: "localhost:9000",
	}
	registry.RegisterStandaloneServer("test-server", server)

	// Verify we can retrieve the server
	registry.mu.RLock()
	retrieved, exists := registry.standaloneServers["test-server"]
	registry.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, server, retrieved)
}

// TestGetStandaloneServerAddress tests retrieving server address
func TestGetStandaloneServerAddress(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	server := &MockStandaloneServer{
		name:    "test-server",
		status:  "stopped",
		address: "localhost:9000",
	}
	registry.RegisterStandaloneServer("test-server", server)

	address, err := registry.GetStandaloneServerAddress("test-server")
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9000", address)
}

// TestGetStandaloneServerAddressNotFound tests address retrieval for non-existent server
func TestGetStandaloneServerAddressNotFound(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	address, err := registry.GetStandaloneServerAddress("non-existent")
	assert.Error(t, err)
	assert.Equal(t, "", address)
	assert.Contains(t, err.Error(), "not registered")
}

// TestThreadSafeOperations tests that registry operations are thread-safe
func TestThreadSafeOperations(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*2)

	// Concurrent registration of adapters
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			adapter := &MockHTTPAdapter{
				name:   "adapter-" + string(rune('0'+idx)),
				status: "stopped",
			}
			if err := registry.RegisterHTTPAdapter("adapter-"+string(rune('0'+idx)), adapter); err != nil {
				errChan <- err
			}
		}(i)
	}

	// Concurrent registration of servers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server := &MockStandaloneServer{
				name:    "server-" + string(rune('0'+idx)),
				status:  "stopped",
				address: "localhost:900" + string(rune('0'+idx)),
			}
			if err := registry.RegisterStandaloneServer("server-"+string(rune('0'+idx)), server); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred during concurrent operations
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify all items were registered
	assert.Equal(t, numGoroutines, len(registry.ListHTTPAdapters()))
	assert.Equal(t, numGoroutines, len(registry.ListStandaloneServers()))
}

// TestListRunningServers tests listing currently running servers
func TestListRunningServers(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Register and start multiple servers
	for i := 1; i <= 3; i++ {
		server := new(MockStandaloneServer)
		serverName := "server-" + string(rune('0'+i))
		server.On("Start", mock.Anything, mock.Anything).Return(nil)
		server.On("Status").Return("running")
		server.On("GetAddress").Return("localhost:900" + string(rune('0'+i)))
		server.On("Name").Return(serverName)

		registry.RegisterStandaloneServer(serverName, server)
		registry.StartStandaloneServer(context.Background(), serverName, nil)
	}

	running := registry.ListRunningServers()
	assert.Equal(t, 3, len(running))
}

// TestRegistryReset tests cleanup of registry state
func TestRegistryReset(t *testing.T) {
	registry := &ProtocolRegistry{
		httpAdapters:      make(map[string]protocol.HTTPProtocolAdapter),
		standaloneServers: make(map[string]protocol.StandaloneProtocolServer),
		runningServers:    make(map[string]protocol.ProtocolServer),
	}

	// Add some data
	httpAdapter := &MockHTTPAdapter{name: "adapter", status: "stopped"}
	registry.RegisterHTTPAdapter("adapter", httpAdapter)

	standaloneServer := &MockStandaloneServer{
		name:    "server",
		status:  "stopped",
		address: "localhost:9000",
	}
	registry.RegisterStandaloneServer("server", standaloneServer)

	// Verify data exists
	assert.Equal(t, 1, len(registry.ListHTTPAdapters()))
	assert.Equal(t, 1, len(registry.ListStandaloneServers()))

	// Reset registry
	registry.mu.Lock()
	registry.httpAdapters = make(map[string]protocol.HTTPProtocolAdapter)
	registry.standaloneServers = make(map[string]protocol.StandaloneProtocolServer)
	registry.runningServers = make(map[string]protocol.ProtocolServer)
	registry.mu.Unlock()

	// Verify cleanup
	assert.Equal(t, 0, len(registry.ListHTTPAdapters()))
	assert.Equal(t, 0, len(registry.ListStandaloneServers()))
	assert.Equal(t, 0, len(registry.ListRunningServers()))
}

// ============================================================================
// Mock implementations
// ============================================================================

// MockHTTPAdapter is a mock implementation of HTTPProtocolAdapter
type MockHTTPAdapter struct {
	mock.Mock
	name   string
	status string
}

func (m *MockHTTPAdapter) Name() string {
	return m.name
}

func (m *MockHTTPAdapter) Status() string {
	return m.status
}

func (m *MockHTTPAdapter) Start(ctx context.Context, config interface{}) error {
	args := m.Called(ctx, config)
	m.status = "running"
	return args.Error(0)
}

func (m *MockHTTPAdapter) Stop() error {
	args := m.Called()
	m.status = "stopped"
	return args.Error(0)
}

func (m *MockHTTPAdapter) RegisterRoutes(mux *http.ServeMux, core protocol.APSCore) error {
	args := m.Called(mux, core)
	return args.Error(0)
}

// MockStandaloneServer is a mock implementation of StandaloneProtocolServer
type MockStandaloneServer struct {
	mock.Mock
	name    string
	status  string
	address string
}

func (m *MockStandaloneServer) Name() string {
	return m.name
}

func (m *MockStandaloneServer) Status() string {
	return m.status
}

func (m *MockStandaloneServer) Start(ctx context.Context, config interface{}) error {
	args := m.Called(ctx, config)
	m.status = "running"
	return args.Error(0)
}

func (m *MockStandaloneServer) Stop() error {
	args := m.Called()
	m.status = "stopped"
	return args.Error(0)
}

func (m *MockStandaloneServer) GetAddress() string {
	return m.address
}
