package messenger

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHandler_ServiceWebhookAcceptsSignedTwilioSMSForm(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_AUTH_TOKEN": "twilio-token",
		},
		Options: map[string]string{
			"default_action":  "reply",
			"provider":        "twilio",
			"from":            "+15550100002",
			"allowed_numbers": "+15550100001",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	handler := newServiceTestHandler(executor)
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("AccountSid", "AC123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", "hello over sms")
	req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/sms-alerts/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(coremessenger.TwilioSignatureHeader, coremessenger.TwilioSignature("twilio-token", "https://hooks.example.test/services/sms-alerts/webhook", form))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "sms-alerts", "sms")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if executor.input.ProfileID != "assistant" || executor.input.ActionID != "reply" {
		t.Fatalf("executor input = %#v, want assistant/reply", executor.input)
	}
}

func TestHandler_ServiceWebhookRejectsDisallowedSMSNumber(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_AUTH_TOKEN": "twilio-token",
		},
		Options: map[string]string{
			"default_action":  "reply",
			"provider":        "twilio",
			"from":            "+15550100002",
			"allowed_numbers": "+15550109999",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}

	executor := &fakeActionExecutor{}
	handler := newServiceTestHandler(executor)
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", "hello over sms")
	req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/sms-alerts/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(coremessenger.TwilioSignatureHeader, coremessenger.TwilioSignature("twilio-token", "https://hooks.example.test/services/sms-alerts/webhook", form))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "sms-alerts", "sms")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
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
