package messenger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"
)

type fixtureAuthHook struct {
	req AuthRequirements
}

func (h fixtureAuthHook) AuthRequirements(ServiceValidationConfig) AuthRequirements {
	return h.req
}

func TestServiceValidator_ProviderAuthHookRequiresToken(t *testing.T) {
	validator := NewServiceValidator()
	validator.Hooks["fixture"] = fixtureAuthHook{req: AuthRequirements{
		Scheme: AuthSchemeToken,
		Header: "X-Fixture-Token",
		Token:  "fixture-secret",
	}}

	service := ServiceValidationConfig{
		ID:      "fixture-bot",
		Adapter: "slack",
		Options: map[string]string{
			"provider": "fixture",
		},
	}

	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: http.Header{},
	})
	if !IsAuthFailed(err) {
		t.Fatalf("missing hook token error = %v, want auth failure", err)
	}

	headers := http.Header{}
	headers.Set("X-Fixture-Token", "fixture-secret")
	if err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: headers,
	}); err != nil {
		t.Fatalf("ValidateRequest with hook token: %v", err)
	}
}

func TestServiceValidator_HMACTimestampAndReplay(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"ok":true}`)
	headers := http.Header{}
	headers.Set("X-APS-Signature", "sha256="+hmacSHA256Hex("secret", body))
	headers.Set("X-APS-Timestamp", now.Format(time.RFC3339))
	headers.Set("X-APS-Delivery-ID", "delivery-1")

	validator := NewServiceValidator()
	validator.Now = func() time.Time { return now }
	service := ServiceValidationConfig{
		ID:      "signed-bot",
		Adapter: "slack",
		Options: map[string]string{
			"signature_secret":     "secret",
			"require_timestamp":    "true",
			"require_replay_check": "true",
			"timestamp_tolerance":  "1m",
		},
	}

	if err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: headers,
		Body:    body,
	}); err != nil {
		t.Fatalf("ValidateRequest first delivery: %v", err)
	}

	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: headers,
		Body:    body,
	})
	if !IsAuthFailed(err) {
		t.Fatalf("duplicate delivery error = %v, want auth failure", err)
	}
}

func TestServiceValidator_RejectsStaleTimestamp(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"ok":true}`)
	headers := http.Header{}
	headers.Set("X-APS-Signature", "sha256="+hmacSHA256Hex("secret", body))
	headers.Set("X-APS-Timestamp", now.Add(-10*time.Minute).Format(time.RFC3339))

	validator := NewServiceValidator()
	validator.Now = func() time.Time { return now }
	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "signed-bot",
			Adapter: "slack",
			Options: map[string]string{
				"signature_secret":    "secret",
				"require_timestamp":   "true",
				"timestamp_tolerance": "1m",
			},
		},
		Headers: headers,
		Body:    body,
	})
	if !IsAuthFailed(err) {
		t.Fatalf("stale timestamp error = %v, want auth failure", err)
	}
}

func TestServiceValidator_AllowedMessageEnforcement(t *testing.T) {
	validator := NewServiceValidator()
	service := ServiceValidationConfig{
		ID:      "sms-alerts",
		Adapter: "sms",
		Options: map[string]string{
			"allowed_numbers": "+15551230001",
		},
	}
	msg := &NormalizedMessage{
		ID:       "msg-1",
		Platform: "sms",
		Sender:   Sender{ID: "+15559870002"},
		Channel:  Channel{ID: "+15559870003"},
	}
	if err := validator.ValidateMessage(service, msg); !IsSenderNotAllowed(err) {
		t.Fatalf("blocked number error = %v, want sender not allowed", err)
	}

	msg.Sender.ID = "+15551230001"
	if err := validator.ValidateMessage(service, msg); err != nil {
		t.Fatalf("ValidateMessage allowed number: %v", err)
	}
}

func hmacSHA256Hex(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
