package messenger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

// The service test --probe payload must impersonate an allowlisted
// sender/channel so a correctly configured service accepts it instead of
// returning 403 from its own allowlist.
func TestSyntheticProbePayload_PassesAllowlistsThroughHandler(t *testing.T) {
	tests := []struct {
		name          string
		service       *core.ServiceConfig
		headers       func(body []byte) http.Header
		wantSender    string
		wantChannel   string
		wantWorkspace string
	}{
		{
			name: "telegram allowed_chats",
			service: messageService("telegram-support", "telegram", map[string]string{
				"default_action": "assistant=handle_telegram",
				"allowed_chats":  "-1009876543210",
				"reply":          "none",
			}, map[string]string{"TELEGRAM_BOT_TOKEN": "bot-token"}),
			wantChannel: "-1009876543210",
		},
		{
			name: "slack allowed_channels with bot mention",
			service: messageService("slack-support", "slack", map[string]string{
				"default_action":      "assistant=handle_slack",
				"allowed_channels":    "C0PROBE",
				"require_bot_mention": "true",
				"bot_user_id":         "U0BOT",
				"reply":               "none",
			}, map[string]string{
				"SLACK_BOT_TOKEN":      "xoxb-test",
				"SLACK_SIGNING_SECRET": "test-secret",
			}),
			headers: func(body []byte) http.Header {
				ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
				return http.Header{
					"X-Slack-Request-Timestamp": {ts},
					"X-Slack-Signature":         {"v0=" + signSlack("test-secret", ts, body)},
				}
			},
			wantChannel: "C0PROBE",
		},
		{
			name: "discord allowed_channels and allowed_guilds",
			service: messageService("discord-support", "discord", map[string]string{
				"default_action":   "assistant=handle_discord",
				"allowed_channels": "1200000000000000099",
				"allowed_guilds":   "1300000000000000099",
				"reply":            "none",
			}, nil),
			wantChannel:   "1200000000000000099",
			wantWorkspace: "1300000000000000099",
		},
		{
			name: "sms generic allowed_numbers",
			service: messageService("sms-alerts", "sms", map[string]string{
				"default_action":  "assistant=handle_sms",
				"provider":        "generic",
				"from":            "+15559990000",
				"allowed_numbers": "+15550001111,+15550002222",
				"reply":           "none",
			}, nil),
			wantSender:  "+15550001111",
			wantChannel: "+15559990000",
		},
		{
			name: "whatsapp cloud allowed_numbers and phone_number_id",
			service: messageService("wa-support", "whatsapp", map[string]string{
				"default_action":  "assistant=handle_whatsapp",
				"provider":        "whatsapp-cloud",
				"phone_number_id": "999888777666555",
				"allowed_numbers": "15551239999",
				"reply":           "none",
			}, map[string]string{"WHATSAPP_ACCESS_TOKEN": "wa-token"}),
			wantSender:  "15551239999",
			wantChannel: "999888777666555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			require.NoError(t, core.SaveService(tt.service))

			payload, identity, err := core.SyntheticMessageWebhookPayload(tt.service.Adapter, tt.service.Options)
			require.NoError(t, err)
			if tt.wantSender != "" {
				assert.Equal(t, tt.wantSender, identity.Sender)
			}
			if tt.wantChannel != "" {
				assert.Equal(t, tt.wantChannel, identity.Channel)
			}
			if tt.wantWorkspace != "" {
				assert.Equal(t, tt.wantWorkspace, identity.Workspace)
			}

			executor := &fakeActionExecutor{}
			handler := newServiceTestHandler(executor)
			req := httptest.NewRequest(http.MethodPost, "https://hooks.example.test/services/"+tt.service.ID+"/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			if tt.headers != nil {
				for key, values := range tt.headers(payload) {
					req.Header[key] = values
				}
			}
			rec := httptest.NewRecorder()

			handler.ServeServiceWebhook(rec, req, tt.service.ID, tt.service.Adapter)

			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Equal(t, "assistant", executor.input.ProfileID)
		})
	}
}

// Without any allowlist the probe keeps its self-describing synthetic
// identities and still passes the validator.
func TestSyntheticProbePayload_NoAllowlistUsesSyntheticIdentity(t *testing.T) {
	payload, identity, err := core.SyntheticMessageWebhookPayload("sms", map[string]string{"provider": "generic"})
	require.NoError(t, err)
	assert.NotEmpty(t, payload)
	assert.Equal(t, "synthetic", identity.SenderSource)
	assert.Equal(t, "synthetic", identity.ChannelSource)

	validator := msgtypes.NewServiceValidator()
	msg, err := NewNormalizer().Normalize("sms", mustJSONMap(t, payload))
	require.NoError(t, err)
	assert.NoError(t, validator.ValidateMessage(msgtypes.ServiceValidationConfig{ID: "sms-alerts", Adapter: "sms", Options: map[string]string{"provider": "generic"}}, msg))
}

func mustJSONMap(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(payload, &out))
	return out
}
