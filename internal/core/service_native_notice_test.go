package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNativeTransportUmaAdapter(t *testing.T) {
	cases := []struct {
		name    string
		service *ServiceConfig
		want    string
		ok      bool
	}{
		{"sms twilio", &ServiceConfig{Type: "message", Adapter: "sms", Options: map[string]string{"provider": "twilio"}}, "twilio.sms", true},
		{"whatsapp twilio", &ServiceConfig{Type: "message", Adapter: "whatsapp", Options: map[string]string{"provider": "twilio"}}, "twilio.whatsapp", true},
		{"whatsapp cloud", &ServiceConfig{Type: "message", Adapter: "whatsapp", Options: map[string]string{"provider": "whatsapp-cloud"}}, "meta.whatsapp", true},
		{"provider is trimmed and lowercased", &ServiceConfig{Type: "message", Adapter: "sms", Options: map[string]string{"provider": " Twilio "}}, "twilio.sms", true},
		{"sms generic has no uma adapter yet", &ServiceConfig{Type: "message", Adapter: "sms", Options: map[string]string{"provider": "generic"}}, "", false},
		{"whatsapp generic has no uma adapter yet", &ServiceConfig{Type: "message", Adapter: "whatsapp", Options: map[string]string{"provider": "generic"}}, "", false},
		{"no provider", &ServiceConfig{Type: "message", Adapter: "sms"}, "", false},
		{"already uma", &ServiceConfig{Type: "message", Adapter: "uma", Options: map[string]string{"provider": "twilio", OptionUmaAdapter: "twilio.sms"}}, "", false},
		{"other message adapter", &ServiceConfig{Type: "message", Adapter: "slack", Options: map[string]string{"provider": "twilio"}}, "", false},
		{"email relay", &ServiceConfig{Type: "message", Adapter: "email"}, "", false},
		{"not a message service", &ServiceConfig{Type: "ticket", Adapter: "sms", Options: map[string]string{"provider": "twilio"}}, "", false},
		{"nil", nil, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NativeTransportUmaAdapter(tc.service)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNativeTransportUmaAdapter_IsInTheBridgeCatalogue(t *testing.T) {
	for _, service := range []*ServiceConfig{
		{Type: "message", Adapter: "sms", Options: map[string]string{"provider": "twilio"}},
		{Type: "message", Adapter: "whatsapp", Options: map[string]string{"provider": "twilio"}},
		{Type: "message", Adapter: "whatsapp", Options: map[string]string{"provider": "whatsapp-cloud"}},
	} {
		key, ok := NativeTransportUmaAdapter(service)
		assert.True(t, ok)
		_, hosted := UmaAdapterSpecFor(key)
		assert.True(t, hosted, "%s must be an adapter the bridge hosts", key)
	}
}
