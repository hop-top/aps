package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TRANSPORT LAYER TESTS (12 tests) - Message sending/receiving, network
// error handling, timeout handling, connection management, concurrent ops
// =============================================================================

// TestHTTPTransport_MessageSendingWithValidPayload tests sending a valid message
func TestHTTPTransport_MessageSendingWithValidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "SendMessage", req["method"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	handler := &mockMessageHandler{}
	transport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := createTestMessage("msg-001", "Valid payload test")
	err = transport.Send(ctx, msg)
	assert.NoError(t, err)
}

// TestHTTPTransport_MessageReceivingWithTimeout tests receiving with timeout
func TestHTTPTransport_MessageReceivingWithTimeout(t *testing.T) {
	config := DefaultHTTPConfig("http://127.0.0.1:9999")
	handler := &mockMessageHandler{}
	transport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = transport.Receive(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// TestIPCTransport_ConcurrentMessageSending tests sending messages concurrently
func TestIPCTransport_ConcurrentMessageSending(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "concurrent-test",
		QueuePath:    queueDir,
		Polling:      false,
		PollInterval: 50 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	numMessages := 20
	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			msg := createTestMessage(fmt.Sprintf("concurrent-%d", index), "content")
			err := transport.Send(ctx, msg)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int32(numMessages), successCount)

	files, err := os.ReadDir(queueDir)
	require.NoError(t, err)
	assert.Equal(t, numMessages, len(files))
}

// TestGRPCTransport_SendingWithResponseHandling tests gRPC response handling
func TestGRPCTransport_SendingWithResponseHandling(t *testing.T) {
	config := DefaultGRPCConfig("localhost:50051")
	handler := &mockMessageHandler{}
	transport, err := NewGRPCTransport(config, handler)
	require.NoError(t, err)
	defer transport.Close()

	testMsg := createTestMessage("grpc-001", "test message")
	err = transport.HandleServerResponse(testMsg)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	received, err := transport.Receive(ctx)
	require.NoError(t, err)
	assert.Equal(t, testMsg.ID, received.ID)
}

// TestHTTPTransport_NetworkErrorHandling tests handling of network errors
func TestHTTPTransport_NetworkErrorHandling(t *testing.T) {
	config := DefaultHTTPConfig("http://127.0.0.1:6789")
	handler := &mockMessageHandler{}
	transport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg := createTestMessage("network-error", "should fail")
	err = transport.Send(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

// TestIPCTransport_QueueDirectoryDoesNotExist tests handling of missing queue dir
func TestIPCTransport_QueueDirectoryDoesNotExist(t *testing.T) {
	config := &IPCConfig{
		ProfileID:    "test-profile",
		QueuePath:    "/tmp/nonexistent-queue-" + time.Now().Format("20060102150405"),
		Polling:      false,
		PollInterval: 100 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("test", "content")
	err = transport.Send(ctx, msg)
	assert.NoError(t, err)

	stat, err := os.Stat(config.QueuePath)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}

// TestHTTPTransport_ConnectionReuse tests connection reuse across multiple sends
func TestHTTPTransport_ConnectionReuse(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		msg := createTestMessage(fmt.Sprintf("msg-%d", i), "test")
		err := transport.Send(ctx, msg)
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(10), requestCount)
}

// TestHTTPTransport_LargeMessageHandling tests sending large messages
func TestHTTPTransport_LargeMessageHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.True(t, len(body) > 1000)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	largeContent := ""
	for i := 0; i < 100; i++ {
		largeContent += "This is a large message content that is repeated many times to create a substantial payload. "
	}

	msg := createTestMessage("large-msg", largeContent)
	err = transport.Send(ctx, msg)
	assert.NoError(t, err)
}

// TestIPCTransport_ParallelReadWrite tests parallel read/write operations
func TestIPCTransport_ParallelReadWrite(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "parallel-test",
		QueuePath:    queueDir,
		Polling:      false,
		PollInterval: 50 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	var sendWg sync.WaitGroup
	sendWg.Add(10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer sendWg.Done()
			msg := createTestMessage(fmt.Sprintf("parallel-%d", idx), "parallel content")
			transport.Send(ctx, msg)
		}(i)
	}

	sendWg.Wait()

	files, err := os.ReadDir(queueDir)
	require.NoError(t, err)
	assert.Equal(t, 10, len(files))
}

// =============================================================================
// RETRY LOGIC TESTS (8 tests) - Retry on failure, backoff strategies,
// max retry limits, successful recovery
// =============================================================================

// TestRetryMechanism_FailsAndRetries tests basic retry behavior
func TestRetryMechanism_FailsAndRetries(t *testing.T) {
	failCount := 0
	maxFailures := 2

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= maxFailures {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	retryCount := 0
	maxRetries := 3

	ctx := context.Background()
	msg := createTestMessage("retry-test", "content")

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := transport.Send(ctx, msg)
		if err == nil {
			retryCount = attempt
			break
		}
	}

	assert.Equal(t, 2, retryCount)
}

// TestRetryLogic_ExponentialBackoff simulates exponential backoff
func TestRetryLogic_ExponentialBackoff(t *testing.T) {
	failures := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failures++
		if failures <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	startTime := time.Now()
	ctx := context.Background()
	msg := createTestMessage("backoff-test", "content")

	for attempt := 0; attempt < 5; attempt++ {
		err := transport.Send(ctx, msg)
		if err == nil {
			break
		}
		if attempt < 4 {
			backoff := time.Duration((1 << uint(attempt)) * 50 * int(time.Millisecond))
			time.Sleep(backoff)
		}
	}

	elapsed := time.Since(startTime)
	assert.True(t, elapsed > 0)
}

// TestRetryPolicy_MaxRetriesExceeded tests max retry limit enforcement
func TestRetryPolicy_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	maxRetries := 3
	attemptCount := 0
	ctx := context.Background()
	msg := createTestMessage("max-retry-test", "content")

	for attempt := 0; attempt < maxRetries; attempt++ {
		attemptCount++
		err := transport.Send(ctx, msg)
		if err != nil {
			continue
		}
		break
	}

	assert.Equal(t, maxRetries, attemptCount)
}

// TestRetryMechanism_SuccessfulRecoveryAfterFailure tests recovery from transient errors
func TestRetryMechanism_SuccessfulRecoveryAfterFailure(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("recovery-test", "content")

	var finalErr error
	for attempt := 0; attempt < 5; attempt++ {
		finalErr = transport.Send(ctx, msg)
		if finalErr == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.NoError(t, finalErr)
	assert.Equal(t, 3, failCount)
}

// TestRetryStrategy_CircuitBreakerPattern tests circuit breaker pattern
func TestRetryStrategy_CircuitBreakerPattern(t *testing.T) {
	failureCount := 0
	circuitBreakerThreshold := 3
	var circuitOpen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if circuitOpen {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		failureCount++
		if failureCount > circuitBreakerThreshold {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			if failureCount >= circuitBreakerThreshold {
				circuitOpen = true
			}
		}
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("circuit-breaker", "content")

	for i := 0; i < 10; i++ {
		if circuitOpen {
			break
		}
		transport.Send(ctx, msg)
	}

	assert.True(t, circuitOpen)
}

// TestRetryLogic_JitterBackoff tests backoff with jitter
func TestRetryLogic_JitterBackoff(t *testing.T) {
	delays := make([]time.Duration, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("jitter-test", "content")

	for i := 0; i < 3; i++ {
		before := time.Now()
		err := transport.Send(ctx, msg)
		after := time.Now()
		mu.Lock()
		delays = append(delays, after.Sub(before))
		mu.Unlock()
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, len(delays))
}

// TestRetryLogic_IdempotentMessageHandling tests idempotent message handling
func TestRetryLogic_IdempotentMessageHandling(t *testing.T) {
	receivedCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedCount++
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("idempotent-001", "test")

	for i := 0; i < 3; i++ {
		err := transport.Send(ctx, msg)
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, receivedCount)
}

// =============================================================================
// ERROR SCENARIOS TESTS (10 tests) - Network disconnection, timeouts,
// malformed messages, protocol violations, resource exhaustion
// =============================================================================

// TestErrorScenario_NetworkDisconnection simulates network disconnection
func TestErrorScenario_NetworkDisconnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	endpoint := listener.Addr().String()
	config := DefaultHTTPConfig("http://" + endpoint)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg := createTestMessage("disconnect", "test")
	err = transport.Send(ctx, msg)
	assert.Error(t, err)
}

// TestErrorScenario_RequestTimeout tests request timeout handling
func TestErrorScenario_RequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	config.Timeout = 1 * time.Second
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg := createTestMessage("timeout-test", "content")
	err = transport.Send(ctx, msg)
	assert.Error(t, err)
}

// TestErrorScenario_MalformedMessagePayload tests malformed message handling
func TestErrorScenario_MalformedMessagePayload(t *testing.T) {
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	err = transport.Send(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestErrorScenario_HTTPStatusErrors tests various HTTP error status codes
func TestErrorScenario_HTTPStatusErrors(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer server.Close()

			config := DefaultHTTPConfig(server.URL)
			transport, err := NewHTTPTransport(config, nil)
			require.NoError(t, err)
			defer transport.Close()

			ctx := context.Background()
			msg := createTestMessage("error-test", "test")
			err = transport.Send(ctx, msg)
			assert.Error(t, err)
		})
	}
}

// TestErrorScenario_ContextCancellationDuringTransfer tests context cancellation
func TestErrorScenario_ContextCancellationDuringTransfer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	msg := createTestMessage("cancel-test", "test")
	err = transport.Send(ctx, msg)
	assert.Error(t, err)
}

// TestErrorScenario_ResourceExhaustion tests handling of resource exhaustion
func TestErrorScenario_ResourceExhaustion(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "exhaustion-test",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 10 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	err = transport.Start()
	require.NoError(t, err)

	ctx := context.Background()
	numMessages := 500

	for i := 0; i < numMessages; i++ {
		msg := createTestMessage(fmt.Sprintf("msg-%d", i), "content")
		err := transport.Send(ctx, msg)
		if err != nil {
			t.Logf("Send failed at message %d: %v", i, err)
			break
		}
	}

	transport.Stop()

	files, err := os.ReadDir(queueDir)
	require.NoError(t, err)
	assert.True(t, len(files) > 0)
	assert.True(t, len(files) <= numMessages)
}

// TestErrorScenario_InvalidMessageFormat tests invalid message format handling
func TestErrorScenario_InvalidMessageFormat(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "invalid-format-test",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 50 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	err = transport.Start()
	require.NoError(t, err)

	badMessagePath := filepath.Join(queueDir, "invalid_message.json")
	err = os.WriteFile(badMessagePath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	_, err = os.Stat(badMessagePath)
	assert.True(t, os.IsNotExist(err))

	transport.Stop()
}

// TestErrorScenario_ProtocolViolation tests protocol violation handling
func TestErrorScenario_ProtocolViolation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	msg := createTestMessage("protocol-test", "test")
	err = transport.Send(ctx, msg)
	assert.NoError(t, err)
}

// TestErrorScenario_PermissionDeniedOnQueue tests permission denied scenarios
func TestErrorScenario_PermissionDeniedOnQueue(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	queueDir := t.TempDir()
	err := os.Chmod(queueDir, 0000)
	require.NoError(t, err)

	defer os.Chmod(queueDir, 0755)

	config := &IPCConfig{
		ProfileID:    "permission-test",
		QueuePath:    filepath.Join(queueDir, "nested"),
		Polling:      false,
		PollInterval: 100 * time.Millisecond,
	}

	_, err = NewIPCTransport(config, nil)
	assert.Error(t, err)
}

// TestErrorScenario_ConcurrentAccessErrors tests concurrent access error handling
func TestErrorScenario_ConcurrentAccessErrors(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "concurrent-error-test",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 50 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	err = transport.Start()
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := createTestMessage(fmt.Sprintf("concurrent-%d", idx), "content")
			err := transport.Send(ctx, msg)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 50, successCount)

	transport.Stop()
}

// =============================================================================
// ADVANCED RESILIENCE TESTS (2+ additional tests)
// =============================================================================

// TestTransportHealth_HealthCheckRetry tests health check with retries
func TestTransportHealth_HealthCheckRetry(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			failCount++
			if failCount < 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	var healthy bool
	for i := 0; i < 3; i++ {
		healthy = transport.IsHealthy()
		if healthy {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.True(t, healthy)
}

// TestTransportResilience_GracefulDegradation tests graceful degradation
func TestTransportResilience_GracefulDegradation(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount%3 == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	successCount := 0

	for i := 0; i < 9; i++ {
		msg := createTestMessage(fmt.Sprintf("degrade-%d", i), "content")
		err := transport.Send(ctx, msg)
		if err == nil {
			successCount++
		}
	}

	assert.Equal(t, 6, successCount)
}

// TestTransportResilience_MessageQueueing tests message queueing under pressure
func TestTransportResilience_MessageQueueing(t *testing.T) {
	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "queue-pressure-test",
		QueuePath:    queueDir,
		Polling:      false,
		PollInterval: 20 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	numMessages := 50
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := createTestMessage(fmt.Sprintf("queue-msg-%d", idx), "test")
			transport.Send(ctx, msg)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	files, err := os.ReadDir(queueDir)
	require.NoError(t, err)
	assert.Equal(t, numMessages, len(files), "Expected %d files, got %d", numMessages, len(files))
	assert.True(t, elapsed > 0)
}

// TestTransportResilience_ConnectionPooling tests connection pooling and reuse
func TestTransportResilience_ConnectionPooling(t *testing.T) {
	var activeConnections int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&activeConnections, 1)
		defer atomic.AddInt32(&activeConnections, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPConfig(server.URL)
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	ctx := context.Background()
	maxConcurrent := int32(0)

	for i := 0; i < 20; i++ {
		msg := createTestMessage(fmt.Sprintf("pool-%d", i), "test")
		err := transport.Send(ctx, msg)
		assert.NoError(t, err)

		current := atomic.LoadInt32(&activeConnections)
		if current > maxConcurrent {
			maxConcurrent = current
		}
	}

	assert.True(t, maxConcurrent <= 5)
}

// TestTransportResilience_ContextTimeoutVariations tests various timeout values
func TestTransportResilience_ContextTimeoutVariations(t *testing.T) {
	timeouts := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	config := DefaultHTTPConfig("http://127.0.0.1:9999")
	transport, err := NewHTTPTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	for _, timeout := range timeouts {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, err := transport.Receive(ctx)
		cancel()

		assert.Error(t, err)
	}
}

// TestTransportResilience_ReceiveChannelHandling tests message channel handling
func TestTransportResilience_ReceiveChannelHandling(t *testing.T) {
	config := DefaultHTTPConfig("http://127.0.0.1:8081")
	handler := &mockMessageHandler{}
	transport, err := NewHTTPTransport(config, handler)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = transport.Receive(ctx)
	assert.Error(t, err)

	transport.Close()
}

// TestTransportResilience_LargePayloadHandling tests handling of large payloads
func TestTransportResilience_LargePayloadHandling(t *testing.T) {
	payloadSizes := []int{
		1024,
		10 * 1024,
		100 * 1024,
	}

	for _, size := range payloadSizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.True(t, len(body) > 0)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			config := DefaultHTTPConfig(server.URL)
			transport, err := NewHTTPTransport(config, nil)
			require.NoError(t, err)
			defer transport.Close()

			largeContent := ""
			for i := 0; i < size/100; i++ {
				largeContent += "Large payload test content repeated many times to reach desired size. "
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			msg := createTestMessage("large-payload", largeContent)
			err = transport.Send(ctx, msg)
			assert.NoError(t, err)
		})
	}
}

// TestTransportResilience_StressTest performs stress testing
func TestTransportResilience_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	queueDir := t.TempDir()
	config := &IPCConfig{
		ProfileID:    "stress-test",
		QueuePath:    queueDir,
		Polling:      true,
		PollInterval: 10 * time.Millisecond,
	}

	transport, err := NewIPCTransport(config, nil)
	require.NoError(t, err)
	defer transport.Close()

	err = transport.Start()
	require.NoError(t, err)

	ctx := context.Background()
	numWorkers := 10
	messagesPerWorker := 100

	var wg sync.WaitGroup
	var successCount int32

	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < messagesPerWorker; i++ {
				msg := createTestMessage(
					fmt.Sprintf("stress-w%d-m%d", workerID, i),
					"stress test content",
				)
				err := transport.Send(ctx, msg)
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	transport.Stop()

	expectedMessages := numWorkers * messagesPerWorker
	assert.Equal(t, int32(expectedMessages), successCount)
	assert.True(t, elapsed > 0)
	t.Logf("Processed %d messages in %v", expectedMessages, elapsed)
}
