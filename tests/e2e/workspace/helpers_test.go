// Package workspace_e2e exercises Phase-6 list commands under
// `aps workspace …`: conflicts list (T-0441), ctx list (T-0442),
// policy list (T-0443). Each test compiles the aps binary into a
// temp dir, seeds workspace state in an isolated HOME, and asserts
// stdout against the listing helper's expected output.
package workspace_e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/core/multidevice"
)

// apsBinary is the absolute path to the compiled aps binary used by
// all child-process invocations. Built once in TestMain.
var apsBinary string

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Remove(apsBinary)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-workspace-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)

	rootDir, err := filepath.Abs("../../..")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runAPS executes the compiled aps binary in an isolated HOME and
// returns stdout/stderr/err.
func runAPS(t *testing.T, home string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)

	override := map[string]bool{
		"HOME":          true,
		"USERPROFILE":   true,
		"XDG_DATA_HOME": true,
		"APS_DATA_PATH": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"APS_DATA_PATH=" + filepath.Join(home, ".local", "share", "aps"),
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if override[key] {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// dataDir returns the APS data dir for the test home (matches the env
// override in runAPS).
func dataDir(home string) string {
	return filepath.Join(home, ".local", "share", "aps")
}

// writeJSON writes v as JSON to path, creating parent dirs.
func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// seedConflicts writes one JSON file per conflict under
// <data>/workspaces/<wsID>/conflicts/<id>.json — matches the shape
// ConflictStore expects.
func seedConflicts(t *testing.T, home, wsID string, conflicts []*multidevice.Conflict) {
	t.Helper()
	dir := filepath.Join(dataDir(home), "workspaces", wsID, "conflicts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir conflicts: %v", err)
	}
	for _, c := range conflicts {
		writeJSON(t, filepath.Join(dir, c.ID+".json"), c)
	}
}

// seedContext writes context.json for a workspace.
func seedContext(t *testing.T, home, wsID string, vars []collab.ContextVariable) {
	t.Helper()
	wsDir := filepath.Join(dataDir(home), "collaboration", wsID)
	writeJSON(t, filepath.Join(wsDir, "context.json"), vars)
}

// seedWorkspace writes manifest.yaml + state.json for a collab workspace
// so commands that load via NewCollaborationStorage can find it.
func seedWorkspace(t *testing.T, home string, ws *collab.Workspace) {
	t.Helper()
	wsDir := filepath.Join(dataDir(home), "collaboration", ws.ID)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", wsDir, err)
	}

	manifestBytes, err := yaml.Marshal(ws.Config)
	if err != nil {
		t.Fatalf("marshal manifest yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "manifest.yaml"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write manifest.yaml: %v", err)
	}

	state := map[string]any{
		"id":         ws.ID,
		"state":      ws.State,
		"agents":     ws.Agents,
		"policy":     ws.Policy,
		"created_at": ws.CreatedAt,
		"updated_at": ws.UpdatedAt,
	}
	writeJSON(t, filepath.Join(wsDir, "state.json"), state)
}

// nowUTC returns time.Now() truncated to seconds for stable formatting.
func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
