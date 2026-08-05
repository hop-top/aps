package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/manifest"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    manifest.AgentManifest
		wantErr error
	}{
		{
			name:    "full manifest",
			fixture: "full.md",
			want: manifest.AgentManifest{
				Name:        "cto",
				Title:       "Chief Technology Officer",
				Slug:        "acme-cto",
				Description: "Owns technical strategy and architecture decisions.",
				ReportsTo:   "ceo",
				Skills:      []string{"architecture-patterns", "code-review-excellence", "roadmap-planning"},
				Body:        "# CTO\n\nOwns the technical vision.\n\n## Responsibilities\n\n- Architecture reviews\n- Hiring plan",
			},
		},
		{
			name:    "minimal manifest (name only)",
			fixture: "minimal.md",
			want: manifest.AgentManifest{
				Name: "scribe",
				Body: "Body only.",
			},
		},
		{
			name:    "no frontmatter",
			fixture: "no-frontmatter.md",
			wantErr: manifest.ErrMissingFrontmatter,
		},
		{
			name:    "reportsTo null",
			fixture: "nullreports.md",
			want: manifest.AgentManifest{
				Name:  "ceo",
				Title: "Chief Executive Officer",
				Body:  "Top of the org chart.",
			},
		},
		{
			name:    "skills flow list",
			fixture: "skills-list.md",
			want: manifest.AgentManifest{
				Name:   "marketer",
				Skills: []string{"copywriting", "seo-audit"},
				Body:   "Growth role.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manifest.Parse(fixture(t, tt.fixture))
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestParse_MissingName(t *testing.T) {
	got, err := manifest.Parse([]byte("---\ntitle: No Name\n---\nbody\n"))
	require.Error(t, err)
	assert.ErrorIs(t, err, manifest.ErrMissingName)
	assert.Nil(t, got)
}

func TestParse_UnterminatedFrontmatter(t *testing.T) {
	got, err := manifest.Parse([]byte("---\nname: x\nno closing delimiter\n"))
	require.Error(t, err)
	assert.ErrorIs(t, err, manifest.ErrMissingFrontmatter)
	assert.Nil(t, got)
}

// CRLF input must parse identically to LF. Built in-code rather than as
// a fixture: a CRLF file on disk is itself rewritten by checkout/editor
// line-ending settings, so it would not reliably carry the \r.
func TestParse_CRLFMatchesLF(t *testing.T) {
	const lf = "---\nname: cto\ntitle: CTO\n---\n\n# CTO\n\nOwns the vision.\n"

	fromLF, err := manifest.Parse([]byte(lf))
	require.NoError(t, err)

	fromCRLF, err := manifest.Parse([]byte(strings.ReplaceAll(lf, "\n", "\r\n")))
	require.NoError(t, err)

	assert.Equal(t, fromLF, fromCRLF)
	assert.NotContains(t, fromCRLF.Body, "\r", "carriage returns must not survive into the body")
}

func TestParseFile(t *testing.T) {
	got, err := manifest.ParseFile(filepath.Join("testdata", "minimal.md"))
	require.NoError(t, err)
	assert.Equal(t, "scribe", got.Name)

	_, err = manifest.ParseFile(filepath.Join("testdata", "does-not-exist.md"))
	require.Error(t, err)
}
