// Package bundle holds e2e tests for `aps bundle list` filter flags.
//
// The list cmd renders builtin + user bundles via internal/cli/listing,
// with --tag, --builtin, --user filters. These tests spawn the compiled
// aps binary against an isolated HOME so user-bundle assets are
// controllable per case.
package bundle

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
	binName := "aps-bundle-e2e"
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

// writeUserBundle writes a YAML bundle into <home>/.config/aps/bundles.
// We set XDG_CONFIG_HOME=<home>/.config in the child env so os.UserConfigDir()
// resolves there on every platform (macOS would otherwise pick
// ~/Library/Application Support).
func writeUserBundle(t *testing.T, home, name string, tags []string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "aps", "bundles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir bundles: %v", err)
	}
	tagsBlock := ""
	if len(tags) > 0 {
		tagsBlock = "tags:\n"
		for _, tg := range tags {
			tagsBlock += "  - " + tg + "\n"
		}
	}
	body := fmt.Sprintf(`name: %s
description: "test bundle"
version: "0.0.1"
%s`, name, tagsBlock)
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write bundle yaml: %v", err)
	}
}

func runBundle(t *testing.T, home string, args ...string) (string, string, error) {
	t.Helper()
	args = append([]string{"bundle", "list"}, args...)
	cmd := exec.Command(apsBinary, args...)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestBundleList_FormatJSON exercises the listing.RenderList JSON path.
func TestBundleList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	writeUserBundle(t, home, "u1", []string{"alpha"})

	out, errStr, err := runBundle(t, home, "--format", "json")
	if err != nil {
		t.Fatalf("aps bundle list --format json: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected ≥1 row, got 0")
	}
	hasU1 := false
	for _, r := range rows {
		if r["name"] == "u1" {
			hasU1 = true
		}
	}
	if !hasU1 {
		t.Fatalf("expected user bundle u1 present; got: %s", out)
	}
}

// TestBundleList_TagFilter narrows to bundles whose tags contain the value.
func TestBundleList_TagFilter(t *testing.T) {
	home := t.TempDir()
	writeUserBundle(t, home, "u-alpha", []string{"alpha", "shared"})
	writeUserBundle(t, home, "u-beta", []string{"beta"})

	out, errStr, err := runBundle(t, home, "--tag", "alpha", "--format", "json", "--user")
	if err != nil {
		t.Fatalf("aps bundle list --tag alpha: %v\nstderr: %s", err, errStr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	names := map[string]bool{}
	for _, r := range rows {
		names[fmt.Sprint(r["name"])] = true
	}
	if !names["u-alpha"] || names["u-beta"] {
		t.Fatalf("expected u-alpha only; got: %v", names)
	}
}

// TestBundleList_BuiltinOnly drops user bundles.
func TestBundleList_BuiltinOnly(t *testing.T) {
	home := t.TempDir()
	writeUserBundle(t, home, "u1", nil)

	out, _, err := runBundle(t, home, "--builtin", "--format", "json")
	if err != nil {
		t.Fatalf("aps bundle list --builtin: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	for _, r := range rows {
		if r["source"] != "built-in" {
			t.Fatalf("--builtin returned non-builtin row: %v", r)
		}
	}
}

// TestBundleList_UserOnly drops built-in bundles.
func TestBundleList_UserOnly(t *testing.T) {
	home := t.TempDir()
	writeUserBundle(t, home, "u1", nil)

	out, _, err := runBundle(t, home, "--user", "--format", "json")
	if err != nil {
		t.Fatalf("aps bundle list --user: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected user row(s)")
	}
	for _, r := range rows {
		if r["source"] == "built-in" {
			t.Fatalf("--user returned built-in row: %v", r)
		}
	}
}

// TestBundleList_BuiltinAndUserMutex rejects --builtin --user together.
func TestBundleList_BuiltinAndUserMutex(t *testing.T) {
	home := t.TempDir()
	_, errStr, err := runBundle(t, home, "--builtin", "--user")
	if err == nil {
		t.Fatalf("expected non-zero exit; stderr=%s", errStr)
	}
	if !strings.Contains(errStr, "mutually exclusive") {
		t.Fatalf("expected mutex error in stderr; got: %s", errStr)
	}
}
