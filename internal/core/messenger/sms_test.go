package messenger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSMSPayload_TwilioAndGeneric(t *testing.T) {
	received := time.Date(2026, 5, 11, 12, 30, 0, 0, time.UTC)
	tests := []struct {
		name    string
		payload map[string]any
		wantID  string
		wantTxt string
	}{
		{
			name: "twilio",
			payload: map[string]any{
				"MessageSid": "SM123",
				"AccountSid": "AC123",
				"From":       "+15550100001",
				"To":         "+15550100002",
				"Body":       "hello from twilio",
			},
			wantID:  "SM123",
			wantTxt: "hello from twilio",
		},
		{
			name: "generic",
			payload: map[string]any{
				"message_id": "generic-1",
				"from":       "+15550100003",
				"to":         "+15550100004",
				"text":       "hello from generic sms",
			},
			wantID:  "generic-1",
			wantTxt: "hello from generic sms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NormalizeSMSPayload(tt.payload, received)
			if err != nil {
				t.Fatalf("NormalizeSMSPayload: %v", err)
			}
			if msg.ID != tt.wantID || msg.Text != tt.wantTxt {
				t.Fatalf("message = %#v, want id=%q text=%q", msg, tt.wantID, tt.wantTxt)
			}
			if msg.Platform != string(PlatformSMS) || msg.Channel.Type != ChannelTypeDirect {
				t.Fatalf("unexpected sms identity: %#v", msg)
			}
		})
	}
}

func TestParseSMSPayload_FormBody(t *testing.T) {
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", "hello")

	payload, err := ParseSMSPayload(map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
	}, []byte(form.Encode()))
	if err != nil {
		t.Fatalf("ParseSMSPayload: %v", err)
	}
	if payload["MessageSid"] != "SM123" || payload["Body"] != "hello" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSMSProvider_DeliverUsesMockableTransport(t *testing.T) {
	transport := &recordingSMSTransport{result: &SMSDeliveryResult{
		MessageSID: "SMDELIVERED",
		Status:     "queued",
		Provider:   "twilio",
		Raw:        map[string]any{"sid": "SMDELIVERED"},
	}}
	provider := NewSMSProvider(SMSProviderConfig{
		ServiceID:  "sms-alerts",
		Provider:   "twilio",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15550100002",
	}, transport)

	receipt, err := provider.DeliverMessage(context.Background(), DeliveryRequest{
		Provider:  "twilio",
		ServiceID: "sms-alerts",
		ChannelID: "+15550100001",
		Text:      "ack",
	})
	if err != nil {
		t.Fatalf("DeliverMessage: %v", err)
	}
	if receipt.DeliveryID != "SMDELIVERED" || receipt.Status != "queued" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if transport.delivery.From != "+15550100002" || transport.delivery.To != "+15550100001" || transport.delivery.Body != "ack" {
		t.Fatalf("delivery = %#v", transport.delivery)
	}
}

func TestTwilioSMSTransport_SendSMS(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := ioReadAllString(r)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sid": "SM123", "status": "queued"})
	}))
	defer server.Close()

	transport := NewTwilioSMSTransport("AC123", "token", server.Client())
	transport.BaseURL = server.URL
	result, err := transport.SendSMS(context.Background(), SMSDelivery{
		From: "+15550100002",
		To:   "+15550100001",
		Body: "hello",
	})
	if err != nil {
		t.Fatalf("SendSMS: %v", err)
	}
	if result.MessageSID != "SM123" || result.Status != "queued" {
		t.Fatalf("result = %#v", result)
	}
	if gotPath != "/2010-04-01/Accounts/AC123/Messages.json" {
		t.Fatalf("path = %q", gotPath)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("AC123:token"))
	if gotAuth != wantAuth {
		t.Fatalf("auth = %q, want %q", gotAuth, wantAuth)
	}
	if !strings.Contains(gotBody, "From=%2B15550100002") || !strings.Contains(gotBody, "To=%2B15550100001") {
		t.Fatalf("body = %q", gotBody)
	}
}

type recordingSMSTransport struct {
	delivery SMSDelivery
	result   *SMSDeliveryResult
	err      error
}

func (t *recordingSMSTransport) SendSMS(_ context.Context, delivery SMSDelivery) (*SMSDeliveryResult, error) {
	t.delivery = delivery
	if t.err != nil {
		return nil, t.err
	}
	return t.result, nil
}

func ioReadAllString(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
