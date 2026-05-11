package core

import (
	"os"
	"path/filepath"
	"testing"

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
	got, err := ResolveServiceType("a2a", "")
	require.NoError(t, err)

	assert.Equal(t, "a2a", got.Type)
	assert.Equal(t, "jsonrpc", got.Adapter)
	assert.False(t, got.Aliased)
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
