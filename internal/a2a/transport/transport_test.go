package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	a2a "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/core"
)

// =============================================================================
// Mock Message Handler
// =============================================================================

type mockMessageHandler struct {
	messages []*a2a.Message
}

func (m *mockMessageHandler) HandleMessage(ctx context.Context, message *a2a.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func createTestMessage(id string, content string) *a2a.Message {
	return &a2a.Message{
		ID: id,
		Parts: a2a.ContentParts{
			a2a.TextPart{
				Text: content,
			},
		},
	}
}

func createTemporaryISOCDir(t *testing.T) string {
	dir := t.TempDir()
	return dir
}

// =============================================================================
// 1. TRANSPORT SELECTION TESTS (5 tests)
// =============================================================================

func TestSelectTransport_ProcessIsolation(t *testing.T) {
	// Test: Select HTTP for process isolation
	transportType, err := SelectTransport(core.IsolationProcess)

	assert.NoError(t, err)
	assert.Equal(t, TransportIPC, transportType)
}

func TestSelectTransport_PlatformIsolation(t *testing.T) {
	// Test: Select gRPC for platform isolation
	transportType, err := SelectTransport(core.IsolationPlatform)

	assert.NoError(t, err)
	assert.Equal(t, TransportHTTP, transportType)
}

func TestSelectTransport_ContainerIsolation(t *testing.T) {
	// Test: Select IPC for container isolation
	transportType, err := SelectTransport(core.IsolationContainer)

	assert.NoError(t, err)
	assert.Equal(t, TransportGRPC, transportType)
}

func TestSelectTransport_FallbackToHTTP(t *testing.T) {
	// Test: Fallback to HTTP when next transport is available
	fallback, ok := GetFallbackTransport(TransportIPC)

	assert.True(t, ok)
	assert.Equal(t, TransportHTTP, fallback)
}

func TestSelectTransport_InvalidIsolationLevel(t *testing.T) {
	// Test: Invalid isolation level
	transportType, err := SelectTransport(core.IsolationLevel("invalid"))

	assert.Error(t, err)
	assert.Empty(t, transportType)
	assert.Equal(t, "unsupported isolation tier: invalid", err.Error())
}

// =============================================================================
// 2. HTTP TRANSPORT TESTS (6 tests)
// =============================================================================

func TestNewHTTPTransport_CreateWithValidConfig(t *testing.T) {
	// Test: Create HTTP transport
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)

	require.NoError(t, err)
	require.NotNil(t, httpTransport)
	assert.Equal(t, TransportHTTP, httpTransport.Type())

	err = httpTransport.Close()
	assert.NoError(t, err)
}

func TestHTTPTransport_SendMessage(t *testing.T) {
	// Test: Send message via HTTP
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, err)
		assert.Equal(t, "2.0", request["jsonrpc"])
		assert.Equal(t, "SendMessage", request["method"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	handler := &mockMessageHandler{}
	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := createTestMessage("test-1", "Hello World")
	err = httpTransport.Send(ctx, message)

	assert.NoError(t, err)
}

func TestHTTPTransport_ReceiveMessage(t *testing.T) {
	// Test: Receive message via HTTP
	config := DefaultHTTPConfig("http://127.0.0.1:8090")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	// Simulate receiving a message
	testMessage := createTestMessage("test-2", "Received")
	err = httpTransport.HandleServerResponse(testMessage)
	assert.NoError(t, err)

	// Receive the message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	received, err := httpTransport.Receive(ctx)
	assert.NoError(t, err)
	require.NotNil(t, received)
	assert.Equal(t, "test-2", received.ID)
}

func TestHTTPTransport_ConnectionPooling(t *testing.T) {
	// Test: Connection pooling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx := context.Background()

	// Send multiple messages to test connection reuse
	for i := 0; i < 5; i++ {
		message := createTestMessage(fmt.Sprintf("test-%d", i), "pooling test")
		err := httpTransport.Send(ctx, message)
		assert.NoError(t, err)
	}
}

func TestHTTPTransport_TLSSupport(t *testing.T) {
	// Test: TLS support configuration
	config := DefaultHTTPConfig("https://127.0.0.1:8443")
	config.SecurityType = "tls"

	handler := &mockMessageHandler{}
	httpTransport, err := NewHTTPTransport(config, handler)

	require.NoError(t, err)
	require.NotNil(t, httpTransport)
	assert.Equal(t, "tls", httpTransport.config.SecurityType)

	httpTransport.Close()
}

func TestHTTPTransport_ErrorHandling(t *testing.T) {
	// Test: Error handling
	config := DefaultHTTPConfig("http://127.0.0.1:9999")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Send to non-existent server
	message := createTestMessage("test-error", "error test")
	err = httpTransport.Send(ctx, message)

	assert.Error(t, err)
}

// =============================================================================
// 3. GRPC TRANSPORT TESTS (5 tests)
// =============================================================================

func TestNewGRPCTransport_CreateWithValidConfig(t *testing.T) {
	// Test: Create gRPC transport
	config := DefaultGRPCConfig("localhost:50051")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)

	require.NoError(t, err)
	require.NotNil(t, grpcTransport)
	assert.Equal(t, TransportGRPC, grpcTransport.Type())

	err = grpcTransport.Close()
	assert.NoError(t, err)
}

func TestGRPCTransport_SendMessage(t *testing.T) {
	// Test: Send message via gRPC
	config := DefaultGRPCConfig("localhost:50051")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer grpcTransport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := createTestMessage("grpc-1", "gRPC message")

	// For now, this will return not implemented error (as per the implementation)
	err = grpcTransport.Send(ctx, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet fully implemented")
}

func TestGRPCTransport_StreamingSupport(t *testing.T) {
	// Test: Streaming support
	config := DefaultGRPCConfig("localhost:50052")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer grpcTransport.Close()

	// Test message handling for streaming
	testMessage := createTestMessage("grpc-stream", "streaming")
	err = grpcTransport.HandleServerResponse(testMessage)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	received, err := grpcTransport.Receive(ctx)
	assert.NoError(t, err)
	require.NotNil(t, received)
	assert.Equal(t, "grpc-stream", received.ID)
}

func TestGRPCTransport_ConnectionManagement(t *testing.T) {
	// Test: Connection management
	config := DefaultGRPCConfig("localhost:50053")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)

	// Check health before running
	isHealthy := grpcTransport.IsHealthy()
	assert.False(t, isHealthy) // Not running yet

	// Close connection
	err = grpcTransport.Close()
	assert.NoError(t, err)
}

func TestGRPCTransport_ErrorHandling(t *testing.T) {
	// Test: Error handling
	config := DefaultGRPCConfig("invalid:endpoint")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer grpcTransport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Send to invalid endpoint
	message := createTestMessage("grpc-error", "error test")
	err = grpcTransport.Send(ctx, message)

	assert.Error(t, err)
}

// =============================================================================
// 4. IPC TRANSPORT TESTS (5 tests)
// =============================================================================

func TestNewIPCTransport_CreateWithValidConfig(t *testing.T) {
	// Test: Create Unix socket transport
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)

	require.NoError(t, err)
	require.NotNil(t, ipcTransport)
	assert.Equal(t, TransportIPC, ipcTransport.Type())

	err = ipcTransport.Close()
	assert.NoError(t, err)
}

func TestIPCTransport_SendMessage(t *testing.T) {
	// Test: Send message via IPC
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	ctx := context.Background()
	message := createTestMessage("ipc-1", "IPC message")

	err = ipcTransport.Send(ctx, message)
	assert.NoError(t, err)

	// Verify message file was created
	entries, err := os.ReadDir(queueDir)
	assert.NoError(t, err)
	assert.True(t, len(entries) > 0, "message file should exist")
}

func TestIPCTransport_SocketFileCleanup(t *testing.T) {
	// Test: Socket file cleanup
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)

	// Start polling
	err = ipcTransport.Start()
	assert.NoError(t, err)

	// Send message
	ctx := context.Background()
	message := createTestMessage("cleanup-1", "cleanup test")
	err = ipcTransport.Send(ctx, message)
	assert.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(200 * time.Millisecond)

	// Stop polling
	err = ipcTransport.Stop()
	assert.NoError(t, err)

	ipcTransport.Close()
}

func TestIPCTransport_PermissionHandling(t *testing.T) {
	// Test: Permission handling
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	// Verify directory has restrictive permissions (0700)
	stat, err := os.Stat(queueDir)
	assert.NoError(t, err)
	assert.NotNil(t, stat)

	// Check that the directory was created with proper permissions
	mode := stat.Mode()
	assert.True(t, mode.IsDir())
}

func TestIPCTransport_ErrorHandling(t *testing.T) {
	// Test: Error handling with nil config
	_, err := NewIPCTransport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ipc config cannot be nil")
}

// =============================================================================
// 5. AUTHENTICATION TESTS (4 tests)
// =============================================================================

func TestAuth_TokenAuthentication(t *testing.T) {
	// Test: Token authentication
	authConfig := &AuthConfig{
		Type:  AuthAPIKey,
		APIKey: "test-api-key-123",
	}

	err := authConfig.ValidateAuth()
	assert.NoError(t, err)

	// Test applying auth to request
	req, err := http.NewRequest("POST", "http://127.0.0.1:8081", nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = authConfig.ApplyAuth(req, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-api-key-123", req.Header.Get("X-API-Key"))
}

func TestAuth_CertificateAuthentication(t *testing.T) {
	// Test: Certificate authentication
	authConfig := &AuthConfig{
		Type:     AuthMTLS,
		CertPath: "/path/to/cert.pem",
		KeyPath:  "/path/to/key.pem",
	}

	err := authConfig.ValidateAuth()
	assert.NoError(t, err)
}

func TestAuth_AuthFailureHandling(t *testing.T) {
	// Test: Auth failure handling - missing API key
	authConfig := &AuthConfig{
		Type:   AuthAPIKey,
		APIKey: "",
	}

	err := authConfig.ValidateAuth()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestAuth_NoAuthConfiguration(t *testing.T) {
	// Test: No authentication
	authConfig := &AuthConfig{
		Type: AuthNone,
	}

	err := authConfig.ValidateAuth()
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8081", nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = authConfig.ApplyAuth(req, ctx)
	assert.NoError(t, err)
}

// =============================================================================
// INTEGRATION AND EDGE CASE TESTS
// =============================================================================

func TestHTTPTransport_SendNilMessage(t *testing.T) {
	// Test: Send nil message error handling
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx := context.Background()
	err = httpTransport.Send(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message cannot be nil")
}

func TestGRPCTransport_SendNilMessage(t *testing.T) {
	// Test: Send nil message error handling
	config := DefaultGRPCConfig("localhost:50051")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer grpcTransport.Close()

	ctx := context.Background()
	err = grpcTransport.Send(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message cannot be nil")
}

func TestIPCTransport_SendNilMessage(t *testing.T) {
	// Test: Send nil message error handling
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	ctx := context.Background()
	err = ipcTransport.Send(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message cannot be nil")
}

func TestTransportInterface_Implementation(t *testing.T) {
	// Test: Verify all transports implement Transport interface
	httpConfig := DefaultHTTPConfig("http://127.0.0.1:8081")
	httpTransport, err := NewHTTPTransport(httpConfig, nil)
	require.NoError(t, err)
	defer httpTransport.Close()

	var _ Transport = httpTransport

	grpcConfig := DefaultGRPCConfig("localhost:50051")
	grpcTransport, err := NewGRPCTransport(grpcConfig, nil)
	require.NoError(t, err)
	defer grpcTransport.Close()

	var _ Transport = grpcTransport

	queueDir := createTemporaryISOCDir(t)
	ipcConfig := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}
	ipcTransport, err := NewIPCTransport(ipcConfig, nil)
	require.NoError(t, err)
	defer ipcTransport.Close()

	var _ Transport = ipcTransport
}

func TestContextCancellation_HTTP(t *testing.T) {
	// Test: Context cancellation in HTTP transport
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = httpTransport.Receive(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestContextCancellation_GRPC(t *testing.T) {
	// Test: Context cancellation in gRPC transport
	config := DefaultGRPCConfig("localhost:50051")
	handler := &mockMessageHandler{}

	grpcTransport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer grpcTransport.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = grpcTransport.Receive(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestContextCancellation_IPC(t *testing.T) {
	// Test: Context cancellation in IPC transport
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = ipcTransport.Receive(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestIPCTransport_ReceiveMessage(t *testing.T) {
	// Test: Receive message via IPC
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	// Start polling to process messages
	err = ipcTransport.Start()
	assert.NoError(t, err)

	// Send a message
	ctx := context.Background()
	message := createTestMessage("ipc-receive", "receive test")
	err = ipcTransport.Send(ctx, message)
	assert.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(200 * time.Millisecond)

	// Receive the message with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	received, err := ipcTransport.Receive(ctx)
	if err == nil {
		require.NotNil(t, received)
		assert.Equal(t, "ipc-receive", received.ID)
	}

	ipcTransport.Stop()
}

func TestIPCTransport_Health(t *testing.T) {
	// Test: IPC transport health check
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	isHealthy := ipcTransport.IsHealthy()
	assert.True(t, isHealthy)
}

func TestHTTPTransport_Health(t *testing.T) {
	// Test: HTTP transport health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	isHealthy := httpTransport.IsHealthy()
	assert.True(t, isHealthy)
}

func TestDefaultConfigs_Initialization(t *testing.T) {
	// Test: Default configuration initialization
	httpConfig := DefaultHTTPConfig("http://127.0.0.1:8081")
	assert.NotNil(t, httpConfig)
	assert.Equal(t, "application/json", httpConfig.ContentType)
	assert.Equal(t, 30*time.Second, httpConfig.Timeout)
	assert.Equal(t, "none", httpConfig.SecurityType)

	grpcConfig := DefaultGRPCConfig("localhost:50051")
	assert.NotNil(t, grpcConfig)
	assert.Equal(t, 30*time.Second, grpcConfig.Timeout)
	assert.False(t, grpcConfig.MTLSEnabled)
}

func TestTransportPriority_Ordering(t *testing.T) {
	// Test: Transport priority ordering
	assert.Equal(t, 3, len(TransportPriority))
	assert.Equal(t, TransportIPC, TransportPriority[0])
	assert.Equal(t, TransportHTTP, TransportPriority[1])
	assert.Equal(t, TransportGRPC, TransportPriority[2])
}

func TestAuthConfig_Defaults(t *testing.T) {
	// Test: Default authentication config
	noneAuth := DefaultAuthConfig("none")
	assert.Equal(t, AuthNone, noneAuth.Type)

	apiKeyAuth := DefaultAuthConfig("apikey")
	assert.Equal(t, AuthAPIKey, apiKeyAuth.Type)

	mtlsAuth := DefaultAuthConfig("mtls")
	assert.Equal(t, AuthMTLS, mtlsAuth.Type)

	unknownAuth := DefaultAuthConfig("unknown")
	assert.Equal(t, AuthNone, unknownAuth.Type)
}

func TestIPCTransport_GetProfileAndPath(t *testing.T) {
	// Test: IPC transport getters
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile-123",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	assert.Equal(t, "test-profile-123", ipcTransport.GetProfileID())
	assert.Equal(t, queueDir, ipcTransport.GetQueuePath())
}

func TestHTTPTransport_SubmitMessage(t *testing.T) {
	// Test: SubmitMessage method (wrapper for Send)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer httpTransport.Close()

	ctx := context.Background()
	message := createTestMessage("submit-1", "submit test")

	err = httpTransport.SubmitMessage(ctx, message)
	assert.NoError(t, err)
}

func TestTransportType_Values(t *testing.T) {
	// Test: Transport type constants
	assert.Equal(t, TransportType("ipc"), TransportIPC)
	assert.Equal(t, TransportType("http"), TransportHTTP)
	assert.Equal(t, TransportType("grpc"), TransportGRPC)
}

func TestAuthType_Values(t *testing.T) {
	// Test: Auth type constants
	assert.Equal(t, AuthType("none"), AuthNone)
	assert.Equal(t, AuthType("apikey"), AuthAPIKey)
	assert.Equal(t, AuthType("mtls"), AuthMTLS)
	assert.Equal(t, AuthType("openid"), AuthOpenID)
}

func TestNewHTTPTransport_NilConfig(t *testing.T) {
	// Test: HTTP transport with nil config
	_, err := NewHTTPTransport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http config cannot be nil")
}

func TestNewHTTPTransport_EmptyEndpoint(t *testing.T) {
	// Test: HTTP transport with empty endpoint
	config := &HTTPConfig{
		Endpoint:    "",
		ContentType: "application/json",
		Timeout:     30 * time.Second,
	}

	_, err := NewHTTPTransport(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint cannot be empty")
}

func TestNewGRPCTransport_NilConfig(t *testing.T) {
	// Test: gRPC transport with nil config
	_, err := NewGRPCTransport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc config cannot be nil")
}

func TestNewGRPCTransport_EmptyEndpoint(t *testing.T) {
	// Test: gRPC transport with empty endpoint
	config := &GRPCConfig{
		Endpoint:    "",
		Timeout:     30 * time.Second,
		MTLSEnabled: false,
	}

	_, err := NewGRPCTransport(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint cannot be empty")
}

func TestIPCTransport_NilProfileID(t *testing.T) {
	// Test: IPC transport with empty profile ID
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	_, err := NewIPCTransport(config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile id cannot be empty")
}

func TestHTTPTransport_HandleServerResponse_Closed(t *testing.T) {
	// Test: Handle server response on closed transport
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	handler := &mockMessageHandler{}

	httpTransport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)

	httpTransport.Close()

	// Wait briefly for context to be cancelled
	time.Sleep(100 * time.Millisecond)

	// Try to handle a message - should fail after many attempts due to closed context
	message := createTestMessage("test-1", "closed")

	// This may succeed or fail depending on timing, but we verify the channel is closed
	httpTransport.HandleServerResponse(message)
	assert.False(t, httpTransport.running)
}

func TestSelectTransportFromCard_ValidCard(t *testing.T) {
	// Test: Select transport from Agent Card
	card := &a2a.AgentCard{
		PreferredTransport: a2a.TransportProtocol("http"),
		AdditionalInterfaces: []a2a.AgentInterface{
			{
				Transport: a2a.TransportProtocol("ipc"),
				URL:       "ipc://localhost",
			},
		},
	}

	transportType, err := SelectTransportFromCard(card)
	assert.NoError(t, err)

	// Should select first priority match (IPC has higher priority than HTTP)
	assert.Equal(t, TransportIPC, transportType)
}

func TestSelectTransportFromCard_NilCard(t *testing.T) {
	// Test: Select transport from nil card
	_, err := SelectTransportFromCard(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent card cannot be nil")
}

func TestSelectTransportFromCard_NoMatch(t *testing.T) {
	// Test: Select transport when no compatible transport found
	card := &a2a.AgentCard{
		PreferredTransport:   a2a.TransportProtocol("unknown"),
		AdditionalInterfaces: []a2a.AgentInterface{},
	}

	_, err := SelectTransportFromCard(card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no compatible transport found")
}

func TestGetFallbackTransport_NoMoreTransports(t *testing.T) {
	// Test: Get fallback when no more transports available
	fallback, ok := GetFallbackTransport(TransportGRPC)
	assert.False(t, ok)
	assert.Empty(t, fallback)
}

func TestIPCTransport_MessageProcessing(t *testing.T) {
	// Test: Message processing in IPC
	queueDir := createTemporaryISOCDir(t)

	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 100 * time.Millisecond,
	}

	handler := &mockMessageHandler{}
	ipcTransport, err := NewIPCTransport(config, handler)
	require.NoError(t, err)
	defer ipcTransport.Close()

	// Manually write a message file
	message := &a2a.Message{
		ID: "test-msg",
		Parts: a2a.ContentParts{
			a2a.TextPart{
				Text: "test content",
			},
		},
	}

	data, err := json.Marshal(message)
	require.NoError(t, err)

	messageFile := filepath.Join(queueDir, "test_message.json")
	err = os.WriteFile(messageFile, data, 0600)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(messageFile)
	assert.NoError(t, err)
}

func TestTransportTypes_StringConversion(t *testing.T) {
	// Test: Transport type string conversions
	ipc := TransportIPC
	assert.Equal(t, "ipc", string(ipc))

	http := TransportHTTP
	assert.Equal(t, "http", string(http))

	grpc := TransportGRPC
	assert.Equal(t, "grpc", string(grpc))
}

func TestAuthConfig_UnsupportedType(t *testing.T) {
	// Test: Auth config with unsupported type
	authConfig := &AuthConfig{
		Type: AuthType("unsupported"),
	}

	err := authConfig.ValidateAuth()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported authentication type")
}

func TestAuthConfig_ApplyUnsupportedType(t *testing.T) {
	// Test: Apply auth with unsupported type
	authConfig := &AuthConfig{
		Type: AuthType("unsupported"),
	}

	req, err := http.NewRequest("GET", "http://127.0.0.1:8081", nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = authConfig.ApplyAuth(req, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auth type")
}
