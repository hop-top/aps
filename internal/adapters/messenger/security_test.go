package messenger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
)

func TestHandler_ServiceWebhookRejectsDisallowedChannelBeforeExecution(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "community-bot",
		Type:    "message",
		Adapter: "discord",
		Profile: "assistant",
		Options: map[string]string{
			"default_action":   "reply",
			"allowed_channels": "1200000000000000002",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	handler := newServiceTestHandler(executor)
	body := `{"id":"m1","author":{"id":"u1","username":"alice"},"channel_id":"blocked-channel","content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/services/community-bot/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "community-bot", "discord")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
	}
}

func TestHandler_ServiceWebhookRejectsMissingProviderTokenBeforeExecution(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "reply",
			"auth_scheme":    "bearer",
			"auth_token":     "super-secret-token-value",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	handler := newServiceTestHandler(executor)
	body := `{"event":{"user":"U12345","channel":"C01ABC2DEF","channel_type":"channel","text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "slack")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "super-secret-token-value") {
		t.Fatalf("response leaked auth token: %s", rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
	}
}

func TestHandler_ServiceWebhookAcceptsProviderHookAuth(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "fixture-bot",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "reply",
			"provider":       "fixture",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	validator := coremessenger.NewServiceValidator()
	validator.Hooks["fixture"] = fixtureHook{req: coremessenger.AuthRequirements{
		Scheme: coremessenger.AuthSchemeToken,
		Header: "X-Fixture-Token",
		Token:  "fixture-secret",
	}}
	handler := newServiceTestHandler(executor, WithServiceValidator(validator))
	body := `{"event":{"user":"U12345","channel":"C01ABC2DEF","channel_type":"channel","text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/services/fixture-bot/webhook", strings.NewReader(body))
	req.Header.Set("X-Fixture-Token", "fixture-secret")
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "fixture-bot", "slack")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if executor.input.ProfileID != "assistant" || executor.input.ActionID != "reply" {
		t.Fatalf("executor input = %#v, want assistant/reply", executor.input)
	}
}

type fixtureHook struct {
	req coremessenger.AuthRequirements
}

func (h fixtureHook) AuthRequirements(coremessenger.ServiceValidationConfig) coremessenger.AuthRequirements {
	return h.req
}

func newServiceTestHandler(executor *fakeActionExecutor, opts ...func(*Handler)) *Handler {
	normalizer := NewNormalizer()
	resolver := &serviceRouteResolver{base: &mockResolver{
		links:   map[string]*coremessenger.ProfileMessengerLink{},
		actions: map[string]string{},
	}}
	router := NewMessageRouterWithExecutor(resolver, normalizer, executor)
	return NewHandler(router, normalizer, nil, opts...)
}
