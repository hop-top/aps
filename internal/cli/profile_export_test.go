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
	// Pinned byte-for-byte: the supported list derives from the format
	// registry, so a row added there must surface here.
	assert.Equal(t,
		`unknown export format "toml" (supported: agentco, or omit for native yaml)`,
		err.Error())
}

// --- format registry ----------------------------------------------------

// TestProfileExportFormats_RowsComplete pins the registry↔handler
// contract. Dispatch derives from profileExportFormats, so a renderer
// cannot exist without a row; this test closes the other direction — a
// row cannot land without a renderer or without the doc fields the
// generated enumeration needs.
func TestProfileExportFormats_RowsComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range profileExportFormats {
		require.NotNil(t, f.render, "format %q declared without a renderer", f.Name)
		require.NotEmpty(t, f.Summary, "format %q declared without a summary", f.Name)
		require.NotEmpty(t, f.Emits, "format %q declared without an emits description", f.Name)
		require.False(t, seen[f.Name], "format %q declared twice", f.Name)
		seen[f.Name] = true
	}
	require.True(t, seen[""], "registry must declare the flag-omitted native yaml default")
}

// TestProfileExportFormats_EveryRowExportsEndToEnd drives
// runProfileExport once per registry row, so a declared format the
// command cannot actually render fails here rather than in an
// operator's hands.
func TestProfileExportFormats_EveryRowExportsEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	require.NoError(t, core.CreateProfile("rows", core.Profile{DisplayName: "Rows"}))

	for _, f := range profileExportFormats {
		name := f.Name
		if name == "" {
			name = "(default)"
		}
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			require.NoError(t, runProfileExport("rows", f.Name, &out))
			assert.NotEmpty(t, out.String())
		})
	}
}

// TestProfileExportFormats_ErrorListsEveryNamedRow asserts the
// unknown-format error enumerates exactly the named registry rows, so
// the operator-facing supported list can never drift from the table.
func TestProfileExportFormats_ErrorListsEveryNamedRow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	require.NoError(t, core.CreateProfile("list", core.Profile{DisplayName: "L"}))

	var out strings.Builder
	err := runProfileExport("list", "no-such-format", &out)
	require.Error(t, err)
	for _, f := range profileExportFormats {
		if f.Name != "" {
			assert.Contains(t, err.Error(), f.Name)
		}
	}
}
