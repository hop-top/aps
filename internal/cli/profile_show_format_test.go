package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/core"
)

// renderShow drives the profile-show renderer for one format and
// returns what would land on stdout.
func renderShow(t *testing.T, format string, p *core.Profile) string {
	t.Helper()
	var buf bytes.Buffer
	if err := writeProfileShow(&buf, format, p); err != nil {
		t.Fatalf("writeProfileShow(%q): %v", format, err)
	}
	return buf.String()
}

func showFixture() *core.Profile {
	return &core.Profile{
		ID:           "worker",
		DisplayName:  "worker",
		Capabilities: []string{"a2a"},
	}
}

// TestProfileShowJSONIsParseable pins Factor 3: in a declared
// machine-readable mode, stdout must parse as a single document of
// that format.
//
// `aps profile show --format json` previously emitted YAML followed by
// an ANSI-styled human block regardless of --format, so an agent that
// asked for JSON received something no JSON parser accepts.
func TestProfileShowJSONIsParseable(t *testing.T) {
	out := renderShow(t, "json", showFixture())

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("stdout is not parseable JSON: %v\n---\n%s", err, out)
	}
	if doc["id"] != "worker" {
		t.Errorf("id = %v, want worker", doc["id"])
	}
	caps, ok := doc["capabilities"].([]any)
	if !ok || len(caps) != 1 || caps[0] != "a2a" {
		t.Errorf("capabilities = %v, want [a2a]", doc["capabilities"])
	}
}

// TestProfileShowJSONHasNoHumanFraming pins the no-interleaving rule:
// machine output must not carry banners or ANSI colour.
func TestProfileShowJSONHasNoHumanFraming(t *testing.T) {
	out := renderShow(t, "json", showFixture())

	if strings.Contains(out, "\x1b[") {
		t.Error("JSON output contains ANSI escape sequences")
	}
	for _, banner := range []string{"Modules:", "Secrets:", "Workspace:"} {
		if strings.Contains(out, banner) {
			t.Errorf("JSON output contains human framing %q", banner)
		}
	}
}

// TestProfileShowYAMLIsParseable pins the same contract for the other
// declared machine format.
func TestProfileShowYAMLIsParseable(t *testing.T) {
	out := renderShow(t, "yaml", showFixture())

	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("stdout is not parseable YAML: %v\n---\n%s", err, out)
	}
	if doc["id"] != "worker" {
		t.Errorf("id = %v, want worker", doc["id"])
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("YAML output contains ANSI escape sequences")
	}
}

// TestProfileShowHumanKeepsRichRender guards the other direction: the
// default human view is what operators rely on, so making the machine
// formats parseable must not strip its detail.
func TestProfileShowHumanKeepsRichRender(t *testing.T) {
	out := renderShow(t, "", showFixture())

	if !strings.Contains(out, "worker") {
		t.Errorf("human render lost the profile id:\n%s", out)
	}
	if !strings.Contains(out, "capabilities") {
		t.Errorf("human render lost the capabilities section:\n%s", out)
	}
}
