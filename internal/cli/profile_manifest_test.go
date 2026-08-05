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

// --- mapping layer ------------------------------------------------------

func TestManifestToProfile_TitlePreferredOverName(t *testing.T) {
	m := &manifest.AgentManifest{Name: "cto", Title: "Chief Technology Officer"}
	id, p := manifestToProfile(m, "")
	assert.Equal(t, "cto", id) // slug empty → slugified name
	assert.Equal(t, "Chief Technology Officer", p.DisplayName)
}

func TestManifestToProfile_NameWhenNoTitle(t *testing.T) {
	m := &manifest.AgentManifest{Name: "scribe"}
	_, p := manifestToProfile(m, "")
	assert.Equal(t, "scribe", p.DisplayName)
}

func TestManifestToProfile_SlugWins(t *testing.T) {
	m := &manifest.AgentManifest{Name: "cto", Slug: "acme-cto"}
	id, _ := manifestToProfile(m, "")
	assert.Equal(t, "acme-cto", id)
}

func TestManifestToProfile_IDOverrideWins(t *testing.T) {
	m := &manifest.AgentManifest{Name: "cto", Slug: "acme-cto"}
	id, _ := manifestToProfile(m, "my-cto")
	assert.Equal(t, "my-cto", id)
}

func TestManifestToProfile_SlugifyNameFallback(t *testing.T) {
	m := &manifest.AgentManifest{Name: "Chief Of Staff"}
	id, _ := manifestToProfile(m, "")
	assert.Equal(t, "chief-of-staff", id)
}

func TestManifestToProfile_CarriesDescriptionAndReportsTo(t *testing.T) {
	m := &manifest.AgentManifest{
		Name:        "cto",
		Description: "Owns technical vision",
		ReportsTo:   "ceo",
	}
	_, p := manifestToProfile(m, "")
	assert.Equal(t, "Owns technical vision", p.Description)
	assert.Equal(t, "ceo", p.ReportsTo)
}

func TestManifestToProfile_OmittedDescriptionAndReportsToStayEmpty(t *testing.T) {
	m := &manifest.AgentManifest{Name: "scribe"}
	_, p := manifestToProfile(m, "")
	assert.Empty(t, p.Description)
	assert.Empty(t, p.ReportsTo)
}

func TestManifestToProfile_NoSecretsNoIsolation(t *testing.T) {
	m := &manifest.AgentManifest{Name: "cto", Title: "CTO"}
	_, p := manifestToProfile(m, "")
	assert.Equal(t, core.IsolationConfig{}, p.Isolation)
	assert.Empty(t, p.Capabilities) // linked separately, post-partition
}

func TestPartitionManifestSkills(t *testing.T) {
	exists := func(name string) bool { return name == "web" || name == "docs" }
	link, skip := partitionManifestSkills([]string{"web", "unknown-a", "docs", "unknown-b"}, exists)
	assert.Equal(t, []string{"web", "docs"}, link)
	assert.Equal(t, []string{"unknown-a", "unknown-b"}, skip)
}

func TestPartitionManifestSkills_Empty(t *testing.T) {
	link, skip := partitionManifestSkills(nil, func(string) bool { return true })
	assert.Empty(t, link)
	assert.Empty(t, skip)
}

func TestSlugifyManifestName(t *testing.T) {
	cases := map[string]string{
		"CTO":            "cto",
		"Chief Of Staff": "chief-of-staff",
		"a_b.c":          "a-b-c",
		"  spaced  ":     "spaced",
		"Multi   Gap":    "multi-gap",
	}
	for in, want := range cases {
		assert.Equal(t, want, slugifyManifestName(in), "input %q", in)
	}
}

// --- command layer ------------------------------------------------------

func writeManifestFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "AGENTS.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

const importManifestDoc = `---
name: cto
title: Chief Technology Officer
slug: acme-cto
skills:
  - a2a
  - not-a-real-capability
---

# CTO

Owns technical vision.
`

func TestRunManifestImport_CreatesProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	path := writeManifestFile(t, t.TempDir(), importManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, &out, &errOut))

	p, err := core.LoadProfile("acme-cto")
	require.NoError(t, err)
	assert.Equal(t, "Chief Technology Officer", p.DisplayName)
	// a2a is builtin → linked; unknown skipped with warning.
	assert.Contains(t, p.Capabilities, "a2a")
	assert.NotContains(t, p.Capabilities, "not-a-real-capability")
	assert.Contains(t, errOut.String(), "not-a-real-capability")

	// Body lands in notes.md (the profile's markdown side-car).
	dir, err := core.GetProfileDir("acme-cto")
	require.NoError(t, err)
	notes, err := os.ReadFile(filepath.Join(dir, "notes.md"))
	require.NoError(t, err)
	assert.Contains(t, string(notes), "Owns technical vision.")

	// Persisted import keeps the affirmative message.
	assert.Contains(t, out.String(), "Profile 'acme-cto' imported from manifest")
}

// Manifest → profile → manifest must not drop description/reportsTo;
// before they were persisted, a round-trip silently lost both.
func TestRunManifestImport_RoundTripPreservesDescriptionAndReportsTo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	const doc = `---
name: cto
title: Chief Technology Officer
slug: acme-cto
description: Owns technical vision
reportsTo: acme-ceo
---

# CTO
`
	path := writeManifestFile(t, t.TempDir(), doc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, &out, &errOut))

	p, err := core.LoadProfile("acme-cto")
	require.NoError(t, err)
	assert.Equal(t, "Owns technical vision", p.Description)
	assert.Equal(t, "acme-ceo", p.ReportsTo)

	// Export back out and reparse: both fields survive the full cycle.
	var exported strings.Builder
	require.NoError(t, runProfileExport("acme-cto", "agentco", &exported))

	m, err := manifest.Parse([]byte(exported.String()))
	require.NoError(t, err)
	assert.Equal(t, "Owns technical vision", m.Description)
	assert.Equal(t, "acme-ceo", m.ReportsTo)
}

func TestRunManifestImport_DryRunWritesNothing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	path := writeManifestFile(t, t.TempDir(), importManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", true, &out, &errOut))

	// Nothing created.
	_, err := core.LoadProfile("acme-cto")
	require.Error(t, err)

	// Preview mentions the profile yaml, the link, and the skip.
	assert.Contains(t, out.String(), "acme-cto")
	assert.Contains(t, out.String(), "a2a")
	assert.Contains(t, out.String(), "not-a-real-capability")

	// Preview phrasing only — the persisted-import message must not
	// leak into dry-run output.
	assert.Contains(t, out.String(), "dry-run — nothing written")
	assert.Contains(t, out.String(), "would link capabilities")
	assert.NotContains(t, out.String(), "imported from manifest")
}

func TestRunManifestImport_IDOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	path := writeManifestFile(t, t.TempDir(), importManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "custom-id", false, &out, &errOut))

	_, err := core.LoadProfile("custom-id")
	require.NoError(t, err)
}

func TestIsAgentManifestPath(t *testing.T) {
	assert.True(t, isAgentManifestPath("AGENTS.md"))
	assert.True(t, isAgentManifestPath("/x/y/role.md"))
	assert.False(t, isAgentManifestPath("cto.aps-profile.yaml"))
	assert.False(t, isAgentManifestPath("bundle.yml"))
}
