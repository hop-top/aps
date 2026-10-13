package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useUmaRelayHost writes a global aps config whose uma relay host settings
// are hosts and allowHTTP, under a throwaway XDG config home.
func useUmaRelayHost(t *testing.T, hosts []string, allowHTTP bool) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	var b strings.Builder
	b.WriteString("uma:\n  relay:\n    callback_hosts:\n")
	for _, h := range hosts {
		b.WriteString("      - \"" + h + "\"\n")
	}
	if allowHTTP {
		b.WriteString("    allow_http: true\n")
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "aps"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "aps", "config.yaml"), []byte(b.String()), 0o600))
}

func umaRelayService() *ServiceConfig {
	return &ServiceConfig{
		ID: "relay-uma", Type: "message", Adapter: "uma", Profile: "assistant",
		Env: map[string]string{
			"RELAY_SIGNING_SECRET": "secret:RELAY_SIGNING_SECRET",
			EnvRelayCallbackToken:  "secret:RELAY_CALLBACK_TOKEN",
		},
		Options: map[string]string{
			OptionUmaAdapter:         UmaAdapterRelay,
			OptionMedium:             "sms",
			"from":                   "+15558675310",
			OptionCallbackURL:        "https://gateway.example.com/uma/reply",
			OptionAuthScheme:         AuthSchemeHMACSHA256,
			OptionSignatureSecretEnv: "RELAY_SIGNING_SECRET",
			"default_action":         "triage",
			"receive":                "webhook",
			"reply":                  "text",
			"allowed_numbers":        "+15551230000",
		},
	}
}

func umaMailService() *ServiceConfig {
	return &ServiceConfig{
		ID: "mail-uma", Type: "message", Adapter: "uma", Profile: "assistant",
		Env: map[string]string{"MAIL_RELAY_TOKEN": "secret:MAIL_RELAY_TOKEN"},
		Options: map[string]string{
			OptionUmaAdapter:     UmaAdapterHTTPMail,
			OptionRecipients:     "support@example.com,help@example.com",
			OptionAuthScheme:     AuthSchemeBearer,
			OptionAuthTokenEnv:   "MAIL_RELAY_TOKEN",
			"default_action":     "triage",
			"receive":            "webhook",
			"reply":              "none",
			OptionAllowedSenders: "*@example.org",
		},
	}
}

func issuesOf(svc *ServiceConfig) string {
	return strings.Join(ValidateServiceConfig(svc).Issues, "\n")
}

func TestUmaAdapterCatalogue_HostsRelayAndMail(t *testing.T) {
	assert.Equal(t, []string{"http.mail", "meta.whatsapp", "relay.messages", "twilio.sms", "twilio.whatsapp"}, UmaAdapterCatalogue())
	relay, ok := UmaAdapterSpecFor(UmaAdapterRelay)
	require.True(t, ok)
	assert.Equal(t, "sms", relay.PlatformFor(map[string]string{OptionMedium: "sms"}))
	assert.Equal(t, "whatsapp", relay.PlatformFor(map[string]string{OptionMedium: "whatsapp"}))
	assert.False(t, relay.ReceiveOnly)
	mail, ok := UmaAdapterSpecFor(UmaAdapterHTTPMail)
	require.True(t, ok)
	assert.Equal(t, "email", mail.PlatformFor(nil))
	assert.True(t, mail.ReceiveOnly)
	sms, _ := UmaAdapterSpecFor("twilio.sms")
	assert.Equal(t, "sms", sms.PlatformFor(nil))
}

func TestValidateServiceConfig_UmaRelayValid(t *testing.T) {
	useUmaRelayHost(t, []string{"gateway.example.com"}, false)
	result := ValidateServiceConfig(umaRelayService())
	assert.True(t, result.Valid, "issues: %v", result.Issues)
}

func TestValidateServiceConfig_UmaRelayIssues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ServiceConfig)
		want   string
	}{
		{"medium required", func(s *ServiceConfig) { delete(s.Options, OptionMedium) }, "option medium"},
		{"medium sms or whatsapp", func(s *ServiceConfig) { s.Options[OptionMedium] = "fax" }, "option medium"},
		{"from required", func(s *ServiceConfig) { delete(s.Options, "from") }, "option from"},
		{"callback url required to reply", func(s *ServiceConfig) { delete(s.Options, OptionCallbackURL) }, "option callback_url"},
		{"callback url https", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "http://gateway.example.com/r" }, "callback_url"},
		{"callback url without query", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "https://gateway.example.com/r?k=1" }, "callback_url"},
		{"callback url without fragment", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "https://gateway.example.com/r#f" }, "callback_url"},
		{"callback url without credentials", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "https://u:p@gateway.example.com/r" }, "callback_url"},
		{"callback host listed", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "https://evil.example.net/r" }, "uma.relay.callback_hosts"},
		{"callback host and port listed", func(s *ServiceConfig) { s.Options[OptionCallbackURL] = "https://gateway.example.com:8443/r" }, "uma.relay.callback_hosts"},
		{"callback credential binding", func(s *ServiceConfig) { delete(s.Env, EnvRelayCallbackToken) }, "missing env binding RELAY_CALLBACK_TOKEN"},
		{"callback header reserved", func(s *ServiceConfig) { s.Options[OptionCallbackHeader] = "Idempotency-Key" }, "callback_header"},
		{"callback header a field name", func(s *ServiceConfig) { s.Options[OptionCallbackHeader] = "X Bad" }, "callback_header"},
		{"webhook auth required", func(s *ServiceConfig) {
			delete(s.Options, OptionAuthScheme)
			delete(s.Options, OptionSignatureSecretEnv)
		}, "no unauthenticated mode"},
		{"webhook secret binding", func(s *ServiceConfig) { delete(s.Env, "RELAY_SIGNING_SECRET") }, "missing env binding RELAY_SIGNING_SECRET"},
		{"literal secret is not a binding", func(s *ServiceConfig) {
			delete(s.Options, OptionSignatureSecretEnv)
			s.Options[OptionSignatureSecret] = "a-literal-secret-in-options"
		}, "--signature-secret-env"},
		{"slack scheme refused", func(s *ServiceConfig) { s.Options[OptionAuthScheme] = AuthSchemeSlack }, "slack-signing-secret"},
		{"unsigned timestamp needs consent", func(s *ServiceConfig) { s.Options[OptionRequireTimestamp] = "true" }, "auth_allow_unsigned"},
		{"unsigned replay id needs consent", func(s *ServiceConfig) { s.Options[OptionReplayIDHeader] = "X-Delivery" }, "auth_allow_unsigned"},
		{"payload on an unsigned scheme", func(s *ServiceConfig) {
			s.Options[OptionAuthScheme] = AuthSchemeBearer
			s.Options[OptionAuthTokenEnv] = "RELAY_SIGNING_SECRET"
			s.Options[OptionAuthPayload] = "{body}"
		}, "auth_payload"},
		{"payload without body", func(s *ServiceConfig) { s.Options[OptionAuthPayload] = "{timestamp}" }, "auth_payload"},
		{"payload placeholder not configured", func(s *ServiceConfig) { s.Options[OptionAuthPayload] = "{timestamp}.{body}" }, "auth_payload"},
		{"tolerance at most 24h", func(s *ServiceConfig) {
			s.Options[OptionRequireTimestamp] = "true"
			s.Options[OptionAuthAllowUnsigned] = "true"
			s.Options[OptionTimestampTolerance] = "25h"
		}, "timestamp_tolerance"},
		{"tolerance a duration", func(s *ServiceConfig) {
			s.Options[OptionRequireTimestamp] = "true"
			s.Options[OptionAuthAllowUnsigned] = "true"
			s.Options[OptionTimestampTolerance] = "soon"
		}, "timestamp_tolerance"},
		{"min secret bytes a positive number", func(s *ServiceConfig) { s.Options[OptionAuthMinSecretBytes] = "0" }, "auth_min_secret_bytes"},
		{"min secret bytes not for ed25519", func(s *ServiceConfig) {
			s.Options[OptionAuthScheme] = AuthSchemeEd25519
			s.Options[OptionAuthMinSecretBytes] = "8"
		}, "auth_min_secret_bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useUmaRelayHost(t, []string{"gateway.example.com"}, false)
			svc := umaRelayService()
			tt.mutate(svc)
			result := ValidateServiceConfig(svc)
			assert.False(t, result.Valid, "want refused")
			assert.Contains(t, strings.Join(result.Issues, "\n"), tt.want)
		})
	}
}

func TestValidateServiceConfig_UmaRelayCallbackHostsFailClosed(t *testing.T) {
	useUmaRelayHost(t, nil, false)
	assert.Contains(t, issuesOf(umaRelayService()), "uma.relay.callback_hosts",
		"with no host listed no callback is allowed")

	useUmaRelayHost(t, []string{"gateway.example.com:8443"}, false)
	svc := umaRelayService()
	svc.Options[OptionCallbackURL] = "https://gateway.example.com:8443/r"
	assert.True(t, ValidateServiceConfig(svc).Valid, "host:port listed as in the URL")

	useUmaRelayHost(t, []string{"GATEWAY.example.com"}, false)
	assert.True(t, ValidateServiceConfig(umaRelayService()).Valid, "hosts compare without case")

	useUmaRelayHost(t, []string{"127.0.0.1:9000"}, false)
	svc.Options[OptionCallbackURL] = "http://127.0.0.1:9000/r"
	assert.Contains(t, issuesOf(svc), "https", "http needs allow_http")
	useUmaRelayHost(t, []string{"127.0.0.1:9000"}, true)
	assert.True(t, ValidateServiceConfig(svc).Valid, "http allowed for development")
}

func TestValidateServiceConfig_UmaRelayReceiveOnlyNeedsNoCallback(t *testing.T) {
	useUmaRelayHost(t, nil, false)
	svc := umaRelayService()
	svc.Options["reply"] = "none"
	delete(svc.Options, OptionCallbackURL)
	delete(svc.Env, EnvRelayCallbackToken)
	result := ValidateServiceConfig(svc)
	assert.True(t, result.Valid, "issues: %v", result.Issues)
}

func TestValidateServiceConfig_UmaRelayConsentAndSignedLayout(t *testing.T) {
	useUmaRelayHost(t, []string{"gateway.example.com"}, false)
	svc := umaRelayService()
	svc.Options[OptionRequireTimestamp] = "true"
	svc.Options[OptionRequireReplayCheck] = "true"
	svc.Options[OptionAuthAllowUnsigned] = "true"
	assert.True(t, ValidateServiceConfig(svc).Valid, "consent accepts aps's legacy layout: %v", ValidateServiceConfig(svc).Issues)

	svc = umaRelayService()
	svc.Options[OptionRequireTimestamp] = "true"
	svc.Options[OptionRequireReplayCheck] = "true"
	svc.Options[OptionAuthPayload] = "{timestamp}.{id}.{body}"
	assert.True(t, ValidateServiceConfig(svc).Valid, "a signed layout needs no consent: %v", ValidateServiceConfig(svc).Issues)
}

func TestUmaWebhookAuthFor_ReproducesTheNativeLayout(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    UmaWebhookAuth
	}{
		{
			"bearer",
			map[string]string{OptionAuthScheme: "bearer", OptionAuthTokenEnv: "TOK"},
			UmaWebhookAuth{Scheme: "bearer", Header: "Authorization", SecretBinding: "TOK"},
		},
		{
			"token",
			map[string]string{OptionAuthScheme: "token", OptionAuthTokenEnv: "TOK"},
			UmaWebhookAuth{Scheme: "token", Header: "X-APS-Token", SecretBinding: "TOK"},
		},
		{
			"bearer inferred from a token binding",
			map[string]string{OptionAuthTokenEnv: "TOK"},
			UmaWebhookAuth{Scheme: "bearer", Header: "Authorization", SecretBinding: "TOK"},
		},
		{
			"hmac inferred from a signature binding",
			map[string]string{OptionSignatureSecretEnv: "SIG"},
			UmaWebhookAuth{
				Scheme: "hmac", Header: "X-APS-Signature", Encoding: "hex", Prefix: "sha256=", PrefixOptional: true,
				Payload: "{body}", SecretBinding: "SIG",
			},
		},
		{
			"hmac with aps's timestamp and replay headers",
			map[string]string{
				OptionAuthScheme: "hmac-sha256", OptionSignatureSecretEnv: "SIG", OptionRequireTimestamp: "true",
				OptionRequireReplayCheck: "true", OptionTimestampTolerance: "2m", OptionAuthAllowUnsigned: "true",
			},
			UmaWebhookAuth{
				Scheme: "hmac", Header: "X-APS-Signature", Encoding: "hex", Prefix: "sha256=", PrefixOptional: true,
				Payload: "{body}", SecretBinding: "SIG",
				TimestampHeader: "X-APS-Timestamp", TimestampFormat: "unix_or_rfc3339", TimestampTolerance: 2 * time.Minute,
				TimestampUnsigned: true, ReplayHeader: "X-APS-Delivery-ID", ReplayUnsigned: true,
			},
		},
		{
			"ed25519 signs its timestamp",
			map[string]string{OptionAuthScheme: "ed25519", OptionSignatureSecretEnv: "PUB"},
			UmaWebhookAuth{
				Scheme: "ed25519", Header: "X-Signature-Ed25519", Encoding: "hex", KeyEncoding: "hex",
				Payload: "{timestamp}{body}", SecretBinding: "PUB",
				TimestampHeader: "X-Signature-Timestamp", TimestampFormat: "unix_or_rfc3339", TimestampTolerance: 5 * time.Minute,
			},
		},
		{
			"header overrides and a short secret",
			map[string]string{
				OptionAuthScheme: "token", OptionAuthTokenEnv: "TOK", OptionAuthHeader: "X-Relay-Token",
				OptionAuthMinSecretBytes: "8",
			},
			UmaWebhookAuth{Scheme: "token", Header: "X-Relay-Token", SecretBinding: "TOK", MinSecretBytes: 8},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, issues := UmaWebhookAuthFor(tt.options)
			assert.Empty(t, issues)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateServiceConfig_UmaMailValid(t *testing.T) {
	result := ValidateServiceConfig(umaMailService())
	assert.True(t, result.Valid, "issues: %v", result.Issues)

	svc := umaMailService()
	svc.Options["reply"] = "text"
	result = ValidateServiceConfig(svc)
	assert.True(t, result.Valid, "a reply mode is warned about, not refused: %v", result.Issues)
}

func TestValidateServiceConfig_UmaMailIssues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ServiceConfig)
		want   string
	}{
		{"recipients required", func(s *ServiceConfig) { delete(s.Options, OptionRecipients) }, "option recipients"},
		{"bare addresses", func(s *ServiceConfig) { s.Options[OptionRecipients] = "Support <support@example.com>" }, "recipients"},
		{"no duplicates", func(s *ServiceConfig) { s.Options[OptionRecipients] = "a@example.com,a@EXAMPLE.com" }, "recipients"},
		{"known formats", func(s *ServiceConfig) { s.Options[OptionFormats] = "json,form" }, "option formats"},
		{"recipient header needs rfc822", func(s *ServiceConfig) { s.Options[OptionRecipientHeader] = "X-Envelope-To" }, "recipient_header"},
		{"results field needs a trusted id", func(s *ServiceConfig) { s.Options[OptionAuthResultsField] = "auth_results" }, "trusted_authserv_id"},
		{"trusted id one token", func(s *ServiceConfig) { s.Options[OptionTrustedAuthservID] = "mx example" }, "trusted_authserv_id"},
		{"webhook auth required", func(s *ServiceConfig) {
			delete(s.Options, OptionAuthScheme)
			delete(s.Options, OptionAuthTokenEnv)
		}, "no unauthenticated mode"},
		{"webhook secret binding", func(s *ServiceConfig) { delete(s.Env, "MAIL_RELAY_TOKEN") }, "missing env binding MAIL_RELAY_TOKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := umaMailService()
			tt.mutate(svc)
			result := ValidateServiceConfig(svc)
			assert.False(t, result.Valid, "want refused")
			assert.Contains(t, strings.Join(result.Issues, "\n"), tt.want)
		})
	}
}

func TestValidateServiceConfig_UmaMailOptionalsAccepted(t *testing.T) {
	svc := umaMailService()
	svc.Options[OptionFormats] = "json,rfc822"
	svc.Options[OptionRecipientHeader] = "X-Envelope-To"
	svc.Options[OptionTrustedAuthservID] = "mx.example.com"
	svc.Options[OptionAuthResultsField] = "authentication_results"
	result := ValidateServiceConfig(svc)
	assert.True(t, result.Valid, "issues: %v", result.Issues)
}

func TestDescribeServiceRuntime_UmaRelayAndMail(t *testing.T) {
	relay := DescribeServiceRuntime(umaRelayService())
	assert.Equal(t, "HTTP POST /services/relay-uma/webhook (uma verifies webhook auth: hmac)", relay.Receives)
	assert.Contains(t, relay.Replies, "callback")
	assert.Equal(t, "ready", relay.Maturity)

	mail := DescribeServiceRuntime(umaMailService())
	assert.Equal(t, "HTTP POST /services/mail-uma/webhook (uma verifies webhook auth: bearer)", mail.Receives)
	assert.Equal(t, "disabled: uma/http.mail receives only; an action's reply is refused, not sent", mail.Replies)
	assert.Empty(t, mail.Metadata.DeliveryModes)
	assert.Contains(t, mail.Metadata.Delivery, "disabled")
}
