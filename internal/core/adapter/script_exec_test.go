package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildScriptEnv_EnvPrefix(t *testing.T) {
	tests := []struct {
		name           string
		manifestPrefix string
		inputs         map[string]string
		wantPrefix     string
		wantKey        string
	}{
		{
			name:           "no prefix falls back to ADAPTER",
			manifestPrefix: "",
			inputs:         map[string]string{"foo": "bar"},
			wantPrefix:     "ADAPTER",
			wantKey:        "ADAPTER_FOO=bar",
		},
		{
			name:           "manifest prefix CAL",
			manifestPrefix: "CAL",
			inputs:         map[string]string{"event-id": "42"},
			wantPrefix:     "CAL",
			wantKey:        "CAL_EVENT_ID=42",
		},
		{
			name:           "manifest prefix EMAIL regression guard",
			manifestPrefix: "EMAIL",
			inputs:         map[string]string{"to": "u@example.com"},
			wantPrefix:     "EMAIL",
			wantKey:        "EMAIL_TO=u@example.com",
		},
		{
			name:           "manifest prefix CONTACT regression guard",
			manifestPrefix: "CONTACT",
			inputs:         map[string]string{"id": "abc"},
			wantPrefix:     "CONTACT",
			wantKey:        "CONTACT_ID=abc",
		},
		{
			name:           "lowercase manifest prefix uppercases",
			manifestPrefix: "cal",
			inputs:         map[string]string{"x": "y"},
			wantPrefix:     "CAL",
			wantKey:        "CAL_X=y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Adapter{
				Name:   "test",
				Config: map[string]any{},
			}
			manifest := &AdapterManifest{
				Name:      "test",
				EnvPrefix: tt.manifestPrefix,
			}

			env := buildScriptEnv(device, manifest, "", tt.inputs)

			var found bool
			for _, e := range env {
				if e == tt.wantKey {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("env missing %q; got: %v", tt.wantKey, env)
			}

			for _, e := range env {
				if !strings.Contains(e, "=") {
					continue
				}
				key := e[:strings.Index(e, "=")]
				if strings.HasPrefix(key, "APS_") {
					continue
				}
				if !strings.HasPrefix(key, tt.wantPrefix+"_") {
					t.Fatalf("env key %q does not match wanted prefix %q", key, tt.wantPrefix)
				}
			}
		})
	}
}

func TestApplyInputDefaults(t *testing.T) {
	schema := &ActionSchema{
		Name: "list",
		Inputs: []ActionInput{
			{Name: "limit", Default: "10"},
			{Name: "folder", Default: "INBOX"},
			{Name: "query"},
			{Name: "id", Required: true},
		},
	}

	tests := []struct {
		name   string
		schema *ActionSchema
		inputs map[string]string
		want   map[string]string
	}{
		{
			name:   "omitted keys take declared defaults",
			schema: schema,
			inputs: map[string]string{},
			want:   map[string]string{"limit": "10", "folder": "INBOX"},
		},
		{
			name:   "supplied value beats default",
			schema: schema,
			inputs: map[string]string{"limit": "50"},
			want:   map[string]string{"limit": "50", "folder": "INBOX"},
		},
		{
			name:   "explicitly empty supplied value beats default",
			schema: schema,
			inputs: map[string]string{"folder": ""},
			want:   map[string]string{"limit": "10", "folder": ""},
		},
		{
			name:   "input without a declared default stays absent",
			schema: schema,
			inputs: map[string]string{"limit": "1", "folder": "Sent"},
			want:   map[string]string{"limit": "1", "folder": "Sent"},
		},
		{
			name:   "required input is never synthesised",
			schema: schema,
			inputs: map[string]string{},
			want:   map[string]string{"limit": "10", "folder": "INBOX"},
		},
		{
			name:   "undeclared supplied key is preserved",
			schema: schema,
			inputs: map[string]string{"not-in-manifest": "leaked"},
			want: map[string]string{
				"not-in-manifest": "leaked",
				"limit":           "10",
				"folder":          "INBOX",
			},
		},
		{
			name:   "schema with no defaults returns inputs unchanged",
			schema: &ActionSchema{Name: "read", Inputs: []ActionInput{{Name: "id", Required: true}}},
			inputs: map[string]string{"id": "42"},
			want:   map[string]string{"id": "42"},
		},
		{
			name:   "nil schema returns inputs unchanged",
			schema: nil,
			inputs: map[string]string{"id": "42"},
			want:   map[string]string{"id": "42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyInputDefaults(tt.schema, tt.inputs)
			if len(got) != len(tt.want) {
				t.Fatalf("applyInputDefaults() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("key %q absent; got %v", k, got)
					continue
				}
				if gotV != v {
					t.Errorf("key %q = %q, want %q", k, gotV, v)
				}
			}
		})
	}
}

// TestApplyInputDefaults_DoesNotMutateCaller guards the copy-on-write
// contract: the caller's map must be left exactly as it was passed.
func TestApplyInputDefaults_DoesNotMutateCaller(t *testing.T) {
	schema := &ActionSchema{
		Inputs: []ActionInput{{Name: "folder", Default: "INBOX"}},
	}
	inputs := map[string]string{"limit": "5"}

	got := applyInputDefaults(schema, inputs)

	if _, ok := inputs["folder"]; ok {
		t.Fatalf("caller map was mutated: %v", inputs)
	}
	if got["folder"] != "INBOX" {
		t.Fatalf("default not applied to result: %v", got)
	}
}

// TestBuildScriptEnv_DefaultsReachScript proves the defaulted map is
// what reaches the env, prefixed like any caller-supplied input.
func TestBuildScriptEnv_DefaultsReachScript(t *testing.T) {
	schema := &ActionSchema{
		Inputs: []ActionInput{
			{Name: "limit", Default: "10"},
			{Name: "folder", Default: "INBOX"},
		},
	}
	device := &Adapter{Name: "email", Config: map[string]any{}}
	manifest := &AdapterManifest{Name: "email", EnvPrefix: "EMAIL"}

	env := buildScriptEnv(device, manifest, "",
		applyInputDefaults(schema, map[string]string{"limit": "50"}))

	want := map[string]bool{"EMAIL_LIMIT=50": false, "EMAIL_FOLDER=INBOX": false}
	for _, e := range env {
		if _, ok := want[e]; ok {
			want[e] = true
		}
		if e == "EMAIL_LIMIT=10" {
			t.Errorf("default overrode caller-supplied limit; env: %v", env)
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("env missing %q; got %v", k, env)
		}
	}
}

func TestResolveEnvPrefix_Precedence(t *testing.T) {
	if got := resolveEnvPrefix(&AdapterManifest{EnvPrefix: "MAN"}); got != "MAN" {
		t.Fatalf("manifest prefix should be returned; got %q", got)
	}
	if got := resolveEnvPrefix(&AdapterManifest{}); got != DefaultEnvPrefix {
		t.Fatalf("default fallback failed; got %q", got)
	}
	if got := resolveEnvPrefix(nil); got != DefaultEnvPrefix {
		t.Fatalf("nil-safe fallback failed; got %q", got)
	}
}

func TestValidateManifestEnvPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"single underscore valid", "_", false},
		{"alpha only", "CAL", false},
		{"alphanumeric", "CAL2", false},
		{"underscore separator", "MY_PREFIX", false},
		{"lowercase valid (uppercased later)", "cal", false},
		{"mixed case valid", "MyPrefix", false},
		{"leading digit invalid", "2CAL", true},
		{"contains space invalid", "MY PREFIX", true},
		{"contains hyphen invalid", "MY-PREFIX", true},
		{"contains dot invalid", "MY.PREFIX", true},
		{"trailing space invalid", "CAL ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManifestEnvPrefix(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateManifestEnvPrefix(%q) err=%v, wantErr=%v", tt.prefix, err, tt.wantErr)
			}
		})
	}
}

func TestValidateManifestEnvPrefix_EscapeHatch(t *testing.T) {
	t.Setenv(envPrefixValidationDisableEnv, "1")
	if err := validateManifestEnvPrefix("MY PREFIX"); err != nil {
		t.Fatalf("escape hatch should bypass validation; got %v", err)
	}
}

func TestLoadManifest_RejectsInvalidEnvPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte("name: test\ntype: messenger\nenv_prefix: \"MY PREFIX\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("LoadManifest should reject invalid env_prefix")
	}
	if !strings.Contains(err.Error(), "env_prefix") {
		t.Fatalf("error should mention env_prefix; got %v", err)
	}
}

func TestLoadManifest_AcceptsValidEnvPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte("name: test\ntype: messenger\nenv_prefix: CAL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest should accept valid prefix; got %v", err)
	}
	if manifest.EnvPrefix != "CAL" {
		t.Fatalf("EnvPrefix not preserved; got %q", manifest.EnvPrefix)
	}
}

func TestUndeclaredInputNames(t *testing.T) {
	declared := &ActionSchema{
		Name: "send",
		Inputs: []ActionInput{
			{Name: "to", Required: true},
			{Name: "subject", Required: true},
			{Name: "cc"},
		},
	}

	tests := []struct {
		name   string
		schema *ActionSchema
		inputs map[string]string
		want   []string
	}{
		{
			name:   "all declared yields none",
			schema: declared,
			inputs: map[string]string{"to": "u@example.com", "cc": "x@example.com"},
			want:   nil,
		},
		{
			name:   "single undeclared key named",
			schema: declared,
			inputs: map[string]string{"to": "u@example.com", "bdy": "typo"},
			want:   []string{"bdy"},
		},
		{
			name:   "multiple undeclared sorted",
			schema: declared,
			inputs: map[string]string{"zeta": "1", "alpha": "2", "to": "3"},
			want:   []string{"alpha", "zeta"},
		},
		{
			// An action with no `input:` key declares no vocabulary, so
			// nothing can be undeclared against it. Warning here would
			// flag every key of every such adapter.
			name:   "schema declaring no inputs never warns",
			schema: &ActionSchema{Name: "raw"},
			inputs: map[string]string{"anything": "1", "else": "2"},
			want:   nil,
		},
		{
			name:   "nil schema never warns",
			schema: nil,
			inputs: map[string]string{"anything": "1"},
			want:   nil,
		},
		{
			name:   "no inputs supplied yields none",
			schema: declared,
			inputs: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := undeclaredInputNames(tt.schema, tt.inputs)
			if len(got) != len(tt.want) {
				t.Fatalf("undeclaredInputNames() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("undeclaredInputNames() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestWarnUndeclaredInputs_Output(t *testing.T) {
	schema := &ActionSchema{
		Name:   "read",
		Inputs: []ActionInput{{Name: "id", Required: true}},
	}

	t.Run("names action and undeclared key", func(t *testing.T) {
		var buf strings.Builder
		warnUndeclaredInputs(&buf, "read", schema, map[string]string{
			"id":              "42",
			"not-in-manifest": "leaked",
		})
		got := buf.String()
		if !strings.HasPrefix(got, "warn: ") {
			t.Errorf("warning should use the warn: prefix; got %q", got)
		}
		if !strings.Contains(got, "not-in-manifest") {
			t.Errorf("warning should name the undeclared key; got %q", got)
		}
		if !strings.Contains(got, `"read"`) {
			t.Errorf("warning should name the action; got %q", got)
		}
		if strings.Contains(got, "\"id\"") {
			t.Errorf("warning should not name declared inputs; got %q", got)
		}
		if !strings.HasSuffix(got, "\n") {
			t.Errorf("warning should end with a newline; got %q", got)
		}
	})

	t.Run("silent when every key is declared", func(t *testing.T) {
		var buf strings.Builder
		warnUndeclaredInputs(&buf, "read", schema, map[string]string{"id": "42"})
		if got := buf.String(); got != "" {
			t.Errorf("expected no warning, got %q", got)
		}
	})

	t.Run("silent when action declares no inputs", func(t *testing.T) {
		var buf strings.Builder
		warnUndeclaredInputs(&buf, "raw", &ActionSchema{Name: "raw"},
			map[string]string{"whatever": "1"})
		if got := buf.String(); got != "" {
			t.Errorf("expected no warning for an action declaring no inputs, got %q", got)
		}
	})
}
