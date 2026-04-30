package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/core/protocol"
)

// TestBuildServerHandler_HealthOK verifies GET /health returns 200 with
// JSON {"status":"healthy"}. Drives the kit Router migration in T-0348.
func TestBuildServerHandler_HealthOK(t *testing.T) {
	if err := adapters.RegisterDefaults(); err != nil {
		// Defaults may already be registered by another test; ignore
		// "already registered" so this test is independent.
		t.Logf("RegisterDefaults: %v (continuing)", err)
	}
	core, err := protocol.NewAPSAdapter()
	if err != nil {
		t.Fatalf("NewAPSAdapter: %v", err)
	}

	handler, err := buildServerHandler(core, "")
	if err != nil {
		t.Fatalf("buildServerHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
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
