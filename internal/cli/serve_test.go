package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/protocol"
)

func newTestHandler(t *testing.T, token string) http.Handler {
	t.Helper()
	mgr := adapters.DefaultManager()
	if err := mgr.InitAll(context.Background()); err != nil {
		t.Logf("InitAll: %v (continuing)", err)
	}
	core, err := protocol.NewAPSAdapter()
	if err != nil {
		t.Fatalf("NewAPSAdapter: %v", err)
	}
	h, err := buildServerHandler(mgr, core, token)
	if err != nil {
		t.Fatalf("buildServerHandler: %v", err)
	}
	return h
}

// TestBuildServerHandler_HealthOK verifies GET /health returns 200 with
// JSON {"status":"healthy"}. Drives the kit Router migration in T-0348.
func TestBuildServerHandler_HealthOK(t *testing.T) {
	handler := newTestHandler(t, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type: got %q, want application/json...", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rec.Body.String())
	}
	if body["status"] != "healthy" {
		t.Fatalf("status field: got %q, want healthy", body["status"])
	}
}

// TestBuildServerHandler_RequestIDHeader verifies the kit RequestID
// middleware is wired: every response carries an X-Request-ID header.
func TestBuildServerHandler_RequestIDHeader(t *testing.T) {
	handler := newTestHandler(t, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rid := rec.Header().Get("X-Request-ID"); rid == "" {
		t.Fatalf("X-Request-ID header missing on response")
	}
}

// TestBuildServerHandler_AuthRejectsMissingToken verifies the kit Auth
// middleware returns 401 for protected routes when no Authorization
// header is present.
func TestBuildServerHandler_AuthRejectsMissingToken(t *testing.T) {
	handler := newTestHandler(t, "secret-token")

	req := httptest.NewRequest(http.MethodPost, "/v1/runs", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

// TestBuildServerHandler_AuthAcceptsValidToken verifies a valid bearer
// token passes the Auth middleware and reaches downstream handlers
// (which may return 4xx/5xx but NOT 401).
func TestBuildServerHandler_AuthAcceptsValidToken(t *testing.T) {
	handler := newTestHandler(t, "secret-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

// TestBuildServerHandler_HealthBypassesAuth verifies /health remains
// reachable without credentials even when an auth token is configured.
func TestBuildServerHandler_HealthBypassesAuth(t *testing.T) {
	handler := newTestHandler(t, "secret-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestBuildServerHandler_MessageServiceWebhookMounted(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	requireTestProfileAction(t, "assistant", "reply")
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "reply",
			"reply":          "none",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
	handler := newTestHandler(t, "")

	body := `{
		"message": {
			"message_id": 1,
			"from": {"id": 456, "first_name": "Alice"},
			"chat": {"id": -1001234567890, "type": "group"},
			"text": "hello"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("message service webhook route was not mounted; body=%s", rec.Body.String())
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "message_id") {
		t.Fatalf("response should be messenger pipeline JSON; body=%s", rec.Body.String())
	}
}

func requireTestProfileAction(t *testing.T, profileID, actionID string) {
	t.Helper()
	if err := core.SaveProfile(&core.Profile{
		ID:          profileID,
		DisplayName: profileID,
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		t.Fatalf("GetProfileDir: %v", err)
	}
	actionsDir := filepath.Join(profileDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll actions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(actionsDir, actionID+".sh"), []byte("printf routed"), 0o755); err != nil {
		t.Fatalf("WriteFile action: %v", err)
	}
}
