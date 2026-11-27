package messenger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
)

type recordingWhatsAppTransport struct {
	mu    sync.Mutex
	sends []coremessenger.WhatsAppDelivery
}

func (r *recordingWhatsAppTransport) SendWhatsApp(_ context.Context, d coremessenger.WhatsAppDelivery) (*coremessenger.WhatsAppDeliveryResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sends = append(r.sends, d)
	return &coremessenger.WhatsAppDeliveryResult{MessageID: "SM-out-1", Status: "queued", Provider: d.Provider}, nil
}

func (r *recordingWhatsAppTransport) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sends)
}

func saveTwilioWhatsAppService(t *testing.T, extra map[string]string) {
	t.Helper()
	options := map[string]string{
		"default_action":  "reply",
		"provider":        "twilio",
		"from":            "whatsapp:+15550100002",
		"allowed_numbers": "+15550100001,whatsapp:+15550100001",
		"webhook_url":     "https://hooks.example.test/services/wa-twilio/webhook",
	}
	for k, v := range extra {
		options[k] = v
	}
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "wa-twilio",
		Type:    "message",
		Adapter: "whatsapp",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_AUTH_TOKEN":  "twilio-token",
			"TWILIO_ACCOUNT_SID": "AC123",
		},
		Options: options,
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
}

func postSignedWhatsAppWebhook(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()
	const hook = "https://hooks.example.test/services/wa-twilio/webhook"
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("AccountSid", "AC123")
	form.Set("From", "whatsapp:+15550100001")
	form.Set("To", "whatsapp:+15550100002")
	form.Set("Body", "hello over whatsapp")
	req := httptest.NewRequest(http.MethodPost, hook, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(coremessenger.TwilioSignatureHeader, coremessenger.TwilioSignature("twilio-token", hook, form))
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "wa-twilio", "whatsapp")
	return rec
}

func TestHandler_ServiceWebhookTwilioWhatsAppAnswersEmptyTwiML(t *testing.T) {
	tests := []struct {
		name      string
		executor  *fakeActionExecutor
		options   map[string]string
		wantSends int
	}{
		{name: "reply sent via API", executor: &fakeActionExecutor{output: "ack"}, wantSends: 1},
		{name: "reply mode none", executor: &fakeActionExecutor{output: "suppressed"}, options: map[string]string{"reply": "none"}},
		{name: "failed action", executor: &fakeActionExecutor{status: protocol.RunStatusFailed, output: "boom"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			saveTwilioWhatsAppService(t, tt.options)
			transport := &recordingWhatsAppTransport{}
			handler := newServiceTestHandler(tt.executor, WithWhatsAppTransport(transport))

			rec := postSignedWhatsAppWebhook(t, handler)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
				t.Fatalf("content type = %q, want text/xml", ct)
			}
			body := rec.Body.String()
			if !strings.HasSuffix(body, "<Response/>") || strings.Contains(body, "<Message>") {
				t.Fatalf("body = %q, want empty <Response/>", body)
			}
			if got := transport.count(); got != tt.wantSends {
				t.Fatalf("outbound sends = %d, want %d", got, tt.wantSends)
			}
		})
	}
}

func TestHandler_ServiceWebhookMetaWhatsAppKeepsJSONResponse(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "wa-meta",
		Type:    "message",
		Adapter: "whatsapp",
		Profile: "assistant",
		Env:     map[string]string{"WHATSAPP_ACCESS_TOKEN": "token"},
		Options: map[string]string{
			"default_action":  "reply",
			"provider":        "whatsapp-cloud",
			"phone_number_id": "123456789012345",
			"allowed_numbers": "+15551230001,15551230001",
		},
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
	body := providerFixture(t, "whatsapp_cloud_message.json")
	transport := &recordingWhatsAppTransport{}
	handler := newServiceTestHandler(&fakeActionExecutor{output: "ack"}, WithWhatsAppTransport(transport))
	req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/wa-meta/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "wa-meta", "whatsapp")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content type = %q, want application/json", ct)
	}
}
