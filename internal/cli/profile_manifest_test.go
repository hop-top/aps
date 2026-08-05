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
	require.NoError(t, runManifestImport(t.Context(), path, "", false, false, &out, &errOut))

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
	// Supervisor first, so reportsTo resolves and no --force is needed.
	require.NoError(t, core.CreateProfileWithContext(
		t.Context(), "acme-ceo", core.Profile{DisplayName: "CEO"}))

	path := writeManifestFile(t, t.TempDir(), doc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, false, &out, &errOut))

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

const reportsToManifestDoc = `---
name: cto
slug: acme-cto
reportsTo: acme-ceo
---
`

// A dangling reportsTo fails the import outright, and must not leave a
// partially-created profile behind.
func TestRunManifestImport_DanglingReportsToFails(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	path := writeManifestFile(t, t.TempDir(), reportsToManifestDoc)

	var out, errOut strings.Builder
	err := runManifestImport(t.Context(), path, "", false, false, &out, &errOut)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "acme-ceo")
	assert.Contains(t, err.Error(), "--force")

	_, loadErr := core.LoadProfile("acme-cto")
	assert.Error(t, loadErr, "failed import must write nothing")
}

// --force downgrades the failure to a warning and keeps the value, for
// importing a hierarchy before its supervisors exist.
func TestRunManifestImport_DanglingReportsToForceWarnsAndKeeps(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	path := writeManifestFile(t, t.TempDir(), reportsToManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, true, &out, &errOut))

	assert.Contains(t, errOut.String(), "acme-ceo")
	assert.Contains(t, errOut.String(), "--force")

	p, err := core.LoadProfile("acme-cto")
	require.NoError(t, err)
	assert.Equal(t, "acme-ceo", p.ReportsTo, "forced import must still persist the value")
}

func TestRunManifestImport_ResolvableReportsToSucceedsSilently(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	// Supervising profile exists first → no warning, no --force needed.
	require.NoError(t, core.CreateProfileWithContext(
		t.Context(), "acme-ceo", core.Profile{DisplayName: "CEO"}))

	path := writeManifestFile(t, t.TempDir(), reportsToManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, false, &out, &errOut))

	assert.NotContains(t, errOut.String(), "reportsTo")

	p, err := core.LoadProfile("acme-cto")
	require.NoError(t, err)
	assert.Equal(t, "acme-ceo", p.ReportsTo)
}

func TestRunManifestImport_NoReportsToNeedsNoForce(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	path := writeManifestFile(t, t.TempDir(), importManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", false, false, &out, &errOut))

	assert.NotContains(t, errOut.String(), "reportsTo")
}

// Dry-run reports the same verdict a real import would: it fails rather
// than previewing a profile that could not actually be imported.
func TestRunManifestImport_DryRunFailsOnDanglingReportsTo(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	path := writeManifestFile(t, t.TempDir(), reportsToManifestDoc)

	var out, errOut strings.Builder
	err := runManifestImport(t.Context(), path, "", true, false, &out, &errOut)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	assert.Empty(t, out.String(), "no preview when the import would fail")
}

func TestRunManifestImport_DryRunForcePreviewsWithWarning(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())
	path := writeManifestFile(t, t.TempDir(), reportsToManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", true, true, &out, &errOut))

	assert.Contains(t, errOut.String(), "--force")
	assert.Contains(t, out.String(), "reports_to: acme-ceo")

	_, loadErr := core.LoadProfile("acme-cto")
	assert.Error(t, loadErr, "dry-run must not write the profile")
}

func TestRunManifestImport_DryRunWritesNothing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)

	path := writeManifestFile(t, t.TempDir(), importManifestDoc)

	var out, errOut strings.Builder
	require.NoError(t, runManifestImport(t.Context(), path, "", true, false, &out, &errOut))

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
	require.NoError(t, runManifestImport(t.Context(), path, "custom-id", false, false, &out, &errOut))

	_, err := core.LoadProfile("custom-id")
	require.NoError(t, err)
}

func TestIsAgentManifestPath(t *testing.T) {
	assert.True(t, isAgentManifestPath("AGENTS.md"))
	assert.True(t, isAgentManifestPath("/x/y/role.md"))
	assert.False(t, isAgentManifestPath("cto.aps-profile.yaml"))
	assert.False(t, isAgentManifestPath("bundle.yml"))
}
