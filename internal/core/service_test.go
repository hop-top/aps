package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveServiceType_Alias(t *testing.T) {
	got, err := ResolveServiceType("slack", "")
	require.NoError(t, err)

	assert.Equal(t, "slack", got.InputType)
	assert.Equal(t, "message", got.Type)
	assert.Equal(t, "slack", got.Adapter)
	assert.True(t, got.Aliased)
}

func TestResolveServiceType_CanonicalWithAdapter(t *testing.T) {
	got, err := ResolveServiceType("ticket", "github")
	require.NoError(t, err)

	assert.Equal(t, "ticket", got.Type)
	assert.Equal(t, "github", got.Adapter)
	assert.False(t, got.Aliased)
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
	}
	require.NoError(t, SaveService(service))

	path, err := GetServicePath("support-bot")
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "type: message")
	assert.Contains(t, string(data), "adapter: slack")

	got, err := LoadService("support-bot")
	require.NoError(t, err)
	assert.Equal(t, service, got)
}
