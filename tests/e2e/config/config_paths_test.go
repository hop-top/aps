// Package config holds e2e tests for `aps config path` and
// `aps config paths` — the kit-shared introspection subcommands wired
// in T-0457 per CLI conventions §7.4.
package config

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
	binName := "aps-config-e2e"
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

// hermeticEnv builds a minimal env that pins HOME and XDG_CONFIG_HOME so
// the aps binary's resolver cannot leak into the developer's real config
// dirs. Returns the home dir + the XDG_CONFIG_HOME path for assertions.
func hermeticEnv(t *testing.T) (home, xdgConfigHome string, env []string) {
	t.Helper()
	raw := t.TempDir()
	// Resolve symlinks so /var/folders/... vs /private/var/folders/...
	// match what `filepath.Abs(cwd)` reports inside the binary on macOS.
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		resolved = raw
	}
	home = resolved
	xdgConfigHome = filepath.Join(home, ".config")
	if err := os.MkdirAll(xdgConfigHome, 0o755); err != nil {
		t.Fatalf("mkdir xdg: %v", err)
	}

	overrides := map[string]string{
		"HOME":            home,
		"USERPROFILE":     home,
		"XDG_CONFIG_HOME": xdgConfigHome,
		"XDG_CONFIG_DIRS": "",
	}
	skip := map[string]bool{}
	env = make([]string, 0, len(os.Environ()))
	for k, v := range overrides {
		env = append(env, k+"="+v)
		skip[k] = true
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if skip[key] {
			continue
		}
		env = append(env, e)
	}
	return home, xdgConfigHome, env
}

func runConfigCmd(t *testing.T, env []string, cwd string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	cmd.Env = env
	cmd.Dir = cwd
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestConfigPath_PrintsUserConfigWhenPresent asserts `aps config path`
// resolves to the user-scope $XDG_CONFIG_HOME/aps/config.yaml when no
// project marker exists but a user config does. Exit 0.
func TestConfigPath_PrintsUserConfigWhenPresent(t *testing.T) {
	home, xdg, env := hermeticEnv(t)
	apsCfgDir := filepath.Join(xdg, "aps")
	if err := os.MkdirAll(apsCfgDir, 0o755); err != nil {
		t.Fatalf("mkdir aps cfg dir: %v", err)
	}
	cfgPath := filepath.Join(apsCfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("prefix: E2E\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	stdout, stderr, err := runConfigCmd(t, env, home, "config", "path")
	if err != nil {
		t.Fatalf("config path: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	got := strings.TrimSpace(stdout)
	if got != cfgPath {
		t.Errorf("config path: got %q, want %q", got, cfgPath)
	}
}

// TestConfigPath_PrefersProjectOverUser asserts cwd-scope project marker
// wins over user-scope when both exist.
func TestConfigPath_PrefersProjectOverUser(t *testing.T) {
	home, xdg, env := hermeticEnv(t)
	// User layer: write a config at $XDG_CONFIG_HOME/aps/config.yaml.
	apsCfgDir := filepath.Join(xdg, "aps")
	if err := os.MkdirAll(apsCfgDir, 0o755); err != nil {
		t.Fatalf("mkdir aps cfg dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apsCfgDir, "config.yaml"), []byte("prefix: USER\n"), 0o644); err != nil {
		t.Fatalf("write user cfg: %v", err)
	}
	// Project layer: drop a .aps.yaml inside cwd.
	projectDir := filepath.Join(home, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectCfg := filepath.Join(projectDir, ".aps.yaml")
	if err := os.WriteFile(projectCfg, []byte("prefix: PROJ\n"), 0o644); err != nil {
		t.Fatalf("write project cfg: %v", err)
	}

	stdout, stderr, err := runConfigCmd(t, env, projectDir, "config", "path")
	if err != nil {
		t.Fatalf("config path: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	got := strings.TrimSpace(stdout)
	if got != projectCfg {
		t.Errorf("config path: got %q, want %q (project marker should win)", got, projectCfg)
	}
}

// TestConfigPath_NoConfigFallsBackToDefaults asserts the synthetic
// `<defaults>` rung is reported when no real file exists. Per kit's
// pathCommand the defaults rung is always present, so exit 0.
func TestConfigPath_NoConfigFallsBackToDefaults(t *testing.T) {
	home, _, env := hermeticEnv(t)
	stdout, stderr, err := runConfigCmd(t, env, home, "config", "path")
	if err != nil {
		t.Fatalf("config path: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	got := strings.TrimSpace(stdout)
	if got != "<defaults>" {
		t.Errorf("config path: got %q, want %q", got, "<defaults>")
	}
}

// TestConfigPaths_ListsLayeredChainText asserts `aps config paths`
// prints the cwd → project walk-up → user → system → default chain in
// text mode. Each rung is one path per line.
func TestConfigPaths_ListsLayeredChainText(t *testing.T) {
	home, xdg, env := hermeticEnv(t)
	stdout, stderr, err := runConfigCmd(t, env, home, "config", "paths")
	if err != nil {
		t.Fatalf("config paths: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	body := stdout
	wantSubstrings := []string{
		filepath.Join(home, ".aps", "config.yaml"),
		filepath.Join(home, ".hop", "aps", "config.yaml"),
		filepath.Join(home, ".aps.yaml"),
		filepath.Join(xdg, "aps", "config.yaml"),
		filepath.Join("/etc", "aps", "config.yaml"),
		"<defaults>",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(body, want) {
			t.Errorf("config paths: expected line %q in output:\n%s", want, body)
		}
	}
}

// TestConfigPaths_JSONHonoursFormatFlag asserts the kit-owned --format
// flag plumbs through and produces the ResolvedPath JSON shape (path,
// source, scope, exists) per §7.4.
func TestConfigPaths_JSONHonoursFormatFlag(t *testing.T) {
	home, xdg, env := hermeticEnv(t)
	stdout, stderr, err := runConfigCmd(t, env, home, "config", "paths", "--format", "json")
	if err != nil {
		t.Fatalf("config paths --format json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var entries []struct {
		Path   string `json:"path"`
		Source string `json:"source"`
		Scope  string `json:"scope"`
		Exists bool   `json:"exists"`
	}
	if err := json.Unmarshal([]byte(stdout), &entries); err != nil {
		t.Fatalf("decode json: %v\nbody=%s", err, stdout)
	}
	if len(entries) == 0 {
		t.Fatalf("expected non-empty chain")
	}

	// Sources should include cwd, user, system, default — at minimum.
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.Source] = true
	}
	for _, want := range []string{"cwd", "user", "system", "default"} {
		if !seen[want] {
			t.Errorf("missing source %q in chain: %+v", want, entries)
		}
	}

	// User entry should point at $XDG_CONFIG_HOME/aps/config.yaml.
	wantUser := filepath.Join(xdg, "aps", "config.yaml")
	var userHit bool
	for _, e := range entries {
		if e.Source == "user" && e.Path == wantUser {
			userHit = true
			break
		}
	}
	if !userHit {
		t.Errorf("user rung not pointing at %q; chain=%+v", wantUser, entries)
	}
}
