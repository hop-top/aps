package messenger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// mockLogger implements MessageLogger for testing.
type mockLogger struct {
	receivedMessages []string
	executedActions  []string
}

func (m *mockLogger) LogMessageReceived(msg *msgtypes.NormalizedMessage) error {
	m.receivedMessages = append(m.receivedMessages, msg.ID)
	return nil
}

func (m *mockLogger) LogActionExecuted(msgID, actionName, status string, durationMS int64) error {
	m.executedActions = append(m.executedActions, msgID+":"+actionName+":"+status)
	return nil
}

func newTestHandler(links map[string]*msgtypes.ProfileMessengerLink, actions map[string]string, logger MessageLogger) *Handler {
	resolver := &mockResolver{
		links:   links,
		actions: actions,
	}
	normalizer := NewNormalizer()
	router := NewMessageRouterWithExecutor(resolver, normalizer, &fakeActionExecutor{})
	return NewHandler(router, normalizer, logger)
}

func TestHandler_ServeHTTP_Success(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		body     string
		linkKey  string
		action   string
	}{
		{
			name:     "telegram webhook",
			platform: "telegram",
			body: `{
				"message": {
					"message_id": 123,
					"from": {"id": 456, "first_name": "Alice", "username": "alice_bot"},
					"chat": {"id": -1001234567890, "title": "Research Team", "type": "group"},
					"text": "Hello research agent!"
				}
			}`,
			linkKey: "telegram:-1001234567890",
			action:  "research-agent=handle_message",
		},
		{
			name:     "slack webhook",
			platform: "slack",
			body: `{
				"event": {
					"user": "U12345",
					"channel": "C01ABC2DEF",
					"channel_type": "channel",
					"text": "Hey from Slack"
				}
			}`,
			linkKey: "slack:C01ABC2DEF",
			action:  "dev-ops=notify",
		},
		{
			name:     "github webhook",
			platform: "github",
			body: `{
				"action": "opened",
				"sender": {"login": "octocat", "id": 1},
				"repository": {"full_name": "org/repo", "name": "repo"},
				"issue": {"number": 42, "title": "Bug found"}
			}`,
			linkKey: "github:org/repo",
			action:  "ci-bot=run_checks",
		},
		{
			name:     "email webhook",
			platform: "email",
			body: `{
				"from": "alice@example.com",
				"to": "bob@example.com",
				"subject": "Test",
				"body": "Email body text"
			}`,
			linkKey: "email:bob@example.com",
			action:  "mail-agent=process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := &msgtypes.ProfileMessengerLink{
				ProfileID:     strings.Split(tt.action, "=")[0],
				MessengerName: tt.platform,
				Enabled:       true,
			}
			handler := newTestHandler(
				map[string]*msgtypes.ProfileMessengerLink{tt.linkKey: link},
				map[string]string{tt.linkKey: tt.action},
				nil,
			)

			path := "/messengers/" + tt.platform + "/webhook"
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp["message_id"] == nil || resp["message_id"] == "" {
				t.Error("response should contain message_id")
			}
			if resp["timestamp"] == nil || resp["timestamp"] == "" {
				t.Error("response should contain timestamp")
			}
		})
	}
}

func TestHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
		nil,
	)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/messengers/telegram/webhook", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status code = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp["error"] != "Method Not Allowed" {
				t.Errorf("error = %v, want %q", resp["error"], "Method Not Allowed")
			}
		})
	}
}

func TestHandler_ServeServiceWebhook_WhatsAppVerificationChallenge(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "wa-support",
		Type:    "message",
		Adapter: "whatsapp",
		Profile: "assistant",
		Env: map[string]string{
			"WHATSAPP_ACCESS_TOKEN": "token",
		},
		Options: map[string]string{
			"default_action":  "reply",
			"provider":        "whatsapp-cloud",
			"phone_number_id": "123456789012345",
			"verify_token":    "verify-me",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
	handler := newTestHandler(map[string]*msgtypes.ProfileMessengerLink{}, map[string]string{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/services/wa-support/webhook?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=challenge-value", nil)
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "wa-support", "whatsapp")

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "challenge-value" {
		t.Fatalf("challenge body = %q", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/services/wa-support/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=challenge-value", nil)
	rec = httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "wa-support", "whatsapp")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token status code = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
		nil,
	)

	tests := []struct {
		name string
		body string
	}{
		{"garbage", "not json at all"},
		{"truncated", `{"message": {`},
		{"empty string", ""},
		{"array instead of object", `[1, 2, 3]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestHandler_ServeHTTP_InvalidPath(t *testing.T) {
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
		nil,
	)

	tests := []struct {
		name string
		path string
	}{
		{"no webhook suffix", "/messengers/telegram"},
		{"missing platform", "/messengers//webhook"},
		{"wrong prefix", "/api/telegram/webhook"},
		{"root path", "/"},
		{"too short", "/webhook"},
		{"completely wrong", "/invalid/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"message": {"from": {"id": 1}, "chat": {"id": 1, "type": "private"}, "text": "hi"}}`
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status code = %d, want %d for path %q", rec.Code, http.StatusNotFound, tt.path)
			}
		})
	}
}

func TestHandler_NilLogger(t *testing.T) {
	key := "telegram:-1001234567890"
	link := &msgtypes.ProfileMessengerLink{
		ProfileID:     "research-agent",
		MessengerName: "telegram",
		Enabled:       true,
	}
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{key: link},
		map[string]string{key: "research-agent=handle_message"},
		nil, // nil logger
	)

	body := `{
		"message": {
			"message_id": 1,
			"from": {"id": 456, "first_name": "Alice"},
			"chat": {"id": -1001234567890, "type": "group"},
			"text": "test"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandler_WithLogger(t *testing.T) {
	key := "telegram:-1001234567890"
	link := &msgtypes.ProfileMessengerLink{
		ProfileID:     "research-agent",
		MessengerName: "telegram",
		Enabled:       true,
	}
	logger := &mockLogger{}
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{key: link},
		map[string]string{key: "research-agent=handle_message"},
		logger,
	)

	body := `{
		"message": {
			"message_id": 1,
			"from": {"id": 456, "first_name": "Alice"},
			"chat": {"id": -1001234567890, "type": "group"},
			"text": "test with logger"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(logger.receivedMessages) != 1 {
		t.Errorf("received messages logged = %d, want 1", len(logger.receivedMessages))
	}
	if len(logger.executedActions) != 1 {
		t.Errorf("executed actions logged = %d, want 1", len(logger.executedActions))
	}
}

func TestHandler_NormalizationFailure(t *testing.T) {
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
		nil,
	)

	// Valid JSON but missing required telegram fields (no message/from/chat).
	body := `{"update_id": 12345}`
	req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandler_RoutingFailure_UnknownChannel(t *testing.T) {
	// No routes configured, message will hit "unknown_channel" in router.
	handler := newTestHandler(
		map[string]*msgtypes.ProfileMessengerLink{},
		map[string]string{},
		nil,
	)

	body := `{
		"message": {
			"message_id": 1,
			"from": {"id": 456, "first_name": "Alice"},
			"chat": {"id": -9999, "type": "private"},
			"text": "unrouted"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/messengers/telegram/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// HandleMessage returns a "failed" ActionResult without an error for unknown channels,
	// so the handler returns 200 with the failure status in the body.
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "failed" {
		t.Errorf("status = %v, want %q", resp["status"], "failed")
	}
}

func TestExtractPlatform(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"standard telegram path", "/messengers/telegram/webhook", "telegram"},
		{"standard slack path", "/messengers/slack/webhook", "slack"},
		{"standard github path", "/messengers/github/webhook", "github"},
		{"standard email path", "/messengers/email/webhook", "email"},
		{"with api prefix", "/api/v1/messengers/telegram/webhook", "telegram"},
		{"trailing slash", "/messengers/telegram/webhook/", "telegram"},
		{"no webhook suffix", "/messengers/telegram", ""},
		{"no messengers prefix", "/api/telegram/webhook", ""},
		{"empty platform", "/messengers//webhook", ""},
		{"root path", "/", ""},
		{"empty string", "", ""},
		{"too short", "/webhook", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlatform(tt.path)
			if got != tt.want {
				t.Errorf("extractPlatform(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractActionFromOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "standard output format",
			output: `action "handle_message" dispatched to profile "research-agent" (message msg_1, platform telegram, channel -123)`,
			want:   "handle_message",
		},
		{
			name:   "different action name",
			output: `action "deploy_notify" dispatched to profile "dev-ops" (message msg_2, platform slack, channel C01)`,
			want:   "deploy_notify",
		},
		{
			name:   "no match",
			output: "some random output text",
			want:   "",
		},
		{
			name:   "empty string",
			output: "",
			want:   "",
		},
		{
			name:   "partial prefix only",
			output: `action "`,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractActionFromOutput(tt.output)
			if got != tt.want {
				t.Errorf("extractActionFromOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
