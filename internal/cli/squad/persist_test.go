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
	if err := runCreate(io.Discard, "", "demo-squad", "platform", "ops", "", []string{"alice", "bob"}, false); err != nil {
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
	if err := runCreate(io.Discard, "", "demo-squad", "platform", "ops", "d", []string{"alice", "bob"}, false); err != nil {
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

func TestSquadCreate_GoldenPathFlag(t *testing.T) {
	fresh := useDataDir(t)
	for _, name := range []string{"with-gp", "without-gp"} {
		cmd := newCreateCmd()
		cmd.Flags().String("format", "", "")
		args := []string{name, "--type", "platform", "--domain", "ops", "--members", "a"}
		if name == "with-gp" {
			args = append(args, "--golden-path-defined")
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	fresh() // new process: field must come back from the manifest
	with, err := defaultManager.Get("with-gp")
	if err != nil {
		t.Fatal(err)
	}
	without, err := defaultManager.Get("without-gp")
	if err != nil {
		t.Fatal(err)
	}
	if !with.GoldenPathDefined {
		t.Error("--golden-path-defined not persisted")
	}
	if without.GoldenPathDefined {
		t.Error("golden path defaulted to true")
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(runShowWith(t, "with-gp", "--format", "json")), &rec); err != nil {
		t.Fatal(err)
	}
	if rec["golden_path_defined"] != true {
		t.Errorf("show json golden_path_defined = %v", rec["golden_path_defined"])
	}
	if out := runShowWith(t, "with-gp"); !strings.Contains(out, "Golden path: defined") {
		t.Errorf("show text missing golden path: %q", out)
	}
}

func TestSquadCreate_GoldenPathAcceptedOnAnyType(t *testing.T) {
	useDataDir(t)
	if err := runCreate(io.Discard, "", "s", "stream-aligned", "d", "", []string{"a"}, true); err != nil {
		t.Fatalf("flag on stream-aligned squad: %v", err)
	}
	s, err := defaultManager.Get("s")
	if err != nil || !s.GoldenPathDefined {
		t.Fatalf("stored = %+v, err=%v", s, err)
	}
}

func TestSquadCheck_GoldenPathFlagSatisfiesCheck(t *testing.T) {
	useDataDir(t)
	if err := runCreate(io.Discard, "", "plat", "platform", "ops", "", []string{"a"}, true); err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	if err := runCheck(&out, &errw, ""); err != nil {
		t.Fatalf("check on flagged platform squad: %v\n%s", err, out.String())
	}

	useDataDir(t)
	if err := runCreate(io.Discard, "", "plat", "platform", "ops", "", []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runCheck(&out, &errw, ""); err == nil {
		t.Fatal("check passed for platform squad without golden path")
	}
	if !strings.Contains(out.String(), "--golden-path-defined") {
		t.Errorf("failure detail does not name the flag: %q", out.String())
	}
}
