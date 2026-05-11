package mobile_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hop.top/aps/internal/core/adapter/mobile"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServerEnv struct {
	server   *mobile.AdapterServer
	registry *mobile.Registry
	tokenMgr *mobile.TokenManager
	addr     string
	cancel   context.CancelFunc
}

func setupTestServer(t *testing.T, opts ...mobile.ServerOption) *testServerEnv {
	t.Helper()
	dir := t.TempDir()

	registry, err := mobile.NewRegistry(filepath.Join(dir, "registry"))
	require.NoError(t, err)

	tokenMgr, err := mobile.NewTokenManager("test-profile", filepath.Join(dir, "keys"))
	require.NoError(t, err)

	allOpts := append([]mobile.ServerOption{mobile.WithMaxAdapters(10)}, opts...)
	server := mobile.NewAdapterServer("test-profile", registry, tokenMgr, allOpts...)

	ctx, cancel := context.WithCancel(context.Background())

	err = server.Start(ctx, ":0") // random port
	require.NoError(t, err)

	t.Cleanup(func() {
		cancel()
		server.Stop()
	})

	return &testServerEnv{
		server:   server,
		registry: registry,
		tokenMgr: tokenMgr,
		addr:     server.GetAddress(),
		cancel:   cancel,
	}
}

func (e *testServerEnv) pairURL() string {
	return fmt.Sprintf("http://%s/aps/adapter/test-profile/pair", e.addr)
}

func (e *testServerEnv) healthURL() string {
	return fmt.Sprintf("http://%s/aps/adapter/test-profile/health", e.addr)
}

func (e *testServerEnv) wsURL() string {
	return fmt.Sprintf("ws://%s/aps/adapter/test-profile/ws", e.addr)
}

func TestServerLifecycle(t *testing.T) {
	t.Run("starts and reports running", func(t *testing.T) {
		env := setupTestServer(t)
		assert.Equal(t, "running", env.server.Status())
		assert.NotEmpty(t, env.server.GetAddress())
	})

	t.Run("stops cleanly", func(t *testing.T) {
		dir := t.TempDir()
		registry, _ := mobile.NewRegistry(filepath.Join(dir, "reg"))
		tokenMgr, _ := mobile.NewTokenManager("p", filepath.Join(dir, "keys"))
		server := mobile.NewAdapterServer("p", registry, tokenMgr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := server.Start(ctx, ":0")
		require.NoError(t, err)
		assert.Equal(t, "running", server.Status())

		err = server.Stop()
		require.NoError(t, err)
		// Status should transition to stopped
		assert.Equal(t, "stopped", server.Status())
	})
}

func TestHealthEndpoint(t *testing.T) {
	env := setupTestServer(t)

	resp, err := http.Get(env.healthURL())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "running", body["status"])
	assert.Equal(t, "test-profile", body["profile"])
}

func TestPairingEndpoint(t *testing.T) {
	t.Run("valid pairing code returns token", func(t *testing.T) {
		env := setupTestServer(t)
		env.server.RegisterPairingCode("TEST-ABC-123", []string{"run:stateless"}, 15*time.Minute)

		reqBody := `{"pairing_code":"TEST-ABC-123","device_name":"Test Phone","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var pairResp mobile.PairingResponse
		err = json.NewDecoder(resp.Body).Decode(&pairResp)
		require.NoError(t, err)

		assert.NotEmpty(t, pairResp.AdapterID)
		assert.NotEmpty(t, pairResp.Token)
		assert.True(t, strings.HasPrefix(pairResp.WSEndpoint, "ws://"), "ws_endpoint should match non-TLS server")
		assert.Equal(t, "test-profile", pairResp.ProfileID)
		assert.Equal(t, "active", pairResp.Status)
	})

	t.Run("invalid pairing code returns 401", func(t *testing.T) {
		env := setupTestServer(t)

		reqBody := `{"pairing_code":"WRONG-CODE","device_name":"Phone","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("expired pairing code returns 410", func(t *testing.T) {
		env := setupTestServer(t)
		// Register with zero-duration expiry (immediately expired)
		env.server.RegisterPairingCode("EXPIRED-CODE", nil, 0)

		time.Sleep(10 * time.Millisecond)

		reqBody := `{"pairing_code":"EXPIRED-CODE","device_name":"Phone","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusGone, resp.StatusCode)
	})

	t.Run("used pairing code returns 409", func(t *testing.T) {
		env := setupTestServer(t)
		env.server.RegisterPairingCode("ONCE-ONLY", []string{"run:stateless"}, 15*time.Minute)

		reqBody := `{"pairing_code":"ONCE-ONLY","device_name":"Phone","device_os":"iOS"}`

		// First use succeeds
		resp1, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		resp1.Body.Close()
		assert.Equal(t, http.StatusCreated, resp1.StatusCode)

		// Second use fails
		resp2, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		resp2.Body.Close()
		assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	})

	t.Run("approval required sets pending status", func(t *testing.T) {
		env := setupTestServer(t, mobile.WithApprovalRequired(true))
		env.server.RegisterPairingCode("PENDING-CODE", []string{"run:stateless"}, 15*time.Minute)

		reqBody := `{"pairing_code":"PENDING-CODE","device_name":"Phone","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var pairResp mobile.PairingResponse
		json.NewDecoder(resp.Body).Decode(&pairResp)
		assert.Equal(t, "pending", pairResp.Status)
	})

	t.Run("max adapters enforced", func(t *testing.T) {
		env := setupTestServer(t, mobile.WithMaxAdapters(1))
		env.server.RegisterPairingCode("CODE-1", []string{"run:stateless"}, 15*time.Minute)
		env.server.RegisterPairingCode("CODE-2", []string{"run:stateless"}, 15*time.Minute)

		// First pairing succeeds
		reqBody := `{"pairing_code":"CODE-1","device_name":"Phone 1","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Second pairing fails (max reached)
		reqBody = `{"pairing_code":"CODE-2","device_name":"Phone 2","device_os":"Android"}`
		resp, err = http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestWebSocketEndpoint(t *testing.T) {
	t.Run("connects with valid token", func(t *testing.T) {
		env := setupTestServer(t)
		env.server.RegisterPairingCode("WS-CODE", []string{"run:stateless"}, 15*time.Minute)

		// Pair first
		reqBody := `{"pairing_code":"WS-CODE","device_name":"WS Phone","device_os":"iOS"}`
		resp, err := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)

		var pairResp mobile.PairingResponse
		json.NewDecoder(resp.Body).Decode(&pairResp)
		resp.Body.Close()

		// Connect via WebSocket
		header := http.Header{}
		header.Set("Authorization", "Bearer "+pairResp.Token)

		conn, wsResp, err := websocket.DefaultDialer.Dial(env.wsURL(), header)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, wsResp.StatusCode)

		// Should receive connection ACK
		var ack mobile.WSMessage
		err = conn.ReadJSON(&ack)
		require.NoError(t, err)
		assert.Equal(t, "status", ack.Type)

		assert.Equal(t, 1, env.server.ActiveConnections())
	})

	t.Run("rejects connection without token", func(t *testing.T) {
		env := setupTestServer(t)

		_, resp, err := websocket.DefaultDialer.Dial(env.wsURL(), nil)
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	})

	t.Run("rejects connection with invalid token", func(t *testing.T) {
		env := setupTestServer(t)

		header := http.Header{}
		header.Set("Authorization", "Bearer invalid-token-here")

		_, resp, err := websocket.DefaultDialer.Dial(env.wsURL(), header)
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

func TestWebSocketMessaging(t *testing.T) {
	t.Run("sends command and receives response", func(t *testing.T) {
		env := setupTestServer(t)
		env.server.RegisterPairingCode("MSG-CODE", []string{"run:stateless"}, 15*time.Minute)

		// Pair
		reqBody := `{"pairing_code":"MSG-CODE","device_name":"Msg Phone","device_os":"iOS"}`
		resp, _ := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		var pairResp mobile.PairingResponse
		json.NewDecoder(resp.Body).Decode(&pairResp)
		resp.Body.Close()

		// Connect
		header := http.Header{}
		header.Set("Authorization", "Bearer "+pairResp.Token)
		conn, _, err := websocket.DefaultDialer.Dial(env.wsURL(), header)
		require.NoError(t, err)
		defer conn.Close()

		// Read ACK
		var ack mobile.WSMessage
		conn.ReadJSON(&ack)

		// Send command
		cmd := mobile.WSMessage{
			ID:     "req-1",
			Type:   "command",
			Action: "run",
			Payload: map[string]any{
				"command": "echo hello",
				"stream":  false,
			},
		}
		err = conn.WriteJSON(cmd)
		require.NoError(t, err)

		// Should receive running, then explicit placeholder acknowledgement.
		var statusMsg mobile.WSMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		err = conn.ReadJSON(&statusMsg)
		require.NoError(t, err)
		assert.Equal(t, "req-1", statusMsg.ID)
		assert.Equal(t, "status", statusMsg.Type)

		var receivedMsg mobile.WSMessage
		err = conn.ReadJSON(&receivedMsg)
		require.NoError(t, err)
		assert.Equal(t, "req-1", receivedMsg.ID)
		assert.Equal(t, "status", receivedMsg.Type)
		payload, ok := receivedMsg.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "received", payload["status"])
		assert.Equal(t, "placeholder", payload["maturity"])
		assert.Equal(t, "none", payload["executes"])
		assert.Contains(t, payload["message"], "received but not executed")
	})

	t.Run("unknown message type returns error", func(t *testing.T) {
		env := setupTestServer(t)
		env.server.RegisterPairingCode("ERR-CODE", []string{"run:stateless"}, 15*time.Minute)

		reqBody := `{"pairing_code":"ERR-CODE","device_name":"Err Phone","device_os":"iOS"}`
		resp, _ := http.Post(env.pairURL(), "application/json", strings.NewReader(reqBody))
		var pairResp mobile.PairingResponse
		json.NewDecoder(resp.Body).Decode(&pairResp)
		resp.Body.Close()

		header := http.Header{}
		header.Set("Authorization", "Bearer "+pairResp.Token)
		conn, _, err := websocket.DefaultDialer.Dial(env.wsURL(), header)
		require.NoError(t, err)
		defer conn.Close()

		// Read ACK
		var ack mobile.WSMessage
		conn.ReadJSON(&ack)

		// Send unknown type
		err = conn.WriteJSON(mobile.WSMessage{ID: "bad-1", Type: "unknown"})
		require.NoError(t, err)

		var errMsg mobile.WSMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		err = conn.ReadJSON(&errMsg)
		require.NoError(t, err)
		assert.Equal(t, "error", errMsg.Type)
	})
}

func TestDetectLANAddress(t *testing.T) {
	addr, err := mobile.DetectLANAddress()
	require.NoError(t, err)
	assert.NotEmpty(t, addr)
	// Should be a valid IP (not empty, not error)
	assert.NotEqual(t, "", addr)
}

func TestListNetworkInterfaces(t *testing.T) {
	ifaces, err := mobile.ListNetworkInterfaces()
	require.NoError(t, err)
	// Should find at least one interface on any machine
	// (may be empty in some CI environments, so just check no error)
	_ = ifaces
}
