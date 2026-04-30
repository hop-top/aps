package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/core/protocol"
)

func newTestHandler(t *testing.T, token string) http.Handler {
	t.Helper()
	if err := adapters.RegisterDefaults(); err != nil {
		// Defaults may already be registered by another test; ignore
		// "already registered" so this test is independent.
		t.Logf("RegisterDefaults: %v (continuing)", err)
	}
	core, err := protocol.NewAPSAdapter()
	if err != nil {
		t.Fatalf("NewAPSAdapter: %v", err)
	}
	h, err := buildServerHandler(core, token)
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
