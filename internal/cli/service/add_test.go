package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xrr "hop.top/xrr"
	xhttp "hop.top/xrr/adapters/http"

	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
)

func TestAddCmd_DryRunShowsAliasResolution(t *testing.T) {
	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "support-bot",
		"--type", "slack",
		"--profile", "assistant",
		"--dry-run",
	})

	require.NoError(t, cmd.Execute())

	assert.Contains(t, out.String(), "input_type: slack")
	assert.Contains(t, out.String(), "type: message")
	assert.Contains(t, out.String(), "adapter: slack")
	assert.Contains(t, out.String(), "resolved_by: kit alias")
	assert.Contains(t, out.String(), "dry_run: true")
}

func TestAddCmd_PersistsCanonicalConfig(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "repo-inbox",
		"--type", "github",
		"--profile", "maintainer",
		"--label", "team=devex",
	})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "type: ticket")
	assert.Contains(t, out.String(), "adapter: github")

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "repo-inbox"})

	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "id: repo-inbox")
	assert.Contains(t, showOut.String(), "type: ticket")
	assert.Contains(t, showOut.String(), "adapter: github")
	assert.Contains(t, showOut.String(), "profile: maintainer")
}

func TestServiceStatus_MessageServiceReportsOperatorFields(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	eventTime := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Env: map[string]string{
			"TELEGRAM_BOT_TOKEN": "secret:telegram",
		},
		Options: map[string]string{
			"default_action": "reply",
			"receive":        "webhook",
			"reply":          "text",
		},
		Delivery: &core.ServiceDelivery{
			Health:    "healthy",
			Status:    "success",
			UpdatedAt: eventTime,
			RetryPolicy: &core.ServiceRetryPolicy{
				MaxAttempts: 3,
				BaseDelay:   "1s",
				MaxDelay:    "30s",
			},
			Attempts: []core.ServiceDeliveryAttempt{
				{
					At:          eventTime.Add(time.Second),
					Provider:    "telegram",
					MessageID:   "msg-1",
					ChannelID:   "-1001",
					Attempt:     1,
					MaxAttempts: 3,
					Status:      "success",
					DeliveryID:  "99",
				},
			},
		},
		LastInbound: &core.ServiceEventMeta{
			At:        eventTime,
			Direction: "inbound",
			MessageID: "msg-1",
			Platform:  "telegram",
			ChannelID: "-1001",
			SenderID:  "42",
			Status:    "received",
		},
		LastOutbound: &core.ServiceEventMeta{
			At:        eventTime.Add(time.Second),
			Direction: "outbound",
			MessageID: "msg-1",
			Platform:  "telegram",
			ChannelID: "-1001",
			SenderID:  "42",
			Status:    "success",
		},
	}))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status", "support-bot", "--base-url", "https://hooks.example.test"})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, out.String(), "webhook_url: https://hooks.example.test/services/support-bot/webhook")
	assert.Contains(t, out.String(), "delivery_health: healthy")
	assert.Contains(t, out.String(), "delivery_status: success")
	assert.Contains(t, out.String(), "retry_policy: max_attempts=3 base_delay=1s max_delay=30s")
	assert.Contains(t, out.String(), "delivery_attempt: 2026-05-11T12:00:01Z provider=telegram message_id=msg-1 channel_id=-1001 attempt=1 max_attempts=3 status=success delivery_id=99")
	assert.Contains(t, out.String(), "last_inbound: 2026-05-11T12:00:00Z message_id=msg-1 platform=telegram channel_id=-1001 sender_id=42 status=received")
	assert.Contains(t, out.String(), "last_outbound: 2026-05-11T12:00:01Z message_id=msg-1 platform=telegram channel_id=-1001 sender_id=42 status=success")
	assert.Contains(t, out.String(), "config_valid: true")
	assert.Contains(t, out.String(), "start: aps service start support-bot")
}

func TestAddCmd_TelegramPersistsWebhookSecretTokenOption(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "assistant",
		"--default-action", "reply",
		"--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
		"--webhook-secret-token-env", "TELEGRAM_WEBHOOK_SECRET",
	})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "config_valid: true")

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "support-bot"})
	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "webhook_secret_token_env: TELEGRAM_WEBHOOK_SECRET")
}

func TestServiceTest_InvalidMessageConfigFailsBeforeProbe(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
	}))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"test", "support-bot"})
	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service config is invalid")
	assert.Contains(t, out.String(), "config_valid: false")
	assert.Contains(t, out.String(), "config_issue: message service requires option default_action")
	assert.Contains(t, out.String(), "config_issue: missing env binding TELEGRAM_BOT_TOKEN")
	assert.Contains(t, out.String(), "webhook_url: http://127.0.0.1:8080/services/support-bot/webhook")
}

func TestServiceShowAndTest_FirstClassMessageProvidersUseXRRProbe(t *testing.T) {
	tests := []struct {
		name        string
		service     *core.ServiceConfig
		wantReplies string
		assertProbe func(*testing.T, *http.Request, []byte)
	}{
		{
			name: "telegram",
			service: &core.ServiceConfig{
				ID:      "telegram-support",
				Type:    "message",
				Adapter: "telegram",
				Profile: "assistant",
				Env:     map[string]string{"TELEGRAM_BOT_TOKEN": "secret:TELEGRAM_BOT_TOKEN"},
				Options: map[string]string{
					"default_action":       "assistant=handle_telegram",
					"webhook_secret_token": "telegram-secret",
					"reply":                "text",
				},
			},
			wantReplies: "replies: telegram text",
			assertProbe: func(t *testing.T, req *http.Request, _ []byte) {
				assert.Equal(t, "telegram-secret", req.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
			},
		},
		{
			name: "slack",
			service: &core.ServiceConfig{
				ID:      "slack-support",
				Type:    "message",
				Adapter: "slack",
				Profile: "assistant",
				Env: map[string]string{
					"SLACK_BOT_TOKEN":      "xoxb-test",
					"SLACK_SIGNING_SECRET": "test-secret",
				},
				Options: map[string]string{
					"default_action": "assistant=handle_slack",
					"reply":          "text",
				},
			},
			wantReplies: "replies: slack text",
			assertProbe: func(t *testing.T, req *http.Request, _ []byte) {
				assert.NotEmpty(t, req.Header.Get("X-Slack-Request-Timestamp"))
				assert.NotEmpty(t, req.Header.Get("X-Slack-Signature"))
			},
		},
		{
			name: "discord",
			service: &core.ServiceConfig{
				ID:      "discord-support",
				Type:    "message",
				Adapter: "discord",
				Profile: "assistant",
				Env:     map[string]string{"DISCORD_BOT_TOKEN": "secret:DISCORD_BOT_TOKEN"},
				Options: map[string]string{
					"default_action": "assistant=handle_discord",
					"reply":          "text",
				},
			},
			wantReplies: "replies: discord text",
		},
		{
			name: "sms",
			service: &core.ServiceConfig{
				ID:      "sms-alerts",
				Type:    "message",
				Adapter: "sms",
				Profile: "assistant",
				Env: map[string]string{
					"TWILIO_ACCOUNT_SID": "AC123",
					"TWILIO_AUTH_TOKEN":  "secret:TWILIO_AUTH_TOKEN",
				},
				Options: map[string]string{
					"default_action":  "assistant=handle_sms",
					"provider":        "twilio",
					"from":            "+15550100002",
					"allowed_numbers": "+15550100001",
					"reply":           "text",
				},
			},
			wantReplies: "replies: twilio text",
			assertProbe: func(t *testing.T, req *http.Request, body []byte) {
				form, err := url.ParseQuery(string(body))
				require.NoError(t, err)
				assert.Equal(t, msgtypes.TwilioSignature("twilio-token", req.URL.String(), form), req.Header.Get(msgtypes.TwilioSignatureHeader))
			},
		},
		{
			name: "whatsapp",
			service: &core.ServiceConfig{
				ID:      "wa-support",
				Type:    "message",
				Adapter: "whatsapp",
				Profile: "assistant",
				Env: map[string]string{
					"WHATSAPP_ACCESS_TOKEN": "secret:WHATSAPP_ACCESS_TOKEN",
					"WHATSAPP_APP_SECRET":   "secret:WHATSAPP_APP_SECRET",
				},
				Options: map[string]string{
					"default_action":  "assistant=handle_whatsapp",
					"provider":        "whatsapp-cloud",
					"phone_number_id": "123456789012345",
					"allowed_numbers": "+15551230001",
					"reply":           "text",
				},
			},
			wantReplies: "replies: whatsapp-cloud text",
			assertProbe: func(t *testing.T, req *http.Request, body []byte) {
				assert.Equal(t, "sha256="+testHMACSHA256Hex("whatsapp-secret", body), req.Header.Get(msgtypes.WhatsAppSignatureHeader))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
			t.Setenv("TWILIO_AUTH_TOKEN", "twilio-token")
			t.Setenv("WHATSAPP_APP_SECRET", "whatsapp-secret")
			require.NoError(t, core.SaveService(tt.service))

			show := newTestServiceCmd()
			var showOut bytes.Buffer
			show.SetOut(&showOut)
			show.SetErr(&showOut)
			show.SetArgs([]string{"show", tt.service.ID})
			require.NoError(t, show.Execute())
			assert.Contains(t, showOut.String(), "receives: HTTP POST /services/"+tt.service.ID+"/webhook")
			assert.Contains(t, showOut.String(), "executes: normalized message execution handoff")
			assert.Contains(t, showOut.String(), tt.wantReplies)
			assert.Contains(t, showOut.String(), "maturity: ready")

			previousClient := http.DefaultClient
			http.DefaultClient = &http.Client{Transport: xrrProbeRoundTripper{t: t, dir: t.TempDir(), assertRequest: tt.assertProbe}}
			t.Cleanup(func() { http.DefaultClient = previousClient })

			testCmd := newTestServiceCmd()
			var testOut bytes.Buffer
			testCmd.SetOut(&testOut)
			testCmd.SetErr(&testOut)
			testCmd.SetArgs([]string{"test", tt.service.ID, "--probe", "--base-url", "https://hooks.example.test"})
			require.NoError(t, testCmd.Execute())
			assert.Contains(t, testOut.String(), "config_valid: true")
			assert.Contains(t, testOut.String(), "webhook_url: https://hooks.example.test/services/"+tt.service.ID+"/webhook")
			assert.Contains(t, testOut.String(), "probe_status: 202")
			assert.Contains(t, testOut.String(), "probe_response: {\"status\":\"accepted\"}")
		})
	}
}

func TestServiceRoutes_MessageService(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	add := newTestServiceCmd()
	add.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "assistant",
		"--default-action", "reply",
		"--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
	})
	require.NoError(t, add.Execute())

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "support-bot"})
	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "receives: HTTP POST /services/support-bot/webhook")
	assert.Contains(t, showOut.String(), "executes: normalized message execution handoff")
	assert.Contains(t, showOut.String(), "maturity: ready")

	routes := newTestServiceCmd()
	var routesOut bytes.Buffer
	routes.SetOut(&routesOut)
	routes.SetErr(&routesOut)
	routes.SetArgs([]string{"routes", "support-bot"})
	require.NoError(t, routes.Execute())
	assert.Equal(t, "/services/support-bot/webhook\n", routesOut.String())
}

type xrrProbeRoundTripper struct {
	t             *testing.T
	dir           string
	assertRequest func(*testing.T, *http.Request, []byte)
}

func (rt xrrProbeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if rt.assertRequest != nil {
		rt.assertRequest(rt.t, req, body)
	}
	xreq := &xhttp.Request{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: map[string]string{"Content-Type": req.Header.Get("Content-Type")},
		Body:    string(body),
	}
	session := xrr.NewSession(xrr.ModeRecord, xrr.NewFileCassette(rt.dir))
	resp, err := session.Record(context.Background(), xhttp.NewAdapter(), xreq, func() (xrr.Response, error) {
		assert.Equal(rt.t, http.MethodPost, req.Method)
		assert.True(rt.t, strings.HasPrefix(req.URL.String(), "https://hooks.example.test/services/"))
		return &xhttp.Response{
			Status: http.StatusAccepted,
			Body:   `{"status":"accepted"}`,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	xresp, ok := resp.(*xhttp.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected xrr response %T", resp)
	}
	return &http.Response{
		StatusCode: xresp.Status,
		Status:     fmt.Sprintf("%d %s", xresp.Status, http.StatusText(xresp.Status)),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(xresp.Body)),
	}, nil
}

func testHMACSHA256Hex(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestServiceShow_ACPAdvertisesStdioOnlyRuntime(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	cmd.SetArgs([]string{
		"add", "dev-acp",
		"--type", "client",
		"--adapter", "acp",
		"--profile", "dev",
	})
	require.NoError(t, cmd.Execute())

	show := newTestServiceCmd()
	var out bytes.Buffer
	show.SetOut(&out)
	show.SetErr(&out)
	show.SetArgs([]string{"show", "dev-acp"})
	require.NoError(t, show.Execute())
	assert.Contains(t, out.String(), "receives: stdio JSON-RPC")
	assert.Contains(t, out.String(), "maturity: ready")
}

func TestServiceShow_SurfaceMaturityLabels(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "api",
			args: []string{"add", "agent-api", "--type", "api", "--profile", "worker"},
			want: []string{
				"type: api",
				"adapter: agent-protocol",
				"receives: Agent Protocol HTTP requests",
				"executes: profile action",
				"replies: JSON run/thread/store responses or SSE output stream",
				"maturity: ready",
			},
		},
		{
			name: "webhook",
			args: []string{"add", "github-hook", "--type", "webhook", "--profile", "ops"},
			want: []string{
				"type: webhook",
				"adapter: generic",
				"receives: HTTP POST /webhook with X-APS-Event",
				"executes: mapped profile action",
				"replies: status JSON, not action stdout",
				"maturity: status-only",
			},
		},
		{
			name: "a2a",
			args: []string{"add", "worker-a2a", "--type", "a2a", "--profile", "worker"},
			want: []string{
				"type: a2a",
				"adapter: jsonrpc",
				"receives: A2A JSON-RPC task messages",
				"executes: placeholder text processing",
				"replies: A2A task response",
				"maturity: placeholder",
			},
		},
		{
			name: "client acp",
			args: []string{"add", "dev-acp-matrix", "--type", "client", "--adapter", "acp", "--profile", "dev"},
			want: []string{
				"type: client",
				"adapter: acp",
				"receives: stdio JSON-RPC",
				"executes: ACP session, filesystem, terminal, and skill methods",
				"replies: JSON-RPC responses",
				"maturity: ready",
			},
		},
		{
			name: "message",
			args: []string{
				"add", "support-bot-matrix", "--type", "telegram", "--profile", "assistant",
				"--default-action", "reply", "--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
			},
			want: []string{
				"type: message",
				"adapter: telegram",
				"receives: HTTP POST /services/support-bot-matrix/webhook",
				"executes: normalized message execution handoff",
				"replies: telegram provider delivery",
				"maturity: ready",
			},
		},
		{
			name: "ticket",
			args: []string{"add", "repo-inbox", "--type", "github", "--profile", "maintainer"},
			want: []string{
				"type: ticket",
				"adapter: github",
				"receives: ticket events",
				"executes: routed profile action with normalized ticket payload",
				"replies: status metadata",
				"maturity: component",
			},
		},
		{
			name: "events",
			args: []string{"add", "watcher", "--type", "events", "--profile", "noor"},
			want: []string{
				"type: events",
				"adapter: bus",
				"receives: bus topics",
				"executes: none",
				"replies: JSONL to stdout",
				"maturity: observe-only",
			},
		},
		{
			name: "mobile",
			args: []string{"add", "mobile-link", "--type", "mobile", "--profile", "assistant"},
			want: []string{
				"type: mobile",
				"adapter: aps",
				"receives: pairing requests and WebSocket command messages",
				"executes: pairing/token flow; command execution placeholder",
				"replies: pairing responses and placeholder command acknowledgements",
				"maturity: placeholder",
			},
		},
		{
			name: "voice",
			args: []string{"add", "voice-web", "--type", "voice", "--adapter", "web", "--profile", "assistant"},
			want: []string{
				"type: voice",
				"adapter: web",
				"receives: component voice adapters only; no service route mounted",
				"executes: backend process lifecycle and session registration only",
				"replies: component-level audio/text frames",
				"maturity: component",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			add := newTestServiceCmd()
			add.SetArgs(tt.args)
			require.NoError(t, add.Execute())

			show := newTestServiceCmd()
			var out bytes.Buffer
			show.SetOut(&out)
			show.SetErr(&out)
			show.SetArgs([]string{"show", tt.args[1]})
			require.NoError(t, show.Execute())

			for _, want := range tt.want {
				assert.Contains(t, out.String(), want)
			}
		})
	}
}

func TestAddCmd_PersistsTicketAdapterOptions(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "jira-intake",
		"--type", "jira",
		"--profile", "triage",
		"--site", "https://example.atlassian.net",
		"--project", "OPS",
		"--jql", "project = OPS",
		"--default-action", "triage",
		"--reply", "comment",
	})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "type: ticket")
	assert.Contains(t, out.String(), "adapter: jira")

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "jira-intake"})

	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "options:")
	assert.Contains(t, showOut.String(), "site: https://example.atlassian.net")
	assert.Contains(t, showOut.String(), "project: OPS")
	assert.Contains(t, showOut.String(), "jql: project = OPS")
	assert.Contains(t, showOut.String(), "default_action: triage")
	assert.Contains(t, showOut.String(), "reply: comment")
}

func TestAddCmd_HelpShowsResolvedAlias(t *testing.T) {
	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"add", "--type", "slack", "--help"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Resolved:")
	assert.Contains(t, out.String(), "input_type: slack")
	assert.Contains(t, out.String(), "type: message")
	assert.Contains(t, out.String(), "adapter: slack")
}

func TestAddCmd_PersistsMessageAdapterOptions(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
	}{
		{
			name: "slack first class options",
			args: []string{
				"add", "slack-support",
				"--type", "slack",
				"--profile", "assistant",
				"--allowed-channel", "C012CHAN",
				"--signing-secret-env", "SLACK_SIGNING_SECRET",
				"--bot-user-id", "U012BOT",
				"--require-bot-mention",
				"--dedup-ttl", "24h",
				"--default-action", "assistant=handle_slack",
				"--reply", "text",
				"--env", "SLACK_BOT_TOKEN=secret:SLACK_BOT_TOKEN",
				"--env", "SLACK_SIGNING_SECRET=secret:SLACK_SIGNING_SECRET",
			},
			wantOutput: []string{
				"type: message",
				"adapter: slack",
				"allowed_channels: C012CHAN",
				"signing_secret_env: SLACK_SIGNING_SECRET",
				"bot_user_id: U012BOT",
				"require_bot_mention: true",
				"dedup_ttl: 24h",
				"default_action: assistant=handle_slack",
				"reply: text",
			},
		},
		{
			name: "discord channel options",
			args: []string{
				"add", "community-bot",
				"--type", "discord",
				"--profile", "assistant",
				"--allowed-channel", "1200000000000000002",
				"--allowed-guild", "1300000000000000003",
				"--default-action", "assistant=handle_discord",
				"--reply", "text",
				"--env", "DISCORD_BOT_TOKEN=secret:DISCORD_BOT_TOKEN",
			},
			wantOutput: []string{
				"type: message",
				"adapter: discord",
				"allowed_channels: 1200000000000000002",
				"allowed_guilds: 1300000000000000003",
				"default_action: assistant=handle_discord",
				"reply: text",
			},
		},
		{
			name: "sms phone options",
			args: []string{
				"add", "sms-alerts",
				"--type", "sms",
				"--profile", "assistant",
				"--provider", "twilio",
				"--from", "+15559870002",
				"--allowed-number", "+15551230001",
				"--reply", "text",
				"--history-turns", "8",
				"--default-action", "assistant=handle_sms",
				"--env", "TWILIO_ACCOUNT_SID=AC123",
				"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
			},
			wantOutput: []string{
				"type: message",
				"adapter: sms",
				"provider: twilio",
				"from: +15559870002",
				"allowed_numbers: +15551230001",
				"reply: text",
				"history_turns: 8",
			},
		},
		{
			name: "whatsapp phone number id",
			args: []string{
				"add", "wa-support",
				"--type", "whatsapp",
				"--profile", "assistant",
				"--provider", "whatsapp-cloud",
				"--phone-number-id", "123456789012345",
				"--allowed-number", "+15551230001",
				"--verify-token-env", "WHATSAPP_VERIFY_TOKEN",
				"--signing-secret-env", "WHATSAPP_APP_SECRET",
				"--template-name", "support_update",
				"--language-code", "en_US",
				"--default-action", "assistant=handle_whatsapp",
				"--env", "WHATSAPP_ACCESS_TOKEN=secret:WHATSAPP_ACCESS_TOKEN",
			},
			wantOutput: []string{
				"type: message",
				"adapter: whatsapp",
				"provider: whatsapp-cloud",
				"phone_number_id: 123456789012345",
				"allowed_numbers: +15551230001",
				"verify_token_env: WHATSAPP_VERIFY_TOKEN",
				"signing_secret_env: WHATSAPP_APP_SECRET",
				"template_name: support_update",
				"language_code: en_US",
			},
		},
		{
			name: "email sender allowlist",
			args: []string{
				"add", "mail-inbox",
				"--type", "message",
				"--adapter", "email",
				"--profile", "assistant",
				"--allowed-sender", "alice@example.com",
				"--allowed-sender", "*@partner.org",
				"--default-action", "assistant=handle_email",
				"--reply", "text",
			},
			wantOutput: []string{
				"type: message",
				"adapter: email",
				"allowed_senders: alice@example.com,*@partner.org",
				"default_action: assistant=handle_email",
				"replies: email text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestServiceCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tt.args)
			require.NoError(t, cmd.Execute())

			show := newTestServiceCmd()
			var showOut bytes.Buffer
			show.SetOut(&showOut)
			show.SetErr(&showOut)
			show.SetArgs([]string{"show", tt.args[1]})
			require.NoError(t, show.Execute())

			for _, want := range tt.wantOutput {
				assert.Contains(t, showOut.String(), want)
			}
		})
	}
}

func TestAddCmd_InvalidConfigIsNotPersisted(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "sms-broken",
		"--type", "sms",
		"--profile", "assistant",
		"--default-action", "reply",
		"--reply", "text",
	})
	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service config is invalid")
	assert.Contains(t, out.String(), "config_valid: false")
	assert.Contains(t, out.String(), "config_issue: sms message service requires option provider")
	assert.NotContains(t, out.String(), "saved:")

	path, err := core.GetServicePath("sms-broken")
	require.NoError(t, err)
	assert.NoFileExists(t, path)
	_, err = core.LoadService("sms-broken")
	require.Error(t, err)
}

func TestAddCmd_ExistingServiceRequiresForce(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	first := newTestServiceCmd()
	first.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "assistant",
		"--default-action", "reply",
		"--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
	})
	require.NoError(t, first.Execute())

	second := newTestServiceCmd()
	var out bytes.Buffer
	second.SetOut(&out)
	second.SetErr(&out)
	second.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "other",
		"--default-action", "triage",
		"--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
	})
	err := second.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `service "support-bot" already exists`)
	assert.Contains(t, err.Error(), "--force")
	assert.NotContains(t, out.String(), "saved:")

	service, err := core.LoadService("support-bot")
	require.NoError(t, err)
	assert.Equal(t, "assistant", service.Profile)
	assert.Equal(t, "reply", service.Options["default_action"])

	forced := newTestServiceCmd()
	var forcedOut bytes.Buffer
	forced.SetOut(&forcedOut)
	forced.SetErr(&forcedOut)
	forced.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "other",
		"--default-action", "triage",
		"--env", "TELEGRAM_BOT_TOKEN=secret:TELEGRAM_BOT_TOKEN",
		"--force",
	})
	require.NoError(t, forced.Execute())
	assert.Contains(t, forcedOut.String(), "saved:")

	service, err = core.LoadService("support-bot")
	require.NoError(t, err)
	assert.Equal(t, "other", service.Profile)
	assert.Equal(t, "triage", service.Options["default_action"])
}

func TestAddCmd_ExistingServiceDryRunReportsConflict(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "repo-inbox",
		Type:    "ticket",
		Adapter: "github",
		Profile: "maintainer",
	}))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"add", "repo-inbox", "--type", "github", "--profile", "other", "--dry-run"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `service "repo-inbox" already exists`)

	service, err := core.LoadService("repo-inbox")
	require.NoError(t, err)
	assert.Equal(t, "maintainer", service.Profile)
}

func TestAddCmd_GenericWebhookAuthFlags(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "mail-inbox",
		"--type", "message",
		"--adapter", "email",
		"--profile", "assistant",
		"--allowed-sender", "alice@example.com",
		"--default-action", "assistant=handle_email",
		"--reply", "text",
		"--auth-scheme", "bearer",
		"--auth-token-env", "MAIL_BRIDGE_TOKEN",
		"--option", "timestamp_header=X-Mail-Timestamp",
		"--option", "require_replay_check=true",
	})
	require.NoError(t, cmd.Execute(), out.String())
	assert.Contains(t, out.String(), "config_valid: true")
	assert.NotContains(t, out.String(), "email service has no webhook auth")

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "mail-inbox"})
	require.NoError(t, show.Execute())
	for _, want := range []string{
		"  auth_scheme: bearer",
		"  auth_token_env: MAIL_BRIDGE_TOKEN",
		"  timestamp_header: X-Mail-Timestamp",
		"  require_replay_check: true",
		"auth:",
		"  scheme: bearer",
		"  header: Authorization",
		"  token_env: MAIL_BRIDGE_TOKEN",
		"  timestamp_header: X-Mail-Timestamp (tolerance 5m0s)",
		"  replay_id_header: X-APS-Delivery-ID",
	} {
		assert.Contains(t, showOut.String(), want)
	}
}

func TestAddCmd_SignatureSecretEnvFlagFeedsGenericAuth(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := newTestServiceCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"add", "mail-inbox",
		"--type", "message",
		"--adapter", "email",
		"--profile", "assistant",
		"--default-action", "assistant=handle_email",
		"--auth-scheme", "hmac-sha256",
		"--signature-secret-env", "MAIL_BRIDGE_SECRET",
	})
	require.NoError(t, cmd.Execute(), out.String())

	service, err := core.LoadService("mail-inbox")
	require.NoError(t, err)
	assert.Equal(t, "hmac-sha256", service.Options["auth_scheme"])
	assert.Equal(t, "MAIL_BRIDGE_SECRET", service.Options["signature_secret_env"])
	_, hasSlackOnly := service.Options["signing_secret_env"]
	assert.False(t, hasSlackOnly, "--signature-secret-env must not land in the Slack-only signing_secret_env option")

	show := newTestServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "mail-inbox"})
	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "  scheme: hmac-sha256")
	assert.Contains(t, showOut.String(), "  header: X-APS-Signature")
	assert.Contains(t, showOut.String(), "  signature_secret_env: MAIL_BRIDGE_SECRET")
}

func TestAddCmd_OptionFlagRejectsMalformedAndYieldsToNamedFlags(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	bad := newTestServiceCmd()
	bad.SetArgs([]string{
		"add", "mail-inbox",
		"--type", "message",
		"--adapter", "email",
		"--profile", "assistant",
		"--default-action", "assistant=handle_email",
		"--option", "auth_scheme",
	})
	err := bad.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--option must be KEY=VALUE")

	precedence := newTestServiceCmd()
	precedence.SetArgs([]string{
		"add", "mail-inbox",
		"--type", "message",
		"--adapter", "email",
		"--profile", "assistant",
		"--default-action", "assistant=handle_email",
		"--option", "auth_scheme=token",
		"--option", "auth_token_env=FROM_OPTION",
		"--auth-token-env", "FROM_FLAG",
	})
	require.NoError(t, precedence.Execute())
	service, err := core.LoadService("mail-inbox")
	require.NoError(t, err)
	assert.Equal(t, "token", service.Options["auth_scheme"])
	assert.Equal(t, "FROM_FLAG", service.Options["auth_token_env"])
}

func TestServiceShow_AuthSummaryForProviderHooksAndNone(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "slack-support",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Env: map[string]string{
			"SLACK_BOT_TOKEN":      "secret:SLACK_BOT_TOKEN",
			"SLACK_SIGNING_SECRET": "secret:SLACK_SIGNING_SECRET",
		},
		Options: map[string]string{"default_action": "assistant=handle_slack"},
	}))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_ACCOUNT_SID": "AC123",
			"TWILIO_AUTH_TOKEN":  "secret:TWILIO_AUTH_TOKEN",
		},
		Options: map[string]string{"default_action": "assistant=handle_sms", "provider": "twilio", "from": "+15550100002"},
	}))
	require.NoError(t, core.SaveService(&core.ServiceConfig{
		ID:      "mail-open",
		Type:    "message",
		Adapter: "email",
		Profile: "assistant",
		Options: map[string]string{"default_action": "assistant=handle_email"},
	}))

	show := func(id string) string {
		cmd := newTestServiceCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"show", id})
		require.NoError(t, cmd.Execute())
		return out.String()
	}

	slack := show("slack-support")
	assert.Contains(t, slack, "auth:\n  scheme: slack-signing-secret\n")
	assert.Contains(t, slack, "  header: X-Slack-Signature")
	assert.Contains(t, slack, "  signature_secret_env: SLACK_SIGNING_SECRET")
	assert.Contains(t, slack, "  timestamp_header: X-Slack-Request-Timestamp")

	sms := show("sms-alerts")
	assert.Contains(t, sms, "auth:\n  scheme: twilio-signature\n")

	open := show("mail-open")
	assert.Contains(t, open, "auth: none\n")
}
