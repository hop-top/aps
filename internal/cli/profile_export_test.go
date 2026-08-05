package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/manifest"
)

// --- rendering layer ----------------------------------------------------

func TestRenderAgentcoManifest_Full(t *testing.T) {
	p := &core.Profile{
		ID:           "acme-cto",
		DisplayName:  "Chief Technology Officer",
		Capabilities: []string{"a2a", "webhooks"},
	}
	out, err := renderAgentcoManifest(p, "# CTO\n\nOwns technical vision.")
	require.NoError(t, err)

	// Round-trips through the manifest parser.
	m, err := manifest.Parse([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, "Chief Technology Officer", m.Name)
	assert.Equal(t, "acme-cto", m.Slug)
	assert.Equal(t, []string{"a2a", "webhooks"}, m.Skills)
	assert.Equal(t, "", m.ReportsTo)
	assert.Contains(t, m.Body, "Owns technical vision.")

	// reportsTo omitted when the profile carries none — aps never
	// derives a reporting relationship of its own.
	assert.NotContains(t, out, "reportsTo")
}

func TestRenderAgentcoManifest_RoundTripsDescriptionAndReportsTo(t *testing.T) {
	p := &core.Profile{
		ID:          "acme-cto",
		DisplayName: "Chief Technology Officer",
		Description: "Owns technical vision",
		ReportsTo:   "acme-ceo",
	}
	out, err := renderAgentcoManifest(p, "")
	require.NoError(t, err)

	m, err := manifest.Parse([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, "Owns technical vision", m.Description)
	assert.Equal(t, "acme-ceo", m.ReportsTo)
}

func TestRenderAgentcoManifest_Minimal(t *testing.T) {
	p := &core.Profile{ID: "scribe", DisplayName: "scribe"}
	out, err := renderAgentcoManifest(p, "")
	require.NoError(t, err)

	m, err := manifest.Parse([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, "scribe", m.Name)
	assert.Equal(t, "scribe", m.Slug)
	assert.Empty(t, m.Skills)
	assert.Equal(t, "", m.Body)
	// Empty optional keys omitted from frontmatter entirely.
	assert.NotContains(t, out, "title:")
	assert.NotContains(t, out, "description:")
	assert.NotContains(t, out, "skills:")
}

// --- hard exclusion -----------------------------------------------------

// TestRunProfileExport_AgentcoExcludesSensitive exports a fully-loaded
// profile (secrets populated, isolation configured, gitconfig present,
// knowledge subscriptions set) and asserts the agentco output contains
// none of it.
func TestRunProfileExport_AgentcoExcludesSensitive(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	profile := core.Profile{
		DisplayName:  "Loaded Agent",
		Email:        "agent@example.com",
		Capabilities: []string{"a2a"},
		Git:          core.GitConfig{Enabled: true},
		SSH:          core.SSHConfig{Enabled: true, KeyPath: "/Users/someone/.ssh/id_ed25519"},
		Isolation:    core.IsolationConfig{Level: core.IsolationProcess},
		Knowledge:    &core.KnowledgeConfig{Subscriptions: "https://registry.internal/subs.yaml"},
	}
	require.NoError(t, core.CreateProfile("loaded", profile))

	dir, err := core.GetProfileDir("loaded")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.env"),
		[]byte("SUPER_SECRET_TOKEN=hunter2\nDB_PASSWORD=swordfish\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.md"),
		[]byte("# Loaded Agent\n\nPersona body.\n"), 0o644))

	var out strings.Builder
	require.NoError(t, runProfileExport("loaded", "agentco", &out))
	got := out.String()

	// Present: identity + skills + body.
	assert.Contains(t, got, "Loaded Agent")
	assert.Contains(t, got, "a2a")
	assert.Contains(t, got, "Persona body.")

	// HARD EXCLUSION: secrets keys/values, isolation, gitconfig,
	// machine-absolute paths, knowledge subscriptions.
	for _, banned := range []string{
		"SUPER_SECRET_TOKEN", "hunter2", "DB_PASSWORD", "swordfish",
		"isolation", "gitconfig", "[user]",
		"/Users/",
		"registry.internal", "subscriptions",
	} {
		assert.NotContains(t, got, banned, "export leaked %q", banned)
	}
}

// --- default format unchanged ------------------------------------------

func TestRunProfileExport_DefaultYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	require.NoError(t, core.CreateProfile("plain", core.Profile{DisplayName: "Plain"}))

	var out strings.Builder
	require.NoError(t, runProfileExport("plain", "", &out))
	got := out.String()

	// Native yaml dump of the profile record.
	assert.Contains(t, got, "id: plain")
	assert.Contains(t, got, "display_name: Plain")
	assert.NotContains(t, got, "---\n") // not a manifest
}

func TestRunProfileExport_UnknownFormat(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	require.NoError(t, core.CreateProfile("fmt", core.Profile{DisplayName: "F"}))

	var out strings.Builder
	err := runProfileExport("fmt", "toml", &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "toml")
}
