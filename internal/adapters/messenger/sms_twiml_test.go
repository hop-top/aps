package messenger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
	coremessenger "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
)

func saveSMSTestService(t *testing.T, provider string, extraOptions map[string]string) {
	t.Helper()
	options := map[string]string{
		"default_action":  "reply",
		"from":            "+15550100002",
		"allowed_numbers": "+15550100001",
	}
	if provider != "" {
		options["provider"] = provider
	}
	for k, v := range extraOptions {
		options[k] = v
	}
	if err := core.SaveService(&core.ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_AUTH_TOKEN": "twilio-token",
		},
		Options: options,
	}); err != nil {
		t.Fatalf("SaveService: %v", err)
	}
}

func postSignedSMSWebhook(t *testing.T, handler *Handler, message string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("AccountSid", "AC123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", message)
	req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/sms-alerts/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(coremessenger.TwilioSignatureHeader, coremessenger.TwilioSignature("twilio-token", "https://hooks.example.test/services/sms-alerts/webhook", form))
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "sms-alerts", "sms")
	return rec
}

func TestHandler_ServiceWebhookTwilioSMSRepliesWithTwiML(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSMSTestService(t, "twilio", nil)

	executor := &fakeActionExecutor{output: `ack <one> & "two"`}
	handler := newServiceTestHandler(executor)
	rec := postSignedSMSWebhook(t, handler, "hello over sms")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
		t.Fatalf("content type = %q, want text/xml", ct)
	}
	body := rec.Body.String()
	want := "<Response><Message>ack &lt;one&gt; &amp; &#34;two&#34;</Message></Response>"
	if !strings.Contains(body, want) {
		t.Fatalf("body = %q, want it to contain %q", body, want)
	}
	if executor.input.ProfileID != "assistant" || executor.input.ActionID != "reply" {
		t.Fatalf("executor input = %#v, want assistant/reply", executor.input)
	}
}

func TestHandler_ServiceWebhookTwilioSMSEmptyReplyIsEmptyResponse(t *testing.T) {
	tests := []struct {
		name     string
		executor *fakeActionExecutor
		options  map[string]string
	}{
		{
			name:     "whitespace-only action output",
			executor: &fakeActionExecutor{output: "   "},
		},
		{
			name:     "reply mode none",
			executor: &fakeActionExecutor{output: "suppressed reply"},
			options:  map[string]string{"reply": "none"},
		},
		{
			name:     "failed action",
			executor: &fakeActionExecutor{status: protocol.RunStatusFailed, output: "boom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			saveSMSTestService(t, "twilio", tt.options)

			handler := newServiceTestHandler(tt.executor)
			rec := postSignedSMSWebhook(t, handler, "hello over sms")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
				t.Fatalf("content type = %q, want text/xml", ct)
			}
			body := rec.Body.String()
			if !strings.Contains(body, "<Response/>") {
				t.Fatalf("body = %q, want empty <Response/>", body)
			}
			if strings.Contains(body, "<Message>") {
				t.Fatalf("body = %q, want no <Message> element", body)
			}
		})
	}
}

func TestHandler_ServiceWebhookGenericSMSKeepsJSONResponse(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSMSTestService(t, "generic", nil)

	executor := &fakeActionExecutor{output: "echo reply"}
	handler := newServiceTestHandler(executor)
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", "hello over sms")
	req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/sms-alerts/webhook", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeServiceWebhook(rec, req, "sms-alerts", "sms")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content type = %q, want application/json", ct)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, rec.Body.String())
	}
	if resp["body"] != "echo reply" {
		t.Fatalf(`resp["body"] = %v, want "echo reply"`, resp["body"])
	}
	if resp["message_id"] == nil || resp["message_id"] == "" {
		t.Fatal("response should contain message_id")
	}
}
