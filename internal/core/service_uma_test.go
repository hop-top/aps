package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func umaTwilioSMSService() *ServiceConfig {
	return &ServiceConfig{
		ID:      "sms-uma",
		Type:    "message",
		Adapter: "uma",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_ACCOUNT_SID": "secret:TWILIO_ACCOUNT_SID",
			"TWILIO_AUTH_TOKEN":  "secret:TWILIO_AUTH_TOKEN",
		},
		Options: map[string]string{
			OptionUmaAdapter:  "twilio.sms",
			"from":            "+15558675310",
			"webhook_url":     "https://hooks.example.com/twilio/sms",
			"default_action":  "triage",
			"receive":         "webhook",
			"reply":           "text",
			"allowed_numbers": "+15551230000",
		},
	}
}

func TestResolveServiceType_UmaAlias(t *testing.T) {
	got, err := ResolveServiceType("uma", "")
	require.NoError(t, err)
	assert.Equal(t, "message", got.Type)
	assert.Equal(t, "uma", got.Adapter)
	assert.True(t, got.Aliased)
}

func TestValidateServiceConfig_UmaTwilioSMSValid(t *testing.T) {
	result := ValidateServiceConfig(umaTwilioSMSService())
	assert.True(t, result.Valid, "issues: %v", result.Issues)
	assert.Empty(t, result.Issues)
}

func TestValidateServiceConfig_UmaIssues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ServiceConfig)
		want   string
	}{
		{"no uma adapter", func(s *ServiceConfig) { delete(s.Options, OptionUmaAdapter) }, "uma_adapter"},
		{"unknown uma adapter", func(s *ServiceConfig) { s.Options[OptionUmaAdapter] = "twilio.fax" }, `unsupported uma adapter "twilio.fax"`},
		{"webhook url required", func(s *ServiceConfig) { delete(s.Options, "webhook_url") }, "webhook_url"},
		{"webhook url with query", func(s *ServiceConfig) { s.Options["webhook_url"] = "https://h.example.com/x?a=1" }, "no query"},
		{"webhook url with fragment", func(s *ServiceConfig) { s.Options["webhook_url"] = "https://h.example.com/x#f" }, "no query"},
		{"webhook url not absolute", func(s *ServiceConfig) { s.Options["webhook_url"] = "/relative" }, "absolute"},
		{"from required", func(s *ServiceConfig) { delete(s.Options, "from") }, "option from"},
		{"account sid binding", func(s *ServiceConfig) { delete(s.Env, "TWILIO_ACCOUNT_SID") }, "missing env binding TWILIO_ACCOUNT_SID"},
		{"auth token binding", func(s *ServiceConfig) { delete(s.Env, "TWILIO_AUTH_TOKEN") }, "missing env binding TWILIO_AUTH_TOKEN"},
		{"default action required", func(s *ServiceConfig) { delete(s.Options, "default_action") }, "default_action"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := umaTwilioSMSService()
			tt.mutate(svc)
			result := ValidateServiceConfig(svc)
			assert.False(t, result.Valid)
			assert.Contains(t, strings.Join(result.Issues, "\n"), tt.want)
		})
	}
}

func TestValidateServiceConfig_UmaTwilioWhatsApp(t *testing.T) {
	svc := umaTwilioSMSService()
	svc.Options[OptionUmaAdapter] = "twilio.whatsapp"
	svc.Options["from"] = "+15558675310"
	assert.True(t, ValidateServiceConfig(svc).Valid)

	svc.Options["from"] = "whatsapp:+15558675310"
	result := ValidateServiceConfig(svc)
	assert.False(t, result.Valid)
	assert.Contains(t, strings.Join(result.Issues, "\n"), "without the whatsapp: prefix")
}

func TestValidateServiceConfig_UmaMetaWhatsApp(t *testing.T) {
	svc := &ServiceConfig{
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
			"phone_number_id": "106540352242922",
			"default_action":  "triage",
			"receive":         "webhook",
			"reply":           "text",
		},
	}
	assert.True(t, ValidateServiceConfig(svc).Valid)

	delete(svc.Env, "WHATSAPP_APP_SECRET")
	result := ValidateServiceConfig(svc)
	assert.False(t, result.Valid, "app secret is mandatory under uma")
	assert.Contains(t, strings.Join(result.Issues, "\n"), "WHATSAPP_APP_SECRET")

	svc.Env["WHATSAPP_APP_SECRET"] = "secret:WHATSAPP_APP_SECRET"
	svc.Options["phone_number_id"] = "not-numeric"
	assert.False(t, ValidateServiceConfig(svc).Valid)
}

func TestUmaAdapterCatalogue(t *testing.T) {
	assert.Equal(t, []string{"meta.whatsapp", "twilio.sms", "twilio.whatsapp"}, UmaAdapterCatalogue())
	spec, ok := UmaAdapterSpecFor("twilio.sms")
	require.True(t, ok)
	assert.Equal(t, []string{"TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN"}, spec.RequiredEnv)
	_, ok = UmaAdapterSpecFor("nope")
	assert.False(t, ok)
}

func TestDescribeServiceRuntime_UmaServiceIsReadyAtTheNativeWebhookRoute(t *testing.T) {
	svc := umaTwilioSMSService()
	info := DescribeServiceRuntime(svc)
	assert.Equal(t, "ready", info.Maturity)
	assert.Equal(t, []string{"/services/" + svc.ID + "/webhook"}, info.Routes,
		"a uma service answers at the route its native predecessor had")
	assert.Equal(t, "HTTP POST /services/"+svc.ID+"/webhook (uma verifies the provider signature)", info.Receives)
	assert.Equal(t, "uma/twilio.sms", info.Metadata.Provider)
	assert.Contains(t, info.Replies, "uma/twilio.sms")
	assert.Equal(t, "webhook", info.Metadata.ReceiveMode)
	assert.NotContains(t, info.Receives, "not mounted")

	meta := umaTwilioSMSService()
	meta.Options[OptionUmaAdapter] = "meta.whatsapp"
	assert.Equal(t, "HTTP GET, POST /services/"+meta.ID+"/webhook (uma verifies the provider signature)", DescribeServiceRuntime(meta).Receives,
		"WhatsApp Cloud verifies its subscription with a GET at the same URL")
}
