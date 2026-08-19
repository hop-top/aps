package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyntheticMessageWebhookPayload_IdentitySources(t *testing.T) {
	tests := []struct {
		name    string
		adapter string
		options map[string]string
		want    SyntheticProbeIdentity
	}{
		{
			name:    "sms without allowlist uses placeholders",
			adapter: "sms",
			options: map[string]string{"provider": "generic"},
			want: SyntheticProbeIdentity{
				Sender: "+15550100001", SenderSource: "synthetic",
				Channel: "+15550100002", ChannelSource: "synthetic",
			},
		},
		{
			name:    "sms first allowed number and from",
			adapter: "sms",
			options: map[string]string{"provider": "generic", "from": "+15559990000", "allowed_numbers": " +15550001111 , +15550002222"},
			want: SyntheticProbeIdentity{
				Sender: "+15550001111", SenderSource: "allowed_numbers",
				Channel: "+15559990000", ChannelSource: "from",
			},
		},
		{
			name:    "telegram allowed chat",
			adapter: "telegram",
			options: map[string]string{"allowed_chats": "-1009876543210"},
			want: SyntheticProbeIdentity{
				Sender: "1001", SenderSource: "synthetic",
				Channel: "-1009876543210", ChannelSource: "allowed_chats",
			},
		},
		{
			name:    "discord channel and guild",
			adapter: "discord",
			options: map[string]string{"allowed_channels": "120", "allowed_guilds": "130"},
			want: SyntheticProbeIdentity{
				Sender: "987654321098765432", SenderSource: "synthetic",
				Channel: "120", ChannelSource: "allowed_channels",
				Workspace: "130", WorkspaceSource: "allowed_guilds",
			},
		},
		{
			name:    "whatsapp cloud phone_number_id and allowed number",
			adapter: "whatsapp",
			options: map[string]string{"provider": "whatsapp-cloud", "phone_number_id": "999", "allowed_numbers": "15551239999"},
			want: SyntheticProbeIdentity{
				Sender: "15551239999", SenderSource: "allowed_numbers",
				Channel: "999", ChannelSource: "phone_number_id",
			},
		},
		{
			name:    "whatsapp twilio uses phone-shaped payload",
			adapter: "whatsapp",
			options: map[string]string{"provider": "twilio", "from": "whatsapp:+15559990000", "allowed_numbers": "whatsapp:+15550001111"},
			want: SyntheticProbeIdentity{
				Sender: "whatsapp:+15550001111", SenderSource: "allowed_numbers",
				Channel: "whatsapp:+15559990000", ChannelSource: "from",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, identity, err := SyntheticMessageWebhookPayload(tt.adapter, tt.options)
			require.NoError(t, err)
			assert.Equal(t, tt.want, identity)
			assert.True(t, json.Valid(payload))
		})
	}
}

func TestSyntheticMessageWebhookPayload_TwilioWhatsAppIsPhoneShaped(t *testing.T) {
	payload, _, err := SyntheticMessageWebhookPayload("whatsapp", map[string]string{"provider": "twilio", "from": "whatsapp:+15559990000"})
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(payload, &body))
	assert.Equal(t, "whatsapp:+15559990000", body["To"])
	assert.NotContains(t, body, "entry")
}

func TestSyntheticMessageWebhookPayload_Errors(t *testing.T) {
	_, _, err := SyntheticMessageWebhookPayload("telegram", map[string]string{"allowed_chats": "not-a-chat-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowed_chats")

	_, _, err = SyntheticMessageWebhookPayload("email", nil)
	require.Error(t, err)
}
