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
