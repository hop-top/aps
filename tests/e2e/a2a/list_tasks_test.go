// Cross-process e2e tests for `aps a2a tasks list` (T-0438).
//
// Each test:
//  1. seeds an aps data dir with a profile (a2a capability) and a
//     handful of synthetic task records under .../a2a/<profile>/tasks,
//  2. spawns the aps binary with --profile + filter flags,
//  3. asserts the rendered table contains the expected task IDs.
//
// Build tag a2a_e2e — excluded from default `go test`. Run with:
//
//	go test -tags a2a_e2e -count=1 ./tests/e2e/a2a/...
//
//go:build a2a_e2e

package a2a

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
)

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
	binName := "aps-a2a-e2e"
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

// seedProfile writes a minimal profile yaml with the a2a capability set.
func seedProfile(t *testing.T, dataDir, profileID string) {
	t.Helper()
	profileDir := filepath.Join(dataDir, "profiles", profileID)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	yaml := fmt.Sprintf(`id: %s
display_name: %s
capabilities:
  - a2a
`, profileID, profileID)
	if err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}
}

// seedTask writes a synthetic a2a Task json record under the storage layout
// internal/a2a/storage.go expects: .../a2a/<profile>/tasks/<id>/meta.json.
func seedTask(t *testing.T, dataDir, profileID, id, state, recipient string, msgs int, ts time.Time) {
	t.Helper()
	taskDir := filepath.Join(dataDir, "a2a", profileID, "tasks", id)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	hist := make([]map[string]any, msgs)
	for i := range hist {
		hist[i] = map[string]any{"messageId": fmt.Sprintf("m%d", i+1), "role": "user", "parts": []any{}}
	}
	task := map[string]any{
		"id":        id,
		"contextId": "ctx-" + id,
		"history":   hist,
		"status": map[string]any{
			"state":     state,
			"timestamp": ts.UTC().Format(time.RFC3339Nano),
		},
	}
	if recipient != "" {
		task["metadata"] = map[string]any{"recipient": recipient}
	}
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "meta.json"), data, 0o600); err != nil {
		t.Fatalf("write meta.json: %v", err)
	}
}

// runAPS executes the compiled binary with APS_DATA_PATH set to dataDir.
func runAPS(t *testing.T, dataDir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	cmd.Env = append(os.Environ(), "APS_DATA_PATH="+dataDir, "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestA2ATasksList_AllStatuses — no filter, all 3 tasks render.
func TestA2ATasksList_AllStatuses(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	now := time.Now().UTC()
	seedTask(t, dataDir, "noor", "task-1", "submitted", "alice", 1, now)
	seedTask(t, dataDir, "noor", "task-2", "working", "bob", 3, now)
	seedTask(t, dataDir, "noor", "task-3", "completed", "", 5, now)

	out, errOut, err := runAPS(t, dataDir, "--profile", "noor", "a2a", "tasks", "list")
	if err != nil {
		t.Fatalf("aps a2a tasks list: %v\nstdout:%s\nstderr:%s", err, out, errOut)
	}
	for _, want := range []string{"task-1", "task-2", "task-3"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in output, got:\n%s", want, out)
		}
	}
}

// TestA2ATasksList_StatusFilter — --status working keeps only task-2.
func TestA2ATasksList_StatusFilter(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	now := time.Now().UTC()
	seedTask(t, dataDir, "noor", "task-1", "submitted", "alice", 1, now)
	seedTask(t, dataDir, "noor", "task-2", "working", "bob", 3, now)
	seedTask(t, dataDir, "noor", "task-3", "completed", "", 5, now)

	out, errOut, err := runAPS(t, dataDir,
		"--profile", "noor", "a2a", "tasks", "list", "--status", "working")
	if err != nil {
		t.Fatalf("aps a2a tasks list: %v\nstdout:%s\nstderr:%s", err, out, errOut)
	}
	if !strings.Contains(out, "task-2") {
		t.Errorf("expected task-2 in --status=working output, got:\n%s", out)
	}
	for _, drop := range []string{"task-1", "task-3"} {
		if strings.Contains(out, drop) {
			t.Errorf("unexpected %s in --status=working output:\n%s", drop, out)
		}
	}
}

// TestA2ATasksList_JSONFormat — --format json emits a JSON array carrying
// the same shape (id, status, profile, recipient, messages, updated_at).
func TestA2ATasksList_JSONFormat(t *testing.T) {
	dataDir := t.TempDir()
	seedProfile(t, dataDir, "noor")
	seedTask(t, dataDir, "noor", "only", "submitted", "echo", 1, time.Now().UTC())

	out, errOut, err := runAPS(t, dataDir,
		"--profile", "noor", "--format", "json", "a2a", "tasks", "list")
	if err != nil {
		t.Fatalf("aps a2a tasks list --format json: %v\nstdout:%s\nstderr:%s",
			err, out, errOut)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json output: %v\nout:%s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d (%v)", len(rows), rows)
	}
	if got := rows[0]["id"]; got != "only" {
		t.Errorf("id = %v, want only", got)
	}
	if got := rows[0]["recipient"]; got != "echo" {
		t.Errorf("recipient = %v, want echo", got)
	}
}
