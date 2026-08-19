package msgroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSenderKey(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		raw      string
		want     string
	}{
		// Channel prefixes are stripped regardless of platform.
		{name: "twilio whatsapp prefix", platform: "whatsapp", raw: "whatsapp:+15551234567", want: "+15551234567"},
		{name: "prefix case-insensitive", platform: "whatsapp", raw: "WhatsApp:+15551234567", want: "+15551234567"},
		{name: "sms prefix", platform: "sms", raw: "sms:+15551234567", want: "+15551234567"},
		{name: "tel prefix", platform: "sms", raw: "tel:+15551234567", want: "+15551234567"},
		{name: "mailto prefix", platform: "slack", raw: "mailto:Jane@Acme.com", want: "jane@acme.com"},
		{name: "prefix with surrounding whitespace", platform: "whatsapp", raw: "  whatsapp: +15551234567 ", want: "+15551234567"},

		// Phone canonicalization: separators removed, leading + added on phone platforms.
		{name: "plain e164", platform: "sms", raw: "+15551234567", want: "+15551234567"},
		{name: "whatsapp cloud wa_id without plus", platform: "whatsapp", raw: "15551234567", want: "+15551234567"},
		{name: "formatted phone", platform: "sms", raw: "+1 (555) 123-4567", want: "+15551234567"},
		{name: "dotted phone", platform: "sms", raw: "1.555.123.4567", want: "+15551234567"},
		{name: "prefix implies phone on non-phone platform", platform: "slack", raw: "tel:1 555 123 4567", want: "+15551234567"},
		{name: "short code on sms", platform: "sms", raw: "12345", want: "+12345"},
		{name: "alphanumeric sender id on sms is untouched", platform: "sms", raw: "VERIFY", want: "VERIFY"},

		// Non-phone platforms keep numeric IDs verbatim.
		{name: "telegram numeric user id", platform: "telegram", raw: "1001", want: "1001"},
		{name: "discord snowflake", platform: "discord", raw: "987654321098765432", want: "987654321098765432"},
		{name: "slack user id keeps case", platform: "slack", raw: "U012ABC", want: "U012ABC"},

		// Emails are lowercased everywhere.
		{name: "email lowercased", platform: "email", raw: "Jane.Doe@Acme.COM", want: "jane.doe@acme.com"},
		{name: "email trimmed", platform: "slack", raw: "  jane@acme.com\n", want: "jane@acme.com"},

		// Degenerate input.
		{name: "empty", platform: "sms", raw: "", want: ""},
		{name: "whitespace only", platform: "sms", raw: "   ", want: ""},
		{name: "prefix only", platform: "whatsapp", raw: "whatsapp:", want: ""},
		{name: "plus only", platform: "sms", raw: "+", want: "+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeSenderKey(tt.platform, tt.raw))
		})
	}
}

func TestNormalizeSenderKey_Idempotent(t *testing.T) {
	inputs := []struct{ platform, raw string }{
		{"whatsapp", "whatsapp:+1 (555) 123-4567"},
		{"sms", "15551234567"},
		{"slack", "U012ABC"},
		{"email", "Jane@Acme.com"},
		{"telegram", "1001"},
	}
	for _, in := range inputs {
		once := NormalizeSenderKey(in.platform, in.raw)
		twice := NormalizeSenderKey(in.platform, once)
		assert.Equal(t, once, twice, "normalizing %q twice must be stable", in.raw)
	}
}

func TestNormalizeSenderPattern(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		raw      string
		want     string
	}{
		{name: "phone glob keeps wildcard", platform: "sms", raw: "+1 555 *", want: "+1555*"},
		{name: "phone glob without plus gets plus on phone platform", platform: "whatsapp", raw: "1555?", want: "+1555?"},
		{name: "prefixed glob", platform: "whatsapp", raw: "whatsapp:+1555*", want: "+1555*"},
		{name: "email glob lowercased", platform: "slack", raw: "*@Acme.com", want: "*@acme.com"},
		{name: "plain id glob untouched", platform: "slack", raw: "U01*", want: "U01*"},
		{name: "bare star is not a phone", platform: "sms", raw: "*", want: "*"},
		{name: "leading wildcard gets no plus", platform: "sms", raw: "?5551234567", want: "?5551234567"},
		{name: "exact key same as NormalizeSenderKey", platform: "sms", raw: "+1 (555) 123-4567", want: "+15551234567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeSenderPattern(tt.platform, tt.raw))
		})
	}
}

func TestIsPhonePlatform(t *testing.T) {
	assert.True(t, IsPhonePlatform("sms"))
	assert.True(t, IsPhonePlatform("whatsapp"))
	assert.True(t, IsPhonePlatform("WhatsApp"))
	assert.False(t, IsPhonePlatform("telegram"))
	assert.False(t, IsPhonePlatform(""))
}
