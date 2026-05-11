package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveServiceType_Alias(t *testing.T) {
	tests := []struct {
		input       string
		wantTyp     string
		wantAdapter string
	}{
		{input: "slack", wantTyp: "message", wantAdapter: "slack"},
		{input: "discord", wantTyp: "message", wantAdapter: "discord"},
		{input: "sms", wantTyp: "message", wantAdapter: "sms"},
		{input: "whatsapp", wantTyp: "message", wantAdapter: "whatsapp"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ResolveServiceType(tt.input, "")
			require.NoError(t, err)

			assert.Equal(t, tt.input, got.InputType)
			assert.Equal(t, tt.wantTyp, got.Type)
			assert.Equal(t, tt.wantAdapter, got.Adapter)
			assert.True(t, got.Aliased)
		})
	}
}

func TestResolveServiceType_CanonicalWithAdapter(t *testing.T) {
	got, err := ResolveServiceType("ticket", "github")
	require.NoError(t, err)

	assert.Equal(t, "ticket", got.Type)
	assert.Equal(t, "github", got.Adapter)
	assert.False(t, got.Aliased)
}

func TestResolveServiceType_TicketAdapterAliases(t *testing.T) {
	tests := []string{"jira", "linear", "gitlab"}
	for _, adapter := range tests {
		t.Run(adapter, func(t *testing.T) {
			got, err := ResolveServiceType(adapter, "")
			require.NoError(t, err)
			assert.Equal(t, "ticket", got.Type)
			assert.Equal(t, adapter, got.Adapter)
			assert.True(t, got.Aliased)
		})
	}
}

func TestResolveServiceType_CanonicalDefaultAdapter(t *testing.T) {
	tests := []struct {
		input       string
		wantAdapter string
	}{
		{"api", "agent-protocol"},
		{"webhook", "generic"},
		{"a2a", "jsonrpc"},
		{"events", "bus"},
		{"mobile", "aps"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ResolveServiceType(tt.input, "")
			require.NoError(t, err)

			assert.Equal(t, tt.input, got.Type)
			assert.Equal(t, tt.wantAdapter, got.Adapter)
			assert.False(t, got.Aliased)
		})
	}
}

func TestResolveServiceType_RequiresAdapter(t *testing.T) {
	_, err := ResolveServiceType("message", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires --adapter")
}

func TestResolveServiceType_AliasAdapterConflict(t *testing.T) {
	_, err := ResolveServiceType("github", "jira")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolves adapter")
}

func TestResolveServiceType_Unknown(t *testing.T) {
	_, err := ResolveServiceType("pagerduty", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service type or alias")
}

func TestSaveLoadService_RoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	service := &ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "slack",
		Profile: "assistant",
		Env: map[string]string{
			"SLACK_BOT_TOKEN": "secret:SLACK_BOT_TOKEN",
		},
		Labels: map[string]string{
			"team": "support",
		},
		Options: map[string]string{
			"allowed_channels": "C01ABC2DEF",
			"reply":            "text",
		},
	}
	require.NoError(t, SaveService(service))

	path, err := GetServicePath("support-bot")
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "type: message")
	assert.Contains(t, string(data), "adapter: slack")
	assert.Contains(t, string(data), "allowed_channels: C01ABC2DEF")

	got, err := LoadService("support-bot")
	require.NoError(t, err)
	assert.Equal(t, service, got)
}

func TestValidateServiceConfig_MessageProviderConfig(t *testing.T) {
	valid := ValidateServiceConfig(&ServiceConfig{
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
	})
	assert.True(t, valid.Valid)
	assert.Empty(t, valid.Issues)

	invalid := ValidateServiceConfig(&ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
		Options: map[string]string{
			"receive": "invalid",
			"reply":   "comment",
		},
	})
	assert.False(t, invalid.Valid)
	assert.Contains(t, invalid.Issues, "message service requires option default_action to dispatch inbound messages")
	assert.Contains(t, invalid.Issues, "message receive mode must be webhook or polling")
	assert.Contains(t, invalid.Issues, "reply mode must be text, auto, or none")
	assert.Contains(t, invalid.Issues, "missing env binding TELEGRAM_BOT_TOKEN")
}

func TestValidateServiceConfig_SMSProviderConfig(t *testing.T) {
	valid := ValidateServiceConfig(&ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Env: map[string]string{
			"TWILIO_ACCOUNT_SID": "AC123",
			"TWILIO_AUTH_TOKEN":  "twilio-token",
		},
		Options: map[string]string{
			"default_action":  "reply",
			"provider":        "twilio",
			"from":            "+15550100002",
			"allowed_numbers": "+15550100001",
			"receive":         "webhook",
			"reply":           "text",
		},
	})
	assert.True(t, valid.Valid)
	assert.Empty(t, valid.Issues)

	invalid := ValidateServiceConfig(&ServiceConfig{
		ID:      "sms-alerts",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Options: map[string]string{
			"default_action": "reply",
			"provider":       "carrier-x",
		},
	})
	assert.False(t, invalid.Valid)
	assert.Contains(t, invalid.Issues, `unsupported sms provider "carrier-x"`)
	assert.Contains(t, invalid.Issues, "sms message service requires option from")
	assert.Contains(t, invalid.Warnings, "sms service has no allowed numbers; any sender can route inbound messages")
}

func TestServiceWebhookURL_MessageService(t *testing.T) {
	got, err := ServiceWebhookURL(&ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
	}, "https://hooks.example.test/")
	require.NoError(t, err)
	assert.Equal(t, "https://hooks.example.test/services/support-bot/webhook", got)
}

func TestRecordServiceEvents_UpdateDeliveryMetadata(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	require.NoError(t, SaveService(&ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "telegram",
		Profile: "assistant",
	}))
	eventTime := time.Date(2026, 5, 11, 13, 0, 0, 0, time.UTC)

	require.NoError(t, RecordServiceInboundEvent("support-bot", ServiceEventMeta{
		At:        eventTime,
		MessageID: "msg-1",
		Platform:  "telegram",
		ChannelID: "-1001",
		SenderID:  "42",
	}))
	require.NoError(t, RecordServiceOutboundEvent("support-bot", ServiceEventMeta{
		At:        eventTime.Add(time.Second),
		MessageID: "msg-1",
		Platform:  "telegram",
		ChannelID: "-1001",
		SenderID:  "42",
		Status:    "success",
		Detail:    "ok",
	}))

	got, err := LoadService("support-bot")
	require.NoError(t, err)
	require.NotNil(t, got.LastInbound)
	assert.Equal(t, "inbound", got.LastInbound.Direction)
	assert.Equal(t, "received", got.LastInbound.Status)
	require.NotNil(t, got.LastOutbound)
	assert.Equal(t, "outbound", got.LastOutbound.Direction)
	assert.Equal(t, "success", got.LastOutbound.Status)
	require.NotNil(t, got.Delivery)
	assert.Equal(t, "healthy", got.Delivery.Health)
	assert.Empty(t, got.Delivery.LastError)
}

func TestDescribeServiceRuntime_TicketAdapters(t *testing.T) {
	tests := []struct {
		adapter     string
		wantReceive string
		wantReply   string
	}{
		{"jira", "Jira issue/comment events", "Jira comment body or status metadata"},
		{"linear", "Linear issue/comment events", "Linear comment body or status metadata"},
		{"gitlab", "GitLab issue/MR/note events", "GitLab note body or status metadata"},
	}

	for _, tt := range tests {
		t.Run(tt.adapter, func(t *testing.T) {
			got := DescribeServiceRuntime(&ServiceConfig{
				ID:      tt.adapter + "-inbox",
				Type:    "ticket",
				Adapter: tt.adapter,
				Profile: "triage",
			})

			assert.Equal(t, tt.wantReceive, got.Receives)
			assert.Equal(t, "routed profile action with normalized ticket payload", got.Executes)
			assert.Equal(t, tt.wantReply, got.Replies)
			assert.Equal(t, "component", got.Maturity)
			assert.Equal(t, []string{"/services/" + tt.adapter + "-inbox/ticket/" + tt.adapter}, got.Routes)
		})
	}
}

func TestDescribeServiceRuntime_ServiceUXSurfaceMatrix(t *testing.T) {
	tests := []struct {
		name        string
		service     *ServiceConfig
		wantReceive string
		wantExecute string
		wantReply   string
		wantMature  string
		wantRoutes  []string
	}{
		{
			name: "api",
			service: &ServiceConfig{
				ID:      "agent-api",
				Type:    "api",
				Adapter: "agent-protocol",
				Profile: "worker",
			},
			wantReceive: "Agent Protocol HTTP requests",
			wantExecute: "profile action",
			wantReply:   "JSON run/thread/store responses or SSE output stream",
			wantMature:  "ready",
			wantRoutes:  []string{"/health", "/v1/runs", "/v1/threads", "/v1/agents", "/v1/store", "/v1/skills"},
		},
		{
			name: "webhook",
			service: &ServiceConfig{
				ID:      "github-hook",
				Type:    "webhook",
				Adapter: "generic",
				Profile: "ops",
			},
			wantReceive: "HTTP POST /webhook with X-APS-Event",
			wantExecute: "mapped profile action",
			wantReply:   "status JSON, not action stdout",
			wantMature:  "status-only",
			wantRoutes:  []string{"/webhook"},
		},
		{
			name: "a2a",
			service: &ServiceConfig{
				ID:      "worker-a2a",
				Type:    "a2a",
				Adapter: "jsonrpc",
				Profile: "worker",
			},
			wantReceive: "A2A JSON-RPC task messages",
			wantExecute: "placeholder text processing",
			wantReply:   "A2A task response",
			wantMature:  "placeholder",
			wantRoutes:  []string{"aps a2a server --profile worker"},
		},
		{
			name: "client acp",
			service: &ServiceConfig{
				ID:      "dev-acp",
				Type:    "client",
				Adapter: "acp",
				Profile: "dev",
			},
			wantReceive: "stdio JSON-RPC",
			wantExecute: "ACP session, filesystem, terminal, and skill methods",
			wantReply:   "JSON-RPC responses",
			wantMature:  "ready",
			wantRoutes:  []string{"aps acp server dev"},
		},
		{
			name: "message",
			service: &ServiceConfig{
				ID:      "support-bot",
				Type:    "message",
				Adapter: "telegram",
				Profile: "assistant",
			},
			wantReceive: "HTTP POST /services/support-bot/webhook",
			wantExecute: "normalized message execution handoff",
			wantReply:   "telegram provider delivery",
			wantMature:  "ready",
			wantRoutes:  []string{"/services/support-bot/webhook"},
		},
		{
			name: "ticket",
			service: &ServiceConfig{
				ID:      "repo-inbox",
				Type:    "ticket",
				Adapter: "github",
				Profile: "maintainer",
			},
			wantReceive: "ticket events",
			wantExecute: "routed profile action with normalized ticket payload",
			wantReply:   "status metadata",
			wantMature:  "component",
			wantRoutes:  []string{"/services/repo-inbox/ticket/github"},
		},
		{
			name: "events",
			service: &ServiceConfig{
				ID:      "watcher",
				Type:    "events",
				Adapter: "bus",
				Profile: "noor",
			},
			wantReceive: "bus topics",
			wantExecute: "none",
			wantReply:   "JSONL to stdout",
			wantMature:  "observe-only",
			wantRoutes:  []string{"aps listen --profile noor"},
		},
		{
			name: "mobile",
			service: &ServiceConfig{
				ID:      "mobile-link",
				Type:    "mobile",
				Adapter: "aps",
				Profile: "assistant",
			},
			wantReceive: "pairing requests and WebSocket command messages",
			wantExecute: "pairing/token flow; command execution placeholder",
			wantReply:   "pairing responses and placeholder command acknowledgements",
			wantMature:  "placeholder",
			wantRoutes:  []string{"aps adapter pair --profile assistant"},
		},
		{
			name: "voice",
			service: &ServiceConfig{
				ID:      "voice-web",
				Type:    "voice",
				Adapter: "web",
				Profile: "assistant",
			},
			wantReceive: "component voice adapters only; no service route mounted",
			wantExecute: "backend process lifecycle and session registration only",
			wantReply:   "component-level audio/text frames",
			wantMature:  "component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeServiceRuntime(tt.service)

			assert.Equal(t, tt.wantReceive, got.Receives)
			assert.Equal(t, tt.wantExecute, got.Executes)
			assert.Equal(t, tt.wantReply, got.Replies)
			assert.Equal(t, tt.wantMature, got.Maturity)
			assert.Equal(t, tt.wantRoutes, got.Routes)
		})
	}
}

func TestDescribeServiceRuntime_MessageRuntimeMetadata(t *testing.T) {
	got := DescribeServiceRuntime(&ServiceConfig{
		ID:      "support-bot",
		Type:    "message",
		Adapter: "sms",
		Profile: "assistant",
		Options: map[string]string{
			"provider": "twilio",
			"receive":  "webhook",
			"reply":    "text",
		},
	})

	assert.Equal(t, "message-provider", got.Metadata.Runtime)
	assert.Equal(t, "twilio", got.Metadata.Provider)
	assert.Equal(t, "native provider webhook ingress", got.Metadata.Ingress)
	assert.Equal(t, "normalized message execution handoff", got.Metadata.Handoff)
	assert.Equal(t, "provider delivery interface", got.Metadata.Delivery)
	assert.Equal(t, "caller-managed retry policy", got.Metadata.Retry)
	assert.Equal(t, []string{"ingress", "normalize", "route", "execute", "deliver", "retry"}, got.Metadata.ErrorHooks)
	assert.Equal(t, "webhook", got.Metadata.ReceiveMode)
	assert.Equal(t, []string{"text", "reaction", "file"}, got.Metadata.DeliveryModes)
}
