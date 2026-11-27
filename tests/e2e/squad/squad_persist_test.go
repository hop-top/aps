package squad_e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

// dataDir is where the binary keeps squads for an isolated HOME (see
// runAPS: XDG_DATA_HOME=<home>/.local/share, APS_DATA_PATH unset).
func squadsDir(home string) string {
	return filepath.Join(home, ".local", "share", "aps", "squads")
}

func mustAPS(t *testing.T, home string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runAPS(t, home, args...)
	if err != nil {
		t.Fatalf("aps %v: %v\nstdout: %s\nstderr: %s", args, err, stdout, stderr)
	}
	return stdout
}

var confirmTokenRe = regexp.MustCompile(`--confirm-token=([0-9a-f]+)`)

// confirmedAPS runs a destructive command through its confirmation gate:
// the first run is refused and prints a token, the second passes it back.
func confirmedAPS(t *testing.T, home string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runAPS(t, home, args...)
	if err == nil {
		t.Fatalf("aps %v ran without confirmation: %s", args, stdout)
	}
	m := confirmTokenRe.FindStringSubmatch(stdout + stderr)
	if m == nil {
		t.Fatalf("aps %v: no confirmation token in\nstdout: %s\nstderr: %s", args, stdout, stderr)
	}
	return mustAPS(t, home, append(args, "--confirm-token="+m[1])...)
}

func createDemo(t *testing.T, home string) {
	t.Helper()
	mustAPS(t, home, "squad", "create", "Demo Squad",
		"--type", "platform", "--domain", "core", "--members", "alice,bob")
}

// The original defect: create succeeded, then a separate process could not
// find the squad.
func TestSquadPersistence_CreateThenShowInNewProcess(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	out := mustAPS(t, home, "squad", "create", "Demo Squad",
		"--type", "platform", "--domain", "core", "--members", "alice,bob")
	if !strings.Contains(out, `Created squad "Demo Squad" (demo-squad)`) {
		t.Errorf("create output = %q", out)
	}

	show := mustAPS(t, home, "squad", "show", "demo-squad")
	for _, want := range []string{"Squad: Demo Squad", "demo-squad", "alice, bob"} {
		if !strings.Contains(show, want) {
			t.Errorf("show output missing %q:\n%s", want, show)
		}
	}

	if _, err := os.Stat(filepath.Join(squadsDir(home), "demo-squad", "manifest.yaml")); err != nil {
		t.Errorf("manifest not under the data dir: %v", err)
	}
}

func TestSquadPersistence_FullLifecycleAcrossProcesses(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	createDemo(t, home)

	if out := mustAPS(t, home, "squad", "list"); !strings.Contains(out, "demo-squad") {
		t.Errorf("list lost the squad:\n%s", out)
	}
	if out := mustAPS(t, home, "squad", "list", "--member", "alice"); !strings.Contains(out, "demo-squad") {
		t.Errorf("list --member alice:\n%s", out)
	}
	if out := mustAPS(t, home, "squad", "list", "--member", "zed"); strings.Contains(out, "demo-squad") {
		t.Errorf("list --member zed should not match:\n%s", out)
	}

	mustAPS(t, home, "squad", "members", "add", "demo-squad", "carol")
	var rec struct {
		Members []string `json:"members"`
	}
	mustJSON(t, mustAPS(t, home, "squad", "show", "demo-squad", "--format", "json"), &rec)
	if strings.Join(rec.Members, ",") != "alice,bob,carol" {
		t.Errorf("after add, members = %v", rec.Members)
	}

	confirmedAPS(t, home, "squad", "members", "remove", "demo-squad", "alice")
	mustJSON(t, mustAPS(t, home, "squad", "show", "demo-squad", "--format", "json"), &rec)
	if strings.Join(rec.Members, ",") != "bob,carol" {
		t.Errorf("after remove, members = %v", rec.Members)
	}

	// check now sees the persisted squad: a platform squad without a
	// golden path fails, so the command exits non-zero, but stdout must
	// still be one JSON array of rows.
	out, _, err := runAPS(t, home, "squad", "check", "--format", "json")
	var checks []map[string]any
	mustJSON(t, out, &checks)
	if err == nil || len(checks) == 0 {
		t.Errorf("check on a platform squad without golden path: err=%v rows=%v", err, checks)
	}
	var failed string
	for _, c := range checks {
		if c["status"] == "FAIL" {
			failed = c["detail"].(string)
		}
	}
	if !strings.Contains(failed, "demo-squad") {
		t.Errorf("check did not evaluate the persisted squad: %q", failed)
	}

	confirmedAPS(t, home, "squad", "delete", "demo-squad")
	if _, stderr, err := runAPS(t, home, "squad", "show", "demo-squad"); err == nil || !strings.Contains(stderr, "squad not found") {
		t.Errorf("show after delete: err=%v stderr=%q", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(squadsDir(home), "demo-squad")); !os.IsNotExist(err) {
		t.Errorf("squad dir should be gone: %v", err)
	}
	var rows []any
	mustJSON(t, mustAPS(t, home, "squad", "list", "--format", "json"), &rows)
	if len(rows) != 0 {
		t.Errorf("list after delete = %v", rows)
	}
}

func TestSquadPersistence_DuplicateCreateFails(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	createDemo(t, home)
	_, stderr, err := runAPS(t, home, "squad", "create", "Demo Squad",
		"--type", "platform", "--domain", "other", "--members", "zed")
	if err == nil || !strings.Contains(stderr, "squad already exists: demo-squad") {
		t.Fatalf("duplicate create: err=%v stderr=%q", err, stderr)
	}
	var rec map[string]any
	mustJSON(t, mustAPS(t, home, "squad", "show", "demo-squad", "--format", "json"), &rec)
	if rec["domain"] != "core" {
		t.Errorf("duplicate create altered the record: %v", rec)
	}
}

func TestSquadPersistence_HonoursAPSDataPath(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	data := filepath.Join(t.TempDir(), "elsewhere")
	env := []string{"APS_DATA_PATH=" + data}
	if _, stderr, err := runAPSEnv(t, home, env, "squad", "create", "S", "--type", "enabling", "--domain", "d"); err != nil {
		t.Fatalf("create: %v %s", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(data, "squads", "s", "manifest.yaml")); err != nil {
		t.Errorf("not under APS_DATA_PATH: %v", err)
	}
	if _, err := os.Stat(squadsDir(home)); !os.IsNotExist(err) {
		t.Errorf("wrote under XDG data dir despite APS_DATA_PATH: %v", err)
	}
}

func TestSquadPersistence_UnsafeNamesRefused(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	outside := filepath.Join(home, ".local", "share", "aps", "outside")
	for _, name := range []string{"../outside", "a/b", "CON", "a:b", "x."} {
		_, stderr, err := runAPS(t, home, "squad", "create", name, "--type", "platform", "--domain", "d")
		if err == nil || !strings.Contains(stderr, "invalid squad ID") {
			t.Errorf("create %q: err=%v stderr=%q", name, err, stderr)
		}
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Errorf("traversal escaped the squads dir: %v", err)
	}
	if ents, _ := os.ReadDir(squadsDir(home)); len(ents) != 0 {
		t.Errorf("unsafe ids left entries: %v", ents)
	}
}

func TestSquadPersistence_CorruptManifest(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	createDemo(t, home)
	bad := filepath.Join(squadsDir(home), "bad", "manifest.yaml")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(": : :\n\t-"), 0o644); err != nil {
		t.Fatal(err)
	}

	// one bad squad does not break the healthy ones
	if out := mustAPS(t, home, "squad", "show", "demo-squad"); !strings.Contains(out, "alice, bob") {
		t.Errorf("healthy show broken by neighbour:\n%s", out)
	}
	stdout, stderr, err := runAPS(t, home, "squad", "list", "--format", "json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var rows []map[string]any
	mustJSON(t, stdout, &rows)
	if len(rows) != 1 || rows[0]["id"] != "demo-squad" {
		t.Errorf("rows = %v", rows)
	}
	if !strings.Contains(stderr, bad) {
		t.Errorf("list stderr should name %s, got %q", bad, stderr)
	}

	// the damaged one is an error naming the file
	if _, stderr, err := runAPS(t, home, "squad", "show", "bad"); err == nil || !strings.Contains(stderr, bad) {
		t.Errorf("show bad: err=%v stderr=%q", err, stderr)
	}
	// check validates the whole topology, so it refuses to run on a partial one
	if _, stderr, err := runAPS(t, home, "squad", "check"); err == nil || !strings.Contains(stderr, bad) {
		t.Errorf("check: err=%v stderr=%q", err, stderr)
	}
	// and the damaged file was left untouched
	if body, _ := os.ReadFile(bad); string(body) != ": : :\n\t-" {
		t.Errorf("corrupt manifest modified: %q", body)
	}
}

// Many processes adding members at once. There is no cross-process lock, so
// an add can be lost; what must hold is that no process fails and the
// manifest stays whole.
func TestSquadPersistence_ConcurrentProcessesKeepStoreIntact(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	createDemo(t, home)

	names := []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	var wg sync.WaitGroup
	for _, n := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, stderr, err := runAPS(t, home, "squad", "members", "add", "demo-squad", n); err != nil {
				t.Errorf("members add %s: %v %s", n, err, stderr)
			}
		}()
	}
	wg.Wait()

	var rec struct {
		Members []string `json:"members"`
	}
	mustJSON(t, mustAPS(t, home, "squad", "show", "demo-squad", "--format", "json"), &rec)
	if len(rec.Members) < 3 || rec.Members[0] != "alice" || rec.Members[1] != "bob" {
		t.Errorf("members = %v", rec.Members)
	}
	ents, _ := os.ReadDir(filepath.Join(squadsDir(home), "demo-squad"))
	if len(ents) != 1 {
		t.Errorf("stray files in squad dir: %v", ents)
	}
}

func TestSquadFormat_ShowJSONAndYAML(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	createDemo(t, home)
	mustAPS(t, home, "squad", "create", "Solo", "--type", "enabling", "--domain", "d")

	wantKeys := "created_at,description,domain,golden_path_defined,id,members,name,type,updated_at"

	var rec map[string]any
	out := mustAPS(t, home, "squad", "show", "demo-squad", "--format", "json")
	mustJSON(t, out, &rec)
	if got := strings.Join(sortedKeys(rec), ","); got != wantKeys {
		t.Errorf("json keys = %s", got)
	}
	if m, _ := rec["members"].([]any); len(m) != 2 || m[0] != "alice" || m[1] != "bob" {
		t.Errorf("members = %v", rec["members"])
	}

	solo := mustAPS(t, home, "squad", "show", "solo", "--format", "json")
	if !strings.Contains(solo, `"members": []`) || strings.Contains(solo, "(none)") || strings.Contains(solo, "null") {
		t.Errorf("empty members must be []:\n%s", solo)
	}

	var y map[string]any
	if err := yaml.Unmarshal([]byte(mustAPS(t, home, "squad", "show", "demo-squad", "--format", "yaml")), &y); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sortedKeys(y), ","); got != wantKeys {
		t.Errorf("yaml keys = %s", got)
	}

	var rows []map[string]any
	mustJSON(t, mustAPS(t, home, "squad", "list", "--format", "json"), &rows)
	if len(rows) != 2 || rows[0]["member_count"] != float64(2) {
		t.Errorf("list rows = %v", rows)
	}
}

func TestSquadFormat_MutationsEmitParseableJSON(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	var rec map[string]any
	mustJSON(t, mustAPS(t, home, "squad", "create", "Demo Squad", "--type", "platform",
		"--domain", "core", "--members", "alice", "--format", "json"), &rec)
	if rec["id"] != "demo-squad" {
		t.Errorf("create = %v", rec)
	}
	rec = nil
	mustJSON(t, mustAPS(t, home, "squad", "members", "add", "demo-squad", "bob", "--format", "json"), &rec)
	if m, _ := rec["members"].([]any); len(m) != 2 {
		t.Errorf("members add = %v", rec)
	}
	rec = nil
	mustJSON(t, confirmedAPS(t, home, "squad", "members", "remove", "demo-squad", "alice", "--format", "json"), &rec)
	if m, _ := rec["members"].([]any); len(m) != 1 || m[0] != "bob" {
		t.Errorf("members remove = %v", rec)
	}
	rec = nil
	mustJSON(t, confirmedAPS(t, home, "squad", "delete", "demo-squad", "--format", "json"), &rec)
	if rec["id"] != "demo-squad" || rec["deleted"] != true {
		t.Errorf("delete = %v", rec)
	}
}

func mustJSON(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, s)
	}
}

func sortedKeys(m map[string]any) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	for i := range k {
		for j := i + 1; j < len(k); j++ {
			if k[j] < k[i] {
				k[i], k[j] = k[j], k[i]
			}
		}
	}
	return k
}

// A platform squad created with --golden-path-defined passes `squad check`
// in a later process; without the flag the failure names the flag.
func TestSquadCheck_GoldenPathFlagAcrossProcesses(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	mustAPS(t, home, "squad", "create", "plat", "--type", "platform",
		"--domain", "infra", "--members", "alice", "--golden-path-defined")

	var rec struct {
		GoldenPath bool `json:"golden_path_defined"`
	}
	mustJSON(t, mustAPS(t, home, "squad", "show", "plat", "--format", "json"), &rec)
	if !rec.GoldenPath {
		t.Error("golden_path_defined not persisted")
	}
	if _, stderr, err := runAPS(t, home, "squad", "check"); err != nil {
		t.Errorf("check on flagged platform squad: %v\n%s", err, stderr)
	}

	mustAPS(t, home, "squad", "create", "plat2", "--type", "platform",
		"--domain", "infra2", "--members", "bob")
	out, _, err := runAPS(t, home, "squad", "check")
	if err == nil || !strings.Contains(out, "--golden-path-defined") || !strings.Contains(out, "plat2") {
		t.Errorf("check without flag: err=%v out=%q", err, out)
	}
}
