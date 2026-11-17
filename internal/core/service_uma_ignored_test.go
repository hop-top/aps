package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func umaMetaWhatsAppService() *ServiceConfig {
	return &ServiceConfig{
		ID:      "wa-uma",
		Type:    "message",
		Adapter: "uma",
		Profile: "assistant",
		Env: map[string]string{
			"WHATSAPP_ACCESS_TOKEN": "secret:WHATSAPP_ACCESS_TOKEN",
			"WHATSAPP_APP_SECRET":   "secret:WHATSAPP_APP_SECRET",
			"WHATSAPP_VERIFY_TOKEN": "secret:WHATSAPP_VERIFY_TOKEN",
		},
		Options: map[string]string{
			OptionUmaAdapter:  "meta.whatsapp",
			"phone_number_id": "104857600000001",
			"default_action":  "triage",
			"allowed_numbers": "+15551230000",
		},
	}
}

func ignoredKeys(service *ServiceConfig) []string {
	hits := UmaIgnoredOptions(service)
	keys := make([]string, 0, len(hits))
	for _, hit := range hits {
		keys = append(keys, hit.Option)
	}
	return keys
}

func TestUmaIgnoredOptions_NothingIgnoredInTheBaseServices(t *testing.T) {
	assert.Empty(t, UmaIgnoredOptions(umaTwilioSMSService()))
	assert.Empty(t, UmaIgnoredOptions(umaMetaWhatsAppService()))
}

func TestUmaIgnoredOptions_EachIgnoredOptionIsReported(t *testing.T) {
	always := map[string]string{
		"template_name": "hello", "language_code": "en", "template_required": "true",
		"verify_token": "tok", "verify_token_env": "VT", "signing_secret_env": "SS",
		"webhook_secret_token": "tok", "webhook_secret_token_env": "WST",
		"auth_scheme": "bearer", "auth_token_env": "AT", "signature_secret_env": "SSE",
		"auth_token": "tok", "auth_header": "X-Token", "signature_secret": "sec",
		"bot_user_id": "U1", "require_bot_mention": "true", "dedup_ttl": "5m",
		"site": "s", "project": "p", "jql": "j", "workspace": "w", "team": "t", "group": "g", "events": "e",
		"provider": "twilio", "receive": "polling",
	}
	for _, base := range []func() *ServiceConfig{umaTwilioSMSService, umaMetaWhatsAppService} {
		for key, value := range always {
			service := base()
			service.Options[key] = value
			assert.Contains(t, ignoredKeys(service), key, "%s on %s", key, service.Options[OptionUmaAdapter])
		}
	}
	twilio := umaTwilioSMSService()
	twilio.Options["phone_number_id"] = "104857600000001"
	assert.Equal(t, []string{"phone_number_id"}, ignoredKeys(twilio), "only meta.whatsapp reads phone_number_id")

	meta := umaMetaWhatsAppService()
	meta.Options["from"] = "+15558675310"
	meta.Options["webhook_url"] = "https://hooks.example.com/wa"
	assert.Equal(t, []string{"from", "webhook_url"}, ignoredKeys(meta), "meta.whatsapp reads neither")
}

func TestUmaIgnoredOptions_FlagAndAdviceAreSet(t *testing.T) {
	service := umaTwilioSMSService()
	service.Options["template_name"] = "hello"
	service.Options["auth_token"] = "tok"
	hits := UmaIgnoredOptions(service)
	require.Len(t, hits, 2)
	byOption := map[string]UmaIgnoredOption{}
	for _, hit := range hits {
		byOption[hit.Option] = hit
		assert.NotEmpty(t, hit.Advice, hit.Option)
	}
	assert.Equal(t, "template-name", byOption["template_name"].Flag)
	assert.Empty(t, byOption["auth_token"].Flag, "auth_token has no flag; only --option reaches it")
}

func TestUmaIgnoredOptions_ConsumedOptionsAreNotReported(t *testing.T) {
	for _, base := range []func() *ServiceConfig{umaTwilioSMSService, umaMetaWhatsAppService} {
		service := base()
		for key, value := range map[string]string{
			"default_action": "triage", "reply": "none", "history_turns": "5", "execution": "chat",
			"allowed_numbers": "+1", "allowed_channels": "c", "allowed_guilds": "g", "allowed_chats": "c",
			"allowed_senders": "a@b", "receive": "webhook", "description": "d",
		} {
			service.Options[key] = value
		}
		assert.Empty(t, UmaIgnoredOptions(service), service.Options[OptionUmaAdapter])
	}
}

func TestUmaIgnoredOptions_UnsetAndDefaultValuesAreNotReported(t *testing.T) {
	service := umaTwilioSMSService()
	service.Options["template_name"] = "  "
	service.Options["template_required"] = "false"
	service.Options["require_bot_mention"] = "false"
	service.Options["receive"] = "Webhook"
	assert.Empty(t, UmaIgnoredOptions(service))
}

func TestUmaIgnoredOptions_OnlyForUmaServices(t *testing.T) {
	assert.Empty(t, UmaIgnoredOptions(nil))
	native := &ServiceConfig{
		ID: "sms-main", Type: "message", Adapter: "sms", Profile: "assistant",
		Options: map[string]string{"provider": "twilio", "template_name": "x", "bot_user_id": "U1"},
	}
	assert.Empty(t, UmaIgnoredOptions(native))
}

func TestUmaIgnoredOptions_AdviceNamesTheReplacement(t *testing.T) {
	meta := umaMetaWhatsAppService()
	meta.Options["verify_token_env"] = "VT"
	meta.Options["signing_secret_env"] = "SS"
	twilio := umaTwilioSMSService()
	twilio.Options["signing_secret_env"] = "SS"

	advice := func(s *ServiceConfig, option string) string {
		for _, hit := range UmaIgnoredOptions(s) {
			if hit.Option == option {
				return hit.Advice
			}
		}
		return ""
	}
	assert.Contains(t, advice(meta, "verify_token_env"), "--env WHATSAPP_VERIFY_TOKEN=secret:")
	assert.Contains(t, advice(meta, "signing_secret_env"), "--env WHATSAPP_APP_SECRET=secret:")
	assert.Contains(t, advice(twilio, "signing_secret_env"), "--env TWILIO_AUTH_TOKEN=secret:")
}
