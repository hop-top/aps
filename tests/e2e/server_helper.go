package e2e

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	a2asdk "github.com/a2aproject/a2a-go/a2a"

	"oss-aps-cli/internal/a2a"
	"oss-aps-cli/internal/core"
)

// TestServerConfig holds configuration for a test server
type TestServerConfig struct {
	Profile    *core.Profile
	ServerAddr string
}

// TestServer wraps an A2A server for testing
type TestServer struct {
	Server     *a2a.Server
	Profile    *core.Profile
	t          *testing.T
	cancelFunc context.CancelFunc
}

// MockTaskHandler implements basic task handling for tests
type MockTaskHandler struct{}

func (h *MockTaskHandler) HandleMessage(ctx context.Context, message *a2asdk.Message) (*a2asdk.Task, error) {
	// Create a simple task response
	task := &a2asdk.Task{
		ID: a2asdk.NewTaskID(),
		Status: a2asdk.TaskStatus{
			State: a2asdk.TaskStateWorking,
		},
	}
	return task, nil
}

// StartTestServer starts an A2A server for testing
func StartTestServer(t *testing.T, config TestServerConfig) *TestServer {
	t.Helper()

	// Ensure the address is available
	if !isPortAvailable(config.ServerAddr) {
		t.Fatalf("Port %s is not available", config.ServerAddr)
	}

	// Create server
	storageConfig := a2a.DefaultStorageConfig()
	server, err := a2a.NewServer(config.Profile, storageConfig)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a context for the server that won't be canceled until Stop() is called
	serverCtx, cancel := context.WithCancel(context.Background())

	// Start server in background
	go func() {
		if err := server.Start(serverCtx, nil); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()

	if !waitForServer(waitCtx, config.ServerAddr) {
		cancel()
		server.Stop()
		t.Fatalf("Server failed to start on %s", config.ServerAddr)
	}

	return &TestServer{
		Server:     server,
		Profile:    config.Profile,
		t:          t,
		cancelFunc: cancel,
	}
}

// Stop stops the test server
func (ts *TestServer) Stop() {
	if ts.cancelFunc != nil {
		ts.cancelFunc()
	}
	if ts.Server != nil {
		if err := ts.Server.Stop(); err != nil {
			ts.t.Logf("Error stopping server: %v", err)
		}
	}
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// waitForServer waits for a server to become available
func waitForServer(ctx context.Context, addr string) bool {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return true
			}
		}
	}
}

// CreateTestProfile creates a profile for testing
func CreateTestProfile(id, displayName, addr string) *core.Profile {
	return &core.Profile{
		ID:           id,
		DisplayName:  displayName,
		Capabilities: []string{"execute", "deploy"},
		A2A: &core.A2AConfig{
			Enabled:         true,
			ProtocolBinding: "jsonrpc",
			ListenAddr:      addr,
			SecurityScheme:  "none",
			IsolationTier:   "process",
		},
	}
}

// GetAvailablePort finds an available port for testing
func GetAvailablePort() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr, nil
}

// GetAvailableAddress returns an available address with port
func GetAvailableAddress(t *testing.T) string {
	t.Helper()
	addr, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	return addr
}

// GetTestProfileWithAvailablePort creates a test profile with an available port
func GetTestProfileWithAvailablePort(t *testing.T, id, displayName string) *core.Profile {
	t.Helper()
	addr := GetAvailableAddress(t)
	return CreateTestProfile(id, displayName, addr)
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Logf("Timeout waiting for: %s", message)
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		}
	}
}

// AssertEventually checks a condition repeatedly until it's true or timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		<-ticker.C
	}

	if len(msgAndArgs) > 0 {
		t.Fatalf("Condition not met within %v: %v", timeout, fmt.Sprint(msgAndArgs...))
	} else {
		t.Fatalf("Condition not met within %v", timeout)
	}
}
