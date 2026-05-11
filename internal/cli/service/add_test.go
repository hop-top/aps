package service

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCmd_DryRunShowsAliasResolution(t *testing.T) {
	cmd := NewServiceCmd()
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

	cmd := NewServiceCmd()
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

	show := NewServiceCmd()
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

func TestServiceRoutes_MessageService(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	add := NewServiceCmd()
	add.SetArgs([]string{
		"add", "support-bot",
		"--type", "telegram",
		"--profile", "assistant",
	})
	require.NoError(t, add.Execute())

	show := NewServiceCmd()
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&showOut)
	show.SetArgs([]string{"show", "support-bot"})
	require.NoError(t, show.Execute())
	assert.Contains(t, showOut.String(), "receives: HTTP POST /services/support-bot/webhook")
	assert.Contains(t, showOut.String(), "executes: profile action")
	assert.Contains(t, showOut.String(), "maturity: ready")

	routes := NewServiceCmd()
	var routesOut bytes.Buffer
	routes.SetOut(&routesOut)
	routes.SetErr(&routesOut)
	routes.SetArgs([]string{"routes", "support-bot"})
	require.NoError(t, routes.Execute())
	assert.Equal(t, "/services/support-bot/webhook\n", routesOut.String())
}

func TestServiceShow_ACPAdvertisesStdioOnlyRuntime(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := NewServiceCmd()
	cmd.SetArgs([]string{
		"add", "dev-acp",
		"--type", "client",
		"--adapter", "acp",
		"--profile", "dev",
	})
	require.NoError(t, cmd.Execute())

	show := NewServiceCmd()
	var out bytes.Buffer
	show.SetOut(&out)
	show.SetErr(&out)
	show.SetArgs([]string{"show", "dev-acp"})
	require.NoError(t, show.Execute())
	assert.Contains(t, out.String(), "receives: stdio JSON-RPC")
	assert.Contains(t, out.String(), "maturity: ready")
}

func TestAddCmd_PersistsTicketAdapterOptions(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cmd := NewServiceCmd()
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

	show := NewServiceCmd()
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
	cmd := NewServiceCmd()
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
			name: "discord channel options",
			args: []string{
				"add", "community-bot",
				"--type", "discord",
				"--profile", "assistant",
				"--allowed-channel", "1200000000000000002",
				"--default-action", "assistant=handle_discord",
				"--reply", "text",
			},
			wantOutput: []string{
				"type: message",
				"adapter: discord",
				"allowed_channels: 1200000000000000002",
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
			},
			wantOutput: []string{
				"type: message",
				"adapter: sms",
				"provider: twilio",
				"from: +15559870002",
				"allowed_numbers: +15551230001",
				"reply: text",
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
			},
			wantOutput: []string{
				"type: message",
				"adapter: whatsapp",
				"provider: whatsapp-cloud",
				"phone_number_id: 123456789012345",
				"allowed_numbers: +15551230001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewServiceCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tt.args)
			require.NoError(t, cmd.Execute())

			show := NewServiceCmd()
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
