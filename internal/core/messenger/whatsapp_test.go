package messenger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWhatsAppVerificationChallenge(t *testing.T) {
	query := url.Values{}
	query.Set("hub.mode", "subscribe")
	query.Set("hub.verify_token", "verify-me")
	query.Set("hub.challenge", "challenge-value")

	challenge, err := WhatsAppVerificationChallenge(ServiceValidationConfig{
		ID: "wa-support",
		Options: map[string]string{
			"verify_token": "verify-me",
		},
	}, query)
	if err != nil {
		t.Fatalf("WhatsAppVerificationChallenge: %v", err)
	}
	if challenge != "challenge-value" {
		t.Fatalf("challenge = %q", challenge)
	}

	query.Set("hub.verify_token", "wrong")
	if _, err := WhatsAppVerificationChallenge(ServiceValidationConfig{
		ID:      "wa-support",
		Options: map[string]string{"verify_token": "verify-me"},
	}, query); !IsAuthFailed(err) {
		t.Fatalf("bad verify token error = %v, want auth failure", err)
	}
}

func TestServiceValidator_WhatsAppCloudSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	headers := http.Header{}
	headers.Set(WhatsAppSignatureHeader, "sha256="+hmacSHA256Hex("app-secret", body))

	validator := NewServiceValidator()
	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "wa-support",
			Adapter: "whatsapp",
			Options: map[string]string{
				"provider":   "whatsapp-cloud",
				"app_secret": "app-secret",
			},
		},
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("ValidateRequest signed WhatsApp Cloud webhook: %v", err)
	}

	headers.Set(WhatsAppSignatureHeader, "sha256=bad")
	err = validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "wa-support",
			Adapter: "whatsapp",
			Options: map[string]string{
				"provider":   "whatsapp-cloud",
				"app_secret": "app-secret",
			},
		},
		Headers: headers,
		Body:    body,
	})
	if !IsAuthFailed(err) {
		t.Fatalf("bad WhatsApp signature error = %v, want auth failure", err)
	}
}

func TestNormalizeWhatsAppPayload_CloudAndTwilio(t *testing.T) {
	cloudBody, err := os.ReadFile("testdata/whatsapp_cloud_message.json")
	if err != nil {
		t.Fatalf("read cloud fixture: %v", err)
	}
	payload, err := ParseWhatsAppPayload(map[string][]string{"Content-Type": {"application/json"}}, cloudBody)
	if err != nil {
		t.Fatalf("ParseWhatsAppPayload cloud: %v", err)
	}
	msg, err := NormalizeWhatsAppPayload(payload, time.Time{})
	if err != nil {
		t.Fatalf("NormalizeWhatsAppPayload cloud: %v", err)
	}
	if msg.ID != "wamid.HBgLMTU1NTAxMDAwMRUCABIYFDNB" || msg.Text != "hello from cloud" {
		t.Fatalf("cloud msg = %#v", msg)
	}
	if msg.Sender.ID != "15550100001" || msg.Sender.Name != "Alice" {
		t.Fatalf("cloud sender = %#v", msg.Sender)
	}
	if msg.Channel.ID != "123456789012345" || msg.Channel.Type != ChannelTypeDirect {
		t.Fatalf("cloud channel = %#v", msg.Channel)
	}
	if msg.Thread == nil || msg.Thread.ID != "wamid.previous" {
		t.Fatalf("cloud thread = %#v", msg.Thread)
	}

	formBody, err := os.ReadFile("testdata/twilio_whatsapp_form.txt")
	if err != nil {
		t.Fatalf("read twilio fixture: %v", err)
	}
	payload, err = ParseWhatsAppPayload(map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}}, formBody)
	if err != nil {
		t.Fatalf("ParseWhatsAppPayload twilio: %v", err)
	}
	msg, err = NormalizeWhatsAppPayload(payload, time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NormalizeWhatsAppPayload twilio: %v", err)
	}
	if msg.ID != "SMXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX" || msg.Text != "hello from twilio" {
		t.Fatalf("twilio msg = %#v", msg)
	}
	if msg.Sender.ID != "whatsapp:+15550100001" || msg.Channel.ID != "whatsapp:+15550100002" {
		t.Fatalf("twilio identity = sender %#v channel %#v", msg.Sender, msg.Channel)
	}
}

func TestWhatsAppProvider_DeliverUsesMockableTransportAndTemplateConstraint(t *testing.T) {
	transport := &recordingWhatsAppTransport{result: &WhatsAppDeliveryResult{
		MessageID: "wamid.delivered",
		Status:    "queued",
		Provider:  "whatsapp-cloud",
		Raw:       map[string]any{"messages": []any{map[string]any{"id": "wamid.delivered"}}},
	}}
	provider := NewWhatsAppProvider(WhatsAppProviderConfig{
		ServiceID:       "wa-support",
		Provider:        "whatsapp-cloud",
		AccessToken:     "token",
		PhoneNumberID:   "123456789012345",
		TemplateName:    "support_update",
		LanguageCode:    "en_US",
		RequireTemplate: true,
	}, transport)

	receipt, err := provider.DeliverMessage(context.Background(), DeliveryRequest{
		Provider:  string(PlatformWhatsApp),
		ServiceID: "wa-support",
		ChannelID: "15550100001",
		Text:      "ack",
	})
	if err != nil {
		t.Fatalf("DeliverMessage template: %v", err)
	}
	if receipt.DeliveryID != "wamid.delivered" || receipt.Status != "queued" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if transport.delivery.TemplateName != "support_update" || transport.delivery.To != "15550100001" || transport.delivery.PhoneNumberID != "123456789012345" {
		t.Fatalf("delivery = %#v", transport.delivery)
	}

	provider.Config.TemplateName = ""
	if _, err := provider.DeliverMessage(context.Background(), DeliveryRequest{
		Provider:  string(PlatformWhatsApp),
		ServiceID: "wa-support",
		ChannelID: "15550100001",
		Text:      "ack",
	}); err == nil || !strings.Contains(err.Error(), "template_name") {
		t.Fatalf("template constraint error = %v", err)
	}
}

func TestWhatsAppCloudTransport_SendTextAndTemplate(t *testing.T) {
	var gotPath, gotAuth string
	var gotBodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotBodies = append(gotBodies, body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []any{map[string]any{"id": "wamid.sent"}},
		})
	}))
	defer server.Close()

	transport := NewWhatsAppCloudTransport("token", server.URL, server.Client())
	result, err := transport.SendWhatsApp(context.Background(), WhatsAppDelivery{
		PhoneNumberID: "123456789012345",
		To:            "whatsapp:+15550100001",
		Body:          "hello",
	})
	if err != nil {
		t.Fatalf("SendWhatsApp text: %v", err)
	}
	if result.MessageID != "wamid.sent" || gotPath != "/123456789012345/messages" || gotAuth != "Bearer token" {
		t.Fatalf("result/path/auth = %#v %q %q", result, gotPath, gotAuth)
	}
	if gotBodies[0]["type"] != "text" || gotBodies[0]["to"] != "+15550100001" {
		t.Fatalf("text body = %#v", gotBodies[0])
	}

	_, err = transport.SendWhatsApp(context.Background(), WhatsAppDelivery{
		PhoneNumberID: "123456789012345",
		To:            "15550100001",
		TemplateName:  "hello_world",
		LanguageCode:  "en_US",
	})
	if err != nil {
		t.Fatalf("SendWhatsApp template: %v", err)
	}
	if gotBodies[1]["type"] != "template" {
		t.Fatalf("template body = %#v", gotBodies[1])
	}
}

func TestTwilioWhatsAppTransport_SendWhatsApp(t *testing.T) {
	var gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sid": "SM123", "status": "queued"})
	}))
	defer server.Close()

	transport := NewTwilioWhatsAppTransport("AC123", "token", server.Client())
	transport.BaseURL = server.URL
	result, err := transport.SendWhatsApp(context.Background(), WhatsAppDelivery{
		From: "whatsapp:+15550100002",
		To:   "+15550100001",
		Body: "hello",
	})
	if err != nil {
		t.Fatalf("SendWhatsApp Twilio: %v", err)
	}
	if result.MessageID != "SM123" || result.Status != "queued" {
		t.Fatalf("result = %#v", result)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("AC123:token"))
	if gotAuth != wantAuth {
		t.Fatalf("auth = %q, want %q", gotAuth, wantAuth)
	}
	if !strings.Contains(gotBody, "From=whatsapp%3A%2B15550100002") || !strings.Contains(gotBody, "To=whatsapp%3A%2B15550100001") {
		t.Fatalf("body = %q", gotBody)
	}
}

type recordingWhatsAppTransport struct {
	delivery WhatsAppDelivery
	result   *WhatsAppDeliveryResult
	err      error
}

func (t *recordingWhatsAppTransport) SendWhatsApp(_ context.Context, delivery WhatsAppDelivery) (*WhatsAppDeliveryResult, error) {
	t.delivery = delivery
	if t.err != nil {
		return nil, t.err
	}
	return t.result, nil
}
