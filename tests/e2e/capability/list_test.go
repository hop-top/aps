// Package capability holds e2e tests for `aps capability list` and
// `aps capability patterns list` filter flags.
package capability

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	binName := "aps-capability-e2e"
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

func runCap(t *testing.T, home string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	cmd.Env = append(os.Environ(), "HOME="+home)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestCapabilityList_FormatJSON exercises the listing.RenderList JSON
// path and confirms builtin rows surface.
func TestCapabilityList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	out, errStr, err := runCap(t, home, "capability", "list", "--format", "json")
	if err != nil {
		t.Fatalf("aps capability list --format json: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected ≥1 builtin capability")
	}
}

// TestCapabilityList_BuiltinOnly drops external rows.
func TestCapabilityList_BuiltinOnly(t *testing.T) {
	home := t.TempDir()
	out, _, err := runCap(t, home,
		"capability", "list", "--builtin", "--format", "json")
	if err != nil {
		t.Fatalf("--builtin: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	for _, r := range rows {
		if r["source"] != "builtin" {
			t.Fatalf("--builtin returned non-builtin: %v", r)
		}
	}
}

// TestCapabilityList_ExternalOnly produces an empty set when no
// externals are installed.
func TestCapabilityList_ExternalOnly(t *testing.T) {
	home := t.TempDir()
	out, _, err := runCap(t, home,
		"capability", "list", "--external", "--format", "json")
	if err != nil {
		t.Fatalf("--external: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) != 0 {
		t.Fatalf("fresh HOME should have 0 external rows; got %d", len(rows))
	}
}

// TestCapabilityList_BuiltinAndExternalMutex rejects both flags together.
func TestCapabilityList_BuiltinAndExternalMutex(t *testing.T) {
	home := t.TempDir()
	_, errStr, err := runCap(t, home,
		"capability", "list", "--builtin", "--external")
	if err == nil {
		t.Fatalf("expected non-zero exit; stderr=%s", errStr)
	}
	if !strings.Contains(errStr, "mutually exclusive") {
		t.Fatalf("expected mutex error; got: %s", errStr)
	}
}

// TestCapabilityList_EnabledOn narrows by profile membership; an
// unknown profile returns 0 rows (no profiles exist in fresh HOME).
func TestCapabilityList_EnabledOn(t *testing.T) {
	home := t.TempDir()
	out, _, err := runCap(t, home,
		"capability", "list", "--enabled-on", "ghost", "--format", "json")
	if err != nil {
		t.Fatalf("--enabled-on: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for unknown profile; got %d", len(rows))
	}
}

// TestPatternsList_FormatJSON exercises the sibling `patterns list`.
func TestPatternsList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	out, errStr, err := runCap(t, home,
		"capability", "patterns", "list", "--format", "json")
	if err != nil {
		t.Fatalf("patterns list: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected ≥1 pattern row")
	}
	for _, r := range rows {
		if r["tool"] == "" || r["default_path"] == "" {
			t.Fatalf("pattern row missing fields: %v", r)
		}
	}
}
