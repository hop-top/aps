package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildToolSpec_NameAndSchema(t *testing.T) {
	spec := buildToolSpec(rootCmd)
	if spec.Name != "aps" {
		t.Errorf("Name = %q, want %q", spec.Name, "aps")
	}
	if spec.SchemaVersion == "" {
		t.Error("SchemaVersion empty")
	}
}

func TestBuildToolSpec_IncludesAllRegisteredCommands(t *testing.T) {
	spec := buildToolSpec(rootCmd)

	// Walk cobra tree, collect every non-hidden command name (excluding root).
	want := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			want[sub.Name()] = true
			walk(sub)
		}
	}
	walk(rootCmd)

	got := map[string]bool{}
	var visit func(cmds []toolspecCommand)
	visit = func(cmds []toolspecCommand) {
		for _, c := range cmds {
			got[c.Name] = true
			visit(c.Children)
		}
	}
	visit(spec.Commands)

	for name := range want {
		if !got[name] {
			t.Errorf("ToolSpec missing command %q", name)
		}
	}
}

func TestBuildToolSpec_ContainsKnownCommands(t *testing.T) {
	spec := buildToolSpec(rootCmd)
	known := []string{"profile", "adapter", "session", "contact", "upgrade"}
	for _, n := range known {
		if spec.FindCommand(n) == nil {
			t.Errorf("FindCommand(%q) = nil; expected present", n)
		}
	}
}

func TestBuildToolSpec_HasErrorPatterns(t *testing.T) {
	spec := buildToolSpec(rootCmd)
	if len(spec.ErrorPatterns) == 0 {
		t.Fatal("ErrorPatterns empty")
	}
	// Spot-check: profile-not-found pattern present.
	found := false
	for _, p := range spec.ErrorPatterns {
		if strings.Contains(p.Pattern, "not found") || strings.Contains(p.Pattern, "unknown") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one not-found / unknown error pattern")
	}
}

func TestBuildToolSpec_HasWorkflows(t *testing.T) {
	spec := buildToolSpec(rootCmd)
	if len(spec.Workflows) == 0 {
		t.Fatal("Workflows empty")
	}
}

func TestToolspecCmd_JSONOutput(t *testing.T) {
	cmd := newToolspecCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"name": "aps"`) {
		t.Errorf("JSON output missing name=aps; got:\n%s", got)
	}
	if !strings.Contains(got, `"commands"`) {
		t.Errorf("JSON output missing commands; got:\n%s", got)
	}
}

func TestToolspecCmd_YAMLOutput(t *testing.T) {
	cmd := newToolspecCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--format", "yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "name: aps") {
		t.Errorf("YAML output missing name: aps; got:\n%s", got)
	}
}

func TestToolspecCmd_DefaultFormatIsJSON(t *testing.T) {
	cmd := newToolspecCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), `"name"`) {
		t.Error("default output not JSON")
	}
}

func TestToolspecCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "toolspec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("toolspec command not registered on root")
	}
}
