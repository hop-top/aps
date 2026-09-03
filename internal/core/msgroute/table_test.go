package msgroute

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_InlineRoutesWithTerminal(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
			{Match: "+1555*", Action: "sales"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}
	table, err := Load(cfg, Options{Platform: "whatsapp", DefaultProfile: "assistant"})
	require.NoError(t, err)
	require.NotNil(t, table)

	routes := table.Routes()
	require.Len(t, routes, 3)
	assert.Equal(t, "acme", routes[0].Profile)
	assert.Equal(t, "assistant", routes[1].Profile, "route without profile inherits the default profile")
	assert.Equal(t, "triage", routes[2].Profile)
	assert.True(t, table.HasTerminal())
}

func TestLoad_RejectsTableWithoutTerminalRoute(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
		},
	}
	_, err := Load(cfg, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingTerminalRoute), "got %v", err)
}

func TestLoad_RejectsEmptyRoutes(t *testing.T) {
	_, err := Load(&Config{}, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingTerminalRoute), "got %v", err)
}

func TestLoad_RejectsRoutesAfterTerminal(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Match: "unknown", Profile: "triage", Action: "triage"},
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
		},
	}
	_, err := Load(cfg, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestLoad_RejectsDuplicateTerminal(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Match: "unknown", Profile: "triage", Action: "triage"},
			{Match: "UNKNOWN", Profile: "triage", Action: "triage"},
		},
	}
	_, err := Load(cfg, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestLoad_RouteFieldValidation(t *testing.T) {
	tests := []struct {
		name    string
		route   Route
		opts    Options
		wantErr string
	}{
		{
			name:    "missing match",
			route:   Route{Profile: "acme", Action: "inbox"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "match is required",
		},
		{
			name:    "missing action",
			route:   Route{Match: "+15551234567", Profile: "acme"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "action is required",
		},
		{
			name:    "action carries profile separator",
			route:   Route{Match: "+15551234567", Profile: "acme", Action: "acme=inbox"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "action must be a plain action name",
		},
		{
			name:    "no profile anywhere",
			route:   Route{Match: "+15551234567", Action: "inbox"},
			opts:    Options{},
			wantErr: "profile is required",
		},
		{
			name:    "bad glob pattern",
			route:   Route{Match: "+1555[", Profile: "acme", Action: "inbox"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "invalid glob",
		},
		{
			name:    "org selector without contacts",
			route:   Route{Match: "org:acme", Profile: "acme", Action: "inbox"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "requires a contacts source",
		},
		{
			name:    "empty org selector",
			route:   Route{Match: "org:", Profile: "acme", Action: "inbox"},
			opts:    Options{DefaultProfile: "assistant"},
			wantErr: "org selector requires a pattern",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Routes: []Route{tt.route, {Match: "unknown", Profile: "triage", Action: "triage"}}}
			tt.opts.Platform = "sms"
			_, err := Load(cfg, tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_AggregatesAllRouteErrors(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Profile: "acme", Action: "inbox"},
			{Match: "+15551234567", Profile: "acme"},
		},
	}
	_, err := Load(cfg, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "match is required")
	assert.Contains(t, err.Error(), "action is required")
	assert.True(t, errors.Is(err, ErrMissingTerminalRoute))
}

func TestLoad_ExternalFileRelativeToBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "acme-routes.yaml"), `
routes:
  - match: "+15551234567"
    profile: acme
    action: inbox
  - match: unknown
    profile: triage
    action: triage
`)
	table, err := Load(&Config{File: "acme-routes.yaml"}, Options{Platform: "sms", DefaultProfile: "assistant", BaseDir: baseDir})
	require.NoError(t, err)
	require.Len(t, table.Routes(), 2)
	assert.Equal(t, filepath.Join(baseDir, "acme-routes.yaml"), table.Source())
}

func TestLoad_ExternalFileTildeExpansion(t *testing.T) {
	home := t.TempDir()
	// os.UserHomeDir() (used for "~" expansion) reads $HOME on POSIX but
	// %USERPROFILE% on Windows — HOME alone does not isolate this test
	// there, so ~ still expands to the real home dir on the runner.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeFile(t, filepath.Join(home, "routes.yaml"), `
routes:
  - match: unknown
    profile: triage
    action: triage
`)
	table, err := Load(&Config{File: "~/routes.yaml"}, Options{Platform: "sms", DefaultProfile: "assistant"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "routes.yaml"), table.Source())
}

func TestLoad_ExternalFileErrors(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "nested.yaml"), `
file: other.yaml
routes:
  - match: unknown
    profile: triage
    action: triage
`)
	writeFile(t, filepath.Join(baseDir, "broken.yaml"), "routes: [")

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name:    "missing file",
			cfg:     &Config{File: "does-not-exist.yaml"},
			wantErr: "read route table",
		},
		{
			name:    "nested file directive",
			cfg:     &Config{File: "nested.yaml"},
			wantErr: "must not declare file",
		},
		{
			name:    "malformed yaml",
			cfg:     &Config{File: "broken.yaml"},
			wantErr: "parse route table",
		},
		{
			name: "file plus inline routes",
			cfg: &Config{
				File:   "nested.yaml",
				Routes: []Route{{Match: "unknown", Profile: "triage", Action: "triage"}},
			},
			wantErr: "either file or routes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(tt.cfg, Options{Platform: "sms", DefaultProfile: "assistant", BaseDir: baseDir})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_ContactsFromFileAndInline(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "contacts.yaml"), `
contacts:
  - id: jane
    name: Jane Doe
    org: Acme
    keys:
      - "whatsapp:+1 (555) 123-4567"
      - "Jane@Acme.com"
`)
	cfg := &Config{
		Contacts: &ContactsConfig{
			Path: "contacts.yaml",
			Entries: []Contact{
				{ID: "bob", Org: "globex", Keys: []string{"+15559876543"}},
			},
		},
		Routes: []Route{
			{Match: "org:acme", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	}
	table, err := Load(cfg, Options{Platform: "whatsapp", DefaultProfile: "assistant", BaseDir: baseDir})
	require.NoError(t, err)

	contacts := table.Contacts()
	require.Len(t, contacts, 2)
	assert.Equal(t, "jane", contacts[0].ID)
	assert.Equal(t, []string{"+15551234567", "jane@acme.com"}, contacts[0].Keys, "contact keys are normalized at load")
	assert.Equal(t, "bob", contacts[1].ID)
}

func TestLoad_ContactsValidation(t *testing.T) {
	tests := []struct {
		name     string
		contacts *ContactsConfig
		wantErr  string
	}{
		{
			name:     "missing id",
			contacts: &ContactsConfig{Entries: []Contact{{Keys: []string{"+15551234567"}}}},
			wantErr:  "id is required",
		},
		{
			name:     "missing keys",
			contacts: &ContactsConfig{Entries: []Contact{{ID: "jane"}}},
			wantErr:  "at least one key",
		},
		{
			name: "duplicate id",
			contacts: &ContactsConfig{Entries: []Contact{
				{ID: "jane", Keys: []string{"+15551234567"}},
				{ID: "jane", Keys: []string{"+15559876543"}},
			}},
			wantErr: "duplicate contact id",
		},
		{
			name: "duplicate key across contacts",
			contacts: &ContactsConfig{Entries: []Contact{
				{ID: "jane", Keys: []string{"whatsapp:+15551234567"}},
				{ID: "bob", Keys: []string{"+1 555 123 4567"}},
			}},
			wantErr: "already belongs to contact",
		},
		{
			name:     "missing file",
			contacts: &ContactsConfig{Path: "nope.yaml"},
			wantErr:  "read contacts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Contacts: tt.contacts,
				Routes:   []Route{{Match: "unknown", Profile: "triage", Action: "triage"}},
			}
			_, err := Load(cfg, Options{Platform: "whatsapp", DefaultProfile: "assistant", BaseDir: t.TempDir()})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_ContactsDeclaredInServiceAndFileConflict(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "routes.yaml"), `
contacts:
  entries:
    - id: jane
      keys: ["+15551234567"]
routes:
  - match: unknown
    profile: triage
    action: triage
`)
	cfg := &Config{
		File:     "routes.yaml",
		Contacts: &ContactsConfig{Entries: []Contact{{ID: "bob", Keys: []string{"+15559876543"}}}},
	}
	_, err := Load(cfg, Options{Platform: "sms", DefaultProfile: "assistant", BaseDir: baseDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contacts declared in both")
}

func TestLoad_ExternalFileCarriesContacts(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "routes.yaml"), `
contacts:
  entries:
    - id: jane
      org: acme
      keys: ["+15551234567"]
routes:
  - match: org:acme
    profile: acme
    action: inbox
  - match: unknown
    profile: triage
    action: triage
`)
	table, err := Load(&Config{File: "routes.yaml"}, Options{Platform: "sms", DefaultProfile: "assistant", BaseDir: baseDir})
	require.NoError(t, err)
	require.Len(t, table.Contacts(), 1)
	require.Len(t, table.Routes(), 2)
}

func TestLoad_NilConfig(t *testing.T) {
	_, err := Load(nil, Options{})
	require.Error(t, err)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
