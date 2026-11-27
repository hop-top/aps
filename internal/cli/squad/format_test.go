package squad

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// squadRoot mirrors the real wiring: a root carrying the persistent
// --format global with the squad command tree below it.
func squadRoot() *cobra.Command {
	root := &cobra.Command{Use: "aps", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("format", "", "output format")
	root.AddCommand(NewSquadCmd())
	return root
}

// runSquad runs `aps squad <args>` against a fresh manager over a temp data
// dir (set by useDataDir) and returns stdout, stderr and the error.
func runSquad(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := squadRoot()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(append([]string{"squad"}, args...))
	err := root.Execute()
	return out.String(), errb.String(), err
}

func mustRun(t *testing.T, args ...string) string {
	t.Helper()
	out, errb, err := runSquad(t, args...)
	if err != nil {
		t.Fatalf("aps squad %v: %v\nstderr: %s", args, err, errb)
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	sort.Strings(k)
	return k
}

var recordKeys = []string{
	"created_at", "description", "domain", "golden_path_defined",
	"id", "members", "name", "type", "updated_at",
}

func TestShow_JSONRecordShape(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice,bob")

	out := mustRun(t, "show", "demo-squad", "--format", "json")
	var rec map[string]any
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("not a single JSON document: %v\n%s", err, out)
	}
	if got := strings.Join(sortedKeys(rec), ","); got != strings.Join(recordKeys, ",") {
		t.Errorf("keys = %s\nwant   %s", got, strings.Join(recordKeys, ","))
	}
	if rec["id"] != "demo-squad" || rec["name"] != "Demo Squad" || rec["type"] != "platform" || rec["domain"] != "core" {
		t.Errorf("record = %v", rec)
	}
	if rec["description"] != "" || rec["golden_path_defined"] != false {
		t.Errorf("empty description / false flag must still be present: %v", rec)
	}
	members, ok := rec["members"].([]any)
	if !ok || len(members) != 2 || members[0] != "alice" || members[1] != "bob" {
		t.Errorf("members = %#v, want [alice bob]", rec["members"])
	}
	for _, k := range []string{"created_at", "updated_at"} {
		s, _ := rec[k].(string)
		if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
			t.Errorf("%s = %q is not RFC 3339: %v", k, rec[k], err)
		}
	}
}

func TestShow_EmptyMembersIsEmptyArray(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Solo", "--type", "enabling", "--domain", "d")

	out := mustRun(t, "show", "solo", "--format", "json")
	if !strings.Contains(out, `"members": []`) {
		t.Errorf("want \"members\": [] in:\n%s", out)
	}
	if strings.Contains(out, "null") || strings.Contains(out, "(none)") {
		t.Errorf("empty members must be [], got:\n%s", out)
	}

	y := mustRun(t, "show", "solo", "--format", "yaml")
	var rec map[string]any
	if err := yaml.Unmarshal([]byte(y), &rec); err != nil {
		t.Fatalf("yaml: %v\n%s", err, y)
	}
	if m, ok := rec["members"].([]any); !ok || len(m) != 0 {
		t.Errorf("yaml members = %#v, want empty list", rec["members"])
	}
}

func TestShow_YAMLRecordShape(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice,bob")
	var rec map[string]any
	out := mustRun(t, "show", "demo-squad", "--format", "yaml")
	if err := yaml.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("yaml: %v\n%s", err, out)
	}
	if got := strings.Join(sortedKeys(rec), ","); got != strings.Join(recordKeys, ",") {
		t.Errorf("keys = %s", got)
	}
	if m, _ := rec["members"].([]any); len(m) != 2 || m[0] != "alice" {
		t.Errorf("members = %#v", rec["members"])
	}
}

func TestShow_HumanOutputUnchanged(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice,bob")
	mustRun(t, "create", "Solo", "--type", "enabling", "--domain", "d")
	out := mustRun(t, "show", "demo-squad")
	for _, want := range []string{"Squad: Demo Squad", "  ID:          demo-squad", "  Type:        platform", "  Members:     alice, bob", "  Created:     "} {
		if !strings.Contains(out, want) {
			t.Errorf("human output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(mustRun(t, "show", "solo"), "  Members:     (none)") {
		t.Error("empty members in human view should read (none)")
	}
	for _, f := range []string{"table", "text", "human"} {
		if got := mustRun(t, "show", "demo-squad", "--format", f); got != out {
			t.Errorf("--format %s differs from default human view", f)
		}
	}
}

func TestShow_ErrorsNameTheProblem(t *testing.T) {
	useDataDir(t)
	if _, _, err := runSquad(t, "show", "nope"); err == nil || !strings.Contains(err.Error(), "squad not found: nope") {
		t.Errorf("unknown id err = %v", err)
	}
	dir, _ := squadsDirForTest(t)
	bad := filepath.Join(dir, "bad", "manifest.yaml")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(": : :\n\t-"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runSquad(t, "show", "bad"); err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("corrupt squad err = %v, want it to name %s", err, bad)
	}
}

func TestShow_HelpDescribesPersistence(t *testing.T) {
	long := newShowCmd().Long
	if strings.Contains(long, "underlying squad manager") {
		t.Error("help still blames the in-memory manager for unknown ids")
	}
	if !strings.Contains(long, "squads/") {
		t.Errorf("help should say where squads are stored:\n%s", long)
	}
}

func TestList_JSONYAMLAndEmpty(t *testing.T) {
	useDataDir(t)
	if out := strings.TrimSpace(mustRun(t, "list", "--format", "json")); out != "[]" {
		t.Errorf("empty list json = %q, want []", out)
	}
	mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice,bob")
	mustRun(t, "create", "Solo", "--type", "enabling", "--domain", "d")

	var rows []map[string]any
	if err := json.Unmarshal([]byte(mustRun(t, "list", "--format", "json")), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["id"] != "demo-squad" || rows[1]["id"] != "solo" {
		t.Fatalf("rows = %v", rows)
	}
	if rows[0]["member_count"] != float64(2) || rows[1]["member_count"] != float64(0) {
		t.Errorf("member_count = %v / %v", rows[0]["member_count"], rows[1]["member_count"])
	}
	var y []map[string]any
	if err := yaml.Unmarshal([]byte(mustRun(t, "list", "--format", "yaml")), &y); err != nil || len(y) != 2 {
		t.Errorf("yaml list: %v %v", err, y)
	}
	if !strings.Contains(mustRun(t, "list"), "demo-squad") {
		t.Error("table list lost the squad")
	}
}

func TestList_SkipsCorruptSquadAndNamesIt(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Good", "--type", "platform", "--domain", "d", "--members", "a")
	dir, _ := squadsDirForTest(t)
	bad := filepath.Join(dir, "bad", "manifest.yaml")
	_ = os.MkdirAll(filepath.Dir(bad), 0o755)
	_ = os.WriteFile(bad, []byte(": : :\n\t-"), 0o644)

	out, errb, err := runSquad(t, "list", "--format", "json")
	if err != nil {
		t.Fatalf("list must survive a corrupt neighbour: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 1 || rows[0]["id"] != "good" {
		t.Errorf("stdout must stay a clean JSON array of the healthy squads: %v\n%s", err, out)
	}
	if !strings.Contains(errb, bad) {
		t.Errorf("stderr must name the damaged manifest, got %q", errb)
	}
}

func TestMutations_StructuredUnderJSON(t *testing.T) {
	useDataDir(t)

	var rec map[string]any
	out := mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice", "--format", "json")
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("create output not JSON: %v\n%s", err, out)
	}
	if rec["id"] != "demo-squad" || len(rec["members"].([]any)) != 1 {
		t.Errorf("create record = %v", rec)
	}

	out = mustRun(t, "members", "add", "demo-squad", "bob", "--format", "json")
	rec = nil
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("members add output not JSON: %v\n%s", err, out)
	}
	if m := rec["members"].([]any); len(m) != 2 || m[1] != "bob" {
		t.Errorf("members add record = %v", rec)
	}

	out = mustRun(t, "members", "remove", "demo-squad", "alice", "--format", "json")
	rec = nil
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("members remove output not JSON: %v\n%s", err, out)
	}
	if m := rec["members"].([]any); len(m) != 1 || m[0] != "bob" {
		t.Errorf("members remove record = %v", rec)
	}

	out = mustRun(t, "delete", "demo-squad", "--format", "json")
	var del map[string]any
	if err := json.Unmarshal([]byte(out), &del); err != nil {
		t.Fatalf("delete output not JSON: %v\n%s", err, out)
	}
	if del["id"] != "demo-squad" || del["deleted"] != true {
		t.Errorf("delete result = %v", del)
	}

	y := mustRun(t, "create", "Again", "--type", "platform", "--domain", "d", "--format", "yaml")
	var yrec map[string]any
	if err := yaml.Unmarshal([]byte(y), &yrec); err != nil || yrec["id"] != "again" {
		t.Errorf("create yaml: %v\n%s", err, y)
	}
}

func TestMutations_HumanTextUnchanged(t *testing.T) {
	useDataDir(t)
	out := mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice,bob")
	for _, want := range []string{`Created squad "Demo Squad" (demo-squad)`, "  Type:    platform", "  Members: alice, bob"} {
		if !strings.Contains(out, want) {
			t.Errorf("create output missing %q:\n%s", want, out)
		}
	}
	if got := mustRun(t, "members", "add", "demo-squad", "carol"); got != "Added \"carol\" to squad \"demo-squad\"\n" {
		t.Errorf("members add = %q", got)
	}
	if got := mustRun(t, "members", "remove", "demo-squad", "carol"); got != "Removed \"carol\" from squad \"demo-squad\"\n" {
		t.Errorf("members remove = %q", got)
	}
	if got := mustRun(t, "delete", "demo-squad"); got != "Deleted squad \"demo-squad\"\n" {
		t.Errorf("delete = %q", got)
	}
}

func TestCheck_FormatAndPersistedSquads(t *testing.T) {
	useDataDir(t)
	mustRun(t, "create", "Demo Squad", "--type", "platform", "--domain", "core", "--members", "alice")

	out, _, _ := runSquad(t, "check", "--format", "json")
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("check --format json is not a JSON array: %v\n%s", err, out)
	}
	if len(rows) == 0 || rows[0]["check"] == nil || rows[0]["status"] == nil {
		t.Errorf("rows = %v", rows)
	}
	tbl, _, _ := runSquad(t, "check")
	if !strings.Contains(tbl, "checks passed") {
		t.Errorf("table output lost its summary:\n%s", tbl)
	}
}

func TestCheck_CorruptManifestIsAnError(t *testing.T) {
	useDataDir(t)
	dir, _ := squadsDirForTest(t)
	bad := filepath.Join(dir, "bad", "manifest.yaml")
	_ = os.MkdirAll(filepath.Dir(bad), 0o755)
	_ = os.WriteFile(bad, []byte(": : :\n\t-"), 0o644)
	_, _, err := runSquad(t, "check")
	if err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("check must fail naming %s, got %v", bad, err)
	}
}
