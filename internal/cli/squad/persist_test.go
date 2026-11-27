package squad

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	coresquad "hop.top/aps/internal/core/squad"

	"gopkg.in/yaml.v3"
)

// useDataDir points the default manager at a temp APS_DATA_PATH and
// returns a constructor for a fresh manager (a "new process").
func useDataDir(t *testing.T) func() {
	t.Helper()
	t.Setenv("APS_DATA_PATH", t.TempDir())
	prev := defaultManager
	t.Cleanup(func() { defaultManager = prev })
	fresh := func() { defaultManager = newManager() }
	fresh()
	return fresh
}

// squadsDirForTest returns the squads directory under the test data dir.
func squadsDirForTest(t *testing.T) (string, error) {
	t.Helper()
	return coresquad.SquadsDir()
}

func runShowWith(t *testing.T, args ...string) string {
	t.Helper()
	cmd := newShowCmd()
	cmd.Flags().String("format", "", "")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("show: %v", err)
	}
	return buf.String()
}

func TestSquadCLI_PersistsAcrossProcesses(t *testing.T) {
	fresh := useDataDir(t)
	if err := runCreate(io.Discard, "", "demo-squad", "platform", "ops", "", []string{"alice", "bob"}); err != nil {
		t.Fatal(err)
	}

	fresh() // new process
	if err := runAddMember(io.Discard, "", "demo-squad", "carol"); err != nil {
		t.Fatal(err)
	}

	fresh()
	s, err := defaultManager.Get("demo-squad")
	if err != nil {
		t.Fatalf("squad lost across processes: %v", err)
	}
	if strings.Join(s.Members, ",") != "alice,bob,carol" {
		t.Errorf("members = %v", s.Members)
	}
	if got := len(defaultManager.List()); got != 1 {
		t.Errorf("list len = %d", got)
	}

	if err := runDelete(io.Discard, "", "demo-squad"); err != nil {
		t.Fatal(err)
	}
	fresh()
	if got := len(defaultManager.List()); got != 0 {
		t.Errorf("delete lost across processes, list len = %d", got)
	}
}

func TestSquadShow_FormatJSONYAML(t *testing.T) {
	fresh := useDataDir(t)
	if err := runCreate(io.Discard, "", "demo-squad", "platform", "ops", "d", []string{"alice", "bob"}); err != nil {
		t.Fatal(err)
	}
	fresh()

	var j map[string]any
	if err := json.Unmarshal([]byte(runShowWith(t, "demo-squad", "--format", "json")), &j); err != nil {
		t.Fatalf("json output not parseable: %v", err)
	}
	if j["id"] != "demo-squad" || j["domain"] != "ops" {
		t.Errorf("json = %v", j)
	}

	var y map[string]any
	if err := yaml.Unmarshal([]byte(runShowWith(t, "demo-squad", "--format", "yaml")), &y); err != nil {
		t.Fatalf("yaml output not parseable: %v", err)
	}
	if y["id"] != "demo-squad" {
		t.Errorf("yaml = %v", y)
	}

	if out := runShowWith(t, "demo-squad"); !strings.Contains(out, "Squad: demo-squad") {
		t.Errorf("default output = %q", out)
	}
}
