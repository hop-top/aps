package messenger

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

type fixtureAuthHook struct {
	req AuthRequirements
}

func TestServiceValidator_DiscordEd25519Signature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"type":1}`)
	timestamp := now.Format(time.RFC3339)
	signature := ed25519.Sign(privateKey, append([]byte(timestamp), body...))

	headers := http.Header{}
	headers.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	headers.Set("X-Signature-Timestamp", timestamp)

	validator := NewServiceValidator()
	validator.Now = func() time.Time { return now }
	err = validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "discord-support",
			Adapter: "discord",
			Options: map[string]string{
				"receive":            "interaction",
				"discord_public_key": hex.EncodeToString(publicKey),
			},
		},
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("ValidateRequest signed Discord interaction: %v", err)
	}

	headers.Set("X-Signature-Ed25519", hex.EncodeToString(make([]byte, ed25519.SignatureSize)))
	if err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "discord-support",
			Adapter: "discord",
			Options: map[string]string{
				"receive":            "interaction",
				"discord_public_key": hex.EncodeToString(publicKey),
			},
		},
		Headers: headers,
		Body:    body,
	}); !IsAuthFailed(err) {
		t.Fatalf("invalid Discord signature error = %v, want auth failure", err)
	}
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

func TestServiceValidator_TelegramSecretToken(t *testing.T) {
	validator := NewServiceValidator()
	service := ServiceValidationConfig{
		ID:      "support-bot",
		Adapter: "telegram",
		Options: map[string]string{
			"webhook_secret_token": "telegram-secret",
		},
	}

	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: http.Header{},
	})
	if !IsAuthFailed(err) {
		t.Fatalf("missing telegram token error = %v, want auth failure", err)
	}

	headers := http.Header{}
	headers.Set(TelegramSecretTokenHeader, "telegram-secret")
	if err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: service,
		Headers: headers,
	}); err != nil {
		t.Fatalf("ValidateRequest with telegram token: %v", err)
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
			"auth_scheme":          "hmac-sha256",
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
				"auth_scheme":         "hmac-sha256",
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

func TestServiceValidator_SlackSigningSecret(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"type":"event_callback","event_id":"Ev1","event":{"type":"app_mention","user":"U1","channel":"C1","text":"<@UBOT> hi","ts":"1710000000.000001"}}`)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := http.Header{}
	headers.Set("X-Slack-Request-Timestamp", ts)
	headers.Set("X-Slack-Signature", "v0="+slackSignature("secret", ts, body))

	t.Setenv("SLACK_SIGNING_SECRET", "secret")
	validator := NewServiceValidator()
	validator.Now = func() time.Time { return now }

	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "slack-bot",
			Adapter: "slack",
			Env:     map[string]string{"SLACK_SIGNING_SECRET": "secret:SLACK_SIGNING_SECRET"},
			Options: map[string]string{},
		},
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("ValidateRequest Slack signature: %v", err)
	}

	headers.Set("X-Slack-Signature", "v0=bad")
	if err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{ID: "slack-bot", Adapter: "slack", Options: map[string]string{}},
		Headers: headers,
		Body:    body,
	}); !IsAuthFailed(err) {
		t.Fatalf("invalid Slack signature error = %v, want auth failure", err)
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

func TestServiceValidator_DiscordAllowedGuilds(t *testing.T) {
	validator := NewServiceValidator()
	service := ServiceValidationConfig{
		ID:      "discord-support",
		Adapter: "discord",
		Options: map[string]string{
			"allowed_guilds": "1300000000000000003",
		},
	}
	msg := &NormalizedMessage{
		ID:          "1100000000000000001",
		Platform:    "discord",
		WorkspaceID: "9999999999999999999",
		Sender:      Sender{ID: "1400000000000000004"},
		Channel:     Channel{ID: "1200000000000000002"},
	}
	if err := validator.ValidateMessage(service, msg); !IsSenderNotAllowed(err) {
		t.Fatalf("blocked guild error = %v, want sender not allowed", err)
	}

	msg.WorkspaceID = "1300000000000000003"
	if err := validator.ValidateMessage(service, msg); err != nil {
		t.Fatalf("ValidateMessage allowed guild: %v", err)
	}
}

func TestServiceValidator_TwilioSignature(t *testing.T) {
	t.Setenv("ALT_TWILIO_AUTH_TOKEN", "twilio-token")

	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("From", "+15550100001")
	form.Set("To", "+15550100002")
	form.Set("Body", "hello")
	requestURL := "https://hooks.example.test/services/sms-alerts/webhook"
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set(TwilioSignatureHeader, TwilioSignature("twilio-token", requestURL, form))

	validator := NewServiceValidator()
	err := validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "sms-alerts",
			Adapter: "sms",
			Env: map[string]string{
				"TWILIO_AUTH_TOKEN": "secret:ALT_TWILIO_AUTH_TOKEN",
			},
			Options: map[string]string{
				"provider": "twilio",
			},
		},
		Method:  http.MethodPost,
		URL:     requestURL,
		Headers: headers,
		Body:    []byte(form.Encode()),
		Form:    form,
	})
	if err != nil {
		t.Fatalf("ValidateRequest signed Twilio webhook: %v", err)
	}

	headers.Set(TwilioSignatureHeader, "invalid")
	err = validator.ValidateRequest(context.Background(), RequestValidationInput{
		Service: ServiceValidationConfig{
			ID:      "sms-alerts",
			Adapter: "sms",
			Env: map[string]string{
				"TWILIO_AUTH_TOKEN": "twilio-token",
			},
			Options: map[string]string{
				"provider": "twilio",
			},
		},
		Method:  http.MethodPost,
		URL:     requestURL,
		Headers: headers,
		Body:    []byte(form.Encode()),
		Form:    form,
	})
	if !IsAuthFailed(err) {
		t.Fatalf("bad Twilio signature error = %v, want auth failure", err)
	}
}

func hmacSHA256Hex(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func slackSignature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + timestamp + ":"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
