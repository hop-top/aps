package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEffectiveType_EmptyDefaultsToAgent(t *testing.T) {
	p := &Profile{ID: "noor"}
	if got := p.EffectiveType(); got != ProfileTypeAgent {
		t.Fatalf("EffectiveType() = %q, want %q", got, ProfileTypeAgent)
	}
}

func TestEffectiveType_ReturnsRawValue(t *testing.T) {
	cases := []struct {
		typ  string
		want string
	}{
		{ProfileTypeAgent, ProfileTypeAgent},
		{ProfileTypeHuman, ProfileTypeHuman},
		// Unknown values pass through untouched so `aps org check`
		// can surface them later.
		{"person", "person"},
	}
	for _, c := range cases {
		p := &Profile{ID: "noor", Type: c.typ}
		if got := p.EffectiveType(); got != c.want {
			t.Errorf("EffectiveType() with Type=%q = %q, want %q", c.typ, got, c.want)
		}
	}
}

func TestValidateType(t *testing.T) {
	cases := []struct {
		typ     string
		wantErr bool
	}{
		{"", false},
		{ProfileTypeAgent, false},
		{ProfileTypeHuman, false},
		{"person", true},
		{"Human", true}, // case-sensitive: only lowercase canonical values
	}
	for _, c := range cases {
		p := &Profile{ID: "noor", Type: c.typ}
		err := p.ValidateType()
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateType() with Type=%q: err=%v, wantErr=%v", c.typ, err, c.wantErr)
		}
	}
}

func TestLoadProfileFromPath_ToleratesUnknownType(t *testing.T) {
	// Read-time must NOT reject unknown type values: ListProfilesFull
	// silently drops profiles whose load errors, so a strict load would
	// make a typo'd profile vanish from lists and graphs.
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	content := "id: typoed\ndisplay_name: Typoed\ntype: person\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadProfileFromPath("typoed", path)
	if err != nil {
		t.Fatalf("LoadProfileFromPath() with unknown type failed: %v", err)
	}
	if p.Type != "person" {
		t.Errorf("Type = %q, want %q", p.Type, "person")
	}
	if got := p.EffectiveType(); got != "person" {
		t.Errorf("EffectiveType() = %q, want %q", got, "person")
	}
}

func TestProfileType_YAMLRoundTrip(t *testing.T) {
	in := &Profile{ID: "human-1", DisplayName: "A Human", Type: ProfileTypeHuman}
	data, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "type: human") {
		t.Errorf("marshaled YAML missing type field:\n%s", data)
	}

	var out Profile
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != ProfileTypeHuman {
		t.Errorf("round-trip Type = %q, want %q", out.Type, ProfileTypeHuman)
	}
}

func TestProfileType_YAMLOmittedWhenEmpty(t *testing.T) {
	in := &Profile{ID: "agent-1", DisplayName: "An Agent"}
	data, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "type:") {
		t.Errorf("empty type should be omitted from YAML:\n%s", data)
	}
}

func TestEnsureRunnable(t *testing.T) {
	agent := &Profile{ID: "agent-1"}
	if err := agent.EnsureRunnable(); err != nil {
		t.Errorf("agent profile should be runnable, got: %v", err)
	}

	human := &Profile{ID: "jane", Type: ProfileTypeHuman}
	err := human.EnsureRunnable()
	if err == nil {
		t.Fatal("human profile should not be runnable")
	}
	if !strings.Contains(err.Error(), `profile "jane" is type human and cannot be run`) {
		t.Errorf("unexpected error message: %v", err)
	}

	// Unknown types are tolerated at read time and stay runnable; `aps
	// org check` is the surface that flags them.
	typoed := &Profile{ID: "typoed", Type: "person"}
	if err := typoed.EnsureRunnable(); err != nil {
		t.Errorf("unknown-type profile should remain runnable, got: %v", err)
	}
}
