package messenger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
)

func TestSlackService_URLVerificationReturnsChallenge(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSlackService(t, map[string]string{"default_action": "reply"})

	executor := &fakeActionExecutor{}
	transport := &fakeSlackTransport{}
	handler := newServiceTestHandler(executor, WithSlackTransport(transport))
	body := []byte(`{"type":"url_verification","challenge":"challenge-value"}`)
	req := signedSlackRequest(body)
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "slack")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "challenge-value" {
		t.Fatalf("body = %q, want Slack challenge", rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
	}
	if transport.calls != 0 {
		t.Fatalf("transport calls = %d, want 0", transport.calls)
	}
}

func TestSlackService_RejectsInvalidSignature(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSlackService(t, map[string]string{"default_action": "reply"})

	handler := newServiceTestHandler(&fakeActionExecutor{}, WithSlackTransport(&fakeSlackTransport{}))
	body := slackFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(time.Now().UTC().Unix(), 10))
	req.Header.Set("X-Slack-Signature", "v0=bad")
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "slack")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestSlackService_DeduplicatesEventsBeforeExecution(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSlackService(t, map[string]string{"default_action": "reply"})

	executor := &fakeActionExecutor{output: "threaded response"}
	transport := &fakeSlackTransport{}
	handler := newServiceTestHandler(executor, WithSlackTransport(transport))
	body := slackFixture(t)

	for i := 0; i < 2; i++ {
		req := signedSlackRequest(body)
		rec := httptest.NewRecorder()
		handler.ServeServiceWebhook(rec, req, "support-bot", "slack")
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200; body=%s", i+1, rec.Code, rec.Body.String())
		}
	}

	if executor.input.ProfileID != "assistant" || executor.input.ActionID != "reply" {
		t.Fatalf("executor input = %#v, want assistant/reply", executor.input)
	}
	if transport.calls != 1 {
		t.Fatalf("transport calls = %d, want only first event delivered", transport.calls)
	}
}

func TestSlackService_AllowedChannelAndBotMention(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSlackService(t, map[string]string{
		"default_action":      "reply",
		"allowed_channels":    "C012CHAN",
		"require_bot_mention": "true",
		"bot_user_id":         "U012BOT",
	})

	executor := &fakeActionExecutor{}
	handler := newServiceTestHandler(executor, WithSlackTransport(&fakeSlackTransport{}))
	req := signedSlackRequest([]byte(`{
		"type":"event_callback",
		"team_id":"T012TEAM",
		"event_id":"EvNoMention",
		"event":{"type":"message","user":"U012USER","channel":"C012CHAN","channel_type":"channel","text":"hello","ts":"1710000000.000002"}
	}`))
	rec := httptest.NewRecorder()

	handler.ServeServiceWebhook(rec, req, "support-bot", "slack")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.ProfileID != "" {
		t.Fatalf("executor profile = %q, want no execution", executor.input.ProfileID)
	}
}

func TestSlackProvider_DeliversThreadReply(t *testing.T) {
	transport := &fakeSlackTransport{}
	provider := NewSlackProvider(SlackProviderConfig{
		BotToken:  "xoxb-test",
		Transport: transport,
		Now:       func() time.Time { return time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC) },
	})

	receipt, err := provider.DeliverMessage(context.Background(), coremessenger.DeliveryRequest{
		Provider:  "slack",
		ServiceID: "support-bot",
		ChannelID: "C012CHAN",
		ThreadID:  "1710000000.000001",
		Text:      "reply text",
	})
	if err != nil {
		t.Fatalf("DeliverMessage: %v", err)
	}
	if receipt.DeliveryID != "1710000000.000099" {
		t.Fatalf("delivery id = %q, want Slack ts", receipt.DeliveryID)
	}
	if transport.last.ThreadTS != "1710000000.000001" {
		t.Fatalf("thread_ts = %q, want thread reply", transport.last.ThreadTS)
	}
}

func saveSlackService(t *testing.T, options map[string]string) {
	t.Helper()
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Env: map[string]string{
			"SLACK_BOT_TOKEN":      "xoxb-test",
			"SLACK_SIGNING_SECRET": "test-secret",
		},
		Options: options,
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
}

func slackFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/slack_event_callback.json")
	if err != nil {
		t.Fatalf("read Slack fixture: %v", err)
	}
	return data
}

func signedSlackRequest(body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/services/support-bot/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", "v0="+signSlack("test-secret", ts, body))
	return req
}

func signSlack(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + timestamp + ":"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type fakeSlackTransport struct {
	calls int
	last  SlackPostMessageRequest
}

func (f *fakeSlackTransport) PostMessage(ctx context.Context, botToken string, req SlackPostMessageRequest) (*SlackPostMessageResponse, error) {
	f.calls++
	f.last = req
	return &SlackPostMessageResponse{
		OK:      true,
		Channel: req.Channel,
		TS:      "1710000000.000099",
	}, nil
}
